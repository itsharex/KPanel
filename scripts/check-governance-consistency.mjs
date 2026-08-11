#!/usr/bin/env node

import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const repoRoot = resolve(fileURLToPath(new URL('..', import.meta.url)));
const failures = [];

const requiredFiles = [
  'PROJECT_RULES.md',
  'AGENTS.md',
  'CLAUDE.md',
  'docs/project-management.md',
  'docs/multi-agent-collaboration.md',
  'docs/development-quality-standard.md',
  'docs/release-acceptance-template.md',
  'docs/quality-improvement-proposal-template.md',
  'scripts/report-release-metrics.mjs',
  'scripts/tests/report-release-metrics.test.mjs',
  '.codex-workflows/README.md',
  '.codex-workflows/session-collaboration.workflow.yaml',
  '.codex-workflows/release-kpanel.workflow.yaml',
  '.codex-workflows/quality-audit-kpanel.workflow.yaml',
  '.codex-workflows/evolve-kpanel.workflow.yaml',
];

function read(relativePath) {
  const absolutePath = resolve(repoRoot, relativePath);
  if (!existsSync(absolutePath)) {
    failures.push(relativePath + ': required file is missing');
    return '';
  }
  return readFileSync(absolutePath, 'utf8');
}

function requireText(relativePath, tokens) {
  const content = read(relativePath);
  for (const token of tokens) {
    if (!content.includes(token)) failures.push(relativePath + ': missing required reference "' + token + '"');
  }
}

function checkRepositoryHygiene() {
  const trackedPaths = execFileSync(
    'git',
    ['ls-files', '--cached', '--others', '--exclude-standard', '-z'],
    {
      cwd: repoRoot,
      encoding: 'utf8',
    },
  ).split('\0').filter(Boolean);
  const forbiddenTrackedPaths = [
    /(^|\/)\.codex-tmp\//i,
    /(^|\/)node_modules\//i,
    /(^|\/)web\/dist\//i,
    /(^|\/)coverage\//i,
  ];
  const forbiddenContent = [
    {
      pattern: /[A-Za-z]:[\\/]+Users[\\/]+[^\\/\s"'`<>]+[\\/]+/i,
      label: 'machine-specific Windows user path',
    },
    {
      pattern: /AppData[\\/]+Local[\\/]+Temp/i,
      label: 'local temporary attachment path',
    },
    {
      pattern: new RegExp('codex-' + 'clipboard', 'i'),
      label: 'temporary clipboard attachment marker',
    },
    {
      pattern: new RegExp('<codex_' + 'delegation|<source_' + 'thread_id', 'i'),
      label: 'Codex session delegation envelope',
    },
  ];

  for (const relativePath of trackedPaths) {
    const normalizedPath = relativePath.replaceAll('\\', '/');
    if (forbiddenTrackedPaths.some((pattern) => pattern.test(normalizedPath))) {
      failures.push(relativePath + ': generated or temporary path must not be tracked');
      continue;
    }

    const raw = readFileSync(resolve(repoRoot, relativePath));
    if (raw.includes(0)) continue;
    const content = raw.toString('utf8');
    for (const rule of forbiddenContent) {
      const match = rule.pattern.exec(content);
      if (!match) continue;
      const line = content.slice(0, match.index).split('\n').length;
      failures.push(relativePath + ':' + line + ': contains ' + rule.label);
    }
  }
}

for (const relativePath of requiredFiles) read(relativePath);
checkRepositoryHygiene();

const adapterTokens = [
  'PROJECT_RULES.md',
  'docs/project-management.md',
  'docs/multi-agent-collaboration.md',
  'Definition of Ready',
  'Definition of Done',
  'make verify-change',
  'make verify-release',
];
requireText('AGENTS.md', adapterTokens);
requireText('CLAUDE.md', adapterTokens);

requireText('PROJECT_RULES.md', [
  'docs/development-quality-standard.md',
  'docs/release-acceptance-template.md',
  '受控自我改进',
  '不得自动放宽',
  '公共默认',
]);
requireText('docs/project-management.md', [
  'Definition of Ready',
  'Definition of Done',
  '标准交付包',
  '受控自我改进循环',
  'docs/quality-improvement-proposal-template.md',
  '.codex-workflows/evolve-kpanel.workflow.yaml',
]);
requireText('.codex-workflows/README.md', ['evolve-kpanel.workflow.yaml']);

requireText('docs/release-acceptance-template.md', [
  '## 发布画像',
  '## 多维质量结论',
  '## 自动门禁',
  '## 隔离真机与浏览器验收',
  '## 生产部署与观察',
  '## 回滚',
  '## 交付节奏数据',
  '首个纳入提交时间',
  '公共默认更新通道决策',
  '## 遗留风险与后续准入',
]);
requireText('docs/quality-improvement-proposal-template.md', [
  '## 观察证据',
  '## 原因假设',
  '## 基线、目标与观察窗口',
  '## 最小改动方案',
  '## 独立复核',
  '## 回滚',
  '## 采纳决策与结果',
]);

const workflows = [
  '.codex-workflows/session-collaboration.workflow.yaml',
  '.codex-workflows/release-kpanel.workflow.yaml',
  '.codex-workflows/quality-audit-kpanel.workflow.yaml',
  '.codex-workflows/evolve-kpanel.workflow.yaml',
];
for (const workflow of workflows) {
  const content = read(workflow);
  for (const key of ['name:', 'description:', 'version:', 'params:', 'updated:']) {
    if (!content.includes('\n' + key)) failures.push(workflow + ': missing frontmatter key "' + key.slice(0, -1) + '"');
  }
  for (const heading of ['## Purpose', '## Prerequisites', '## Steps', '## Verification', '## Notes']) {
    if (!content.includes(heading)) failures.push(workflow + ': missing section "' + heading + '"');
  }
}

for (const adapter of ['AGENTS.md', 'CLAUDE.md']) {
  const content = read(adapter);
  if (content.includes('# KPanel 永久工程规范')) {
    failures.push(adapter + ': tool adapter must not duplicate PROJECT_RULES.md');
  }
}

if (failures.length > 0) {
  process.stderr.write('Governance consistency check failed:\n');
  for (const failure of [...new Set(failures)]) process.stderr.write('- ' + failure + '\n');
  process.exit(1);
}

process.stdout.write('Governance consistency check passed.\n');
