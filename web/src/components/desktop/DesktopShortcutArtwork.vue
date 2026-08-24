<script setup lang="ts">
import type { Component } from 'vue'

const DIRECTORY_ARTWORK_URL = '/desktop-icons/folder-open-shortcut-kpanel-flat-v1.webp'

defineProps<{
  kind: 'file' | 'directory'
  icon?: Component
}>()
</script>

<template>
  <span
    class="desktop__shortcut-artwork"
    :class="`desktop__shortcut-artwork--${kind}`"
    aria-hidden="true"
  >
    <img
      v-if="kind === 'directory'"
      class="desktop__shortcut-directory-image"
      :src="DIRECTORY_ARTWORK_URL"
      alt=""
      draggable="false"
      width="58"
      height="58"
    />
    <component
      v-else-if="icon"
      :is="icon"
      :size="25"
      :stroke-width="1.8"
    />
  </span>
</template>
