import { get, post } from './client'
import type { Resource } from '../types'

export function listResources() {
  return get<Resource[]>('/resources')
}

export function createSubdomainResource(domain_zone_id: number, value: string) {
  return post<Resource>('/resources/subdomain', { domain_zone_id, value })
}

export function createTCPPortResource(value: string) {
  return post<Resource>('/resources/tcp-port', { value })
}

export function createUDPPortResource(value: string) {
  return post<Resource>('/resources/udp-port', { value })
}
