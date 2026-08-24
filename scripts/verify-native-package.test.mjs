import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { selectNativeArtifacts } from './verify-native-package.mjs'

async function fixture(t, names) {
  const directory = await fs.mkdtemp(path.join(os.tmpdir(), 'aivo-native-plan-'))
  t.after(() => fs.rm(directory, { recursive: true, force: true }))
  for (const name of names) await fs.writeFile(path.join(directory, name), name)
  return directory
}

test('AT-UPDATE-001 selects exactly the native handoff packages', async (t) => {
  const mac = await fixture(t, ['Aivo.dmg', 'Aivo-mac.zip'])
  const windows = await fixture(t, ['Aivo Setup.exe'])
  const linux = await fixture(t, ['Aivo.AppImage'])

  assert.equal(path.basename(selectNativeArtifacts(mac, 'darwin').dmg), 'Aivo.dmg')
  assert.equal(path.basename(selectNativeArtifacts(windows, 'win32').installer), 'Aivo Setup.exe')
  assert.equal(path.basename(selectNativeArtifacts(linux, 'linux').appImage), 'Aivo.AppImage')
})

test('AT-UPDATE-001 refuses ambiguous native handoff packages', async (t) => {
  const directory = await fixture(t, ['first.dmg', 'second.dmg', 'Aivo-mac.zip'])
  assert.throws(() => selectNativeArtifacts(directory, 'darwin'), /exactly one macOS DMG/)
})
