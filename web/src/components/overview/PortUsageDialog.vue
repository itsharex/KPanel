<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { RefreshCw, Search } from '@lucide/vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import { ApiError, api } from '@/lib/api'
import type { PortUsageSnapshot } from '@/types/api'

const props = withDefaults(defineProps<{ open: boolean; readable: boolean; unavailableReason?: string }>(), {
  unavailableReason: '',
})
const emit = defineEmits<{ close: [] }>()
const snapshot = ref<PortUsageSnapshot>()
const loading = ref(false)
const refreshing = ref(false)
const error = ref('')
const search = ref('')
let controller: AbortController | undefined

const entries = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return snapshot.value?.entries || []
  return (snapshot.value?.entries || []).filter((entry) =>
    [entry.protocol, entry.state, entry.localAddress, entry.localPort, entry.process, String(entry.pid || ''), entry.raw]
      .some((value) => String(value || '').toLowerCase().includes(keyword)),
  )
})

function endpoint(address: string, port: string): string {
  const host = address.includes(':') && !address.startsWith('[') ? `[${address}]` : address
  return `${host || '*'}:${port || '*'}`
}

async function load(silent = false): Promise<void> {
  if (!props.open || !props.readable) return
  controller?.abort()
  controller = new AbortController()
  if (silent) refreshing.value = true
  else loading.value = true
  error.value = ''
  try {
    snapshot.value = await api.system.portUsage(controller.signal)
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return
    error.value = reason instanceof ApiError ? reason.message : '无法读取端口占用状态。'
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

watch(() => [props.open, props.readable] as const, ([open, readable]) => {
  if (open && readable) void load()
  else controller?.abort()
}, { immediate: true })

onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <ModalDialog
    :open="open"
    title="端口占用查看"
    description="通过 kejilion.sh 读取当前 TCP / UDP 监听端口、进程和 PID。"
    size="wide"
    @close="emit('close')"
  >
    <div class="system-resource-dialog">
      <div v-if="!readable" class="inline-alert inline-alert--warning">
        {{ unavailableReason || '当前 Agent 的端口占用适配器未就绪。' }}
      </div>
      <LoadingState v-else-if="loading && !snapshot" :rows="5" />
      <ErrorState v-else-if="error && !snapshot" :message="error" @retry="load()" />
      <template v-else-if="snapshot">
        <header class="system-resource-dialog__summary">
          <span>共 {{ snapshot.total }} 条监听记录</span>
          <span v-if="snapshot.truncated" class="text-warning">仅显示前 512 条</span>
          <button class="icon-button" type="button" :disabled="refreshing" title="刷新端口占用" aria-label="刷新端口占用" @click="load(true)">
            <RefreshCw :size="16" :class="{ spin: refreshing }" />
          </button>
        </header>
        <label class="field system-resource-search">
          <span>筛选</span>
          <span class="system-resource-search__input"><Search :size="15" /><input v-model="search" autocomplete="off" placeholder="端口、地址、协议、进程或 PID" /></span>
        </label>
        <EmptyState v-if="!entries.length" title="没有匹配的监听端口" description="调整筛选条件或刷新后重试。" />
        <div v-else class="system-resource-list">
          <article v-for="(entry, index) in entries" :key="`${index}-${entry.raw}`" class="system-resource-item">
            <div class="system-resource-item__main">
              <strong>{{ endpoint(entry.localAddress, entry.localPort) }}</strong>
              <span>{{ entry.process || '未识别进程' }}<template v-if="entry.pid"> · PID {{ entry.pid }}</template></span>
              <code>{{ entry.raw }}</code>
            </div>
            <span class="system-resource-state is-on">{{ entry.protocol.toUpperCase() }}</span>
            <span class="system-resource-item__meta">{{ entry.state }}</span>
          </article>
        </div>
      </template>
    </div>
  </ModalDialog>
</template>
