import { useState, useCallback } from 'react'
import { Routes, Route, useParams, useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Layout } from '@/components/Layout'
import { DomainSidebar } from '@/components/DomainSidebar'
import { MemoryList } from '@/components/MemoryList'
import { MemoryEditor } from '@/components/MemoryEditor'
import { NewMemoryDialog } from '@/components/NewMemoryDialog'
import { useDomains, useMemories, useMemory, useReindex, keys } from '@/queries'
import { useDebounce } from '@/lib/useDebounce'
import type { MemoryDetail } from '@/api/types'

function AppShell() {
  const { domain, id } = useParams<{ domain?: string; id?: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()

  const [searchQuery, setSearchQuery] = useState('')
  const [showNew, setShowNew] = useState(false)

  const debouncedQuery = useDebounce(searchQuery, 300)

  const { data: domains = [] } = useDomains()
  const { data: memories = [], isLoading } = useMemories({ q: debouncedQuery, domain })
  const { data: selectedMemory } = useMemory(id ?? null)

  const reindexMutation = useReindex()

  const handleSaved = useCallback((updated: MemoryDetail) => {
    navigate(domain ? `/domains/${domain}/${updated.id}` : `/${updated.id}`, { replace: true })
  }, [navigate, domain])

  const handleDeleted = useCallback(() => {
    navigate(domain ? `/domains/${domain}` : '/')
  }, [navigate, domain])

  const handleCreated = useCallback((newId: string) => {
    setShowNew(false)
    navigate(domain ? `/domains/${domain}/${newId}` : `/${newId}`)
  }, [navigate, domain])

  return (
    <>
      <Layout
        sidebar={
          <DomainSidebar
            domains={domains}
            selected={domain ?? null}
            onSelect={d => navigate(d ? `/domains/${d}` : '/')}
            onReindex={() => reindexMutation.mutate(undefined)}
            onAddDomain={() => qc.invalidateQueries({ queryKey: keys.domains() })}
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
            onSelect={memId => navigate(domain ? `/domains/${domain}/${memId}` : `/${memId}`)}
            onNew={() => setShowNew(true)}
          />
        }
        editor={
          selectedMemory ? (
            <MemoryEditor
              memory={selectedMemory}
              onSaved={handleSaved}
              onDeleted={handleDeleted}
            />
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
