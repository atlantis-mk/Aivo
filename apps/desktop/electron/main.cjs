const { app, BrowserWindow, WebContentsView, dialog, ipcMain, shell, session } = require('electron')
const { spawn } = require('node:child_process')
const fs = require('node:fs')
const http = require('node:http')
const os = require('node:os')
const path = require('node:path')

const isDev = Boolean(process.env.VITE_DEV_SERVER_URL)
const isMac = process.platform === 'darwin'
const coreUrl = process.env.AIVO_CORE_URL || 'http://127.0.0.1:43117'
const browserToolBridgePort = Number(process.env.AIVO_BROWSER_BRIDGE_PORT || '43118')

let coreProcess = null
let logFile = null
let browserToolBridgeServer = null
const DEFAULT_BROWSER_TAB_ID = 'builtin-browser'
let browserTabs = new Map()
let browserOpenRequestId = 0
const browserOpenRequests = new Map()

function createEmptyBrowserState() {
  return {
    url: '',
    title: '',
    favicon: '',
    loading: false,
    canGoBack: false,
    canGoForward: false,
    error: '',
  }
}

function createBrowserTab(id) {
  return {
    id,
    view: null,
    ownerWindow: null,
    bounds: { x: 0, y: 0, width: 0, height: 0 },
    visible: false,
    faviconRequestId: 0,
    consoleMessages: [],
    networkRequests: [],
    state: createEmptyBrowserState(),
  }
}

function normalizeBrowserTabId(tabId) {
  const value = typeof tabId === 'string' ? tabId.trim() : ''
  return value || DEFAULT_BROWSER_TAB_ID
}

function getBrowserTab(tabId) {
  const id = normalizeBrowserTabId(tabId)
  const existing = browserTabs.get(id)
  if (existing) return existing
  const tab = createBrowserTab(id)
  browserTabs.set(id, tab)
  return tab
}

function browserPartitionForTab(tab) {
  return `persist:aivo-built-in-browser:${encodeURIComponent(tab.id)}`
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
    env: {
      ...process.env,
      AIVO_BROWSER_BRIDGE_URL: `http://127.0.0.1:${browserToolBridgePort}`,
    },
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
      webviewTag: true,
    },
  })

  mainWindow.webContents.on('did-attach-webview', (_event, contents) => {
    contents.setWindowOpenHandler((details) => {
      try {
        const target = normalizeBrowserURL(details.url)
        mainWindow.webContents.send('aivo:browser-navigate-current', {
          sourceWebContentsId: contents.id,
          url: target,
        })
      } catch {
        appendLog('info', 'blocked unsupported browser popup', {
          sourceWebContentsId: contents.id,
          url: details.url,
        })
      }
      return { action: 'deny' }
    })
  })

  if (isDev) {
    mainWindow.loadURL(process.env.VITE_DEV_SERVER_URL)
    mainWindow.webContents.openDevTools({ mode: 'detach' })
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

function normalizeBrowserURL(input) {
  const raw = typeof input === 'string' ? input.trim() : ''
  if (!raw) {
    throw new Error('Browser address is required')
  }
  const withScheme = /^[a-zA-Z][a-zA-Z\d+.-]*:/.test(raw) ? raw : `https://${raw}`
  const parsed = new URL(withScheme)
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error('Only http(s) addresses can be opened in the built-in browser')
  }
  return parsed.toString()
}

function normalizeBrowserFaviconURL(input, pageURL) {
  const value = typeof input === 'string' ? input.trim() : ''
  if (!value) return ''
  if (value.startsWith('data:image/')) return value
  try {
    const parsed = new URL(value, pageURL)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return ''
    return parsed.toString()
  } catch {
    return ''
  }
}

