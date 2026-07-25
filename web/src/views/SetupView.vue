<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Eye, EyeOff, KeyRound, LoaderCircle, ShieldCheck } from '@lucide/vue'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import { ApiError } from '@/lib/api'
import { useSession } from '@/stores/session'

const router = useRouter()
const session = useSession()
const form = reactive({
  token: '',
  username: 'admin',
  password: '',
  confirmPassword: '',
})
const showPassword = ref(false)
const error = ref('')
const submitted = ref(false)

const passwordChecks = computed(() => [
  { label: '至少 12 个字符', valid: form.password.length >= 12 },
  { label: '包含字母和数字', valid: /[A-Za-z]/.test(form.password) && /\d/.test(form.password) },
])

const canSubmit = computed(
  () =>
    form.token.trim().length > 0 &&
    /^[A-Za-z0-9._-]{3,32}$/.test(form.username) &&
    passwordChecks.value.every((item) => item.valid) &&
    form.password === form.confirmPassword,
)

async function submit(): Promise<void> {
  submitted.value = true
  error.value = ''
  if (!canSubmit.value) return

  try {
    await session.setup({
      token: form.token.trim(),
      username: form.username.trim(),
      password: form.password,
    })
    await router.replace('/overview')
  } catch (reason) {
    error.value = reason instanceof ApiError ? reason.message : '初始化失败，请确认凭据后重试。'
  }
}
</script>

<template>
  <AuthLayout>
    <div class="auth-card__heading">
      <span class="auth-card__icon"><ShieldCheck :size="22" /></span>
      <div>
        <span class="eyebrow">首次使用</span>
        <h2>初始化管理账户</h2>
      </div>
    </div>
    <p class="auth-card__intro">
      输入安装完成时显示的一次性初始化凭据。提交成功后该凭据立即失效，不会修改现有
      <code>kejilion.sh</code>。
    </p>

    <div v-if="session.state.error" class="inline-alert inline-alert--danger" role="alert">
      {{ session.state.error }}
      <button type="button" @click="session.refresh(true)">重试连接</button>
    </div>
    <div v-if="error" class="inline-alert inline-alert--danger" role="alert">{{ error }}</div>

    <form class="form-stack" novalidate @submit.prevent="submit">
      <label class="field">
        <span>初始化凭据</span>
        <span class="input-wrap">
          <KeyRound :size="17" aria-hidden="true" />
          <input v-model.trim="form.token" autocomplete="one-time-code" placeholder="粘贴一次性凭据" required />
        </span>
        <small v-if="submitted && !form.token">请输入安装时生成的一次性凭据。</small>
      </label>

      <label class="field">
        <span>管理员用户名</span>
        <input v-model.trim="form.username" autocomplete="username" maxlength="32" required />
        <small v-if="submitted && !/^[A-Za-z0-9._-]{3,32}$/.test(form.username)">
          使用 3–32 位字母、数字、点、下划线或连字符。
        </small>
      </label>

      <label class="field">
        <span>管理员密码</span>
        <span class="input-wrap input-wrap--action">
          <input
            v-model="form.password"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="new-password"
            placeholder="设置一个强密码"
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

      <div class="password-checks" aria-label="密码要求">
        <span v-for="check in passwordChecks" :key="check.label" :class="{ 'is-valid': check.valid }">
          <i aria-hidden="true" /> {{ check.label }}
        </span>
      </div>

      <label class="field">
        <span>确认密码</span>
        <input v-model="form.confirmPassword" type="password" autocomplete="new-password" required />
        <small v-if="submitted && form.password !== form.confirmPassword">两次输入的密码不一致。</small>
      </label>

      <button class="button button--primary button--block" type="submit" :disabled="session.state.loading">
        <LoaderCircle v-if="session.state.loading" class="spin" :size="17" />
        {{ session.state.loading ? '正在初始化…' : '完成初始化' }}
      </button>
    </form>
  </AuthLayout>
</template>
