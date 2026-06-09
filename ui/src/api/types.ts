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
  importance: number
  score: number
}

export type Lifecycle = 'evergreen' | 'expires'

export type RelationshipType =
  | 'related_to'
  | 'about_project'
  | 'uses_tool'
  | 'depends_on'
  | 'decided_by'
  | 'supersedes'
  | 'contradicts'
  | 'references_person'

export interface Relationship {
  target_id: string
  type: RelationshipType
  note?: string
}

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
  relationships: Relationship[]
  importance: number
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
  relationships?: Relationship[]
  importance?: number
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
  relationships?: Relationship[]
  importance?: number
}
