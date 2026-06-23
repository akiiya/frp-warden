<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NModal, NForm, NFormItem, NInput, NAlert, NTag, NPopconfirm, NTabs, NTabPane, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { listTenants, createTenant, disableTenant, enableTenant, resetTenantToken } from '../api/tenants'
import { get } from '../api/client'
import type { Tenant } from '../types'

const message = useMessage()
const loading = ref(false)
const tenants = ref<Tenant[]>([])
const showCreate = ref(false)
const createLoading = ref(false)
const createModel = ref({ code: '', name: '', description: '' })

// Token / frpc 配置弹窗。
const showModal = ref(false)
const modalTitle = ref('')
const modalToken = ref('')
const modalFrpcConfig = ref('')
const modalIsOneTime = ref(false) // true=创建/重置时的一次性展示

// 查看 frpc 配置弹窗(模板模式)。
const showFrpcTemplate = ref(false)
const frpcTemplateConfig = ref('')
const frpcTemplateTenantCode = ref('')

const columns: DataTableColumns<Tenant> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: 'Code', key: 'code', width: 120 },
  { title: '名称', key: 'name', width: 150 },
  {
    title: '状态', key: 'status', width: 80,
    render: (row) => row.status === 'enabled'
      ? h(NTag, { type: 'success', size: 'small' }, { default: () => '启用' })
      : h(NTag, { type: 'error', size: 'small' }, { default: () => '禁用' }),
  },
  { title: '描述', key: 'description', ellipsis: { tooltip: true } },
  { title: '最后在线', key: 'last_seen_at', width: 160, render: (row) => row.last_seen_at || '—' },
  {
    title: '操作', key: 'actions', width: 320,
    render: (row) => h(NSpace, { size: 4 }, {
      default: () => [
        h(NButton, { size: 'small', quaternary: true, onClick: () => handleViewFrpcConfig(row) },
          { default: () => 'frpc 配置' }),
        h(NPopconfirm, { onPositiveClick: () => handleToggle(row) }, {
          trigger: () => h(NButton, { size: 'small', type: row.status === 'enabled' ? 'warning' : 'success', quaternary: true },
            { default: () => row.status === 'enabled' ? '禁用' : '启用' }),
          default: () => `确定${row.status === 'enabled' ? '禁用' : '启用'}租户 ${row.code}？`,
        }),
        h(NPopconfirm, { onPositiveClick: () => handleResetToken(row) }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true }, { default: () => '重置 Token' }),
          default: () => `确定重置租户 ${row.code} 的 token？旧 token 将立即失效。`,
        }),
      ],
    }),
  },
]

async function fetchTenants() {
  loading.value = true
  const res = await listTenants()
  if (res.ok) tenants.value = (res.data || []) as Tenant[]
  loading.value = false
}

async function handleCreate() {
  if (!createModel.value.code || !createModel.value.name) {
    message.warning('code 和 name 不能为空')
    return
  }
  createLoading.value = true
  const res = await createTenant(createModel.value.code, createModel.value.name, createModel.value.description)
  createLoading.value = false
  if (res.ok && res.data) {
    showCreate.value = false
    createModel.value = { code: '', name: '', description: '' }
    message.success('租户创建成功')

    // 显示一次性 token + 完整 frpc 配置。
    const data = res.data as any
    modalTitle.value = '租户创建成功 — 请复制保存'
    modalToken.value = data.plain_token
    modalFrpcConfig.value = data.frpc_config || ''
    modalIsOneTime.value = true
    showModal.value = true
    fetchTenants()
  } else {
    message.error(res.error?.message || '创建失败')
  }
}

async function handleToggle(row: Tenant) {
  const fn = row.status === 'enabled' ? disableTenant : enableTenant
  const res = await fn(row.id)
  if (res.ok) {
    message.success(row.status === 'enabled' ? '已禁用' : '已启用')
    fetchTenants()
  } else {
    message.error(res.error?.message || '操作失败')
  }
}

