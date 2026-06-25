import { describe, expect, it } from 'vitest'
import { existsSync, statSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('README cover image and logo', () => {
 it('keeps polished dashboard cover and logo above README fold', () => {
 const repoRoot = resolve(process.cwd(), '..')
 const coverPath = resolve(repoRoot, 'docs/assets/recall-readme-cover.png')
 const logoPath = resolve(repoRoot, 'ui/src/assets/recall-logo.svg')
 const readme = readFileSync(resolve(repoRoot, 'README.md'), 'utf8')
 const indexHtml = readFileSync(resolve(repoRoot, 'ui/index.html'), 'utf8')

 expect(existsSync(coverPath)).toBe(true)
 expect(statSync(coverPath).size).toBeGreaterThan(100_000)
 expect(existsSync(logoPath)).toBe(true)
 expect(readFileSync(logoPath, 'utf8')).toContain('<svg')
 expect(readme.slice(0, 600)).toContain('ui/src/assets/recall-logo.svg')
 expect(readme.slice(0, 600)).toContain('Recall logo')
 expect(readme.slice(0, 600)).toContain('docs/assets/recall-readme-cover.png')
 expect(readme.slice(0, 600)).toContain('Recall dashboard')
 expect(indexHtml).toContain('/src/assets/recall-logo.svg')
 })
})
