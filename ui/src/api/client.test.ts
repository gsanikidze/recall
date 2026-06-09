import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createDomain,
  createMemory,
  deleteMemory,
  getMemory,
  listDomains,
  updateMemory,
} from './client'

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
}

function mockFetch(response: Response) {
  const fetchMock = vi.fn().mockResolvedValue(response)
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function requestHeaders(fetchMock: ReturnType<typeof vi.fn>) {
  const init = fetchMock.mock.calls.at(-1)?.[1] as RequestInit
  return new Headers(init.headers)
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('api client', () => {
  it('returns successful JSON responses and sends Accept header', async () => {
    const fetchMock = mockFetch(jsonResponse({ domains: [{ name: 'tools', description: 'Tools' }] }))

    await expect(listDomains()).resolves.toEqual([{ name: 'tools', description: 'Tools' }])

    expect(requestHeaders(fetchMock).get('Accept')).toBe('application/json')
  })

  it('throws JSON error responses', async () => {
    mockFetch(jsonResponse({ error: 'bad request' }, { status: 422 }))

    await expect(listDomains()).rejects.toThrow('bad request')
  })

  it('throws useful non-JSON error responses', async () => {
    mockFetch(new Response('proxy exploded', { status: 502, headers: { 'Content-Type': 'text/plain' } }))

    await expect(listDomains()).rejects.toThrow('proxy exploded')
  })

  it('supports 204/no-body responses', async () => {
    const fetchMock = mockFetch(new Response(null, { status: 204 }))

    await expect(deleteMemory('01ABC')).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledWith('/api/memories/01ABC', expect.objectContaining({ method: 'DELETE' }))
  })

  it('sets Content-Type only when a body exists', async () => {
    const fetchMock = mockFetch(jsonResponse({ id: '01ABC', path: 'tools/x.md' }))

    await createMemory({ title: 'x', body: 'y', domain: 'tools', importance: 5 })

    expect(fetchMock).toHaveBeenCalledWith('/api/memories', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ title: 'x', body: 'y', domain: 'tools', importance: 5 }),
    }))
    expect(requestHeaders(fetchMock).get('Content-Type')).toBe('application/json')

    fetchMock.mockClear()
    fetchMock.mockResolvedValue(jsonResponse({ domains: [] }))
    await listDomains()
    expect(requestHeaders(fetchMock).has('Content-Type')).toBe(false)
  })

  it('creates domains through the API', async () => {
    const fetchMock = mockFetch(jsonResponse({ name: 'personal-notes', description: 'Private notes' }, { status: 201 }))

    await expect(createDomain({ name: 'personal-notes', description: 'Private notes' })).resolves.toEqual({
      name: 'personal-notes',
      description: 'Private notes',
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/domains', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'personal-notes', description: 'Private notes' }),
    }))
  })

  it('encodes path params', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ id: 'a/b c', title: 'x' }))
      .mockResolvedValueOnce(jsonResponse({ id: 'a/b c', title: 'new' }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await getMemory('a/b c')
    await updateMemory('a/b c', { title: 'new' })
    await deleteMemory('a/b c')

    expect(fetchMock.mock.calls.map(call => call[0])).toEqual([
      '/api/memories/a%2Fb%20c',
      '/api/memories/a%2Fb%20c',
      '/api/memories/a%2Fb%20c',
    ])
  })
})
