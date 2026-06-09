import { describe, expect, it } from 'vitest'
import { cn } from './utils'

describe('cn', () => {
  it('merges class names and resolves Tailwind conflicts', () => {
    const hidden = false
    expect(cn('px-2 text-sm', hidden && 'hidden', 'px-4')).toBe('text-sm px-4')
  })
})
