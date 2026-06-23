import { get, post } from './client'
import type { Admin } from '../types'

export function login(username: string, password: string) {
  return post<{ admin: Admin }>('/auth/login', { username, password })
}

export function logout() {
  return post('/auth/logout')
}

export function getMe() {
  return get<Admin>('/auth/me')
}

export function changePassword(old_password: string, new_password: string) {
  return post('/auth/change-password', { old_password, new_password })
}
