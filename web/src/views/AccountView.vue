<script setup lang="ts">
import { ref, computed } from 'vue'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, NAlert, NText, useMessage } from 'naive-ui'
import { useAuthStore } from '../stores/auth'
import { changePassword } from '../api/auth'

const message = useMessage()
const auth = useAuthStore()
const loading = ref(false)
const model = ref({ old_password: '', new_password: '', confirm_password: '' })

const mustChange = computed(() => auth.admin?.must_change_password)

async function handleChangePassword() {
  if (!model.value.old_password || !model.value.new_password) {
    message.warning('请填写旧密码和新密码')
    return
  }
  if (model.value.new_password.length < 10) {
    message.warning('新密码长度至少 10 位')
    return
  }
  if (model.value.new_password !== model.value.confirm_password) {
    message.warning('两次输入的新密码不一致')
    return
  }

  loading.value = true
  const res = await changePassword(model.value.old_password, model.value.new_password)
  loading.value = false

  if (res.ok) {
    message.success('密码修改成功')
    model.value = { old_password: '', new_password: '', confirm_password: '' }
    // 刷新管理员信息。
    auth.fetchMe()
  } else {
    message.error(res.error?.message || '修改失败')
  }
}
</script>

<template>
  <n-space vertical :size="16">
    <n-card title="账号设置">
      <n-space vertical :size="12">
        <n-text>用户名：<strong>{{ auth.admin?.username }}</strong></n-text>
        <n-alert v-if="mustChange" type="warning" :bordered="false">
          您的密码为初始密码，建议立即修改。
        </n-alert>
      </n-space>
    </n-card>

    <n-card title="修改密码">
      <n-form :model="model" label-placement="left" label-width="120" style="max-width: 400px">
        <n-form-item label="旧密码" required>
          <n-input v-model:value="model.old_password" type="password" show-password-on="click" />
        </n-form-item>
        <n-form-item label="新密码" required>
          <n-input v-model:value="model.new_password" type="password" show-password-on="click" placeholder="至少 10 位" />
        </n-form-item>
        <n-form-item label="确认新密码" required>
          <n-input v-model:value="model.confirm_password" type="password" show-password-on="click" />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" :loading="loading" @click="handleChangePassword">修改密码</n-button>
        </n-form-item>
      </n-form>
    </n-card>
  </n-space>
</template>
