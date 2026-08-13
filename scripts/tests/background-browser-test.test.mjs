import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import test from 'node:test';

const repoRoot = resolve(import.meta.dirname, '..', '..');
const runner = resolve(repoRoot, 'scripts', 'background-browser-test.mjs');

function waitForTerminalState(statePath) {
  const deadline = Date.now() + 10000;
  while (Date.now() < deadline) {
    const state = JSON.parse(readFileSync(statePath, 'utf8'));
    if (['passed', 'failed', 'timed_out'].includes(state.status)) return state;
    Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, 50);
  }
  throw new Error('background test did not finish');
}

test('background browser runner returns immediately and preserves result evidence', () => {
  const root = mkdtempSync(join(tmpdir(), 'kpanel-browser-job-'));
  const evidenceDir = join(root, 'evidence');
  const specPath = join(root, 'spec.json');
  writeFileSync(specPath, JSON.stringify({
    command: process.execPath,
    args: ['-e', 'console.log("browser-job-ok")'],
    cwd: repoRoot,
  }));
  const output = execFileSync(process.execPath, [
    runner,
    'start',
    '--environment', 'arena-154',
    '--candidate-ref', '0000000000000000000000000000000000000000',
    '--spec', specPath,
    '--evidence-dir', evidenceDir,
    '--timeout-seconds', '10',
  ], { encoding: 'utf8' });
  const job = JSON.parse(output);
  assert.equal(job.evidenceDir, evidenceDir);
  const state = waitForTerminalState(join(evidenceDir, 'state.json'));
  assert.equal(state.status, 'passed');
  assert.equal(state.exitCode, 0);
  assert.equal(state.candidateRef, '0000000000000000000000000000000000000000');
  assert.match(readFileSync(join(evidenceDir, 'browser-test.log'), 'utf8'), /browser-job-ok/);
  assert.deepEqual(
    JSON.parse(readFileSync(join(evidenceDir, 'command-spec.json'), 'utf8')),
    JSON.parse(readFileSync(specPath, 'utf8')),
  );
});

test('background browser runner rejects prod-108 before launching', () => {
  const root = mkdtempSync(join(tmpdir(), 'kpanel-browser-job-'));
  const specPath = join(root, 'spec.json');
  writeFileSync(specPath, JSON.stringify({ command: process.execPath, args: ['-e', 'process.exit(0)'] }));
  assert.throws(() => execFileSync(process.execPath, [
    runner,
    'start',
    '--environment', 'prod-108',
    '--candidate-ref', '0000000000000000000000000000000000000000',
    '--spec', specPath,
    '--evidence-dir', join(root, 'evidence'),
    '--timeout-seconds', '10',
  ], { encoding: 'utf8', stdio: 'pipe' }));
});

test('background browser runner records a hard timeout as failure evidence', () => {
  const root = mkdtempSync(join(tmpdir(), 'kpanel-browser-job-'));
  const evidenceDir = join(root, 'evidence');
  const specPath = join(root, 'spec.json');
  writeFileSync(specPath, JSON.stringify({
    command: process.execPath,
    args: ['-e', 'setTimeout(() => {}, 5000)'],
  }));
  execFileSync(process.execPath, [
    runner,
    'start',
    '--environment', 'arena-154',
    '--candidate-ref', '0000000000000000000000000000000000000000',
    '--spec', specPath,
    '--evidence-dir', evidenceDir,
    '--timeout-seconds', '1',
  ], { encoding: 'utf8' });
  const state = waitForTerminalState(join(evidenceDir, 'state.json'));
  assert.equal(state.status, 'timed_out');
  assert.notEqual(state.exitCode, 0);
});

test('background browser runner rejects mutable candidate aliases', () => {
  const root = mkdtempSync(join(tmpdir(), 'kpanel-browser-job-'));
  const specPath = join(root, 'spec.json');
  writeFileSync(specPath, JSON.stringify({ command: process.execPath, args: ['-e', 'process.exit(0)'] }));
  assert.throws(() => execFileSync(process.execPath, [
    runner,
    'start',
    '--environment', 'arena-154',
    '--candidate-ref', 'HEAD',
    '--spec', specPath,
    '--evidence-dir', join(root, 'evidence'),
    '--timeout-seconds', '10',
  ], { encoding: 'utf8', stdio: 'pipe' }));
});

test('background browser runner refuses persisted secrets in command specs', () => {
  const root = mkdtempSync(join(tmpdir(), 'kpanel-browser-job-'));
  const specPath = join(root, 'spec.json');
  writeFileSync(specPath, JSON.stringify({
    command: process.execPath,
    args: ['-e', 'process.exit(0)'],
    env: { TEST_TOKEN: 'must-not-be-written' },
  }));
  assert.throws(() => execFileSync(process.execPath, [
    runner,
    'start',
    '--environment', 'arena-154',
    '--candidate-ref', '0000000000000000000000000000000000000000',
    '--spec', specPath,
    '--evidence-dir', join(root, 'evidence'),
    '--timeout-seconds', '10',
  ], { encoding: 'utf8', stdio: 'pipe' }));
});
