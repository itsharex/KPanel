import { describe, expect, it } from 'vitest'
import { shortcutFileGradient } from '@/lib/fileEntryPresentation'

describe('shortcutFileGradient', () => {
  it('keeps directories visually distinct from regular files', () => {
    expect(shortcutFileGradient('nginx', 'directory')).toContain('#facc15')
    expect(shortcutFileGradient('README.md', 'file')).toContain('#94a3b8')
  })

  it('uses restrained category colors for common file types', () => {
    expect(shortcutFileGradient('compose.yaml', 'file')).toContain('#38bdf8')
    expect(shortcutFileGradient('photo.webp', 'file')).toContain('#a78bfa')
    expect(shortcutFileGradient('report.xlsx', 'file')).toContain('#34d399')
    expect(shortcutFileGradient('backup.tar.gz', 'file')).toContain('#fb923c')
    expect(shortcutFileGradient('data.sqlite', 'file')).toContain('#2dd4bf')
    expect(shortcutFileGradient('server.pem', 'file')).toContain('#f87171')
  })

  it('uses the neutral fallback for unknown file types', () => {
    expect(shortcutFileGradient('artifact.unknown', 'file'))
      .toBe('linear-gradient(145deg, #94a3b8 0%, #475569 100%)')
  })
})
