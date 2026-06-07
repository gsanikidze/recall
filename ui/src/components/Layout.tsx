import type { ReactNode } from 'react'

interface Props {
  sidebar: ReactNode
  list: ReactNode
  editor: ReactNode
}

export function Layout({ sidebar, list, editor }: Props) {
  return (
    <div className="flex h-full overflow-hidden">
      {/* Sidebar */}
      <div className="w-48 flex-shrink-0 flex flex-col">{sidebar}</div>

      {/* Memory list */}
      <div className="w-72 flex-shrink-0 flex flex-col">{list}</div>

      {/* Editor */}
      <div className="flex-1 flex flex-col overflow-hidden">{editor}</div>
    </div>
  )
}
