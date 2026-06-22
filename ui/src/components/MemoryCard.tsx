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
        'w-full rounded-3xl border p-4 text-left transition-all',
        selected
          ? 'border-sky-400/40 bg-gradient-to-br from-sky-400/15 to-violet-500/12 shadow-[0_18px_44px_rgba(56,189,248,0.12)]'
          : 'border-white/10 bg-white/[0.045] hover:border-sky-400/25 hover:bg-white/[0.07]',
      )}
    >
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <span className="rounded-full bg-sky-300 px-2.5 py-1 text-[10px] font-extrabold uppercase tracking-wide text-slate-950">
          {memory.domain}
        </span>
        {memory.semantic_score != null && memory.semantic_score > 0 && (
          <span className="rounded-full bg-emerald-400/15 px-2 py-1 font-mono text-[10px] text-emerald-200">
            semantic {memory.semantic_score.toFixed(2)}
          </span>
        )}
        {memory.keyword_score != null && memory.keyword_score > 0 && (
          <span className="rounded-full bg-sky-400/15 px-2 py-1 font-mono text-[10px] text-sky-200">
            keyword {memory.keyword_score.toFixed(2)}
          </span>
        )}
      </div>
      <p className="line-clamp-2 text-sm font-extrabold leading-snug text-white">{memory.title}</p>
      {memory.snippet && (
        <p className="mt-2 line-clamp-2 text-xs leading-5 text-slate-400">{memory.snippet}</p>
      )}
      <div className="mt-3 flex items-center justify-between text-[11px] text-slate-500">
        <span>score {memory.score.toFixed(2)}</span>
        <span>★ {memory.importance}</span>
      </div>
    </button>
  )
}
