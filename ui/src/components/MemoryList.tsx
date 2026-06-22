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
    <div className="flex h-full flex-col">
      <div className="border-b border-white/10 p-4">
        <div className="mb-3 flex items-center gap-2">
          <div className="min-w-0 flex-1">
            <SearchBar value={searchQuery} onChange={onSearchChange} />
          </div>
          <button
            onClick={onGraph}
            className={`flex flex-shrink-0 items-center gap-1.5 rounded-xl px-3 py-2 text-xs font-bold transition-all ${graphSelected ? 'bg-gradient-to-r from-sky-400 to-violet-400 text-slate-950 shadow-[0_12px_30px_rgba(56,189,248,0.18)]' : 'border border-white/10 bg-white/[0.05] text-slate-300 hover:border-sky-400/30 hover:bg-sky-400/10 hover:text-white'}`}
          >
            <Network className="h-3.5 w-3.5" /> Graph
          </button>
        </div>

        <p className="mb-3 text-[11px] leading-4 text-slate-400">
          Agent-written memories. Browse, search, and inspect stored context; write through MCP/CLI/API.
        </p>
        <div role="group" aria-label="Search mode" className="grid grid-cols-3 gap-1 rounded-2xl bg-white/[0.06] p-1">
          {modes.map(mode => (
            <button
              key={mode.value}
              type="button"
              aria-pressed={searchMode === mode.value}
              onClick={() => onSearchModeChange(mode.value)}
              className={`rounded-xl px-2 py-2 text-[11px] font-bold transition-all ${searchMode === mode.value ? 'bg-gradient-to-r from-sky-400 to-violet-400 text-slate-950' : 'text-slate-400 hover:bg-white/[0.06] hover:text-white'}`}
            >
              {mode.label}
            </button>
          ))}
        </div>
      </div>

      <div className="recall-scrollbar min-h-0 flex-1 overflow-y-auto p-3">
        {loading && memories.length === 0 && (
          <div className="flex h-24 items-center justify-center text-sm text-white/30">
            Loading…
          </div>
        )}
        {!loading && memories.length === 0 && (
          <div className="flex h-24 items-center justify-center text-sm text-white/30">
            No memories found
          </div>
        )}
        <div className="space-y-3">
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
    </div>
  )
}
