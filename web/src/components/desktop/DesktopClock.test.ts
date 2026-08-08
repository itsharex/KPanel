// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import DesktopClock from '@/components/desktop/DesktopClock.vue'
import { resetLocaleForTest } from '@/i18n'

describe('DesktopClock', () => {
  afterEach(() => {
    vi.useRealTimers()
    resetLocaleForTest()
    vi.clearAllMocks()
  })

  it('renders the local time as HH:MM:SS', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-07T14:23:07'))
    const wrapper = mount(DesktopClock)
    await nextTick()
    expect(wrapper.find('.desktop-clock__time').text()).toBe('14:23:07')
    wrapper.unmount()
  })

  it('renders a non-empty local date line', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-07T14:23:07'))
    const wrapper = mount(DesktopClock)
    await nextTick()
    expect(wrapper.find('.desktop-clock__date').text().length).toBeGreaterThan(0)
    wrapper.unmount()
  })

  it('updates the local time every second', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-07T14:23:07'))
    const wrapper = mount(DesktopClock)
    await nextTick()
    expect(wrapper.find('.desktop-clock__time').text()).toBe('14:23:07')
    vi.advanceTimersByTime(1000)
    await nextTick()
    expect(wrapper.find('.desktop-clock__time').text()).toBe('14:23:08')
    wrapper.unmount()
  })

  it('keeps the system timezone independent from the public-IP location', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-07T06:23:07Z'))
    const wrapper = mount(DesktopClock, {
      props: {
        systemTimezone: 'Asia/Seoul',
        network: {
          timezone: 'America/New_York',
          country: '美国',
          countryCode: 'US',
          city: '纽约',
        },
      },
    })
    await nextTick()
    expect(wrapper.find('.desktop-clock__server-time').text()).toBe('15:23:07')
    expect(wrapper.find('.desktop-clock__server-location').text()).toContain('美国 · 纽约')
    expect(wrapper.find('.desktop-clock__server-timezone').text()).toBe('Asia/Seoul')
    wrapper.unmount()
  })

  it('shows the country flag and location beneath the server time', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-07T06:23:07Z'))
    const wrapper = mount(DesktopClock, {
      props: {
        systemTimezone: 'Asia/Shanghai',
        network: {
          timezone: 'Asia/Shanghai',
          country: '中国',
          countryCode: 'CN',
          city: '上海',
        },
      },
    })
    await vi.waitFor(() => {
      expect(wrapper.find('.desktop-clock__server-location .country-flag').exists()).toBe(true)
    })
    expect(wrapper.find('.desktop-clock__header').text()).toContain('本地时间')
    expect(wrapper.find('.desktop-clock__server-label').text()).toBe('服务器时间')
    const timezone = wrapper.find('.desktop-clock__server-timezone').text()
    expect(timezone).toBe('Asia/Shanghai')
    expect(timezone).not.toContain('中国')
    expect(wrapper.find('.desktop-clock__server-location').text()).toContain('中国 · 上海')
    wrapper.unmount()
  })

  it('hides the server section when server data is unavailable', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-07T06:23:07Z'))
    const wrapper = mount(DesktopClock, { props: { network: {} } })
    await nextTick()
    expect(wrapper.find('.desktop-clock__time').text()).not.toBe('--:--:--')
    expect(wrapper.find('.desktop-clock__server').exists()).toBe(false)
    wrapper.unmount()
  })

  it('clears its interval on unmount', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-07T14:23:07'))
    const clearSpy = vi.spyOn(window, 'clearInterval')
    const wrapper = mount(DesktopClock)
    wrapper.unmount()
    expect(clearSpy).toHaveBeenCalled()
    clearSpy.mockRestore()
  })
})
