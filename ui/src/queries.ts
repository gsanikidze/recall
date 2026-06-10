import {
  useQuery,
  useMutation,
  useQueryClient,
  keepPreviousData,
} from '@tanstack/react-query'
import {
  listDomains,
  createDomain,
  listMemories,
  getMemory,
  getGraph,
  createMemory,
  updateMemory,
  deleteMemory,
  reindex,
} from '@/api/client'
import type { MemoryFilter, CreateDomainParams, CreateMemoryParams, UpdateMemoryParams } from '@/api/types'

export const keys = {
  domains: () => ['domains'] as const,
  memories: (filter: MemoryFilter) => ['memories', filter] as const,
  memory: (id: string | null) => ['memory', id] as const,
  graph: (domain: string | null) => ['graph', domain] as const,
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