async function loadBrowserFaviconDataURL(faviconURL) {
  if (faviconURL.startsWith('data:image/')) return faviconURL
  const response = await fetch(faviconURL, {
    signal: AbortSignal.timeout(5000),
  })
  if (!response.ok) {
    throw new Error(`Favicon request failed with ${response.status}`)
  }
  const contentType = response.headers.get('content-type') || 'image/x-icon'
  if (!contentType.toLowerCase().startsWith('image/')) {
    throw new Error(`Favicon response is not an image: ${contentType}`)
  }
  const buffer = Buffer.from(await response.arrayBuffer())
  if (buffer.byteLength === 0 || buffer.byteLength > 1024 * 1024) {
    throw new Error(`Favicon response has invalid size: ${buffer.byteLength}`)
  }
  return `data:${contentType.split(';')[0]};base64,${buffer.toString('base64')}`
}

function updateBrowserFavicon(tab, favicons, pageURL) {
  const requestId = (tab.faviconRequestId += 1)
  const candidates = Array.isArray(favicons) ? favicons : []
  const urls = [
    ...candidates.map((favicon) => normalizeBrowserFaviconURL(favicon, pageURL)),
    normalizeBrowserFaviconURL('/favicon.ico', pageURL),
  ].filter(Boolean)
  const uniqueUrls = [...new Set(urls)]
  if (uniqueUrls.length === 0) {
    updateBrowserState(tab, { favicon: '' })
    return
  }

  void (async () => {
    for (const faviconURL of uniqueUrls) {
      try {
        const dataURL = await loadBrowserFaviconDataURL(faviconURL)
        if (requestId !== tab.faviconRequestId) return
        const currentURL = tab.view?.webContents?.getURL() || ''
        if (pageURL && currentURL && currentURL !== pageURL) return
        updateBrowserState(tab, { favicon: dataURL })
        return
      } catch {
        // Try the next favicon candidate.
      }
    }
    if (requestId === tab.faviconRequestId) {
      updateBrowserState(tab, { favicon: '' })
    }
  })()
}

function getBrowserOwnerWindow(event) {
  return BrowserWindow.fromWebContents(event.sender) || BrowserWindow.getAllWindows()[0] || null
}

function emitBrowserState(tab) {
  const payload = { tabId: tab.id, state: { ...tab.state } }
  for (const window of BrowserWindow.getAllWindows()) {
    if (!window.isDestroyed()) {
      window.webContents.send('aivo:browser-state', payload)
    }
  }
}

function updateBrowserState(tab, patch = {}) {
  if (tab.view && !tab.view.webContents.isDestroyed()) {
    tab.state = {
      ...tab.state,
      ...patch,
      url: patch.url ?? tab.view.webContents.getURL() ?? tab.state.url,
      title: patch.title ?? tab.view.webContents.getTitle() ?? tab.state.title,
      loading: patch.loading ?? tab.view.webContents.isLoading(),
      canGoBack: tab.view.webContents.canGoBack(),
      canGoForward: tab.view.webContents.canGoForward(),
    }
  } else {
    tab.state = { ...tab.state, ...patch }
  }
  emitBrowserState(tab)
}

function pushBoundedBrowserEvent(list, item, limit = 200) {
  list.push({
    time: new Date().toISOString(),
    ...item,
  })
  if (list.length > limit) {
    list.splice(0, list.length - limit)
  }
}

function visibleBrowserState(tab) {
  return {
    ...tab.state,
    visible: Boolean(tab.visible),
    tabId: tab.id,
  }
}

function applyBrowserBounds(tab) {
  if (!tab.view) return
  const nextBounds = tab.visible
    ? {
        x: Math.max(0, Math.round(tab.bounds.x)),
        y: Math.max(0, Math.round(tab.bounds.y)),
        width: Math.max(0, Math.round(tab.bounds.width)),
        height: Math.max(0, Math.round(tab.bounds.height)),
      }
    : { x: 0, y: 0, width: 0, height: 0 }
  tab.view.setBounds(nextBounds)
}

function hideOtherBrowserTabs(activeTab) {
  for (const tab of browserTabs.values()) {
    if (tab === activeTab || !tab.visible) continue
    tab.visible = false
    applyBrowserBounds(tab)
    updateBrowserState(tab)
  }
}

