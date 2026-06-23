import { get } from './client'
import type { AuditLog } from '../types'

export function listAuditLogs() {
  return get<AuditLog[]>('/audit-logs')
}
