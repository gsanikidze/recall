import { CalendarDays, GitBranch, Link2, Tag } from 'lucide-react'
import type { ReactNode } from 'react'
import type { MemoryDetail } from '@/api/types'

interface Props {
  memory: MemoryDetail
}

function optionalValue(value: string | undefined | null) {
  return value && value.trim() ? value : '—'
}

function formatDateRange(memory: MemoryDetail) {
  if (memory.created === memory.updated) return memory.created
  return `${memory.created} → ${memory.updated}`
}

export function MemoryReadView({ memory }: Props) {
  return (
    <div className="flex h-full flex-col overflow-hidden">
      <header className="border-b border-white/10 bg-slate-950/35 px-5 py-4">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <div className="mb-3 flex flex-wrap items-center gap-2">
              <span className="rounded-full bg-emerald-400/15 px-2.5 py-1 text-[11px] font-bold text-emerald-200">
                Agent-written memory
              </span>
              <span className="rounded-full bg-sky-300 px-2.5 py-1 font-mono text-[11px] font-extrabold text-slate-950">
                {memory.domain}
              </span>
              <span className="rounded-full bg-white/[0.06] px-2.5 py-1 text-[11px] font-semibold text-slate-300">★ {memory.importance}</span>
            </div>
            <h1 className="truncate text-2xl font-extrabold tracking-tight text-white">{memory.title}</h1>
            <p className="mt-2 break-all font-mono text-xs text-slate-500">{memory.path}</p>
          </div>
        </div>
        <p className="mt-3 max-w-3xl text-xs leading-5 text-slate-400">
          Read-only view. Recall is optimized for agent-prepared durable data; use MCP, CLI, or API writes to change stored memories.
        </p>
      </header>

      <main className="recall-scrollbar flex-1 space-y-5 overflow-y-auto p-5">
        <section aria-label="Memory body" className="rounded-3xl border border-white/10 bg-black/20 p-5 shadow-inner shadow-white/[0.02]">
          <pre className="whitespace-pre-wrap break-words font-sans text-sm leading-7 text-slate-200">{memory.body || 'No body stored.'}</pre>
        </section>

        <section aria-label="Memory metadata" className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <MetaItem label="Project" value={optionalValue(memory.project)} />
          <MetaItem label="Source" value={optionalValue(memory.source)} />
          <MetaItem label="Lifecycle" value={memory.lifecycle} />
          <MetaItem label="Expires on" value={optionalValue(memory.expires_on)} />
          <MetaItem label="Importance" value={String(memory.importance)} />
          <MetaItem label="Created / updated" value={formatDateRange(memory)} />
        </section>

        <Panel icon={<Tag className="h-3.5 w-3.5" />} title="Tags">
          {memory.tags.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {memory.tags.map(tag => (
                <span key={tag} className="rounded-full bg-white/[0.08] px-3 py-1 text-xs font-semibold text-slate-300">{tag}</span>
              ))}
            </div>
          ) : (
            <p className="text-sm text-white/30">No tags.</p>
          )}
        </Panel>

        <Panel icon={<GitBranch className="h-3.5 w-3.5" />} title="Relationships">
          {memory.relationships.length > 0 ? (
            <div className="space-y-2">
              {memory.relationships.map((rel, index) => (
                <div key={`${rel.target_id}-${rel.type}-${index}`} className="rounded-2xl border border-white/10 bg-white/[0.05] p-3 text-xs">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="rounded-full bg-violet-400/15 px-2 py-1 font-mono text-violet-200">{rel.type}</span>
                    <span className="break-all font-mono text-slate-300">{rel.target_id}</span>
                  </div>
                  {rel.note && <p className="mt-2 text-slate-400">{rel.note}</p>}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-white/30">No relationships.</p>
          )}
        </Panel>

        <Panel icon={<Link2 className="h-3.5 w-3.5" />} title="Legacy links">
          {memory.links.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {memory.links.map(link => (
                <span key={link} className="rounded-xl bg-white/[0.08] px-2 py-1 font-mono text-xs text-slate-300">{link}</span>
              ))}
            </div>
          ) : (
            <p className="text-sm text-white/30">No legacy links.</p>
          )}
        </Panel>
      </main>
    </div>
  )
}

function MetaItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-black/20 p-4">
      <div className="flex items-center gap-2 text-[11px] font-bold uppercase tracking-wide text-slate-500">
        <CalendarDays className="h-3 w-3" /> {label}
      </div>
      <div className="mt-2 break-words text-sm text-slate-200">{value}</div>
    </div>
  )
}

function Panel({ icon, title, children }: { icon: ReactNode; title: string; children: ReactNode }) {
  return (
    <section className="rounded-3xl border border-white/10 bg-black/20 p-5">
      <div className="mb-3 flex items-center gap-2 text-xs font-bold uppercase tracking-wide text-slate-500">
        {icon} {title}
      </div>
      {children}
    </section>
  )
}
