import { describe, expect, it } from 'vitest'
import { detectOperatingSystemIdentity } from './operatingSystem'

describe('operating system identity', () => {
  it.each([
    {
      name: 'Rocky stays Rocky despite CentOS compatibility',
      input: {
        os: 'Rocky Linux 9.7 (Blue Onyx)',
        osId: 'rocky',
        osLike: ['rhel', 'centos', 'fedora'],
      },
      expected: { key: 'rocky', label: 'Rocky Linux' },
    },
    {
      name: 'AlmaLinux stays AlmaLinux',
      input: {
        os: 'AlmaLinux 9.6 (Sage Margay)',
        osId: 'almalinux',
        osLike: ['rhel', 'centos', 'fedora'],
      },
      expected: { key: 'alma', label: 'AlmaLinux' },
    },
    {
      name: 'RHEL does not become Fedora',
      input: {
        os: 'Red Hat Enterprise Linux 9.6 (Plow)',
        osId: 'rhel',
        osLike: ['fedora'],
      },
      expected: { key: 'rhel', label: 'Red Hat Enterprise Linux' },
    },
    {
      name: 'Oracle Linux uses its own identity',
      input: {
        os: 'Oracle Linux Server 9.6',
        osId: 'ol',
        osLike: ['fedora'],
      },
      expected: { key: 'oracle', label: 'Oracle Linux' },
    },
    {
      name: 'Manjaro does not become Arch',
      input: {
        os: 'Manjaro Linux',
        osId: 'manjaro',
        osLike: ['arch'],
      },
      expected: { key: 'manjaro', label: 'Manjaro' },
    },
    {
      name: 'openSUSE has a distinct identity',
      input: {
        os: 'openSUSE Leap 15.6',
        osId: 'opensuse-leap',
        osLike: ['suse', 'opensuse'],
      },
      expected: { key: 'opensuse', label: 'openSUSE' },
    },
    {
      name: 'SLES does not use the openSUSE brand',
      input: {
        os: 'SUSE Linux Enterprise Server 15 SP6',
        osId: 'sles',
        osLike: ['suse'],
      },
      expected: { key: 'suse', label: 'SUSE Linux Enterprise' },
    },
    {
      name: 'Ubuntu remains exact before Debian compatibility',
      input: {
        os: 'Ubuntu 24.04.2 LTS',
        osId: 'ubuntu',
        osLike: ['debian'],
      },
      expected: { key: 'ubuntu', label: 'Ubuntu' },
    },
  ])('$name', ({ input, expected }) => {
    expect(detectOperatingSystemIdentity(input)).toEqual(expected)
  })

  it('never treats an unknown derivative compatibility family as its brand', () => {
    expect(
      detectOperatingSystemIdentity({
        os: 'Vendor Linux 1',
        osId: 'vendorlinux',
        osLike: ['ubuntu', 'debian', 'fedora', 'centos', 'arch'],
      }),
    ).toEqual({ key: 'linux', label: 'Vendor Linux 1' })
  })

  it('uses the precise system name when osId is unavailable', () => {
    expect(
      detectOperatingSystemIdentity({
        os: 'Rocky Linux 9.7 (Blue Onyx)',
        osLike: ['centos'],
      }),
    ).toEqual({ key: 'rocky', label: 'Rocky Linux' })
  })
})
