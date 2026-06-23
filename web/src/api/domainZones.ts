import { get, post } from './client'
import type { DomainZone } from '../types'

export function listDomainZones() {
  return get<DomainZone[]>('/domain-zones')
}

export function createDomainZone(name: string, zone: string) {
  return post<DomainZone>('/domain-zones', { name, zone })
}

export function disableDomainZone(id: number) {
  return post(`/domain-zones/${id}/disable`)
}

export function enableDomainZone(id: number) {
  return post(`/domain-zones/${id}/enable`)
}
