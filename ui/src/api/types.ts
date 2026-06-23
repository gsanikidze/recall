export interface Domain {
  name: string
  description: string
}

export interface Status {
  project_path: string
  vault_path: string
  db_path: string
}

export interface DoctorInvalidFile {
  path: string
  error: string
}

export interface DoctorMissingIndex {
  id: string
  path: string
}

export interface DoctorUnindexedVaultFile {
  id: string
  path: string
}

export interface DoctorDuplicateVaultID {
  id: string
  paths: string[]
}

export interface DoctorEmbeddings {
  provider: string
  model: string
  server_url?: string
  reachable: boolean
  model_available: boolean
  server_error?: string
  available_models?: string[]
  embedded: number
  missing: number
  coverage: number
  missing_embedding_ids?: string[]
}

export interface DoctorSuggestion {
  id: string
  title: string
  severity: string
  prompt: string
}

export interface DoctorReport {
  ok: boolean
  project_path: string
  config_path: string
  vault_path: string
  db_path: string
  domains: number
  memories: number
  vault_memories?: number
  index_memories?: number
  invalid_files?: DoctorInvalidFile[]
  stale_index_ids?: string[]
  missing_index_paths?: DoctorMissingIndex[]
  unindexed_vault_files?: DoctorUnindexedVaultFile[]
  duplicate_vault_ids?: DoctorDuplicateVaultID[]
  embeddings?: DoctorEmbeddings
  suggestions?: DoctorSuggestion[]
  errors: string[]
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
  keyword_score?: number
  semantic_score?: number
}

export type SearchMode = 'keyword' | 'semantic' | 'hybrid'

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

export interface GraphNode {
  id: string
  title: string
  domain: string
  importance: number
  path: string
  missing: boolean
}

export interface GraphEdge {
  id: string
  source: string
  target: string
  type: RelationshipType
  note?: string
}

export interface GraphData {
  nodes: GraphNode[]
  edges: GraphEdge[]
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
  mode?: SearchMode
  provider?: string
  model?: string
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
