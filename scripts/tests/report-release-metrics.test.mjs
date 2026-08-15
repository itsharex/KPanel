import test from 'node:test';
import assert from 'node:assert/strict';

import {
  classifyChangeFailure,
  durationHours,
  extractAcceptanceMetrics,
  isStableReleaseTag,
  parseArguments,
  productionLeadHours,
  renderMarkdown,
  summarizeReleaseMetrics,
  validateAcceptanceMetrics,
} from '../report-release-metrics.mjs';

function release(tag, createdAt, acceptance = {}) {
  return {
    tag,
    createdAt: new Date(createdAt),
    commit: tag + '-commit',
    acceptance: {
      exists: acceptance.exists ?? true,
      path: 'docs/release-' + tag + '-acceptance.md',
      metrics: {
        firstIncludedCommitAt: null,
        candidateFrozenAt: null,
        productionCompletedAt: null,
        commitToProduction: null,
        changeFailure: null,
        recovery: null,
        ...acceptance.metrics,
      },
    },
  };
}

test('extractAcceptanceMetrics keeps absent production evidence unreported', () => {
  const metrics = extractAcceptanceMetrics([
    '## 交付节奏数据',
    '- 首个纳入提交时间：2026-08-10T07:00:00+08:00',
    '- 候选冻结时间：2026-08-10T08:00:00+08:00',
    '- 生产完成时间：未验证',
    '- 提交到生产用时：',
    '- 是否回滚、紧急热修复或重复发布：否',
    '- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用',
  ].join('\n'));

  assert.equal(metrics.firstIncludedCommitAt, '2026-08-10T07:00:00+08:00');
  assert.equal(metrics.candidateFrozenAt, '2026-08-10T08:00:00+08:00');
  assert.equal(metrics.productionCompletedAt, null);
  assert.equal(metrics.commitToProduction, null);
  assert.equal(metrics.changeFailure, '否');
  assert.equal(metrics.recovery, null);
});

test('durationHours accepts explicit timestamps and rejects invalid intervals', () => {
  assert.equal(durationHours('2026-08-10T08:00:00Z', '2026-08-10T10:30:00Z'), 2.5);
  assert.equal(durationHours('2026-08-10T10:30:00Z', '2026-08-10T08:00:00Z'), null);
  assert.equal(durationHours('未记录', '2026-08-10T08:00:00Z'), null);
});

test('productionLeadHours derives timestamps and accepts standardized hour fallback', () => {
  assert.equal(productionLeadHours({
    firstIncludedCommitAt: '2026-08-10T07:00:00Z',
    productionCompletedAt: '2026-08-10T10:30:00Z',
    commitToProduction: '99 小时',
  }), 3.5);
  assert.equal(productionLeadHours({
    firstIncludedCommitAt: null,
    productionCompletedAt: null,
    commitToProduction: '2.75 小时',
  }), 2.75);
  assert.equal(productionLeadHours({ commitToProduction: '未记录' }), null);
});

test('summarizeReleaseMetrics never treats missing failure data as success', () => {
  const report = summarizeReleaseMetrics([
    release('v1.2.0', '2026-08-10T12:00:00Z', {
      metrics: {
        firstIncludedCommitAt: '2026-08-10T06:00:00Z',
        candidateFrozenAt: '2026-08-10T08:00:00Z',
        productionCompletedAt: '2026-08-10T10:00:00Z',
        changeFailure: '是（已回滚）',
        recovery: '10:05 发现，10:20 恢复',
      },
    }),
    release('v1.1.0', '2026-08-09T12:00:00Z', { metrics: { changeFailure: '否' } }),
    release('v1.0.0', '2026-07-01T12:00:00Z', { exists: false }),
  ], {
    days: 14,
    releases: 3,
    now: new Date('2026-08-11T00:00:00Z'),
  });

  assert.equal(report.window.releaseCount, 2);
  assert.equal(report.recent.acceptanceCount, 2);
  assert.equal(report.recent.changeFailureReported, 2);
  assert.equal(report.recent.failedReleaseCount, 1);
  assert.equal(report.recent.changeFailureRate, 0.5);
  assert.equal(report.recent.productionLeadTimeHoursMedian, 4);
  assert.equal(report.recent.freezeToProductionHoursMedian, 2);
  assert.equal(report.recent.recoveryReported, 1);
});

