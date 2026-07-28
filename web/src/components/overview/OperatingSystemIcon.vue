<script setup lang="ts">
import almaIcon from '@/assets/os/alma.png'
import alpineIcon from '@/assets/os/alpine.png'
import archIcon from '@/assets/os/arch.png'
import centosIcon from '@/assets/os/centos.png'
import debianIcon from '@/assets/os/debian.png'
import fedoraIcon from '@/assets/os/fedora.png'
import linuxIcon from '@/assets/os/linux.png'
import oracleIcon from '@/assets/os/oracle.png'
import rockyIcon from '@/assets/os/rocky.png'
import suseIcon from '@/assets/os/suse.png'
import ubuntuIcon from '@/assets/os/ubuntu.png'

const props = defineProps<{
  distro: string
  label: string
}>()

const icons: Record<string, string> = {
  ubuntu: ubuntuIcon,
  debian: debianIcon,
  centos: centosIcon,
  rocky: rockyIcon,
  alma: almaIcon,
  fedora: fedoraIcon,
  alpine: alpineIcon,
  arch: archIcon,
  suse: suseIcon,
  oracle: oracleIcon,
  linux: linuxIcon,
}
</script>

<template>
  <span class="os-identity__mark" :title="props.label">
    <img :src="icons[props.distro] || linuxIcon" :alt="props.label" width="28" height="28" />
  </span>
</template>