function ensureBrowserView(ownerWindow, tabId) {
  if (!ownerWindow || ownerWindow.isDestroyed()) {
    throw new Error('No active Aivo window is available for the built-in browser')
  }
  const tab = getBrowserTab(tabId)
  if (tab.view && !tab.view.webContents.isDestroyed()) {
    if (tab.ownerWindow !== ownerWindow) {
      try {
        tab.ownerWindow?.contentView?.removeChildView(tab.view)
      } catch {
        // Best-effort cleanup when moving between windows.
      }
      ownerWindow.contentView.addChildView(tab.view)
      tab.ownerWindow = ownerWindow
      applyBrowserBounds(tab)
    }
    return tab
  }

  const partition = browserPartitionForTab(tab)
  const browserSession = session.fromPartition(partition)
  browserSession.setPermissionRequestHandler((_webContents, permission, callback) => {
    appendLog('info', 'browser permission denied', { permission })
    callback(false)
  })

  tab.view = new WebContentsView({
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
      partition,
    },
  })
  tab.ownerWindow = ownerWindow
  ownerWindow.contentView.addChildView(tab.view)
  tab.view.setBackgroundColor('#00000000')
  applyBrowserBounds(tab)

  const contents = tab.view.webContents
  contents.setWindowOpenHandler(({ url }) => {
    try {
      const target = normalizeBrowserURL(url)
      void contents.loadURL(target)
    } catch {
      void shell.openExternal(url).catch(() => undefined)
    }
    return { action: 'deny' }
  })
  contents.on('will-navigate', (event, targetURL) => {
    try {
      normalizeBrowserURL(targetURL)
    } catch {
      event.preventDefault()
      updateBrowserState(tab, { error: '已阻止非 http(s) 导航' })
    }
  })
  contents.on('did-start-loading', () => updateBrowserState(tab, { loading: true, error: '' }))
  contents.on('did-stop-loading', () => updateBrowserState(tab, { loading: false }))
  contents.on('dom-ready', () => installBrowserPageBehavior(contents))
  contents.on('did-navigate', (_event, url) => {
    tab.faviconRequestId += 1
    updateBrowserState(tab, { url, favicon: '', error: '' })
    updateBrowserFavicon(tab, [], url)
    installBrowserPageBehavior(contents)
  })
  contents.on('did-navigate-in-page', (_event, url) => {
    updateBrowserState(tab, { url, error: '' })
    installBrowserPageBehavior(contents)
  })
  contents.on('page-title-updated', (_event, title) => updateBrowserState(tab, { title }))
  contents.on('page-favicon-updated', (_event, favicons) => {
    updateBrowserFavicon(tab, favicons, contents.getURL())
  })
  contents.on('console-message', (_event, level, message, line, sourceId) => {
    pushBoundedBrowserEvent(tab.consoleMessages, {
      level,
      message: String(message || ''),
      line: Number(line) || 0,
      sourceId: String(sourceId || ''),
      url: contents.getURL(),
    })
  })
  contents.on('did-fail-load', (_event, errorCode, errorDescription, validatedURL, isMainFrame) => {
    if (!isMainFrame || errorCode === -3) return
    updateBrowserState(tab, {
      error: errorDescription || `Load failed (${errorCode})`,
      loading: false,
      url: validatedURL || tab.state.url,
    })
  })
  contents.on('destroyed', () => {
    tab.view = null
    tab.ownerWindow = null
    tab.visible = false
    updateBrowserState(tab, createEmptyBrowserState())
  })
  browserSession.webRequest.onCompleted((details) => {
    if (details.webContentsId !== contents.id) return
    pushBoundedBrowserEvent(tab.networkRequests, {
      method: details.method,
      url: details.url,
      statusCode: details.statusCode,
      resourceType: details.resourceType,
      fromCache: Boolean(details.fromCache),
    })
  })
  browserSession.webRequest.onErrorOccurred((details) => {
    if (details.webContentsId !== contents.id) return
    pushBoundedBrowserEvent(tab.networkRequests, {
      method: details.method,
      url: details.url,
      error: details.error,
      resourceType: details.resourceType,
    })
  })

  return tab
}