test('markdown output discloses evidence completeness', () => {
  const report = summarizeReleaseMetrics([
    release('v1.0.0', '2026-08-10T12:00:00Z', { exists: false }),
  ], {
    days: 14,
    releases: 1,
    now: new Date('2026-08-11T00:00:00Z'),
  });
  const output = renderMarkdown(report);
  assert.match(output, /验收记录覆盖率 \| 0\/1/);
  assert.match(output, /变更失败率 \| 未报告/);
  assert.match(output, /不把缺失数据推断为成功/);
});

test('argument parser rejects invalid windows', () => {
  assert.throws(() => parseArguments(['--days', '0']), /positive integer/);
  assert.throws(() => parseArguments(['--format', 'csv']), /markdown or json/);
  assert.equal(parseArguments(['--ref', 'v1.2.3']).ref, 'v1.2.3');
  assert.equal(classifyChangeFailure('否（未发生）'), 'no');
  assert.equal(classifyChangeFailure('是（已回滚）'), 'yes');
  assert.equal(classifyChangeFailure(null), 'unreported');
  assert.equal(isStableReleaseTag('v1.2.3'), true);
  for (const tag of ['v1.2.3-rc.1', 'v1.2.3+build.1', 'v1.2.3-nightly', 'vfoo', 'v1', 'v1.2']) {
    assert.equal(isStableReleaseTag(tag), false, tag);
  }
});

test('acceptance validation requires machine-readable delivery evidence', () => {
  const valid = [
    '## 交付节奏数据',
    '- 首个纳入提交时间：2026-08-15T11:21:11+08:00',
    '- 候选冻结时间：2026-08-15T11:26:43+08:00',
    '- 生产完成时间：2026-08-15T12:00:39+08:00',
    '- 提交到生产用时：0.66 小时',
    '- 是否回滚、紧急热修复或重复发布：否',
    '- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用',
  ].join('\n');
  assert.deepEqual(validateAcceptanceMetrics(valid), []);

  const missing = validateAcceptanceMetrics('## 交付节奏数据');
  assert.equal(missing.length, 6);

  const inconsistent = validateAcceptanceMetrics(valid.replace('0.66 小时', '2.00 小时'));
  assert.match(inconsistent.join('\n'), /does not match/);

  const looseDate = validateAcceptanceMetrics(valid.replace(
    '2026-08-15T11:21:11+08:00',
    'August 15, 2026 11:21:11 GMT+0800',
  ));
  assert.match(looseDate.join('\n'), /must be an ISO timestamp/);

  const impossibleDate = validateAcceptanceMetrics(valid.replace(
    '2026-08-15T11:21:11+08:00',
    '2026-02-30T11:21:11+08:00',
  ));
  assert.match(impossibleDate.join('\n'), /must be an ISO timestamp/);
});

test('acceptance validation permits explicit non-production evidence without inventing success', () => {
  const errors = validateAcceptanceMetrics([
    '## 交付节奏数据',
    '- 首个纳入提交时间：2026-08-15T11:21:11+08:00',
    '- 候选冻结时间：2026-08-15T11:26:43+08:00',
    '- 生产完成时间：未验证',
    '- 提交到生产用时：未验证',
    '- 是否回滚、紧急热修复或重复发布：未验证',
    '- 若发生失败，发现时间、恢复时间和逃逸门禁：未验证',
  ].join('\n'));
  assert.deepEqual(errors, []);

  const historicalUnknown = validateAcceptanceMetrics([
    '## 交付节奏数据',
    '- 首个纳入提交时间：未记录',
    '- 候选冻结时间：未记录',
    '- 生产完成时间：未记录',
    '- 提交到生产用时：未记录',
    '- 是否回滚、紧急热修复或重复发布：未记录',
    '- 若发生失败，发现时间、恢复时间和逃逸门禁：未记录',
  ].join('\n'));
  assert.deepEqual(historicalUnknown, []);
});

