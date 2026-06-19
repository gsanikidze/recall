import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { MemoryReadView } from './MemoryReadView'
import type { MemoryDetail } from '@/api/types'

function memory(overrides: Partial<MemoryDetail> = {}): MemoryDetail {
  return {
    id: '01MEMORY',
    title: 'Original title',
    domain: 'tools',
    tags: ['mcp', 'agent-written'],
    project: 'recall',
    lifecycle: 'evergreen',
    expires_on: '',
    created: '2026-06-08',
    updated: '2026-06-09',
    source: 'Hermes Agent',
    links: ['01LINK'],
    relationships: [{ target_id: '01TARGET000000000000000001', type: 'uses_tool', note: 'via MCP' }],
    importance: 4,
    path: 'tools/original.md',
    body: 'Original **body**',
    ...overrides,
  }
}

describe('MemoryReadView', () => {
  it('presents memory data as read-only agent-written content', () => {
    render(<MemoryReadView memory={memory()} />)

    expect(screen.getByRole('heading', { name: 'Original title' })).toBeInTheDocument()
    expect(screen.getByText(/agent-written memory/i)).toBeInTheDocument()
    expect(screen.getByText('tools/original.md')).toBeInTheDocument()
    expect(screen.getByText('Original **body**')).toBeInTheDocument()
    expect(screen.getByText('recall')).toBeInTheDocument()
    expect(screen.getByText('Hermes Agent')).toBeInTheDocument()
    expect(screen.getByText('uses_tool')).toBeInTheDocument()
    expect(screen.getByText(/via MCP/i)).toBeInTheDocument()
  })

  it('does not expose editor-style mutation controls', () => {
    render(<MemoryReadView memory={memory()} />)

    expect(screen.queryByRole('button', { name: /^save$/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /delete/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: /body/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: /memory title/i })).not.toBeInTheDocument()
  })
})
