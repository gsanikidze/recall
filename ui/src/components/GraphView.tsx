import { ReactFlow, Background, Controls, Handle, Position, type Edge, type Node, type NodeProps } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import type { GraphData, GraphEdge } from '@/api/types'

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

type MemoryNode = Node<NodeData, 'memory'>

const domainColor: Record<string, string> = {
  people: '#a78bfa',
  projects: '#38bdf8',
  decisions: '#fbbf24',
  tools: '#34d399',
  goals: '#fb7185',
  research: '#c084fc',
  inbox: '#94a3b8',
}

function edgeCounts(edges: GraphEdge[]) {
  const counts = new Map<string, number>()
  for (const edge of edges) {
    counts.set(edge.source, (counts.get(edge.source) ?? 0) + 1)
    counts.set(edge.target, (counts.get(edge.target) ?? 0) + 1)
  }
  return counts
}

function nodePosition(index: number, total: number) {
  if (index === 0) return { x: 0, y: 0 }

  const ringIndex = index - 1
  const outerCount = Math.max(total - 1, 1)
  const angle = (ringIndex / outerCount) * Math.PI * 2 - Math.PI / 2
  const radiusX = Math.max(420, outerCount * 48)
  const radiusY = 280

  return {
    x: Math.round(Math.cos(angle) * radiusX),
    y: Math.round(Math.sin(angle) * radiusY),
  }
}

function toFlowNodes(graph: GraphData): MemoryNode[] {
  const counts = edgeCounts(graph.edges)
  return [...graph.nodes]
    .sort((a, b) => (counts.get(b.id) ?? 0) - (counts.get(a.id) ?? 0) || a.title.localeCompare(b.title))
    .map((node, index, nodes) => ({
      id: node.id,
      type: 'memory',
      position: nodePosition(index, nodes.length),
      data: {
        label: node.title,
        domain: node.domain,
        importance: node.importance,
        missing: node.missing,
      },
      draggable: true,
    }))
}

function edgeLabel(type: string) {
  return type
}

function MemoryGraphNode({ data }: NodeProps<MemoryNode>) {
  const color = domainColor[data.domain] ?? '#8b5cf6'
  return (
    <div
      className="w-56 rounded-2xl border bg-[#15111f]/95 px-4 py-3 text-left shadow-2xl shadow-black/40 backdrop-blur"
      style={{ borderColor: `${color}99` }}
    >
      <Handle type="target" position={Position.Top} className="!h-2 !w-2 !border-0" style={{ background: color }} />
      <Handle type="source" position={Position.Bottom} className="!h-2 !w-2 !border-0" style={{ background: color }} />
      <div className="mb-2 flex items-center justify-between gap-2">
        <span
          className="rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-black"
          style={{ background: color }}
        >
          {data.domain}
        </span>
        <span className="text-[10px] text-white/35">★ {data.importance}</span>
      </div>
      <div className="line-clamp-2 text-sm font-semibold leading-snug text-white">{data.label}</div>
      {data.missing && <div className="mt-2 text-[11px] text-red-300">Missing target</div>}
    </div>
  )
}

const nodeTypes = { memory: MemoryGraphNode }

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

  const nodes = toFlowNodes(graph)
  const edges: Edge[] = graph.edges.map(edge => ({
    id: edge.id,
    source: edge.source,
    target: edge.target,
    label: edgeLabel(edge.type),
    animated: edge.type === 'uses_tool' || edge.type === 'depends_on',
    type: 'smoothstep',
    labelBgPadding: [6, 3],
    labelBgBorderRadius: 999,
    labelBgStyle: { fill: '#171322', fillOpacity: 0.92 },
    labelStyle: { fill: '#c4b5fd', fontSize: 10, fontWeight: 700 },
    style: { stroke: '#7c3aed', strokeWidth: 1.6, opacity: 0.68 },
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
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.22 }}
          minZoom={0.25}
          maxZoom={1.5}
          proOptions={{ hideAttribution: true }}
          onNodeClick={(_, node) => {
            const data = node.data as NodeData
            if (!data.missing) onSelectMemory(node.id)
          }}
        >
          <Controls showInteractive={false} />
          <Background color="#312447" gap={28} size={1} />
        </ReactFlow>
      </div>
    </div>
  )
}
