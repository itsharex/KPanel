import test from 'node:test';
import assert from 'node:assert/strict';

import {
  cleanShellEnvironment,
  gitBashCandidates,
  parseRunArguments,
  resolveBash,
} from '../run-repo-bash.mjs';

test('Windows resolves Git for Windows Bash instead of an ambiguous WSL bash', () => {
  const gitExecutable = 'C:\\Program Files\\Git\\cmd\\git.exe';
  const candidates = gitBashCandidates(gitExecutable, { ProgramFiles: 'C:\\Program Files' });
  assert.equal(candidates[0], 'C:\\Program Files\\Git\\bin\\bash.exe');
  assert.equal(resolveBash({
    platform: 'win32',
    environment: { ProgramFiles: 'C:\\Program Files' },
    gitExecutable,
    pathExists: (path) => path === 'C:\\Program Files\\Git\\bin\\bash.exe',
  }), 'C:\\Program Files\\Git\\bin\\bash.exe');
});

test('non-Windows keeps the platform Bash and Windows fails closed when Git Bash is absent', () => {
  assert.equal(resolveBash({ platform: 'linux', environment: {} }), 'bash');
  assert.throws(() => resolveBash({
    platform: 'win32',
    environment: {},
    gitExecutable: null,
    pathExists: () => false,
  }), /Git for Windows Bash was not found/);
});

test('launcher arguments keep verification environment explicit', () => {
  assert.deepEqual(parseRunArguments([
    '--env', 'VERIFY_LEVEL=l2', '--', 'scripts/verify-change.sh', 'origin/main',
  ]), {
    environment: { VERIFY_LEVEL: 'l2' },
    script: 'scripts/verify-change.sh',
    arguments: ['origin/main'],
  });
  assert.throws(() => parseRunArguments(['--env', 'VERIFY_LEVEL']), /KEY=VALUE/);
  assert.throws(() => parseRunArguments(['--env', 'GIT_DIR=foreign', 'script.sh']), /only accepts/);
  assert.throws(() => parseRunArguments(['--env', 'VERIFY_LEVEL=unknown', 'script.sh']), /only accepts/);
  assert.throws(() => parseRunArguments([
    '--env', 'VERIFY_LEVEL=l2', '--env', 'VERIFY_LEVEL=l3', 'script.sh',
  ]), /duplicate/);
});

test('launcher removes caller Git repository overrides without dropping unrelated variables', () => {
  const environment = cleanShellEnvironment({
    GIT_DIR: 'foreign',
    GIT_WORK_TREE: 'foreign',
    GIT_INDEX_FILE: 'foreign',
    GIT_COMMON_DIR: 'foreign',
    GIT_OBJECT_DIRECTORY: 'foreign',
    GIT_ALTERNATE_OBJECT_DIRECTORIES: 'foreign',
    PATH: 'kept',
  });
  assert.equal(environment.PATH, 'kept');
  for (const key of Object.keys(environment)) assert.equal(key.startsWith('GIT_'), false);
});
