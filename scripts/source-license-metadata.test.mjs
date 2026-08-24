import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const licenseId = 'PolyForm-Noncommercial-1.0.0'
const repositoryUrl = 'git+https://github.com/atlantis-mk/Aivo.git'
const canonicalLicenseSha256 = 'ffcca38841adb694b6f380647e15f17c446a4d1656fed51a1e2041d064c94cc8'

async function read(relativePath) {
  return readFile(path.join(root, relativePath), 'utf8')
}

test('CT-LICENSE-001 keeps the canonical PolyForm Noncommercial text', async () => {
  const license = await read('LICENSE')
  assert.equal(createHash('sha256').update(license).digest('hex'), canonicalLicenseSha256)
  assert.match(license, /PolyForm Noncommercial License 1\.0\.0/)
  assert.match(license, /Any noncommercial purpose is a permitted purpose\./)
})

test('CT-LICENSE-001 keeps pnpm workspace metadata consistent', async () => {
  for (const manifestPath of ['package.json', 'apps/desktop/package.json']) {
    const manifest = JSON.parse(await read(manifestPath))
    assert.equal(manifest.private, true, `${manifestPath} must remain private`)
    assert.equal(manifest.license, licenseId, `${manifestPath} license mismatch`)
    assert.equal(manifest.repository?.url, repositoryUrl, `${manifestPath} repository mismatch`)
  }
})

test('CT-LICENSE-001 keeps licensing boundaries explicit', async () => {
  const readme = await read('README.md')
  const licensing = await read('LICENSING.md')
  const commercial = await read('COMMERCIAL-LICENSE.md')
  const contributing = await read('CONTRIBUTING.md')

  assert.match(readme, /source-available/)
  assert.match(readme, /不是 OSI 认可的开源软件/)
  assert.match(readme, /PolyForm-Noncommercial-1\.0\.0/)
  assert.match(licensing, /^Required Notice: Copyright 2026 atlantis-mk <atlanxg@gmail\.com>$/m)
  assert.match(licensing, /只有双方另行签署的书面协议才授予商业权利/)
  assert.match(licensing, /仅覆盖授权方拥有或有权许可的权利/)
  assert.match(commercial, /不授予任何商业使用/)
  assert.match(contributing, /不会自动把贡献的商业再许可权或版权转让/)
})
