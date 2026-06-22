import { describe, expect, it } from 'vitest'
import { existsSync, statSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('README cover image', () => {
  it('keeps a polished dashboard cover above the README fold', () => {
    const repoRoot = resolve(process.cwd(), '..')
    const coverPath = resolve(repoRoot, 'docs/assets/recall-readme-cover.png')
    const readme = readFileSync(resolve(repoRoot, 'README.md'), 'utf8')

    expect(existsSync(coverPath)).toBe(true)
    expect(statSync(coverPath).size).toBeGreaterThan(100_000)
    expect(readme.slice(0, 400)).toContain('docs/assets/recall-readme-cover.png')
    expect(readme.slice(0, 400)).toContain('Recall dashboard')
  })
})
