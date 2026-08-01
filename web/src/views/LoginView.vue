<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Eye, EyeOff, LoaderCircle, LockKeyhole } from '@lucide/vue'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import { ApiError } from '@/lib/api'
import { prefetchNavigationRoute } from '@/lib/navigation'
import { useSession } from '@/stores/session'

const route = useRoute()
const router = useRouter()
const session = useSession()
const form = reactive({
  username: '',
  password: '',
  totpCode: '',
})
const showPassword = ref(false)
const totpRequired = ref(false)
const error = ref('')
const loginPhase = ref<'idle' | 'authenticating' | 'entering'>('idle')

const destination = computed(() => (
  typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/')
    ? route.query.redirect
    : '/overview'
))
const destinationPath = computed(() => destination.value.split(/[?#]/, 1)[0] || '/overview')
const busy = computed(() => loginPhase.value !== 'idle' || session.state.loading)
const submitLabel = computed(() => {
  if (loginPhase.value === 'entering') return '登录成功，正在进入控制台…'
  if (loginPhase.value === 'authenticating' || session.state.loading) return '正在验证…'
  return '安全登录'
})

const canSubmit = computed(
  () => form.username.trim().length > 0 && form.password.length > 0 && (!totpRequired.value || /^\d{6}$/.test(form.totpCode)),
)

async function submit(): Promise<void> {
  if (!canSubmit.value || busy.value) return
  error.value = ''
  loginPhase.value = 'authenticating'

  try {
    await session.login({
      username: form.username.trim(),
      password: form.password,
      totpCode: totpRequired.value ? form.totpCode : undefined,
    })
    loginPhase.value = 'entering'
    await router.replace(destination.value)
  } catch (reason) {
    if (reason instanceof ApiError && reason.code === 'totp_required') {
      totpRequired.value = true
      error.value = '请输入身份验证器中的 6 位验证码。'
      return
    }
    error.value = session.state.authenticated
      ? '登录成功，但控制台资源加载失败，请刷新页面重试。'
      : reason instanceof ApiError ? reason.message : '登录失败，请稍后重试。'
  } finally {
    loginPhase.value = 'idle'
  }
}

onMounted(() => {
  // Warm only the destination view. This overlaps the small chunk download
  // with credential entry without loading protected data or the full console.
  void prefetchNavigationRoute(destinationPath.value)
})
</script>

<template>
  <AuthLayout>
    <div class="auth-card__heading">
      <span class="auth-card__icon"><LockKeyhole :size="21" /></span>
      <div>
        <span class="eyebrow">欢迎回来</span>
        <h2>登录 KPanel</h2>
      </div>
    </div>
    <p class="auth-card__intro">使用本机管理账户继续。连续失败会触发安全限速。</p>

    <div v-if="session.state.error" class="inline-alert inline-alert--danger" role="alert">
      {{ session.state.error }}
      <button type="button" @click="session.refresh(true)">重试连接</button>
    </div>
    <div v-if="error" class="inline-alert inline-alert--danger" role="alert">{{ error }}</div>

    <form class="form-stack" @submit.prevent="submit">
      <label class="field">
        <span>用户名</span>
        <input v-model.trim="form.username" autocomplete="username" autofocus required />
      </label>

      <label class="field">
        <span>密码</span>
        <span class="input-wrap input-wrap--action">
          <input
            v-model="form.password"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="current-password"
            required
          />
          <button
            class="input-action"
            type="button"
            :aria-label="showPassword ? '隐藏密码' : '显示密码'"
            @click="showPassword = !showPassword"
          >
            <EyeOff v-if="showPassword" :size="17" />
            <Eye v-else :size="17" />
          </button>
        </span>
      </label>

      <label v-if="totpRequired" class="field">
        <span>两步验证码</span>
        <input
          v-model.trim="form.totpCode"
          inputmode="numeric"
          autocomplete="one-time-code"
          maxlength="6"
          placeholder="000000"
          required
        />
      </label>

      <button class="button button--primary button--block" type="submit" :disabled="!canSubmit || busy">
        <LoaderCircle v-if="busy" class="spin" :size="17" />
        {{ submitLabel }}
      </button>
    </form>

    <p class="auth-card__security">Session 仅保存在安全的 HttpOnly Cookie 中。</p>

    <Transition name="fade">
      <div v-if="loginPhase === 'entering'" class="login-transition" role="status" aria-live="polite">
        <span class="login-transition__icon"><LoaderCircle class="spin" :size="24" /></span>
        <strong>登录成功</strong>
        <small>正在加载控制台…</small>
      </div>
    </Transition>
  </AuthLayout>
</template>
