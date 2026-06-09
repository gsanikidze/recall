export interface Domain {
  name: string
  description: string
}

export interface CreateDomainParams {
  name: string
  description?: string
}

export interface MemoryHit {
  id: string
  title: string
  domain: string
  snippet: string
  path: string
  score: number
}

export type Lifecycle = 'evergreen' | 'expires'

export interface MemoryDetail {
  id: string
  title: string
  domain: string
  tags: string[]
  project: string
  lifecycle: Lifecycle
  expires_on: string
  created: string
  updated: string
  source: string
  links: string[]
  path: string
  body: string
}

export interface MemoryFilter {
  q?: string
  domain?: string
  tags?: string
  project?: string
  lifecycle?: Lifecycle
  since?: string
  until?: string
  include_expired?: boolean
  limit?: number
}

export interface CreateMemoryParams {
  title: string
  body: string
  domain: string
  tags?: string[]
  project?: string
  lifecycle?: Lifecycle
  expires_on?: string
  source?: string
  links?: string[]
}

export interface UpdateMemoryParams {
  title?: string
  body?: string
  tags?: string[]
  project?: string
  lifecycle?: Lifecycle
  expires_on?: string
  source?: string
  links?: string[]
}
