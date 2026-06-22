import type { ReactNode } from 'react'

interface Props {
  sidebar: ReactNode
  list: ReactNode
  editor: ReactNode
  projectPath?: string | undefined
}

export function Layout({ sidebar, list, editor, projectPath }: Props) {
  return (
    <div
      className={`flex h-full overflow-hidden ${
        list ? 'lg:grid lg:grid-cols-[15.5rem_minmax(20rem,24rem)_1fr]' : 'lg:grid lg:grid-cols-[15.5rem_1fr]'
      }`}
    >
      <div className="min-h-0 border-r border-white/10 bg-slate-950/45">
        {projectPath && (
          <div className="border-b border-white/10 px-4 py-3 text-xs">
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
  )
}
