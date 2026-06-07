import { Layers, Plus, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Domain } from '@/api/types'

interface Props {
  domains: Domain[]
  selected: string | null
  onSelect: (domain: string | null) => void
  onReindex: () => void
  onAddDomain: () => void
  reindexing: boolean
}

export function DomainSidebar({
  domains, selected, onSelect, onReindex, onAddDomain, reindexing,
}: Props) {
  return (
    <aside className="flex flex-col h-full bg-[#141414] border-r border-white/5">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-white/5">
        <Layers className="w-4 h-4 text-violet-400" />
        <span className="text-sm font-semibold text-white/80 tracking-wide">recall</span>
      </div>

      {/* Domain list */}
      <div className="flex-1 overflow-y-auto py-2">
        <button
          onClick={() => onSelect(null)}
          className={cn(
            'w-full text-left px-4 py-1.5 text-sm hover:bg-white/5 transition-colors',
            !selected ? 'text-violet-300 font-medium' : 'text-white/50',
          )}
        >
          All memories
        </button>
        {domains.map(d => (
          <button
            key={d.name}
            onClick={() => onSelect(d.name)}
            title={d.description}
            className={cn(
              'w-full text-left px-4 py-1.5 text-sm hover:bg-white/5 transition-colors truncate',
              selected === d.name ? 'text-violet-300 font-medium' : 'text-white/50',
            )}
          >
            {d.name}
          </button>
        ))}
      </div>

      {/* Footer actions */}
      <div className="border-t border-white/5 p-2 flex flex-col gap-1">
        <button
          onClick={onAddDomain}
          className="flex items-center gap-2 px-3 py-1.5 text-xs text-white/40 hover:text-white/70 hover:bg-white/5 rounded transition-colors"
        >
          <Plus className="w-3 h-3" /> New domain
        </button>
        <button
          onClick={onReindex}
          disabled={reindexing}
          className="flex items-center gap-2 px-3 py-1.5 text-xs text-white/40 hover:text-white/70 hover:bg-white/5 rounded transition-colors disabled:opacity-40"
        >
          <RefreshCw className={cn('w-3 h-3', reindexing && 'animate-spin')} />
          {reindexing ? 'Reindexing…' : 'Reindex vault'}
        </button>
      </div>
    </aside>
  )
}
