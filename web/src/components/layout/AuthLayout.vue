<script setup lang="ts">
import { CheckCircle2, ShieldCheck } from '@lucide/vue'
import LogoMark from '@/components/common/LogoMark.vue'
</script>

<template>
  <main class="auth-layout">
    <section class="auth-layout__brand">
      <LogoMark />
      <div class="auth-layout__message">
        <span class="eyebrow"><ShieldCheck :size="16" /> 安全的宿主机管理边界</span>
        <h1>让 Kejilion 的日常管理，更直观、更安心。</h1>
        <p>读取真实系统状态，所有变更经过白名单 Agent 校验，不执行任意 Shell。</p>
        <ul>
          <li><CheckCircle2 :size="17" /> 不修改现有 kejilion.sh</li>
          <li><CheckCircle2 :size="17" /> Agent 离线自动降级只读</li>
          <li><CheckCircle2 :size="17" /> 每项操作均有任务与审计记录</li>
        </ul>
      </div>
      <p class="auth-layout__footnote">Kejilion Panel · Linux 管理面板</p>
    </section>
    <section class="auth-layout__form">
      <div class="auth-card">
        <slot />
      </div>
    </section>
  </main>
</template>
