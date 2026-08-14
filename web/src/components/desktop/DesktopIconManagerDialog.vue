<script setup lang="ts">
import { Link2, Pencil, Plus, RotateCcw, Trash2 } from '@lucide/vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import { useI18n } from '@/i18n'
import type { DesktopEntry } from '@/lib/desktopEntries'
import type { DesktopShortcut } from '@/types/api'

defineProps<{
  open: boolean
  hiddenEntries: DesktopEntry[]
  shortcuts: DesktopShortcut[]
  busy?: boolean
}>()

const emit = defineEmits<{
  close: []
  add: []
  edit: [shortcut: DesktopShortcut]
  remove: [shortcut: DesktopShortcut]
  restore: [entry: DesktopEntry]
}>()

const i18n = useI18n()
</script>

<template>
  <ModalDialog
    :open="open"
    :title="i18n.t('desktop.iconManagerTitle')"
    size="small"
    @close="emit('close')"
  >
    <div class="desktop-icon-manager">
      <p class="desktop-icon-manager__hint">
        {{ i18n.t('desktop.iconManagerHint') }}
      </p>

      <section>
        <header>
          <div>
            <strong>{{ i18n.t('desktop.hiddenEntriesTitle') }}</strong>
            <small>{{ i18n.t('desktop.hiddenEntriesHint') }}</small>
          </div>
          <span>{{ hiddenEntries.length }}</span>
        </header>
        <div v-if="hiddenEntries.length" class="desktop-icon-manager__list">
          <article v-for="entry in hiddenEntries" :key="entry.key">
            <span class="desktop-icon-manager__glyph" aria-hidden="true">
              <img v-if="entry.iconURL" :src="entry.iconURL" alt="" />
              <span v-else>{{ entry.name.trim().slice(0, 1).toLocaleUpperCase() }}</span>
            </span>
            <div>
              <strong>{{ entry.name }}</strong>
              <small>{{ entry.kind === 'app'
                ? i18n.t('desktop.detailApp')
                : i18n.t('desktop.detailSite') }}</small>
            </div>
            <button
              class="button button--ghost"
              type="button"
              :disabled="busy"
              @click="emit('restore', entry)"
            >
              <RotateCcw :size="14" aria-hidden="true" />
              {{ i18n.t('desktop.restoreToDesktop') }}
            </button>
          </article>
        </div>
        <p v-else class="desktop-icon-manager__empty">{{ i18n.t('desktop.hiddenEntriesEmpty') }}</p>
      </section>

      <section>
        <header>
          <div>
            <strong>{{ i18n.t('desktop.customShortcutsTitle') }}</strong>
            <small>{{ i18n.t('desktop.customShortcutsHint') }}</small>
          </div>
          <button class="button button--ghost" type="button" :disabled="busy" @click="emit('add')">
            <Plus :size="14" aria-hidden="true" />
            {{ i18n.t('desktop.shortcutAdd') }}
          </button>
        </header>
        <div v-if="shortcuts.length" class="desktop-icon-manager__list">
          <article v-for="shortcut in shortcuts" :key="shortcut.id">
            <span class="desktop-icon-manager__glyph" aria-hidden="true">
              <img v-if="shortcut.iconURL" :src="shortcut.iconURL" alt="" />
              <Link2 v-else :size="20" />
            </span>
            <div>
              <strong>{{ shortcut.name }}</strong>
              <small>{{ shortcut.description || shortcut.url }}</small>
            </div>
            <span class="desktop-icon-manager__actions">
              <button
                class="button button--ghost button--icon"
                type="button"
                :title="i18n.t('desktop.shortcutEdit')"
                :aria-label="i18n.t('desktop.shortcutEdit')"
                :disabled="busy"
                @click="emit('edit', shortcut)"
              >
                <Pencil :size="14" aria-hidden="true" />
              </button>
              <button
                class="button button--ghost button--icon"
                type="button"
                :title="i18n.t('desktop.shortcutDelete')"
                :aria-label="i18n.t('desktop.shortcutDelete')"
                :disabled="busy"
                @click="emit('remove', shortcut)"
              >
                <Trash2 :size="14" aria-hidden="true" />
              </button>
            </span>
          </article>
        </div>
        <p v-else class="desktop-icon-manager__empty">{{ i18n.t('desktop.customShortcutsEmpty') }}</p>
      </section>
    </div>
    <template #footer>
      <button class="button button--primary" type="button" @click="emit('close')">
        {{ i18n.t('common.closeDialog') }}
      </button>
    </template>
  </ModalDialog>
</template>
