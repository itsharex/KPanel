import type {
  DockerContainerCreateEnvironment,
  DockerContainerCreateMount,
  DockerContainerCreatePort,
  DockerMaintenanceInput,
} from '@/types/api'

export const maxComposeSourceBytes = 24 * 1024

export type DockerDeploymentAnalysis =
  | { kind: 'empty' }
  | { kind: 'invalid'; message: string }
  | { kind: 'docker-run'; input: DockerMaintenanceInput }
  | { kind: 'compose'; compose: string; projectName: string; services: string[] }

interface TokenizeResult {
  tokens: string[]
  error?: string
}

export function analyzeDockerDeployment(source: string): DockerDeploymentAnalysis {
  const value = source.trim()
  if (!value) return { kind: 'empty' }
  if (new TextEncoder().encode(source).length > maxComposeSourceBytes) {
    return { kind: 'invalid', message: '部署内容不能超过 24 KiB。' }
  }
  if (looksLikeDockerRun(value)) return parseDockerRun(value)
  if (/^\s*(?:name\s*:|version\s*:|services\s*:)/m.test(value)) {
    const services = composeServices(value)
    if (!services.length) {
      return { kind: 'invalid', message: '没有识别到 Compose services，请粘贴完整的 Compose YAML。' }
    }
    const explicitName = value.match(/^name\s*:\s*["']?([a-z0-9][a-z0-9_-]{0,62})["']?\s*$/m)?.[1]
    return {
      kind: 'compose',
      compose: source,
      projectName: explicitName || normalizeProjectName(services[0] || 'stack'),
      services,
    }
  }
  if (/\bdocker\s+compose\b/.test(value)) {
    return { kind: 'invalid', message: '请粘贴 Compose YAML，而不是只有 docker compose up 命令。' }
  }
  return { kind: 'invalid', message: '请粘贴一条 docker run 命令，或完整的 Compose YAML。' }
}

function looksLikeDockerRun(value: string): boolean {
  return /^(?:sudo\s+)?docker\s+run(?:\s|$)/.test(value.replace(/\\\r?\n/g, ' '))
}

function parseDockerRun(source: string): DockerDeploymentAnalysis {
  const tokenized = tokenizeShell(source.replace(/\\\r?\n/g, ' '))
  if (tokenized.error) return { kind: 'invalid', message: tokenized.error }
  const tokens = tokenized.tokens
  if (tokens[0] === 'sudo') tokens.shift()
  if (tokens.shift() !== 'docker' || tokens.shift() !== 'run') {
    return { kind: 'invalid', message: '只支持单条 docker run 命令。' }
  }

  let name = ''
  let network = 'bridge'
  // Preserve Docker CLI semantics for pasted commands. The manual form keeps
  // KPanel's friendlier unless-stopped default.
  let restartPolicy: DockerMaintenanceInput['restartPolicy'] = 'no'
  const ports: DockerContainerCreatePort[] = []
  const mounts: DockerContainerCreateMount[] = []
  const environment: DockerContainerCreateEnvironment[] = []
  let image = ''
  const command: string[] = []

  const nextValue = (index: number, inline: string | undefined, option: string): [string, number] | string => {
    if (inline !== undefined) return [inline, index]
    const value = tokens[index + 1]
    if (!value || isShellOperator(value)) return `${option} 缺少参数。`
    return [value, index + 1]
  }

  for (let index = 0; index < tokens.length; index += 1) {
    const token = tokens[index]!
    if (isShellOperator(token)) {
      return { kind: 'invalid', message: '一次只能部署一条 docker run 命令，不能包含管道、重定向或后续 Shell 命令。' }
    }
    if (image) {
      command.push(token)
      continue
    }
    if (!token.startsWith('-')) {
      image = token
      continue
    }
    if (token === '-d' || token === '--detach') continue
    if (/^-[dit]+$/.test(token) && token.includes('d') && !token.includes('i') && !token.includes('t')) continue
    if (token === '-i' || token === '-t' || token === '-it' || token === '-ti' || token === '--interactive' || token === '--tty') {
      return { kind: 'invalid', message: '交互式 -it 容器不适合后台部署，请移除该参数或改用 Compose。' }
    }

    const [option, inline] = token.startsWith('--') && token.includes('=')
      ? [token.slice(0, token.indexOf('=')), token.slice(token.indexOf('=') + 1)]
      : [token, undefined]
    if (option === '--name') {
      const result = nextValue(index, inline, option)
      if (typeof result === 'string') return { kind: 'invalid', message: result }
      ;[name, index] = result
      continue
    }
    if (option === '--network' || option === '--net') {
      const result = nextValue(index, inline, option)
      if (typeof result === 'string') return { kind: 'invalid', message: result }
      ;[network, index] = result
      continue
    }
    if (option === '--restart') {
      const result = nextValue(index, inline, option)
      if (typeof result === 'string') return { kind: 'invalid', message: result }
      const [value, nextIndex] = result
      if (!['no', 'always', 'unless-stopped', 'on-failure'].includes(value)) {
        return { kind: 'invalid', message: `不支持的重启策略：${value}` }
      }
      restartPolicy = value as DockerMaintenanceInput['restartPolicy']
      index = nextIndex
      continue
    }
    if (option === '-p' || option === '--publish') {
      const result = nextValue(index, inline, option)
      if (typeof result === 'string') return { kind: 'invalid', message: result }
      const parsed = parsePort(result[0])
      if (typeof parsed === 'string') return { kind: 'invalid', message: parsed }
      ports.push(parsed)
      index = result[1]
      continue
    }
    if (option === '-v' || option === '--volume') {
      const result = nextValue(index, inline, option)
      if (typeof result === 'string') return { kind: 'invalid', message: result }
      const parsed = parseVolume(result[0])
      if (typeof parsed === 'string') return { kind: 'invalid', message: parsed }
      mounts.push(parsed)
      index = result[1]
      continue
    }
    if (option === '--mount') {
      const result = nextValue(index, inline, option)
      if (typeof result === 'string') return { kind: 'invalid', message: result }
      const parsed = parseMount(result[0])
      if (typeof parsed === 'string') return { kind: 'invalid', message: parsed }
      mounts.push(parsed)
      index = result[1]
      continue
    }
    if (option === '-e' || option === '--env') {
      const result = nextValue(index, inline, option)
      if (typeof result === 'string') return { kind: 'invalid', message: result }
      const separator = result[0].indexOf('=')
      if (separator <= 0) return { kind: 'invalid', message: `${option} 必须使用 NAME=VALUE。` }
      environment.push({ name: result[0].slice(0, separator), value: result[0].slice(separator + 1) })
      index = result[1]
      continue
    }
    if (option === '--pull') {
      const result = nextValue(index, inline, option)
      if (typeof result === 'string') return { kind: 'invalid', message: result }
      if (result[0] !== 'missing') {
        return { kind: 'invalid', message: '当前仅支持 Docker 默认的 --pull=missing；如需更新镜像请先在镜像页拉取。' }
      }
      index = result[1]
      continue
    }
    return { kind: 'invalid', message: `暂不支持 ${option}；复杂参数建议改用 Compose YAML。` }
  }

  if (!image) return { kind: 'invalid', message: 'docker run 命令缺少镜像。' }
  return {
    kind: 'docker-run',
    input: {
      action: 'container_create',
      name,
      image,
      network,
      restartPolicy,
      command,
      ports,
      mounts,
      environment,
    },
  }
}

function tokenizeShell(source: string): TokenizeResult {
  const tokens: string[] = []
  let current = ''
  let quote: "'" | '"' | '' = ''
  let escaped = false
  const flush = () => {
    if (current) tokens.push(current)
    current = ''
  }
  for (let index = 0; index < source.length; index += 1) {
    const char = source[index]!
    if (escaped) {
      current += char
      escaped = false
      continue
    }
    if (char === '\\' && quote !== "'") {
      escaped = true
      continue
    }
    if (quote) {
      if (char === quote) quote = ''
      else current += char
      continue
    }
    if (char === "'" || char === '"') {
      quote = char
      continue
    }
    if (/\s/.test(char)) {
      flush()
      continue
    }
    if (';|<>&'.includes(char)) {
      flush()
      const pair = source.slice(index, index + 2)
      if (pair === '&&' || pair === '||' || pair === '>>' || pair === '<<') {
        tokens.push(pair)
        index += 1
      } else tokens.push(char)
      continue
    }
    current += char
  }
  if (escaped) return { tokens, error: '命令末尾存在未完成的转义符。' }
  if (quote) return { tokens, error: '命令中存在未闭合的引号。' }
  flush()
  return { tokens }
}

function isShellOperator(value: string): boolean {
  return [';', '|', '||', '&', '&&', '>', '>>', '<', '<<'].includes(value)
}

function parsePort(value: string): DockerContainerCreatePort | string {
  let protocol: 'tcp' | 'udp' = 'tcp'
  if (value.endsWith('/udp')) {
    protocol = 'udp'
    value = value.slice(0, -4)
  } else if (value.endsWith('/tcp')) value = value.slice(0, -4)
  let hostIp = '0.0.0.0'
  let publicValue = ''
  let privateValue = ''
  if (value.startsWith('[')) {
    const end = value.indexOf(']:')
    if (end < 0) return `端口映射格式无效：${value}`
    hostIp = value.slice(1, end)
    const parts = value.slice(end + 2).split(':')
    publicValue = parts[0] || ''
    privateValue = parts[1] || ''
  } else {
    const parts = value.split(':')
    if (parts.length === 2) {
      publicValue = parts[0] || ''
      privateValue = parts[1] || ''
    } else if (parts.length === 3) {
      hostIp = parts[0] || ''
      publicValue = parts[1] || ''
      privateValue = parts[2] || ''
    }
    else return `端口映射必须使用 主机端口:容器端口：${value}`
  }
  const publicPort = Number(publicValue)
  const privatePort = Number(privateValue)
  if (![publicPort, privatePort].every((port) => Number.isInteger(port) && port >= 1 && port <= 65535)) {
    return `端口必须是 1-65535 的整数：${value}`
  }
  return { publicPort, privatePort, protocol, hostIp }
}

function parseVolume(value: string): DockerContainerCreateMount | string {
  const parts = value.split(':')
  if (parts.length < 2 || parts.length > 3) return `存储挂载格式无效：${value}`
  const [source, target, mode = 'rw'] = parts
  if (!source || !target?.startsWith('/')) return `存储挂载必须包含来源和容器绝对路径：${value}`
  if (!['ro', 'rw'].includes(mode)) return `暂不支持挂载模式 ${mode}，请改用 Compose。`
  return { type: source.startsWith('/') ? 'bind' : 'volume', source, target, readOnly: mode === 'ro' }
}

function parseMount(value: string): DockerContainerCreateMount | string {
  const options = new Map<string, string>()
  let readOnly = false
  for (const part of value.split(',')) {
    const [rawKey, ...rest] = part.split('=')
    const key = rawKey?.trim() || ''
    if (key === 'readonly' || key === 'ro') {
      readOnly = true
      continue
    }
    options.set(key, rest.join('=').trim())
  }
  const type = options.get('type') || 'volume'
  const source = options.get('source') || options.get('src') || ''
  const target = options.get('target') || options.get('dst') || options.get('destination') || ''
  if ((type !== 'bind' && type !== 'volume') || !source || !target.startsWith('/')) {
    return `--mount 需要有效的 type、source 和 target：${value}`
  }
  return { type, source, target, readOnly }
}

function composeServices(source: string): string[] {
  const lines = source.split(/\r?\n/)
  const servicesIndex = lines.findIndex((line) => /^\s*services\s*:\s*(?:#.*)?$/.test(line))
  if (servicesIndex < 0) return []
  const baseIndent = lines[servicesIndex]!.match(/^\s*/)?.[0].length || 0
  const result: string[] = []
  let serviceIndent = 0
  for (const line of lines.slice(servicesIndex + 1)) {
    if (!line.trim() || line.trimStart().startsWith('#')) continue
    const indent = line.match(/^\s*/)?.[0].length || 0
    if (indent <= baseIndent) break
    const match = line.match(/^\s+([A-Za-z0-9][A-Za-z0-9_.-]*)\s*:\s*(?:#.*)?$/)
    if (match && indent > baseIndent) {
      if (!serviceIndent) serviceIndent = indent
      if (indent === serviceIndent) result.push(match[1]!)
    }
  }
  return [...new Set(result)]
}

function normalizeProjectName(value: string): string {
  const normalized = value.toLowerCase().replace(/[^a-z0-9_-]+/g, '-').replace(/^-+|-+$/g, '')
  return (normalized || 'stack').slice(0, 63)
}
