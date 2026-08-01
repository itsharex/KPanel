<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  Archive,
  ChevronRight,
  ClipboardPaste,
  Code2,
  Copy,
  Download,
  Eye,
  File,
  FileAudio,
  FileImage,
  FileText,
  FileVideo,
  Folder,
  FolderOpen,
  HardDrive,
  LayoutGrid,
  List,
  MoreHorizontal,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  Scissors,
  Search,
  ShieldCheck,
  Trash2,
  Upload,
  WrapText,
  X,
} from '@lucide/vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import { ApiError, api } from '@/lib/api'
import type { CodeLanguage } from '@/lib/code-editor-language'
import { useToast } from '@/stores/toast'
import type { FileActionInput, FileDirectory, FileEntry, FileTrashEntry } from '@/types/api'

const CodeEditor = defineAsyncComponent(() => import('@/components/files/CodeEditor.vue'))

type DialogAction = 'mkdir' | 'rename' | 'chmod' | 'compress' | 'extract' | 'trash'
type PreviewMode = 'text' | 'image' | 'audio' | 'video' | 'pdf' | 'metadata'
type ClipboardMode = 'copy' | 'move'
type ArchiveFormat = 'tar.gz' | 'zip' | 'tar'
type FileViewMode = 'list' | 'grid'

interface FileClipboard {
  mode: ClipboardMode
  entries: FileEntry[]
}

interface CodeEditorHandle {
  getValue: () => string
  markClean: () => void
  openSearch: () => void
  focus: () => void
}

interface CodeEditorStatus {
  line: number
  column: number
  lines: number
}

const toast = useToast()
const directory = ref<FileDirectory>()
const currentPath = ref('/home')
const search = ref('')
const sortKey = ref<'name' | 'size' | 'modified'>('name')
const sortDescending = ref(false)
const viewMode = ref<FileViewMode>('list')
const loading = ref(false)
const dragging = ref(false)
const selected = ref(new Set<string>())
const selectionAnchor = ref<string>()
const uploadInput = ref<HTMLInputElement>()
const uploadProgress = ref<Record<string, number>>({})
const dialogAction = ref<DialogAction>()
const dialogValue = ref('')
const dialogFormat = ref<ArchiveFormat>('tar.gz')
const dialogBusy = ref(false)
const dialogEntries = ref<FileEntry[]>([])
const contextMenu = ref<{ entry?: FileEntry; x: number; y: number }>()
const clipboard = ref<FileClipboard>()
const pasteBusy = ref(false)
const previewEntry = ref<FileEntry>()
const previewContent = ref('')
const previewLoading = ref(false)
const previewSaving = ref(false)
const previewDirty = ref(false)
const editorInfo = ref<Pick<CodeLanguage, 'label' | 'highlighted' | 'reason'> & { loadMs: number }>()
const codeEditorRef = ref<CodeEditorHandle>()
const editorStatus = ref<CodeEditorStatus>()
const editorLineWrap = ref(false)
const trashOpen = ref(false)
const trashLoading = ref(false)
const trashBusy = ref(false)
const trashEntries = ref<FileTrashEntry[]>([])
const trashTotal = ref(0)
const trashTruncated = ref(false)
const selectedTrash = ref(new Set<string>())
const thumbnailFailures = ref(new Set<string>())
let directoryController: AbortController | undefined
let archiveController: AbortController | undefined
let searchTimer: number | undefined
let unmounted = false

const fileViewStorageKey = 'kpanel:files:view:v1'
const thumbnailSourceMaxBytes = 12 * 1024 * 1024

const entries = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  const values = query
    ? (directory.value?.entries || []).filter((entry) =>
    entry.name.toLocaleLowerCase().includes(query),
      )
    : [...(directory.value?.entries || [])]
  values.sort((left, right) => {
    if (left.kind === 'directory' && right.kind !== 'directory') return -1
    if (left.kind !== 'directory' && right.kind === 'directory') return 1
    let result = 0
    if (sortKey.value === 'size') result = left.sizeBytes - right.sizeBytes
    else if (sortKey.value === 'modified')
      result = new Date(left.modifiedAt).getTime() - new Date(right.modifiedAt).getTime()
    else result = left.name.localeCompare(right.name, 'zh-CN', { numeric: true, sensitivity: 'base' })
    return sortDescending.value ? -result : result
  })
  return values
})

const breadcrumbs = computed(() => {
  const parts = currentPath.value.split('/').filter(Boolean)
  return [
    { name: '根目录', path: '/' },
    ...parts.map((name, index) => ({
      name,
      path: `/${parts.slice(0, index + 1).join('/')}`,
    })),
  ]
})

const selectedEntries = computed(() =>
  (directory.value?.entries || []).filter((entry) => selected.value.has(entry.path)),
)
const allVisibleSelected = computed(
  () => entries.value.length > 0 && entries.value.every((entry) => selected.value.has(entry.path)),
)
const previewMode = computed<PreviewMode>(() => {
  const entry = previewEntry.value
  if (!entry) return 'metadata'
  if (entry.editable) return 'text'
  if (entry.mime?.startsWith('image/')) return 'image'
  if (entry.mime?.startsWith('audio/')) return 'audio'
  if (entry.mime?.startsWith('video/')) return 'video'
  if (entry.mime === 'application/pdf') return 'pdf'
  return 'metadata'
})
const previewURL = computed(() =>
  previewEntry.value ? api.files.contentUrl(previewEntry.value.path, 'inline') : '',
)
const dialogTitle = computed(() => {
  const titles: Record<DialogAction, string> = {
    mkdir: '新建文件夹',
    rename: '重命名',
    chmod: dialogEntries.value.length > 1 ? `修改 ${dialogEntries.value.length} 项权限` : '修改权限',
    compress: dialogEntries.value.length > 1 ? `压缩 ${dialogEntries.value.length} 项` : '压缩文件',
    extract: '解压文件',
    trash: dialogEntries.value.length > 1 ? `移入回收站（${dialogEntries.value.length} 项）` : '移入回收站',
  }
  return dialogAction.value ? titles[dialogAction.value] : '文件操作'
})

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message
  if (error instanceof Error) return error.message
  return '操作未完成，请稍后重试。'
}

function archiveFormat(entry: FileEntry): ArchiveFormat | undefined {
  if (entry.kind !== 'file') return undefined
  const name = entry.name.toLocaleLowerCase()
  if (name.endsWith('.tar.gz') || name.endsWith('.tgz')) return 'tar.gz'
  if (name.endsWith('.zip')) return 'zip'
  if (name.endsWith('.tar')) return 'tar'
  return undefined
}

function archiveSuffix(format: ArchiveFormat): string {
  return format === 'tar.gz' ? '.tar.gz' : `.${format}`
}

function withoutArchiveSuffix(name: string): string {
  return name.replace(/\.(?:tar\.gz|tgz|zip|tar)$/i, '') || 'archive'
}

function normalizedArchiveName(name: string, format: ArchiveFormat): string {
  return `${withoutArchiveSuffix(name.trim())}${archiveSuffix(format)}`
}

async function loadDirectory(path = currentPath.value, append = false): Promise<void> {
  if (append && !directory.value?.nextOffset) return
  directoryController?.abort()
  const controller = new AbortController()
  directoryController = controller
  loading.value = true
  contextMenu.value = undefined
  try {
    const result = await api.files.list(
      path,
      {
        offset: append ? directory.value?.nextOffset : 0,
        search: search.value.trim() || undefined,
      },
      controller.signal,
    )
    if (append && directory.value?.path === result.path) {
      const known = new Set(directory.value.entries.map((entry) => entry.path))
      directory.value = {
        ...result,
        entries: [
          ...directory.value.entries,
          ...result.entries.filter((entry) => !known.has(entry.path)),
        ],
      }
    } else {
      directory.value = result
    }
    currentPath.value = result.path
    if (!append) {
      selected.value = new Set()
      selectionAnchor.value = undefined
      thumbnailFailures.value = new Set()
    }
  } catch (error) {
    if (controller.signal.aborted) return
    toast.danger('目录读取失败', errorMessage(error))
  } finally {
    if (directoryController === controller) {
      loading.value = false
      directoryController = undefined
    }
  }
}

function setViewMode(mode: FileViewMode): void {
  viewMode.value = mode
  try {
    window.localStorage?.setItem(fileViewStorageKey, mode)
  } catch {
    // Browser privacy modes may reject preference storage; the current view still works.
  }
}

function restoreViewMode(): void {
  try {
    const stored = window.localStorage?.getItem(fileViewStorageKey)
    if (stored === 'list' || stored === 'grid') viewMode.value = stored
  } catch {
    // Keep the lightweight list default when browser storage is unavailable.
  }
}

function openEntry(entry: FileEntry): void {
  contextMenu.value = undefined
  if (entry.kind === 'directory') {
    void loadDirectory(entry.path)
    return
  }
  if (entry.kind === 'file') void openPreview(entry)
}

