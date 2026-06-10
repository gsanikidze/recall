import { describe, expect, it } from 'vitest'
import { domainRoute, memoryRoute, graphRoute, routeParam } from '@/lib/routes'

describe('route param helpers', () => {
  it('encodes domain and memory id path segments', () => {
    expect(domainRoute('research notes')).toBe('/domains/research%20notes')
    expect(memoryRoute('research notes', 'id/with spaces')).toBe('/domains/research%20notes/id%2Fwith%20spaces')
    expect(memoryRoute(null, 'id/with spaces')).toBe('/id%2Fwith%20spaces')
    expect(graphRoute(null)).toBe('/graph')
    expect(graphRoute('research notes')).toBe('/domains/research%20notes/graph')
  })

  it('keeps router-decoded params stable and leaves missing params as undefined', () => {
    expect(routeParam('id/with spaces')).toBe('id/with spaces')
    expect(routeParam('a%2Fb')).toBe('a%2Fb')
    expect(routeParam('already decoded % value')).toBe('already decoded % value')
    expect(routeParam(undefined)).toBeUndefined()
  })
})
