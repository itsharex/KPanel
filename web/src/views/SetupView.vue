<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Eye, EyeOff, KeyRound, LoaderCircle, ShieldCheck } from '@lucide/vue'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import { useI18n } from '@/i18n'
import { localizeError } from '@/i18n/errors'
import { useSession } from '@/stores/session'

const router = useRouter()
const session = useSession()
const i18n = useI18n()
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
  { label: i18n.t('auth.passwordLength'), valid: form.password.length >= 12 },
  { label: i18n.t('auth.passwordComposition'), valid: /[A-Za-z]/.test(form.password) && /\d/.test(form.password) },
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
    error.value = localizeError(reason, 'auth.setupFailed')
  }
}
</script>

<template>
  <AuthLayout>
    <div class="auth-card__heading">
      <span class="auth-card__icon"><ShieldCheck :size="22" /></span>
      <div>
        <span class="eyebrow">{{ i18n.t('auth.firstUse') }}</span>
        <h2>{{ i18n.t('auth.setupTitle') }}</h2>
      </div>
    </div>
    <p class="auth-card__intro">
      {{ i18n.t('auth.setupIntro') }}
    </p>

    <div v-if="session.state.error" class="inline-alert inline-alert--danger" role="alert">
      {{ localizeError(session.state.error, 'error.authenticationRequired') }}
      <button type="button" @click="session.refresh(true)">{{ i18n.t('common.retryConnection') }}</button>
    </div>
    <div v-if="error" class="inline-alert inline-alert--danger" role="alert">{{ error }}</div>

    <form class="form-stack" novalidate @submit.prevent="submit">
      <label class="field">
        <span>{{ i18n.t('auth.bootstrapToken') }}</span>
        <span class="input-wrap">
          <KeyRound :size="17" aria-hidden="true" />
          <input v-model.trim="form.token" autocomplete="one-time-code" :placeholder="i18n.t('auth.bootstrapPlaceholder')" required />
        </span>
        <small v-if="submitted && !form.token">{{ i18n.t('auth.bootstrapRequired') }}</small>
      </label>

      <label class="field">
        <span>{{ i18n.t('auth.adminUsername') }}</span>
        <input v-model.trim="form.username" autocomplete="username" maxlength="32" required />
        <small v-if="submitted && !/^[A-Za-z0-9._-]{3,32}$/.test(form.username)">
          {{ i18n.t('auth.usernameRule') }}
        </small>
      </label>

      <label class="field">
        <span>{{ i18n.t('auth.adminPassword') }}</span>
        <span class="input-wrap input-wrap--action">
          <input
            v-model="form.password"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="new-password"
            :placeholder="i18n.t('auth.strongPasswordPlaceholder')"
            required
          />
          <button
            class="input-action"
            type="button"
            :aria-label="i18n.t(showPassword ? 'auth.hidePassword' : 'auth.showPassword')"
            @click="showPassword = !showPassword"
          >
            <EyeOff v-if="showPassword" :size="17" />
            <Eye v-else :size="17" />
          </button>
        </span>
      </label>

      <div class="password-checks" :aria-label="i18n.t('auth.passwordRequirements')">
        <span v-for="check in passwordChecks" :key="check.label" :class="{ 'is-valid': check.valid }">
          <i aria-hidden="true" /> {{ check.label }}
        </span>
      </div>

      <label class="field">
        <span>{{ i18n.t('auth.confirmPassword') }}</span>
        <input v-model="form.confirmPassword" type="password" autocomplete="new-password" required />
        <small v-if="submitted && form.password !== form.confirmPassword">{{ i18n.t('auth.passwordMismatch') }}</small>
      </label>

      <button class="button button--primary button--block" type="submit" :disabled="session.state.loading">
        <LoaderCircle v-if="session.state.loading" class="spin" :size="17" />
        {{ i18n.t(session.state.loading ? 'auth.initializing' : 'auth.finishSetup') }}
      </button>
    </form>
  </AuthLayout>
</template>
