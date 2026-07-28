<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { Maximize2, Minimize2, X } from '@lucide/vue'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    size?: 'small' | 'medium' | 'large' | 'wide'
    allowFullscreen?: boolean
  }>(),
  {
    description: '',
    size: 'medium',
    allowFullscreen: false,
  },
)

const emit = defineEmits<{
  close: []
}>()
const fullscreen = ref(false)

function close(): void {
  fullscreen.value = false
  emit('close')
}

function onKeyDown(event: KeyboardEvent): void {
  if (event.key === 'Escape') close()
}

watch(
  () => props.open,
  (open) => {
    document.body.classList.toggle('has-modal', open)
    if (open) window.addEventListener('keydown', onKeyDown)
    else window.removeEventListener('keydown', onKeyDown)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  document.body.classList.remove('has-modal')
  window.removeEventListener('keydown', onKeyDown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="modal-backdrop" role="presentation" @mousedown.self="close">
        <section
          class="modal-panel"
          :class="[`modal-panel--${size}`, { 'modal-panel--fullscreen': fullscreen }]"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
        >
          <header class="modal-panel__header">
            <div>
              <h2>{{ title }}</h2>
              <p v-if="description">{{ description }}</p>
            </div>
            <div class="modal-panel__actions">
              <button
                v-if="allowFullscreen"
                class="icon-button"
                type="button"
                :aria-label="fullscreen ? '退出全屏' : '全屏显示'"
                @click="fullscreen = !fullscreen"
              >
                <Minimize2 v-if="fullscreen" :size="18" />
                <Maximize2 v-else :size="18" />
              </button>
              <button class="icon-button" type="button" aria-label="关闭对话框" @click="close">
                <X :size="19" />
              </button>
            </div>
          </header>
          <div class="modal-panel__body">
            <slot />
          </div>
          <footer v-if="$slots.footer" class="modal-panel__footer">
            <slot name="footer" />
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>
