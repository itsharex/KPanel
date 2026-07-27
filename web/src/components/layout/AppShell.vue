<script setup lang="ts">
import type { Component } from 'vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Boxes,
  ClipboardList,
  Container,
  LayoutDashboard,
  LogOut,
  Menu,
  Moon,
  Settings,
  Store,
  Sun,
  X,
} from '@lucide/vue'
import AgentBanner from '@/components/layout/AgentBanner.vue'
import LogoMark from '@/components/common/LogoMark.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import { useSession } from '@/stores/session'
import { usePanelState } from '@/stores/panel'
import { useTheme } from '@/stores/theme'
import { useToast } from '@/stores/toast'
import { api } from '@/lib/api'

interface NavigationItem {
  label: string
  to: string
  icon: Component
}

const navigation: NavigationItem[] = [
  { label: '概览', to: '/overview', icon: LayoutDashboard },
  { label: '网站', to: '/sites', icon: Boxes },
  { label: '应用市场', to: '/apps', icon: Store },
  { label: 'Docker', to: '/docker', icon: Container },
  { label: '活动记录', to: '/activity', icon: ClipboardList },
  { label: '设置', to: '/settings', icon: Settings },
]

const route = useRoute()
const router = useRouter()
const session = useSession()
const panel = usePanelState()
const theme = useTheme()
const toast = useToast()
const menuOpen = ref(false)
const signingOut = ref(false)

const pageTitle = computed(() => String(route.meta.title || 'KPanel'))
const agentStatus = computed(() => {
  const agent = panel.state.agent
  if (!agent?.connected) return { status: 'offline', label: 'Agent 离线' }
  if (!agent.compatible) return { status: 'incompatible', label: '版本不兼容' }
  if (agent.readOnly) return { status: 'read_only', label: '写入依赖未就绪' }
  return { status: 'connected', label: 'Agent 在线' }
})
let agentTimer: number | undefined

function closeMenu(): void {
  menuOpen.value = false
}

function toggleTheme(): void {
  theme.setTheme(theme.resolved.value === 'dark' ? 'light' : 'dark')
}

async function signOut(): Promise<void> {
  signingOut.value = true
  try {
    await session.logout()
    await router.replace('/login')
  } catch {
    toast.danger('退出失败', '请刷新页面后重试。')
  } finally {
    signingOut.value = false
  }
}

async function refreshAgent(): Promise<void> {
  try {
    const status = await api.agent.health()
    panel.setAgent(status)
    session.state.agent = status
  } catch (error) {
    const previous = panel.state.agent
    panel.setAgent({
      connected: false,
      compatible: previous?.compatible ?? true,
      readOnly: true,
      version: previous?.version,
      protocolVersion: previous?.protocolVersion,
      lastSeenAt: previous?.lastSeenAt,
      reason: '无法连接到宿主机 Agent。',
    })
  }
}

onMounted(() => {
  void refreshAgent()
  agentTimer = window.setInterval(refreshAgent, 30_000)
})

onBeforeUnmount(() => {
  if (agentTimer) window.clearInterval(agentTimer)
})
</script>

<template>
  <div class="app-shell">
    <Transition name="fade">
      <button v-if="menuOpen" class="mobile-overlay" type="button" aria-label="关闭导航" @click="closeMenu" />
    </Transition>

    <aside class="sidebar" :class="{ 'sidebar--open': menuOpen }">
      <div class="sidebar__brand">
        <LogoMark />
        <button class="icon-button sidebar__close" type="button" aria-label="关闭导航" @click="closeMenu">
          <X :size="19" />
        </button>
      </div>

      <nav class="sidebar__nav" aria-label="主导航">
        <RouterLink
          v-for="item in navigation"
          :key="item.to"
          :to="item.to"
          class="sidebar__link"
          @click="closeMenu"
        >
          <component :is="item.icon" :size="18" :stroke-width="1.9" aria-hidden="true" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar__footer">
        <div class="sidebar__agent">
          <StatusBadge :status="agentStatus.status" :label="agentStatus.label" subtle />
          <small v-if="panel.state.agent?.version">v{{ panel.state.agent.version }}</small>
        </div>
        <button class="sidebar__user" type="button" :disabled="signingOut" @click="signOut">
          <span class="avatar">{{ session.state.user?.username?.slice(0, 1).toUpperCase() || 'A' }}</span>
          <span>
            <strong>{{ session.state.user?.displayName || session.state.user?.username || '管理员' }}</strong>
            <small>退出登录</small>
          </span>
          <LogOut :size="16" aria-hidden="true" />
        </button>
      </div>
    </aside>

    <div class="app-shell__main">
      <header class="topbar">
        <div class="topbar__title">
          <button class="icon-button topbar__menu" type="button" aria-label="打开导航" @click="menuOpen = true">
            <Menu :size="20" />
          </button>
          <div>
            <span>控制台</span>
            <strong>{{ pageTitle }}</strong>
          </div>
        </div>
        <div class="topbar__actions">
          <StatusBadge :status="agentStatus.status" :label="agentStatus.label" subtle />
          <button class="icon-button" type="button" aria-label="切换浅色或深色主题" @click="toggleTheme">
            <Sun v-if="theme.resolved.value === 'dark'" :size="18" />
            <Moon v-else :size="18" />
          </button>
        </div>
      </header>

      <AgentBanner />

      <main class="page-content">
        <RouterView />
      </main>
    </div>
  </div>
</template>
