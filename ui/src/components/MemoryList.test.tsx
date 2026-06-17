import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryList } from './MemoryList'
import type { SearchMode } from '@/api/types'

function renderList(overrides: Partial<React.ComponentProps<typeof MemoryList>> = {}) {
  const props: React.ComponentProps<typeof MemoryList> = {
    memories: [],
    loading: false,
    selectedId: null,
    searchQuery: 'phone sync',
    searchMode: 'keyword',
    onSearchChange: vi.fn(),
    onSearchModeChange: vi.fn(),
    onSelect: vi.fn(),
    onGraph: vi.fn(),
    ...overrides,
  }
  render(<MemoryList {...props} />)
  return props
}

describe('MemoryList search mode toggle', () => {
  it('frames the UI as read/view-first without a primary manual new-memory button', () => {
    renderList()

    expect(screen.getByText(/agent-written/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /graph/i })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^new$/i })).not.toBeInTheDocument()
  })

  it('shows Keyword, Semantic, and Hybrid search mode options', () => {
    renderList()

    expect(screen.getByRole('group', { name: /search mode/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Keyword' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Semantic' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Hybrid' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('calls onSearchModeChange when user selects a mode', async () => {
    const user = userEvent.setup()
    const onSearchModeChange = vi.fn<(mode: SearchMode) => void>()
    renderList({ onSearchModeChange })

    await user.click(screen.getByRole('button', { name: 'Semantic' }))
    await user.click(screen.getByRole('button', { name: 'Hybrid' }))

    expect(onSearchModeChange).toHaveBeenNthCalledWith(1, 'semantic')
    expect(onSearchModeChange).toHaveBeenNthCalledWith(2, 'hybrid')
  })
})
