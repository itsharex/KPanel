import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import test from 'node:test';

import {
  successfulCandidateRun,
  verifyGovernanceCandidateCi,
  workflowRunsUrl,
} from '../check-governance-candidate-ci.mjs';

const sha = 'a'.repeat(40);
const scriptSource = readFileSync(resolve(import.meta.dirname, '..', 'check-governance-candidate-ci.mjs'), 'utf8');

function response(workflowRuns, { ok = true, status = 200 } = {}) {
  return {
    ok,
    status,
    async json() {
      return { workflow_runs: workflowRuns };
    },
  };
}

test('workflow run lookup is fixed to the CI workflow and exact SHA', () => {
  assert.equal(
    workflowRunsUrl('kejilion/KPanel', sha),
    'https://api.github.com/repos/kejilion/KPanel/actions/workflows/ci.yml/runs?event=push&head_sha=' + sha + '&per_page=100',
  );
  assert.throws(() => workflowRunsUrl('invalid', sha), /owner\/name/);
  assert.throws(() => workflowRunsUrl('kejilion/KPanel', 'short'), /full lowercase commit SHA/);
});

test('only a completed successful non-main candidate run for the exact SHA is accepted', () => {
  const accepted = {
    id: 10,
    event: 'push',
    status: 'completed',
    conclusion: 'success',
    head_sha: sha,
    head_branch: 'docs/governance-candidate',
    html_url: 'https://example.test/run/10',
  };
  const rejected = [
    { ...accepted, id: 1, head_branch: 'main' },
    { ...accepted, id: 2, head_sha: 'b'.repeat(40) },
    { ...accepted, id: 3, conclusion: 'failure' },
    { ...accepted, id: 4, status: 'in_progress', conclusion: null },
    { ...accepted, id: 5, head_branch: 'topic/not-ci-enabled' },
  ];
  assert.equal(successfulCandidateRun([...rejected, accepted], sha, ''), accepted);
  assert.equal(successfulCandidateRun([accepted], sha, '10'), undefined);
});

test('main CI passes only when the API returns an exact successful candidate run', async () => {
  const environment = {
    CI: 'true',
    GITHUB_REF_NAME: 'main',
    GITHUB_SHA: sha,
    GITHUB_REPOSITORY: 'kejilion/KPanel',
    GITHUB_RUN_ID: '99',
    GITHUB_TOKEN: 'secret',
  };
  const run = {
    id: 10,
    event: 'push',
    status: 'completed',
    conclusion: 'success',
    head_sha: sha,
    head_branch: 'fix/governance-gate',
    html_url: 'https://example.test/run/10',
  };
  let authorization = '';
  const result = await verifyGovernanceCandidateCi({
    environment,
    fetchImpl: async (_url, options) => {
      authorization = options.headers.Authorization;
      return response([run]);
    },
  });
  assert.deepEqual(result, {
    status: 'passed',
    branch: 'fix/governance-gate',
    runId: 10,
    url: 'https://example.test/run/10',
  });
  assert.equal(authorization, 'Bearer secret');
});

test('missing candidate evidence and API errors fail closed', async () => {
  const environment = {
    CI: 'true',
    GITHUB_REF_NAME: 'main',
    GITHUB_SHA: sha,
    GITHUB_REPOSITORY: 'kejilion/KPanel',
    GITHUB_RUN_ID: '99',
    GITHUB_TOKEN: 'secret',
  };
  await assert.rejects(
    verifyGovernanceCandidateCi({ environment, fetchImpl: async () => response([]) }),
    /require the exact main SHA/,
  );
  await assert.rejects(
    verifyGovernanceCandidateCi({ environment, fetchImpl: async () => response([], { ok: false, status: 503 }) }),
    /HTTP 503/,
  );
});

test('local and candidate CI do not perform remote candidate lookup', async () => {
  let called = false;
  const result = await verifyGovernanceCandidateCi({
    environment: { CI: 'true', GITHUB_REF_NAME: 'docs/governance-candidate' },
    fetchImpl: async () => {
      called = true;
      return response([]);
    },
  });
  assert.deepEqual(result, { status: 'skipped', reason: 'not-main-ci' });
  assert.equal(called, false);
});

test('CLI failures defer process termination so network resources can close cleanly', () => {
  assert.match(scriptSource, /process\.exitCode = 1/);
  assert.doesNotMatch(scriptSource, /process\.exit\(1\)/);
});
