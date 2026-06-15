import type { ReactNode } from 'react'

interface Props {
  sidebar: ReactNode
  list: ReactNode
  editor: ReactNode
  projectPath?: string | undefined
}

export function Layout({ sidebar, list, editor, projectPath }: Props) {
  return (
    <div className="flex h-full overflow-hidden">
      {/* Sidebar */}
      <div className="w-48 flex-shrink-0 flex flex-col">
        {projectPath && (
          <div className="border-b border-white/10 px-3 py-2 text-xs text-white/50">
            <div className="uppercase tracking-wide text-white/30">Active project</div>
            <div className="truncate text-white/70" title={projectPath}>{projectPath}</div>
          </div>
        )}
        {sidebar}
      </div>

      {/* Memory list */}
      {list && <div data-testid="memory-list-pane" className="w-72 flex-shrink-0 flex flex-col">{list}</div>}

      {/* Editor */}
      <div className="flex-1 flex flex-col overflow-hidden">{editor}</div>
    </div>
  )
}
