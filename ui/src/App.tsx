import { useState, useCallback } from 'react'
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

function App() {
  const [selectedDomain, setSelectedDomain] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [showNew, setShowNew] = useState(false)
  const [reindexing, setReindexing] = useState(false)

  const { domains, reload: reloadDomains } = useDomains()
  const { memories, reload: reloadList } = useMemories({
    q: searchQuery,
    domain: selectedDomain ?? undefined,
  })
  const { memory: selectedMemory } = useMemory(selectedId)

  const handleReindex = useCallback(async () => {
    setReindexing(true)
    try { await reindex() } finally {
      setReindexing(false)
      reloadList()
    }
  }, [reloadList])

  const handleSaved = useCallback((updated: MemoryDetail) => {
    // Update local detail cache by re-selecting the same id.
    setSelectedId(null)
    setTimeout(() => setSelectedId(updated.id), 0)
    reloadList()
  }, [reloadList])

  const handleDeleted = useCallback(() => {
    setSelectedId(null)
    reloadList()
  }, [reloadList])

  const handleCreated = useCallback((id: string) => {
    setShowNew(false)
    reloadList()
    setSelectedId(id)
  }, [reloadList])

  return (
    <>
      <Layout
        sidebar={
          <DomainSidebar
            domains={domains}
            selected={selectedDomain}
            onSelect={setSelectedDomain}
            onReindex={handleReindex}
            onAddDomain={() => { /* TODO: add domain dialog */ reloadDomains() }}
            reindexing={reindexing}
          />
        }
        list={
          <MemoryList
            memories={memories}
            loading={false}
            selectedId={selectedId}
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            onSelect={setSelectedId}
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

export default App
