import { useEffect, useState } from 'react'
import MDEditor from '@uiw/react-md-editor'
import { Save, Trash2 } from 'lucide-react'
import { MetadataPanel } from './MetadataPanel'
import { useUpdateMemory, useDeleteMemory } from '@/queries'
import type { MemoryDetail } from '@/api/types'

interface Props {
  memory: MemoryDetail
  onSaved: (updated: MemoryDetail) => void
  onDeleted: () => void
  onDirtyChange?: (dirty: boolean) => void
}

export function MemoryEditor({ memory, onSaved, onDeleted, onDirtyChange }: Props) {
  const [draft, setDraft] = useState<MemoryDetail>(memory)
  const [error, setError] = useState<string | null>(null)

  const updateMutation = useUpdateMemory()
  const deleteMutation = useDeleteMemory()

  const handleSave = () => {
    setError(null)
    if (draft.lifecycle === 'expires' && !draft.expires_on) {
      setError('Expiry date is required when lifecycle is expires.')
      return
    }
    const expiresOn = draft.lifecycle === 'evergreen' ? '' : draft.expires_on
    updateMutation.mutate(
      {
        id: memory.id,
        params: {
          title: draft.title,
          body: draft.body,
          tags: draft.tags,
          project: draft.project,
          lifecycle: draft.lifecycle,
          expires_on: expiresOn,
          source: draft.source,
          links: draft.links,
          importance: draft.importance,
        },
      },
      {
        onSuccess: onSaved,
        onError: (e) => setError(e instanceof Error ? e.message : String(e)),
      },
    )
  }

  const handleDelete = () => {
    if (!confirm(`Delete "${memory.title}"?`)) return
    setError(null)
    deleteMutation.mutate(memory.id, {
      onSuccess: onDeleted,
      onError: (e) => setError(e instanceof Error ? e.message : String(e)),
    })
  }

  const isDirty =
    draft.title !== memory.title ||
    draft.body !== memory.body ||
    draft.project !== memory.project ||
    draft.lifecycle !== memory.lifecycle ||
    draft.expires_on !== memory.expires_on ||
    draft.source !== memory.source ||
    draft.importance !== memory.importance ||
    draft.tags.join('\0') !== memory.tags.join('\0') ||
    draft.links.join('\0') !== memory.links.join('\0')

  useEffect(() => {
    onDirtyChange?.(isDirty)
  }, [isDirty, onDirtyChange])

  useEffect(() => {
    if (!isDirty) return
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ''
    }
    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [isDirty])

  const saving = updateMutation.isPending
  const deleting = deleteMutation.isPending

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Title */}
      <div className="px-4 pt-4 pb-2 border-b border-white/5">
        <input
          aria-label="Memory title"
          value={draft.title}
          onChange={e => setDraft(d => ({ ...d, title: e.target.value }))}
          className="w-full bg-transparent text-lg font-semibold text-white/90 placeholder:text-white/30 focus:outline-none"
          placeholder="Memory title…"
        />
        <div className="flex items-center gap-2 mt-1">
          <span className="text-[11px] px-1.5 py-0.5 rounded bg-violet-500/20 text-violet-300 font-mono">
            {memory.domain}
          </span>
          <span className="text-[11px] text-white/30">{memory.path}</span>
        </div>
      </div>

      {/* Metadata (collapsible) */}
      <MetadataPanel
        memory={draft}
        onChange={patch => setDraft(d => ({ ...d, ...patch }))}
      />

      {/* MD Editor */}
      <div className="flex-1 overflow-hidden">
        <MDEditor
          value={draft.body}
          onChange={(v: string | undefined) => setDraft(d => ({ ...d, body: v ?? '' }))}
          height="100%"
          preview="live"
          visibleDragbar={false}
        />
      </div>

      {/* Footer */}
      <div className="flex items-center gap-3 px-4 py-3 border-t border-white/5 bg-[#111]">
        {error && <span className="flex-1 text-xs text-red-400">{error}</span>}
        {!error && isDirty && (
          <span className="flex-1 text-xs text-white/30">Unsaved changes</span>
        )}
        {!error && !isDirty && <span className="flex-1" />}
        <button
          onClick={handleDelete}
          disabled={deleting}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 rounded transition-colors disabled:opacity-40"
        >
          <Trash2 className="w-3.5 h-3.5" />
          {deleting ? 'Deleting…' : 'Delete'}
        </button>
        <button
          onClick={handleSave}
          disabled={saving || !isDirty}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-violet-600 hover:bg-violet-500 text-white rounded transition-colors disabled:opacity-40"
        >
          <Save className="w-3.5 h-3.5" />
          {saving ? 'Saving…' : 'Save'}
        </button>
      </div>
    </div>
  )
}
