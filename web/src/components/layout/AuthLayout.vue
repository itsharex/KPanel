<script setup lang="ts">
import { CheckCircle2, ShieldCheck } from '@lucide/vue'
import LogoMark from '@/components/common/LogoMark.vue'
</script>

<template>
  <main class="auth-layout">
    <section class="auth-layout__brand">
      <LogoMark />
      <div class="auth-layout__message">
        <span class="eyebrow"><ShieldCheck :size="16" /> kejilion.sh 同生态管理</span>
        <h1>脚本能管理的，网页也能管理。</h1>
        <p>直接读取宿主机真实产物，通过结构化 Agent 执行完整业务动作，不以资源来源限制管理员。</p>
        <ul>
          <li><CheckCircle2 :size="17" /> 脚本与 Web 产物双向互通</li>
          <li><CheckCircle2 :size="17" /> Agent 离线时明确显示依赖状态</li>
          <li><CheckCircle2 :size="17" /> 每项变更均有记录与审计追踪</li>
        </ul>
      </div>
      <p class="auth-layout__footnote">KPanel · Linux 管理面板</p>
    </section>
    <section class="auth-layout__form">
      <div class="auth-card">
        <slot />
      </div>
    </section>
  </main>
</template>
