import { createHash } from 'node:crypto'
import fs from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { pathToFileURL } from 'node:url'

const platformContracts = {
  'darwin-aarch64': [
    { sourceExtension: '.dmg', targetSuffix: '.dmg', contentType: 'application/x-apple-diskimage' },
    { sourceExtension: '.zip', targetSuffix: '.zip', contentType: 'application/zip' },
  ],
  'darwin-x86_64': [
    { sourceExtension: '.dmg', targetSuffix: '.dmg', contentType: 'application/x-apple-diskimage' },
    { sourceExtension: '.zip', targetSuffix: '.zip', contentType: 'application/zip' },
  ],
  'windows-x86_64': [
    { sourceExtension: '.exe', targetSuffix: '-setup.exe', contentType: 'application/vnd.microsoft.portable-executable' },
  ],
  'linux-x86_64': [
    { sourceExtension: '.AppImage', targetSuffix: '.AppImage', contentType: 'application/octet-stream' },
  ],
}

const semverPattern = /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/

function parseArguments(argv) {
  const [command, ...rest] = argv
  const values = new Map()
  for (let index = 0; index < rest.length; index += 2) {
    const key = rest[index]
    const value = rest[index + 1]
    if (!key?.startsWith('--') || value === undefined) {
      throw new Error(`Invalid argument sequence near ${key ?? '<end>'}`)
    }
    values.set(key.slice(2), value)
  }
  return { command, values }
}

function requireArgument(values, name) {
  const value = values.get(name)
  if (!value) {
    throw new Error(`Missing required --${name}`)
  }
  return value
}

function validateVersion(version) {
  if (!semverPattern.test(version)) {
    throw new Error(`Release version must be SemVer without a leading v: ${version}`)
  }
}

async function sha256(filePath) {
  const content = await fs.readFile(filePath)
  return createHash('sha256').update(content).digest('hex')
}

async function regularFiles(directory) {
  const entries = await fs.readdir(directory, { withFileTypes: true })
  return entries.filter((entry) => entry.isFile()).map((entry) => entry.name).sort()
}

export async function collectPlatformArtifacts({ input, output, platform, version }) {
  validateVersion(version)
  const contract = platformContracts[platform]
  if (!contract) {
    throw new Error(`Unsupported release platform: ${platform}`)
  }

  const files = await regularFiles(input)
  await fs.mkdir(output, { recursive: true })
  const collected = []

  for (const artifact of contract) {
    const matches = files.filter((name) => name.endsWith(artifact.sourceExtension))
    if (matches.length !== 1) {
      throw new Error(
        `${platform} requires exactly one ${artifact.sourceExtension} artifact, found ${matches.length}: ${matches.join(', ')}`,
      )
    }
    const targetName = `Aivo_${version}_${platform}${artifact.targetSuffix}`
    const sourcePath = path.join(input, matches[0])
    const targetPath = path.join(output, targetName)
    await fs.copyFile(sourcePath, targetPath)
    collected.push({
      name: targetName,
      contentType: artifact.contentType,
      sha256: await sha256(targetPath),
      size: (await fs.stat(targetPath)).size,
    })
  }

  await fs.writeFile(
    path.join(output, `${platform}.artifacts.json`),
    `${JSON.stringify({ platform, version, artifacts: collected }, null, 2)}\n`,
  )
  return collected
}

function normalizePrefix(prefix) {
  const normalized = prefix.replace(/^\/+|\/+$/g, '')
  if (!normalized || !/^[a-z0-9][a-z0-9/_-]*$/.test(normalized)) {
    throw new Error(`R2 prefix must be a lowercase path segment: ${prefix}`)
  }
  return normalized
}

