import { useEffect } from 'react'
import { CalendarDays, GitBranch, Link2, Tag } from 'lucide-react'
import type { MemoryDetail } from '@/api/types'

interface Props {
  memory: MemoryDetail
  onDirtyChange?: (dirty: boolean) => void
}

function optionalValue(value: string | undefined | null) {
  return value && value.trim() ? value : '—'
}

function formatDateRange(memory: MemoryDetail) {
  if (memory.created === memory.updated) return memory.created
  return `${memory.created} → ${memory.updated}`
}

export function MemoryEditor({ memory, onDirtyChange }: Props) {
  useEffect(() => {
    onDirtyChange?.(false)
  }, [memory.id, onDirtyChange])

  return (
    <div className="flex flex-col h-full overflow-hidden bg-[#111]">
      <header className="px-5 py-4 border-b border-white/5">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-[11px] px-1.5 py-0.5 rounded bg-emerald-500/15 text-emerald-300 font-medium">
                Agent-written memory
              </span>
              <span className="text-[11px] px-1.5 py-0.5 rounded bg-violet-500/20 text-violet-300 font-mono">
                {memory.domain}
              </span>
            </div>
            <h1 className="text-xl font-semibold text-white/90 truncate">{memory.title}</h1>
            <p className="mt-1 text-xs text-white/35 font-mono break-all">{memory.path}</p>
          </div>
        </div>
        <p className="mt-3 text-xs text-white/40 max-w-3xl">
          Read-only view. Recall is optimized for agent-prepared durable data; use MCP, CLI, or API writes to change stored memories.
        </p>
      </header>

      <main className="flex-1 overflow-y-auto p-5 space-y-5">
        <section aria-label="Memory body" className="rounded-xl border border-white/8 bg-black/20 p-4">
          <pre className="whitespace-pre-wrap break-words text-sm leading-6 text-white/80 font-sans">{memory.body || 'No body stored.'}</pre>
        </section>

        <section aria-label="Memory metadata" className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <MetaItem label="Project" value={optionalValue(memory.project)} />
          <MetaItem label="Source" value={optionalValue(memory.source)} />
          <MetaItem label="Lifecycle" value={memory.lifecycle} />
          <MetaItem label="Expires on" value={optionalValue(memory.expires_on)} />
          <MetaItem label="Importance" value={String(memory.importance)} />
          <MetaItem label="Created / updated" value={formatDateRange(memory)} />
        </section>

        <section className="rounded-xl border border-white/8 bg-black/20 p-4">
          <div className="flex items-center gap-2 text-xs font-medium text-white/50 mb-3">
            <Tag className="w-3.5 h-3.5" /> Tags
          </div>
          {memory.tags.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {memory.tags.map(tag => (
                <span key={tag} className="rounded-full bg-white/8 px-2 py-1 text-xs text-white/65">{tag}</span>
              ))}
            </div>
          ) : (
            <p className="text-sm text-white/30">No tags.</p>
          )}
        </section>

        <section className="rounded-xl border border-white/8 bg-black/20 p-4">
          <div className="flex items-center gap-2 text-xs font-medium text-white/50 mb-3">
            <GitBranch className="w-3.5 h-3.5" /> Relationships
          </div>
          {memory.relationships.length > 0 ? (
            <div className="space-y-2">
              {memory.relationships.map((rel, index) => (
                <div key={`${rel.target_id}-${rel.type}-${index}`} className="rounded-lg bg-white/5 p-3 text-xs">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="rounded bg-violet-500/15 px-1.5 py-0.5 text-violet-200 font-mono">{rel.type}</span>
                    <span className="font-mono text-white/60 break-all">{rel.target_id}</span>
                  </div>
                  {rel.note && <p className="mt-2 text-white/45">{rel.note}</p>}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-white/30">No relationships.</p>
          )}
        </section>

        <section className="rounded-xl border border-white/8 bg-black/20 p-4">
          <div className="flex items-center gap-2 text-xs font-medium text-white/50 mb-3">
            <Link2 className="w-3.5 h-3.5" /> Legacy links
          </div>
          {memory.links.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {memory.links.map(link => (
                <span key={link} className="rounded bg-white/8 px-2 py-1 text-xs font-mono text-white/65">{link}</span>
              ))}
            </div>
          ) : (
            <p className="text-sm text-white/30">No legacy links.</p>
          )}
        </section>
      </main>
    </div>
  )
}

function MetaItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-white/8 bg-black/20 p-3">
      <div className="flex items-center gap-2 text-[11px] uppercase tracking-wide text-white/35">
        <CalendarDays className="w-3 h-3" /> {label}
      </div>
      <div className="mt-1 text-sm text-white/75 break-words">{value}</div>
    </div>
  )
}
