import test from 'node:test';
import assert from 'node:assert/strict';

import {
  evaluateFreshness,
  extractBusinessContextMetadata,
} from '../check-business-context-freshness.mjs';

test('extractBusinessContextMetadata reads the canonical machine fields', () => {
  const metadata = extractBusinessContextMetadata([
    '- 复核日期：`2026-08-15`',
    '- 基线提交：`565e476623159247ec3ebb6967ab0d6753f165d1`',
    '- 基线版本：`v0.73.2`',
  ].join('\n'));
  assert.deepEqual(metadata, {
    reviewedAt: '2026-08-15',
    baselineCommit: '565e476623159247ec3ebb6967ab0d6753f165d1',
    baselineVersion: 'v0.73.2',
  });
});

test('freshness remains cheap until a meaningful change-volume threshold is reached', () => {
  assert.deepEqual(evaluateFreshness({ commitCount: 49, releaseCount: 7 }), { stale: false, reasons: [] });
  assert.equal(evaluateFreshness({ commitCount: 50, releaseCount: 0 }).stale, true);
  assert.equal(evaluateFreshness({ commitCount: 0, releaseCount: 8 }).stale, false);
  assert.equal(evaluateFreshness({ commitCount: 20, releaseCount: 8 }).stale, true);
});
