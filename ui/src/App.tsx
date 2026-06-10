import { lazy, Suspense, useState, useCallback } from 'react'
import { Routes, Route, useParams, useNavigate, useLocation } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Layout } from '@/components/Layout'
import { DomainSidebar } from '@/components/DomainSidebar'
import { MemoryList } from '@/components/MemoryList'
import { NewDomainDialog } from '@/components/NewDomainDialog'
import { NewMemoryDialog } from '@/components/NewMemoryDialog'
import { useDomains, useMemories, useMemory, useGraph, useReindex, keys } from '@/queries'
import { useDebounce } from '@/lib/useDebounce'
import { domainRoute, memoryRoute, graphRoute, routeParam } from '@/lib/routes'
import type { MemoryDetail, MemoryFilter } from '@/api/types'

const MemoryEditor = lazy(() => import('@/components/MemoryEditor').then(module => ({ default: module.MemoryEditor })))
const GraphView = lazy(() => import('@/components/GraphView').then(module => ({ default: module.GraphView })))

function AppShell() {
  const params = useParams<{ domain?: string; id?: string }>()
  const domain = routeParam(params.domain)
  const id = routeParam(params.id)
  const navigate = useNavigate()
  const location = useLocation()
  const qc = useQueryClient()

  const [searchQuery, setSearchQuery] = useState('')
  const [showNewDomain, setShowNewDomain] = useState(false)
  const [showNew, setShowNew] = useState(false)
  const [editorDirty, setEditorDirty] = useState(false)

  const debouncedQuery = useDebounce(searchQuery, 300)
  const isGraph = location.pathname === '/graph' || location.pathname.endsWith('/graph')

  const { data: domains = [] } = useDomains()
  const memoryFilter: MemoryFilter = { q: debouncedQuery }
  if (domain) memoryFilter.domain = domain
  const { data: memories = [], isLoading } = useMemories(memoryFilter)
  const { data: selectedMemory } = useMemory(!isGraph ? id ?? null : null)
  const { data: graph = { nodes: [], edges: [] }, isLoading: graphLoading, error: graphError } = useGraph(domain ?? null)

  const reindexMutation = useReindex()

  const guardedNavigate = useCallback((to: string, options?: { replace?: boolean }) => {
    if (editorDirty && !window.confirm('Discard unsaved changes?')) return
    setEditorDirty(false)
    navigate(to, options)
  }, [editorDirty, navigate])

  const handleSaved = useCallback((updated: MemoryDetail) => {
    setEditorDirty(false)
    navigate(memoryRoute(domain ?? null, updated.id), { replace: true })
  }, [navigate, domain])

  const handleDeleted = useCallback(() => {
    setEditorDirty(false)
    navigate(domainRoute(domain ?? null))
  }, [navigate, domain])

  const handleCreated = useCallback((newId: string) => {
    setShowNew(false)
    setEditorDirty(false)
    navigate(memoryRoute(domain ?? null, newId))
  }, [navigate, domain])

  const handleDomainCreated = useCallback((name: string) => {
    setShowNewDomain(false)
    qc.invalidateQueries({ queryKey: keys.domains() })
    guardedNavigate(domainRoute(name))
  }, [guardedNavigate, qc])

  return (
    <>
      <Layout
        sidebar={
          <DomainSidebar
            domains={domains}
            selected={domain ?? null}
            onSelect={d => guardedNavigate(domainRoute(d))}
            onReindex={() => reindexMutation.mutate(undefined)}
            onAddDomain={() => setShowNewDomain(true)}
            reindexing={reindexMutation.isPending}
          />
        }
        list={
          <MemoryList
            memories={memories}
            loading={isLoading}
            selectedId={id ?? null}
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            onSelect={memId => guardedNavigate(memoryRoute(domain ?? null, memId))}
            onNew={() => setShowNew(true)}
            onGraph={() => guardedNavigate(graphRoute(domain ?? null))}
            graphSelected={isGraph}
          />
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
            <Suspense fallback={<div className="flex items-center justify-center h-full text-white/30 text-sm">Loading editor…</div>}>
              <MemoryEditor
                key={selectedMemory.id}
                memory={selectedMemory}
                onSaved={handleSaved}
                onDeleted={handleDeleted}
                onDirtyChange={setEditorDirty}
              />
            </Suspense>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-white/20 text-sm gap-2">
              <span>Select a memory to edit</span>
              <span className="text-xs">or create a new one</span>
            </div>
          )
        }
      />

      {showNew && (
        <NewMemoryDialog
          domains={domains}
          onCreated={handleCreated}
          onClose={() => setShowNew(false)}
        />
      )}

      {showNewDomain && (
        <NewDomainDialog
          onCreated={handleDomainCreated}
          onClose={() => setShowNewDomain(false)}
        />
      )}
    </>
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
