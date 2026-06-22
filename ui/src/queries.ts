import {
  useQuery,
  useMutation,
  useQueryClient,
  keepPreviousData,
} from '@tanstack/react-query'
import {
  listDomains,
  getStatus,
  createDomain,
  listMemories,
  getMemory,
  getGraph,
  createMemory,
  updateMemory,
  deleteMemory,
  reindex,
  getDoctor,
} from '@/api/client'
import type { MemoryFilter, CreateDomainParams, CreateMemoryParams, UpdateMemoryParams } from '@/api/types'

export const keys = {
  status: () => ['status'] as const,
  domains: () => ['domains'] as const,
  memories: (filter: MemoryFilter) => ['memories', filter] as const,
  memory: (id: string | null) => ['memory', id] as const,
  graph: (domain: string | null) => ['graph', domain] as const,
  doctor: (deep: boolean, embeddings: boolean) => ['doctor', { deep, embeddings }] as const,
}

export function useStatus() {
  return useQuery({
    queryKey: keys.status(),
    queryFn: ({ signal }) => getStatus(signal),
  })
}

export function useDomains() {
  return useQuery({
    queryKey: keys.domains(),
    queryFn: ({ signal }) => listDomains(signal),
  })
}

export function useCreateDomain() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params: CreateDomainParams) => createDomain(params),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.domains() }),
  })
}

export function useMemories(filter: MemoryFilter) {
  return useQuery({
    queryKey: keys.memories(filter),
    queryFn: ({ signal }) => listMemories(filter, signal),
    placeholderData: keepPreviousData,
  })
}

export function useMemory(id: string | null) {
  return useQuery({
    queryKey: keys.memory(id),
    queryFn: ({ signal }) => getMemory(id!, signal),
    enabled: !!id,
  })
}

export function useGraph(domain: string | null) {
  return useQuery({
    queryKey: keys.graph(domain),
    queryFn: ({ signal }) => getGraph(domain, signal),
    placeholderData: keepPreviousData,
  })
}

export function useCreateMemory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params: CreateMemoryParams) => createMemory(params),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.domains() })
      qc.invalidateQueries({ queryKey: ['memories'] })
    },
  })
}

export function useUpdateMemory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, params }: { id: string; params: UpdateMemoryParams }) =>
      updateMemory(id, params),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ['memories'] })
      qc.invalidateQueries({ queryKey: keys.memory(id) })
    },
  })
}

export function useDeleteMemory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteMemory(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ['memories'] })
      qc.removeQueries({ queryKey: keys.memory(id) })
    },
  })
}

export function useReindex() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: reindex,
    onSuccess: () => qc.invalidateQueries(),
  })
}

export function useDoctor(opts: { deep?: boolean; embeddings?: boolean; provider?: string; model?: string } = {}) {
  const deep = !!opts.deep
  // Embeddings audit runs by default so the panel surfaces Ollama/model
  // problems without the user having to opt in.
  const embeddings = opts.embeddings !== false
  return useQuery({
    queryKey: keys.doctor(deep, embeddings),
    queryFn: ({ signal }) =>
      getDoctor(
        {
          deep,
          embeddings,
          ...(opts.provider ? { provider: opts.provider } : {}),
          ...(opts.model ? { model: opts.model } : {}),
        },
        signal,
      ),
  })
}
