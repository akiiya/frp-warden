<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NText, NTag, NSpace } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { listAuditLogs } from '../api/auditLogs'
import type { AuditLog } from '../types'

const loading = ref(false)
const logs = ref<AuditLog[]>([])

const columns: DataTableColumns<AuditLog> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '时间', key: 'created_at', width: 170 },
  { title: 'Actor', key: 'actor_type', width: 80 },
  { title: 'Actor ID', key: 'actor_id', width: 80, render: (row) => (row as any).actor_id || '—' },
  {
    title: '操作', key: 'action', width: 180,
    render: (row) => {
      const action = row.action
      let type: 'success' | 'error' | 'warning' | 'info' = 'info'
      if (action.includes('success') || action.includes('created') || action.includes('enabled')) type = 'success'
      else if (action.includes('failed') || action.includes('disabled')) type = 'error'
      else if (action.includes('reset') || action.includes('changed')) type = 'warning'
      return h(NTag, { type, size: 'small' }, { default: () => action })
    },
  },
  { title: '目标类型', key: 'target_type', width: 100 },
  { title: '目标 ID', key: 'target_id', width: 80 },
  { title: '消息', key: 'message', ellipsis: { tooltip: true } },
  { title: 'IP', key: 'ip', width: 130 },
]

async function fetchLogs() {
  loading.value = true
  const res = await listAuditLogs()
  if (res.ok) logs.value = (res.data || []) as AuditLog[]
  loading.value = false
}

onMounted(fetchLogs)
</script>

<template>
  <n-space vertical :size="16">
    <n-card title="审计日志">
      <n-text depth="3" style="display: block; margin-bottom: 12px; font-size: 13px">
        审计日志记录系统操作，不会记录密码、明文 token、session token 等敏感信息。
      </n-text>
      <n-data-table :columns="columns" :data="logs" :loading="loading" :bordered="false" :single-line="false" :max-height="600" />
    </n-card>
  </n-space>
</template>
