import { spawn } from 'node:child_process'
import fs from 'node:fs'
import { createRequire } from 'node:module'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const rootDir = process.cwd()
const require = createRequire(import.meta.url)
const { parseCoreReadyLine } = require('../apps/desktop/electron/core-endpoint.cjs')
const coreExecutable = process.platform === 'win32' ? 'aivo-core.exe' : 'aivo-core'
const corePath = path.join(rootDir, 'build', 'aivo-core', coreExecutable)
const desktopOutputDir = path.join(rootDir, 'build', 'desktop')
const expectInstallers = process.env.AIVO_EXPECT_INSTALLERS === '1'
let healthUrl = ''
let readinessError = null

export function macAppCandidates(outputDirectory) {
  return ['mac', 'mac-arm64', 'mac-x64'].map((directory) =>
    path.join(outputDirectory, directory, 'Aivo.app'),
  )
}

function assertFile(filePath, label) {
  if (!fs.existsSync(filePath)) {
    throw new Error(`${label} is missing at ${filePath}`)
  }
}

function assertAnyFile(candidates, label) {
  const found = candidates.find((candidate) => fs.existsSync(candidate))
  if (!found) {
    throw new Error(`${label} is missing; checked ${candidates.join(', ')}`)
  }
  return found
}

function assertArtifact(predicate, label) {
  if (!expectInstallers) {
    return
  }
  const entries = fs.existsSync(desktopOutputDir) ? fs.readdirSync(desktopOutputDir) : []
  const found = entries.find(predicate)
  if (!found) {
    throw new Error(`${label} is missing in ${desktopOutputDir}`)
  }
}

async function healthy(timeoutMs = 500) {
  try {
    const response = await fetch(healthUrl, {
      signal: AbortSignal.timeout(timeoutMs),
    })
    return response.ok
  } catch {
    return false
  }
}

async function waitForCore(child) {
  for (let attempt = 0; attempt < 60; attempt += 1) {
    if (readinessError) throw readinessError
    if (healthUrl && await healthy()) {
      return
    }
    if (child.exitCode !== null || child.signalCode !== null) {
      throw new Error('aivo-core exited before /health became ready')
    }
    await new Promise((resolve) => setTimeout(resolve, 250))
  }
  throw new Error(`${healthUrl} did not become healthy`)
}

async function main() {
  assertFile(path.join(rootDir, 'apps', 'desktop', 'dist', 'index.html'), 'desktop build')
  assertFile(path.join(rootDir, 'apps', 'desktop', 'electron', 'main.cjs'), 'electron main')
  assertFile(path.join(rootDir, 'apps', 'desktop', 'electron', 'preload.cjs'), 'electron preload')
  assertFile(corePath, 'core binary')

  const macApps = macAppCandidates(desktopOutputDir)
  const macResourceCores = macApps.map((app) =>
    path.join(app, 'Contents', 'Resources', 'aivo-core', coreExecutable),
  )
  const linuxResourceCore = path.join(desktopOutputDir, 'linux-unpacked', 'resources', 'aivo-core', coreExecutable)
  const winResourceCore = path.join(desktopOutputDir, 'win-unpacked', 'resources', 'aivo-core', coreExecutable)

  const macAppIndex = macApps.findIndex((candidate) => fs.existsSync(candidate))
  if (process.platform === 'darwin' && macAppIndex >= 0) {
    assertFile(macResourceCores[macAppIndex], 'packaged mac core binary')
    assertArtifact((entry) => entry.endsWith('.dmg'), 'mac DMG artifact')
    assertArtifact((entry) => entry.endsWith('-mac.zip'), 'mac zip artifact')
  } else if (process.platform === 'linux' && fs.existsSync(path.join(desktopOutputDir, 'linux-unpacked'))) {
    assertFile(linuxResourceCore, 'packaged linux core binary')
    assertArtifact((entry) => entry.endsWith('.AppImage'), 'linux AppImage artifact')
  } else if (process.platform === 'win32' && fs.existsSync(path.join(desktopOutputDir, 'win-unpacked'))) {
    assertFile(winResourceCore, 'packaged windows core binary')
    assertArtifact((entry) => entry.endsWith('.exe'), 'windows installer artifact')
  } else if (expectInstallers) {
    assertAnyFile([...macResourceCores, linuxResourceCore, winResourceCore], 'packaged app core binary')
  }

  const child = spawn(corePath, [], {
    stdio: ['ignore', 'pipe', 'pipe'],
    env: {
      ...process.env,
      AIVO_CORE_ADDR: '127.0.0.1:0',
      AIVO_CORE_READY_STDOUT: '1',
    },
  })
  let stdoutBuffer = ''
  child.stdout.on('data', (chunk) => {
    const output = chunk.toString()
    process.stdout.write(output)
    stdoutBuffer += output
    let newline = stdoutBuffer.indexOf('\n')
    while (newline >= 0) {
      const line = stdoutBuffer.slice(0, newline).trimEnd()
      stdoutBuffer = stdoutBuffer.slice(newline + 1)
      try {
        const readyUrl = parseCoreReadyLine(line)
        if (readyUrl) {
          if (healthUrl) throw new Error('aivo-core emitted more than one readiness record')
          healthUrl = `${readyUrl}/health`
        }
      } catch (error) {
        readinessError = error instanceof Error ? error : new Error(String(error))
      }
      newline = stdoutBuffer.indexOf('\n')
    }
  })
  child.stderr.on('data', (chunk) => process.stderr.write(chunk))

  try {
    await waitForCore(child)
    console.log('Release smoke passed: core binary started and /health is ready.')
  } finally {
    if (child.exitCode === null && child.signalCode === null) {
      child.kill()
    }
  }
}

if (fileURLToPath(import.meta.url) === path.resolve(process.argv[1] ?? '')) {
  main().catch((error) => {
    console.error(error instanceof Error ? error.message : String(error))
    process.exit(1)
  })
}
