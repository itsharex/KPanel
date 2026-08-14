import type { DesktopWorkspaceDraft } from '@/stores/desktopIcons'
import { useDesktopIcons } from '@/stores/desktopIcons'
import type { DesktopShortcut, DesktopWorkspaceUpdate, FileEntry } from '@/types/api'

export const DESKTOP_FILE_DRAG_TYPE = 'application/x-kpanel-desktop-file-shortcut'
export const MAX_DESKTOP_SHORTCUTS = 64

export type DesktopFileEntry = Pick<FileEntry, 'name' | 'path' | 'kind'> & Partial<Pick<FileEntry, 'resourceVersion'>>
export type DesktopFileShortcutInput = DesktopWorkspaceUpdate['shortcuts'][number]

export interface DesktopFileShortcutAddResult {
  added: DesktopFileShortcutInput[]
  duplicates: DesktopShortcut[]
  ignored: DesktopFileEntry[]
}

export class DesktopShortcutLimitError extends Error {
  constructor(readonly available: number, readonly requested: number) {
    super('desktop_shortcut_limit')
  }
}

interface ActiveDesktopFileDrag {
  token: string
  entries: DesktopFileEntry[]
}

let activeDrag: ActiveDesktopFileDrag | undefined

function randomToken(): string {
  const bytes = new Uint8Array(16)
  globalThis.crypto?.getRandomValues?.(bytes)
  if (bytes.some(Boolean)) return [...bytes].map((value) => value.toString(16).padStart(2, '0')).join('')
  return `${Date.now().toString(16)}${Math.random().toString(16).slice(2)}`
}

function supportedEntry(entry: DesktopFileEntry): boolean {
  return entry.kind === 'file' || entry.kind === 'directory'
}

function cleanShortcutName(entry: DesktopFileEntry): string {
  const cleaned = entry.name.replace(/[\s\p{Cc}]+/gu, ' ').trim()
  const fallback = entry.kind === 'directory' ? '文件夹' : '文件'
  const characters = [...(cleaned || fallback)]
  return characters.length <= 48 ? characters.join('') : `${characters.slice(0, 47).join('')}…`
}

function targetKey(targetType: DesktopFileShortcutInput['targetType'], path?: string): string {
  return `${targetType}\0${path || ''}`
}

export async function addFileEntriesToDesktop(
  entries: readonly DesktopFileEntry[],
  place?: (
    draft: DesktopWorkspaceDraft,
    added: readonly DesktopFileShortcutInput[],
  ) => void,
): Promise<DesktopFileShortcutAddResult> {
  const desktopIcons = useDesktopIcons()
  const candidates = entries.filter(supportedEntry)
  const ignored = entries.filter((entry) => !supportedEntry(entry))
  const result: DesktopFileShortcutAddResult = { added: [], duplicates: [], ignored }
  if (!candidates.length) return result

  await desktopIcons.mutate((draft) => {
    const existingByTarget = new Map<string, DesktopShortcut>()
    for (const shortcut of desktopIcons.workspace.value.shortcuts) {
      if ((shortcut.targetType === 'file' || shortcut.targetType === 'directory') && shortcut.path) {
        existingByTarget.set(targetKey(shortcut.targetType, shortcut.path), shortcut)
      }
    }
    const seen = new Set(existingByTarget.keys())
    const additions: DesktopFileShortcutInput[] = []
    const duplicates: DesktopShortcut[] = []
    for (const entry of candidates) {
      const targetType = entry.kind as 'file' | 'directory'
      const key = targetKey(targetType, entry.path)
      const duplicate = existingByTarget.get(key)
      if (duplicate) {
        duplicates.push(duplicate)
        continue
      }
      if (seen.has(key)) continue
      seen.add(key)
      additions.push({
        id: desktopIcons.generateShortcutID(),
        name: cleanShortcutName(entry),
        description: '',
        targetType,
        path: entry.path,
      })
    }
    const available = Math.max(0, MAX_DESKTOP_SHORTCUTS - draft.shortcuts.length)
    if (additions.length > available) throw new DesktopShortcutLimitError(available, additions.length)
    result.duplicates = duplicates
    if (!additions.length) return false
    draft.shortcuts.push(...additions)
    place?.(draft, additions)
    result.added = additions
  })
  return result
}

export function beginDesktopFileDrag(event: DragEvent, entries: readonly DesktopFileEntry[]): boolean {
  const dataTransfer = event.dataTransfer
  const supported = entries.filter(supportedEntry)
  if (!dataTransfer || !supported.length) return false
  const token = randomToken()
  activeDrag = { token, entries: supported.map((entry) => ({ ...entry })) }
  dataTransfer.effectAllowed = 'all'
  dataTransfer.setData(DESKTOP_FILE_DRAG_TYPE, token)
  dataTransfer.setData('text/plain', supported.length === 1 ? supported[0]!.name : `${supported.length} 个项目`)
  return true
}

export function hasDesktopFileDrag(event: DragEvent): boolean {
  return Boolean(activeDrag && Array.from(event.dataTransfer?.types || []).includes(DESKTOP_FILE_DRAG_TYPE))
}

export function desktopFileDragEntries(event: DragEvent): DesktopFileEntry[] {
  const token = event.dataTransfer?.getData(DESKTOP_FILE_DRAG_TYPE)
  if (!activeDrag || !token || token !== activeDrag.token) return []
  return activeDrag.entries.map((entry) => ({ ...entry }))
}

/** Drag data is protected before drop in some browsers, so hover uses only the active in-memory payload. */
export function peekDesktopFileDragEntries(event: DragEvent): DesktopFileEntry[] {
  if (!hasDesktopFileDrag(event) || !activeDrag) return []
  return activeDrag.entries.map((entry) => ({ ...entry }))
}

export function clearDesktopFileDrag(): void {
  activeDrag = undefined
}

export function resetDesktopFileDragForTest(): void {
  activeDrag = undefined
}
