<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Eye, EyeOff, LoaderCircle, LockKeyhole } from '@lucide/vue'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import { ApiError } from '@/lib/api'
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

const canSubmit = computed(
  () => form.username.trim().length > 0 && form.password.length > 0 && (!totpRequired.value || /^\d{6}$/.test(form.totpCode)),
)

async function submit(): Promise<void> {
  if (!canSubmit.value) return
  error.value = ''

  try {
    await session.login({
      username: form.username.trim(),
      password: form.password,
      totpCode: totpRequired.value ? form.totpCode : undefined,
    })
    const redirect = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/')
      ? route.query.redirect
      : '/overview'
    await router.replace(redirect)
  } catch (reason) {
    if (reason instanceof ApiError && reason.code === 'totp_required') {
      totpRequired.value = true
      error.value = '请输入身份验证器中的 6 位验证码。'
      return
    }
    error.value = reason instanceof ApiError ? reason.message : '登录失败，请稍后重试。'
  }
}
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

      <button class="button button--primary button--block" type="submit" :disabled="!canSubmit || session.state.loading">
        <LoaderCircle v-if="session.state.loading" class="spin" :size="17" />
        {{ session.state.loading ? '正在验证…' : '安全登录' }}
      </button>
    </form>

    <p class="auth-card__security">Session 仅保存在安全的 HttpOnly Cookie 中。</p>
  </AuthLayout>
</template>
