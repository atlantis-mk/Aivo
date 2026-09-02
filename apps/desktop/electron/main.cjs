const { app, BrowserWindow, dialog, ipcMain, Menu, shell } = require('electron')
const { spawn } = require('node:child_process')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const { createDesktopUpdater } = require('./desktop-updater.cjs')
const { coreURLArgument, parseCoreReadyLine } = require('./core-endpoint.cjs')
const { createExtensionViewManager, registerExtensionScheme } = require('./extension-views.cjs')

registerExtensionScheme()

const isDev = Boolean(process.env.VITE_DEV_SERVER_URL)
const isMac = process.platform === 'darwin'
let coreUrl = process.env.AIVO_CORE_URL || 'http://127.0.0.1:43117'
const rendererBackgroundColor = '#ffffff'
const maxComposerAttachmentBytes = 50 * 1024 * 1024
const maxComposerLocalResources = 32
const newConversationWindowOffset = 28

app.setName('Aivo')

const hasSingleInstanceLock = app.requestSingleInstanceLock()
if (!hasSingleInstanceLock) {
  app.quit()
}

let coreProcess = null
let logFile = null
let extensionViewManager = null
let desktopUpdater = null
let startupWindow = null

function focusAivoWindow() {
  const window = BrowserWindow.getAllWindows()[0]
  if (!window || window.isDestroyed()) return
  if (window.isMinimized()) window.restore()
  window.show()
  window.focus()
}

if (hasSingleInstanceLock) {
  app.on('second-instance', () => {
    focusAivoWindow()
  })
}

