<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NModal, NForm, NFormItem, NInput, NSelect, NTag, NText, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { listResources, createSubdomainResource, createTCPPortResource, createUDPPortResource } from '../api/resources'
import { listDomainZones } from '../api/domainZones'
import type { Resource, DomainZone } from '../types'

const message = useMessage()
const loading = ref(false)
const resources = ref<Resource[]>([])
const zones = ref<DomainZone[]>([])
const showCreate = ref(false)
const createModel = ref({ type: 'subdomain', value: '', domain_zone_id: null as number | null })

const typeOptions = [
  { label: 'subdomain（子域名）', value: 'subdomain' },
  { label: 'tcp_port（TCP 端口）', value: 'tcp_port' },
  { label: 'udp_port（UDP 端口）', value: 'udp_port' },
]

const columns: DataTableColumns<Resource> = [
  { title: 'ID', key: 'id', width: 60 },
  {
    title: '类型', key: 'type', width: 120,
    render: (row) => h(NTag, {
      type: row.type === 'subdomain' ? 'info' : row.type === 'tcp_port' ? 'success' : 'warning',
      size: 'small',
    }, { default: () => row.type }),
  },
  { title: 'Value', key: 'value', width: 200 },
  {
    title: '域名区域 ID', key: 'domain_zone_id', width: 120,
    render: (row) => (row as any).domain_zone_id || '—',
  },
  {
    title: '状态', key: 'status', width: 80,
    render: (row) => row.status === 'available'
      ? h(NTag, { type: 'success', size: 'small' }, { default: () => '可用' })
      : h(NTag, { type: 'error', size: 'small' }, { default: () => '禁用' }),
  },
]

async function fetchResources() {
  loading.value = true
  const [resRes, zoneRes] = await Promise.all([listResources(), listDomainZones()])
  if (resRes.ok) resources.value = (resRes.data || []) as Resource[]
  if (zoneRes.ok) zones.value = (zoneRes.data || []) as DomainZone[]
  loading.value = false
}

const zoneOptions = () => zones.value
  .filter(z => z.status === 'enabled')
  .map(z => ({ label: `${z.name}（${z.zone}）`, value: z.id }))

async function handleCreate() {
  if (!createModel.value.value) {
    message.warning('value 不能为空')
    return
  }

  let res
  if (createModel.value.type === 'subdomain') {
    if (!createModel.value.domain_zone_id) {
      message.warning('subdomain 资源必须选择域名区域')
      return
    }
    res = await createSubdomainResource(createModel.value.domain_zone_id, createModel.value.value)
  } else if (createModel.value.type === 'tcp_port') {
    const port = parseInt(createModel.value.value)
    if (isNaN(port) || port < 1 || port > 65535) {
      message.warning('端口范围应在 1-65535')
      return
    }
    res = await createTCPPortResource(createModel.value.value)
  } else {
    const port = parseInt(createModel.value.value)
    if (isNaN(port) || port < 1 || port > 65535) {
      message.warning('端口范围应在 1-65535')
      return
    }
    res = await createUDPPortResource(createModel.value.value)
  }

  if (res.ok) {
    showCreate.value = false
    createModel.value = { type: 'subdomain', value: '', domain_zone_id: null }
    message.success('资源创建成功')
    fetchResources()
  } else {
    message.error(res.error?.message || '创建失败')
  }
}

onMounted(fetchResources)
</script>

<template>
  <n-space vertical :size="16">
    <n-card title="资源池管理">
      <template #header-extra>
        <n-button type="primary" @click="showCreate = true">新建资源</n-button>
      </template>
      <n-text depth="3" style="display: block; margin-bottom: 12px; font-size: 13px">
        子域名资源需要先配置并启用顶级域名区域。端口资源直接创建即可。
      </n-text>
      <n-data-table :columns="columns" :data="resources" :loading="loading" :bordered="false" :single-line="false" />
    </n-card>

    <n-modal v-model:show="showCreate" preset="dialog" title="新建资源" positive-text="创建" negative-text="取消"
      @positive-click="handleCreate">
      <n-form :model="createModel" label-placement="left" label-width="100">
        <n-form-item label="资源类型" required>
          <n-select v-model:value="createModel.type" :options="typeOptions" />
        </n-form-item>
        <n-form-item v-if="createModel.type === 'subdomain'" label="域名区域" required>
          <n-select v-model:value="createModel.domain_zone_id" :options="zoneOptions()" placeholder="选择启用的域名区域" />
        </n-form-item>
        <n-form-item label="Value" required>
          <n-input v-model:value="createModel.value" :placeholder="createModel.type === 'subdomain' ? '子域名前缀，如 ufi001' : '端口号，如 61001'" />
        </n-form-item>
      </n-form>
    </n-modal>
  </n-space>
</template>