async function handleResetToken(row: Tenant) {
  const res = await resetTenantToken(row.id)
  if (res.ok && res.data) {
    message.success('Token 已重置')
    const data = res.data as any
    modalTitle.value = `租户 ${row.code} Token 已重置 — 请复制保存`
    modalToken.value = data.plain_token
    modalFrpcConfig.value = data.frpc_config || ''
    modalIsOneTime.value = true
    showModal.value = true
  } else {
    message.error(res.error?.message || '重置失败')
  }
}

async function handleViewFrpcConfig(row: Tenant) {
  const res = await get<any>(`/tenants/${row.id}/frpc-config`)
  if (res.ok && res.data) {
    frpcTemplateConfig.value = res.data.config
    frpcTemplateTenantCode.value = row.code
    showFrpcTemplate.value = true
  } else {
    message.error(res.error?.message || '获取配置失败')
  }
}

onMounted(fetchTenants)

async function copyToClipboard(text: string, label: string) {
  try {
    await navigator.clipboard.writeText(text)
    message.success(`${label} 已复制到剪贴板`)
  } catch {
    message.error('复制失败，请手动选择复制')
  }
}
</script>

<template>
  <n-space vertical :size="16">
    <n-card title="租户管理">
      <template #header-extra>
        <n-button type="primary" @click="showCreate = true">新建租户</n-button>
      </template>
      <n-data-table :columns="columns" :data="tenants" :loading="loading" :bordered="false" :single-line="false" />
    </n-card>

    <!-- 新建租户弹窗 -->
    <n-modal v-model:show="showCreate" preset="dialog" title="新建租户" positive-text="创建" negative-text="取消"
      :loading="createLoading" @positive-click="handleCreate">
      <n-form :model="createModel" label-placement="left" label-width="80">
        <n-form-item label="Code" required>
          <n-input v-model:value="createModel.code" placeholder="如 ufi001（对应 frpc 的 user）" />
        </n-form-item>
        <n-form-item label="名称" required>
          <n-input v-model:value="createModel.name" placeholder="如 测试设备 001" />
        </n-form-item>
        <n-form-item label="描述">
          <n-input v-model:value="createModel.description" placeholder="可选" />
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- Token + frpc 配置弹窗(创建/重置时的一次性展示) -->
    <n-modal v-model:show="showModal" preset="dialog" :title="modalTitle" style="max-width: 680px">
      <n-alert v-if="modalIsOneTime" type="warning" :bordered="false" style="margin-bottom: 12px">
        以下 token 和完整配置仅显示一次，请立即复制保存。关闭后将无法再次查看真实 token。
        如忘记 token，请重置 token。
      </n-alert>

      <n-tabs type="line" default-value="config">
        <n-tab-pane name="token" tab="Token">
          <n-space vertical :size="8">
            <n-input :value="modalToken" readonly type="textarea" :rows="2" style="font-family: monospace" />
            <n-button size="small" @click="copyToClipboard(modalToken, 'Token')">复制 Token</n-button>
          </n-space>
        </n-tab-pane>
        <n-tab-pane name="config" tab="frpc.toml 完整配置">
          <n-space vertical :size="8">
            <n-input :value="modalFrpcConfig" readonly type="textarea" :rows="16" style="font-family: monospace; font-size: 13px" />
            <n-button size="small" @click="copyToClipboard(modalFrpcConfig, 'frpc 配置')">复制 frpc.toml</n-button>
          </n-space>
        </n-tab-pane>
      </n-tabs>
    </n-modal>

    <!-- 查看 frpc 配置模板弹窗(平时查看,token 占位符) -->
    <n-modal v-model:show="showFrpcTemplate" preset="dialog" :title="`frpc 配置 — ${frpcTemplateTenantCode}`" style="max-width: 680px">
      <n-alert type="info" :bordered="false" style="margin-bottom: 12px">
        当前配置使用 token 占位符。真实 token 只在创建租户或重置 token 时显示一次。
        如忘记 token，请重置 token。
      </n-alert>
      <n-space vertical :size="8">
        <n-input :value="frpcTemplateConfig" readonly type="textarea" :rows="16" style="font-family: monospace; font-size: 13px" />
        <n-button size="small" @click="copyToClipboard(frpcTemplateConfig, 'frpc 配置')">复制配置</n-button>
      </n-space>
    </n-modal>
  </n-space>
</template>