async function openPreview(entry: FileEntry): Promise<void> {
  previewEntry.value = entry
  previewContent.value = ''
  previewDirty.value = false
  editorInfo.value = undefined
  editorStatus.value = undefined
  editorLineWrap.value = false
  if (!entry.editable) return
  previewLoading.value = true
  try {
    previewContent.value = await api.files.text(entry.path)
  } catch (error) {
    toast.danger('文件打开失败', errorMessage(error))
    previewEntry.value = undefined
  } finally {
    previewLoading.value = false
  }
}

function closePreview(): void {
  if (previewDirty.value && !window.confirm('文件尚未保存，确认关闭吗？')) return
  previewEntry.value = undefined
  previewContent.value = ''
  previewDirty.value = false
  editorInfo.value = undefined
  editorStatus.value = undefined
  editorLineWrap.value = false
}

async function savePreview(content?: string): Promise<void> {
  const entry = previewEntry.value
  if (!entry || !entry.editable || !previewDirty.value) return
  const nextContent = content ?? codeEditorRef.value?.getValue() ?? previewContent.value
  previewContent.value = nextContent
  previewSaving.value = true
  try {
    const result = await api.files.write(entry.path, nextContent, entry.resourceVersion)
    previewEntry.value = result.entry
    if ((codeEditorRef.value?.getValue() ?? nextContent) === nextContent) {
      previewDirty.value = false
      codeEditorRef.value?.markClean()
    }
    toast.success('已保存', entry.name)
    if (!unmounted) await loadDirectory()
  } catch (error) {
    toast.danger('保存失败', errorMessage(error))
  } finally {
    previewSaving.value = false
  }
}

function handleEditorReady(info: CodeLanguage & { loadMs: number }): void {
  editorInfo.value = {
    label: info.label,
    highlighted: info.highlighted,
    reason: info.reason,
    loadMs: info.loadMs,
  }
}

function handleEditorStatus(status: CodeEditorStatus): void {
  editorStatus.value = status
}

function toggleEntry(path: string): void {
  const next = new Set(selected.value)
  if (next.has(path)) next.delete(path)
  else next.add(path)
  selected.value = next
  selectionAnchor.value = path
}

function selectEntry(event: MouseEvent, path: string): void {
  if (event.shiftKey && selectionAnchor.value) {
    const visiblePaths = entries.value.map((entry) => entry.path)
    const anchorIndex = visiblePaths.indexOf(selectionAnchor.value)
    const currentIndex = visiblePaths.indexOf(path)
    if (anchorIndex >= 0 && currentIndex >= 0) {
      const start = Math.min(anchorIndex, currentIndex)
      const end = Math.max(anchorIndex, currentIndex)
      const range = visiblePaths.slice(start, end + 1)
      selected.value =
        event.ctrlKey || event.metaKey
          ? new Set([...selected.value, ...range])
          : new Set(range)
      return
    }
  }
  if (event.ctrlKey || event.metaKey) {
    toggleEntry(path)
    return
  }
  if (selected.value.size === 1 && selected.value.has(path)) {
    clearSelection()
    return
  }
  selected.value = new Set([path])
  selectionAnchor.value = path
}

function toggleAll(): void {
  const clearVisible = allVisibleSelected.value
  const next = new Set(selected.value)
  if (clearVisible) entries.value.forEach((entry) => next.delete(entry.path))
  else entries.value.forEach((entry) => next.add(entry.path))
  selected.value = next
  selectionAnchor.value = clearVisible ? undefined : entries.value[0]?.path
}

function clearSelection(): void {
  selected.value = new Set()
  selectionAnchor.value = undefined
}

function preventNativeSelection(event: Event): void {
  event.preventDefault()
}

function selectForContext(entry: FileEntry): void {
  if (selected.value.has(entry.path)) return
  selected.value = new Set([entry.path])
  selectionAnchor.value = entry.path
}

function showContext(event: MouseEvent, entry: FileEntry): void {
  event.preventDefault()
  selectForContext(entry)
  contextMenu.value = {
    entry,
    x: Math.max(8, Math.min(event.clientX, window.innerWidth - 210)),
    y: Math.max(8, Math.min(event.clientY, window.innerHeight - 430)),
  }
}

function showDirectoryContext(event: MouseEvent): void {
  const target = event.target as HTMLElement
  if (
    target.closest(
      '.file-row--entry, .file-grid-card, .file-toolbar, .clipboard-bar, .upload-strip, .file-limit, .drop-overlay',
    )
  ) {
    return
  }
  event.preventDefault()
  contextMenu.value = {
    x: Math.max(8, Math.min(event.clientX, window.innerWidth - 230)),
    y: Math.max(8, Math.min(event.clientY, window.innerHeight - 220)),
  }
}

function openDialog(action: DialogAction, entry?: FileEntry): void {
  contextMenu.value = undefined
  dialogEntries.value = entry ? [entry] : [...selectedEntries.value]
  if ((action === 'compress' || action === 'extract') && !dialogEntries.value.length) return
  dialogAction.value = action
  if (action === 'mkdir') dialogValue.value = ''
  else if (action === 'rename') dialogValue.value = dialogEntries.value[0]?.name || ''
  else if (action === 'chmod') dialogValue.value = '644'
  else if (action === 'compress') {
    dialogFormat.value = 'tar.gz'
    dialogValue.value = dialogEntries.value.length === 1
      ? `${dialogEntries.value[0]?.name || 'archive'}${archiveSuffix(dialogFormat.value)}`
      : `archive${archiveSuffix(dialogFormat.value)}`
  } else if (action === 'extract') {
    const source = dialogEntries.value[0]
    const format = source ? archiveFormat(source) : undefined
    if (!source || !format) {
      dialogAction.value = undefined
      toast.danger('无法解压', '仅支持 .tar.gz、.tgz、.zip 和 .tar 文件。')
      return
    }
    dialogFormat.value = format
    dialogValue.value = withoutArchiveSuffix(source.name)
  }
}

function closeDialog(): void {
  if (dialogBusy.value) return
  dialogAction.value = undefined
  dialogValue.value = ''
  dialogEntries.value = []
}

function cancelArchive(): void {
  archiveController?.abort()
}

function setClipboard(mode: ClipboardMode, entry?: FileEntry): void {
  contextMenu.value = undefined
  const entriesToStore = entry ? [entry] : [...selectedEntries.value]
  if (!entriesToStore.length) return
  clipboard.value = { mode, entries: entriesToStore }
  clearSelection()
  toast.success(
    mode === 'copy' ? '已复制到文件剪贴板' : '已剪切到文件剪贴板',
    `${entriesToStore.length} 项，进入目标文件夹后点击“粘贴”`,
  )
}

function clearClipboard(): void {
  clipboard.value = undefined
}

async function pasteClipboard(target = currentPath.value): Promise<void> {
  const stored = clipboard.value
  if (!stored?.entries.length || pasteBusy.value) return
  contextMenu.value = undefined
  pasteBusy.value = true
  try {
    const result = await api.files.action({
      action: stored.mode,
      sources: stored.entries.map((entry) => entry.path),
      target,
    })
    if (stored.mode === 'move') {
      const failed = new Set(result.failed.map((item) => item.path))
      const remaining = stored.entries.filter((entry) => failed.has(entry.path))
      clipboard.value = remaining.length ? { mode: 'move', entries: remaining } : undefined
    }
    if (result.failed.length) {
      toast.danger(
        result.succeeded.length ? '部分文件未粘贴' : '粘贴未完成',
        `${result.succeeded.length} 项成功，${result.failed.length} 项失败：${result.failed[0]?.detail || '请刷新后重试'}`,
      )
    } else {
      toast.success(
        stored.mode === 'copy' ? '复制完成' : '移动完成',
        `${result.succeeded.length} 项已粘贴到 ${target}`,
      )
    }
    if (!unmounted) await loadDirectory()
  } catch (error) {
    toast.danger('粘贴失败', errorMessage(error))
    await loadDirectory()
  } finally {
    pasteBusy.value = false
  }
}

