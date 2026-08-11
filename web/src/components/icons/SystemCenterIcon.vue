<script setup lang="ts">
withDefaults(
  defineProps<{
    size?: number | string
    strokeWidth?: number | string
  }>(),
  {
    size: 24,
    strokeWidth: 1.8,
  },
)
</script>

<template>
  <svg
    class="system-center-icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path d="M8.25 4.5H5.6a1.1 1.1 0 0 0-1.1 1.1v2.65" />
    <path d="M15.75 4.5h2.65a1.1 1.1 0 0 1 1.1 1.1v2.65" />
    <path d="M19.5 15.75v2.65a1.1 1.1 0 0 1-1.1 1.1h-2.65" />
    <path d="M8.25 19.5H5.6a1.1 1.1 0 0 1-1.1-1.1v-2.65" />
    <circle cx="12" cy="12" r="3.25" />
    <path d="M12 6.75v2M17.25 12h-2M12 17.25v-2M6.75 12h2" />
    <circle cx="12" cy="12" r="1" fill="currentColor" stroke="none" />
  </svg>
</template>