function installBrowserPageBehavior(contents) {
  if (!contents || contents.isDestroyed()) return
  void contents.executeJavaScript(`
    (() => {
      if (window.__aivoCurrentTabLinkHandlingInstalled) return;
      window.__aivoCurrentTabLinkHandlingInstalled = true;

      const shouldHandleHref = (href) => {
        if (!href) return false;
        const normalized = String(href).trim().toLowerCase();
        return normalized && !normalized.startsWith("javascript:");
      };

      const rewriteLinkTargets = (root) => {
        root.querySelectorAll?.("a[target]").forEach((anchor) => {
          anchor.setAttribute("target", "_self");
        });
      };

      rewriteLinkTargets(document);
      new MutationObserver((mutations) => {
        for (const mutation of mutations) {
          for (const node of mutation.addedNodes) {
            if (node.nodeType !== Node.ELEMENT_NODE) continue;
            if (node.matches?.("a[target]")) {
              node.setAttribute("target", "_self");
            }
            rewriteLinkTargets(node);
          }
        }
      }).observe(document.documentElement, { childList: true, subtree: true });

      document.addEventListener("click", (event) => {
        const anchor = event.target?.closest?.("a[href]");
        if (!anchor || !shouldHandleHref(anchor.href)) return;
        const target = String(anchor.getAttribute("target") || "").toLowerCase();
        if (!target || target === "_self") return;
        event.preventDefault();
        event.stopPropagation();
        window.location.href = anchor.href;
      }, true);

      const originalOpen = window.open;
      window.open = function(url, target, features) {
        if (shouldHandleHref(url)) {
          window.location.href = new URL(String(url), window.location.href).toString();
          return window;
        }
        return originalOpen.call(window, url, target || "_self", features);
      };
    })();
  `).catch(() => undefined)
}

function browserToolOwnerWindow() {
  return BrowserWindow.getAllWindows().find((window) => !window.isDestroyed()) || null
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function waitForBrowserLoad(tab, timeoutMs = 10000) {
  const contents = tab?.view?.webContents
  if (!contents || contents.isDestroyed() || !contents.isLoading()) {
    return Promise.resolve()
  }
  return new Promise((resolve) => {
    const timeout = setTimeout(done, timeoutMs)
    function done() {
      clearTimeout(timeout)
      contents.removeListener('did-stop-loading', done)
      contents.removeListener('did-fail-load', done)
      resolve()
    }
    contents.once('did-stop-loading', done)
    contents.once('did-fail-load', done)
  })
}

function requestBrowserTabOpen(tabId, targetUrl = '') {
  const ownerWindow = browserToolOwnerWindow()
  if (!ownerWindow) {
    return Promise.reject(new Error('No active Aivo window is available for the built-in browser'))
  }
  const id = `browser-open-${Date.now().toString(36)}-${++browserOpenRequestId}`
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      browserOpenRequests.delete(id)
      reject(new Error('Timed out waiting for the desktop UI to open the built-in browser'))
    }, 2500)
    browserOpenRequests.set(id, {
      resolve: () => {
        clearTimeout(timeout)
        resolve()
      },
      reject: (error) => {
        clearTimeout(timeout)
        reject(error)
      },
    })
    ownerWindow.webContents.send('aivo:browser-open-request', {
      requestId: id,
      tabId: normalizeBrowserTabId(tabId),
      url: targetUrl || '',
    })
  })
}

async function ensureBrowserTabForTool(tabId, targetUrl = '') {
  const id = normalizeBrowserTabId(tabId)
  await requestBrowserTabOpen(id, targetUrl).catch((error) => {
    appendLog('info', 'browser open request was not acknowledged', { tabId: id, error: error.message })
  })
  const tab = ensureBrowserView(browserToolOwnerWindow(), id)
  for (let attempt = 0; attempt < 15 && (tab.bounds.width <= 0 || tab.bounds.height <= 0); attempt += 1) {
    await delay(100)
  }
  tab.visible = true
  applyBrowserBounds(tab)
  updateBrowserState(tab)
  return tab
}

function normalizeBrowserToolLimit(value, fallback, max) {
  const next = Number(value)
  if (!Number.isFinite(next) || next <= 0) return fallback
  return Math.min(Math.round(next), max)
}

