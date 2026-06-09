import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryEditor } from './MemoryEditor'
import type { MemoryDetail } from '@/api/types'

const updateMutate = vi.fn()
const deleteMutate = vi.fn()

vi.mock('@uiw/react-md-editor', () => ({
  default: ({ value, onChange }: { value?: string; onChange?: (value?: string) => void }) => (
    <textarea aria-label="Body" value={value ?? ''} onChange={e => onChange?.(e.target.value)} />
  ),
}))

vi.mock('@/queries', () => ({
  useUpdateMemory: () => ({ mutate: updateMutate, isPending: false }),
  useDeleteMemory: () => ({ mutate: deleteMutate, isPending: false }),
}))

function memory(overrides: Partial<MemoryDetail> = {}): MemoryDetail {
  return {
    id: '01MEMORY',
    title: 'Original title',
    domain: 'tools',
    tags: [],
    project: '',
    lifecycle: 'evergreen',
    expires_on: '',
    created: '2026-06-08',
    updated: '2026-06-08',
    source: '',
    links: [],
    importance: 3,
    path: 'tools/original.md',
    body: 'Original body',
    ...overrides,
  }
}

describe('MemoryEditor expiry validation', () => {
  beforeEach(() => {
    updateMutate.mockReset()
    deleteMutate.mockReset()
  })

  it('blocks save when lifecycle expires has no date and shows inline error', async () => {
    const user = userEvent.setup()
    render(<MemoryEditor memory={memory()} onSaved={vi.fn()} onDeleted={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: /metadata/i }))
    await user.selectOptions(screen.getByLabelText(/lifecycle/i), 'expires')
    await user.clear(screen.getByRole('textbox', { name: /memory title/i }))
    await user.type(screen.getByRole('textbox', { name: /memory title/i }), 'Changed title')
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    expect(updateMutate).not.toHaveBeenCalled()
    expect(screen.getByText(/expiry date is required/i)).toBeInTheDocument()
  })

  it('clears expires_on before saving evergreen memories', async () => {
    const user = userEvent.setup()
    render(<MemoryEditor memory={memory({ lifecycle: 'expires', expires_on: '2026-12-31' })} onSaved={vi.fn()} onDeleted={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: /metadata/i }))
    await user.selectOptions(screen.getByLabelText(/lifecycle/i), 'evergreen')
    await user.clear(screen.getByRole('textbox', { name: /memory title/i }))
    await user.type(screen.getByRole('textbox', { name: /memory title/i }), 'Evergreen now')
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    expect(updateMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        params: expect.objectContaining({ lifecycle: 'evergreen', expires_on: '' }),
      }),
      expect.any(Object),
    )
  })

  it('notifies dirty state and guards browser unload', async () => {
    const user = userEvent.setup()
    const onDirtyChange = vi.fn()
    render(<MemoryEditor memory={memory()} onSaved={vi.fn()} onDeleted={vi.fn()} onDirtyChange={onDirtyChange} />)

    await user.clear(screen.getByRole('textbox', { name: /memory title/i }))
    await user.type(screen.getByRole('textbox', { name: /memory title/i }), 'Unsaved title')

    expect(onDirtyChange).toHaveBeenLastCalledWith(true)

    const event = new Event('beforeunload', { cancelable: true })
    window.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)
  })

  it('saves changed importance from metadata panel', async () => {
    const user = userEvent.setup()
    render(<MemoryEditor memory={memory({ importance: 3 })} onSaved={vi.fn()} onDeleted={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: /metadata/i }))
    await user.selectOptions(screen.getByLabelText(/importance/i), '5')
    await user.click(screen.getByRole('button', { name: /^save$/i }))

    expect(updateMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        params: expect.objectContaining({ importance: 5 }),
      }),
      expect.any(Object),
    )
  })
})
