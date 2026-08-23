#!/usr/bin/env node

import process from 'node:process';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const SHA_PATTERN = /^[0-9a-f]{40}$/;
const REPOSITORY_PATTERN = /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const CANDIDATE_BRANCH_PATTERN = /^(docs|feature|fix|release)\//;

export function workflowRunsUrl(repository, sha) {
  if (!REPOSITORY_PATTERN.test(repository)) throw new Error('GITHUB_REPOSITORY must use owner/name');
  if (!SHA_PATTERN.test(sha)) throw new Error('GITHUB_SHA must be a full lowercase commit SHA');
  return 'https://api.github.com/repos/' + repository +
    '/actions/workflows/ci.yml/runs?event=push&head_sha=' + sha + '&per_page=100';
}

export function successfulCandidateRun(runs, sha, currentRunId = '') {
  return runs.find((run) =>
    String(run.id) !== String(currentRunId) &&
    run.event === 'push' &&
    run.status === 'completed' &&
    run.conclusion === 'success' &&
    run.head_sha === sha &&
    typeof run.head_branch === 'string' &&
    CANDIDATE_BRANCH_PATTERN.test(run.head_branch),
  );
}

export async function verifyGovernanceCandidateCi({ environment = process.env, fetchImpl = fetch } = {}) {
  if (environment.CI !== 'true' || environment.GITHUB_REF_NAME !== 'main') {
    return { status: 'skipped', reason: 'not-main-ci' };
  }

  const sha = environment.GITHUB_SHA || '';
  const repository = environment.GITHUB_REPOSITORY || '';
  const token = environment.GITHUB_TOKEN || '';
  if (!token) throw new Error('GITHUB_TOKEN is required for the governance candidate CI check');

  const response = await fetchImpl(workflowRunsUrl(repository, sha), {
    headers: {
      Accept: 'application/vnd.github+json',
      Authorization: 'Bearer ' + token,
      'User-Agent': 'KPanel-governance-candidate-gate',
      'X-GitHub-Api-Version': '2022-11-28',
    },
  });
  if (!response.ok) {
    throw new Error('GitHub Actions candidate lookup failed with HTTP ' + response.status);
  }

  const payload = await response.json();
  const runs = Array.isArray(payload.workflow_runs) ? payload.workflow_runs : [];
  const match = successfulCandidateRun(runs, sha, environment.GITHUB_RUN_ID || '');
  if (!match) {
    throw new Error(
      'governance changes require the exact main SHA to pass CI on a docs/, feature/, fix/, or release/ candidate branch first',
    );
  }

  return {
    status: 'passed',
    branch: match.head_branch,
    runId: match.id,
    url: match.html_url,
  };
}

const isMainModule = process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url);
if (isMainModule) {
  try {
    const result = await verifyGovernanceCandidateCi();
    if (result.status === 'skipped') {
      process.stdout.write('governance_candidate_ci=skipped reason=' + result.reason + '\n');
    } else {
      process.stdout.write(
        'governance_candidate_ci=pass branch=' + result.branch + ' run_id=' + result.runId + ' url=' + result.url + '\n',
      );
    }
  } catch (error) {
    process.stderr.write('governance_candidate_ci=fail ' + error.message + '\n');
    process.exitCode = 1;
  }
}
