import { useState, useEffect, useRef } from 'react'
import { listMemories } from '@/api/client'
import type { MemoryHit, MemoryFilter } from '@/api/types'

export function useMemories(filter: MemoryFilter) {
  const [memories, setMemories] = useState<MemoryHit[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const load = (f: MemoryFilter) => {
    setLoading(true)
    listMemories(f)
      .then(setMemories)
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    if (timerRef.current) clearTimeout(timerRef.current)
    // Debounce text search; other filter changes are immediate.
    const delay = filter.q ? 300 : 0
    timerRef.current = setTimeout(() => load(filter), delay)
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [filter.q, filter.domain, filter.lifecycle, filter.include_expired])

  return { memories, loading, error, reload: () => load(filter) }
}
