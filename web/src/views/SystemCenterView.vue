<script setup lang="ts">
import { usePhraseCatalog } from '@/i18n/phrase'
import OverviewView from './OverviewView.vue'

usePhraseCatalog(() => import('@/i18n/pages/SystemCenterView/en-US').then((module) => module.default))
</script>

<template>
  <OverviewView system-center-only />
</template>
