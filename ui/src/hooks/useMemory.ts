import { useState, useEffect } from 'react'
import { getMemory } from '@/api/client'
import type { MemoryDetail } from '@/api/types'

export function useMemory(id: string | null) {
  const [memory, setMemory] = useState<MemoryDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) { setMemory(null); return }
    setLoading(true)
    setError(null)
    getMemory(id)
      .then(setMemory)
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
      .finally(() => setLoading(false))
  }, [id])

  return { memory, loading, error }
}
