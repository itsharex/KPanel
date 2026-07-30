<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { EditorView as EditorViewType } from '@codemirror/view'
import { loadCodeLanguage, type CodeLanguage } from '@/lib/code-editor-language'

const props = withDefaults(
  defineProps<{
    modelValue: string
    fileName: string
    mime?: string
    sizeBytes: number
    editable?: boolean
  }>(),
  {
    mime: '',
    editable: true,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  dirty: []
  save: []
  ready: [value: CodeLanguage & { loadMs: number }]
}>()

const host = ref<HTMLElement>()
const loading = ref(true)
const loadError = ref('')
let editor: EditorViewType | undefined
let applyingExternalValue = false
let cancelled = false

async function initialize(): Promise<void> {
  const startedAt = performance.now()
  try {
    const [
      { EditorState },
      {
        EditorView,
        drawSelection,
        dropCursor,
        highlightActiveLine,
        highlightActiveLineGutter,
        highlightSpecialChars,
        keymap,
        lineNumbers,
      },
      { defaultKeymap, history, historyKeymap, indentWithTab },
      { bracketMatching, HighlightStyle, indentOnInput, syntaxHighlighting },
      { tags },
      language,
    ] = await Promise.all([
      import('@codemirror/state'),
      import('@codemirror/view'),
      import('@codemirror/commands'),
      import('@codemirror/language'),
      import('@lezer/highlight'),
      loadCodeLanguage(props.fileName, props.mime, props.sizeBytes),
    ])
    if (cancelled || !host.value) return

    const highlightStyle = HighlightStyle.define([
      { tag: tags.comment, color: 'var(--code-comment)' },
      { tag: [tags.keyword, tags.controlKeyword, tags.operatorKeyword], color: 'var(--code-keyword)' },
      { tag: [tags.string, tags.special(tags.string)], color: 'var(--code-string)' },
      { tag: [tags.number, tags.bool, tags.null], color: 'var(--code-number)' },
      { tag: [tags.function(tags.variableName), tags.definition(tags.variableName)], color: 'var(--code-function)' },
      { tag: [tags.typeName, tags.className], color: 'var(--code-type)' },
      { tag: tags.tagName, color: 'var(--code-tag)' },
      { tag: [tags.attributeName, tags.propertyName], color: 'var(--code-property)' },
      { tag: tags.invalid, color: 'var(--danger)', textDecoration: 'underline' },
    ])

    const extensions = [
      lineNumbers(),
      highlightActiveLineGutter(),
      highlightSpecialChars(),
      history(),
      drawSelection(),
      dropCursor(),
      indentOnInput(),
      bracketMatching(),
      highlightActiveLine(),
      syntaxHighlighting(highlightStyle),
      EditorState.tabSize.of(2),
      EditorState.readOnly.of(!props.editable),
      EditorView.editable.of(props.editable),
      EditorView.contentAttributes.of({ 'aria-label': '文件内容' }),
      EditorView.updateListener.of((update) => {
        if (!update.docChanged || applyingExternalValue) return
        emit('update:modelValue', update.state.doc.toString())
        emit('dirty')
      }),
      EditorView.theme({
        '&': {
          height: '100%',
          color: 'var(--code-text)',
          backgroundColor: 'var(--code-background)',
          fontSize: '13px',
        },
        '.cm-scroller': {
          overflow: 'auto',
          fontFamily: 'ui-monospace, SFMono-Regular, Consolas, monospace',
          lineHeight: '1.65',
        },
        '.cm-content': {
          minWidth: 'max-content',
          padding: '14px 0',
          caretColor: 'var(--code-caret)',
        },
        '.cm-line': { padding: '0 16px' },
        '.cm-gutters': {
          minWidth: '52px',
          color: 'var(--code-line-number)',
          backgroundColor: 'var(--code-gutter)',
          border: '0',
        },
        '.cm-lineNumbers .cm-gutterElement': {
          minWidth: '46px',
          padding: '0 10px 0 6px',
        },
        '.cm-activeLine': { backgroundColor: 'var(--code-active-line)' },
        '.cm-activeLineGutter': {
          color: 'var(--code-text)',
          backgroundColor: 'var(--code-active-line)',
        },
        '&.cm-focused': { outline: 'none' },
        '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
          backgroundColor: 'var(--code-selection) !important',
        },
      }),
      keymap.of([
        {
          key: 'Mod-s',
          preventDefault: true,
          run: () => {
            emit('save')
            return true
          },
        },
        indentWithTab,
        ...defaultKeymap,
        ...historyKeymap,
      ]),
    ]
    if (language.extension) extensions.push(language.extension)

    editor = new EditorView({
      parent: host.value,
      state: EditorState.create({ doc: props.modelValue, extensions }),
    })
    editor.focus()
    emit('ready', { ...language, loadMs: performance.now() - startedAt })
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '编辑器加载失败'
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  (value) => {
    if (!editor || editor.state.doc.toString() === value) return
    applyingExternalValue = true
    editor.dispatch({
      changes: { from: 0, to: editor.state.doc.length, insert: value },
    })
    applyingExternalValue = false
  },
)

onMounted(() => {
  void initialize()
})

onBeforeUnmount(() => {
  cancelled = true
  editor?.destroy()
  editor = undefined
})
</script>

<template>
  <div class="code-editor-shell">
    <div ref="host" class="code-editor-host" />
    <div v-if="loading" class="code-editor-state">正在加载代码编辑器…</div>
    <div v-else-if="loadError" class="code-editor-state code-editor-state--error">
      代码编辑器加载失败：{{ loadError }}
    </div>
  </div>
</template>

<style scoped>
.code-editor-shell {
  --code-background: #0b1120;
  --code-gutter: #0e1728;
  --code-text: #d8e3f5;
  --code-caret: #5eead4;
  --code-line-number: #60708b;
  --code-active-line: rgb(94 234 212 / 7%);
  --code-selection: rgb(59 130 246 / 32%);
  --code-comment: #718096;
  --code-keyword: #d58cff;
  --code-string: #8fd694;
  --code-number: #f2b36f;
  --code-function: #73b7ff;
  --code-type: #f5d76e;
  --code-tag: #ff7b8b;
  --code-property: #5eead4;
  position: relative;
  height: 100%;
  overflow: hidden;
  background: var(--code-background);
}

.code-editor-host {
  height: 100%;
}

.code-editor-state {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: var(--code-line-number);
  background: var(--code-background);
}

.code-editor-state--error {
  color: var(--danger);
}
</style>
