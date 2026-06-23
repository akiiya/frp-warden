<script setup lang="ts">
import { ref, h, onMounted, watch } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NModal, NForm, NFormItem, NSelect, NTag, NPopconfirm, NText, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { listGrants, createGrant, disableGrant, enableGrant } from '../api/grants'
import { listTenants } from '../api/tenants'
import { listResources } from '../api/resources'
import type { ResourceGrant, Tenant, Resource } from '../types'

const message = useMessage()
const loading = ref(false)
const grants = ref<ResourceGrant[]>([])
const tenants = ref<Tenant[]>([])
const resources = ref<Resource[]>([])
const filterTenantId = ref<number | null>(null)
const showCreate = ref(false)
const createModel = ref({ tenant_id: null as number | null, resource_id: null as number | null })

const columns: DataTableColumns<ResourceGrant> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: 'Tenant ID', key: 'tenant_id', width: 100 },
  { title: 'Resource ID', key: 'resource_id', width: 100 },
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
          default: () => `确定${row.status === 'enabled' ? '禁用' : '启用'}该授权？`,
        }),
      ],
    }),
  },
]

async function fetchData() {
  loading.value = true
  const [grantRes, tenantRes, resRes] = await Promise.all([
    listGrants(filterTenantId.value || undefined),
    listTenants(),
    listResources(),
  ])
  if (grantRes.ok) grants.value = (grantRes.data || []) as ResourceGrant[]
  if (tenantRes.ok) tenants.value = (tenantRes.data || []) as Tenant[]
  if (resRes.ok) resources.value = (resRes.data || []) as Resource[]
  loading.value = false
}

const tenantOptions = () => tenants.value.map(t => ({ label: `${t.code}（${t.name}）`, value: t.id }))
const resourceOptions = () => resources.value
  .filter(r => r.status === 'available')
  .map(r => ({ label: `[${r.type}] ${r.value}`, value: r.id }))

async function handleCreate() {
  if (!createModel.value.tenant_id || !createModel.value.resource_id) {
    message.warning('请选择租户和资源')
    return
  }
  const res = await createGrant(createModel.value.tenant_id, createModel.value.resource_id)
  if (res.ok) {
    showCreate.value = false
    createModel.value = { tenant_id: null, resource_id: null }
    message.success('授权创建成功')
    fetchData()
  } else {
    message.error(res.error?.message || '创建失败')
  }
}

async function handleToggle(row: ResourceGrant) {
  const fn = row.status === 'enabled' ? disableGrant : enableGrant
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
    <n-card title="授权管理">
      <template #header-extra>
        <n-space :size="8">
          <n-select v-model:value="filterTenantId" :options="tenantOptions()" placeholder="按租户筛选" clearable style="width: 200px" />
          <n-button type="primary" @click="showCreate = true">新建授权</n-button>
        </n-space>
      </template>
      <n-text depth="3" style="display: block; margin-bottom: 12px; font-size: 13px">
        一个资源只能授权给一个租户。授权后，该资源不能再次授权给其他租户。
      </n-text>
      <n-data-table :columns="columns" :data="grants" :loading="loading" :bordered="false" :single-line="false" />
    </n-card>

    <n-modal v-model:show="showCreate" preset="dialog" title="新建授权" positive-text="创建" negative-text="取消"
      @positive-click="handleCreate">
      <n-form :model="createModel" label-placement="left" label-width="80">
        <n-form-item label="租户" required>
          <n-select v-model:value="createModel.tenant_id" :options="tenantOptions()" placeholder="选择租户" />
        </n-form-item>
        <n-form-item label="资源" required>
          <n-select v-model:value="createModel.resource_id" :options="resourceOptions()" placeholder="选择可用资源" />
        </n-form-item>
      </n-form>
    </n-modal>
  </n-space>
</template>
