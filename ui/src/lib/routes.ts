export function routeParam(value: string | undefined): string | undefined {
  return value
}

export function domainRoute(domain: string | null): string {
  return domain ? `/domains/${encodeURIComponent(domain)}` : '/'
}

export function memoryRoute(domain: string | null, id: string): string {
  const encodedId = encodeURIComponent(id)
  return domain ? `${domainRoute(domain)}/${encodedId}` : `/${encodedId}`
}
