import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type { MemoryDetail } from '@/api/types'

interface Props {
  memory: MemoryDetail
  onChange: (patch: Partial<MemoryDetail>) => void
}

export function MetadataPanel({ memory, onChange }: Props) {
  const [open, setOpen] = useState(false)

  const setTags = (raw: string) => {
    const tags = raw.split(',').map(t => t.trim()).filter(Boolean)
    onChange({ tags })
  }

  return (
    <div className="border-b border-white/5">
      <button
        onClick={() => setOpen(v => !v)}
        className="flex items-center gap-2 w-full px-4 py-2 text-xs text-white/40 hover:text-white/60 transition-colors"
      >
        {open ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
        Metadata
      </button>
      {open && (
        <div className="px-4 pb-3 grid grid-cols-2 gap-3 text-xs">
          {/* Tags */}
          <div className="col-span-2">
            <label className="text-white/40 block mb-1">Tags (comma-separated)</label>
            <input
              aria-label="Tags"
              value={memory.tags.join(', ')}
              onChange={e => setTags(e.target.value)}
              className="w-full bg-white/5 border border-white/10 rounded px-2 py-1.5 text-white/80 focus:outline-none focus:border-violet-500/50"
            />
          </div>
          {/* Project */}
          <div>
            <label className="text-white/40 block mb-1">Project</label>
            <input
              aria-label="Project"
              value={memory.project}
              onChange={e => onChange({ project: e.target.value })}
              className="w-full bg-white/5 border border-white/10 rounded px-2 py-1.5 text-white/80 focus:outline-none focus:border-violet-500/50"
            />
          </div>
          {/* Lifecycle */}
          <div>
            <label className="text-white/40 block mb-1">Lifecycle</label>
            <select
              aria-label="Lifecycle"
              value={memory.lifecycle}
              onChange={e => onChange({ lifecycle: e.target.value as MemoryDetail['lifecycle'] })}
              className="w-full bg-white/5 border border-white/10 rounded px-2 py-1.5 text-white/80 focus:outline-none focus:border-violet-500/50"
            >
              <option value="evergreen">evergreen</option>
              <option value="expires">expires</option>
            </select>
          </div>
          {/* Importance */}
          <div>
            <label className="text-white/40 block mb-1">Importance</label>
            <select
              aria-label="Importance"
              value={memory.importance}
              onChange={e => onChange({ importance: Number(e.target.value) })}
              className="w-full bg-white/5 border border-white/10 rounded px-2 py-1.5 text-white/80 focus:outline-none focus:border-violet-500/50"
            >
              <option value={1}>1 — low</option>
              <option value={2}>2 — useful</option>
              <option value={3}>3 — default</option>
              <option value={4}>4 — high</option>
              <option value={5}>5 — critical</option>
            </select>
          </div>
          {/* Expires on */}
          {memory.lifecycle === 'expires' && (
            <div>
              <label className="text-white/40 block mb-1">Expires on</label>
              <input
                aria-label="Expires on"
                type="date"
                value={memory.expires_on}
                onChange={e => onChange({ expires_on: e.target.value })}
                className="w-full bg-white/5 border border-white/10 rounded px-2 py-1.5 text-white/80 focus:outline-none focus:border-violet-500/50"
              />
            </div>
          )}
          {/* Source */}
          <div>
            <label className="text-white/40 block mb-1">Source</label>
            <input
              aria-label="Source"
              value={memory.source}
              onChange={e => onChange({ source: e.target.value })}
              className="w-full bg-white/5 border border-white/10 rounded px-2 py-1.5 text-white/80 focus:outline-none focus:border-violet-500/50"
            />
          </div>
          {/* Read-only relationships */}
          <div className="col-span-2">
            <div className="text-white/40 block mb-1">Relationships</div>
            {memory.relationships.length === 0 ? (
              <div className="text-white/25">No relationships</div>
            ) : (
              <div className="space-y-1">
                {memory.relationships.map(rel => (
                  <div
                    key={`${rel.target_id}-${rel.type}`}
                    className="rounded border border-white/10 bg-white/5 px-2 py-1 text-white/60"
                  >
                    <span className="font-mono text-violet-300">{rel.type}</span>
                    <span className="mx-1 text-white/25">→</span>
                    <span className="font-mono">{rel.target_id}</span>
                    {rel.note && <span className="ml-2 text-white/40">{rel.note}</span>}
                  </div>
                ))}
              </div>
            )}
          </div>
          {/* Read-only dates */}
          <div className="col-span-2 flex gap-4 text-white/30 mt-1">
            <span>Created: {memory.created}</span>
            <span>Updated: {memory.updated}</span>
          </div>
        </div>
      )}
    </div>
  )
}
