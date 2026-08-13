const { contextBridge, ipcRenderer, webUtils } = require('electron')

const CORE_URL = process.env.AIVO_CORE_URL || 'http://127.0.0.1:43117'
const MAX_COMPOSER_LOCAL_RESOURCES = 32

async function invoke(method, ...args) {
  const response = await fetch(`${CORE_URL}/api/rpc/${encodeURIComponent(method)}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ args }),
  })

  const payload = await response.json().catch(() => null)
  if (!response.ok) {
    throw new Error(payload?.error || `Aivo core RPC failed: ${method}`)
  }
  return payload
}

contextBridge.exposeInMainWorld('aivo', {
  platform: process.platform,
  versions: {
    chrome: process.versions.chrome,
    electron: process.versions.electron,
    node: process.versions.node,
  },
  coreUrl: CORE_URL,
  invoke,
  selectProjectDirectory: () => ipcRenderer.invoke('aivo:select-project-directory'),
  selectExtensionDirectory: () => ipcRenderer.invoke('aivo:select-extension-directory'),
  selectComposerFileOrDirectory: () => ipcRenderer.invoke('aivo:select-composer-file-or-directory'),
  inspectDroppedComposerResources: (files) => {
    if (!Array.isArray(files)) return Promise.resolve([])
    if (files.length > MAX_COMPOSER_LOCAL_RESOURCES) {
      return Promise.reject(new Error(`一次最多拖入 ${MAX_COMPOSER_LOCAL_RESOURCES} 个文件或文件夹。`))
    }
    const selectedPaths = files
      .map((file) => {
        try {
          return webUtils.getPathForFile(file)
        } catch {
          return ''
        }
      })
      .filter(Boolean)
    if (selectedPaths.length !== files.length) {
      return Promise.reject(new Error('无法读取一个或多个拖放资源。'))
    }
    return ipcRenderer.invoke('aivo:inspect-dropped-composer-resources', selectedPaths)
  },
  openExternal: (target) => ipcRenderer.invoke('aivo:open-external', target),
  openPath: (target) => ipcRenderer.invoke('aivo:open-path', target),
  focusWindow: () => ipcRenderer.invoke('aivo:focus-window'),
  toggleMaximize: () => ipcRenderer.invoke('aivo:toggle-maximize'),
  exportDiagnostics: () => ipcRenderer.invoke('aivo:export-diagnostics'),
  openExtensionView: (input) => ipcRenderer.invoke('aivo:open-extension-view', input),
  mountEmbeddedExtensionView: (input) => ipcRenderer.invoke('aivo:mount-embedded-extension-view', input),
  updateEmbeddedExtensionViewBounds: (input) => ipcRenderer.invoke('aivo:update-embedded-extension-view-bounds', input),
  updateEmbeddedExtensionViewContext: (input) => ipcRenderer.invoke('aivo:update-embedded-extension-view-context', input),
  closeEmbeddedExtensionView: (input) => ipcRenderer.invoke('aivo:close-embedded-extension-view', input),
  onEmbeddedExtensionViewClosed: (listener) => {
    if (typeof listener !== 'function') return () => {}
    const handler = (_event, payload) => listener(payload)
    ipcRenderer.on('aivo:embedded-extension-view-closed', handler)
    return () => ipcRenderer.removeListener('aivo:embedded-extension-view-closed', handler)
  },
})