function browserPageSnapshotScript(maxTextChars, maxElements) {
  return `
    (() => {
      const maxTextChars = ${Number(maxTextChars) || 12000};
      const maxElements = ${Number(maxElements) || 80};
      const visible = (el) => {
        const style = window.getComputedStyle(el);
        const rect = el.getBoundingClientRect();
        return style.visibility !== "hidden" && style.display !== "none" && rect.width > 0 && rect.height > 0;
      };
      const labelFor = (el) => {
        const aria = el.getAttribute("aria-label") || el.getAttribute("alt") || el.getAttribute("title") || "";
        const text = (el.innerText || el.value || el.placeholder || el.name || "").trim();
        return (aria || text || el.id || el.tagName.toLowerCase()).replace(/\\s+/g, " ").slice(0, 160);
      };
      const selectorFor = (el) => {
        if (el.id) return "#" + CSS.escape(el.id);
        const name = el.getAttribute("name");
        const testId = el.getAttribute("data-testid") || el.getAttribute("data-test");
        if (name) return el.tagName.toLowerCase() + "[name=" + JSON.stringify(name) + "]";
        if (testId) return el.tagName.toLowerCase() + "[" + (el.getAttribute("data-testid") ? "data-testid" : "data-test") + "=" + JSON.stringify(testId) + "]";
        const parts = [];
        let node = el;
        while (node && node.nodeType === Node.ELEMENT_NODE && parts.length < 4) {
          let part = node.tagName.toLowerCase();
          if (node.classList && node.classList.length > 0) {
            part += "." + Array.from(node.classList).slice(0, 2).map((name) => CSS.escape(name)).join(".");
          }
          const parent = node.parentElement;
          if (parent) {
            const siblings = Array.from(parent.children).filter((child) => child.tagName === node.tagName);
            if (siblings.length > 1) part += ":nth-of-type(" + (siblings.indexOf(node) + 1) + ")";
          }
          parts.unshift(part);
          node = parent;
        }
        return parts.join(" > ");
      };
      const candidates = Array.from(document.querySelectorAll("a,button,input,textarea,select,[role=button],[role=link],[contenteditable=true]"));
      const elements = candidates.filter(visible).slice(0, maxElements).map((el, index) => {
        const rect = el.getBoundingClientRect();
        return {
          index,
          tag: el.tagName.toLowerCase(),
          role: el.getAttribute("role") || "",
          type: el.getAttribute("type") || "",
          label: labelFor(el),
          selector: selectorFor(el),
          href: el.href || "",
          disabled: Boolean(el.disabled || el.getAttribute("aria-disabled") === "true"),
          rect: { x: Math.round(rect.x), y: Math.round(rect.y), width: Math.round(rect.width), height: Math.round(rect.height) },
        };
      });
      const text = (document.body?.innerText || "").replace(/\\n{3,}/g, "\\n\\n").trim();
      return {
        url: location.href,
        title: document.title,
        text: text.slice(0, maxTextChars),
        textTruncated: text.length > maxTextChars,
        elements,
      };
    })();
  `
}

