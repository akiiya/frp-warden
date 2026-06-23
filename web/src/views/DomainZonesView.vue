<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NModal, NForm, NFormItem, NInput, NTag, NPopconfirm, NText, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { listDomainZones, createDomainZone, disableDomainZone, enableDomainZone } from '../api/domainZones'
import type { DomainZone } from '../types'

const message = useMessage()
const loading = ref(false)
const zones = ref<DomainZone[]>([])
const showCreate = ref(false)
const createModel = ref({ name: '', zone: '' })

const columns: DataTableColumns<DomainZone> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name', width: 150 },
  { title: 'Zone', key: 'zone', width: 250 },
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
          default: () => `确定${row.status === 'enabled' ? '禁用' : '启用'}该域名区域？`,
        }),
      ],
    }),
  },
]

async function fetchZones() {
  loading.value = true
  const res = await listDomainZones()
  if (res.ok) zones.value = (res.data || []) as DomainZone[]
  loading.value = false
}

async function handleCreate() {
  if (!createModel.value.name || !createModel.value.zone) {
    message.warning('name 和 zone 不能为空')
    return
  }
  const res = await createDomainZone(createModel.value.name, createModel.value.zone)
  if (res.ok) {
    showCreate.value = false
    createModel.value = { name: '', zone: '' }
    message.success('域名区域创建成功')
    fetchZones()
  } else {
    message.error(res.error?.message || '创建失败')
  }
}

async function handleToggle(row: DomainZone) {
  const fn = row.status === 'enabled' ? disableDomainZone : enableDomainZone
  const res = await fn(row.id)
  if (res.ok) {
    message.success(row.status === 'enabled' ? '已禁用' : '已启用')
    fetchZones()
  } else {
    message.error(res.error?.message || '操作失败')
  }
}

onMounted(fetchZones)
</script>

<template>
  <n-space vertical :size="16">
    <n-card title="域名区域管理">
      <template #header-extra>
        <n-button type="primary" @click="showCreate = true">新建域名区域</n-button>
      </template>
      <n-text depth="3" style="display: block; margin-bottom: 12px; font-size: 13px">
        域名区域用于分配 subdomain 资源。只有启用的域名区域才能创建 subdomain 资源。
      </n-text>
      <n-data-table :columns="columns" :data="zones" :loading="loading" :bordered="false" :single-line="false" />
    </n-card>

    <n-modal v-model:show="showCreate" preset="dialog" title="新建域名区域" positive-text="创建" negative-text="取消"
      @positive-click="handleCreate">
      <n-form :model="createModel" label-placement="left" label-width="80">
        <n-form-item label="名称" required>
          <n-input v-model:value="createModel.name" placeholder="如 默认泛域名" />
        </n-form-item>
        <n-form-item label="Zone" required>
          <n-input v-model:value="createModel.zone" placeholder="如 *.frp.example.com" />
        </n-form-item>
      </n-form>
    </n-modal>
  </n-space>
</template>
