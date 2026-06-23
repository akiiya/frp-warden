// 统一 API 客户端。使用 fetch + credentials:include 确保 cookie session 生效。
// 401 时自动跳转登录页；错误显示后端 message。

import type { ApiResponse } from '../types'

const BASE = '/api'

async function request<T>(method: string, path: string, body?: unknown): Promise<ApiResponse<T>> {
  const opts: RequestInit = {
    method,
    headers: {},
    credentials: 'include',
  }
  if (body !== undefined) {
    opts.headers = { 'Content-Type': 'application/json' }
    opts.body = JSON.stringify(body)
  }

  const res = await fetch(`${BASE}${path}`, opts)

  // 401 → 跳转登录。
  if (res.status === 401) {
    // 避免在登录页本身循环跳转。
    if (!window.location.hash.includes('/login')) {
      window.location.hash = '#/login'
    }
    const data = await res.json().catch(() => ({ ok: false, error: { code: 'UNAUTHORIZED', message: '请先登录' } }))
    return data as ApiResponse<T>
  }

  return res.json()
}

export function get<T>(path: string) { return request<T>('GET', path) }
export function post<T>(path: string, body?: unknown) { return request<T>('POST', path, body) }
export function put<T>(path: string, body?: unknown) { return request<T>('PUT', path, body) }
export function del<T>(path: string) { return request<T>('DELETE', path) }
