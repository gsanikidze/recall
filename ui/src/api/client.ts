import type {
  Domain, MemoryHit, MemoryDetail, MemoryFilter,
  CreateMemoryParams, UpdateMemoryParams,
} from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error ?? `HTTP ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export async function listDomains(signal?: AbortSignal): Promise<Domain[]> {
  const data = await request<{ domains: Domain[] }>('/api/domains', { signal })
  return data.domains
}

export async function listMemories(filter: MemoryFilter = {}, signal?: AbortSignal): Promise<MemoryHit[]> {
  const params = new URLSearchParams()
  if (filter.q) params.set('q', filter.q)
  if (filter.domain) params.set('domain', filter.domain)
  if (filter.tags) params.set('tags', filter.tags)
  if (filter.project) params.set('project', filter.project)
  if (filter.lifecycle) params.set('lifecycle', filter.lifecycle)
  if (filter.since) params.set('since', filter.since)
  if (filter.until) params.set('until', filter.until)
  if (filter.include_expired) params.set('include_expired', 'true')
  if (filter.limit) params.set('limit', String(filter.limit))
  const data = await request<{ memories: MemoryHit[] }>(`/api/memories?${params}`, { signal })
  return data.memories ?? []
}

export async function getMemory(id: string, signal?: AbortSignal): Promise<MemoryDetail> {
  return request<MemoryDetail>(`/api/memories/${id}`, { signal })
}

export async function createMemory(params: CreateMemoryParams): Promise<{ id: string; path: string }> {
  return request('/api/memories', { method: 'POST', body: JSON.stringify(params) })
}

export async function updateMemory(id: string, params: UpdateMemoryParams): Promise<MemoryDetail> {
  return request<MemoryDetail>(`/api/memories/${id}`, {
    method: 'PUT',
    body: JSON.stringify(params),
  })
}

export async function deleteMemory(id: string): Promise<void> {
  return request(`/api/memories/${id}`, { method: 'DELETE' })
}

export async function reindex(): Promise<{ indexed: number; deleted: number }> {
  return request('/api/reindex', { method: 'POST' })
}
