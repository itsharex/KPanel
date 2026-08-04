<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'

const props=defineProps<{content:string}>()
const parser=new MarkdownIt({html:false,linkify:true,typographer:false,highlight:()=>''})
const rendered=computed(()=>DOMPurify.sanitize(parser.render(props.content),{USE_PROFILES:{html:true},FORBID_TAGS:['style','iframe','form'],FORBID_ATTR:['style']}))
</script>
<template><div class="ai-markdown" v-html="rendered" /></template>
