const { app, BrowserWindow, dialog, ipcMain, shell } = require('electron')
const { spawn } = require('node:child_process')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')

const isDev = Boolean(process.env.VITE_DEV_SERVER_URL)
const isMac = process.platform === 'darwin'
const coreUrl = process.env.AIVO_CORE_URL || 'http://127.0.0.1:43117'

let coreProcess = null
let logFile = null

function appendLog(level, message, metadata = {}) {
  const line = JSON.stringify({
    time: new Date().toISOString(),
    level,
    message,
    ...metadata,
  })

  if (logFile) {
    fs.appendFile(logFile, `${line}${os.EOL}`, () => {})
  }

  if (level === 'error') {
    console.error(line)
    return
  }
  console.log(line)
}

function initializeDiagnostics() {
  const logDir = path.join(app.getPath('userData'), 'logs')
  fs.mkdirSync(logDir, { recursive: true })
  logFile = path.join(logDir, 'main.log')
  appendLog('info', 'desktop starting', { version: app.getVersion(), isDev })

  process.on('uncaughtException', (error) => {
    appendLog('error', 'uncaught exception', { error: error.stack || error.message })
  })
  process.on('unhandledRejection', (reason) => {
    appendLog('error', 'unhandled rejection', {
      error: reason instanceof Error ? reason.stack || reason.message : String(reason),
    })
  })
}

async function isCoreHealthy() {
  try {
    const response = await fetch(`${coreUrl}/health`, {
      signal: AbortSignal.timeout(500),
    })
    return response.ok
  } catch {
    return false
  }
}

function resolvePackagedCorePath() {
  const executable = process.platform === 'win32' ? 'aivo-core.exe' : 'aivo-core'
  return path.join(process.resourcesPath, 'aivo-core', executable)
}

async function startPackagedCore() {
  if (isDev || process.env.AIVO_CORE_URL) {
    return
  }

  if (await isCoreHealthy()) {
    appendLog('info', 'using existing healthy core', { coreUrl })
    return
  }

  const corePath = resolvePackagedCorePath()
  if (!fs.existsSync(corePath)) {
    throw new Error(`Packaged core binary was not found at ${corePath}`)
  }

  coreProcess = spawn(corePath, [], {
    stdio: ['ignore', 'pipe', 'pipe'],
  })

  coreProcess.stdout.on('data', (chunk) => appendLog('info', 'core stdout', { output: chunk.toString().trimEnd() }))
  coreProcess.stderr.on('data', (chunk) => appendLog('error', 'core stderr', { output: chunk.toString().trimEnd() }))
  coreProcess.once('exit', (code, signal) => {
    appendLog(code === 0 ? 'info' : 'error', 'core exited', { code, signal })
    coreProcess = null
  })

  for (let attempt = 0; attempt < 60; attempt += 1) {
    if (await isCoreHealthy()) {
      appendLog('info', 'packaged core is healthy', { coreUrl })
      return
    }
    if (!coreProcess || coreProcess.exitCode !== null || coreProcess.signalCode !== null) {
      throw new Error('Packaged core exited before it became healthy')
    }
    await new Promise((resolve) => setTimeout(resolve, 250))
  }

  throw new Error(`Packaged core did not become healthy at ${coreUrl}`)
}

function createWindow() {
  const mainWindow = new BrowserWindow({
    width: 1280,
    height: 860,
    minWidth: 960,
    minHeight: 640,
    title: 'Aivo',
    frame: isMac,
    ...(isMac
      ? {
          titleBarStyle: 'hidden',
          trafficLightPosition: { x: 10, y: 10 },
        }
      : {}),
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  })

  if (isDev) {
    mainWindow.loadURL(process.env.VITE_DEV_SERVER_URL)
    mainWindow.webContents.openDevTools({ mode: 'detach' })
  } else {
    mainWindow.loadFile(path.join(__dirname, '..', 'dist', 'index.html'))
  }

  mainWindow.webContents.on('render-process-gone', (_event, details) => {
    appendLog('error', 'renderer process gone', details)
  })
  mainWindow.webContents.on('did-fail-load', (_event, errorCode, errorDescription, validatedURL) => {
    appendLog('error', 'renderer failed to load', { errorCode, errorDescription, validatedURL })
  })
}

ipcMain.handle('aivo:select-project-directory', async () => {
  const result = await dialog.showOpenDialog({
    properties: ['openDirectory'],
  })

  if (result.canceled || result.filePaths.length === 0) {
    return ''
  }

  return result.filePaths[0]
})

ipcMain.handle('aivo:open-external', async (_event, target) => {
  if (typeof target !== 'string' || target.length === 0) {
    return
  }
  await shell.openExternal(target)
})

ipcMain.handle('aivo:open-path', async (_event, target) => {
  if (typeof target !== 'string' || target.length === 0) {
    return
  }
  if (!path.isAbsolute(target)) {
    throw new Error('openPath target must be an absolute path')
  }
  const message = await shell.openPath(target)
  if (message) {
    throw new Error(message)
  }
})

ipcMain.handle('aivo:focus-window', (event) => {
  const window = BrowserWindow.fromWebContents(event.sender) || BrowserWindow.getAllWindows()[0]
  if (!window) return
  if (window.isMinimized()) {
    window.restore()
  }
  window.show()
  window.focus()
})

ipcMain.handle('aivo:export-diagnostics', async () => {
  const target = await dialog.showSaveDialog({
    title: 'Export Aivo Diagnostics',
    defaultPath: `aivo-diagnostics-${new Date().toISOString().replace(/[:.]/g, '-')}.json`,
    filters: [{ name: 'JSON', extensions: ['json'] }],
  })
  if (target.canceled || !target.filePath) {
    return ''
  }

  const payload = {
    time: new Date().toISOString(),
    appVersion: app.getVersion(),
    platform: process.platform,
    arch: process.arch,
    electron: process.versions.electron,
    chrome: process.versions.chrome,
    node: process.versions.node,
    coreUrl,
    coreRunning: Boolean(coreProcess && coreProcess.exitCode === null),
    logFile,
    recentMainLog: logFile && fs.existsSync(logFile) ? fs.readFileSync(logFile, 'utf8').slice(-100_000) : '',
  }
  fs.writeFileSync(target.filePath, JSON.stringify(payload, null, 2))
  return target.filePath
})

ipcMain.handle('aivo:toggle-maximize', (event) => {
  const window = BrowserWindow.fromWebContents(event.sender)
  if (!window) return
  if (window.isMaximized()) {
    window.unmaximize()
    return
  }
  window.maximize()
})

app.whenReady().then(async () => {
  initializeDiagnostics()
  try {
    await startPackagedCore()
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    appendLog('error', 'core startup failed', { error: message })
    dialog.showErrorBox(
      'Aivo core failed to start',
      `${message}\n\nThe desktop UI will open, but agent features require the local core service.`,
    )
  }

  createWindow()

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    }
  })
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

app.on('before-quit', () => {
  if (coreProcess && coreProcess.exitCode === null) {
    coreProcess.kill()
  }
})
