import assert from 'node:assert/strict';
import test from 'node:test';

import {
  classifyUpdate,
  compareVersions,
  goExecutable,
  immutableDigestCandidate,
  isStableVersion,
  npmInvocation,
  parseConcatenatedJson,
  renderMarkdown,
  summarize,
  validatePolicy,
} from '../report-dependency-freshness.mjs';

test('stable version filter excludes prereleases and floating labels', () => {
  assert.equal(isStableVersion('v1.2.3'), true);
  assert.equal(isStableVersion('1.2.3-rc.1'), false);
  assert.equal(isStableVersion('latest'), false);
});

test('npm invocation avoids Windows command-shell wrappers and supports overrides', () => {
  assert.deepEqual(npmInvocation('win32', {}, 'C:\\Node\\node.exe'), {
    command: 'C:\\Node\\node.exe',
    prefixArguments: ['C:\\Node\\node_modules\\npm\\bin\\npm-cli.js'],
  });
  assert.deepEqual(npmInvocation('linux', {}, '/usr/bin/node'), { command: 'npm', prefixArguments: [] });
  assert.deepEqual(npmInvocation('win32', { NPM: 'custom-npm' }, 'node'), { command: 'custom-npm', prefixArguments: [] });
  assert.equal(goExecutable({}), 'go');
  assert.equal(goExecutable({ GO: 'custom-go' }), 'custom-go');
});

test('version comparison and classification separate patch, minor, and major/toolchain changes', () => {
  assert.equal(compareVersions('1.2.3', '1.2.4'), -1);
  assert.equal(compareVersions('2.0.0', '1.9.9'), 1);
  assert.equal(classifyUpdate('1.2.3', '1.2.4'), 'compatible-patch');
  assert.equal(classifyUpdate('1.2.3', '1.3.0'), 'minor');
  assert.equal(classifyUpdate('1.2.3', '2.0.0'), 'major-toolchain-base');
  assert.equal(classifyUpdate('1.2.3', '1.2.4', 'toolchain'), 'major-toolchain-base');
});

test('immutable digest drift is visible even when the image tag is unchanged', () => {
  assert.equal(immutableDigestCandidate('image', 'sha256:' + 'a'.repeat(64), 'sha256:' + 'a'.repeat(64), 'registry'), null);
  const drift = immutableDigestCandidate('image', 'sha256:' + 'a'.repeat(64), 'sha256:' + 'b'.repeat(64), 'registry');
  assert.equal(drift.updateClass, 'major-toolchain-base');
  assert.match(drift.current, /^sha256:a/);
  assert.match(drift.candidate, /^sha256:b/);
});

test('concatenated Go JSON parser keeps nested values and escaped braces intact', () => {
  const values = parseConcatenatedJson('{"Path":"a","Update":{"Version":"v1.2.3"}}\n{"Path":"b","Note":"}"}');
  assert.equal(values.length, 2);
  assert.equal(values[0].Update.Version, 'v1.2.3');
  assert.equal(values[1].Note, '}');
});

test('policy validation requires complete coverage and prohibits automatic main changes', () => {
  const policy = {
    schemaVersion: 1,
    requiredGroups: [],
    groups: [],
    exceptionRequiredFields: ['reason', 'owner', 'reviewDate', 'exitCondition', 'rollbackPoint'],
    exceptions: [],
    automationBoundary: { automaticMainCommit: true, automaticRelease: false, automaticProductionDeployment: false },
  };
  const failures = validatePolicy(policy, process.cwd());
  assert.ok(failures.some((failure) => failure.includes('go-modules')));
  assert.ok(failures.some((failure) => failure.includes('automatic main')));
});

test('summary does not turn missing sources into an all-current conclusion', () => {
  const report = summarize({ policyVersion: 'test' }, [
    { id: 'go-modules', status: 'ok', candidates: [{ component: 'example', current: '1.0.0', candidate: '1.0.1', updateClass: 'compatible-patch', source: 'test' }], error: null },
    { id: 'npm-packages', status: 'error', candidates: [], error: 'offline' },
  ], new Date('2026-08-12T00:00:00Z'));
  assert.equal(report.failedSources, 1);
  assert.equal(report.candidateCount, 1);
  assert.equal(report.classCounts['compatible-patch'], 1);
  const markdown = renderMarkdown(report);
  assert.match(markdown, /只发现候选/);
  assert.match(markdown, /数据缺口/);
});

test('markdown report keeps multiline and pipe errors inside one bounded table cell', () => {
  const markdown = renderMarkdown(summarize({ policyVersion: 'test' }, [
    { id: 'source', status: 'error', candidates: [], error: 'line one\nline | two ' + 'x'.repeat(400) },
  ], new Date('2026-08-12T00:00:00Z')));
  assert.doesNotMatch(markdown, /line one\nline/);
  assert.match(markdown, /line \\| two/);
  assert.ok(markdown.split('\n').find((line) => line.includes('line one')).length < 400);
});