async function submitDialog(): Promise<void> {
  const action = dialogAction.value
  if (!action) return
  const controller = action === 'compress' || action === 'extract'
    ? new AbortController()
    : undefined
  archiveController = controller
  dialogBusy.value = true
  try {
    let input: FileActionInput
    if (action === 'mkdir') {
      input = { action, target: currentPath.value, name: dialogValue.value.trim() }
    } else if (action === 'rename') {
      const entry = dialogEntries.value[0]
      if (!entry) throw new Error('请选择需要重命名的文件。')
      const parent = entry.path.slice(0, Math.max(entry.path.lastIndexOf('/'), 1))
      input = {
        action,
        sources: [entry.path],
        target: `${parent === '/' ? '' : parent}/${dialogValue.value.trim()}`,
        expectedResourceVersion: entry.resourceVersion,
      }
    } else if (action === 'trash') {
      input = {
        action,
        sources: dialogEntries.value.map((entry) => entry.path),
        expectedResourceVersions: Object.fromEntries(
          dialogEntries.value.map((entry) => [entry.path, entry.resourceVersion]),
        ),
      }
    } else if (action === 'chmod') {
      input = {
        action,
        sources: dialogEntries.value.map((entry) => entry.path),
        mode: dialogValue.value.trim(),
      }
    } else if (action === 'compress') {
      input = {
        action,
        sources: dialogEntries.value.map((entry) => entry.path),
        target: currentPath.value,
        name: normalizedArchiveName(dialogValue.value, dialogFormat.value),
        format: dialogFormat.value,
        expectedResourceVersions: Object.fromEntries(
          dialogEntries.value.map((entry) => [entry.path, entry.resourceVersion]),
        ),
      }
    } else if (action === 'extract') {
      const entry = dialogEntries.value[0]
      if (!entry) throw new Error('请选择需要解压的文件。')
      input = {
        action,
        sources: [entry.path],
        target: currentPath.value,
        name: dialogValue.value.trim(),
        format: dialogFormat.value,
        expectedResourceVersion: entry.resourceVersion,
      }
    } else {
      const unsupportedAction: never = action
      throw new Error(`不支持的文件操作：${unsupportedAction}`)
    }
    const result = controller
      ? await api.files.action(input, controller.signal)
      : await api.files.action(input)
    if (result.failed.length) {
      toast.danger(
        result.succeeded.length ? '部分文件未处理' : '文件操作未完成',
        `${result.succeeded.length} 项成功，${result.failed.length} 项失败：${result.failed[0]?.detail || '请刷新后重试'}`,
      )
    } else {
      toast.success(
        action === 'trash'
          ? '已移入回收站'
          : action === 'compress'
            ? '压缩完成'
            : action === 'extract'
              ? '解压完成'
              : '文件操作完成',
        `${Math.max(result.succeeded.length, 1)} 项已处理`,
      )
    }
    dialogAction.value = undefined
    dialogValue.value = ''
    dialogEntries.value = []
    if (!unmounted) await loadDirectory()
  } catch (error) {
    if (controller?.signal.aborted) {
      if (!unmounted) toast.success('操作已停止', '未完成的临时文件已清理。')
      dialogAction.value = undefined
      dialogValue.value = ''
      dialogEntries.value = []
    } else {
      toast.danger('文件操作失败', errorMessage(error))
    }
    if (!unmounted) await loadDirectory()
  } finally {
    if (archiveController === controller) archiveController = undefined
    dialogBusy.value = false
  }
}

async function openTrash(): Promise<void> {
  trashOpen.value = true
  selectedTrash.value = new Set()
  await loadTrash()
}

async function loadTrash(): Promise<void> {
  trashLoading.value = true
  try {
    const result = await api.files.trash()
    trashEntries.value = result.entries
    trashTotal.value = result.total
    trashTruncated.value = result.truncated
    selectedTrash.value = new Set(
      [...selectedTrash.value].filter((id) => result.entries.some((entry) => entry.id === id)),
    )
  } catch (error) {
    toast.danger('回收站读取失败', errorMessage(error))
  } finally {
    trashLoading.value = false
  }
}

function toggleTrash(id: string): void {
  const next = new Set(selectedTrash.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedTrash.value = next
}

async function runTrashAction(action: 'trash_restore' | 'trash_delete' | 'trash_empty'): Promise<void> {
  const chosen = trashEntries.value.filter((entry) => selectedTrash.value.has(entry.id))
  if (action !== 'trash_empty' && !chosen.length) return
  if (action === 'trash_delete' && !window.confirm(`彻底删除选中的 ${chosen.length} 项？此操作不可恢复。`)) return
  if (action === 'trash_empty' && !window.confirm(`清空回收站中的 ${trashTotal.value} 项？此操作不可恢复。`)) return
  trashBusy.value = true
  try {
    const input: FileActionInput = {
      action,
      trashIds: action === 'trash_empty' ? undefined : chosen.map((entry) => entry.id),
      expectedResourceVersions:
        action === 'trash_empty'
          ? undefined
          : Object.fromEntries(chosen.map((entry) => [entry.id, entry.resourceVersion])),
    }
    const result = await api.files.action(input)
    if (result.failed.length) {
      toast.danger(
        result.succeeded.length ? '部分回收站项目未处理' : '回收站操作失败',
        `${result.succeeded.length} 项成功，${result.failed.length} 项失败：${result.failed[0]?.detail || '请刷新后重试'}`,
      )
    } else {
      const title = action === 'trash_restore' ? '恢复完成' : action === 'trash_empty' ? '回收站已清空' : '已彻底删除'
      toast.success(title, `${result.succeeded.length} 项已处理`)
    }
    selectedTrash.value = new Set()
    await Promise.all([loadTrash(), loadDirectory()])
  } catch (error) {
    toast.danger('回收站操作失败', errorMessage(error))
  } finally {
    trashBusy.value = false
  }
}

function download(entry: FileEntry): void {
  contextMenu.value = undefined
  const anchor = document.createElement('a')
  anchor.href = api.files.contentUrl(entry.path, 'attachment')
  anchor.download = entry.name
  anchor.rel = 'noopener'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
}

function downloadSelected(): void {
  selectedEntries.value.filter((entry) => entry.kind === 'file').forEach(download)
}

function setSort(key: 'name' | 'size' | 'modified'): void {
  if (sortKey.value === key) sortDescending.value = !sortDescending.value
  else {
    sortKey.value = key
    sortDescending.value = false
  }
}

async function uploadFiles(files: FileList | File[]): Promise<void> {
  const values = Array.from(files)
  if (!values.length) return
  for (const file of values) {
    uploadProgress.value = { ...uploadProgress.value, [file.name]: 0 }
    try {
      await api.files.upload(currentPath.value, file, false, (progress) => {
        uploadProgress.value = { ...uploadProgress.value, [file.name]: progress }
      })
      uploadProgress.value = { ...uploadProgress.value, [file.name]: 100 }
    } catch (error) {
      if (
        error instanceof ApiError &&
        error.status === 409 &&
        window.confirm(`${file.name} 已存在，是否覆盖？`)
      ) {
        try {
          await api.files.upload(currentPath.value, file, true, (progress) => {
            uploadProgress.value = { ...uploadProgress.value, [file.name]: progress }
          })
          uploadProgress.value = { ...uploadProgress.value, [file.name]: 100 }
          continue
        } catch (overwriteError) {
          toast.danger(`${file.name} 覆盖失败`, errorMessage(overwriteError))
        }
      } else {
        toast.danger(`${file.name} 上传失败`, errorMessage(error))
      }
      const next = { ...uploadProgress.value }
      delete next[file.name]
      uploadProgress.value = next
    }
  }
  if (uploadInput.value) uploadInput.value.value = ''
  await loadDirectory()
  window.setTimeout(() => {
    uploadProgress.value = {}
  }, 1800)
}

function onDrop(event: DragEvent): void {
  dragging.value = false
  if (event.dataTransfer?.files?.length) void uploadFiles(event.dataTransfer.files)
}

function entryIcon(entry: FileEntry) {
  if (entry.kind === 'directory') return Folder
  if (entry.mime?.startsWith('image/')) return FileImage
  if (entry.mime?.startsWith('audio/')) return FileAudio
  if (entry.mime?.startsWith('video/')) return FileVideo
  if (entry.editable) return Code2
  if (
    ['application/zip', 'application/gzip', 'application/x-tar', 'application/x-7z-compressed'].includes(
      entry.mime || '',
    )
  )
    return Archive
  return entry.previewable ? FileText : File
}

function canShowThumbnail(entry: FileEntry): boolean {
  return entry.kind === 'file' &&
    entry.sizeBytes > 0 &&
    entry.sizeBytes <= thumbnailSourceMaxBytes &&
    ['image/jpeg', 'image/png', 'image/gif'].includes(entry.mime || '') &&
    !thumbnailFailures.value.has(entry.path)
}

function thumbnailURL(entry: FileEntry): string {
  return api.files.thumbnailUrl(entry.path, entry.resourceVersion)
}

function markThumbnailFailed(path: string): void {
  const next = new Set(thumbnailFailures.value)
  next.add(path)
  thumbnailFailures.value = next
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes)) return '—'
  if (bytes < 1024) return `${bytes} B`
  const units = ['KiB', 'MiB', 'GiB', 'TiB']
  let value = bytes / 1024
  let index = 0
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index += 1
  }
  return `${value >= 10 ? value.toFixed(1) : value.toFixed(2)} ${units[index]}`
}

function formatTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime())
    ? '—'
    : new Intl.DateTimeFormat('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      }).format(date)
}

function handleWindowClick(event: MouseEvent): void {
  const target = event.target as HTMLElement
  if (!target.closest('.file-context-menu')) contextMenu.value = undefined
}