function configureApplicationMenu() {
  if (!isMac) {
    Menu.setApplicationMenu(null)
    return
  }

  const menu = Menu.buildFromTemplate([
    {
      label: 'Aivo',
      submenu: [
        { role: 'about', label: '关于 Aivo' },
        { type: 'separator' },
        {
          label: '检查更新…',
          click: () => {
            const window = BrowserWindow.getFocusedWindow() || BrowserWindow.getAllWindows()[0]
            if (window && desktopUpdater) void checkAndOfferUpdate(window, true)
          },
        },
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

async function isCoreHealthy(target = coreUrl) {
  try {
    const response = await fetch(`${target}/health`, {
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

  const corePath = resolvePackagedCorePath()
  if (!fs.existsSync(corePath)) {
    throw new Error(`Packaged core binary was not found at ${corePath}`)
  }

  coreProcess = spawn(corePath, [], {
    stdio: ['ignore', 'pipe', 'pipe'],
    env: {
      ...process.env,
      AIVO_CORE_ADDR: '127.0.0.1:0',
      AIVO_CORE_READY_STDOUT: '1',
    },
  })

  let stdoutBuffer = ''
  let announcedCoreUrl = ''
  let readinessError = null
  coreProcess.once('error', (error) => {
    readinessError = error instanceof Error ? error : new Error(String(error))
    appendLog('error', 'core process error', { error: readinessError.message })
  })
  coreProcess.stdout.on('data', (chunk) => {
    const output = chunk.toString()
    appendLog('info', 'core stdout', { output: output.trimEnd() })
    stdoutBuffer += output
    if (stdoutBuffer.length > 8_192) {
      readinessError = new Error('Packaged Core readiness output exceeded its bound')
      return
    }
    let newline = stdoutBuffer.indexOf('\n')
    while (newline >= 0) {
      const line = stdoutBuffer.slice(0, newline).trimEnd()
      stdoutBuffer = stdoutBuffer.slice(newline + 1)
      try {
        const nextCoreUrl = parseCoreReadyLine(line)
        if (nextCoreUrl) {
          if (announcedCoreUrl) {
            throw new Error('Packaged Core emitted more than one readiness record')
          }
          announcedCoreUrl = nextCoreUrl
        }
      } catch (error) {
        readinessError = error instanceof Error ? error : new Error(String(error))
        if (coreProcess && coreProcess.exitCode === null && coreProcess.signalCode === null) coreProcess.kill()
      }
      newline = stdoutBuffer.indexOf('\n')
    }
  })
  coreProcess.stderr.on('data', (chunk) => appendLog('error', 'core stderr', { output: chunk.toString().trimEnd() }))
  coreProcess.once('exit', (code, signal) => {
    appendLog(code === 0 ? 'info' : 'error', 'core exited', { code, signal })
    coreProcess = null
  })

  for (let attempt = 0; attempt < 60; attempt += 1) {
    if (readinessError) {
      if (coreProcess && coreProcess.exitCode === null && coreProcess.signalCode === null) coreProcess.kill()
      throw readinessError
    }
    if (announcedCoreUrl && await isCoreHealthy(announcedCoreUrl)) {
      coreUrl = announcedCoreUrl
      appendLog('info', 'packaged core is healthy', { coreUrl })
      return
    }
    if (!coreProcess || coreProcess.exitCode !== null || coreProcess.signalCode !== null) {
      throw new Error('Packaged core exited before it became healthy')
    }
    await new Promise((resolve) => setTimeout(resolve, 250))
  }

  if (coreProcess && coreProcess.exitCode === null && coreProcess.signalCode === null) coreProcess.kill()
  throw new Error('Packaged Core did not announce a healthy dynamic loopback endpoint')
}

function createStartupWindow() {
  const startupWindow = new BrowserWindow({
    width: 1280,
    height: 860,
    minWidth: 960,
    minHeight: 640,
    show: false,
    backgroundColor: rendererBackgroundColor,
    title: 'Aivo',
    // Keep the native caption buttons on Windows and Linux. macOS still uses
    // its hidden title-bar treatment below.
    frame: true,
    ...(isMac
      ? {
          titleBarStyle: 'hidden',
          trafficLightPosition: { x: 10, y: 10 },
        }
      : {}),
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  })

  startupWindow.once('ready-to-show', () => {
    if (!startupWindow.isDestroyed()) startupWindow.show()
  })
  startupWindow.loadFile(path.join(__dirname, 'startup.html'))
  return startupWindow
}

function createWindow(initialRoute = '', position) {
  const mainWindow = new BrowserWindow({
    width: 1280,
    height: 860,
    minWidth: 960,
    minHeight: 640,
    show: false,
    backgroundColor: rendererBackgroundColor,
    ...(position ? { x: position.x, y: position.y } : {}),
    title: 'Aivo',
    // Keep the native caption buttons on Windows and Linux. macOS still uses
    // its hidden title-bar treatment below.
    frame: true,
    ...(isMac
      ? {
          titleBarStyle: 'hidden',
          trafficLightPosition: { x: 10, y: 10 },
        }
      : {}),
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      additionalArguments: app.isPackaged && !process.env.AIVO_CORE_URL
        ? [coreURLArgument(coreUrl)]
        : [],
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  })

  mainWindow.once('ready-to-show', () => {
    if (mainWindow.isDestroyed()) return
    if (startupWindow && !startupWindow.isDestroyed()) {
      mainWindow.setBounds(startupWindow.getBounds())
      mainWindow.show()
      startupWindow.destroy()
      startupWindow = null
      return
    }
    mainWindow.show()
  })

  if (isDev) {
    const rendererUrl = new URL(initialRoute || '/', process.env.VITE_DEV_SERVER_URL)
    mainWindow.loadURL(rendererUrl.toString())
  } else {
    const loadOptions = initialRoute ? { hash: initialRoute } : undefined
    mainWindow.loadFile(path.join(__dirname, '..', 'dist', 'index.html'), loadOptions)
  }

  mainWindow.webContents.on('render-process-gone', (_event, details) => {
    appendLog('error', 'renderer process gone', details)
  })
  mainWindow.webContents.on('did-fail-load', (_event, errorCode, errorDescription, validatedURL) => {
    appendLog('error', 'renderer failed to load', { errorCode, errorDescription, validatedURL })
  })
  return mainWindow
}

function requireMainRenderer(event) {
  if (!event.senderFrame || event.senderFrame !== event.sender.mainFrame) {
    throw new Error('Desktop capability is available only to the main Aivo renderer')
  }
}

ipcMain.handle('aivo:new-conversation-window', (event) => {
  requireMainRenderer(event)
  const sourceWindow = BrowserWindow.fromWebContents(event.sender)
  const sourcePosition = sourceWindow?.getPosition()
  createWindow(
    '/projects/chat',
    sourcePosition
      ? {
          x: sourcePosition[0] + newConversationWindowOffset,
          y: sourcePosition[1] + newConversationWindowOffset,
        }
      : undefined,
  )
})

ipcMain.handle('aivo:update:get-state', (event) => {
  requireMainRenderer(event)
  return desktopUpdater?.getState()
})

ipcMain.handle('aivo:update:check', (event) => {
  requireMainRenderer(event)
  return desktopUpdater?.check()
})

ipcMain.handle('aivo:update:download', (event) => {
  requireMainRenderer(event)
  return desktopUpdater?.download()
})

ipcMain.handle('aivo:update:install', (event) => {
  requireMainRenderer(event)
  return desktopUpdater?.install()
})

ipcMain.handle('aivo:update:cancel', (event) => {
  requireMainRenderer(event)
  return desktopUpdater?.cancel()
})

async function checkAndOfferUpdate(mainWindow, reportCurrent = false) {
  const state = await desktopUpdater.check()
  if (mainWindow.isDestroyed()) return
  if (state.phase !== 'available') {
    if (!reportCurrent) return
    const failed = state.phase === 'error'
    await dialog.showMessageBox(mainWindow, {
      type: failed ? 'error' : 'info',
      title: failed ? '无法检查 Aivo 更新' : 'Aivo 软件更新',
      message: state.phase === 'up-to-date'
        ? `Aivo v${state.currentVersion} 已是最新版本`
        : state.message,
      buttons: ['好'],
      defaultId: 0,
      noLink: true,
    })
    return
  }
  const offer = await dialog.showMessageBox(mainWindow, {
    type: 'info',
    title: 'Aivo 更新可用',
    message: `发现 Aivo v${state.availableVersion}`,
    detail: '更新包会同时核对 R2 与 GitHub Release，并在下载后验证 SHA-256。安装仍会显示操作系统安全提示。',
    buttons: ['下载更新', '稍后'],
    defaultId: 0,
    cancelId: 1,
    noLink: true,
  })
  if (offer.response !== 0) return
  const downloaded = await desktopUpdater.download()
  if (downloaded.phase !== 'ready' || mainWindow.isDestroyed()) return
  const ready = await dialog.showMessageBox(mainWindow, {
    type: 'info',
    title: 'Aivo 更新已验证',
    message: `Aivo v${downloaded.availableVersion} 已准备好`,
    detail: process.platform === 'linux'
      ? '将显示验证后的 AppImage，由你按当前安装方式完成替换。'
      : '将打开验证后的更新包；请按操作系统提示完成安装。',
    buttons: [process.platform === 'linux' ? '显示更新包' : '打开安装包', '稍后'],
    defaultId: 0,
    cancelId: 1,
    noLink: true,
  })
  if (ready.response === 0) await desktopUpdater.install()
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
  if (typeof target !== 'string') {
    throw new Error('外部链接无效。')
  }
  let url
  try {
    url = new URL(target)
  } catch {
    throw new Error('外部链接无效。')
  }
  if (!['http:', 'https:', 'mailto:'].includes(url.protocol)) {
    throw new Error('仅支持 HTTP、HTTPS 或邮件链接。')
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

if (hasSingleInstanceLock) {
  app.whenReady().then(async () => {
  configureApplicationMenu()
  initializeDiagnostics()
  startupWindow = createStartupWindow()
  try {
    await startPackagedCore()
  } catch (error) {
    if (startupWindow && !startupWindow.isDestroyed()) startupWindow.destroy()
    startupWindow = null
    const message = error instanceof Error ? error.message : String(error)
    appendLog('error', 'core startup failed', { error: message })
    dialog.showErrorBox(
      'Aivo core failed to start',
      `${message}\n\nAivo will close because the packaged desktop requires its owned local Core service.`,
    )
    app.quit()
    return
  }

  extensionViewManager = createExtensionViewManager({ ipcMain, coreUrl })
  desktopUpdater = createDesktopUpdater({
    appVersion: app.getVersion(),
    platform: process.platform,
    arch: process.arch,
    isPackaged: app.isPackaged,
    tempRoot: app.getPath('temp'),
    shell,
    onState: (state) => {
      for (const window of BrowserWindow.getAllWindows()) {
        if (!window.isDestroyed()) window.webContents.send('aivo:update:state', state)
      }
      appendLog(state.phase === 'error' ? 'error' : 'info', 'desktop update state', {
        phase: state.phase,
        currentVersion: state.currentVersion,
        availableVersion: state.availableVersion,
        errorCode: state.errorCode,
      })
    },
  })
  const mainWindow = createWindow()
  if (app.isPackaged) {
    mainWindow.webContents.once('did-finish-load', () => {
      setTimeout(() => {
        void checkAndOfferUpdate(mainWindow)
      }, 3_000)
    })
  }

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    }
  })
  })
}

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

app.on('before-quit', () => {
  desktopUpdater?.dispose()
  extensionViewManager?.closeAll()
  if (coreProcess && coreProcess.exitCode === null) {
    coreProcess.kill()
  }
})
