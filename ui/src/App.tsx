import { useState, useCallback } from 'react'
import { Routes, Route, useParams, useNavigate } from 'react-router-dom'
import { Layout } from '@/components/Layout'
import { DomainSidebar } from '@/components/DomainSidebar'
import { MemoryList } from '@/components/MemoryList'
import { MemoryEditor } from '@/components/MemoryEditor'
import { NewMemoryDialog } from '@/components/NewMemoryDialog'
import { useDomains } from '@/hooks/useDomains'
import { useMemories } from '@/hooks/useMemories'
import { useMemory } from '@/hooks/useMemory'
import { reindex } from '@/api/client'
import type { MemoryDetail } from '@/api/types'

function AppShell() {
  const { domain, id } = useParams<{ domain?: string; id?: string }>()
  const navigate = useNavigate()

  const [searchQuery, setSearchQuery] = useState('')
  const [showNew, setShowNew] = useState(false)
  const [reindexing, setReindexing] = useState(false)

  const { domains, reload: reloadDomains } = useDomains()
  const { memories, reload: reloadList } = useMemories({
    q: searchQuery,
    domain,
  })
  const { memory: selectedMemory } = useMemory(id ?? null)

  const handleReindex = useCallback(async () => {
    setReindexing(true)
    try { await reindex() } finally {
      setReindexing(false)
      reloadList()
    }
  }, [reloadList])

  const handleSaved = useCallback((updated: MemoryDetail) => {
    reloadList()
    navigate(domain ? `/domains/${domain}/${updated.id}` : `/${updated.id}`, { replace: true })
  }, [reloadList, navigate, domain])

  const handleDeleted = useCallback(() => {
    reloadList()
    navigate(domain ? `/domains/${domain}` : '/')
  }, [reloadList, navigate, domain])

  const handleCreated = useCallback((newId: string) => {
    setShowNew(false)
    reloadList()
    navigate(domain ? `/domains/${domain}/${newId}` : `/${newId}`)
  }, [reloadList, navigate, domain])

  return (
    <>
      <Layout
        sidebar={
          <DomainSidebar
            domains={domains}
            selected={domain ?? null}
            onSelect={d => navigate(d ? `/domains/${d}` : '/')}
            onReindex={handleReindex}
            onAddDomain={() => reloadDomains()}
            reindexing={reindexing}
          />
        }
        list={
          <MemoryList
            memories={memories}
            loading={false}
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
