const { app, BrowserWindow, dialog, ipcMain, Menu, shell } = require('electron')
const { spawn } = require('node:child_process')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const { createExtensionViewManager, registerExtensionScheme } = require('./extension-views.cjs')

registerExtensionScheme()

const isDev = Boolean(process.env.VITE_DEV_SERVER_URL)
const isMac = process.platform === 'darwin'
const coreUrl = process.env.AIVO_CORE_URL || 'http://127.0.0.1:43117'
const maxComposerAttachmentBytes = 50 * 1024 * 1024
const maxComposerLocalResources = 32

app.setName('Aivo')

let coreProcess = null
let logFile = null
let extensionViewManager = null

function configureApplicationMenu() {
  if (!isMac) return

  const menu = Menu.buildFromTemplate([
    {
      label: 'Aivo',
      submenu: [
        { role: 'about', label: '关于 Aivo' },
        { type: 'separator' },
        { role: 'services', label: '服务' },
        { type: 'separator' },
        { role: 'hide', label: '隐藏 Aivo' },
        { role: 'hideOthers', label: '隐藏其他' },
        { role: 'unhide', label: '全部显示' },
        { type: 'separator' },
        { role: 'quit', label: '退出 Aivo' },
      ],
    },
    {
      label: '文件',
      submenu: [{ role: 'close', label: '关闭窗口' }],
    },
    {
      label: '编辑',
      submenu: [
        { role: 'undo', label: '撤销' },
        { role: 'redo', label: '重做' },
        { type: 'separator' },
        { role: 'cut', label: '剪切' },
        { role: 'copy', label: '复制' },
        { role: 'paste', label: '粘贴' },
        { role: 'delete', label: '删除' },
        { role: 'selectAll', label: '全选' },
      ],
    },
    {
      label: '视图',
      submenu: [
        { role: 'reload', label: '重新加载' },
        { role: 'forceReload', label: '强制重新加载' },
        { role: 'toggleDevTools', label: '切换开发者工具' },
        { type: 'separator' },
        { role: 'resetZoom', label: '实际大小' },
        { role: 'zoomIn', label: '放大' },
        { role: 'zoomOut', label: '缩小' },
        { type: 'separator' },
        { role: 'togglefullscreen', label: '切换全屏' },
      ],
    },
    {
      label: '窗口',
      submenu: [
        { role: 'minimize', label: '最小化' },
        { role: 'zoom', label: '缩放' },
        { type: 'separator' },
        { role: 'front', label: '全部置于最前面' },
      ],
    },
  ])
  Menu.setApplicationMenu(menu)
}

function composerAttachmentMimeType(filePath) {
  switch (path.extname(filePath).toLowerCase()) {
    case '.png': return 'image/png'
    case '.jpg':
    case '.jpeg': return 'image/jpeg'
    case '.gif': return 'image/gif'
    case '.webp': return 'image/webp'
    case '.pdf': return 'application/pdf'
    case '.docx': return 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
    case '.xlsx': return 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    case '.pptx': return 'application/vnd.openxmlformats-officedocument.presentationml.presentation'
    case '.txt':
    case '.md': return 'text/plain'
    case '.css': return 'text/css'
    case '.html':
    case '.htm': return 'text/html'
    case '.js':
    case '.jsx':
    case '.mjs':
    case '.cjs': return 'text/javascript'
    case '.ts':
    case '.tsx': return 'text/typescript'
    case '.json': return 'application/json'
    case '.csv': return 'text/csv'
    case '.xml': return 'application/xml'
    case '.yaml':
    case '.yml': return 'application/yaml'
    case '.toml': return 'application/toml'
    case '.c':
    case '.cc':
    case '.conf':
    case '.cpp':
    case '.cs':
    case '.env':
    case '.go':
    case '.h':
    case '.hpp':
    case '.ini':
    case '.java':
    case '.jsonl':
    case '.kt':
    case '.kts':
    case '.log':
    case '.lua':
    case '.php':
    case '.pl':
    case '.properties':
    case '.py':
    case '.rb':
    case '.rs':
    case '.sh':
    case '.sql':
    case '.swift':
    case '.vue': return 'text/plain'
    default: return 'application/octet-stream'
  }
}

async function composerLocalSelectionFromPath(selectedPath, byteBudget = maxComposerAttachmentBytes) {
  if (typeof selectedPath !== 'string' || !path.isAbsolute(selectedPath)) {
    throw new Error('本地资源路径无效。')
  }
  const selectedInfo = await fs.promises.lstat(selectedPath)
  if (selectedInfo.isSymbolicLink()) {
    throw new Error('不能将符号链接作为模型附件发送。')
  }
  if (selectedInfo.isDirectory()) {
    return { kind: 'directory', path: selectedPath }
  }
  if (!selectedInfo.isFile()) {
    throw new Error('只能选择普通文件或文件夹')
  }
  if (selectedInfo.size > byteBudget) {
    throw new Error(`${path.basename(selectedPath)} 使本次附件总大小超过 50 MB。`)
  }

  const handle = await fs.promises.open(selectedPath, 'r')
  try {
    const openedInfo = await handle.stat()
    if (!openedInfo.isFile()) {
      throw new Error('只能选择普通文件或文件夹')
    }
    if (openedInfo.size > byteBudget) {
      throw new Error(`${path.basename(selectedPath)} 使本次附件总大小超过 50 MB。`)
    }
    const data = await handle.readFile()
    if (data.byteLength > byteBudget) {
      throw new Error(`${path.basename(selectedPath)} 使本次附件总大小超过 50 MB。`)
    }
    return {
      kind: 'file',
      name: path.basename(selectedPath),
      mimeType: composerAttachmentMimeType(selectedPath),
      size: data.byteLength,
      data: data.toString('base64'),
    }
  } finally {
    await handle.close()
  }
}

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

ipcMain.handle('aivo:select-extension-directory', async () => {
  const result = await dialog.showOpenDialog({
    title: '选择 Aivo 扩展文件夹',
    buttonLabel: '选择扩展',
    properties: ['openDirectory'],
  })

  if (result.canceled || result.filePaths.length === 0) {
    return ''
  }

  return result.filePaths[0]
})

ipcMain.handle('aivo:select-composer-file-or-directory', async () => {
  const result = await dialog.showOpenDialog({
    title: '选择文件或文件夹',
    properties: ['openFile', 'openDirectory'],
  })

  if (result.canceled || result.filePaths.length === 0) {
    return null
  }

  return composerLocalSelectionFromPath(result.filePaths[0])
})

ipcMain.handle('aivo:inspect-dropped-composer-resources', async (_event, selectedPaths) => {
  if (!Array.isArray(selectedPaths)) {
    throw new Error('拖放资源列表无效。')
  }
  const uniquePaths = [...new Set(selectedPaths)]
  if (uniquePaths.length > maxComposerLocalResources) {
    throw new Error(`一次最多拖入 ${maxComposerLocalResources} 个文件或文件夹。`)
  }

  let remainingBytes = maxComposerAttachmentBytes
  const selections = []
  for (const selectedPath of uniquePaths) {
    const selection = await composerLocalSelectionFromPath(selectedPath, remainingBytes)
    selections.push(selection)
    if (selection.kind === 'file') {
      remainingBytes -= selection.size
    }
  }
  return selections
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
  configureApplicationMenu()
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

  extensionViewManager = createExtensionViewManager({ ipcMain, coreUrl })
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
  extensionViewManager?.closeAll()
  if (coreProcess && coreProcess.exitCode === null) {
    coreProcess.kill()
  }
})
