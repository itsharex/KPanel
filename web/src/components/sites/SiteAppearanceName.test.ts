// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import SiteAppearanceName from './SiteAppearanceName.vue'

vi.mock('@/lib/api', () => ({
  api: {
    sites: {
      appearance: vi.fn(),
    },
  },
}))

beforeEach(() => {
  vi.clearAllMocks()
})

describe('SiteAppearanceName', () => {
  it('renders the fetched website name as a note and refreshes with the list', async () => {
    vi.mocked(api.sites.appearance)
      .mockResolvedValueOnce({ name: '  科技狮网站  ' })
      .mockResolvedValueOnce({ name: '科技狮新站名' })
    const wrapper = mount(SiteAppearanceName, {
      props: { siteId: 'a'.repeat(32), refreshKey: 0 },
    })
    await flushPromises()

    expect(wrapper.get('small.site-appearance-name').text()).toBe('科技狮网站')
    expect(wrapper.get('small').attributes('title')).toBe('科技狮网站')

    await wrapper.setProps({ refreshKey: 1 })
    await flushPromises()
    expect(wrapper.text()).toBe('科技狮新站名')
    expect(api.sites.appearance).toHaveBeenCalledTimes(2)
  })

  it('stays hidden when appearance metadata is unavailable', async () => {
    vi.mocked(api.sites.appearance).mockRejectedValueOnce(new Error('offline'))
    const wrapper = mount(SiteAppearanceName, {
      props: { siteId: 'b'.repeat(32), refreshKey: 0 },
    })
    await flushPromises()

    expect(wrapper.find('small').exists()).toBe(false)
  })
})
