<script setup lang="ts">
import { ref, h, onMounted, watch } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NModal, NForm, NFormItem, NInput, NSelect, NInputNumber, NTag, NPopconfirm, NText, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { listProxies, createProxy, disableProxy, enableProxy } from '../api/proxies'
import { listTenants } from '../api/tenants'
import { listResources } from '../api/resources'
import type { Proxy, Tenant, Resource } from '../types'

const message = useMessage()
const loading = ref(false)
const proxies = ref<Proxy[]>([])
const tenants = ref<Tenant[]>([])
const resources = ref<Resource[]>([])
const filterTenantId = ref<number | null>(null)
const showCreate = ref(false)
const createModel = ref({
  tenant_id: null as number | null,
  resource_id: null as number | null,
  name: '',
  proxy_type: 'http',
  local_ip: '127.0.0.1',
  local_port: 8080,
})

const typeOptions = [
  { label: 'http', value: 'http' },
  { label: 'https', value: 'https' },
  { label: 'tcp', value: 'tcp' },
  { label: 'udp', value: 'udp' },
]

const columns: DataTableColumns<Proxy> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name', width: 100 },
  { title: 'Tenant ID', key: 'tenant_id', width: 80 },
  {
    title: '类型', key: 'proxy_type', width: 80,
    render: (row) => h(NTag, { size: 'small' }, { default: () => row.proxy_type }),
  },
  { title: 'Resource ID', key: 'resource_id', width: 100 },
  { title: '本地地址', key: 'local_ip', width: 120 },
  { title: '本地端口', key: 'local_port', width: 80 },
  {
    title: '状态', key: 'status', width: 80,
    render: (row) => row.status === 'enabled'
      ? h(NTag, { type: 'success', size: 'small' }, { default: () => '启用' })
      : h(NTag, { type: 'error', size: 'small' }, { default: () => '禁用' }),
  },
  {
    title: '操作', key: 'actions', width: 150,
    render: (row) => h(NSpace, { size: 4 }, {
      default: () => [
        h(NPopconfirm, { onPositiveClick: () => handleToggle(row) }, {
          trigger: () => h(NButton, { size: 'small', type: row.status === 'enabled' ? 'warning' : 'success', quaternary: true },
            { default: () => row.status === 'enabled' ? '禁用' : '启用' }),
          default: () => `确定${row.status === 'enabled' ? '禁用' : '启用'}该 proxy？`,
        }),
      ],
    }),
  },
]

async function fetchData() {
  loading.value = true
  const [proxyRes, tenantRes, resRes] = await Promise.all([
    listProxies(filterTenantId.value || undefined),
    listTenants(),
    listResources(),
  ])
  if (proxyRes.ok) proxies.value = (proxyRes.data || []) as Proxy[]
  if (tenantRes.ok) tenants.value = (tenantRes.data || []) as Tenant[]
  if (resRes.ok) resources.value = (resRes.data || []) as Resource[]
  loading.value = false
}

const tenantOptions = () => tenants.value.map(t => ({ label: `${t.code}（${t.name}）`, value: t.id }))
const resourceOptions = () => resources.value
  .filter(r => r.status === 'available')
  .map(r => ({ label: `[${r.type}] ${r.value}`, value: r.id }))

async function handleCreate() {
  if (!createModel.value.tenant_id || !createModel.value.resource_id || !createModel.value.name) {
    message.warning('请填写完整信息')
    return
  }
  if (createModel.value.local_port < 1 || createModel.value.local_port > 65535) {
    message.warning('本地端口范围应在 1-65535')
    return
  }
  const res = await createProxy({
    tenant_id: createModel.value.tenant_id,
    resource_id: createModel.value.resource_id,
    name: createModel.value.name,
    proxy_type: createModel.value.proxy_type,
    local_ip: createModel.value.local_ip || '127.0.0.1',
    local_port: createModel.value.local_port,
  })
  if (res.ok) {
    showCreate.value = false
    createModel.value = { tenant_id: null, resource_id: null, name: '', proxy_type: 'http', local_ip: '127.0.0.1', local_port: 8080 }
    message.success('映射创建成功')
    fetchData()
  } else {
    message.error(res.error?.message || '创建失败')
  }
}

async function handleToggle(row: Proxy) {
  const fn = row.status === 'enabled' ? disableProxy : enableProxy
  const res = await fn(row.id)
  if (res.ok) {
    message.success(row.status === 'enabled' ? '已禁用' : '已启用')
    fetchData()
  } else {
    message.error(res.error?.message || '操作失败')
  }
}

watch(filterTenantId, () => fetchData())
onMounted(fetchData)
</script>

<template>
  <n-space vertical :size="16">
    <n-card title="映射管理">
      <template #header-extra>
        <n-space :size="8">
          <n-select v-model:value="filterTenantId" :options="tenantOptions()" placeholder="按租户筛选" clearable style="width: 200px" />
          <n-button type="primary" @click="showCreate = true">新建映射</n-button>
        </n-space>
      </template>
      <n-text depth="3" style="display: block; margin-bottom: 12px; font-size: 13px">
        http/https 必须使用 subdomain 资源；tcp 必须使用 tcp_port 资源；udp 必须使用 udp_port 资源。
      </n-text>
      <n-data-table :columns="columns" :data="proxies" :loading="loading" :bordered="false" :single-line="false" />
    </n-card>

    <n-modal v-model:show="showCreate" preset="dialog" title="新建映射" positive-text="创建" negative-text="取消"
      @positive-click="handleCreate" style="max-width: 520px">
      <n-form :model="createModel" label-placement="left" label-width="100">
        <n-form-item label="租户" required>
          <n-select v-model:value="createModel.tenant_id" :options="tenantOptions()" placeholder="选择租户" />
        </n-form-item>
        <n-form-item label="资源" required>
          <n-select v-model:value="createModel.resource_id" :options="resourceOptions()" placeholder="选择已授权的资源" />
        </n-form-item>
        <n-form-item label="名称" required>
          <n-input v-model:value="createModel.name" placeholder="如 web、ssh" />
        </n-form-item>
        <n-form-item label="类型" required>
          <n-select v-model:value="createModel.proxy_type" :options="typeOptions" />
        </n-form-item>
        <n-form-item label="本地 IP">
          <n-input v-model:value="createModel.local_ip" placeholder="默认 127.0.0.1" />
        </n-form-item>
        <n-form-item label="本地端口" required>
          <n-input-number v-model:value="createModel.local_port" :min="1" :max="65535" style="width: 100%" />
        </n-form-item>
      </n-form>
    </n-modal>
  </n-space>
</template>
