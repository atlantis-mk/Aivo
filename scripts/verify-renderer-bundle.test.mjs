import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { verifyRendererBundle } from './verify-renderer-bundle.mjs'

async function rendererFixture(t, html, assets = []) {
  const directory = await fs.mkdtemp(path.join(os.tmpdir(), 'aivo-renderer-bundle-'))
  t.after(() => fs.rm(directory, { recursive: true, force: true }))
  await fs.writeFile(path.join(directory, 'index.html'), html)
  for (const asset of assets) {
    const assetPath = path.join(directory, asset)
    await fs.mkdir(path.dirname(assetPath), { recursive: true })
    await fs.writeFile(assetPath, asset)
  }
  return directory
}

test('packaged renderer accepts relative assets that exist beside index.html', async (t) => {
  const directory = await rendererFixture(
    t,
    '<link href="./assets/app.css"><script src="./assets/app.js"></script>',
    ['assets/app.css', 'assets/app.js'],
  )

  assert.deepEqual(verifyRendererBundle(directory), ['./assets/app.css', './assets/app.js'])
})

test('packaged renderer refuses root-relative assets that fail under file loading', async (t) => {
  const directory = await rendererFixture(
    t,
    '<script src="/assets/app.js"></script>',
    ['assets/app.js'],
  )

  assert.throws(
    () => verifyRendererBundle(directory),
    /must be relative for Electron file loading/,
  )
})

test('packaged renderer refuses missing relative assets', async (t) => {
  const directory = await rendererFixture(t, '<script src="./assets/missing.js"></script>')

  assert.throws(
    () => verifyRendererBundle(directory),
    /missing from the build output/,
  )
})