function handleFileShortcut(event: KeyboardEvent): void {
  const target = event.target as HTMLElement | null
  if (
    previewEntry.value ||
    target?.matches('input, textarea, select, [contenteditable="true"]')
  ) {
    return
  }
  const key = event.key.toLocaleLowerCase()
  if ((event.ctrlKey || event.metaKey) && key === 'a' && entries.value.length) {
    event.preventDefault()
    selected.value = new Set(entries.value.map((entry) => entry.path))
    selectionAnchor.value = entries.value[0]?.path
  } else if ((event.ctrlKey || event.metaKey) && key === 'c' && selectedEntries.value.length) {
    event.preventDefault()
    setClipboard('copy')
  } else if ((event.ctrlKey || event.metaKey) && key === 'x' && selectedEntries.value.length) {
    event.preventDefault()
    setClipboard('move')
  } else if ((event.ctrlKey || event.metaKey) && key === 'v' && clipboard.value?.entries.length) {
    event.preventDefault()
    void pasteClipboard()
  } else if (event.key === 'Delete' && selectedEntries.value.length) {
    event.preventDefault()
    openDialog('trash')
  } else if (event.key === 'Escape') {
    contextMenu.value = undefined
  }
}

onMounted(() => {
  window.addEventListener('click', handleWindowClick)
  window.addEventListener('keydown', handleFileShortcut)
  restoreViewMode()
  void loadDirectory('/')
})

watch(search, () => {
  if (searchTimer !== undefined) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    void loadDirectory(currentPath.value)
  }, 250)
})

onBeforeUnmount(() => {
  unmounted = true
  directoryController?.abort()
  archiveController?.abort()
  if (searchTimer !== undefined) window.clearTimeout(searchTimer)
  window.removeEventListener('click', handleWindowClick)
  window.removeEventListener('keydown', handleFileShortcut)
})
</script>

