const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('aivoExtension', Object.freeze({
  version: 1,
  getContext: () => ipcRenderer.invoke('aivo:extension-view-context'),
  resize: (size) => ipcRenderer.invoke('aivo:extension-view-resize', size),
  close: () => ipcRenderer.invoke('aivo:extension-view-close'),
  notify: (notification) => ipcRenderer.invoke('aivo:extension-view-notify', notification),
  invokeAction: (action, data) => ipcRenderer.invoke('aivo:extension-view-action', { action, data }),
}))
