#!/usr/bin/env node

import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const DEFAULT_MAX_COMMITS = 50;
const DEFAULT_MAX_RELEASES = 8;
const DEFAULT_MIN_COMMITS_WITH_RELEASES = 20;

function runGit(repo, arguments_) {
  return execFileSync('git', ['-C', repo, ...arguments_], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  }).trim();
}

export function extractBusinessContextMetadata(markdown) {
  const match = (label) => markdown.match(new RegExp('^- ' + label + '：`([^`]+)`$', 'm'))?.[1] ?? null;
  return {
    reviewedAt: match('复核日期'),
    baselineCommit: match('基线提交'),
    baselineVersion: match('基线版本'),
  };
}

export function evaluateFreshness({
  commitCount,
  releaseCount,
  maxCommits = DEFAULT_MAX_COMMITS,
  maxReleases = DEFAULT_MAX_RELEASES,
  minCommitsWithReleases = DEFAULT_MIN_COMMITS_WITH_RELEASES,
}) {
  const reasons = [];
  if (commitCount >= maxCommits) reasons.push('commits=' + commitCount + '>=' + maxCommits);
  if (commitCount >= minCommitsWithReleases && releaseCount >= maxReleases) {
    reasons.push('commits=' + commitCount + '>=' + minCommitsWithReleases +
      ' and releases=' + releaseCount + '>=' + maxReleases);
  }
  return { stale: reasons.length > 0, reasons };
}

function releaseCountAfter(repo, baseline, target) {
  const commits = new Set(runGit(repo, ['rev-list', baseline + '..' + target]).split(/\r?\n/).filter(Boolean));
  const tags = runGit(repo, [
    'for-each-ref',
    '--merged=' + target,
    '--format=%(refname:short)%09%(*objectname)%09%(objectname)',
    'refs/tags/v*',
  ]).split(/\r?\n/).filter(Boolean);
  return tags.filter((line) => {
    const [, peeled, object] = line.split('\t');
    return commits.has(peeled || object);
  }).length;
}

export function checkBusinessContext({ repo, document, target = 'HEAD', maxCommits = DEFAULT_MAX_COMMITS, maxReleases = DEFAULT_MAX_RELEASES }) {
  const markdown = readFileSync(document, 'utf8');
  const metadata = extractBusinessContextMetadata(markdown);
  if (!metadata.reviewedAt || !/^\d{4}-\d{2}-\d{2}$/.test(metadata.reviewedAt)) {
    throw new Error('current business context is missing a valid 复核日期');
  }
  if (!metadata.baselineCommit) throw new Error('current business context is missing 基线提交');
  if (!metadata.baselineVersion || !/^v\d+\.\d+\.\d+$/.test(metadata.baselineVersion)) {
    throw new Error('current business context is missing a semantic 基线版本');
  }

  const baseline = runGit(repo, ['rev-parse', '--verify', metadata.baselineCommit + '^{commit}']);
  const targetCommit = runGit(repo, ['rev-parse', '--verify', target + '^{commit}']);
  try {
    execFileSync('git', ['-C', repo, 'merge-base', '--is-ancestor', baseline, targetCommit], { stdio: 'ignore' });
  } catch {
    throw new Error('business context baseline is not an ancestor of ' + target);
  }
  const describedVersion = runGit(repo, ['describe', '--tags', '--abbrev=0', baseline]);
  if (describedVersion !== metadata.baselineVersion) {
    throw new Error('business context baseline version mismatch: expected ' + describedVersion + ', recorded ' + metadata.baselineVersion);
  }

  const commitCount = Number(runGit(repo, ['rev-list', '--count', baseline + '..' + targetCommit]));
  const releaseCount = releaseCountAfter(repo, baseline, targetCommit);
  const freshness = evaluateFreshness({ commitCount, releaseCount, maxCommits, maxReleases });
  return { ...metadata, baseline, targetCommit, commitCount, releaseCount, ...freshness };
}

function main() {
  const repo = resolve(fileURLToPath(new URL('..', import.meta.url)));
  const document = resolve(repo, 'docs/product-quality-review-current.md');
  const result = checkBusinessContext({ repo, document });
  if (result.stale) {
    throw new Error('business context review is stale (' + result.reasons.join(', ') + '); refresh the canonical review before continuing');
  }
  process.stdout.write('Business context freshness passed: baseline=' + result.baseline.slice(0, 7) +
    ' commits=' + result.commitCount + ' releases=' + result.releaseCount + '.\n');
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    main();
  } catch (error) {
    process.stderr.write('Business context freshness failed: ' + error.message + '\n');
    process.exitCode = 1;
  }
}
