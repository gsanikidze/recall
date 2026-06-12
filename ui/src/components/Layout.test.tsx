import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Layout } from './Layout'

describe('Layout', () => {
  it('renders the memory list pane when content is provided', () => {
    render(
      <Layout
        sidebar={<div>sidebar</div>}
        list={<div>memory list</div>}
        editor={<div>editor</div>}
      />,
    )

    expect(screen.getByTestId('memory-list-pane')).toHaveTextContent('memory list')
  })

  it('lets full-width views omit the memory list pane', () => {
    render(
      <Layout
        sidebar={<div>sidebar</div>}
        list={null}
        editor={<div>graph workspace</div>}
      />,
    )

    expect(screen.getByText('sidebar')).toBeInTheDocument()
    expect(screen.getByText('graph workspace')).toBeInTheDocument()
    expect(screen.queryByTestId('memory-list-pane')).not.toBeInTheDocument()
  })
})
