import { Database, Layers, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Domain } from '@/api/types'

interface Props {
  domains: Domain[]
  selected: string | null
  onSelect: (domain: string | null) => void
  onReindex: () => void
  reindexing: boolean
}

const domainColors = ['bg-sky-400', 'bg-emerald-400', 'bg-amber-300', 'bg-rose-400', 'bg-violet-400', 'bg-slate-400']

const activeCls = 'border-sky-400/25 bg-gradient-to-r from-sky-400/15 to-violet-500/15 text-white shadow-[0_14px_34px_rgba(56,189,248,0.10)]'
const idleCls = 'border-transparent text-slate-400 hover:bg-white/[0.05] hover:text-white'

export function DomainSidebar({
  domains, selected, onSelect, onReindex, reindexing,
}: Props) {
  return (
    <aside className="flex h-[calc(100%-2rem)] flex-col px-4 pb-4 pt-5">
      <div className="mb-6 flex items-center gap-3 px-1">
        <div className="grid h-12 w-12 place-items-center rounded-2xl border border-white/15 bg-gradient-to-br from-sky-400/25 to-violet-500/30 shadow-[0_18px_50px_rgba(56,189,248,0.18)]">
          <Layers className="h-5 w-5 text-sky-200" />
        </div>
        <div>
          <div className="text-base font-extrabold tracking-tight text-white">Recall</div>
          <div className="text-xs text-slate-400">Local memory vault</div>
        </div>
      </div>

      <div className="mb-3 px-1 text-[11px] font-bold uppercase tracking-[0.18em] text-slate-500">Vault</div>
      <div className="recall-scrollbar min-h-0 flex-1 overflow-y-auto pr-1">
        <button
          onClick={() => onSelect(null)}
          className={cn(
            'mb-2 flex w-full items-center justify-between rounded-2xl border px-3 py-3 text-left text-sm transition-colors duration-150',
            !selected ? activeCls : idleCls,
          )}
        >
          <span className="flex items-center gap-3 font-semibold"><span className="h-2.5 w-2.5 rounded-full bg-sky-400 shadow-[0_0_18px_currentColor]" />All memories</span>
          <span className="text-xs text-slate-500">all</span>
        </button>
        {domains.map((d, index) => (
          <button
            key={d.name}
            onClick={() => onSelect(d.name)}
            title={d.description}
            className={cn(
              'mb-2 flex w-full items-center justify-between gap-3 rounded-2xl border px-3 py-3 text-left text-sm transition-colors duration-150',
              selected === d.name ? activeCls : idleCls,
            )}
          >
            <span className="flex min-w-0 items-center gap-3 font-semibold">
              <span className={cn('h-2.5 w-2.5 flex-shrink-0 rounded-full shadow-[0_0_18px_currentColor]', domainColors[index % domainColors.length])} />
              <span className="truncate">{d.name}</span>
            </span>
          </button>
        ))}
      </div>

      <div className="mt-4 rounded-2xl border border-white/10 bg-black/20 p-3">
        <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
          <Database className="h-3.5 w-3.5" /> Index
        </div>
        <button
          onClick={onReindex}
          disabled={reindexing}
          className="flex w-full items-center justify-center gap-2 rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-xs font-semibold text-slate-300 transition-colors hover:border-sky-400/30 hover:bg-sky-400/10 hover:text-white disabled:opacity-40"
        >
          <RefreshCw className={cn('h-3.5 w-3.5', reindexing && 'animate-spin')} />
          {reindexing ? 'Reindexing…' : 'Reindex vault'}
        </button>
      </div>
    </aside>
  )
}
