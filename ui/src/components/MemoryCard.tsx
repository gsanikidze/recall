import { cn } from '@/lib/utils'
import type { MemoryHit } from '@/api/types'

interface Props {
  memory: MemoryHit
  selected: boolean
  onClick: () => void
}

export function MemoryCard({ memory, selected, onClick }: Props) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'w-full text-left px-4 py-3 border-b border-white/5 hover:bg-white/5 transition-colors',
        selected && 'bg-white/10 border-l-2 border-l-violet-500',
      )}
    >
      <div className="flex items-center gap-2 mb-1">
        <span className="text-[11px] px-1.5 py-0.5 rounded bg-violet-500/20 text-violet-300 font-mono">
          {memory.domain}
        </span>
        {memory.semantic_score != null && memory.semantic_score > 0 && (
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-300 font-mono">
            semantic {memory.semantic_score.toFixed(2)}
          </span>
        )}
        {memory.keyword_score != null && memory.keyword_score > 0 && (
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-sky-500/10 text-sky-300 font-mono">
            keyword {memory.keyword_score.toFixed(2)}
          </span>
        )}
      </div>
      <p className="text-sm font-medium text-white/90 truncate">{memory.title}</p>
      {memory.snippet && (
        <p className="text-xs text-white/40 truncate mt-0.5">{memory.snippet}</p>
      )}
    </button>
  )
}