test('acceptance validation keeps known failure state when historical completion time is missing', () => {
  const knownSuccess = validateAcceptanceMetrics([
    '## 交付节奏数据',
    '- 首个纳入提交时间：2026-08-15T14:04:08+08:00',
    '- 候选冻结时间：2026-08-15T14:37:23+08:00',
    '- 生产完成时间：未记录',
    '- 提交到生产用时：未记录',
    '- 是否回滚、紧急热修复或重复发布：否',
    '- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用',
  ].join('\n'));
  assert.deepEqual(knownSuccess, []);

  const failedWithoutRecovery = validateAcceptanceMetrics([
    '## 交付节奏数据',
    '- 首个纳入提交时间：未记录',
    '- 候选冻结时间：未记录',
    '- 生产完成时间：未记录',
    '- 提交到生产用时：未记录',
    '- 是否回滚、紧急热修复或重复发布：是',
    '- 若发生失败，发现时间、恢复时间和逃逸门禁：已回滚',
  ].join('\n'));
  assert.match(failedWithoutRecovery.join('\n'), /requires discovery, recovery/);

  const failedWithDetails = validateAcceptanceMetrics([
    '## 交付节奏数据',
    '- 首个纳入提交时间：未记录',
    '- 候选冻结时间：未记录',
    '- 生产完成时间：未记录',
    '- 提交到生产用时：未记录',
    '- 是否回滚、紧急热修复或重复发布：是',
    '- 若发生失败，发现时间、恢复时间和逃逸门禁：发现时间：2026-08-15T10:05:00+08:00；恢复时间：2026-08-15T10:20:00+08:00；逃逸门禁：候选冻结后缺少回归',
  ].join('\n'));
  assert.deepEqual(failedWithDetails, []);

  const unknownDetails = validateAcceptanceMetrics([
    '## 交付节奏数据',
    '- 首个纳入提交时间：未记录',
    '- 候选冻结时间：未记录',
    '- 生产完成时间：未记录',
    '- 提交到生产用时：未记录',
    '- 是否回滚、紧急热修复或重复发布：是',
    '- 若发生失败，发现时间、恢复时间和逃逸门禁：发现时间：不知道；恢复时间：稍后；逃逸门禁：未知',
  ].join('\n'));
  assert.match(unknownDetails.join('\n'), /requires discovery, recovery/);

  const reversedRecovery = validateAcceptanceMetrics([
    '## 交付节奏数据',
    '- 首个纳入提交时间：未记录',
    '- 候选冻结时间：未记录',
    '- 生产完成时间：未记录',
    '- 提交到生产用时：未记录',
    '- 是否回滚、紧急热修复或重复发布：是',
    '- 若发生失败，发现时间、恢复时间和逃逸门禁：发现时间：2026-08-15T13:00:00+08:00；恢复时间：2026-08-15T12:30:00+08:00；逃逸门禁：候选冻结后缺少回归',
  ].join('\n'));
  assert.match(reversedRecovery.join('\n'), /requires discovery, recovery/);

  const keywordsWithoutStructure = validateAcceptanceMetrics([
    '## 交付节奏数据',
    '- 首个纳入提交时间：未记录',
    '- 候选冻结时间：未记录',
    '- 生产完成时间：未记录',
    '- 提交到生产用时：未记录',
    '- 是否回滚、紧急热修复或重复发布：是',
    '- 若发生失败，发现时间、恢复时间和逃逸门禁：已发现并恢复，复查逃逸门禁',
  ].join('\n'));
  assert.match(keywordsWithoutStructure.join('\n'), /requires discovery, recovery/);
});
