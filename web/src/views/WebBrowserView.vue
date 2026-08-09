<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ExternalLink, Globe2, RefreshCw, ShieldCheck, TriangleAlert } from '@lucide/vue'
import { useRoute } from 'vue-router'
import { resolveEmbeddedBrowserTarget } from '@/lib/embeddedBrowser'
import { useI18n } from '@/i18n'
import { usePhraseCatalog } from '@/i18n/phrase'

const route = useRoute()
const i18n = useI18n()
usePhraseCatalog(() => import('@/i18n/pages/WebBrowserView/en-US').then((module) => module.default))
const frameVersion = ref(0)
const target = computed(() => resolveEmbeddedBrowserTarget(route.query.url))

watch(
  () => route.query.url,
  () => { frameVersion.value += 1 },
)

function reload(): void {
  if (target.value && !target.value.mixedContent) frameVersion.value += 1
}

function openExternal(): void {
  if (!target.value) return
  window.open(target.value.href, '_blank', 'noopener,noreferrer')
}
</script>

<template>
  <section class="embedded-browser">
    <header class="embedded-browser__toolbar">
      <span class="embedded-browser__identity" aria-hidden="true"><Globe2 :size="16" /></span>
      <div class="embedded-browser__address" :title="target?.href">
        <ShieldCheck :size="14" aria-hidden="true" />
        <span>{{ target?.href || i18n.t('desktop.browserInvalidURL') }}</span>
      </div>
      <button
        class="embedded-browser__tool"
        type="button"
        :disabled="!target || target.mixedContent"
        :title="i18n.t('desktop.browserReload')"
        :aria-label="i18n.t('desktop.browserReload')"
        @click="reload"
      >
        <RefreshCw :size="16" aria-hidden="true" />
      </button>
      <button
        class="button button--primary button--small embedded-browser__external"
        type="button"
        :disabled="!target"
        @click="openExternal"
      >
        <ExternalLink :size="14" aria-hidden="true" />
        <span>{{ i18n.t('desktop.browserOpenExternal') }}</span>
      </button>
    </header>

    <div v-if="!target" class="embedded-browser__state" role="alert">
      <span><TriangleAlert :size="24" aria-hidden="true" /></span>
      <strong>{{ i18n.t('desktop.browserInvalidURL') }}</strong>
    </div>
    <div v-else-if="target.mixedContent" class="embedded-browser__state" role="alert">
      <span><ShieldCheck :size="24" aria-hidden="true" /></span>
      <strong>{{ i18n.t('desktop.browserMixedContentTitle') }}</strong>
      <p>{{ i18n.t('desktop.browserMixedContentMessage') }}</p>
      <button class="button button--primary" type="button" @click="openExternal">
        <ExternalLink :size="15" aria-hidden="true" />
        {{ i18n.t('desktop.browserOpenExternal') }}
      </button>
    </div>
    <template v-else>
      <iframe
        :key="`${target.href}:${frameVersion}`"
        class="embedded-browser__frame"
        :src="target.href"
        :title="target.hostname"
        sandbox="allow-downloads allow-forms allow-modals allow-popups allow-popups-to-escape-sandbox allow-same-origin allow-scripts"
        referrerpolicy="no-referrer"
        allow="fullscreen"
      />
      <footer class="embedded-browser__hint">
        <ShieldCheck :size="13" aria-hidden="true" />
        <span>{{ i18n.t('desktop.browserEmbedHint') }}</span>
      </footer>
    </template>
  </section>
</template>

<style scoped>
.embedded-browser {
  display: grid;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  grid-template-rows: auto minmax(0, 1fr) auto;
  color: var(--text);
  background: var(--surface-subtle);
}

.embedded-browser__toolbar {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
  background: color-mix(in srgb, var(--surface-raised) 94%, var(--bg));
}

.embedded-browser__identity,
.embedded-browser__tool {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  place-items: center;
  padding: 0;
  color: var(--text-soft);
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: 9px;
}

.embedded-browser__identity {
  color: var(--brand);
  background: var(--brand-soft);
  border-color: color-mix(in srgb, var(--brand) 22%, var(--border));
}

.embedded-browser__tool:not(:disabled) { cursor: pointer; }
.embedded-browser__tool:not(:disabled):hover { color: var(--text); background: var(--surface); }
.embedded-browser__tool:disabled { opacity: .45; }

.embedded-browser__address {
  display: flex;
  min-width: 0;
  height: 32px;
  flex: 1 1 auto;
  align-items: center;
  gap: 7px;
  padding: 0 11px;
  color: var(--text-soft);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
  font-size: 11px;
}

.embedded-browser__address svg { flex: 0 0 auto; color: var(--success); }
.embedded-browser__address span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.embedded-browser__external { flex: 0 0 auto; white-space: nowrap; }

.embedded-browser__frame {
  width: 100%;
  height: 100%;
  min-height: 0;
  background: #fff;
  border: 0;
}

.embedded-browser__hint {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
  padding: 5px 11px;
  overflow: hidden;
  color: var(--muted);
  background: var(--surface);
  border-top: 1px solid var(--border);
  font-size: 10px;
  white-space: nowrap;
}

.embedded-browser__hint svg { flex: 0 0 auto; color: var(--success); }
.embedded-browser__hint span { overflow: hidden; text-overflow: ellipsis; }

.embedded-browser__state {
  display: flex;
  min-height: 260px;
  align-items: center;
  align-self: stretch;
  justify-content: center;
  flex-direction: column;
  gap: 10px;
  padding: 28px;
  color: var(--muted);
  text-align: center;
}

.embedded-browser__state > span {
  display: grid;
  width: 48px;
  height: 48px;
  place-items: center;
  color: var(--warning);
  background: var(--warning-soft);
  border-radius: 14px;
}

.embedded-browser__state strong { color: var(--text); font-size: 15px; }
.embedded-browser__state p { max-width: 480px; margin: 0; font-size: 12px; line-height: 1.7; }

@container desktop-window (max-width: 580px) {
  .embedded-browser__external span { display: none; }
  .embedded-browser__external { width: 34px; padding: 0; }
  .embedded-browser__identity { display: none; }
}
</style>
