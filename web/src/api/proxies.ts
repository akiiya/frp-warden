import { get, post } from './client'
import type { Proxy } from '../types'

export function listProxies(tenantId?: number) {
  const q = tenantId ? `?tenant_id=${tenantId}` : ''
  return get<Proxy[]>(`/proxies${q}`)
}

export function createProxy(data: {
  tenant_id: number
  resource_id: number
  name: string
  proxy_type: string
  local_ip?: string
  local_port: number
}) {
  return post<Proxy>('/proxies', data)
}

export function disableProxy(id: number) {
  return post(`/proxies/${id}/disable`)
}

export function enableProxy(id: number) {
  return post(`/proxies/${id}/enable`)
}
