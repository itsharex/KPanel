#!/usr/bin/env node

import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const RELEASE_TAG = /^v\d+\.\d+\.\d+$/;
const EMPTY_VALUE = /^(?:|[-—]|待填写|未记录|未验证|未知|不知道|稍后|不适用|N\/A)$/i;

export function isStableReleaseTag(value) {
  return RELEASE_TAG.test(value);
}

export function parseArguments(argv) {
  const options = {
    days: 14,
    releases: 20,
    format: 'markdown',
    repo: process.cwd(),
    ref: null,
    now: new Date(),
    acceptanceFiles: [],
  };

  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    const value = argv[index + 1];
    if (argument === '--validate-acceptance') {
      if (value === undefined) throw new Error('Missing value for ' + argument);
      options.acceptanceFiles.push(value);
      index += 1;
      continue;
    }
    if (argument === '--days' || argument === '--releases' || argument === '--format' || argument === '--repo' || argument === '--ref' || argument === '--now') {
      if (value === undefined) {
        throw new Error('Missing value for ' + argument);
      }
      index += 1;
      if (argument === '--days') options.days = Number(value);
      if (argument === '--releases') options.releases = Number(value);
      if (argument === '--format') options.format = value;
      if (argument === '--repo') options.repo = value;
      if (argument === '--ref') options.ref = value;
      if (argument === '--now') options.now = new Date(value);
      continue;
    }
    if (argument === '--help' || argument === '-h') {
      options.help = true;
      continue;
    }
    throw new Error('Unknown argument: ' + argument);
  }

  if (!Number.isInteger(options.days) || options.days < 1) throw new Error('--days must be a positive integer');
  if (!Number.isInteger(options.releases) || options.releases < 1) throw new Error('--releases must be a positive integer');
  if (!['markdown', 'json'].includes(options.format)) throw new Error('--format must be markdown or json');
  if (Number.isNaN(options.now.getTime())) throw new Error('--now must be a valid date');
  options.repo = resolve(options.repo);
  options.acceptanceFiles = options.acceptanceFiles.map((path) => resolve(options.repo, path));
  return options;
}

function runGit(repo, arguments_) {
  return execFileSync('git', ['-C', repo, ...arguments_], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  }).trim();
}

function tryGit(repo, arguments_) {
  try {
    return runGit(repo, arguments_);
  } catch {
    return null;
  }
}

function validValue(value) {
  const normalized = (value ?? '').trim();
  return EMPTY_VALUE.test(normalized) ? null : normalized;
}

function normalizedFieldName(value) {
  return value.trim().normalize('NFKC');
}

function comparableFieldText(value) {
  return value.replace(DEFAULT_IGNORABLE_GLOBAL, '').normalize('NFKC').replace(/\p{White_Space}/gu, '');
}

function suspiciousFieldName(line) {
  const comparableLine = comparableFieldText(line);
  return ACCEPTANCE_FIELDS.find((field) => {
    const fieldName = comparableFieldText(field);
    const index = comparableLine.indexOf(fieldName);
    return index >= 0 && /^[:=﹕∶]/u.test(comparableLine.slice(index + fieldName.length));
  });
}

const DEFAULT_IGNORABLE = /\p{Default_Ignorable_Code_Point}/u;
const DEFAULT_IGNORABLE_GLOBAL = /\p{Default_Ignorable_Code_Point}/gu;

function withoutHtmlComments(markdown) {
  return markdown.replace(/<!--[\s\S]*?(?:-->|$)/g, (comment) => comment.replace(/[^\r\n]/g, ''));
}

