import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { describe, expect, it } from 'vitest'
import TerminalToolbar from './TerminalToolbar.vue'

describe('TerminalToolbar', () => {
  it('keeps jump-to-top before the fullscreen action', async () => {
    const html = await renderToString(createSSRApp(TerminalToolbar, { fullscreen: false }))

    expect(html.indexOf('回到顶部')).toBeGreaterThanOrEqual(0)
    expect(html.indexOf('铺满网页')).toBeGreaterThan(html.indexOf('回到顶部'))
  })

  it('changes fullscreen into the restore action', async () => {
    const html = await renderToString(createSSRApp(TerminalToolbar, { fullscreen: true }))

    expect(html).toContain('恢复窗口')
    expect(html).not.toContain('铺满网页')
  })
})
