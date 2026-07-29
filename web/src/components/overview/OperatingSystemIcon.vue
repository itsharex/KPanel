<script setup lang="ts">
import { computed } from 'vue'
import {
  siAlmalinux,
  siAlpinelinux,
  siArchlinux,
  siCentos,
  siDebian,
  siFedora,
  siLinux,
  siOpensuse,
  siRockylinux,
  siUbuntu,
  type SimpleIcon,
} from 'simple-icons'

const props = defineProps<{
  distro: string
  label: string
}>()

interface OperatingSystemMark {
  icon: SimpleIcon
  accent?: string
  foreground?: string
}

const linuxMark: OperatingSystemMark = {
  icon: siLinux,
  foreground: '#141816',
}

const icons: Record<string, OperatingSystemMark> = {
  ubuntu: { icon: siUbuntu },
  debian: { icon: siDebian },
  centos: { icon: siCentos },
  rocky: { icon: siRockylinux },
  alma: { icon: siAlmalinux },
  fedora: { icon: siFedora },
  alpine: { icon: siAlpinelinux },
  arch: { icon: siArchlinux },
  suse: { icon: siOpensuse },
  oracle: { icon: siLinux, accent: 'C74634' },
  linux: linuxMark,
}

const mark = computed<OperatingSystemMark>(() =>
  Object.prototype.hasOwnProperty.call(icons, props.distro) ? icons[props.distro]! : linuxMark,
)
const style = computed(() => ({
  '--os-accent': `#${mark.value.accent || mark.value.icon.hex}`,
  '--os-foreground': mark.value.foreground || '#ffffff',
}))
</script>

<template>
  <span class="os-identity__mark" :title="props.label" :style="style">
    <svg viewBox="0 0 24 24" role="img" :aria-label="props.label">
      <path :d="mark.icon.path" />
    </svg>
  </span>
</template>