function acceptanceFields(markdown) {
  const fields = new Map();
  const duplicates = new Set();
  const malformed = new Set();
  let defaultIgnorables = false;
  let fence = null;
  for (const line of withoutHtmlComments(markdown).split(/\r?\n/)) {
    if (fence !== null) {
      const closingFence = line.match(/^ {0,3}(`{3,}|~{3,})[ \t]*$/)?.[1];
      if (closingFence?.[0] === fence.character && closingFence.length >= fence.length) fence = null;
      continue;
    }
    const openingFence = line.match(/^ {0,3}(`{3,}|~{3,})(.*)$/);
    if (openingFence && !(openingFence[1][0] === '`' && openingFence[2].includes('`'))) {
      fence = { character: openingFence[1][0], length: openingFence[1].length };
      continue;
    }

    const visibleBullet = line.match(/^ {0,3}-[ \t]+(.*)$/)?.[1];
    const suspiciousField = suspiciousFieldName(line);
    if (suspiciousField && DEFAULT_IGNORABLE.test(line)) defaultIgnorables = true;
    if (visibleBullet === undefined) {
      if (suspiciousField) malformed.add(normalizedFieldName(suspiciousField));
      continue;
    }

    const bullet = visibleBullet;
    const canonicalBullet = comparableFieldText(bullet);
    const expectedField = ACCEPTANCE_FIELDS.find((field) =>
      canonicalBullet.startsWith(comparableFieldText(field)));
    const match = bullet.match(/^([^：:]+)[：:]\s*(.*)$/);
    if (match) {
      const field = normalizedFieldName(match[1]);
      if (fields.has(field)) duplicates.add(field);
      fields.set(field, match[2].trim());
      if (expectedField && field !== normalizedFieldName(expectedField)) {
        malformed.add(normalizedFieldName(expectedField));
      }
    } else if (expectedField) {
      malformed.add(normalizedFieldName(expectedField));
    }
  }
  return { fields, duplicates, malformed, defaultIgnorables };
}

export function extractAcceptanceMetrics(markdown) {
  const { fields } = acceptanceFields(markdown);
  const field = (name) => fields.get(normalizedFieldName(name));

  return {
    firstIncludedCommitAt: validValue(field('首个纳入提交时间')),
    candidateFrozenAt: validValue(field('候选冻结时间')),
    productionCompletedAt: validValue(field('生产完成时间')),
    commitToProduction: validValue(field('提交到生产用时')),
    changeFailure: validValue(field('是否回滚、紧急热修复或重复发布')),
    recovery: validValue(field('若发生失败，发现时间、恢复时间和逃逸门禁')),
  };
}

const ACCEPTANCE_FIELDS = [
  '首个纳入提交时间',
  '候选冻结时间',
  '生产完成时间',
  '提交到生产用时',
  '是否回滚、紧急热修复或重复发布',
  '若发生失败，发现时间、恢复时间和逃逸门禁',
];

const ISO_TIMESTAMP = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,3})?(?:Z|[+-](\d{2}):(\d{2}))$/;

function validDate(value) {
  if (value === null) return false;
  const match = value.match(ISO_TIMESTAMP);
  if (!match) return false;
  const [, year, month, day, hour, minute, second, offsetHour = '00', offsetMinute = '00'] = match;
  const parts = [year, month, day, hour, minute, second, offsetHour, offsetMinute].map(Number);
  const [y, mo, d, h, mi, s, oh, om] = parts;
  if (mo < 1 || mo > 12 || d < 1 || h > 23 || mi > 59 || s > 59 || oh > 23 || om > 59) return false;
  const calendar = new Date(Date.UTC(y, mo - 1, d, h, mi, s));
  if (calendar.getUTCFullYear() !== y || calendar.getUTCMonth() !== mo - 1 || calendar.getUTCDate() !== d ||
      calendar.getUTCHours() !== h || calendar.getUTCMinutes() !== mi || calendar.getUTCSeconds() !== s) return false;
  return !Number.isNaN(Date.parse(value));
}

