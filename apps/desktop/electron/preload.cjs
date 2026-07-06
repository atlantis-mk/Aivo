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
  browser: {
    getState: (tabId) => ipcRenderer.invoke('aivo:browser:get-state', tabId),
    setVisible: (tabId, visible) => ipcRenderer.invoke('aivo:browser:set-visible', tabId, visible),
    close: (tabId) => ipcRenderer.invoke('aivo:browser:close', tabId),
    setBounds: (tabId, bounds) => ipcRenderer.invoke('aivo:browser:set-bounds', tabId, bounds),
    navigate: (tabId, target) => ipcRenderer.invoke('aivo:browser:navigate', tabId, target),
    goBack: (tabId) => ipcRenderer.invoke('aivo:browser:go-back', tabId),
    goForward: (tabId) => ipcRenderer.invoke('aivo:browser:go-forward', tabId),
    reload: (tabId) => ipcRenderer.invoke('aivo:browser:reload', tabId),
    stop: (tabId) => ipcRenderer.invoke('aivo:browser:stop', tabId),
    loadFavicon: (favicons, pageURL) => ipcRenderer.invoke('aivo:browser:load-favicon', favicons, pageURL),
    onNavigateCurrent: (callback) => {
      const listener = (_event, payload) => {
        callback(payload)
      }
      ipcRenderer.on('aivo:browser-navigate-current', listener)
      return () => ipcRenderer.removeListener('aivo:browser-navigate-current', listener)
    },
    onStateChange: (callback, tabId) => {
      const listener = (_event, payload) => {
        const state = payload?.state || payload
        const nextTabId = payload?.tabId || ''
        if (tabId && nextTabId !== tabId) return
        callback(state, nextTabId)
      }
      ipcRenderer.on('aivo:browser-state', listener)
      return () => ipcRenderer.removeListener('aivo:browser-state', listener)
    },
    onOpenRequest: (callback) => {
      const listener = (_event, payload) => {
        Promise.resolve()
          .then(() => callback(payload))
          .then(() => {
            ipcRenderer.send('aivo:browser-open-response', {
              requestId: payload?.requestId,
              ok: true,
            })
          })
          .catch((error) => {
            ipcRenderer.send('aivo:browser-open-response', {
              requestId: payload?.requestId,
              ok: false,
              error: error instanceof Error ? error.message : String(error),
            })
          })
      }
      ipcRenderer.on('aivo:browser-open-request', listener)
      return () => ipcRenderer.removeListener('aivo:browser-open-request', listener)
    },
  },
})
