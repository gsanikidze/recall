import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { GraphView } from './GraphView'
import type { GraphData } from '@/api/types'

vi.mock('@xyflow/react', () => {
  type MockNode = { id: string; data: { label: string; domain: string; importance: number } }
  type MockEdge = { id: string; label?: string }
  type MockReactFlowProps = {
    nodes: MockNode[]
    edges: MockEdge[]
    onNodeClick?: (event: unknown, node: MockNode) => void
  }
  const ReactFlow = ({ nodes, edges, onNodeClick }: MockReactFlowProps) => (
    <div data-testid="react-flow">
      {nodes.map((node) => (
        <button key={node.id} onClick={() => onNodeClick?.({}, node)}>
          {node.data.label} {node.data.domain} importance {node.data.importance}
        </button>
      ))}
      {edges.map((edge) => (
        <div key={edge.id}>{edge.label}</div>
      ))}
    </div>
  )
  return {
    default: ReactFlow,
    ReactFlow,
    Background: () => <div data-testid="background" />,
    Controls: () => <div data-testid="controls" />,
    MiniMap: () => <div data-testid="minimap" />,
  }
})

const graph: GraphData = {
  nodes: [
    { id: '01A', title: 'Hermes MCP', domain: 'tools', importance: 5, path: 'tools/hermes.md', missing: false },
    { id: '01B', title: 'Recall project', domain: 'projects', importance: 4, path: 'projects/recall.md', missing: false },
  ],
  edges: [
    { id: '01A->01B:uses_tool', source: '01A', target: '01B', type: 'uses_tool', note: 'stdio MCP' },
  ],
}

describe('GraphView', () => {
  it('renders graph nodes and typed relationship labels', () => {
    render(<GraphView graph={graph} loading={false} onSelectMemory={vi.fn()} />)

    expect(screen.getByTestId('react-flow')).toBeInTheDocument()
    expect(screen.getByText(/Hermes MCP tools importance 5/i)).toBeInTheDocument()
    expect(screen.getByText(/Recall project projects importance 4/i)).toBeInTheDocument()
    expect(screen.getByText(/uses_tool — stdio MCP/i)).toBeInTheDocument()
  })

  it('selects non-missing memory nodes', async () => {
    const user = userEvent.setup()
    const onSelectMemory = vi.fn()
    render(<GraphView graph={graph} loading={false} onSelectMemory={onSelectMemory} />)

    await user.click(screen.getByRole('button', { name: /Hermes MCP/i }))

    expect(onSelectMemory).toHaveBeenCalledWith('01A')
  })

  it('shows empty state', () => {
    render(<GraphView graph={{ nodes: [], edges: [] }} loading={false} onSelectMemory={vi.fn()} />)

    expect(screen.getByText(/no relationships yet/i)).toBeInTheDocument()
  })
})