<template>
  <section class="files-page">
    <PageHeader title="文件管理" description="轻量管理宿主机文件；KPanel 凭据与状态目录已隔离保护。">
      <template #actions>
        <button class="button button--secondary" type="button" :disabled="loading" @click="loadDirectory()">
          <RefreshCw :size="16" :class="{ spinning: loading }" />
          刷新
        </button>
        <button class="button button--secondary" type="button" @click="openTrash">
          <Trash2 :size="16" />
          回收站
        </button>
        <button class="button button--secondary" type="button" @click="openDialog('mkdir')">
          <Plus :size="16" />
          新建文件夹
        </button>
        <button class="button button--primary" type="button" @click="uploadInput?.click()">
          <Upload :size="16" />
          上传文件
        </button>
        <input
          ref="uploadInput"
          class="sr-only"
          type="file"
          aria-label="选择上传文件"
          multiple
          @change="($event.target as HTMLInputElement).files && uploadFiles(($event.target as HTMLInputElement).files!)"
        />
      </template>
    </PageHeader>

    <div class="file-guard">
      <ShieldCheck :size="18" />
      <span>可管理范围 <strong>/</strong>，符号链接、内核写入口与 KPanel 数据目录不会开放。</span>
      <small>单文件上传 512 MiB · 文本编辑 2 MiB · 批量最多 100 项</small>
    </div>

    <nav class="file-shortcuts" aria-label="常用目录">
      <button v-for="item in ['/', '/home', '/root', '/etc', '/var']" :key="item" type="button" @click="loadDirectory(item)">
        {{ item === '/' ? '根目录 /' : item }}
      </button>
    </nav>

    <section
      class="file-browser"
      :class="{ 'file-browser--dragging': dragging }"
      @dragenter.prevent="dragging = true"
      @dragover.prevent
      @dragleave.self="dragging = false"
      @drop.prevent="onDrop"
      @contextmenu="showDirectoryContext"
    >
      <header class="file-toolbar">
        <nav class="breadcrumbs" aria-label="文件路径">
          <button
            v-for="(item, index) in breadcrumbs"
            :key="item.path"
            type="button"
            :disabled="item.path === currentPath"
            @click="loadDirectory(item.path)"
          >
            <HardDrive v-if="index === 0" :size="15" />
            <span>{{ item.name }}</span>
            <ChevronRight v-if="index < breadcrumbs.length - 1" :size="14" />
          </button>
        </nav>
        <div class="file-toolbar__controls">
          <label class="file-search">
            <Search :size="16" />
            <input v-model="search" type="search" aria-label="搜索当前目录" placeholder="搜索当前目录" />
            <button v-if="search" type="button" aria-label="清除搜索" @click="search = ''">
              <X :size="14" />
            </button>
          </label>
          <div v-if="viewMode === 'grid'" class="file-grid-sort">
            <select v-model="sortKey" aria-label="大图标排序方式">
              <option value="name">名称</option>
              <option value="modified">修改时间</option>
              <option value="size">大小</option>
            </select>
            <button
              type="button"
              :aria-label="sortDescending ? '切换为升序' : '切换为降序'"
              :title="sortDescending ? '当前降序' : '当前升序'"
              @click="sortDescending = !sortDescending"
            >{{ sortDescending ? '↓' : '↑' }}</button>
          </div>
          <div class="file-view-switch" role="group" aria-label="文件排版方式">
            <button
              type="button"
              :class="{ 'is-active': viewMode === 'list' }"
              :aria-pressed="viewMode === 'list'"
              aria-label="列表排版"
              title="列表排版"
              @click="setViewMode('list')"
            ><List :size="17" /></button>
            <button
              type="button"
              :class="{ 'is-active': viewMode === 'grid' }"
              :aria-pressed="viewMode === 'grid'"
              aria-label="大图标排版"
              title="大图标排版"
              @click="setViewMode('grid')"
            ><LayoutGrid :size="17" /></button>
          </div>
        </div>
      </header>

      <Transition name="slide">
        <div v-if="clipboard?.entries.length" class="clipboard-bar">
          <span class="clipboard-bar__icon">
            <Copy v-if="clipboard.mode === 'copy'" :size="17" />
            <Scissors v-else :size="17" />
          </span>
          <span>
            <strong>{{ clipboard.mode === 'copy' ? '已复制' : '已剪切' }} {{ clipboard.entries.length }} 项</strong>
            <small>
              {{ clipboard.entries[0]?.name }}
              <template v-if="clipboard.entries.length > 1"> 等 {{ clipboard.entries.length }} 项</template>
            </small>
          </span>
          <button type="button" :disabled="pasteBusy" @click="pasteClipboard()">
            <ClipboardPaste :size="15" />{{ pasteBusy ? '粘贴中…' : `粘贴到 ${currentPath}` }}
          </button>
          <button type="button" :disabled="pasteBusy" @click="clearClipboard">取消</button>
        </div>
      </Transition>

      <div v-if="Object.keys(uploadProgress).length" class="upload-strip">
        <div v-for="(progress, name) in uploadProgress" :key="name">
          <span>{{ name }}</span>
          <div><i :style="{ width: `${progress}%` }" /></div>
          <strong>{{ progress }}%</strong>
        </div>
      </div>

      <div
        v-if="viewMode === 'list'"
        class="file-table"
        role="table"
        aria-label="文件列表"
        @selectstart="preventNativeSelection"
      >
        <div class="file-row file-row--header" role="row">
          <span>
            <input type="checkbox" :checked="allVisibleSelected" aria-label="选择全部" @change="toggleAll" />
          </span>
          <button type="button" @click="setSort('name')">名称 {{ sortKey === 'name' ? (sortDescending ? '↓' : '↑') : '' }}</button>
          <button type="button" @click="setSort('size')">大小 {{ sortKey === 'size' ? (sortDescending ? '↓' : '↑') : '' }}</button>
          <span>权限</span>
          <span>所有者</span>
          <button type="button" @click="setSort('modified')">修改时间 {{ sortKey === 'modified' ? (sortDescending ? '↓' : '↑') : '' }}</button>
          <span />
        </div>

        <div
          v-for="entry in entries"
          :key="entry.path"
          class="file-row file-row--entry"
          :class="{ 'file-row--selected': selected.has(entry.path) }"
          role="row"
          tabindex="0"
          @click="selectEntry($event, entry.path)"
          @dblclick="openEntry(entry)"
          @keydown.enter="openEntry(entry)"
          @contextmenu.stop="showContext($event, entry)"
        >
          <span @click.stop="toggleEntry(entry.path)">
            <input
              type="checkbox"
              :checked="selected.has(entry.path)"
              :aria-label="`选择 ${entry.name}`"
              @change="toggleEntry(entry.path)"
              @click.stop
            />
          </span>
          <span class="file-name">
            <span class="file-icon" :class="{ 'file-icon--folder': entry.kind === 'directory' }">
              <component :is="entryIcon(entry)" :size="19" />
            </span>
            <span>
              <strong>{{ entry.name }}</strong>
              <small>{{ entry.kind === 'directory' ? '文件夹' : entry.mime || '文件' }}</small>
            </span>
          </span>
          <span>{{ entry.kind === 'directory' ? '—' : formatBytes(entry.sizeBytes) }}</span>
          <span class="mono">{{ entry.mode }}</span>
          <span>{{ entry.owner }}<small v-if="entry.group">:{{ entry.group }}</small></span>
          <span>{{ formatTime(entry.modifiedAt) }}</span>
          <span>
            <button
              class="row-menu"
              type="button"
              :aria-label="`${entry.name} 操作`"
              @click.stop="showContext($event, entry)"
            >
              <MoreHorizontal :size="18" />
            </button>
          </span>
        </div>
      </div>

      <div
        v-else
        class="file-grid"
        role="list"
        aria-label="文件大图标列表"
        @selectstart="preventNativeSelection"
      >
        <div
          v-for="entry in entries"
          :key="entry.path"
          class="file-grid-card"
          :class="{ 'file-grid-card--selected': selected.has(entry.path) }"
          role="listitem"
          tabindex="0"
          @click="selectEntry($event, entry.path)"
          @dblclick="openEntry(entry)"
          @keydown.enter="openEntry(entry)"
          @contextmenu.stop="showContext($event, entry)"
        >
          <input
            class="file-grid-card__check"
            type="checkbox"
            :checked="selected.has(entry.path)"
            :aria-label="`选择 ${entry.name}`"
            @change="toggleEntry(entry.path)"
            @click.stop
          />
          <button
            class="row-menu file-grid-card__menu"
            type="button"
            :aria-label="`${entry.name} 操作`"
            @click.stop="showContext($event, entry)"
          ><MoreHorizontal :size="18" /></button>
          <div class="file-grid-card__visual">
            <img
              v-if="canShowThumbnail(entry)"
              :src="thumbnailURL(entry)"
              :alt="entry.name"
              loading="lazy"
              decoding="async"
              draggable="false"
              @error="markThumbnailFailed(entry.path)"
            />
            <span
              v-else
              class="file-grid-card__icon"
              :class="{ 'file-grid-card__icon--folder': entry.kind === 'directory' }"
            ><component :is="entryIcon(entry)" :size="48" /></span>
          </div>
          <strong :title="entry.name">{{ entry.name }}</strong>
          <small>
            {{ entry.kind === 'directory' ? '文件夹' : formatBytes(entry.sizeBytes) }}
            <span aria-hidden="true">·</span>
            {{ formatTime(entry.modifiedAt) }}
          </small>
        </div>
      </div>

      <div v-if="!loading && !entries.length" class="file-empty">
        <FolderOpen :size="34" />
        <strong>{{ search ? '没有匹配的文件' : '这个文件夹是空的' }}</strong>
        <span>{{ search ? '换一个关键词试试。' : '可直接拖入文件，或在右上角新建文件夹。' }}</span>
      </div>
      <div v-if="loading" class="file-loading"><RefreshCw :size="22" class="spinning" />正在读取目录…</div>
      <footer v-if="directory?.truncated" class="file-limit">
        <span v-if="directory.scanTruncated">目录超过 20,000 项，搜索和分页仅覆盖已扫描范围。</span>
        <span v-else-if="directory.totalKnown">
          已显示 {{ directory.entries.length }} / {{ directory.total }} 项。
        </span>
        <span v-else>已显示 {{ directory.entries.length }} 项，可继续加载。</span>
        <button
          v-if="directory.nextOffset"
          class="button button--secondary"
          type="button"
          :disabled="loading"
          @click="loadDirectory(currentPath, true)"
        >
          加载更多
        </button>
      </footer>

      <div v-if="dragging" class="drop-overlay">
        <Upload :size="34" />
        <strong>松开以上传到 {{ currentPath }}</strong>
      </div>
    </section>

    <Transition name="batch-dock">
      <div
        v-if="selected.size"
        class="batch-bar"
        role="toolbar"
        aria-label="批量文件操作"
      >
        <strong>已选 {{ selected.size }} 项</strong>
        <button
          v-if="selectedEntries.some((entry) => entry.kind === 'file')"
          type="button"
          @click="downloadSelected"
        ><Download :size="15" />下载</button>
        <button type="button" @click="openDialog('compress')"><Archive :size="15" />压缩</button>
        <button type="button" @click="setClipboard('copy')"><Copy :size="15" />复制</button>
        <button type="button" @click="setClipboard('move')"><Scissors :size="15" />剪切</button>
        <button type="button" @click="openDialog('chmod')"><ShieldCheck :size="15" />权限</button>
        <button class="danger-link" type="button" @click="openDialog('trash')">
          <Trash2 :size="15" />回收站
        </button>
        <button type="button" @click="clearSelection">取消选择</button>
      </div>
    </Transition>

    <div
      v-if="contextMenu"
      class="file-context-menu"
      :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
      role="menu"
    >
      <button v-if="contextMenu.entry" type="button" @click="openEntry(contextMenu.entry)">
        <Eye :size="15" />{{ contextMenu.entry.kind === 'directory' ? '打开' : '查看' }}
      </button>
      <button v-if="contextMenu.entry?.kind === 'file'" type="button" @click="download(contextMenu.entry)">
        <Download :size="15" />下载
      </button>
      <button
        v-if="contextMenu.entry && archiveFormat(contextMenu.entry)"
        type="button"
        @click="openDialog('extract', contextMenu.entry)"
      >
        <FolderOpen :size="15" />解压到文件夹
      </button>
      <button v-if="contextMenu.entry" type="button" @click="openDialog('compress', contextMenu.entry)">
        <Archive :size="15" />压缩
      </button>
      <hr v-if="contextMenu.entry" />
      <button v-if="contextMenu.entry" type="button" @click="openDialog('rename', contextMenu.entry)">
        <Pencil :size="15" />重命名
      </button>
      <button v-if="contextMenu.entry" type="button" @click="setClipboard('copy', contextMenu.entry)"><Copy :size="15" />复制</button>
      <button v-if="contextMenu.entry" type="button" @click="setClipboard('move', contextMenu.entry)"><Scissors :size="15" />剪切</button>
      <button
        v-if="clipboard?.entries.length && contextMenu.entry?.kind === 'directory'"
        type="button"
        :disabled="pasteBusy"
        @click="pasteClipboard(contextMenu.entry.path)"
      ><ClipboardPaste :size="15" />粘贴到此文件夹</button>
      <button
        v-if="clipboard?.entries.length"
        type="button"
        :disabled="pasteBusy"
        @click="pasteClipboard()"
      ><ClipboardPaste :size="15" />粘贴到当前目录</button>
      <button v-if="contextMenu.entry" type="button" @click="openDialog('chmod', contextMenu.entry)">
        <ShieldCheck :size="15" />修改权限
      </button>
      <button v-else type="button" @click="openDialog('mkdir')">
        <Plus :size="15" />新建文件夹
      </button>
      <hr v-if="contextMenu.entry" />
      <button v-if="contextMenu.entry" class="danger-link" type="button" @click="openDialog('trash', contextMenu.entry)">
        <Trash2 :size="15" />移入回收站
      </button>
    </div>

    <ModalDialog
      :open="Boolean(dialogAction)"
      :title="dialogTitle"
      :description="
        dialogAction === 'trash'
          ? '文件将移动到 KPanel 隔离回收区，可在回收站中恢复。'
          : dialogAction === 'compress'
            ? '默认使用适合 Linux 服务器的 tar.gz；也可选择 ZIP 或 TAR。'
            : dialogAction === 'extract'
              ? '内容将解压到全新的文件夹，不覆盖已有文件。'
              : ''
      "
      size="small"
      @close="closeDialog"
    >
      <div v-if="dialogAction !== 'trash'" class="operation-form">
        <label>
          <span>
            {{
              dialogAction === 'mkdir'
                ? '文件夹名称'
                : dialogAction === 'rename'
                  ? '新名称'
                  : dialogAction === 'chmod'
                    ? '权限（八进制）'
                    : dialogAction === 'compress'
                      ? '压缩包名称'
                      : '目标文件夹名称'
            }}
          </span>
          <input
            v-model="dialogValue"
            :placeholder="
              dialogAction === 'chmod'
                ? '例如 644 或 755'
                : dialogAction === 'compress'
                  ? '例如 website.tar.gz'
                  : dialogAction === 'extract'
                    ? '例如 website'
                    : '输入名称'
            "
            autocomplete="off"
            @keydown.enter="submitDialog"
          />
        </label>
        <label v-if="dialogAction === 'compress'">
          <span>压缩格式</span>
          <select v-model="dialogFormat">
            <option value="tar.gz">TAR.GZ（推荐）</option>
            <option value="zip">ZIP（跨平台）</option>
            <option value="tar">TAR（不压缩）</option>
          </select>
        </label>
        <small v-if="dialogAction === 'compress' || dialogAction === 'extract'" class="archive-hint">
          单次最多 100 项、10,000 个条目或解压后 10 GiB；不支持符号链接、硬链接和设备文件。
        </small>
      </div>
      <div v-else class="trash-summary">
        <Trash2 :size="24" />
        <strong>确认移动 {{ dialogEntries.length }} 项？</strong>
        <span>稍后可从文件管理右上角的回收站恢复或彻底删除。</span>
      </div>
      <div class="dialog-actions">
        <button
          class="button button--secondary"
          type="button"
          :disabled="dialogBusy && dialogAction !== 'compress' && dialogAction !== 'extract'"
          @click="dialogBusy ? cancelArchive() : closeDialog()"
        >
          {{ dialogBusy && (dialogAction === 'compress' || dialogAction === 'extract') ? '停止' : '取消' }}
        </button>
        <button
          class="button"
          :class="dialogAction === 'trash' ? 'button--danger' : 'button--primary'"
          type="button"
          :disabled="dialogBusy || (dialogAction !== 'trash' && !dialogValue.trim())"
          @click="submitDialog"
        >
          {{
            dialogBusy
              ? dialogAction === 'compress'
                ? '压缩中…'
                : dialogAction === 'extract'
                  ? '解压中…'
                  : '处理中…'
              : dialogAction === 'trash'
                ? '移入回收站'
                : dialogAction === 'compress'
                  ? '开始压缩'
                  : dialogAction === 'extract'
                    ? '开始解压'
                    : '确认'
          }}
        </button>
      </div>
    </ModalDialog>

    <ModalDialog
      :open="trashOpen"
      title="回收站"
      description="删除的文件保存在 Agent 隔离目录中；恢复时不会覆盖同名文件。"
      size="wide"
      @close="!trashBusy && (trashOpen = false)"
    >
      <div class="trash-manager">
        <header>
          <span>共 {{ trashTotal }} 项<span v-if="trashTruncated">（显示最近 500 项）</span></span>
          <div>
            <button class="button button--secondary" type="button" :disabled="trashLoading || trashBusy" @click="loadTrash">
              <RefreshCw :size="15" :class="{ spinning: trashLoading }" />刷新
            </button>
            <button
              class="button button--secondary"
              type="button"
              :disabled="trashBusy || !selectedTrash.size || trashEntries.filter((entry) => selectedTrash.has(entry.id)).some((entry) => !entry.restorable)"
              @click="runTrashAction('trash_restore')"
            ><RotateCcw :size="15" />恢复</button>
            <button class="button button--danger" type="button" :disabled="trashBusy || !selectedTrash.size" @click="runTrashAction('trash_delete')">
              <Trash2 :size="15" />彻底删除
            </button>
            <button class="button button--danger" type="button" :disabled="trashBusy || !trashTotal" @click="runTrashAction('trash_empty')">
              清空回收站
            </button>
          </div>
        </header>
        <div v-if="trashLoading" class="file-loading"><RefreshCw :size="22" class="spinning" />正在读取回收站…</div>
        <div v-else-if="trashEntries.length" class="trash-list">
          <label v-for="entry in trashEntries" :key="entry.id" class="trash-item">
            <input type="checkbox" :checked="selectedTrash.has(entry.id)" @change="toggleTrash(entry.id)" />
            <span class="file-icon" :class="{ 'file-icon--folder': entry.kind === 'directory' }">
              <Folder v-if="entry.kind === 'directory'" :size="19" />
              <File v-else :size="19" />
            </span>
            <span>
              <strong>{{ entry.name }}</strong>
              <small>{{ entry.originalPath || '旧版回收站项目（仅支持彻底删除）' }}</small>
            </span>
            <span>{{ entry.kind === 'directory' ? '文件夹' : formatBytes(entry.sizeBytes) }}</span>
            <span>{{ formatTime(entry.deletedAt) }}</span>
          </label>
        </div>
        <div v-else class="file-empty">
          <Trash2 :size="34" />
          <strong>回收站是空的</strong>
          <span>移入回收站的文件会显示在这里。</span>
        </div>
      </div>
    </ModalDialog>

    <ModalDialog
      :open="Boolean(previewEntry)"
      :title="previewEntry?.name || '文件查看器'"
      :description="previewEntry ? `${previewEntry.path} · ${formatBytes(previewEntry.sizeBytes)}` : ''"
      size="wide"
      allow-fullscreen
      @close="closePreview"
    >
      <div v-if="previewLoading" class="preview-loading"><RefreshCw :size="22" class="spinning" />正在打开文件…</div>
      <div v-else-if="previewEntry && previewMode === 'text'" class="code-viewer">
        <header>
          <span><Code2 :size="15" />{{ previewEntry.mime || 'UTF-8 文本' }}</span>
          <span class="code-viewer__header-right">
            <span>{{ previewEntry.mode }} · {{ previewEntry.owner }}:{{ previewEntry.group }}</span>
            <span class="code-editor-tools">
              <button
                class="code-editor-tool"
                type="button"
                title="查找或替换（Ctrl+F）"
                aria-label="查找或替换"
                @click="codeEditorRef?.openSearch()"
              >
                <Search :size="15" />
              </button>
              <button
                class="code-editor-tool"
                :class="{ 'is-active': editorLineWrap }"
                type="button"
                title="切换自动换行"
                aria-label="切换自动换行"
                :aria-pressed="editorLineWrap"
                @click="editorLineWrap = !editorLineWrap"
              >
                <WrapText :size="15" />
              </button>
            </span>
          </span>
        </header>
        <div class="code-editor">
          <CodeEditor
            ref="codeEditorRef"
            v-model="previewContent"
            :file-name="previewEntry.name"
            :mime="previewEntry.mime"
            :size-bytes="previewEntry.sizeBytes"
            :editable="previewEntry.editable"
            :line-wrap="editorLineWrap"
            @dirty="previewDirty = true"
            @save="savePreview"
            @status="handleEditorStatus"
            @ready="handleEditorReady"
          />
        </div>
        <footer>
          <span>
            {{ editorStatus?.lines || 1 }} 行
            <template v-if="editorStatus"> · 行 {{ editorStatus.line }}，列 {{ editorStatus.column }}</template>
            · UTF-8
            <template v-if="editorInfo">
              · {{ editorInfo.label }}
              {{ editorInfo.highlighted ? '语法着色' : editorInfo.reason === 'large-file' ? '大文件纯文本' : '纯文本' }}
            </template>
          </span>
          <span v-if="previewDirty">有未保存修改</span>
          <span class="code-editor-actions">
            <button class="button button--primary button--small" type="button" :disabled="previewSaving || !previewDirty" @click="savePreview()">
              <Save :size="15" />{{ previewSaving ? '保存中…' : '保存 Ctrl+S' }}
            </button>
          </span>
        </footer>
      </div>
      <div v-else-if="previewEntry" class="media-viewer">
        <img v-if="previewMode === 'image'" :src="previewURL" :alt="previewEntry.name" />
        <audio v-else-if="previewMode === 'audio'" :src="previewURL" controls />
        <video v-else-if="previewMode === 'video'" :src="previewURL" controls />
        <iframe v-else-if="previewMode === 'pdf'" :src="previewURL" :title="previewEntry.name" />
        <div v-else class="metadata-viewer">
          <component :is="entryIcon(previewEntry)" :size="44" />
          <strong>此格式暂不在浏览器内解析</strong>
          <span>{{ previewEntry.mime || '未知格式' }} · {{ formatBytes(previewEntry.sizeBytes) }}</span>
          <button class="button button--primary" type="button" @click="download(previewEntry)">
            <Download :size="16" />下载文件
          </button>
        </div>
      </div>
    </ModalDialog>
  </section>
