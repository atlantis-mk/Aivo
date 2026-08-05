const { BrowserWindow, Notification, protocol, session } = require('electron')
const crypto = require('node:crypto')
const path = require('node:path')

const EXTENSION_ID = /^[a-z0-9][a-z0-9._-]*$/
const VIEW_ID = /^[A-Za-z0-9][A-Za-z0-9._-]*$/
const ALLOWED_SURFACES = new Set(['page', 'dialog', 'tool-detail', 'settings', 'notification'])
const DEFAULT_CSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

function registerExtensionScheme() {
  protocol.registerSchemesAsPrivileged([{
    scheme: 'aivo-extension',
    privileges: { standard: true, secure: true, supportFetchAPI: true, corsEnabled: false, allowServiceWorkers: false },
  }])
}

function createExtensionViewManager({ ipcMain, coreUrl }) {
  const backends = new Map()
  const views = new Map()

  protocol.handle('aivo-extension', async (request) => {
    const logical = new URL(request.url)
    const backend = backends.get(logical.hostname)
    if (!backend) return new Response('Extension view is unavailable', { status: 404, headers: { 'Content-Security-Policy': DEFAULT_CSP } })
	const status = await callCore(coreUrl, 'GetExtensionStatus', { id: logical.hostname }).catch(() => null)
	if (!status || (status.state !== 'ready' && status.state !== 'active')) {
	  backends.delete(logical.hostname)
	  return new Response('Extension view has stopped', { status: 410, headers: { 'Content-Security-Policy': DEFAULT_CSP } })
	}
    const target = new URL(`${logical.pathname}${logical.search}`, backend.origin)
	if (!['GET', 'HEAD', 'POST'].includes(request.method)) return new Response('Method not allowed', { status: 405, headers: { 'Content-Security-Policy': DEFAULT_CSP } })
    const headers = new Headers()
    for (const name of ['accept', 'accept-language', 'content-type']) {
      const value = request.headers.get(name)
      if (value) headers.set(name, value)
    }
    if (backend.token) headers.set('Authorization', `Bearer ${backend.token}`)
	let body
	if (request.method === 'POST') {
	  const declaredLength = Number(request.headers.get('content-length') || 0)
	  if (declaredLength > 1_048_576) return new Response('Request body is too large', { status: 413, headers: { 'Content-Security-Policy': DEFAULT_CSP } })
	  const rawBody = await request.arrayBuffer()
	  if (rawBody.byteLength > 1_048_576) return new Response('Request body is too large', { status: 413, headers: { 'Content-Security-Policy': DEFAULT_CSP } })
	  body = rawBody
	}
	const response = await fetch(target, {
      method: request.method,
      headers,
	  body,
      redirect: 'manual',
      signal: request.signal,
    })
    const responseHeaders = new Headers()
    for (const name of ['content-type', 'cache-control', 'etag', 'last-modified']) {
      const value = response.headers.get(name)
      if (value) responseHeaders.set(name, value)
    }
    responseHeaders.set('Content-Security-Policy', backend.csp || DEFAULT_CSP)
    responseHeaders.set('Cross-Origin-Resource-Policy', 'same-origin')
    responseHeaders.set('X-Content-Type-Options', 'nosniff')
    return new Response(response.body, { status: response.status, headers: responseHeaders })
  })

  ipcMain.handle('aivo:open-extension-view', async (_event, input) => {
    const request = validateOpenRequest(input)
    const descriptor = await callCore(coreUrl, 'ResolveExtensionView', { id: request.extensionId, viewId: request.viewId })
    validateDescriptor(request, descriptor)
    const logicalURL = new URL(descriptor.logicalUrl)
    const backendURL = new URL(descriptor.backendUrl)
    backends.set(request.extensionId, {
      origin: backendURL.origin,
      token: typeof descriptor.backendToken === 'string' ? descriptor.backendToken : '',
      csp: typeof descriptor.csp === 'string' && descriptor.csp.length <= 2_000 ? descriptor.csp : DEFAULT_CSP,
    })

    const partition = `aivo-extension-${request.extensionId}-${crypto.randomUUID()}`
    const isolatedSession = session.fromPartition(partition, { cache: false })
    isolatedSession.setPermissionCheckHandler(() => false)
    isolatedSession.setPermissionRequestHandler((_webContents, _permission, callback) => callback(false))
    isolatedSession.webRequest.onBeforeSendHeaders((details, callback) => {
      const headers = { ...details.requestHeaders }
      for (const name of Object.keys(headers)) {
        const lower = name.toLowerCase()
        if (lower === 'authorization' || lower === 'cookie' || lower.startsWith('sec-ch-ua')) delete headers[name]
      }
      callback({ requestHeaders: headers })
    })

    const window = new BrowserWindow({
      width: request.surface === 'page' ? 1100 : 760,
      height: request.surface === 'page' ? 760 : 620,
      minWidth: 360,
      minHeight: 240,
      title: typeof descriptor.title === 'string' ? descriptor.title : request.viewId,
      parent: BrowserWindow.getFocusedWindow() || undefined,
      modal: request.surface === 'dialog',
      show: false,
      webPreferences: {
        preload: path.join(__dirname, 'extension-preload.cjs'),
        partition,
        contextIsolation: true,
        nodeIntegration: false,
        sandbox: true,
        webSecurity: true,
        allowRunningInsecureContent: false,
        webviewTag: false,
      },
    })
    const allowedOrigin = logicalURL.origin
    window.webContents.setWindowOpenHandler(() => ({ action: 'deny' }))
    window.webContents.on('will-navigate', (event, target) => {
      try {
        if (new URL(target).origin !== allowedOrigin) event.preventDefault()
      } catch {
        event.preventDefault()
      }
    })
    window.webContents.on('will-attach-webview', (event) => event.preventDefault())
    const viewState = {
      window,
      extensionId: request.extensionId,
      viewId: request.viewId,
      surface: request.surface,
      actions: new Set(Array.isArray(descriptor.actions) ? descriptor.actions : []),
      context: boundedClone(request.context, 64_000),
    }
	await callCore(coreUrl, 'OpenExtensionView', { id: request.extensionId })
    views.set(window.webContents.id, viewState)
    window.on('closed', () => {
      views.delete(window.webContents.id)
	  callCore(coreUrl, 'CloseExtensionView', { id: request.extensionId }).catch(() => {})
	  if (![...views.values()].some((view) => view.extensionId === request.extensionId)) backends.delete(request.extensionId)
      isolatedSession.clearStorageData().catch(() => {})
    })
    window.once('ready-to-show', () => window.show())
	try {
	  await window.loadURL(descriptor.logicalUrl)
	} catch (error) {
	  window.destroy()
	  throw error
	}
    return { opened: true, extensionId: request.extensionId, viewId: request.viewId, surface: request.surface }
  })

  ipcMain.handle('aivo:extension-view-context', (event) => {
    const view = requireView(views, event.sender.id)
    return { version: 1, theme: 'system', locale: Intl.DateTimeFormat().resolvedOptions().locale, surface: view.surface, context: view.context }
  })
  ipcMain.handle('aivo:extension-view-resize', (event, input) => {
    const view = requireView(views, event.sender.id)
    const width = boundedInteger(input?.width, 320, 1600)
    const height = boundedInteger(input?.height, 200, 1200)
    view.window.setContentSize(width, height, true)
    return { width, height }
  })
  ipcMain.handle('aivo:extension-view-close', (event) => requireView(views, event.sender.id).window.close())
  ipcMain.handle('aivo:extension-view-notify', (event, input) => {
    const view = requireView(views, event.sender.id)
    const title = boundedText(input?.title, 100)
    const body = boundedText(input?.body, 500)
    if (!title && !body) throw new Error('notification title or body is required')
    if (Notification.isSupported()) new Notification({ title: title || view.extensionId, body }).show()
  })
  ipcMain.handle('aivo:extension-view-action', async (event, input) => {
    const view = requireView(views, event.sender.id)
    const action = boundedText(input?.action, 100)
    if (!view.actions.has(action)) throw new Error('extension action is not declared')
    return callCore(coreUrl, 'InvokeExtensionViewAction', {
      id: view.extensionId,
      viewId: view.viewId,
      action,
      data: boundedClone(input?.data, 64_000),
    })
  })

  return {
    closeAll() {
      for (const view of views.values()) view.window.destroy()
      views.clear()
      backends.clear()
    },
  }
}

