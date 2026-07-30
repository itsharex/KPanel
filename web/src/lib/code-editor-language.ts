import type { Extension } from '@codemirror/state'

export const CODE_HIGHLIGHT_MAX_BYTES = 1024 * 1024

export interface CodeLanguage {
  id: string
  label: string
  highlighted: boolean
  extension?: Extension
  reason?: 'unsupported' | 'large-file'
}

interface DetectedLanguage {
  id: string
  label: string
}

const extensionLanguages: Record<string, DetectedLanguage> = {
  bash: { id: 'shell', label: 'Shell' },
  cjs: { id: 'javascript', label: 'JavaScript' },
  conf: { id: 'properties', label: '配置文件' },
  css: { id: 'css', label: 'CSS' },
  env: { id: 'properties', label: '环境变量' },
  go: { id: 'go', label: 'Go' },
  htm: { id: 'html', label: 'HTML' },
  html: { id: 'html', label: 'HTML' },
  ini: { id: 'properties', label: 'INI' },
  js: { id: 'javascript', label: 'JavaScript' },
  json: { id: 'json', label: 'JSON' },
  jsonc: { id: 'json', label: 'JSON' },
  jsx: { id: 'jsx', label: 'JSX' },
  md: { id: 'markdown', label: 'Markdown' },
  mjs: { id: 'javascript', label: 'JavaScript' },
  nginx: { id: 'nginx', label: 'Nginx' },
  php: { id: 'php', label: 'PHP' },
  properties: { id: 'properties', label: 'Properties' },
  py: { id: 'python', label: 'Python' },
  sh: { id: 'shell', label: 'Shell' },
  sql: { id: 'sql', label: 'SQL' },
  ts: { id: 'typescript', label: 'TypeScript' },
  tsx: { id: 'tsx', label: 'TSX' },
  vue: { id: 'html', label: 'Vue/HTML' },
  xml: { id: 'xml', label: 'XML' },
  yaml: { id: 'yaml', label: 'YAML' },
  yml: { id: 'yaml', label: 'YAML' },
  zsh: { id: 'shell', label: 'Shell' },
}

const mimeLanguages: Array<[RegExp, DetectedLanguage]> = [
  [/javascript|ecmascript/i, { id: 'javascript', label: 'JavaScript' }],
  [/typescript/i, { id: 'typescript', label: 'TypeScript' }],
  [/json/i, { id: 'json', label: 'JSON' }],
  [/html/i, { id: 'html', label: 'HTML' }],
  [/css/i, { id: 'css', label: 'CSS' }],
  [/markdown/i, { id: 'markdown', label: 'Markdown' }],
  [/python/i, { id: 'python', label: 'Python' }],
  [/ya?ml/i, { id: 'yaml', label: 'YAML' }],
  [/php/i, { id: 'php', label: 'PHP' }],
  [/sql/i, { id: 'sql', label: 'SQL' }],
  [/\bgo\b/i, { id: 'go', label: 'Go' }],
  [/xml/i, { id: 'xml', label: 'XML' }],
  [/shell|bash|x-sh/i, { id: 'shell', label: 'Shell' }],
]

export function detectCodeLanguage(fileName: string, mime = ''): DetectedLanguage | undefined {
  const normalized = fileName.trim().toLocaleLowerCase()
  if (/^dockerfile(?:\.|$)/.test(normalized)) return { id: 'dockerfile', label: 'Dockerfile' }
  if (normalized === 'nginx.conf' || normalized.startsWith('nginx.')) {
    return { id: 'nginx', label: 'Nginx' }
  }
  const extension = normalized.includes('.') ? normalized.split('.').pop() || '' : ''
  if (extensionLanguages[extension]) return extensionLanguages[extension]
  return mimeLanguages.find(([pattern]) => pattern.test(mime))?.[1]
}

export async function loadCodeLanguage(
  fileName: string,
  mime: string | undefined,
  sizeBytes: number,
): Promise<CodeLanguage> {
  const detected = detectCodeLanguage(fileName, mime)
  if (sizeBytes > CODE_HIGHLIGHT_MAX_BYTES) {
    return {
      id: detected?.id || 'plain-text',
      label: detected?.label || '纯文本',
      highlighted: false,
      reason: 'large-file',
    }
  }
  if (!detected) {
    return {
      id: 'plain-text',
      label: '纯文本',
      highlighted: false,
      reason: 'unsupported',
    }
  }

  let extension: Extension
  switch (detected.id) {
    case 'javascript':
      extension = (await import('@codemirror/lang-javascript')).javascript()
      break
    case 'jsx':
      extension = (await import('@codemirror/lang-javascript')).javascript({ jsx: true })
      break
    case 'typescript':
      extension = (await import('@codemirror/lang-javascript')).javascript({ typescript: true })
      break
    case 'tsx':
      extension = (await import('@codemirror/lang-javascript')).javascript({
        jsx: true,
        typescript: true,
      })
      break
    case 'html':
      extension = (await import('@codemirror/lang-html')).html()
      break
    case 'css':
      extension = (await import('@codemirror/lang-css')).css()
      break
    case 'json':
      extension = (await import('@codemirror/lang-json')).json()
      break
    case 'markdown':
      extension = (await import('@codemirror/lang-markdown')).markdown()
      break
    case 'python':
      extension = (await import('@codemirror/lang-python')).python()
      break
    case 'yaml':
      extension = (await import('@codemirror/lang-yaml')).yaml()
      break
    case 'php':
      extension = (await import('@codemirror/lang-php')).php()
      break
    case 'sql':
      extension = (await import('@codemirror/lang-sql')).sql()
      break
    case 'go':
      extension = (await import('@codemirror/lang-go')).go()
      break
    case 'xml':
      extension = (await import('@codemirror/lang-xml')).xml()
      break
    case 'shell': {
      const [{ StreamLanguage }, { shell }] = await Promise.all([
        import('@codemirror/language'),
        import('@codemirror/legacy-modes/mode/shell'),
      ])
      extension = StreamLanguage.define(shell)
      break
    }
    case 'nginx': {
      const [{ StreamLanguage }, { nginx }] = await Promise.all([
        import('@codemirror/language'),
        import('@codemirror/legacy-modes/mode/nginx'),
      ])
      extension = StreamLanguage.define(nginx)
      break
    }
    case 'dockerfile': {
      const [{ StreamLanguage }, { dockerFile }] = await Promise.all([
        import('@codemirror/language'),
        import('@codemirror/legacy-modes/mode/dockerfile'),
      ])
      extension = StreamLanguage.define(dockerFile)
      break
    }
    case 'properties': {
      const [{ StreamLanguage }, { properties }] = await Promise.all([
        import('@codemirror/language'),
        import('@codemirror/legacy-modes/mode/properties'),
      ])
      extension = StreamLanguage.define(properties)
      break
    }
    default:
      return {
        id: 'plain-text',
        label: '纯文本',
        highlighted: false,
        reason: 'unsupported',
      }
  }
  return { ...detected, highlighted: true, extension }
}