</template>

<style scoped>
.files-page {
  display: grid;
  gap: 18px;
}

.file-guard {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 10px 15px;
  border: 1px solid color-mix(in srgb, var(--brand) 22%, var(--border));
  border-radius: 13px;
  color: var(--muted);
  background: color-mix(in srgb, var(--brand) 7%, var(--surface));
}

.file-guard svg {
  flex: 0 0 auto;
  color: var(--brand);
}

.file-guard strong {
  color: var(--text);
}

.file-guard small {
  margin-left: auto;
  white-space: nowrap;
}

.file-shortcuts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: -8px;
}

.file-shortcuts button {
  padding: 7px 11px;
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--muted);
  background: var(--surface);
  cursor: pointer;
}

.file-shortcuts button:hover {
  border-color: color-mix(in srgb, var(--brand) 45%, var(--border));
  color: var(--brand);
}

.file-browser {
  position: relative;
  min-height: 540px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.file-browser--dragging {
  border-color: var(--brand);
}

.file-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 13px 15px;
  border-bottom: 1px solid var(--border);
  background: color-mix(in srgb, var(--surface-subtle) 45%, var(--surface));
}

.breadcrumbs {
  display: flex;
  min-width: 0;
  overflow-x: auto;
}

.breadcrumbs button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-width: max-content;
  padding: 7px 4px;
  border: 0;
  color: var(--muted);
  background: transparent;
  cursor: pointer;
}

