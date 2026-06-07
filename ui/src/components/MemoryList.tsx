import { Plus } from 'lucide-react'
import { MemoryCard } from './MemoryCard'
import { SearchBar } from './SearchBar'
import type { MemoryHit } from '@/api/types'

interface Props {
  memories: MemoryHit[]
  loading: boolean
  selectedId: string | null
  searchQuery: string
  onSearchChange: (q: string) => void
  onSelect: (id: string) => void
  onNew: () => void
}

export function MemoryList({
  memories, loading, selectedId, searchQuery,
  onSearchChange, onSelect, onNew,
}: Props) {
  return (
    <div className="flex flex-col h-full border-r border-white/5">
      {/* Toolbar */}
      <div className="flex items-center gap-2 p-3 border-b border-white/5">
        <div className="flex-1">
          <SearchBar value={searchQuery} onChange={onSearchChange} />
        </div>
        <button
          onClick={onNew}
          className="flex-shrink-0 flex items-center gap-1 px-2.5 py-2 text-xs font-medium bg-violet-600 hover:bg-violet-500 text-white rounded-lg transition-colors"
        >
          <Plus className="w-3.5 h-3.5" /> New
        </button>
      </div>

      {/* List */}
      <div className="flex-1 overflow-y-auto">
        {loading && memories.length === 0 && (
          <div className="flex items-center justify-center h-24 text-sm text-white/30">
            Loading…
          </div>
        )}
        {!loading && memories.length === 0 && (
          <div className="flex items-center justify-center h-24 text-sm text-white/30">
            No memories found
          </div>
        )}
        {memories.map(m => (
          <MemoryCard
            key={m.id}
            memory={m}
            selected={m.id === selectedId}
            onClick={() => onSelect(m.id)}
          />
        ))}
      </div>
    </div>
  )
}
