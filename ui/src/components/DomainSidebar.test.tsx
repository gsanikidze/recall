import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { DomainSidebar } from './DomainSidebar'

const domains = [
  { name: 'tools', description: 'Tools and commands' },
  { name: 'decisions', description: 'Decisions made' },
]

function renderSidebar(overrides: Partial<React.ComponentProps<typeof DomainSidebar>> = {}) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })
  const props: React.ComponentProps<typeof DomainSidebar> = {
    domains,
    selected: null,
    onSelect: vi.fn(),
    onReindex: vi.fn(),
    reindexing: false,
    ...overrides,
  }
  render(
    <QueryClientProvider client={qc}>
      <DomainSidebar {...props} />
    </QueryClientProvider>,
  )
  return props
}

describe('DomainSidebar read-only product navigation', () => {
  it('does not expose manual domain creation controls', () => {
    renderSidebar()

    expect(screen.getByRole('button', { name: /all memories/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /reindex vault/i })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /new domain/i })).not.toBeInTheDocument()
  })

  it('renders the doctor panel above the reindex button', () => {
    renderSidebar()

    const doctorHeader = screen.getByText('Doctor', { exact: true })
    const doctor = doctorHeader.closest('section')
    const reindex = screen.getByRole('button', { name: /reindex vault/i })
    expect(doctor).toBeInTheDocument()
    // doctor section precedes the reindex button in DOM order
    expect(doctor!.compareDocumentPosition(reindex)).toBe(Node.DOCUMENT_POSITION_FOLLOWING)
  })
})
