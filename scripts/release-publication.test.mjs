import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import {
  collectPlatformArtifacts,
  createPublication,
  validateReleaseSource,
  validateStableReleaseSource,
} from './release-publication.mjs'
import { macAppCandidates } from './smoke-release.mjs'

async function temporaryDirectory(t) {
  const directory = await fs.mkdtemp(path.join(os.tmpdir(), 'aivo-release-test-'))
  t.after(() => fs.rm(directory, { recursive: true, force: true }))
  return directory
}

async function writeFixture(directory, name, content = name) {
  await fs.mkdir(directory, { recursive: true })
  await fs.writeFile(path.join(directory, name), content)
}

test('CT-RELEASE-001 normalizes the required native platform artifacts', async (t) => {
  const root = await temporaryDirectory(t)
  const fixtures = {
    'darwin-aarch64': ['Aivo-1.2.3-arm64.dmg', 'Aivo-1.2.3-arm64-mac.zip'],
    'darwin-x86_64': ['Aivo-1.2.3.dmg', 'Aivo-1.2.3-mac.zip'],
    'windows-x86_64': ['Aivo Setup 1.2.3.exe'],
    'linux-x86_64': ['Aivo-1.2.3.AppImage'],
  }
  const merged = path.join(root, 'merged')

  for (const [platform, names] of Object.entries(fixtures)) {
    const input = path.join(root, platform)
    for (const name of names) {
      await writeFixture(input, name)
    }
    await collectPlatformArtifacts({ input, output: merged, platform, version: '1.2.3' })
  }

  const publication = await createPublication({
    input: merged,
    output: path.join(root, 'publication'),
    version: '1.2.3',
    baseUrl: 'https://downloads.example.test/',
    prefix: 'aivo',
    publishedAt: '2026-08-24T00:00:00Z',
  })

  assert.equal(publication.artifacts.length, 6)
  assert.equal(publication.latest.version, '1.2.3')
  assert.equal(publication.latest.artifacts.length, 7)
  assert.equal(
    publication.latest.releaseBaseUrl,
    'https://downloads.example.test/aivo/releases/v1.2.3',
  )
  const rows = await fs.readFile(path.join(root, 'publication', 'immutable.tsv'), 'utf8')
  assert.match(rows, /aivo\/releases\/v1\.2\.3\/Aivo_1\.2\.3_windows-x86_64-setup\.exe/)
  assert.match(rows, /aivo\/releases\/v1\.2\.3\/SHA256SUMS/)
})

test('CT-RELEASE-001 refuses ambiguous and mutated build outputs', async (t) => {
  const root = await temporaryDirectory(t)
  const input = path.join(root, 'input')
  await writeFixture(input, 'first.dmg')
  await writeFixture(input, 'second.dmg')
  await writeFixture(input, 'only.zip')
  await assert.rejects(
    collectPlatformArtifacts({
      input,
      output: path.join(root, 'output'),
      platform: 'darwin-aarch64',
      version: '1.2.3',
    }),
    /requires exactly one \.dmg artifact/,
  )
})

test('CT-RELEASE-001 binds a tag to both package manifests and a release record', async (t) => {
  const root = await temporaryDirectory(t)
  await writeFixture(root, 'package.json', JSON.stringify({ version: '1.2.3' }))
  await writeFixture(
    path.join(root, 'apps', 'desktop'),
    'package.json',
    JSON.stringify({ version: '1.2.3' }),
  )
  await writeFixture(path.join(root, 'releases'), 'v1.2.3.md', '# Release')

  const record = await validateReleaseSource({ root, version: '1.2.3', tag: 'v1.2.3' })
  assert.equal(record, path.join(root, 'releases', 'v1.2.3.md'))
  await assert.rejects(
    validateReleaseSource({ root, version: '1.2.3', tag: 'v1.2.4' }),
    /does not match package version/,
  )
})

test('CT-RELEASE-002 keeps non-stable tags out of operator-triggered stable publication', async (t) => {
  const root = await temporaryDirectory(t)
  await writeFixture(root, 'package.json', JSON.stringify({ version: '1.2.3-rc.1' }))
  await writeFixture(
    path.join(root, 'apps', 'desktop'),
    'package.json',
    JSON.stringify({ version: '1.2.3-rc.1' }),
  )
  await writeFixture(path.join(root, 'releases'), 'v1.2.3-rc.1.md', '# Preview')

  await assert.rejects(
    validateStableReleaseSource({ root, version: '1.2.3-rc.1', tag: 'v1.2.3-rc.1' }),
    /must be a plain SemVer version/,
  )
})

test('CT-RELEASE-001 keeps packaging names safe and disables implicit publication', async () => {
  const desktop = JSON.parse(
    await fs.readFile(path.join(import.meta.dirname, '..', 'apps', 'desktop', 'package.json'), 'utf8'),
  )

  assert.equal(desktop.build?.executableName, 'aivo')
  for (const name of ['package', 'package:mac', 'package:win', 'package:linux']) {
    assert.match(desktop.scripts?.[name] ?? '', /--publish never(?:\s|$)/, `${name} may publish implicitly`)
  }
})

test('CT-RELEASE-001 smokes native macOS output directories', () => {
  assert.deepEqual(
    macAppCandidates('/build/desktop'),
    [
      path.join('/build/desktop', 'mac', 'Aivo.app'),
      path.join('/build/desktop', 'mac-arm64', 'Aivo.app'),
      path.join('/build/desktop', 'mac-x64', 'Aivo.app'),
    ],
  )
})

