<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NGrid, NGi, NSpace, NH3, NText, NAlert, NButton } from 'naive-ui'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const mustChange = computed(() => auth.admin?.must_change_password)
</script>

<template>
  <n-space vertical :size="16">
    <n-alert v-if="mustChange" type="warning" :bordered="false">
      您的密码为初始密码，建议立即修改。
      <n-button text type="warning" @click="router.push('/account')">前往修改</n-button>
    </n-alert>

    <n-card>
      <n-space vertical :size="8">
        <n-h3 style="margin: 0">欢迎使用 frp-warden</n-h3>
        <n-text depth="3">当前管理员：{{ auth.admin?.username }}</n-text>
      </n-space>
    </n-card>

    <n-grid :cols="2" :x-gap="16" :y-gap="16">
      <n-gi>
        <n-card title="部署模型">
          <n-space vertical :size="8">
            <n-text>• frps 与 frp-warden 部署在<strong>公网服务器 / VPS</strong>（同机或同内网）。</n-text>
            <n-text>• frp-warden 的 plugin 接口默认仅监听 <code>127.0.0.1</code>，只允许本机 frps 调用。</n-text>
            <n-text>• <strong>随身 WiFi / Debian 设备只运行 frpc</strong>，不运行 frps、也不运行 frp-warden。</n-text>
            <n-text depth="3" style="font-size: 12px">运维在管理页面创建租户、授权资源、生成 frpc 配置后，复制到设备启动即可。</n-text>
          </n-space>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="快捷操作">
          <n-space vertical :size="8">
            <n-button block @click="router.push('/tenants')">👤 创建租户</n-button>
            <n-button block @click="router.push('/resources')">📦 创建资源</n-button>
            <n-button block @click="router.push('/grants')">🔑 创建授权</n-button>
            <n-button block @click="router.push('/audit-logs')">📋 查看审计日志</n-button>
          </n-space>
        </n-card>
      </n-gi>
    </n-grid>

    <n-card title="工作流程">
      <n-space vertical :size="8">
        <n-text>1. 在<strong>域名区域</strong>中添加泛域名（如 <code>*.frp.example.com</code>）。</n-text>
        <n-text>2. 在<strong>资源池</strong>中创建 subdomain 或端口资源。</n-text>
        <n-text>3. 在<strong>租户管理</strong>中创建租户（会生成独立 token）。</n-text>
        <n-text>4. 在<strong>授权管理</strong>中将资源授权给租户。</n-text>
        <n-text>5. 在<strong>映射管理</strong>中为租户创建 proxy（绑定已授权资源）。</n-text>
        <n-text>6. 将 frpc 配置复制到设备上启动。</n-text>
      </n-space>
    </n-card>
  </n-space>
</template>