export async function createPublication({ input, output, version, baseUrl, prefix = 'aivo', publishedAt }) {
  validateVersion(version)
  const normalizedPrefix = normalizePrefix(prefix)
  const normalizedBaseUrl = baseUrl.replace(/\/+$/g, '')
  if (!/^https:\/\//.test(normalizedBaseUrl)) {
    throw new Error('R2 public base URL must use https')
  }
  const timestamp = new Date(publishedAt)
  if (Number.isNaN(timestamp.valueOf())) {
    throw new Error(`Invalid publication timestamp: ${publishedAt}`)
  }

  const descriptors = []
  for (const platform of Object.keys(platformContracts)) {
    const descriptorPath = path.join(input, `${platform}.artifacts.json`)
    const descriptor = JSON.parse(await fs.readFile(descriptorPath, 'utf8'))
    if (descriptor.platform !== platform || descriptor.version !== version) {
      throw new Error(`Artifact descriptor identity mismatch: ${descriptorPath}`)
    }
    descriptors.push(descriptor)
  }

  const artifacts = descriptors.flatMap((descriptor) => descriptor.artifacts)
  const names = new Set()
  for (const artifact of artifacts) {
    if (names.has(artifact.name)) {
      throw new Error(`Duplicate normalized release artifact: ${artifact.name}`)
    }
    names.add(artifact.name)
    const artifactPath = path.join(input, artifact.name)
    const stat = await fs.stat(artifactPath)
    const digest = await sha256(artifactPath)
    if (stat.size !== artifact.size || digest !== artifact.sha256) {
      throw new Error(`Artifact changed after collection: ${artifact.name}`)
    }
  }

  await fs.mkdir(output, { recursive: true })
  const sortedArtifacts = artifacts.sort((left, right) => left.name.localeCompare(right.name))
  const checksumPath = path.join(output, 'SHA256SUMS')
  await fs.writeFile(
    checksumPath,
    `${sortedArtifacts.map((artifact) => `${artifact.sha256}  ${artifact.name}`).join('\n')}\n`,
  )
  const checksum = {
    name: 'SHA256SUMS',
    contentType: 'text/plain; charset=utf-8',
    sha256: await sha256(checksumPath),
    size: (await fs.stat(checksumPath)).size,
  }

  const releaseBaseUrl = `${normalizedBaseUrl}/${normalizedPrefix}/releases/v${version}`
  const latest = {
    version,
    tag: `v${version}`,
    publishedAt: timestamp.toISOString(),
    releaseBaseUrl,
    artifacts: [...sortedArtifacts, checksum].map((artifact) => ({
      name: artifact.name,
      url: `${releaseBaseUrl}/${artifact.name}`,
      sha256: artifact.sha256,
      size: artifact.size,
    })),
  }
  const latestPath = path.join(output, 'latest.json')
  await fs.writeFile(latestPath, `${JSON.stringify(latest, null, 2)}\n`)

  const immutableRows = [...sortedArtifacts, checksum].map((artifact) => {
    const source = artifact.name === checksum.name ? checksumPath : path.join(input, artifact.name)
    const key = `${normalizedPrefix}/releases/v${version}/${artifact.name}`
    return [source, key, artifact.contentType, artifact.sha256].join('\t')
  })
  await fs.writeFile(path.join(output, 'immutable.tsv'), `${immutableRows.join('\n')}\n`)
  return { artifacts: sortedArtifacts, checksum, latest, latestPath }
}

export async function validateReleaseSource({ root, version, tag }) {
  validateVersion(version)
  if (tag !== `v${version}`) {
    throw new Error(`Tag ${tag} does not match package version ${version}`)
  }
  const rootManifest = JSON.parse(await fs.readFile(path.join(root, 'package.json'), 'utf8'))
  const desktopManifest = JSON.parse(await fs.readFile(path.join(root, 'apps/desktop/package.json'), 'utf8'))
  if (rootManifest.version !== version || desktopManifest.version !== version) {
    throw new Error(
      `Release version ${version} must match root (${rootManifest.version}) and desktop (${desktopManifest.version}) manifests`,
    )
  }
  const releaseRecord = path.join(root, 'releases', `${tag}.md`)
  await fs.access(releaseRecord)
  return releaseRecord
}

async function main() {
  const { command, values } = parseArguments(process.argv.slice(2))
  if (command === 'collect') {
    await collectPlatformArtifacts({
      input: requireArgument(values, 'input'),
      output: requireArgument(values, 'output'),
      platform: requireArgument(values, 'platform'),
      version: requireArgument(values, 'version'),
    })
    return
  }
  if (command === 'plan') {
    await createPublication({
      input: requireArgument(values, 'input'),
      output: requireArgument(values, 'output'),
      version: requireArgument(values, 'version'),
      baseUrl: requireArgument(values, 'base-url'),
      prefix: values.get('prefix') ?? 'aivo',
      publishedAt: requireArgument(values, 'published-at'),
    })
    return
  }
  if (command === 'validate') {
    await validateReleaseSource({
      root: requireArgument(values, 'root'),
      version: requireArgument(values, 'version'),
      tag: requireArgument(values, 'tag'),
    })
    return
  }
  throw new Error(`Unknown release publication command: ${command ?? '<none>'}`)
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((error) => {
    console.error(error instanceof Error ? error.message : String(error))
    process.exit(1)
  })
}