function hasFailureRecoveryDetails(value) {
  if (value === null || DEFAULT_IGNORABLE.test(value)) return false;
  const normalizedValue = value.normalize('NFKC');
  const segments = normalizedValue.split(';').map((segment) => segment.trim());
  if (segments.length !== 3) return false;
  const detail = (segment, label) => validValue(segment.match(
    new RegExp('^' + label + '\\s*:\\s*(.+)$'),
  )?.[1]);
  const discoveredAt = detail(segments[0], '发现时间');
  const recoveredAt = detail(segments[1], '恢复时间');
  const escapedGate = detail(segments[2], '逃逸门禁');
  const escapedGateMatch = escapedGate?.match(/^(?:已逃逸|未逃逸)\s*:\s*(.+)$/);
  const escapedGateDetail = validValue(escapedGateMatch?.[1]);
  const compactGateDetail = escapedGateDetail?.toLowerCase().replace(/[\p{White_Space}\p{P}\p{S}]+/gu, '') ?? '';
  const escapedGatePlaceholder = /^(?:无|无(?:缺口|门禁|异常|问题)|待.*|tbd|todo|none|null|unknown|na|notapplicable|具体(?:缺口|原因))$/i;
  const concreteEscapedGate = escapedGateDetail !== null && escapedGateDetail.length >= 4 &&
    !escapedGatePlaceholder.test(compactGateDetail) &&
    !/待(?:进一步)?(?:确认|分析|调查|复核|补充|填写|定)|请填写|<[^>]*>|已逃逸|未逃逸|逃逸门禁/.test(escapedGateDetail);
  return validDate(discoveredAt) && validDate(recoveredAt) && concreteEscapedGate &&
    Date.parse(recoveredAt) >= Date.parse(discoveredAt);
}

export function validateAcceptanceMetrics(markdown, label = 'acceptance record') {
  const errors = [];
  const { fields, duplicates, malformed, defaultIgnorables } = acceptanceFields(markdown);
  if (defaultIgnorables) {
    errors.push(label + ': structured acceptance evidence must not contain default-ignorable characters');
  }
  for (const field of ACCEPTANCE_FIELDS) {
    const normalizedField = normalizedFieldName(field);
    if (!fields.has(normalizedField)) errors.push(label + ': missing structured field "' + field + '"');
    if (duplicates.has(normalizedField)) errors.push(label + ': duplicate structured field "' + field + '"');
    if (malformed.has(normalizedField)) errors.push(label + ': malformed structured field "' + field + '"');
  }
  if (errors.length > 0) return errors;

  const metrics = extractAcceptanceMetrics(markdown);
  if (metrics.firstIncludedCommitAt !== null && !validDate(metrics.firstIncludedCommitAt)) {
    errors.push(label + ': 首个纳入提交时间 must be an ISO timestamp or an explicit unverified marker');
  }
  if (metrics.candidateFrozenAt !== null && !validDate(metrics.candidateFrozenAt)) {
    errors.push(label + ': 候选冻结时间 must be an ISO timestamp or an explicit unverified marker');
  }
  if (metrics.productionCompletedAt !== null && !validDate(metrics.productionCompletedAt)) {
    errors.push(label + ': 生产完成时间 must be an ISO timestamp or an explicit unverified marker');
  }

  if (validDate(metrics.firstIncludedCommitAt) && validDate(metrics.candidateFrozenAt) &&
      new Date(metrics.candidateFrozenAt) < new Date(metrics.firstIncludedCommitAt)) {
    errors.push(label + ': 候选冻结时间 cannot precede 首个纳入提交时间');
  }

  if (metrics.productionCompletedAt === null) {
    if (metrics.commitToProduction !== null) {
      errors.push(label + ': 提交到生产用时 must stay unreported when production completion is unverified');
    }
    const failure = classifyChangeFailure(metrics.changeFailure);
    if (failure === 'yes' && !hasFailureRecoveryDetails(metrics.recovery)) {
      errors.push(label + ': a failed change requires discovery, recovery, and escaped-gate details');
    }
    if (failure !== 'yes' && metrics.recovery !== null) {
      errors.push(label + ': recovery details require an explicit failed change state');
    }
    return errors;
  }

  if (!validDate(metrics.firstIncludedCommitAt) || !validDate(metrics.candidateFrozenAt)) {
    errors.push(label + ': a completed production deployment requires explicit included-commit and candidate-freeze timestamps');
  }

  if (validDate(metrics.candidateFrozenAt) &&
      new Date(metrics.productionCompletedAt) < new Date(metrics.candidateFrozenAt)) {
    errors.push(label + ': 生产完成时间 cannot precede 候选冻结时间');
  }

  const derivedHours = durationHours(metrics.firstIncludedCommitAt, metrics.productionCompletedAt);
  const reportedHours = metrics.commitToProduction?.match(/^(\d+(?:\.\d+)?)\s*(?:小时|h(?:ours?)?)?$/i);
  if (derivedHours === null || !reportedHours) {
    errors.push(label + ': 提交到生产用时 must be a decimal hour value when production completed');
  } else if (Math.abs(derivedHours - Number(reportedHours[1])) > 0.02) {
    errors.push(label + ': 提交到生产用时 does not match the structured timestamps');
  }

  const failure = classifyChangeFailure(metrics.changeFailure);
  if (failure === 'unreported') {
    errors.push(label + ': production completion requires an explicit 是/否 change failure state');
  }
  if (failure === 'yes' && !hasFailureRecoveryDetails(metrics.recovery)) {
    errors.push(label + ': a failed change requires discovery, recovery, and escaped-gate details');
  }
  if (failure !== 'yes' && metrics.recovery !== null) {
    errors.push(label + ': recovery details require an explicit failed change state');
  }
  return errors;
}