function browserFindElementScript(input, action, value = '') {
  const selector = typeof input.selector === 'string' ? input.selector.trim() : ''
  const text = typeof input.text === 'string' ? input.text.trim() : ''
  const index = Number.isInteger(input.index) ? input.index : -1
  return `
    (() => {
      const selector = ${JSON.stringify(selector)};
      const text = ${JSON.stringify(text)};
      const index = ${index};
      const value = ${JSON.stringify(value)};
      const visible = (el) => {
        const style = window.getComputedStyle(el);
        const rect = el.getBoundingClientRect();
        return style.visibility !== "hidden" && style.display !== "none" && rect.width > 0 && rect.height > 0;
      };
      const labelFor = (el) => [
        el.getAttribute("aria-label"),
        el.getAttribute("alt"),
        el.getAttribute("title"),
        el.innerText,
        el.value,
        el.placeholder,
        el.name,
        el.id,
      ].filter(Boolean).join(" ").replace(/\\s+/g, " ").trim();
      let el = null;
      if (selector) el = document.querySelector(selector);
      if (!el && index >= 0) {
        el = Array.from(document.querySelectorAll("a,button,input,textarea,select,[role=button],[role=link],[contenteditable=true]")).filter(visible)[index] || null;
      }
      if (!el && text) {
        const needle = text.toLowerCase();
        el = Array.from(document.querySelectorAll("a,button,input,textarea,select,label,[role=button],[role=link],[contenteditable=true]"))
          .filter(visible)
          .find((candidate) => labelFor(candidate).toLowerCase().includes(needle)) || null;
      }
      if (!el) return { ok: false, error: "element not found" };
      el.scrollIntoView({ block: "center", inline: "center" });
      el.focus?.();
      if (${JSON.stringify(action)} === "fill") {
        if ("value" in el) {
          el.value = value;
          el.dispatchEvent(new Event("input", { bubbles: true }));
          el.dispatchEvent(new Event("change", { bubbles: true }));
        } else if (el.isContentEditable) {
          el.textContent = value;
          el.dispatchEvent(new InputEvent("input", { bubbles: true, inputType: "insertText", data: value }));
        } else {
          return { ok: false, error: "element is not fillable" };
        }
      } else {
        el.click();
      }
      const rect = el.getBoundingClientRect();
      return {
        ok: true,
        tag: el.tagName.toLowerCase(),
        label: labelFor(el).slice(0, 160),
        rect: { x: Math.round(rect.x), y: Math.round(rect.y), width: Math.round(rect.width), height: Math.round(rect.height) },
      };
    })();
  `
}

async function executeBrowserTool(tool, args = {}) {
  const tabId = normalizeBrowserTabId(args.tabId)
  const targetUrl = typeof args.url === 'string' ? args.url : ''
  const tab = await ensureBrowserTabForTool(tabId, targetUrl)
  const contents = tab.view.webContents

  switch (tool) {
    case 'state':
      updateBrowserState(tab)
      return { ok: true, content: JSON.stringify(visibleBrowserState(tab), null, 2), structured: visibleBrowserState(tab) }
    case 'navigate': {
      const url = normalizeBrowserURL(args.url)
      updateBrowserState(tab, { url, title: '', favicon: '', loading: true, error: '' })
      await contents.loadURL(url)
      await waitForBrowserLoad(tab, normalizeBrowserToolLimit(args.timeoutMs, 10000, 30000))
      updateBrowserState(tab)
      return { ok: true, content: JSON.stringify(visibleBrowserState(tab), null, 2), structured: visibleBrowserState(tab) }
    }
    case 'snapshot': {
      await waitForBrowserLoad(tab, normalizeBrowserToolLimit(args.timeoutMs, 3000, 15000))
      const snapshot = await contents.executeJavaScript(browserPageSnapshotScript(normalizeBrowserToolLimit(args.maxTextChars, 12000, 50000), normalizeBrowserToolLimit(args.maxElements, 80, 200)))
      const structured = { ...visibleBrowserState(tab), snapshot }
      const content = [
        `URL: ${snapshot.url}`,
        `Title: ${snapshot.title}`,
        '',
        'Text:',
        snapshot.text || '',
        '',
        'Interactive elements:',
        ...(snapshot.elements || []).map((element) => `${element.index}. ${element.tag}${element.type ? `[${element.type}]` : ''} "${element.label}" selector=${element.selector}`),
      ].join('\n')
      return { ok: true, content, structured }
    }
    case 'click': {
      const result = await contents.executeJavaScript(browserFindElementScript(args, 'click'))
      await delay(250)
      updateBrowserState(tab)
      return { ok: Boolean(result?.ok), content: JSON.stringify({ ...result, state: visibleBrowserState(tab) }, null, 2), structured: { result, state: visibleBrowserState(tab) }, error: result?.ok ? '' : result?.error || 'click failed' }
    }
    case 'fill': {
      const value = typeof args.value === 'string' ? args.value : ''
      const result = await contents.executeJavaScript(browserFindElementScript(args, 'fill', value))
      updateBrowserState(tab)
      return { ok: Boolean(result?.ok), content: JSON.stringify({ ...result, state: visibleBrowserState(tab) }, null, 2), structured: { result, state: visibleBrowserState(tab) }, error: result?.ok ? '' : result?.error || 'fill failed' }
    }
    case 'press_key': {
      const key = typeof args.key === 'string' && args.key.trim() ? args.key.trim() : 'Enter'
      contents.sendInputEvent({ type: 'keyDown', keyCode: key })
      contents.sendInputEvent({ type: 'keyUp', keyCode: key })
      await delay(150)
      updateBrowserState(tab)
      return { ok: true, content: JSON.stringify(visibleBrowserState(tab), null, 2), structured: visibleBrowserState(tab) }
    }
    case 'evaluate': {
      const script = typeof args.script === 'string' ? args.script : ''
      if (!script.trim()) throw new Error('script is required')
      const value = await contents.executeJavaScript(script, Boolean(args.userGesture))
      return { ok: true, content: JSON.stringify({ value }, null, 2), structured: { value, state: visibleBrowserState(tab) } }
    }
    case 'screenshot': {
      const image = await contents.capturePage()
      const dataUrl = image.toDataURL()
      const structured = { ...visibleBrowserState(tab), mimeType: 'image/png', dataUrl, bytes: Buffer.byteLength(dataUrl) }
      return { ok: true, content: JSON.stringify({ ...visibleBrowserState(tab), mimeType: 'image/png', dataUrl }, null, 2), structured }
    }
    case 'console': {
      const limit = normalizeBrowserToolLimit(args.limit, 50, 200)
      const messages = tab.consoleMessages.slice(-limit)
      return { ok: true, content: JSON.stringify(messages, null, 2), structured: { messages, state: visibleBrowserState(tab) } }
    }
    case 'network': {
      const limit = normalizeBrowserToolLimit(args.limit, 50, 200)
      const requests = tab.networkRequests.slice(-limit)
      return { ok: true, content: JSON.stringify(requests, null, 2), structured: { requests, state: visibleBrowserState(tab) } }
    }
    default:
      throw new Error(`Unknown browser tool operation: ${tool}`)
  }
}

