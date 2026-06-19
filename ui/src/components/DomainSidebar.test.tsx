import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { DomainSidebar } from './DomainSidebar'

const domains = [
  { name: 'tools', description: 'Tools and commands' },
  { name: 'decisions', description: 'Decisions made' },
]

describe('DomainSidebar read-only product navigation', () => {
  it('does not expose manual domain creation controls', () => {
    render(
      <DomainSidebar
        domains={domains}
        selected={null}
        onSelect={vi.fn()}
        onReindex={vi.fn()}
        reindexing={false}
      />,
    )

    expect(screen.getByRole('button', { name: /all memories/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /reindex vault/i })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /new domain/i })).not.toBeInTheDocument()
  })
})
