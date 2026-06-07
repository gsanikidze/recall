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
              value={memory.tags.join(', ')}
              onChange={e => setTags(e.target.value)}
              className="w-full bg-white/5 border border-white/10 rounded px-2 py-1.5 text-white/80 focus:outline-none focus:border-violet-500/50"
            />
          </div>
          {/* Project */}
          <div>
            <label className="text-white/40 block mb-1">Project</label>
            <input
              value={memory.project}
              onChange={e => onChange({ project: e.target.value })}
              className="w-full bg-white/5 border border-white/10 rounded px-2 py-1.5 text-white/80 focus:outline-none focus:border-violet-500/50"
            />
          </div>
          {/* Lifecycle */}
          <div>
            <label className="text-white/40 block mb-1">Lifecycle</label>
            <select
              value={memory.lifecycle}
              onChange={e => onChange({ lifecycle: e.target.value as MemoryDetail['lifecycle'] })}
              className="w-full bg-white/5 border border-white/10 rounded px-2 py-1.5 text-white/80 focus:outline-none focus:border-violet-500/50"
            >
              <option value="evergreen">evergreen</option>
              <option value="expires">expires</option>
            </select>
          </div>
          {/* Expires on */}
          {memory.lifecycle === 'expires' && (
            <div>
              <label className="text-white/40 block mb-1">Expires on</label>
              <input
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
              value={memory.source}
              onChange={e => onChange({ source: e.target.value })}
              className="w-full bg-white/5 border border-white/10 rounded px-2 py-1.5 text-white/80 focus:outline-none focus:border-violet-500/50"
            />
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
