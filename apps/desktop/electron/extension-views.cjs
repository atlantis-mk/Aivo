const { BrowserWindow, Notification, WebContentsView, protocol, session } = require('electron')
const crypto = require('node:crypto')
const path = require('node:path')

const EXTENSION_ID = /^[a-z0-9][a-z0-9._-]*$/
const VIEW_ID = /^[A-Za-z0-9][A-Za-z0-9._-]*$/
const ALLOWED_SURFACES = new Set(['page', 'dialog', 'tool-detail', 'settings', 'notification'])
const DEFAULT_CSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"
const MAX_RUNTIME_MESSAGE_BYTES = 64 * 1024
const MAX_RUNTIME_PORTS_PER_VIEW = 8
const MAX_RUNTIME_PORT_EVENTS_PER_SECOND = 256
const RUNTIME_REQUEST_TIMEOUT_MS = 10_000

function registerExtensionScheme() {
  protocol.registerSchemesAsPrivileged([{
    scheme: 'aivo-extension',
    privileges: { standard: true, secure: true, supportFetchAPI: true, corsEnabled: false, allowServiceWorkers: false },
  }])
}

function createExtensionViewManager({ ipcMain, coreUrl }) {
  const backends = new Map()
  const views = new Map()
  const embeddedByOwner = new Map()
  const embeddedMountRequests = new Map()
  const embeddedMountGenerations = new Map()

  const handleExtensionRequest = async (request) => {
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
  }
  protocol.handle('aivo-extension', handleExtensionRequest)

  ipcMain.handle('aivo:open-extension-view', async (_event, input) => {
    const request = validateOpenRequest(input)
    const context = boundedClone(request.context, 64_000)
    const descriptor = await callCore(coreUrl, 'ResolveExtensionView', { id: request.extensionId, viewId: request.viewId })
    validateDescriptor(request, descriptor)
    const logicalURL = new URL(descriptor.logicalUrl)
    const backendURL = new URL(descriptor.backendUrl)
    backends.set(request.extensionId, {
      origin: backendURL.origin,
      token: typeof descriptor.backendToken === 'string' ? descriptor.backendToken : '',
      csp: typeof descriptor.csp === 'string' && descriptor.csp.length <= 2_000 ? descriptor.csp : DEFAULT_CSP,
    })

	const { partition, isolatedSession } = createIsolatedExtensionSession(request.extensionId, handleExtensionRequest)

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
		...extensionWebPreferences(partition),
      },
    })
	secureExtensionWebContents(window.webContents, logicalURL.origin)
    const viewState = {
      window,
      extensionId: request.extensionId,
      viewId: request.viewId,
      surface: request.surface,
      actions: new Set(Array.isArray(descriptor.actions) ? descriptor.actions : []),
		context,
		contextRevision: 1,
		runtimeMessaging: Array.isArray(descriptor.permissions) && descriptor.permissions.includes('runtime.messaging'),
		runtimePorts: new Map(),
		isolatedSession,
	}
	try {
	  await callCore(coreUrl, 'OpenExtensionView', { id: request.extensionId })
	} catch (error) {
	  window.destroy()
	  disposeIsolatedExtensionSession(isolatedSession)
	  if (![...views.values()].some((view) => view.extensionId === request.extensionId)) backends.delete(request.extensionId)
	  throw error
	}
    views.set(window.webContents.id, viewState)
    window.on('closed', () => {
      views.delete(window.webContents.id)
	  closeAllRuntimePorts(viewState, 'view-closed', false)
	  callCore(coreUrl, 'CloseExtensionView', { id: request.extensionId }).catch(() => {})
	  if (![...views.values()].some((view) => view.extensionId === request.extensionId)) backends.delete(request.extensionId)
      disposeIsolatedExtensionSession(isolatedSession)
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

  ipcMain.handle('aivo:mount-embedded-extension-view', async (event, input) => {
	const request = validateOpenRequest(input)
	const context = boundedClone(request.context, 64_000)
	if (request.surface !== 'tool-detail' && request.surface !== 'page') throw new Error('only tool detail or page views can be embedded')
	const owner = BrowserWindow.fromWebContents(event.sender)
	if (!owner || owner.isDestroyed()) throw new Error('extension view owner is unavailable')
	const requestId = boundedIdentifier(input?.requestId, 100)
	if (!requestId) throw new Error('embedded extension view requestId is required')
	const generation = (embeddedMountGenerations.get(owner.id) || 0) + 1
	embeddedMountGenerations.set(owner.id, generation)
	embeddedMountRequests.set(owner.id, { generation, requestId })
	const previous = embeddedByOwner.get(owner.id)
	if (previous) previous.close('replaced')

	const descriptor = await callCore(coreUrl, 'ResolveExtensionView', { id: request.extensionId, viewId: request.viewId })
	if (owner.isDestroyed() || embeddedMountGenerations.get(owner.id) !== generation) throw new Error('embedded extension view request was superseded')
	validateDescriptor(request, descriptor)
	const logicalURL = new URL(descriptor.logicalUrl)
	const backendURL = new URL(descriptor.backendUrl)
	backends.set(request.extensionId, {
	  origin: backendURL.origin,
	  token: typeof descriptor.backendToken === 'string' ? descriptor.backendToken : '',
	  csp: typeof descriptor.csp === 'string' && descriptor.csp.length <= 2_000 ? descriptor.csp : DEFAULT_CSP,
	})

	const { partition, isolatedSession } = createIsolatedExtensionSession(request.extensionId, handleExtensionRequest)
	const container = new WebContentsView({ webPreferences: extensionWebPreferences(partition) })
	container.setBackgroundColor('#00000000')
	secureExtensionWebContents(container.webContents, logicalURL.origin)
	const mountId = crypto.randomUUID()
	const viewState = {
	  mountId,
	  owner,
	  container,
	  isolatedSession,
	  extensionId: request.extensionId,
	  viewId: request.viewId,
	  surface: request.surface,
	  actions: new Set(Array.isArray(descriptor.actions) ? descriptor.actions : []),
	  context,
	  contextRevision: 1,
	  runtimeMessaging: Array.isArray(descriptor.permissions) && descriptor.permissions.includes('runtime.messaging'),
	  runtimePorts: new Map(),
	  closed: false,
	  close: null,
	}
	const ownerClosed = () => viewState.close('owner-closed')
	container.webContents.on('render-process-gone', () => viewState.close('render-process-gone'))
	container.webContents.on('unresponsive', () => viewState.close('unresponsive'))
	viewState.close = (reason = 'host') => {
	  if (viewState.closed) return
	  viewState.closed = true
	  closeAllRuntimePorts(viewState, reason, false)
	  views.delete(container.webContents.id)
	  if (embeddedByOwner.get(owner.id) === viewState) embeddedByOwner.delete(owner.id)
	  owner.removeListener('closed', ownerClosed)
	  if (!owner.isDestroyed()) owner.contentView.removeChildView(container)
	  if (!container.webContents.isDestroyed()) container.webContents.close({ waitForBeforeUnload: false })
	  callCore(coreUrl, 'CloseExtensionView', { id: request.extensionId }).catch(() => {})
	  if (![...views.values()].some((view) => view.extensionId === request.extensionId)) backends.delete(request.extensionId)
	  disposeIsolatedExtensionSession(isolatedSession)
	  if (!owner.isDestroyed()) owner.webContents.send('aivo:embedded-extension-view-closed', { mountId, reason })
	}

	try {
	  await callCore(coreUrl, 'OpenExtensionView', { id: request.extensionId })
	} catch (error) {
	  if (embeddedMountRequests.get(owner.id)?.generation === generation) embeddedMountRequests.delete(owner.id)
	  if (!container.webContents.isDestroyed()) container.webContents.close({ waitForBeforeUnload: false })
	  disposeIsolatedExtensionSession(isolatedSession)
	  if (![...views.values()].some((view) => view.extensionId === request.extensionId)) backends.delete(request.extensionId)
	  throw error
	}
	if (owner.isDestroyed() || embeddedMountGenerations.get(owner.id) !== generation) {
	  if (!container.webContents.isDestroyed()) container.webContents.close({ waitForBeforeUnload: false })
	  disposeIsolatedExtensionSession(isolatedSession)
	  callCore(coreUrl, 'CloseExtensionView', { id: request.extensionId }).catch(() => {})
	  if (![...views.values()].some((view) => view.extensionId === request.extensionId)) backends.delete(request.extensionId)
	  throw new Error('embedded extension view request was superseded')
	}
	views.set(container.webContents.id, viewState)
	embeddedByOwner.set(owner.id, viewState)
	embeddedMountRequests.delete(owner.id)
	owner.once('closed', ownerClosed)
	try {
	  owner.contentView.addChildView(container)
	  container.setBounds(boundedViewBounds(input?.bounds, owner))
	  void container.webContents.loadURL(descriptor.logicalUrl).catch(() => viewState.close('load-failed'))
	} catch (error) {
	  viewState.close('load-failed')
	  throw error
	}
	return { mounted: true, mountId, extensionId: request.extensionId, viewId: request.viewId, surface: request.surface }
  })

  ipcMain.handle('aivo:update-embedded-extension-view-bounds', (event, input) => {
	const owner = BrowserWindow.fromWebContents(event.sender)
	const view = owner ? embeddedByOwner.get(owner.id) : null
	if (!view || view.closed || view.mountId !== input?.mountId) return { updated: false }
	const bounds = boundedViewBounds(input?.bounds, owner)
	view.container.setBounds(bounds)
	return { updated: true, bounds }
  })

  ipcMain.handle('aivo:update-embedded-extension-view-context', (event, input) => {
	const owner = BrowserWindow.fromWebContents(event.sender)
	const view = owner ? embeddedByOwner.get(owner.id) : null
	if (!view || view.closed || view.mountId !== input?.mountId) return { updated: false }
	view.context = boundedClone(input?.context, 64_000)
	view.contextRevision += 1
	const payload = extensionViewContextPayload(view)
	if (!view.container.webContents.isDestroyed()) {
	  view.container.webContents.send('aivo:extension-view-context-changed', payload)
	}
	return { updated: true, revision: view.contextRevision }
  })

  ipcMain.handle('aivo:close-embedded-extension-view', (event, input) => {
	const owner = BrowserWindow.fromWebContents(event.sender)
	const pending = owner ? embeddedMountRequests.get(owner.id) : null
	if (pending && pending.requestId === input?.requestId) {
	  const generation = (embeddedMountGenerations.get(owner.id) || pending.generation) + 1
	  embeddedMountGenerations.set(owner.id, generation)
	  embeddedMountRequests.delete(owner.id)
	}
	const view = owner ? embeddedByOwner.get(owner.id) : null
	if (!view || view.mountId !== input?.mountId) return { closed: Boolean(pending && pending.requestId === input?.requestId) }
	view.close('host')
	return { closed: true }
  })

  ipcMain.handle('aivo:extension-view-context', (event) => {
    const view = requireView(views, event.sender.id)
    return extensionViewContextPayload(view)
  })
  ipcMain.handle('aivo:extension-view-resize', (event, input) => {
	const view = requireView(views, event.sender.id)
	const width = boundedInteger(input?.width, 320, 1600)
	const height = boundedInteger(input?.height, 200, 1200)
	if (view.window) {
	  view.window.setContentSize(width, height, true)
	  return { width, height }
	}
	const bounds = view.container.getBounds()
	return { width: bounds.width, height: bounds.height }
  })
  ipcMain.handle('aivo:extension-view-close', (event) => {
	const view = requireView(views, event.sender.id)
	if (view.window) return view.window.close()
	view.close('guest')
  })
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

  ipcMain.handle('aivo:extension-runtime-send-message', async (event, input) => {
	const view = requireRuntimeMessagingView(views, event.sender.id)
	const backend = await requireCurrentRuntimeBackend(coreUrl, backends, view)
	const message = boundedClone(input?.message, MAX_RUNTIME_MESSAGE_BYTES)
	const response = await fetch(runtimeMessageURL(backend), {
	  method: 'POST',
	  headers: runtimeHeaders(backend, { 'Content-Type': 'application/json', 'X-Aivo-View-ID': view.viewId }),
	  body: JSON.stringify({ message }),
	  redirect: 'manual',
	  signal: AbortSignal.timeout(RUNTIME_REQUEST_TIMEOUT_MS),
	})
	return readRuntimeJSONResponse(response)
  })

  ipcMain.handle('aivo:extension-runtime-open-port', async (event, input) => {
	const view = requireRuntimeMessagingView(views, event.sender.id)
	const clientPortId = boundedIdentifier(input?.clientPortId, 100)
	const name = boundedIdentifier(input?.name, 100)
	if (!clientPortId || !name) throw new Error('extension runtime Port id and name are required')
	if (view.runtimePorts.has(clientPortId)) throw new Error('extension runtime Port already exists')
	if (view.runtimePorts.size >= MAX_RUNTIME_PORTS_PER_VIEW) throw new Error('extension runtime Port limit exceeded')
	const backend = await requireCurrentRuntimeBackend(coreUrl, backends, view)
	const port = {
	  abortController: new AbortController(),
	  backend,
	  clientPortId,
	  closed: false,
	  eventCount: 0,
	  eventWindowStartedAt: Date.now(),
	  name,
	  portId: crypto.randomUUID(),
	}
	view.runtimePorts.set(clientPortId, port)
	const connectTimer = setTimeout(() => port.abortController.abort(), RUNTIME_REQUEST_TIMEOUT_MS)
	try {
	  const response = await fetch(runtimePortURL(backend, port.portId), {
		method: 'GET',
		headers: runtimeHeaders(backend, {
		  Accept: 'application/x-ndjson',
		  'X-Aivo-Port-Name': name,
		  'X-Aivo-View-ID': view.viewId,
		}),
		redirect: 'manual',
		signal: port.abortController.signal,
	  })
	  clearTimeout(connectTimer)
	  if (!response.ok || !response.body) throw new Error(`extension runtime Port failed with HTTP ${response.status}`)
	  void consumeRuntimePort(view, port, response.body)
	  return { opened: true, clientPortId }
	} catch (error) {
	  clearTimeout(connectTimer)
	  disconnectRuntimePort(view, port, 'open-failed', true, false)
	  throw error
	}
  })

  ipcMain.handle('aivo:extension-runtime-post-port-message', async (event, input) => {
	const view = requireRuntimeMessagingView(views, event.sender.id)
	const port = view.runtimePorts.get(input?.clientPortId)
	if (!port || port.closed) return { posted: false }
	const message = boundedClone(input?.message, MAX_RUNTIME_MESSAGE_BYTES)
	try {
	  const response = await fetch(`${runtimePortURL(port.backend, port.portId)}/messages`, {
		method: 'POST',
		headers: runtimeHeaders(port.backend, { 'Content-Type': 'application/json', 'X-Aivo-View-ID': view.viewId }),
		body: JSON.stringify({ message }),
		redirect: 'manual',
		signal: AbortSignal.timeout(RUNTIME_REQUEST_TIMEOUT_MS),
	  })
	  if (!response.ok) throw new Error(`extension runtime Port message failed with HTTP ${response.status}`)
	  await discardBoundedRuntimeResponse(response)
	  return { posted: true }
	} catch (error) {
	  disconnectRuntimePort(view, port, 'post-failed', true, true)
	  throw error
	}
  })

  ipcMain.handle('aivo:extension-runtime-close-port', (event, input) => {
	const view = requireRuntimeMessagingView(views, event.sender.id)
	const port = view.runtimePorts.get(input?.clientPortId)
	if (!port || port.closed) return { closed: false }
	disconnectRuntimePort(view, port, 'guest', true, true)
	return { closed: true }
  })

  async function consumeRuntimePort(view, port, body) {
	const reader = body.getReader()
	const decoder = new TextDecoder()
	let pending = ''
	try {
	  while (!port.closed) {
		const { done, value } = await reader.read()
		if (done) break
		pending += decoder.decode(value, { stream: true })
		let newline = pending.indexOf('\n')
		while (newline >= 0) {
		  const line = pending.slice(0, newline).trim()
		  pending = pending.slice(newline + 1)
		  if (line) deliverRuntimePortLine(view, port, line)
		  newline = pending.indexOf('\n')
		}
		if (Buffer.byteLength(pending) > MAX_RUNTIME_MESSAGE_BYTES) throw new Error('extension runtime Port event exceeds limit')
	  }
	  pending += decoder.decode()
	  if (pending.trim()) deliverRuntimePortLine(view, port, pending.trim())
	  disconnectRuntimePort(view, port, 'eof', true, false)
	} catch (error) {
	  if (!port.closed) disconnectRuntimePort(view, port, error?.name === 'AbortError' ? 'cancelled' : 'stream-failed', true, false)
	} finally {
	  reader.cancel().catch(() => {})
	}
  }

  function deliverRuntimePortLine(view, port, line) {
	if (Buffer.byteLength(line) > MAX_RUNTIME_MESSAGE_BYTES) throw new Error('extension runtime Port event exceeds limit')
	const now = Date.now()
	if (now - port.eventWindowStartedAt >= 1_000) {
	  port.eventWindowStartedAt = now
	  port.eventCount = 0
	}
	port.eventCount += 1
	if (port.eventCount > MAX_RUNTIME_PORT_EVENTS_PER_SECOND) throw new Error('extension runtime Port event rate exceeds limit')
	const message = boundedClone(JSON.parse(line), MAX_RUNTIME_MESSAGE_BYTES)
	if (!view.container && (!view.window || view.window.isDestroyed())) throw new Error('extension runtime View is unavailable')
	const webContents = view.container ? view.container.webContents : view.window.webContents
	if (webContents.isDestroyed()) throw new Error('extension runtime View is unavailable')
	webContents.send('aivo:extension-runtime-port-message', { clientPortId: port.clientPortId, message })
  }

  function disconnectRuntimePort(view, port, reason, notify, closeRemote) {
	if (port.closed) return
	port.closed = true
	port.abortController.abort()
	if (view.runtimePorts.get(port.clientPortId) === port) view.runtimePorts.delete(port.clientPortId)
	if (closeRemote) {
	  void fetch(runtimePortURL(port.backend, port.portId), {
		method: 'DELETE',
		headers: runtimeHeaders(port.backend, { 'X-Aivo-View-ID': view.viewId }),
		redirect: 'manual',
		signal: AbortSignal.timeout(RUNTIME_REQUEST_TIMEOUT_MS),
	  }).catch(() => {})
	}
	if (!notify) return
	const webContents = view.container ? view.container.webContents : view.window?.webContents
	if (webContents && !webContents.isDestroyed()) {
	  webContents.send('aivo:extension-runtime-port-disconnected', { clientPortId: port.clientPortId, reason })
	}
  }

  function closeAllRuntimePorts(view, reason, notify) {
	for (const port of [...view.runtimePorts.values()]) disconnectRuntimePort(view, port, reason, notify, true)
  }

  return {
	closeAll() {
	  for (const view of [...views.values()]) {
		if (view.window) view.window.destroy()
		else view.close('shutdown')
	  }
	  views.clear()
	  embeddedByOwner.clear()
	  embeddedMountRequests.clear()
	  embeddedMountGenerations.clear()
	  backends.clear()
	},
  }
}

function createIsolatedExtensionSession(extensionId, handleExtensionRequest) {
  const partition = `aivo-extension-${extensionId}-${crypto.randomUUID()}`
  const isolatedSession = session.fromPartition(partition, { cache: false })
  isolatedSession.protocol.handle('aivo-extension', handleExtensionRequest)
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
  return { partition, isolatedSession }
}

function disposeIsolatedExtensionSession(isolatedSession) {
  isolatedSession.protocol.unhandle('aivo-extension')
  isolatedSession.clearStorageData().catch(() => {})
}

function extensionWebPreferences(partition) {
  return {
	preload: path.join(__dirname, 'extension-preload.cjs'),
	partition,
	contextIsolation: true,
	nodeIntegration: false,
	sandbox: true,
	webSecurity: true,
	allowRunningInsecureContent: false,
	webviewTag: false,
  }
}

function secureExtensionWebContents(webContents, allowedOrigin) {
  webContents.setWindowOpenHandler(() => ({ action: 'deny' }))
  webContents.on('will-navigate', (event, target) => {
	try {
	  if (new URL(target).origin !== allowedOrigin) event.preventDefault()
	} catch {
	  event.preventDefault()
	}
  })
  webContents.on('will-attach-webview', (event) => event.preventDefault())
}

function boundedViewBounds(input, owner) {
  const content = owner.getContentBounds()
  const x = boundedInteger(input?.x, 0, Math.max(0, content.width - 1))
  const y = boundedInteger(input?.y, 0, Math.max(0, content.height - 1))
  return {
	x,
	y,
	width: boundedInteger(input?.width, 1, Math.max(1, content.width - x)),
	height: boundedInteger(input?.height, 1, Math.max(1, content.height - y)),
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

function requireRuntimeMessagingView(views, senderId) {
  const view = requireView(views, senderId)
  if (!view.runtimeMessaging) throw new Error('extension runtime messaging is not permitted')
  return view
}

async function requireCurrentRuntimeBackend(coreUrl, backends, view) {
  const descriptor = await callCore(coreUrl, 'ResolveExtensionView', { id: view.extensionId, viewId: view.viewId })
  validateDescriptor({ extensionId: view.extensionId, viewId: view.viewId, surface: view.surface }, descriptor)
  if (!Array.isArray(descriptor.permissions) || !descriptor.permissions.includes('runtime.messaging')) {
	throw new Error('extension runtime messaging is not permitted')
  }
  const backendURL = new URL(descriptor.backendUrl)
  const backend = {
	origin: backendURL.origin,
	token: typeof descriptor.backendToken === 'string' ? descriptor.backendToken : '',
	csp: typeof descriptor.csp === 'string' && descriptor.csp.length <= 2_000 ? descriptor.csp : DEFAULT_CSP,
  }
  backends.set(view.extensionId, backend)
  return backend
}

function runtimeMessageURL(backend) {
  return new URL('/.well-known/aivo-runtime/messages', backend.origin)
}

function runtimePortURL(backend, portId) {
  return new URL(`/.well-known/aivo-runtime/ports/${encodeURIComponent(portId)}`, backend.origin)
}

function runtimeHeaders(backend, values = {}) {
  const headers = new Headers(values)
  if (backend.token) headers.set('Authorization', `Bearer ${backend.token}`)
  return headers
}

async function readRuntimeJSONResponse(response) {
  const raw = await readBoundedRuntimeResponse(response)
  if (!response.ok) throw new Error(`extension runtime message failed with HTTP ${response.status}`)
  if (raw.length === 0) return null
  try {
	return boundedClone(JSON.parse(raw.toString('utf8')), MAX_RUNTIME_MESSAGE_BYTES)
  } catch {
	throw new Error('extension runtime message returned malformed JSON')
  }
}

async function discardBoundedRuntimeResponse(response) {
  await readBoundedRuntimeResponse(response)
}

async function readBoundedRuntimeResponse(response) {
  if (!response.body) return Buffer.alloc(0)
  const reader = response.body.getReader()
  const chunks = []
  let total = 0
  try {
	while (true) {
	  const { done, value } = await reader.read()
	  if (done) break
	  total += value.byteLength
	  if (total > MAX_RUNTIME_MESSAGE_BYTES) throw new Error('extension runtime response exceeds limit')
	  chunks.push(Buffer.from(value))
	}
	return Buffer.concat(chunks, total)
  } finally {
	reader.cancel().catch(() => {})
  }
}

function extensionViewContextPayload(view) {
  return {
	version: 1,
	revision: view.contextRevision,
	theme: 'system',
	locale: Intl.DateTimeFormat().resolvedOptions().locale,
	surface: view.surface,
	context: view.context,
  }
}

function boundedText(value, max) {
  return typeof value === 'string' ? value.trim().slice(0, max) : ''
}

function boundedIdentifier(value, max) {
  const text = typeof value === 'string' ? value.trim() : ''
  return text.length > 0 && text.length <= max && /^[A-Za-z0-9._-]+$/.test(text) ? text : ''
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
