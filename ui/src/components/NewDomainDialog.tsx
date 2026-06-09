import { useState } from 'react'
import { X } from 'lucide-react'
import { useCreateDomain } from '@/queries'

interface Props {
  onCreated: (name: string) => void
  onClose: () => void
}

export function NewDomainDialog({ onCreated, onClose }: Props) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string | null>(null)
  const createMutation = useCreateDomain()

  const handleCreate = () => {
    const trimmedName = name.trim()
    const trimmedDescription = description.trim()
    if (!trimmedName) {
      setError('Domain name is required.')
      return
    }
    if (!/^[a-z0-9][a-z0-9-]*$/.test(trimmedName)) {
      setError('Use lowercase letters, digits, and dashes only.')
      return
    }
    setError(null)
    createMutation.mutate(
      { name: trimmedName, description: trimmedDescription },
      {
        onSuccess: (domain) => onCreated(domain.name),
        onError: (e) => setError(e instanceof Error ? e.message : String(e)),
      },
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#1a1a1a] border border-white/10 rounded-xl shadow-2xl w-full max-w-md mx-4">
        <div className="flex items-center justify-between px-5 py-4 border-b border-white/5">
          <h2 className="text-sm font-semibold text-white/90">New domain</h2>
          <button aria-label="Close" onClick={onClose} className="text-white/30 hover:text-white/60">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-5 flex flex-col gap-4">
          <div>
            <label htmlFor="domain-name" className="text-xs text-white/40 block mb-1">Domain name *</label>
            <input
              id="domain-name"
              autoFocus
              value={name}
              onChange={e => setName(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleCreate()}
              placeholder="personal-notes"
              className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white/80 placeholder:text-white/30 focus:outline-none focus:border-violet-500/50"
            />
            <p className="mt-1 text-[11px] text-white/30">Lowercase letters, digits, and dashes.</p>
          </div>
          <div>
            <label htmlFor="domain-description" className="text-xs text-white/40 block mb-1">Description</label>
            <textarea
              id="domain-description"
              value={description}
              onChange={e => setDescription(e.target.value)}
              rows={3}
              placeholder="What belongs in this domain…"
              className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white/80 placeholder:text-white/30 focus:outline-none focus:border-violet-500/50 resize-none"
            />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
        </div>

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
            {createMutation.isPending ? 'Creating…' : 'Create domain'}
          </button>
        </div>
      </div>
    </div>
  )
}