function startBrowserToolBridge() {
  if (browserToolBridgeServer) {
    return Promise.resolve()
  }
  browserToolBridgeServer = http.createServer(async (req, res) => {
    if (req.method !== 'POST' || req.url !== '/browser-tool') {
      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ ok: false, error: 'not found' }))
      return
    }
    let raw = ''
    req.setEncoding('utf8')
    req.on('data', (chunk) => {
      raw += chunk
      if (raw.length > 1024 * 1024) req.destroy()
    })
    req.on('end', async () => {
      try {
        const payload = raw ? JSON.parse(raw) : {}
        const result = await executeBrowserTool(String(payload.tool || ''), payload.args || {})
        res.writeHead(result.ok ? 200 : 422, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify(result))
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error)
        appendLog('error', 'browser tool failed', { error: message })
        res.writeHead(500, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ ok: false, error: message }))
      }
    })
  })
  return new Promise((resolve, reject) => {
    browserToolBridgeServer.once('error', reject)
    browserToolBridgeServer.listen(browserToolBridgePort, '127.0.0.1', () => {
      browserToolBridgeServer.off('error', reject)
      appendLog('info', 'browser tool bridge listening', { port: browserToolBridgePort })
      resolve()
    })
  })
}

ipcMain.on('aivo:browser-open-response', (_event, payload) => {
  const requestId = typeof payload?.requestId === 'string' ? payload.requestId : ''
  const pending = browserOpenRequests.get(requestId)
  if (!pending) return
  browserOpenRequests.delete(requestId)
  if (payload?.ok === false) {
    pending.reject(new Error(payload?.error || 'Failed to open browser tab'))
    return
  }
  pending.resolve()
})

