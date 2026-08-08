import { createSSRApp, h } from 'vue'
import { renderToString, type SSRContext } from 'vue/server-renderer'
import { describe, expect, it } from 'vitest'
import ModalDialog from './ModalDialog.vue'

describe('ModalDialog server rendering', () => {
  it('renders an open dialog without browser globals', async () => {
    expect(typeof document).toBe('undefined')
    expect(typeof window).toBe('undefined')

    const context: SSRContext = {}
    await expect(
      renderToString(
        createSSRApp({
          render: () => h(ModalDialog, { open: true, title: 'Server dialog' }),
        }),
        context,
      ),
    ).resolves.toContain('teleport start')
  })
})
