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
  people: '#fb7185',
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
  const radiusX = Math.max(440, outerCount * 54)
  const radiusY = 300

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

function relationshipColor(type: string) {
  if (type === 'uses_tool') return '#34d399'
  if (type === 'depends_on') return '#38bdf8'
  if (type === 'decided_by') return '#fbbf24'
  if (type === 'references_person') return '#fb7185'
  return '#8b5cf6'
}

function MemoryGraphNode({ data }: NodeProps<MemoryNode>) {
  const color = domainColor[data.domain] ?? '#8b5cf6'
  return (
    <div
      className="w-60 rounded-3xl border bg-[#15111f]/95 px-4 py-3 text-left shadow-[0_24px_70px_rgba(0,0,0,0.42),inset_0_1px_0_rgba(255,255,255,0.08)] backdrop-blur"
      style={{ borderColor: `${color}99`, boxShadow: `0 0 46px ${color}1f, 0 24px 70px rgba(0,0,0,.42)` }}
    >
      <Handle type="target" position={Position.Top} className="!h-2.5 !w-2.5 !border-0" style={{ background: color }} />
      <Handle type="source" position={Position.Bottom} className="!h-2.5 !w-2.5 !border-0" style={{ background: color }} />
      <div className="mb-2 flex items-center justify-between gap-2">
        <span
          className="rounded-full px-2.5 py-1 text-[10px] font-extrabold uppercase tracking-wide text-slate-950"
          style={{ background: color }}
        >
          {data.domain}
        </span>
        <span className="text-[10px] font-semibold text-slate-500">★ {data.importance}</span>
      </div>
      <div className="line-clamp-2 text-sm font-extrabold leading-snug text-white">{data.label}</div>
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
  const edges: Edge[] = graph.edges.map(edge => {
    const color = relationshipColor(edge.type)
    return {
      id: edge.id,
      source: edge.source,
      target: edge.target,
      label: edgeLabel(edge.type),
      animated: edge.type === 'uses_tool' || edge.type === 'depends_on',
      type: 'smoothstep',
      labelBgPadding: [8, 4],
      labelBgBorderRadius: 999,
      labelBgStyle: { fill: '#171322', fillOpacity: 0.94 },
      labelStyle: { fill: '#d8b4fe', fontSize: 10, fontWeight: 800 },
      style: { stroke: color, strokeWidth: 2, opacity: 0.72, filter: `drop-shadow(0 0 8px ${color}55)` },
    }
  })

  const domains = new Set(graph.nodes.map(node => node.domain)).size
  const relationshipTypes = new Set(graph.edges.map(edge => edge.type)).size

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="border-b border-white/10 bg-slate-950/35 px-5 py-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-extrabold tracking-tight text-white">Relationship graph</h2>
            <p className="mt-1 text-xs text-slate-500">{graph.nodes.length} nodes · {graph.edges.length} edges</p>
          </div>
          <div className="flex flex-wrap gap-2 text-[11px] font-bold uppercase tracking-wide">
            <span className="rounded-full border border-sky-400/20 bg-sky-400/10 px-3 py-1.5 text-sky-100">{domains} domains</span>
            <span className="rounded-full border border-violet-400/20 bg-violet-400/10 px-3 py-1.5 text-violet-100">{relationshipTypes} relation types</span>
          </div>
        </div>
      </div>
      <div className="min-h-0 flex-1 bg-[radial-gradient(circle_at_center,rgba(139,92,246,0.14),transparent_58%)]">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.24 }}
          minZoom={0.25}
          maxZoom={1.5}
          proOptions={{ hideAttribution: true }}
          onNodeClick={(_, node) => {
            const data = node.data as NodeData
            if (!data.missing) onSelectMemory(node.id)
          }}
        >
          <Controls showInteractive={false} />
          <Background color="#334155" gap={30} size={1} />
        </ReactFlow>
      </div>
    </div>
  )
}
