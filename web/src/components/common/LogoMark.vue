<script setup lang="ts">
withDefaults(
  defineProps<{
    compact?: boolean
  }>(),
  {
    compact: false,
  },
)

// A bound public URL avoids test/build tooling treating the root-relative path
// as a local file URL on Windows.
const logoPath = '/icons/kpanel.svg'
</script>

<template>
  <div class="brand" :class="{ 'brand--compact': compact }">
    <span class="brand__mark" aria-hidden="true">
      <img :src="logoPath" alt="" width="38" height="38" />
    </span>
    <span v-if="!compact" class="brand__text">
      <strong>KPanel</strong>
    </span>
  </div>
</template>
