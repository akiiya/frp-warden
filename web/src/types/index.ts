// 前端类型定义，与后端 API 响应对齐。

export interface ApiError {
  code: string
  message: string
}

export interface ApiResponse<T = unknown> {
  ok: boolean
  data?: T
  error?: ApiError
}

export interface Admin {
  id: number
  username: string
  must_change_password: boolean
}

export interface Tenant {
  id: number
  code: string
  name: string
  status: string
  description: string
  last_seen_at: string | null
}

export interface DomainZone {
  id: number
  name: string
  zone: string
  status: string
}

export interface Resource {
  id: number
  type: string
  value: string
  domain_zone_id?: number
  status: string
}

export interface ResourceGrant {
  id: number
  tenant_id: number
  resource_id: number
  status: string
}

export interface Proxy {
  id: number
  tenant_id: number
  resource_id: number
  name: string
  proxy_type: string
  local_ip: string
  local_port: number
  status: string
}

export interface AuditLog {
  id: number
  actor_type: string
  actor_id: number | null
  action: string
  target_type: string
  target_id: string
  message: string
  ip: string
  created_at: string
}
