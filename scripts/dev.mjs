import { spawn } from 'node:child_process'
import process from 'node:process'

const rootDir = process.cwd()
const npmCommand = process.platform === 'win32' ? 'npm.cmd' : 'npm'
const coreHealthUrl = 'http://127.0.0.1:43117/health'
const children = new Set()

function start(command, args, options = {}) {
  const child = spawn(command, args, {
    cwd: rootDir,
    stdio: 'inherit',
    shell: false,
    ...options,
  })

  children.add(child)
  child.once('exit', () => children.delete(child))
  return child
}

function stopAll(signal = 'SIGTERM') {
  for (const child of children) {
    if (!child.killed) {
      child.kill(signal)
    }
  }
}

async function waitForCore() {
  for (let attempt = 0; attempt < 60; attempt += 1) {
    if (core.exitCode !== null || core.signalCode !== null) {
      throw new Error('Go core exited before becoming healthy; port 43117 may already be in use by a stale process.')
    }

    try {
      const response = await fetch(coreHealthUrl)
      if (response.ok) {
        return
      }
    } catch {
      // Core is still starting.
    }

    await new Promise((resolve) => setTimeout(resolve, 250))
  }

  throw new Error(`Go core did not become healthy at ${coreHealthUrl}`)
}

async function assertCorePortFree() {
  try {
    const response = await fetch(coreHealthUrl, {
      signal: AbortSignal.timeout(300),
    })
    if (response.ok) {
      throw new Error(
        `Port 43117 already has an aivo-core process. Stop the stale process before running npm run dev.`,
      )
    }
  } catch (error) {
    if (error instanceof Error && error.message.includes('already has an aivo-core')) {
      throw error
    }
  }
}

process.once('SIGINT', () => {
  stopAll('SIGINT')
})

process.once('SIGTERM', () => {
  stopAll('SIGTERM')
})

try {
  await assertCorePortFree()
} catch (error) {
  console.error(error.message)
  process.exit(1)
}

const core = start('go', ['run', './cmd/aivo-core'], {
  cwd: `${rootDir}/core`,
  env: {
    ...process.env,
    AIVO_BROWSER_BRIDGE_URL: 'http://127.0.0.1:43118',
  },
})

try {
  await waitForCore()
} catch (error) {
  console.error(error.message)
  stopAll()
  process.exit(1)
}

const desktop = start(npmCommand, ['--workspace', '@aivo/desktop', 'run', 'dev'])

desktop.once('exit', (code, signal) => {
  stopAll()
  if (signal) {
    process.kill(process.pid, signal)
    return
  }

  process.exit(code ?? 0)
})
