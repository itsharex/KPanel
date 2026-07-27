<script setup lang="ts">
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

const icons: Record<string, SimpleIcon> = {
  ubuntu: siUbuntu,
  debian: siDebian,
  centos: siCentos,
  rocky: siRockylinux,
  alma: siAlmalinux,
  fedora: siFedora,
  alpine: siAlpinelinux,
  arch: siArchlinux,
  suse: siOpensuse,
  oracle: siLinux,
  linux: siLinux,
}
</script>

<template>
  <span class="os-identity__mark" :class="`is-${props.distro}`" :title="props.label">
    <svg viewBox="0 0 24 24" role="img" :aria-label="props.label">
      <path :d="(icons[props.distro] || siLinux).path" />
    </svg>
  </span>
</template>
