import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const readView = (name: string) => readFileSync(new URL(`./${name}.vue`, import.meta.url), 'utf8')

const views = [
  'AppsView',
  'AuditView',
  'ClusterView',
  'DiagnosticsView',
  'DockerView',
  'EnvironmentView',
  'FilesView',
  'JobsView',
  'MonitoringView',
  'OverviewView',
  'ProcessManagerView',
  'SettingsView',
  'SitesView',
  'TerminalView',
]

describe('page-level action placement', () => {
  it('keeps route actions out of the shared page heading', () => {
    for (const view of views) expect(readView(view)).not.toContain('<template #actions>')
  })

  it('places primary actions beside the content they affect', () => {
    expect(readView('AppsView')).toContain('class="market-hero__actions"')
    expect(readView('ClusterView')).toContain('class="cluster-hero__actions"')
    expect(readView('AppsView')).toContain('class="market-stats"')
    expect(readView('ClusterView')).toContain('class="cluster-stats"')
    expect(readView('SitesView')).toContain('class="page-command-bar"')
    expect(readView('EnvironmentView')).toContain('class="page-command-bar"')
    expect(readView('DockerView')).toContain('class="docker-command-center__actions"')
    expect(readView('FilesView')).toContain('class="file-command-bar__actions"')
    expect(readView('ProcessManagerView')).toContain('class="process-toolbar__controls"')
  })

  it('keeps refresh controls inside each page toolbar or workspace header', () => {
    expect(readView('AuditView')).toContain('aria-label="刷新审计记录"')
    expect(readView('JobsView')).toContain('aria-label="刷新变更记录"')
    expect(readView('MonitoringView')).toContain('aria-label="刷新监控数据"')
    expect(readView('OverviewView')).toContain('aria-label="刷新系统状态"')
    expect(readView('SettingsView')).toContain('aria-label="检查 Agent 连接"')
    expect(readView('DiagnosticsView')).not.toContain('aria-label="刷新体检命令"')
    expect(readView('TerminalView')).toContain(":aria-label=\"t('terminal.refreshConnections')\"")
  })
})