export function classifyChangeFailure(value) {
  if (!value) return 'unreported';
  if (/^否(?:\b|（|\(|$)/.test(value)) return 'no';
  if (/^是(?:\b|（|\(|$)/.test(value)) return 'yes';
  return 'unreported';
}

export function durationHours(startValue, endValue) {
  if (!startValue || !endValue) return null;
  const start = new Date(startValue);
  const end = new Date(endValue);
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime()) || end < start) return null;
  return Number(((end - start) / 3_600_000).toFixed(2));
}

export function productionLeadHours(metrics) {
  const derived = durationHours(metrics.firstIncludedCommitAt, metrics.productionCompletedAt);
  if (derived !== null) return derived;
  if (!metrics.commitToProduction) return null;
  const match = metrics.commitToProduction.match(/^(\d+(?:\.\d+)?)\s*(?:小时|h(?:ours?)?)?$/i);
  return match ? Number(match[1]) : null;
}

export function summarizeReleaseMetrics(releases, options) {
  const cutoff = new Date(options.now.getTime() - options.days * 86_400_000);
  const windowReleases = releases.filter((release) => release.createdAt >= cutoff && release.createdAt <= options.now);
  const releasesPerDay = new Map();
  for (const release of windowReleases) {
    const day = release.createdAt.toISOString().slice(0, 10);
    releasesPerDay.set(day, (releasesPerDay.get(day) ?? 0) + 1);
  }

  const selected = releases.slice(0, options.releases);
  const acceptanceCount = selected.filter((release) => release.acceptance.exists).length;
  const failureStates = selected.map((release) => classifyChangeFailure(release.acceptance.metrics.changeFailure));
  const reportedFailureCount = failureStates.filter((state) => state !== 'unreported').length;
  const failedReleaseCount = failureStates.filter((state) => state === 'yes').length;
  const leadTimes = selected
    .map((release) => productionLeadHours(release.acceptance.metrics))
    .filter((value) => value !== null);
  const freezeTimes = selected
    .map((release) => durationHours(release.acceptance.metrics.candidateFrozenAt, release.acceptance.metrics.productionCompletedAt))
    .filter((value) => value !== null);

  return {
    generatedAt: options.now.toISOString(),
    source: {
      ref: options.evidenceRef ?? null,
      commit: options.evidenceCommit ?? null,
    },
    window: {
      days: options.days,
      releaseCount: windowReleases.length,
      releaseDays: releasesPerDay.size,
      maxReleasesPerDay: Math.max(0, ...releasesPerDay.values()),
    },
    recent: {
      requested: options.releases,
      available: selected.length,
      acceptanceCount,
      acceptanceCoverage: selected.length === 0 ? null : Number((acceptanceCount / selected.length).toFixed(4)),
      productionLeadTimeReported: leadTimes.length,
      productionLeadTimeHoursMedian: median(leadTimes),
      freezeToProductionReported: freezeTimes.length,
      freezeToProductionHoursMedian: median(freezeTimes),
      changeFailureReported: reportedFailureCount,
      failedReleaseCount,
      changeFailureRate: reportedFailureCount === 0 ? null : Number((failedReleaseCount / reportedFailureCount).toFixed(4)),
      recoveryReported: selected.filter((release) => release.acceptance.metrics.recovery !== null).length,
    },
    releases: selected,
  };
}