.breadcrumbs button:last-child {
  color: var(--text);
  font-weight: 700;
}

.breadcrumbs button:disabled {
  cursor: default;
}

.file-search {
  display: flex;
  align-items: center;
  gap: 8px;
  width: min(280px, 34vw);
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}

.file-search input {
  width: 100%;
  height: 36px;
  border: 0;
  outline: 0;
  color: var(--text);
  background: transparent;
}

.file-search button,
.row-menu {
  display: grid;
  place-items: center;
  padding: 5px;
  border: 0;
  border-radius: 7px;
  color: var(--muted);
  background: transparent;
  cursor: pointer;
}

.file-search button:hover,
.row-menu:hover {
  color: var(--text);
  background: var(--surface-subtle);
}

.file-toolbar__controls,
.file-grid-sort,
.file-view-switch {
  display: flex;
  align-items: center;
  gap: 7px;
}

.file-toolbar__controls {
  flex: 0 1 auto;
  justify-content: flex-end;
}

.file-grid-sort,
.file-view-switch {
  min-height: 38px;
  padding: 3px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}

.file-grid-sort select {
  height: 30px;
  padding: 0 6px;
  border: 0;
  color: var(--muted);
  background: transparent;
  outline: none;
}

.file-grid-sort button,
.file-view-switch button {
  display: grid;
  width: 31px;
  height: 30px;
  place-items: center;
  padding: 0;
  border: 0;
  border-radius: 7px;
  color: var(--muted);
  background: transparent;
  cursor: pointer;
}

.file-grid-sort button:hover,
.file-view-switch button:hover,
.file-view-switch button.is-active {
  color: var(--text);
  background: var(--surface-subtle);
}

.file-view-switch button.is-active {
  color: var(--brand);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--brand) 24%, transparent);
}

.batch-bar {
  position: fixed;
  z-index: 45;
  bottom: max(16px, env(safe-area-inset-bottom));
  left: calc(var(--app-shell-inline-offset) + (100vw - var(--app-shell-inline-offset)) / 2);
  width: min(760px, calc(100vw - var(--app-shell-inline-offset) - 32px));
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  overflow-x: auto;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--brand) 30%, var(--border));
  border-radius: 14px;
  background: color-mix(in srgb, var(--brand) 8%, var(--surface));
  box-shadow: 0 12px 34px rgb(0 0 0 / 20%);
  transform: translateX(-50%);
}

.batch-bar strong {
  margin-right: 8px;
}

.batch-bar button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 6px 9px;
  border: 0;
  border-radius: 8px;
  color: var(--muted);
  background: transparent;
  cursor: pointer;
}

.batch-bar button:hover {
  color: var(--text);
  background: var(--surface);
}

.clipboard-bar {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 10px;
  padding: 9px 15px;
  border-bottom: 1px solid color-mix(in srgb, var(--brand) 24%, var(--border));
  background: color-mix(in srgb, var(--brand) 7%, var(--surface));
}

.clipboard-bar__icon {
  width: 30px;
  height: 30px;
  display: grid;
  place-items: center;
  border-radius: 9px;
  color: var(--brand);
  background: color-mix(in srgb, var(--brand) 12%, transparent);
}

.clipboard-bar > span:nth-child(2) {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.clipboard-bar small {
  overflow: hidden;
  color: var(--muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.clipboard-bar button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 6px 10px;
  border: 0;
  border-radius: 8px;
  color: var(--muted);
  background: transparent;
  cursor: pointer;
}

.clipboard-bar button:first-of-type {
  color: #fff;
  background: var(--brand);
}

.clipboard-bar button:hover:not(:disabled) {
  color: var(--text);
  background: var(--surface);
}

.clipboard-bar button:first-of-type:hover:not(:disabled) {
  color: #fff;
  background: var(--brand-strong);
}

.clipboard-bar button:disabled {
  opacity: .6;
  cursor: wait;
}

.danger-link {
  color: var(--danger) !important;
}

.upload-strip {
  display: grid;
  gap: 7px;
  padding: 10px 15px;
  border-bottom: 1px solid var(--border);
}

.upload-strip > div {
  display: grid;
  grid-template-columns: minmax(120px, 220px) 1fr 42px;
  align-items: center;
  gap: 10px;
  font-size: 12px;
}

.upload-strip > div > div {
  height: 4px;
  overflow: hidden;
  border-radius: 99px;
  background: var(--surface-subtle);
}

.upload-strip i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--brand);
}

.file-table {
  display: grid;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(166px, 1fr));
  align-content: start;
  gap: 12px;
  min-height: 430px;
  padding: 16px;
  -webkit-user-select: none;
  user-select: none;
}

.file-grid-card {
  position: relative;
  display: grid;
  min-width: 0;
  gap: 7px;
  padding: 10px;
  border: 1px solid transparent;
  border-radius: 13px;
  color: var(--text);
  background: transparent;
  cursor: default;
  outline: none;
  transition: border-color 0.14s ease, background-color 0.14s ease, box-shadow 0.14s ease;
}

.file-grid-card:hover,
.file-grid-card:focus-visible {
  border-color: var(--border);
  background: var(--surface-subtle);
}

.file-grid-card--selected {
  border-color: color-mix(in srgb, var(--brand) 48%, var(--border));
  background: color-mix(in srgb, var(--brand) 8%, var(--surface));
  box-shadow: 0 7px 20px color-mix(in srgb, var(--brand) 8%, transparent);
}

.file-grid-card__check,
.file-grid-card__menu {
  position: absolute;
  z-index: 2;
  top: 16px;
  opacity: 0;
  transition: opacity 0.12s ease;
}

.file-grid-card__check {
  left: 16px;
  width: 16px;
  height: 16px;
  accent-color: var(--brand);
}

.file-grid-card__menu {
  right: 16px;
  color: var(--text);
  background: color-mix(in srgb, var(--surface) 88%, transparent);
  box-shadow: 0 2px 8px rgb(0 0 0 / 10%);
}

.file-grid-card:hover .file-grid-card__check,
.file-grid-card:hover .file-grid-card__menu,
.file-grid-card:focus-within .file-grid-card__check,
.file-grid-card:focus-within .file-grid-card__menu,
.file-grid-card--selected .file-grid-card__check,
.file-grid-card--selected .file-grid-card__menu {
  opacity: 1;
}

.file-grid-card__visual {
  display: grid;
  width: 100%;
  aspect-ratio: 4 / 3;
  place-items: center;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--border) 72%, transparent);
  border-radius: 10px;
  background:
    linear-gradient(45deg, color-mix(in srgb, var(--surface-subtle) 75%, transparent) 25%, transparent 25%) 0 0 / 16px 16px,
    linear-gradient(-45deg, color-mix(in srgb, var(--surface-subtle) 75%, transparent) 25%, transparent 25%) 0 0 / 16px 16px,
    var(--surface);
}

.file-grid-card__visual img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.file-grid-card__icon {
  display: grid;
  width: 76px;
  height: 76px;
  place-items: center;
  border-radius: 20px;
  color: var(--blue);
  background: color-mix(in srgb, var(--blue) 10%, var(--surface));
}

.file-grid-card__icon--folder {
  color: var(--brand);
  background: color-mix(in srgb, var(--brand) 11%, var(--surface));
}

.file-grid-card > strong,
.file-grid-card > small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-grid-card > strong {
  padding: 0 2px;
  font-size: 13px;
}

.file-grid-card > small {
  display: flex;
  gap: 5px;
  padding: 0 2px 2px;
  color: var(--muted);
  font-size: 11px;
}

.file-row {
  display: grid;
  grid-template-columns: 42px minmax(220px, 2fr) minmax(85px, 0.6fr) minmax(100px, 0.8fr) minmax(110px, 0.8fr) minmax(125px, 0.8fr) 46px;
  align-items: center;
  width: 100%;
  min-height: 58px;
  padding: 0 12px;
  border: 0;
  border-bottom: 1px solid var(--border);
  color: var(--text);
  text-align: left;
  background: var(--surface);
  -webkit-user-select: none;
  user-select: none;
}

.file-row--header {
  min-height: 40px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  background: var(--surface-subtle);
}

.file-row--header > button {
  padding: 8px;
  border: 0;
  color: var(--muted);
  font: inherit;
  text-align: left;
  background: transparent;
  cursor: pointer;
}

.file-row--header > button:hover {
  color: var(--text);
}

.file-row--entry {
  font: inherit;
  cursor: default;
}

.file-row--entry:hover,
.file-row--selected {
  background: color-mix(in srgb, var(--brand) 6%, var(--surface));
}

.file-row > span {
  min-width: 0;
  padding: 8px;
  color: var(--muted);
  font-size: 13px;
}

.file-row input {
  accent-color: var(--brand);
}

.file-name {
  display: flex;
  align-items: center;
  gap: 11px;
  color: var(--text) !important;
  cursor: pointer;
}

