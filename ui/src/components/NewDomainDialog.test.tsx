import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { NewDomainDialog } from './NewDomainDialog'

function renderDialog(onCreated = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={client}>
      <NewDomainDialog onCreated={onCreated} onClose={onClose} />
    </QueryClientProvider>,
  )
  return { onCreated, onClose }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('NewDomainDialog', () => {
  it('requires a valid domain name', async () => {
    const user = userEvent.setup()
    renderDialog()

    await user.click(screen.getByRole('button', { name: /create domain/i }))

    expect(screen.getByText('Domain name is required.')).toBeInTheDocument()
  })

  it('creates a domain and notifies parent', async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      name: 'personal-notes',
      description: 'Private notes',
    }), { status: 201, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    const { onCreated } = renderDialog()

    await user.type(screen.getByLabelText(/domain name/i), 'personal-notes')
    await user.type(screen.getByLabelText(/description/i), 'Private notes')
    await user.click(screen.getByRole('button', { name: /create domain/i }))

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith('personal-notes'))
    expect(fetchMock).toHaveBeenCalledWith('/api/domains', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'personal-notes', description: 'Private notes' }),
    }))
  })
})
