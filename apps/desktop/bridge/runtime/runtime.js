const listeners = new Map()
const registeredEventNames = new Set()
let eventSource
let sawEventSourceError = false

function ensureEventSource() {
  if (eventSource || !window.aivo?.coreUrl) return
  eventSource = new EventSource(`${window.aivo.coreUrl}/api/events`)
  eventSource.addEventListener('open', () => {
    if (sawEventSourceError) {
      dispatchLocal('events.reconnected', {})
    }
    sawEventSourceError = false
  })
  eventSource.addEventListener('error', () => {
    sawEventSourceError = true
  })
  for (const eventName of listeners.keys()) {
    registerEventSourceListener(eventName)
  }
}

function registerEventSourceListener(eventName) {
  if (!eventSource || registeredEventNames.has(eventName) || eventName === 'events.reconnected') return
  registeredEventNames.add(eventName)
  eventSource.addEventListener(eventName, dispatch)
}

function dispatch(event) {
  const callbacks = listeners.get(event.type)
  if (!callbacks) return
  let payload
  try {
    payload = JSON.parse(event.data)
  } catch {
    payload = event.data
  }
  for (const callback of [...callbacks]) {
    callback(payload)
  }
}

function dispatchLocal(eventName, payload) {
  const callbacks = listeners.get(eventName)
  if (!callbacks) return
  for (const callback of [...callbacks]) {
    callback(payload)
  }
}

export function EventsOnMultiple(eventName, callback, maxCallbacks) {
  const callbacks = listeners.get(eventName) ?? new Set()
  let count = 0
  const wrapped = (...args) => {
    count += 1
    callback(...args)
    if (maxCallbacks > 0 && count >= maxCallbacks) {
      EventsOff(eventName, wrapped)
    }
  }
  callbacks.add(wrapped)
  listeners.set(eventName, callbacks)
  ensureEventSource()
  registerEventSourceListener(eventName)
  return () => EventsOff(eventName, wrapped)
}

export function EventsOn(eventName, callback) {
  return EventsOnMultiple(eventName, callback, -1)
}

export function EventsOff(eventName, callback) {
  const callbacks = listeners.get(eventName)
  if (!callbacks) return
  if (callback) {
    callbacks.delete(callback)
  } else {
    callbacks.clear()
  }
  if (callbacks.size > 0) return
  listeners.delete(eventName)
  if (eventName !== 'events.reconnected') {
    registeredEventNames.delete(eventName)
    eventSource?.removeEventListener(eventName, dispatch)
  }
}

export function EventsOffAll() {
  for (const eventName of registeredEventNames) {
    eventSource?.removeEventListener(eventName, dispatch)
  }
  registeredEventNames.clear()
  listeners.clear()
}

export function EventsOnce(eventName, callback) {
  return EventsOnMultiple(eventName, callback, 1)
}

export function EventsEmit() {}
export function BrowserOpenURL(url) { return window.aivo?.openExternal(url) }
export function WindowToggleMaximise() { return window.aivo?.toggleMaximize() }
export function WindowMaximise() { return window.aivo?.toggleMaximize() }

export function LogPrint(message) { console.log(message) }
export function LogTrace(message) { console.trace(message) }
export function LogDebug(message) { console.debug(message) }
export function LogInfo(message) { console.info(message) }
export function LogWarning(message) { console.warn(message) }
export function LogError(message) { console.error(message) }
export function LogFatal(message) { console.error(message) }

export function WindowReload() { window.location.reload() }
export function WindowReloadApp() { window.location.reload() }
export function WindowSetAlwaysOnTop() {}
export function WindowSetSystemDefaultTheme() {}
export function WindowSetLightTheme() {}
export function WindowSetDarkTheme() {}
export function WindowCenter() {}
export function WindowSetTitle(title) { document.title = title }
export function WindowFullscreen() {}
export function WindowUnfullscreen() {}
export function WindowIsFullscreen() { return Promise.resolve(false) }
export function WindowGetSize() { return Promise.resolve({ w: window.innerWidth, h: window.innerHeight }) }
export function WindowSetSize() {}
export function WindowSetMaxSize() {}
export function WindowSetMinSize() {}
export function WindowSetPosition() {}
export function WindowGetPosition() { return Promise.resolve({ x: window.screenX, y: window.screenY }) }
export function WindowHide() {}
export function WindowShow() {}
export function WindowUnmaximise() {}
export function WindowIsMaximised() { return Promise.resolve(false) }
export function WindowMinimise() {}
export function WindowUnminimise() {}
export function WindowSetBackgroundColour() {}
export function ScreenGetAll() { return Promise.resolve([]) }
export function Environment() { return Promise.resolve({ platform: window.aivo?.platform }) }
export function Quit() {}
export function Hide() {}
export function Show() {}
export function ClipboardGetText() { return navigator.clipboard?.readText?.() ?? Promise.resolve("") }
export function ClipboardSetText(text) { return navigator.clipboard?.writeText?.(text) }
export function OnFileDrop() { return () => {} }
export function OnFileDropOff() {}
export function CanResolveFilePaths() { return false }
export function ResolveFilePaths(files) { return files }
export function InitializeNotifications() { return Promise.resolve() }
export function CleanupNotifications() { return Promise.resolve() }
export function IsNotificationAvailable() { return Promise.resolve(false) }
export function RequestNotificationAuthorization() { return Promise.resolve("denied") }
export function CheckNotificationAuthorization() { return Promise.resolve("denied") }
export function SendNotification() { return Promise.resolve() }
export function SendNotificationWithActions() { return Promise.resolve() }
export function RegisterNotificationCategory() { return Promise.resolve() }
export function RemoveNotificationCategory() { return Promise.resolve() }
export function RemoveAllPendingNotifications() { return Promise.resolve() }
export function RemovePendingNotification() { return Promise.resolve() }
export function RemoveAllDeliveredNotifications() { return Promise.resolve() }
export function RemoveDeliveredNotification() { return Promise.resolve() }
export function RemoveNotification() { return Promise.resolve() }
