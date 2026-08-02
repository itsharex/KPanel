<script setup lang="ts">
import { CheckCircle2, ShieldCheck } from '@lucide/vue'
import LanguageSelector from '@/components/common/LanguageSelector.vue'
import LogoMark from '@/components/common/LogoMark.vue'
import { useI18n } from '@/i18n'

const i18n = useI18n()
</script>

<template>
  <main class="auth-layout">
    <LanguageSelector class="auth-layout__language" />
    <section class="auth-layout__brand">
      <LogoMark />
      <div class="auth-layout__message">
        <span class="eyebrow"><ShieldCheck :size="16" /> {{ i18n.t('auth.ecosystem') }}</span>
        <h1>{{ i18n.t('auth.heroTitle') }}</h1>
        <p>{{ i18n.t('auth.heroDescription') }}</p>
        <ul>
          <li><CheckCircle2 :size="17" /> {{ i18n.t('auth.featureInterop') }}</li>
          <li><CheckCircle2 :size="17" /> {{ i18n.t('auth.featureOffline') }}</li>
          <li><CheckCircle2 :size="17" /> {{ i18n.t('auth.featureAudit') }}</li>
        </ul>
      </div>
      <p class="auth-layout__footnote">{{ i18n.t('auth.product') }}</p>
    </section>
    <section class="auth-layout__form">
      <div class="auth-card">
        <slot />
      </div>
    </section>
  </main>
</template>
