import { lazy, Suspense, useState, useCallback } from 'react'
import { Routes, Route, useParams, useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Layout } from '@/components/Layout'
import { DomainSidebar } from '@/components/DomainSidebar'
import { MemoryList } from '@/components/MemoryList'
import { NewDomainDialog } from '@/components/NewDomainDialog'
import { NewMemoryDialog } from '@/components/NewMemoryDialog'
import { useDomains, useMemories, useMemory, useReindex, keys } from '@/queries'
import { useDebounce } from '@/lib/useDebounce'
import { domainRoute, memoryRoute, routeParam } from '@/lib/routes'
import type { MemoryDetail, MemoryFilter } from '@/api/types'

const MemoryEditor = lazy(() => import('@/components/MemoryEditor').then(module => ({ default: module.MemoryEditor })))

function AppShell() {
  const params = useParams<{ domain?: string; id?: string }>()
  const domain = routeParam(params.domain)
  const id = routeParam(params.id)
  const navigate = useNavigate()
  const qc = useQueryClient()

  const [searchQuery, setSearchQuery] = useState('')
  const [showNewDomain, setShowNewDomain] = useState(false)
  const [showNew, setShowNew] = useState(false)
  const [editorDirty, setEditorDirty] = useState(false)

  const debouncedQuery = useDebounce(searchQuery, 300)

  const { data: domains = [] } = useDomains()
  const memoryFilter: MemoryFilter = { q: debouncedQuery }
  if (domain) memoryFilter.domain = domain
  const { data: memories = [], isLoading } = useMemories(memoryFilter)
  const { data: selectedMemory } = useMemory(id ?? null)

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
          />
        }
        editor={
          selectedMemory ? (
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
      <Route path="/:id" element={<AppShell />} />
      <Route path="/domains/:domain" element={<AppShell />} />
      <Route path="/domains/:domain/:id" element={<AppShell />} />
    </Routes>
  )
}
