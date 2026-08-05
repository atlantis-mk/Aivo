const { contextBridge, ipcRenderer } = require('electron')

const CORE_URL = process.env.AIVO_CORE_URL || 'http://127.0.0.1:43117'

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
  openExternal: (target) => ipcRenderer.invoke('aivo:open-external', target),
  openPath: (target) => ipcRenderer.invoke('aivo:open-path', target),
  focusWindow: () => ipcRenderer.invoke('aivo:focus-window'),
  toggleMaximize: () => ipcRenderer.invoke('aivo:toggle-maximize'),
  exportDiagnostics: () => ipcRenderer.invoke('aivo:export-diagnostics'),
  openExtensionView: (input) => ipcRenderer.invoke('aivo:open-extension-view', input),
})
