import { Network } from 'lucide-react'
import { MemoryCard } from './MemoryCard'
import { SearchBar } from './SearchBar'
import type { MemoryHit, SearchMode } from '@/api/types'

interface Props {
  memories: MemoryHit[]
  loading: boolean
  selectedId: string | null
  searchQuery: string
  searchMode: SearchMode
  onSearchChange: (q: string) => void
  onSearchModeChange: (mode: SearchMode) => void
  onSelect: (id: string) => void
  onGraph: () => void
  graphSelected?: boolean
}

export function MemoryList({
  memories, loading, selectedId, searchQuery, searchMode,
  onSearchChange, onSearchModeChange, onSelect, onGraph, graphSelected = false,
}: Props) {
  const modes: Array<{ value: SearchMode; label: string }> = [
    { value: 'keyword', label: 'Keyword' },
    { value: 'semantic', label: 'Semantic' },
    { value: 'hybrid', label: 'Hybrid' },
  ]

  return (
    <div className="flex flex-col h-full border-r border-white/5">
      {/* Toolbar */}
      <div className="flex items-center gap-2 p-3 border-b border-white/5">
        <div className="flex-1">
          <SearchBar value={searchQuery} onChange={onSearchChange} />
        </div>
        <button
          onClick={onGraph}
          className={`flex-shrink-0 flex items-center gap-1 px-2.5 py-2 text-xs font-medium rounded-lg transition-colors ${graphSelected ? 'bg-white/15 text-white' : 'bg-white/5 hover:bg-white/10 text-white/70'}`}
        >
          <Network className="w-3.5 h-3.5" /> Graph
        </button>
      </div>

      <div className="px-3 py-2 border-b border-white/5 space-y-2">
        <p className="text-[11px] leading-4 text-white/35">
          Agent-written memories. Browse, search, and inspect stored context; write through MCP/CLI/API.
        </p>
        <div role="group" aria-label="Search mode" className="flex items-center gap-1 rounded-lg bg-white/5 p-1">
          {modes.map(mode => (
            <button
              key={mode.value}
              type="button"
              aria-pressed={searchMode === mode.value}
              onClick={() => onSearchModeChange(mode.value)}
              className={`flex-1 rounded-md px-2 py-1.5 text-[11px] font-medium transition-colors ${searchMode === mode.value ? 'bg-violet-600 text-white' : 'text-white/50 hover:text-white/80 hover:bg-white/5'}`}
            >
              {mode.label}
            </button>
          ))}
        </div>
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
