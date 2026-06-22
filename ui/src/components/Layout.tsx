import type { ReactNode } from 'react'

interface Props {
  sidebar: ReactNode
  list: ReactNode
  editor: ReactNode
  projectPath?: string | undefined
}

export function Layout({ sidebar, list, editor, projectPath }: Props) {
  return (
    <div className="relative h-full overflow-hidden p-3 sm:p-4 lg:p-6">
      <div className="pointer-events-none absolute left-1/2 top-8 h-[44rem] w-[44rem] -translate-x-1/2 rounded-full bg-[conic-gradient(from_120deg,#38bdf8,#8b5cf6,#fb7185,#34d399,#38bdf8)] opacity-20 blur-3xl" />
      <div className="relative grid h-full overflow-hidden rounded-[2rem] border border-white/15 bg-slate-950/70 shadow-[0_40px_130px_rgba(0,0,0,0.58),inset_0_1px_0_rgba(255,255,255,0.06)] backdrop-blur-xl lg:grid-cols-[15.5rem_minmax(20rem,24rem)_1fr]">
        <div className="min-h-0 border-r border-white/10 bg-slate-950/45">
          <div className="flex gap-2 px-5 pt-5">
            <span className="h-3 w-3 rounded-full bg-[#ff5f57]" />
            <span className="h-3 w-3 rounded-full bg-[#ffbd2e]" />
            <span className="h-3 w-3 rounded-full bg-[#28c840]" />
          </div>
          {projectPath && (
            <div className="mx-4 mt-5 rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-xs shadow-inner shadow-white/[0.02]">
              <div className="uppercase tracking-[0.18em] text-white/35">Active project</div>
              <div className="mt-1 truncate font-mono text-white/75" title={projectPath}>{projectPath}</div>
            </div>
          )}
          {sidebar}
        </div>

        {list && (
          <div data-testid="memory-list-pane" className="hidden min-h-0 border-r border-white/10 bg-slate-950/35 lg:flex lg:flex-col">
            {list}
          </div>
        )}

        <div className="min-w-0 min-h-0 flex flex-col overflow-hidden bg-slate-950/20">{editor}</div>
      </div>
    </div>
  )
}
