<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, NH2, NText, NAlert } from 'naive-ui'
import type { FormInst, FormRules } from 'naive-ui'
import { login } from '../api/auth'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const error = ref('')

const model = ref({ username: '', password: '' })

const rules: FormRules = {
  username: { required: true, message: '请输入用户名', trigger: 'blur' },
  password: { required: true, message: '请输入密码', trigger: 'blur' },
}

async function handleLogin() {
  error.value = ''
  try {
    await formRef.value?.validate()
  } catch { return }

  loading.value = true
  const res = await login(model.value.username, model.value.password)
  loading.value = false

  if (res.ok && res.data) {
    auth.setAdmin(res.data.admin)
    router.push('/dashboard')
  } else {
    error.value = res.error?.message || '登录失败'
  }
}
</script>

<template>
  <div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; background: #f5f5f5">
    <n-card style="width: 400px">
      <n-space vertical align="center" :size="4">
        <n-h2 style="margin: 0">frp-warden</n-h2>
        <n-text depth="3">frp 多租户授权控制面板</n-text>
        <n-text depth="3" style="font-size: 12px; margin-bottom: 12px">Multi-tenant authorization control plane for frp.</n-text>
      </n-space>
      <n-alert v-if="error" type="error" :bordered="false" style="margin-bottom: 16px">{{ error }}</n-alert>
      <n-form ref="formRef" :model="model" :rules="rules" label-placement="left" label-width="0">
        <n-form-item path="username">
          <n-input v-model:value="model.username" placeholder="用户名" @keyup.enter="handleLogin" />
        </n-form-item>
        <n-form-item path="password">
          <n-input v-model:value="model.password" type="password" placeholder="密码" show-password-on="click" @keyup.enter="handleLogin" />
        </n-form-item>
        <n-button type="primary" block :loading="loading" @click="handleLogin">登录</n-button>
      </n-form>
    </n-card>
  </div>
</template>
