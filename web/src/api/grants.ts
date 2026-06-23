import { get, post } from './client'
import type { ResourceGrant } from '../types'

export function listGrants(tenantId?: number) {
  const q = tenantId ? `?tenant_id=${tenantId}` : ''
  return get<ResourceGrant[]>(`/grants${q}`)
}

export function createGrant(tenant_id: number, resource_id: number) {
  return post<ResourceGrant>('/grants', { tenant_id, resource_id })
}

export function disableGrant(id: number) {
  return post(`/grants/${id}/disable`)
}

export function enableGrant(id: number) {
  return post(`/grants/${id}/enable`)
}
