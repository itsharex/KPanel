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
  siManjaro,
  siOpensuse,
  siRedhat,
  siRockylinux,
  siSuse,
  siUbuntu,
  type SimpleIcon,
} from 'simple-icons'
import oracleIcon from '@/assets/os/oracle.png'

const props = defineProps<{
  distro: string
  label: string
}>()

interface OperatingSystemMark {
  icon?: SimpleIcon
  image?: string
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
  rhel: { icon: siRedhat },
  manjaro: { icon: siManjaro },
  arch: { icon: siArchlinux },
  opensuse: { icon: siOpensuse },
  suse: { icon: siSuse },
  oracle: { image: oracleIcon, accent: 'C74634' },
  linux: linuxMark,
}

const mark = computed<OperatingSystemMark>(() =>
  Object.prototype.hasOwnProperty.call(icons, props.distro) ? icons[props.distro]! : linuxMark,
)
const style = computed(() => ({
  '--os-accent': `#${mark.value.accent || mark.value.icon?.hex || siLinux.hex}`,
  '--os-foreground': mark.value.foreground || '#ffffff',
}))
</script>

<template>
  <span
    class="os-identity__mark"
    :title="props.label"
    :style="style"
    aria-hidden="true"
  >
    <img v-if="mark.image" :src="mark.image" alt="" aria-hidden="true" />
    <svg v-else viewBox="0 0 24 24" aria-hidden="true">
      <path :d="mark.icon?.path || siLinux.path" />
    </svg>
  </span>
</template>
