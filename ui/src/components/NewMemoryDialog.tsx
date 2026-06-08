import { useState, useEffect } from 'react'
import { X } from 'lucide-react'
import { useCreateMemory } from '@/queries'
import type { Domain } from '@/api/types'

interface Props {
  domains: Domain[]
  onCreated: (id: string) => void
  onClose: () => void
}

export function NewMemoryDialog({ domains, onCreated, onClose }: Props) {
  const [title, setTitle] = useState('')
  const [domain, setDomain] = useState(domains[0]?.name ?? '')
  const [body, setBody] = useState('')
  const [error, setError] = useState<string | null>(null)

  const createMutation = useCreateMemory()

  useEffect(() => {
    if (domain === '' && domains.length > 0) setDomain(domains[0].name)
  }, [domains, domain])

  const handleCreate = () => {
    if (!title.trim() || !domain) {
      setError('Title and domain are required.')
      return
    }
    setError(null)
    createMutation.mutate(
      { title: title.trim(), body, domain },
      {
        onSuccess: (result) => onCreated(result.id),
        onError: (e) => setError(e instanceof Error ? e.message : String(e)),
      },
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#1a1a1a] border border-white/10 rounded-xl shadow-2xl w-full max-w-lg mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-white/5">
          <h2 className="text-sm font-semibold text-white/90">New memory</h2>
          <button onClick={onClose} className="text-white/30 hover:text-white/60">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="p-5 flex flex-col gap-4">
          <div>
            <label className="text-xs text-white/40 block mb-1">Title *</label>
            <input
              autoFocus
              value={title}
              onChange={e => setTitle(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleCreate()}
              placeholder="Short headline for this fact…"
              className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white/80 placeholder:text-white/30 focus:outline-none focus:border-violet-500/50"
            />
          </div>
          <div>
            <label className="text-xs text-white/40 block mb-1">Domain *</label>
            <select
              value={domain}
              onChange={e => setDomain(e.target.value)}
              className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white/80 focus:outline-none focus:border-violet-500/50"
            >
              {domains.map(d => (
                <option key={d.name} value={d.name}>{d.name}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-xs text-white/40 block mb-1">Body (optional — you can edit after creating)</label>
            <textarea
              value={body}
              onChange={e => setBody(e.target.value)}
              rows={4}
              placeholder="The fact itself…"
              className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white/80 placeholder:text-white/30 focus:outline-none focus:border-violet-500/50 resize-none"
            />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-3 px-5 py-4 border-t border-white/5">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-white/50 hover:text-white/80 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleCreate}
            disabled={createMutation.isPending}
            className="px-4 py-2 text-sm font-medium bg-violet-600 hover:bg-violet-500 text-white rounded-lg transition-colors disabled:opacity-40"
          >
            {createMutation.isPending ? 'Creating…' : 'Create memory'}
          </button>
        </div>
      </div>
    </div>
  )
}