function median(values) {
  if (values.length === 0) return null;
  const sorted = [...values].sort((left, right) => left - right);
  const middle = Math.floor(sorted.length / 2);
  if (sorted.length % 2 === 1) return sorted[middle];
  return Number(((sorted[middle - 1] + sorted[middle]) / 2).toFixed(2));
}

function collectReleases(repo, evidenceCommit, acceptanceLimit) {
  const mergedTags = new Set(
    runGit(repo, ['tag', '--merged', evidenceCommit, '--list', 'v*']).split(/\r?\n/).filter(Boolean),
  );
  const output = runGit(repo, ['for-each-ref', '--sort=-creatordate', '--format=%(refname:short)%09%(creatordate:iso-strict)', 'refs/tags']);
  if (!output) return [];

  return output
    .split(/\r?\n/)
    .map((line) => {
      const [tag, created] = line.split('\t');
      return { tag, createdAt: new Date(created) };
    })
    .filter((release) => mergedTags.has(release.tag) && isStableReleaseTag(release.tag) && !Number.isNaN(release.createdAt.getTime()))
    .map((release, index) => {
      const acceptancePath = 'docs/release-' + release.tag + '-acceptance.md';
      const markdown = index < acceptanceLimit ? tryGit(repo, ['show', evidenceCommit + ':' + acceptancePath]) : null;
      return {
        tag: release.tag,
        createdAt: release.createdAt,
        commit: runGit(repo, ['rev-parse', release.tag + '^{commit}']),
        acceptance: {
          exists: markdown !== null,
          path: acceptancePath,
          metrics: extractAcceptanceMetrics(markdown ?? ''),
        },
      };
    });
}

function percentage(value) {
  return value === null ? '未报告' : (value * 100).toFixed(1) + '%';
}

function metric(value, suffix = '') {
  return value === null ? '未报告' : String(value) + suffix;
}

function failureLabel(value) {
  return { yes: '是', no: '否', unreported: '未报告' }[classifyChangeFailure(value)];
}

