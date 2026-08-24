import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

const repoRoot = resolve(import.meta.dirname, '..', '..');
const orchestrator = resolve(repoRoot, 'scripts', 'run-production-evidence.mjs');
const remoteScript = resolve(repoRoot, 'scripts', 'run-production-evidence-remote.sh');

function createFixture() {
  const root = mkdtempSync(join(tmpdir(), 'kpanel-production-evidence-test-'));
  const repo = join(root, 'repo');
  mkdirSync(join(repo, 'scripts'), { recursive: true });
  spawnSync('git', ['init', '--initial-branch=main', repo], { encoding: 'utf8', shell: false });
  writeFileSync(join(repo, 'scripts', 'run-production-evidence-remote.sh'), readFileSync(remoteScript));
  return { root, repo };
}

function cleanup(root) {
  const prefix = resolve(tmpdir()) + (process.platform === 'win32' ? '\\' : '/');
  assert.ok(resolve(root).startsWith(prefix));
  rmSync(root, { recursive: true, force: true });
}

test('prepare-only creates a hashed preflight plan without shell interpolation', () => {
  const fixture = createFixture();
  try {
    const artifactDir = join(fixture.root, 'artifacts');
    const result = spawnSync(process.execPath, [
      orchestrator,
      '--repo', fixture.repo,
      '--phase', 'preflight',
      '--run-id', 'v0.98.0-production',
      '--expected-version', '0.97.3',
      '--artifact-dir', artifactDir,
      '--prepare-only',
    ], { cwd: fixture.repo, encoding: 'utf8', shell: false });
    assert.equal(result.status, 0, `${result.stdout}\n${result.stderr}`);
    const plan = readFileSync(join(artifactDir, 'plan.env'), 'utf8');
    assert.match(plan, /^PHASE=preflight$/m);
    assert.match(plan, /^EXPECTED_VERSION=0\.97\.3$/m);
    assert.match(plan, /^REMOTE_SCRIPT_SHA256=[0-9a-f]{64}$/m);
  } finally {
    cleanup(fixture.root);
  }
});

test('postdeploy fails closed without immutable revision, digest, and baseline', () => {
  const fixture = createFixture();
  try {
    const result = spawnSync(process.execPath, [
      orchestrator,
      '--repo', fixture.repo,
      '--phase', 'postdeploy',
      '--run-id', 'v0.98.0-production',
      '--expected-version', '0.98.0',
      '--artifact-dir', join(fixture.root, 'postdeploy'),
      '--prepare-only',
    ], { cwd: fixture.repo, encoding: 'utf8', shell: false });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /full expected revision/);
  } finally {
    cleanup(fixture.root);
  }
});

test('production evidence target is fixed to arena-154', () => {
  const fixture = createFixture();
  try {
    const result = spawnSync(process.execPath, [
      orchestrator,
      '--repo', fixture.repo,
      '--target', '108',
      '--phase', 'preflight',
      '--run-id', 'v0.98.0-production',
      '--expected-version', '0.97.3',
      '--artifact-dir', join(fixture.root, 'wrong-target'),
      '--prepare-only',
    ], { cwd: fixture.repo, encoding: 'utf8', shell: false });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /must be arena-154/);
  } finally {
    cleanup(fixture.root);
  }
});

test('remote production entrypoint is strict and never evaluates the plan', () => {
  const bash = process.env.KPANEL_TEST_BASH ||
    (process.platform === 'win32' ? 'C:\\Program Files\\Git\\bin\\bash.exe' : 'bash');
  const syntax = spawnSync(bash, ['-n', 'scripts/run-production-evidence-remote.sh'], {
    cwd: repoRoot,
    encoding: 'utf8',
  });
  assert.equal(syntax.status, 0, `${syntax.stdout}\n${syntax.stderr}`);
  const content = readFileSync(remoteScript, 'utf8');
  assert.match(content, /duplicate production plan key/);
  assert.match(content, /docker compose -f "\$compose" stop/);
  assert.match(content, /protected\.sha256/);
  assert.match(content, /sha256sum -c SHA256SUMS/);
  assert.doesNotMatch(content, /\beval\b/);
  assert.doesNotMatch(content, /(?:^|\n)\s*(?:source|\.)\s+"?\$plan/m);
});
