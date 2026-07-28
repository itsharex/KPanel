<script setup lang="ts">
withDefaults(
  defineProps<{
    compact?: boolean
  }>(),
  {
    compact: false,
  },
)
</script>

<template>
  <div class="brand" :class="{ 'brand--compact': compact }">
    <span class="brand__mark" aria-hidden="true">
      <img src="/icons/kpanel.svg" alt="" width="38" height="38" />
    </span>
    <span v-if="!compact" class="brand__text">
      <strong>KPanel</strong>
    </span>
  </div>
</template>