export function renderMarkdown(report) {
  const lines = [
    '# KPanel 滚动发布指标',
    '',
    '- 生成时间：' + report.generatedAt,
    '- 证据引用：' + report.source.ref + ' @ ' + report.source.commit,
    '- 窗口：最近 ' + report.window.days + ' 天、最近 ' + report.recent.requested + ' 个正式版本',
    '- 数据来源：Git 标签与版本验收记录；未记录的生产事实保持“未报告”',
    '',
    '## 自动可验证指标',
    '',
    '| 指标 | 结果 |',
    '| --- | --- |',
    '| 窗口内正式版本数 | ' + report.window.releaseCount + ' |',
    '| 有发布的自然日 | ' + report.window.releaseDays + ' |',
    '| 单日最大发布数 | ' + report.window.maxReleasesPerDay + ' |',
    '| 验收记录覆盖率 | ' + report.recent.acceptanceCount + '/' + report.recent.available + '（' + percentage(report.recent.acceptanceCoverage) + '） |',
    '',
    '## 生产交付指标',
    '',
    '| 指标 | 结果 | 数据完整性 |',
    '| --- | --- | --- |',
    '| 首个纳入提交到生产完成中位数 | ' + metric(report.recent.productionLeadTimeHoursMedian, ' 小时') + ' | ' + report.recent.productionLeadTimeReported + '/' + report.recent.available + ' |',
    '| 候选冻结到生产完成中位数 | ' + metric(report.recent.freezeToProductionHoursMedian, ' 小时') + ' | ' + report.recent.freezeToProductionReported + '/' + report.recent.available + ' |',
    '| 变更失败率 | ' + percentage(report.recent.changeFailureRate) + ' | ' + report.recent.changeFailureReported + '/' + report.recent.available + ' |',
    '| 已报告失败恢复详情 | ' + report.recent.recoveryReported + ' | ' + report.recent.recoveryReported + '/' + report.recent.available + ' |',
    '',
    '## 最近版本证据',
    '',
    '| 标签 | 标签时间 | 验收记录 | 提交到生产 | 失败状态 |',
    '| --- | --- | --- | --- | --- |',
  ];

  for (const release of report.releases) {
    const leadTime = productionLeadHours(release.acceptance.metrics);
    lines.push('| ' + release.tag + ' | ' + release.createdAt.toISOString() + ' | ' +
      (release.acceptance.exists ? release.acceptance.path : '缺失') + ' | ' + metric(leadTime, ' 小时') + ' | ' +
      failureLabel(release.acceptance.metrics.changeFailure) + ' |');
  }

  lines.push('', '> 标签时间不等于生产完成时间；变更失败率只以明确填报“是/否”的验收记录为分母，不把缺失数据推断为成功。');
  return lines.join('\n');
}

function help() {
  return [
    'Usage: node scripts/report-release-metrics.mjs [options]',
    '  --days <n>       Rolling calendar window (default: 14)',
    '  --releases <n>   Number of recent release tags (default: 20)',
    '  --format <type>  markdown or json (default: markdown)',
    '  --repo <path>    Repository root (default: current directory)',
    '  --ref <ref>      Evidence ref (default: origin/main, fallback: HEAD)',
    '  --now <date>     Override current time for reproducible checks',
    '  --validate-acceptance <path>  Validate one release record; repeat for multiple files',
  ].join('\n');
}

export function main(argv) {
  const options = parseArguments(argv);
  if (options.help) {
    process.stdout.write(help() + '\n');
    return;
  }
  if (options.acceptanceFiles.length > 0) {
    const errors = options.acceptanceFiles.flatMap((path) =>
      validateAcceptanceMetrics(readFileSync(path, 'utf8'), path));
    if (errors.length > 0) throw new Error('\n- ' + errors.join('\n- '));
    process.stdout.write('Release acceptance metrics validation passed (' + options.acceptanceFiles.length + ' file(s)).\n');
    return;
  }
  runGit(options.repo, ['rev-parse', '--is-inside-work-tree']);
  options.evidenceRef = options.ref ?? (tryGit(options.repo, ['rev-parse', '--verify', 'origin/main^{commit}']) ? 'origin/main' : 'HEAD');
  options.evidenceCommit = runGit(options.repo, ['rev-parse', '--verify', options.evidenceRef + '^{commit}']);
  const report = summarizeReleaseMetrics(collectReleases(options.repo, options.evidenceCommit, options.releases), options);
  process.stdout.write((options.format === 'json' ? JSON.stringify(report, null, 2) : renderMarkdown(report)) + '\n');
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    main(process.argv.slice(2));
  } catch (error) {
    process.stderr.write('release metrics failed: ' + error.message + '\n');
    process.exitCode = 1;
  }
}
