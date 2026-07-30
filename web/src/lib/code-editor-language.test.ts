import { describe, expect, it } from 'vitest'
import {
  CODE_HIGHLIGHT_MAX_BYTES,
  detectCodeLanguage,
  loadCodeLanguage,
} from '@/lib/code-editor-language'

describe('code editor language loading', () => {
  it('detects common file names and MIME fallbacks', () => {
    expect(detectCodeLanguage('app.tsx')).toMatchObject({ id: 'tsx', label: 'TSX' })
    expect(detectCodeLanguage('Dockerfile.production')).toMatchObject({
      id: 'dockerfile',
      label: 'Dockerfile',
    })
    expect(detectCodeLanguage('nginx.conf')).toMatchObject({ id: 'nginx', label: 'Nginx' })
    expect(detectCodeLanguage('script', 'text/x-shellscript')).toMatchObject({
      id: 'shell',
      label: 'Shell',
    })
  })

  it('uses plain text without loading a parser for unsupported files', async () => {
    await expect(loadCodeLanguage('README.unknown', 'text/plain', 1024)).resolves.toMatchObject({
      id: 'plain-text',
      highlighted: false,
      reason: 'unsupported',
    })
  })

  it('disables syntax parsing above the large-file threshold', async () => {
    await expect(
      loadCodeLanguage('large.js', 'text/javascript', CODE_HIGHLIGHT_MAX_BYTES + 1),
    ).resolves.toMatchObject({
      id: 'javascript',
      highlighted: false,
      reason: 'large-file',
    })
  })

  it.each([
    ['app.js', 'javascript'],
    ['page.html', 'html'],
    ['theme.css', 'css'],
    ['settings.json', 'json'],
    ['README.md', 'markdown'],
    ['worker.py', 'python'],
    ['compose.yaml', 'yaml'],
    ['index.php', 'php'],
    ['schema.sql', 'sql'],
    ['main.go', 'go'],
    ['feed.xml', 'xml'],
    ['deploy.sh', 'shell'],
    ['nginx.conf', 'nginx'],
    ['Dockerfile', 'dockerfile'],
    ['service.ini', 'properties'],
  ])('loads syntax support for %s', async (fileName, languageId) => {
    const result = await loadCodeLanguage(fileName, '', 1024)
    expect(result).toMatchObject({ id: languageId, highlighted: true })
    expect(result.extension).toBeDefined()
  })
})
