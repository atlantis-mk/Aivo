const { contextBridge, ipcRenderer } = require('electron')

const runtimePorts = new Map()
let runtimePortSequence = 0

ipcRenderer.on('aivo:extension-runtime-port-message', (_event, payload) => {
  const state = runtimePorts.get(payload?.clientPortId)
  if (!state || state.disconnected) return
  for (const listener of state.messageListeners) {
    try {
      listener(payload.message, state.port)
    } catch {}
  }
})

ipcRenderer.on('aivo:extension-runtime-port-disconnected', (_event, payload) => {
  disconnectLocalRuntimePort(payload?.clientPortId, payload?.reason || 'host')
})

function connectRuntimePort(options) {
  const name = typeof options?.name === 'string' ? options.name.trim() : ''
  const clientPortId = `port-${Date.now().toString(36)}-${++runtimePortSequence}`
  const state = {
    clientPortId,
    disconnected: false,
    disconnectListeners: new Set(),
    messageListeners: new Set(),
    pendingMessages: [],
    port: null,
    ready: false,
  }
  const port = Object.freeze({
    name,
    postMessage(message) {
      if (state.disconnected) throw new Error('extension runtime Port is disconnected')
      if (!state.ready) {
        if (state.pendingMessages.length >= 32) throw new Error('extension runtime Port pending message limit exceeded')
        state.pendingMessages.push(message)
        return
      }
      postRuntimePortMessage(state, message)
    },
    disconnect() {
      if (state.disconnected) return
      void ipcRenderer.invoke('aivo:extension-runtime-close-port', { clientPortId }).finally(() => {
        disconnectLocalRuntimePort(clientPortId, 'guest')
      })
    },
    onMessage: listenerCollection(state.messageListeners),
    onDisconnect: listenerCollection(state.disconnectListeners),
  })
  state.port = port
  runtimePorts.set(clientPortId, state)
  void ipcRenderer.invoke('aivo:extension-runtime-open-port', { clientPortId, name }).then(() => {
    if (state.disconnected) {
      void ipcRenderer.invoke('aivo:extension-runtime-close-port', { clientPortId })
      return
    }
    state.ready = true
    for (const message of state.pendingMessages.splice(0)) postRuntimePortMessage(state, message)
  }).catch(() => {
    disconnectLocalRuntimePort(clientPortId, 'open-failed')
  })
  return port
}

function postRuntimePortMessage(state, message) {
  void ipcRenderer.invoke('aivo:extension-runtime-post-port-message', {
    clientPortId: state.clientPortId,
    message,
  }).catch(() => {
    disconnectLocalRuntimePort(state.clientPortId, 'post-failed')
  })
}

function listenerCollection(listeners) {
  return Object.freeze({
    addListener(listener) {
      if (typeof listener === 'function') listeners.add(listener)
    },
    removeListener(listener) {
      listeners.delete(listener)
    },
    hasListener(listener) {
      return listeners.has(listener)
    },
  })
}

function disconnectLocalRuntimePort(clientPortId, reason) {
  const state = runtimePorts.get(clientPortId)
  if (!state || state.disconnected) return
  state.disconnected = true
  state.pendingMessages.length = 0
  runtimePorts.delete(clientPortId)
  const event = Object.freeze({ reason })
  for (const listener of state.disconnectListeners) {
    try {
      listener(state.port, event)
    } catch {}
  }
  state.messageListeners.clear()
  state.disconnectListeners.clear()
}

contextBridge.exposeInMainWorld('aivoExtension', Object.freeze({
  version: 1,
  runtime: Object.freeze({
    sendMessage: (message) => ipcRenderer.invoke('aivo:extension-runtime-send-message', { message }),
    connect: connectRuntimePort,
  }),
  getContext: () => ipcRenderer.invoke('aivo:extension-view-context'),
  onContextChanged: (listener) => {
    if (typeof listener !== 'function') return () => {}
    const handler = (_event, payload) => listener(payload)
    ipcRenderer.on('aivo:extension-view-context-changed', handler)
    return () => ipcRenderer.removeListener('aivo:extension-view-context-changed', handler)
  },
  resize: (size) => ipcRenderer.invoke('aivo:extension-view-resize', size),
  close: () => ipcRenderer.invoke('aivo:extension-view-close'),
  notify: (notification) => ipcRenderer.invoke('aivo:extension-view-notify', notification),
  invokeAction: (action, data) => ipcRenderer.invoke('aivo:extension-view-action', { action, data }),
}))
