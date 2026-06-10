import type {
  Domain, MemoryHit, MemoryDetail, MemoryFilter, GraphData,
  CreateDomainParams, CreateMemoryParams, UpdateMemoryParams,
} from './types'

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body != null && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const res = await fetch(path, {
    ...init,
    headers,
  })
  if (!res.ok) {
    const contentType = res.headers.get('Content-Type') ?? ''
    if (contentType.includes('application/json')) {
      const body = await res.json().catch(() => ({}))
      throw new Error(body.error ?? `HTTP ${res.status}`)
    }
    const text = await res.text().catch(() => '')
    throw new Error(text || `HTTP ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

function encodePathParam(value: string) {
  return encodeURIComponent(value)
}

function initWithSignal(signal?: AbortSignal): RequestInit {
  return signal ? { signal } : {}
}

export async function listDomains(signal?: AbortSignal): Promise<Domain[]> {
  const data = await request<{ domains: Domain[] }>('/api/domains', initWithSignal(signal))
  return data.domains
}

export async function createDomain(params: CreateDomainParams): Promise<Domain> {
  return request<Domain>('/api/domains', { method: 'POST', body: JSON.stringify(params) })
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
  const data = await request<{ memories: MemoryHit[] }>(`/api/memories?${params}`, initWithSignal(signal))
  return data.memories ?? []
}

export async function getMemory(id: string, signal?: AbortSignal): Promise<MemoryDetail> {
  return request<MemoryDetail>(`/api/memories/${encodePathParam(id)}`, initWithSignal(signal))
}

export async function getGraph(domain?: string | null, signal?: AbortSignal): Promise<GraphData> {
  const params = new URLSearchParams()
  if (domain) params.set('domain', domain)
  const qs = params.toString()
  return request<GraphData>(`/api/graph${qs ? `?${qs}` : ''}`, initWithSignal(signal))
}

export async function createMemory(params: CreateMemoryParams): Promise<{ id: string; path: string }> {
  return request('/api/memories', { method: 'POST', body: JSON.stringify(params) })
}

export async function updateMemory(id: string, params: UpdateMemoryParams): Promise<MemoryDetail> {
  return request<MemoryDetail>(`/api/memories/${encodePathParam(id)}`, {
    method: 'PUT',
    body: JSON.stringify(params),
  })
}

export async function deleteMemory(id: string): Promise<void> {
  return request(`/api/memories/${encodePathParam(id)}`, { method: 'DELETE' })
}

export async function reindex(): Promise<{ indexed: number; deleted: number }> {
  return request('/api/reindex', { method: 'POST' })
}
