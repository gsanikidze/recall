import { lazy, Suspense, useState, useCallback } from 'react'
import { Routes, Route, useParams, useNavigate, useLocation } from 'react-router-dom'
import { Layout } from '@/components/Layout'
import { DomainSidebar } from '@/components/DomainSidebar'
import { MemoryList } from '@/components/MemoryList'
import { useDomains, useStatus, useMemories, useMemory, useGraph, useReindex } from '@/queries'
import { useDebounce } from '@/lib/useDebounce'
import { domainRoute, memoryRoute, graphRoute, routeParam } from '@/lib/routes'
import type { MemoryFilter, SearchMode } from '@/api/types'

const MemoryReadView = lazy(() => import('@/components/MemoryReadView').then(module => ({ default: module.MemoryReadView })))
const GraphView = lazy(() => import('@/components/GraphView').then(module => ({ default: module.GraphView })))

function AppShell() {
  const params = useParams<{ domain?: string; id?: string }>()
  const domain = routeParam(params.domain)
  const id = routeParam(params.id)
  const navigate = useNavigate()
  const location = useLocation()

  const [searchQuery, setSearchQuery] = useState('')
  const [searchMode, setSearchMode] = useState<SearchMode>('keyword')

  const debouncedQuery = useDebounce(searchQuery, 300)
  const isGraph = location.pathname === '/graph' || location.pathname.endsWith('/graph')

  const { data: status } = useStatus()
  const { data: domains = [] } = useDomains()
  const memoryFilter: MemoryFilter = { q: debouncedQuery }
  if (domain) memoryFilter.domain = domain
  if (searchMode !== 'keyword' && debouncedQuery.trim()) {
    memoryFilter.mode = searchMode
    memoryFilter.provider = 'ollama'
    memoryFilter.model = 'nomic-embed-text'
  }
  const { data: memories = [], isLoading } = useMemories(memoryFilter)
  const { data: selectedMemory } = useMemory(!isGraph ? id ?? null : null)
  const { data: graph = { nodes: [], edges: [] }, isLoading: graphLoading, error: graphError } = useGraph(domain ?? null)

  const reindexMutation = useReindex()

  const guardedNavigate = useCallback((to: string, options?: { replace?: boolean }) => {
    navigate(to, options)
  }, [navigate])

  return (
    <Layout
      projectPath={status?.project_path}
      sidebar={
        <DomainSidebar
          domains={domains}
          selected={domain ?? null}
          onSelect={d => guardedNavigate(domainRoute(d))}
          onReindex={() => reindexMutation.mutate(undefined)}
          reindexing={reindexMutation.isPending}
        />
      }
      list={
        isGraph ? null : (
          <MemoryList
            memories={memories}
            loading={isLoading}
            selectedId={id ?? null}
            searchQuery={searchQuery}
            searchMode={searchMode}
            onSearchChange={setSearchQuery}
            onSearchModeChange={setSearchMode}
            onSelect={memId => guardedNavigate(memoryRoute(domain ?? null, memId))}
            onGraph={() => guardedNavigate(graphRoute(domain ?? null))}
            graphSelected={isGraph}
          />
        )
      }
      editor={
        isGraph ? (
          <Suspense fallback={<div className="flex items-center justify-center h-full text-white/30 text-sm">Loading graph…</div>}>
            <GraphView
              graph={graph}
              loading={graphLoading}
              error={graphError instanceof Error ? graphError : null}
              onSelectMemory={memId => guardedNavigate(memoryRoute(domain ?? null, memId))}
            />
          </Suspense>
        ) : selectedMemory ? (
          <Suspense fallback={<div className="flex items-center justify-center h-full text-white/30 text-sm">Loading memory…</div>}>
            <MemoryReadView
              key={selectedMemory.id}
              memory={selectedMemory}
            />
          </Suspense>
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-white/20 text-sm gap-2">
            <span>Select a memory to view</span>
            <span className="text-xs">agent-written context appears here read-only</span>
          </div>
        )
      }
    />
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<AppShell />} />
      <Route path="/graph" element={<AppShell />} />
      <Route path="/:id" element={<AppShell />} />
      <Route path="/domains/:domain" element={<AppShell />} />
      <Route path="/domains/:domain/graph" element={<AppShell />} />
      <Route path="/domains/:domain/:id" element={<AppShell />} />
    </Routes>
  )
}
