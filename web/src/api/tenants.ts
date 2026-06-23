import { get, post } from './client'
import type { Tenant } from '../types'

export function listTenants() {
  return get<Tenant[]>('/tenants')
}

export function createTenant(code: string, name: string, description?: string) {
  return post<{ tenant: Tenant; plain_token: string }>('/tenants', { code, name, description: description || '' })
}

export function disableTenant(id: number) {
  return post(`/tenants/${id}/disable`)
}

export function enableTenant(id: number) {
  return post(`/tenants/${id}/enable`)
}

export function resetTenantToken(id: number) {
  return post<{ plain_token: string }>(`/tenants/${id}/reset-token`)
}