test('CT-RELEASE-001 keeps R2-first GitHub publication resumable and digest-bound', async () => {
  const workflow = await fs.readFile(
    path.join(import.meta.dirname, '..', '.github', 'workflows', 'publish-release.yml'),
    'utf8',
  )

  assert.match(workflow, /needs: \[build, publish_r2\]/)
  assert.match(workflow, /release_record="releases\/\$\{tag\}\.md"/)
  assert.match(workflow, /Release record must contain exactly one non-empty H1/)
  assert.match(workflow, /--title "\$release_title"/)
  assert.match(workflow, /--notes-file "\$release_record"/)
  assert.match(workflow, /gh release edit "\$tag"[\s\S]*--title "\$release_title"[\s\S]*--notes-file "\$release_record"/)
  assert.match(workflow, /workflow_dispatch:/)
  assert.match(workflow, /RELEASE_TAG: \$\{\{ inputs\.release_tag \|\| github\.ref_name \}\}/)
  assert.match(workflow, /--json databaseId --jq '\.databaseId'/)
  assert.match(workflow, /releases\/\$\{release_id\}/)
  assert.match(workflow, /source_run_id:/)
  assert.match(workflow, /release-publication\.mjs validate-stable/)
  assert.match(workflow, /Recover GitHub Release from published artifacts/)
  assert.match(workflow, /git rev-parse "\$RELEASE_TAG\^\{commit\}"/)
  assert.match(workflow, /run-id: \$\{\{ inputs\.source_run_id \}\}/)
  assert.match(workflow, /Refusing to reuse GitHub asset without digest evidence/)
  assert.match(workflow, /if \[\[ "\$\(jq -r '\.draft'/)
  assert.ok(workflow.indexOf('Publish stable manifest last') < workflow.indexOf('Publish GitHub Release assets'))
})

test('AT-UPDATE-001 runs release quality on every native update target', async () => {
  const workflow = await fs.readFile(
    path.join(import.meta.dirname, '..', '.github', 'workflows', 'release-quality.yml'),
    'utf8',
  )

  for (const runner of ['macos-15', 'macos-15-intel', 'windows-2025', 'ubuntu-24.04']) {
    assert.match(workflow, new RegExp(`os: ${runner.replaceAll('.', '\\.')}`))
  }
  assert.match(workflow, /Verify native installer handoff package/)
  assert.match(workflow, /node scripts\/verify-native-package\.mjs --input build\/desktop/)
})

test('CT-RELEASE-002 keeps stable publication operator-managed and mechanically bound', async () => {
  const workflow = await fs.readFile(
    path.join(import.meta.dirname, '..', '.github', 'workflows', 'publish-release.yml'),
    'utf8',
  )

  assert.match(workflow, /workflow_dispatch:/)
  assert.match(workflow, /\n\s+push:/)
  for (const platform of ['darwin-aarch64', 'darwin-x86_64', 'windows-x86_64', 'linux-x86_64']) {
    assert.match(workflow, new RegExp(platform))
  }
  assert.doesNotMatch(workflow, /\n\s+verify:\n/)
  assert.doesNotMatch(workflow, /needs: verify/)
  for (const gate of ['pnpm docs:check', 'pnpm scripts:test', 'pnpm test:core', 'pnpm lint', 'pnpm build']) {
    assert.doesNotMatch(workflow, new RegExp(gate.replaceAll(':', '\\:')))
  }
  assert.doesNotMatch(workflow, /pnpm smoke:release/)
  assert.doesNotMatch(workflow, /verify-native-package\.mjs --input build\/desktop/)
  assert.match(workflow, /release-publication\.mjs validate-stable/)
  assert.match(workflow, /release-publication\.mjs collect/)
  assert.match(workflow, /Publish stable manifest last and verify R2 readback/)
  assert.match(workflow, /Refusing to overwrite an immutable object with different content/)
  assert.match(workflow, /Refusing to replace GitHub asset with different content/)
  assert.match(workflow, /- "!v\*-\*"/)
  assert.match(workflow, /- "!v\*\\\\\+\*"/)
})

test('CT-RELEASE-001 presents v0.1.0 as a user-facing bilingual release', async () => {
  const record = await fs.readFile(
    path.join(import.meta.dirname, '..', 'releases', 'v0.1.0.md'),
    'utf8',
  )

  const h1s = record.match(/^# .+$/gm) ?? []
  assert.deepEqual(h1s, ['# v0.1.0 Aivo 首个公开版本 / Initial Public Release'])
  assert.match(record, /^## 新增 \/ Highlights$/m)
  assert.match(record, /^## 下载 \/ Download$/m)
  assert.match(record, /\| 系统 System \| 芯片 Chip \| 格式 Format \| 下载 Download \|/)
  for (const asset of [
    'Aivo_0.1.0_windows-x86_64-setup.exe',
    'Aivo_0.1.0_darwin-aarch64.dmg',
    'Aivo_0.1.0_darwin-x86_64.dmg',
    'Aivo_0.1.0_linux-x86_64.AppImage',
    'Aivo_0.1.0_darwin-aarch64.zip',
    'Aivo_0.1.0_darwin-x86_64.zip',
    'SHA256SUMS',
  ]) {
    assert.match(record, new RegExp(`releases/download/v0\\.1\\.0/${asset.replaceAll('.', '\\.')}`))
  }
  assert.match(record, /macOS and Windows packages are unsigned/)
  assert.match(record, /source-available under a noncommercial license/)
  assert.ok(record.indexOf('## 下载 / Download') < record.indexOf('## 发布记录 / Release record'))
})