ipcMain.handle('aivo:select-project-directory', async () => {
  const result = await dialog.showOpenDialog({
    properties: ['openDirectory'],
  })

  if (result.canceled || result.filePaths.length === 0) {
    return ''
  }

  return result.filePaths[0]
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

ipcMain.handle('aivo:browser:get-state', (event, tabId) => {
  const tab = ensureBrowserView(getBrowserOwnerWindow(event), tabId)
  updateBrowserState(tab)
  return tab.state
})

ipcMain.handle('aivo:browser:set-visible', (event, tabId, visible) => {
  const tab = ensureBrowserView(getBrowserOwnerWindow(event), tabId)
  tab.visible = Boolean(visible)
  if (tab.visible) {
    hideOtherBrowserTabs(tab)
  }
  applyBrowserBounds(tab)
  updateBrowserState(tab)
  return tab.state
})

ipcMain.handle('aivo:browser:close', (_event, tabId) => {
  const id = normalizeBrowserTabId(tabId)
  const tab = browserTabs.get(id)
  if (!tab) return createEmptyBrowserState()
  tab.visible = false
  if (tab.view && !tab.view.webContents.isDestroyed()) {
    try {
      tab.ownerWindow?.contentView?.removeChildView(tab.view)
    } catch {
      // Best-effort cleanup before destroying the tab view.
    }
    tab.view.webContents.destroy()
  }
  browserTabs.delete(id)
  return createEmptyBrowserState()
})

ipcMain.handle('aivo:browser:set-bounds', (event, tabId, bounds) => {
  const tab = ensureBrowserView(getBrowserOwnerWindow(event), tabId)
  if (bounds && typeof bounds === 'object') {
    tab.bounds = {
      x: Number(bounds.x) || 0,
      y: Number(bounds.y) || 0,
      width: Number(bounds.width) || 0,
      height: Number(bounds.height) || 0,
    }
  }
  applyBrowserBounds(tab)
  return tab.state
})

ipcMain.handle('aivo:browser:navigate', async (event, tabId, target) => {
  const tab = ensureBrowserView(getBrowserOwnerWindow(event), tabId)
  const url = normalizeBrowserURL(target)
  updateBrowserState(tab, { url, title: '', favicon: '', loading: true, error: '' })
  await tab.view.webContents.loadURL(url)
  updateBrowserState(tab)
  return tab.state
})

ipcMain.handle('aivo:browser:go-back', (event, tabId) => {
  const tab = ensureBrowserView(getBrowserOwnerWindow(event), tabId)
  if (tab.view.webContents.canGoBack()) {
    tab.view.webContents.goBack()
  }
  updateBrowserState(tab)
  return tab.state
})

ipcMain.handle('aivo:browser:go-forward', (event, tabId) => {
  const tab = ensureBrowserView(getBrowserOwnerWindow(event), tabId)
  if (tab.view.webContents.canGoForward()) {
    tab.view.webContents.goForward()
  }
  updateBrowserState(tab)
  return tab.state
})

ipcMain.handle('aivo:browser:reload', (event, tabId) => {
  const tab = ensureBrowserView(getBrowserOwnerWindow(event), tabId)
  if (tab.view.webContents.getURL()) {
    tab.view.webContents.reload()
  }
  updateBrowserState(tab)
  return tab.state
})

ipcMain.handle('aivo:browser:stop', (event, tabId) => {
  const tab = ensureBrowserView(getBrowserOwnerWindow(event), tabId)
  tab.view.webContents.stop()
  updateBrowserState(tab, { loading: false })
  return tab.state
})

ipcMain.handle('aivo:browser:load-favicon', async (_event, favicons, pageURL) => {
  const candidates = Array.isArray(favicons) ? favicons : []
  const urls = [
    ...candidates.map((favicon) => normalizeBrowserFaviconURL(favicon, pageURL)),
    normalizeBrowserFaviconURL('/favicon.ico', pageURL),
  ].filter(Boolean)
  const uniqueUrls = [...new Set(urls)]

  for (const faviconURL of uniqueUrls) {
    try {
      return await loadBrowserFaviconDataURL(faviconURL)
    } catch {
      // Try the next favicon candidate.
    }
  }

  return ''
})

app.whenReady().then(async () => {
  initializeDiagnostics()
  try {
    await startBrowserToolBridge()
  } catch (error) {
    appendLog('error', 'browser tool bridge failed to start', {
      error: error instanceof Error ? error.message : String(error),
    })
  }
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
  if (browserToolBridgeServer) {
    browserToolBridgeServer.close()
    browserToolBridgeServer = null
  }
  if (coreProcess && coreProcess.exitCode === null) {
    coreProcess.kill()
  }
})