.file-name > span:last-child {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.file-name strong,
.file-name small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-name small,
.file-row > span > small {
  color: var(--muted);
  font-size: 11px;
}

.file-icon {
  display: grid;
  flex: 0 0 34px;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 9px;
  color: var(--blue);
  background: color-mix(in srgb, var(--blue) 11%, var(--surface));
}

.file-icon--folder {
  color: var(--brand);
  background: color-mix(in srgb, var(--brand) 11%, var(--surface));
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 12px !important;
}

.file-empty,
.file-loading {
  display: grid;
  place-items: center;
  align-content: center;
  gap: 8px;
  min-height: 360px;
  color: var(--muted);
}

.file-empty strong {
  color: var(--text);
}

.file-limit {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 10px 15px;
  color: var(--amber);
  font-size: 12px;
  text-align: center;
}

.file-limit .button {
  min-height: 32px;
  padding: 6px 12px;
}

.drop-overlay {
  position: absolute;
  z-index: 4;
  inset: 10px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
  border: 2px dashed var(--brand);
  border-radius: 14px;
  color: var(--brand);
  background: color-mix(in srgb, var(--surface) 90%, transparent);
  backdrop-filter: blur(5px);
  pointer-events: none;
}

.file-context-menu {
  position: fixed;
  z-index: 90;
  display: grid;
  width: 196px;
  padding: 6px;
  border: 1px solid var(--border);
  border-radius: 11px;
  background: var(--surface);
  box-shadow: var(--shadow-md);
}

.file-context-menu button {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 8px 10px;
  border: 0;
  border-radius: 7px;
  color: var(--text);
  text-align: left;
  background: transparent;
  cursor: pointer;
}

.file-context-menu button:hover {
  background: var(--surface-subtle);
}

.file-context-menu hr {
  width: 100%;
  margin: 4px 0;
  border: 0;
  border-top: 1px solid var(--border);
}

.operation-form {
  display: grid;
  gap: 14px;
}

.operation-form label {
  display: grid;
  gap: 8px;
}

.operation-form label > span {
  font-weight: 700;
}

.operation-form input,
.operation-form select {
  width: 100%;
  height: 42px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  color: var(--text);
  background: var(--surface);
  outline: none;
}

.operation-form input:focus,
.operation-form select:focus {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--brand) 13%, transparent);
}

.archive-hint {
  color: var(--muted);
  line-height: 1.6;
}

.trash-summary {
  display: grid;
  place-items: center;
  gap: 8px;
  padding: 16px 0 8px;
  text-align: center;
}

.trash-summary svg {
  color: var(--danger);
}

.trash-summary span {
  max-width: 380px;
  color: var(--muted);
  font-size: 13px;
}

.trash-manager {
  display: grid;
  gap: 12px;
}

.trash-manager > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.trash-manager > header > div {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.trash-list {
  max-height: min(56vh, 560px);
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: 12px;
}

.trash-item {
  display: grid;
  grid-template-columns: 28px 38px minmax(180px, 1fr) 100px 128px;
  align-items: center;
  gap: 8px;
  min-height: 62px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  cursor: pointer;
}

.trash-item:last-child {
  border-bottom: 0;
}

.trash-item:hover {
  background: var(--surface-subtle);
}

.trash-item > span:nth-child(3) {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.trash-item strong,
.trash-item small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.trash-item small,
.trash-item > span:nth-last-child(-n + 2) {
  color: var(--muted);
  font-size: 12px;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
  padding-top: 20px;
}

.preview-loading {
  display: flex;
  min-height: 420px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--muted);
}

.code-viewer {
  overflow: hidden;
  border: 1px solid var(--terminal-shell-border, #29383a);
  border-radius: var(--terminal-shell-radius, 12px);
  background: var(--terminal-shell-background, #0b1214);
  box-shadow: var(--terminal-shell-shadow, inset 0 1px 0 rgb(255 255 255 / 3%));
}

.code-viewer > header,
.code-viewer > footer {
  display: flex;
  min-height: 42px;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 7px 12px;
  color: var(--terminal-shell-muted, #8a9695);
  font-size: 12px;
  background: var(--terminal-shell-panel, #111a1d);
}

.code-viewer > header > span:first-child {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--terminal-shell-text, #d8dddc);
}

.code-viewer__header-right,
.code-editor-tools {
  display: flex;
  align-items: center;
  gap: 8px;
}

.code-editor-tools {
  gap: 3px;
}

.code-editor-tool {
  display: inline-grid;
  width: 28px;
  height: 28px;
  place-items: center;
  padding: 0;
  border: 0;
  border-radius: 6px;
  color: var(--terminal-shell-muted, #8a9695);
  background: transparent;
  cursor: pointer;
}

.code-editor-tool:hover,
.code-editor-tool.is-active {
  color: var(--terminal-shell-text, #d8dddc);
  background: var(--terminal-shell-panel-raised, #182326);
}

.code-editor-tool.is-active {
  color: var(--brand, #35cba6);
}

.code-viewer > footer {
  justify-content: flex-start;
  flex-wrap: wrap;
}

.code-editor-actions {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-left: auto;
}

.code-editor-actions .button {
  min-height: 30px;
  padding: 5px 9px;
}

.code-editor {
  position: relative;
  height: min(60vh, 620px);
  overflow: hidden;
}

.code-editor > * {
  height: 100%;
}

.media-viewer {
  display: grid;
  min-height: 480px;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 12px;
  background:
    linear-gradient(45deg, var(--surface-subtle) 25%, transparent 25%) 0 0 / 20px 20px,
    linear-gradient(-45deg, var(--surface-subtle) 25%, transparent 25%) 0 0 / 20px 20px,
    var(--surface);
}

.media-viewer img,
.media-viewer video {
  max-width: 100%;
  max-height: 68vh;
}

.media-viewer audio {
  width: min(620px, 84%);
}

.media-viewer iframe {
  width: 100%;
  height: 68vh;
  border: 0;
  background: #fff;
}

.metadata-viewer {
  display: grid;
  place-items: center;
  gap: 11px;
  padding: 40px;
  color: var(--muted);
  text-align: center;
}

.metadata-viewer strong {
  color: var(--text);
}

.spinning {
  animation: spin 0.9s linear infinite;
}

.slide-enter-active,
.slide-leave-active {
  transition: 0.16s ease;
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

.batch-dock-enter-active,
.batch-dock-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.batch-dock-enter-from,
.batch-dock-leave-to {
  opacity: 0;
  transform: translate(-50%, 10px);
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1040px) {
  .file-row {
    grid-template-columns: 42px minmax(220px, 2fr) 90px 118px 46px;
  }

  .file-row > span:nth-child(4),
  .file-row > span:nth-child(5) {
    display: none;
  }
}

@media (max-width: 920px) {
  .batch-bar {
    left: 50%;
    width: calc(100vw - 32px);
  }
}

@media (max-width: 720px) {
  .file-guard {
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .file-guard small {
    width: 100%;
    margin-left: 28px;
    white-space: normal;
  }

  .file-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .file-search {
    width: 100%;
  }

  .file-toolbar__controls {
    width: 100%;
    flex-wrap: wrap;
  }

  .file-toolbar__controls .file-search {
    flex: 1 1 100%;
  }

  .file-grid-sort {
    margin-right: auto;
  }

  .file-grid {
    grid-template-columns: repeat(auto-fill, minmax(138px, 1fr));
    gap: 8px;
    padding: 10px;
  }

  .file-grid-card {
    padding: 8px;
  }

  .file-grid-card__check,
  .file-grid-card__menu {
    top: 13px;
    opacity: 1;
  }

  .file-grid-card__check {
    left: 13px;
  }

  .file-grid-card__menu {
    right: 13px;
  }

  .batch-bar button,
  .batch-bar strong {
    flex: 0 0 auto;
  }

  .batch-bar {
    bottom: max(10px, env(safe-area-inset-bottom));
    width: calc(100vw - 20px);
    border-radius: 12px;
  }

  .clipboard-bar {
    grid-template-columns: auto minmax(0, 1fr) auto;
  }

  .clipboard-bar button:first-of-type {
    grid-column: 1 / -1;
    justify-content: center;
    grid-row: 2;
  }

  .file-row {
    grid-template-columns: 38px minmax(180px, 1fr) 46px;
  }

  .file-row > :nth-child(3),
  .file-row > :nth-child(4),
  .file-row > :nth-child(5),
  .file-row > :nth-child(6) {
    display: none;
  }

  .code-editor {
    grid-template-columns: 44px 1fr;
  }

  .code-editor-actions {
    margin-left: auto;
  }

  .code-viewer__header-right > span:first-child {
    display: none;
  }

  .trash-manager > header {
    align-items: stretch;
    flex-direction: column;
  }

  .trash-manager > header > div {
    justify-content: flex-start;
  }

  .trash-item {
    grid-template-columns: 26px 34px minmax(0, 1fr);
  }

  .trash-item > span:nth-last-child(-n + 2) {
    display: none;
  }
}
</style>
