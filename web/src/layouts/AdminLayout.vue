<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NLayout, NLayoutSider, NLayoutHeader, NLayoutContent, NMenu, NButton, NSpace, NText } from 'naive-ui'
import { useAuthStore } from '../stores/auth'
import { logout } from '../api/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const collapsed = ref(false)

const menuOptions = [
  { label: '仪表盘', key: 'dashboard', icon: () => '📊' },
  { label: '租户管理', key: 'tenants', icon: () => '👤' },
  { label: '域名区域', key: 'domain-zones', icon: () => '🌐' },
  { label: '资源池', key: 'resources', icon: () => '📦' },
  { label: '授权管理', key: 'grants', icon: () => '🔑' },
  { label: '映射管理', key: 'proxies', icon: () => '🔗' },
  { label: '审计日志', key: 'audit-logs', icon: () => '📋' },
  { label: '账号设置', key: 'account', icon: () => '⚙️' },
]

const activeKey = computed(() => route.name as string)

function handleMenuUpdate(key: string) {
  router.push({ name: key })
}

async function handleLogout() {
  await logout()
  auth.clear()
  router.push('/login')
}
</script>

<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="220"
      :collapsed="collapsed"
      show-trigger
      @collapse="collapsed = true"
      @expand="collapsed = false"
      :native-scrollbar="false"
      style="background: #fff"
    >
      <div style="padding: 16px; text-align: center; border-bottom: 1px solid #efeff5">
        <n-text strong style="font-size: 18px" v-if="!collapsed">frp-warden</n-text>
        <n-text strong style="font-size: 18px" v-else>fw</n-text>
      </div>
      <n-menu
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="20"
        :options="menuOptions"
        :value="activeKey"
        @update:value="handleMenuUpdate"
      />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered style="height: 56px; display: flex; align-items: center; justify-content: space-between; padding: 0 24px">
        <n-text depth="3" style="font-size: 13px">frp 多租户授权控制面板</n-text>
        <n-space align="center" :size="12">
          <n-text>{{ auth.admin?.username }}</n-text>
          <n-button size="small" quaternary @click="handleLogout">退出</n-button>
        </n-space>
      </n-layout-header>
      <n-layout-content content-style="padding: 24px;" :native-scrollbar="false" style="background: #f5f5f5">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>
