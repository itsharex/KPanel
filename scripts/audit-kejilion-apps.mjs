import { createHash } from 'node:crypto'
import { readFile, writeFile } from 'node:fs/promises'

const scriptPath = process.argv[2]
if (!scriptPath) {
  throw new Error('usage: node scripts/audit-kejilion-apps.mjs /path/to/kejilion.sh [output.json]')
}

const source = await readFile(scriptPath, 'utf8')
const functionStart = source.indexOf('linux_panel() {')
if (functionStart < 0) throw new Error('linux_panel was not found')
const menuStart = source.indexOf('case $sub_choice in', functionStart)
const functionEnd = source.indexOf('\nlinux_work() {', menuStart)
if (menuStart < 0 || functionEnd < 0) throw new Error('linux_panel application case was not found')
const menu = source.slice(menuStart, functionEnd)

const headers = [...menu.matchAll(/^[\t ]{1,4}(\d+)\|[^\r\n]+\)/gm)]
const result = []
for (let index = 0; index < headers.length; index += 1) {
  const header = headers[index]
  const next = headers[index + 1]
  const block = menu.slice(header.index, next?.index ?? menu.length)
  const readLocal = (name) => {
    const match = block.match(new RegExp(`local ${name}=(?:"([^"]+)"|([^\\s\\r\\n]+))`))
    return match?.[1] || match?.[2] || ''
  }
  const service = readLocal('docker_app_service').replace(/[“”]/g, '')
  result.push({
    num: Number(header[1]),
    container: readLocal('docker_name').replace(/[“”]/g, ''),
    ...(service ? { service } : {}),
    image: readLocal('docker_img'),
    defaultPort: Number.parseInt(readLocal('docker_port'), 10) || 0,
    usesDockerApp: /\bdocker_app(?:_plus)?\b/.test(block),
    usesPanelInstaller: /\binstall_panel\b/.test(block),
  })
}

const snapshot = {
  schemaVersion: 1,
  scriptSha256: createHash('sha256').update(source).digest('hex'),
  apps: result,
}
const output = `${JSON.stringify(snapshot, null, 2)}\n`
if (process.argv[3]) {
  await writeFile(process.argv[3], output, { encoding: 'utf8', mode: 0o644 })
  process.stdout.write(`Audited ${result.length} built-in applications.\n`)
} else {
  process.stdout.write(output)
}