function validateOpenRequest(input) {
  const extensionId = typeof input?.extensionId === 'string' ? input.extensionId.trim() : ''
  const viewId = typeof input?.viewId === 'string' ? input.viewId.trim() : ''
  const surface = typeof input?.surface === 'string' ? input.surface.trim() : ''
  if (!EXTENSION_ID.test(extensionId) || !VIEW_ID.test(viewId) || !ALLOWED_SURFACES.has(surface)) throw new Error('invalid extension view request')
  return { extensionId, viewId, surface, context: input?.context }
}

function validateDescriptor(request, descriptor) {
  if (!descriptor || descriptor.extensionId !== request.extensionId || descriptor.viewId !== request.viewId) throw new Error('extension view descriptor identity mismatch')
  if (!Array.isArray(descriptor.surfaces) || !descriptor.surfaces.includes(request.surface)) throw new Error('extension view does not support the requested surface')
  const logical = new URL(descriptor.logicalUrl)
  if (logical.protocol !== 'aivo-extension:' || logical.hostname !== request.extensionId) throw new Error('invalid extension logical URL')
  const backend = new URL(descriptor.backendUrl)
  const loopback = backend.hostname === '127.0.0.1' || backend.hostname === 'localhost' || backend.hostname === '::1'
  if (backend.protocol !== 'https:' && !(backend.protocol === 'http:' && loopback)) throw new Error('extension backend must use HTTPS or loopback HTTP')
}

async function callCore(coreUrl, method, input) {
  const response = await fetch(`${coreUrl}/api/rpc/${encodeURIComponent(method)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ args: [input] }),
    signal: AbortSignal.timeout(20_000),
  })
  const payload = await response.json().catch(() => null)
  if (!response.ok) throw new Error(payload?.error || `Core rejected ${method}`)
  return payload
}

function requireView(views, senderId) {
  const view = views.get(senderId)
  if (!view) throw new Error('extension view is not registered')
  return view
}

function boundedText(value, max) {
  return typeof value === 'string' ? value.trim().slice(0, max) : ''
}

function boundedInteger(value, min, max) {
  const number = Number.isFinite(value) ? Math.round(value) : min
  return Math.max(min, Math.min(max, number))
}

function boundedClone(value, maxBytes) {
  if (value === undefined) return null
  const raw = JSON.stringify(value)
  if (Buffer.byteLength(raw) > maxBytes) throw new Error('extension view data exceeds the bounded bridge limit')
  return JSON.parse(raw)
}

module.exports = { createExtensionViewManager, registerExtensionScheme }
