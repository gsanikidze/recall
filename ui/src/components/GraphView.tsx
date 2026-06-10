import { ReactFlow, Background, Controls, MiniMap, type Edge, type Node } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import type { GraphData, GraphNode } from '@/api/types'

interface Props {
  graph: GraphData
  loading: boolean
  error?: Error | null
  onSelectMemory: (id: string) => void
}

interface NodeData extends Record<string, unknown> {
  label: string
  domain: string
  importance: number
  missing: boolean
}

function nodePosition(index: number) {
  const columns = 3
  return {
    x: (index % columns) * 260,
    y: Math.floor(index / columns) * 180,
  }
}

function toFlowNode(node: GraphNode, index: number): Node<NodeData> {
  return {
    id: node.id,
    position: nodePosition(index),
    data: {
      label: node.title,
      domain: node.domain,
      importance: node.importance,
      missing: node.missing,
    },
    className: node.missing
      ? 'rounded-xl border border-dashed border-red-400/50 bg-red-950/40 text-red-100 px-3 py-2 text-sm shadow-lg'
      : 'rounded-xl border border-violet-400/40 bg-[#181225] text-white px-3 py-2 text-sm shadow-lg',
  }
}

function edgeLabel(type: string, note?: string) {
  return note ? `${type} — ${note}` : type
}

export function GraphView({ graph, loading, error, onSelectMemory }: Props) {
  if (loading) {
    return <div className="flex h-full items-center justify-center text-sm text-white/40">Loading graph…</div>
  }
  if (error) {
    return <div className="flex h-full items-center justify-center text-sm text-red-300">{error.message}</div>
  }
  if (graph.nodes.length === 0 || graph.edges.length === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 text-sm text-white/30">
        <span>No relationships yet</span>
        <span className="text-xs">Add typed relationships to memories to build the graph.</span>
      </div>
    )
  }

  const nodes: Node<NodeData>[] = graph.nodes.map(toFlowNode)
  const edges: Edge[] = graph.edges.map(edge => ({
    id: edge.id,
    source: edge.source,
    target: edge.target,
    label: edgeLabel(edge.type, edge.note),
    animated: edge.type === 'uses_tool' || edge.type === 'depends_on',
    className: 'text-violet-200',
    labelStyle: { fill: '#ddd6fe', fontSize: 11, fontWeight: 600 },
    style: { stroke: '#8b5cf6', strokeWidth: 2 },
  }))

  return (
    <div className="flex h-full flex-col overflow-hidden bg-[#0d0d12]">
      <div className="border-b border-white/5 px-4 py-3">
        <h2 className="text-sm font-semibold text-white/90">Relationship graph</h2>
        <p className="text-xs text-white/35">{graph.nodes.length} nodes · {graph.edges.length} edges</p>
      </div>
      <div className="min-h-0 flex-1">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          fitView
          onNodeClick={(_, node) => {
            const data = node.data as NodeData
            if (!data.missing) onSelectMemory(node.id)
          }}
        >
          <MiniMap pannable zoomable />
          <Controls />
          <Background />
        </ReactFlow>
      </div>
    </div>
  )
}
