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
    <span class="brand__mark" aria-hidden="true"><span>K</span></span>
    <span v-if="!compact" class="brand__text">
      <strong>KPanel</strong>
    </span>
  </div>
</template>
