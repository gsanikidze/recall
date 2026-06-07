import { useState, useEffect } from 'react'
import { listDomains } from '@/api/client'
import type { Domain } from '@/api/types'

export function useDomains() {
  const [domains, setDomains] = useState<Domain[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = () => {
    setLoading(true)
    listDomains()
      .then(setDomains)
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [])

  return { domains, loading, error, reload: load }
}
