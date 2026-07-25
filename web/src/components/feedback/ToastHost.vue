<script setup lang="ts">
import { CheckCircle2, CircleAlert, Info, X } from '@lucide/vue'
import { useToast } from '@/stores/toast'

const { items, remove } = useToast()
</script>

<template>
  <div class="toast-region" aria-live="polite" aria-atomic="false">
    <TransitionGroup name="toast">
      <article v-for="item in items" :key="item.id" class="toast" :class="`toast--${item.tone}`">
        <CheckCircle2 v-if="item.tone === 'success'" :size="19" aria-hidden="true" />
        <CircleAlert v-else-if="item.tone === 'danger'" :size="19" aria-hidden="true" />
        <Info v-else :size="19" aria-hidden="true" />
        <div class="toast__body">
          <strong>{{ item.title }}</strong>
          <p v-if="item.message">{{ item.message }}</p>
        </div>
        <button class="icon-button icon-button--small" type="button" aria-label="关闭通知" @click="remove(item.id)">
          <X :size="16" />
        </button>
      </article>
    </TransitionGroup>
  </div>
</template>
