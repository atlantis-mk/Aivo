const crypto = require('node:crypto')
const fs = require('node:fs')
const path = require('node:path')

const STABLE_MANIFEST_URL = 'https://pub-bf5092e77ab5409ba39fb34c4a76c1b1.r2.dev/aivo/channels/stable/latest.json'
const R2_ORIGIN = 'https://pub-bf5092e77ab5409ba39fb34c4a76c1b1.r2.dev'
const GITHUB_API_ORIGIN = 'https://api.github.com'
const GITHUB_REPOSITORY = 'atlantis-mk/Aivo'
const MAX_MANIFEST_BYTES = 512 * 1024
const MAX_RELEASE_BYTES = 2 * 1024 * 1024
const MAX_PACKAGE_BYTES = 768 * 1024 * 1024
const METADATA_TIMEOUT_MS = 15_000
const DOWNLOAD_TIMEOUT_MS = 30 * 60_000
const STABLE_VERSION = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/
const SHA256 = /^[a-f0-9]{64}$/

class UpdaterError extends Error {
  constructor(code, message) {
    super(message)
    this.name = 'UpdaterError'
    this.code = code
  }
}

function compareStableVersions(left, right) {
  if (!STABLE_VERSION.test(left) || !STABLE_VERSION.test(right)) {
    throw new UpdaterError('invalid-version', '更新通道返回了无效的稳定版本号。')
  }
  const leftParts = left.split('.').map(Number)
  const rightParts = right.split('.').map(Number)
  for (let index = 0; index < 3; index += 1) {
    if (leftParts[index] !== rightParts[index]) {
      return leftParts[index] < rightParts[index] ? -1 : 1
    }
  }
  return 0
}

function artifactNameFor(version, platform, arch) {
  if (!STABLE_VERSION.test(version)) return null
  if (platform === 'darwin' && arch === 'arm64') return `Aivo_${version}_darwin-aarch64.dmg`
  if (platform === 'darwin' && arch === 'x64') return `Aivo_${version}_darwin-x86_64.dmg`
  if (platform === 'win32' && arch === 'x64') return `Aivo_${version}_windows-x86_64-setup.exe`
  if (platform === 'linux' && arch === 'x64') return `Aivo_${version}_linux-x86_64.AppImage`
  return null
}

function isPlainObject(value) {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function validateR2ArtifactUrl(rawUrl, version, name) {
  let parsed
  try {
    parsed = new URL(rawUrl)
  } catch {
    throw new UpdaterError('invalid-artifact-url', '更新包地址无效。')
  }
  const expectedPath = `/aivo/releases/v${version}/${name}`
  if (parsed.origin !== R2_ORIGIN || parsed.protocol !== 'https:' || parsed.pathname !== expectedPath || parsed.search || parsed.hash) {
    throw new UpdaterError('invalid-artifact-url', '更新包不在受信任的不可变发布路径中。')
  }
  return parsed.href
}

function validateManifest(input, platform, arch) {
  if (!isPlainObject(input) || !STABLE_VERSION.test(input.version) || input.tag !== `v${input.version}` || !Array.isArray(input.artifacts)) {
    throw new UpdaterError('invalid-manifest', '稳定更新清单格式无效。')
  }
  const expectedBase = `${R2_ORIGIN}/aivo/releases/v${input.version}`
  if (input.releaseBaseUrl !== expectedBase) {
    throw new UpdaterError('invalid-manifest', '稳定更新清单的发布路径无效。')
  }
  const name = artifactNameFor(input.version, platform, arch)
  if (!name) {
    return { supported: false, version: input.version, tag: input.tag }
  }
  const matches = input.artifacts.filter((artifact) => isPlainObject(artifact) && artifact.name === name)
  if (matches.length !== 1) {
    throw new UpdaterError('ambiguous-artifact', '稳定更新清单缺少唯一的当前平台安装包。')
  }
  const artifact = matches[0]
  if (!Number.isSafeInteger(artifact.size) || artifact.size <= 0 || artifact.size > MAX_PACKAGE_BYTES || !SHA256.test(artifact.sha256)) {
    throw new UpdaterError('invalid-artifact', '当前平台安装包的大小或摘要无效。')
  }
  return {
    supported: true,
    version: input.version,
    tag: input.tag,
    artifact: {
      name,
      url: validateR2ArtifactUrl(artifact.url, input.version, name),
      size: artifact.size,
      sha256: artifact.sha256,
    },
  }
}

function validateGitHubRelease(input, candidate) {
  if (!isPlainObject(input) || input.tag_name !== candidate.tag || input.draft !== false || input.prerelease !== false || !Array.isArray(input.assets)) {
    throw new UpdaterError('invalid-github-release', 'GitHub Release 与稳定更新版本不一致。')
  }
  const matches = input.assets.filter((asset) => isPlainObject(asset) && asset.name === candidate.artifact.name)
  if (matches.length !== 1) {
    throw new UpdaterError('ambiguous-github-asset', 'GitHub Release 缺少唯一的当前平台安装包。')
  }
  const asset = matches[0]
  if (asset.size !== candidate.artifact.size || asset.digest !== `sha256:${candidate.artifact.sha256}`) {
    throw new UpdaterError('release-mismatch', 'R2 与 GitHub Release 的更新包大小或摘要不一致。')
  }
  return candidate
}

function combinedAbortSignal(parentSignal, timeoutMs) {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(new UpdaterError('timeout', '更新请求超时。')), timeoutMs)
  const abort = () => controller.abort(parentSignal.reason)
  if (parentSignal.aborted) abort()
  else parentSignal.addEventListener('abort', abort, { once: true })
  return {
    signal: controller.signal,
    dispose() {
      clearTimeout(timeout)
      parentSignal.removeEventListener('abort', abort)
    },
  }
}

async function fetchBoundedJson(fetchImpl, url, maxBytes, signal) {
  const response = await fetchImpl(url, {
    headers: {
      Accept: 'application/vnd.github+json, application/json',
      'User-Agent': 'Aivo-Desktop-Updater',
    },
    redirect: 'error',
    signal,
  })
  if (!response.ok || !response.body) {
    throw new UpdaterError('metadata-unavailable', '无法读取更新元数据。')
  }
  const declaredLength = Number(response.headers.get('content-length') || 0)
  if (declaredLength > maxBytes) {
    throw new UpdaterError('metadata-too-large', '更新元数据超过大小限制。')
  }
  const reader = response.body.getReader()
  const chunks = []
  let total = 0
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    total += value.byteLength
    if (total > maxBytes) {
      await reader.cancel()
      throw new UpdaterError('metadata-too-large', '更新元数据超过大小限制。')
    }
    chunks.push(value)
  }
  try {
    return JSON.parse(Buffer.concat(chunks, total).toString('utf8'))
  } catch {
    throw new UpdaterError('invalid-json', '更新元数据不是有效 JSON。')
  }
}

async function sha256File(filePath) {
  const hash = crypto.createHash('sha256')
  const stream = fs.createReadStream(filePath)
  for await (const chunk of stream) hash.update(chunk)
  return hash.digest('hex')
}

function publicError(error) {
  if (error instanceof UpdaterError) return error
  if (error?.name === 'AbortError') return new UpdaterError('cancelled', '更新操作已取消。')
  return new UpdaterError('unexpected', '更新操作失败，请稍后重试。')
}

function createDesktopUpdater(options) {
  const {
    appVersion,
    platform,
    arch,
    isPackaged,
    tempRoot,
    shell,
    fetchImpl = globalThis.fetch,
    onState = () => {},
  } = options

  let candidate = null
  let verifiedFile = ''
  let operation = null
  let operationController = null
  let disposed = false
  let state = {
    phase: artifactNameFor('0.0.0', platform, arch) ? 'idle' : 'unsupported',
    currentVersion: appVersion,
    availableVersion: '',
    progress: 0,
    message: artifactNameFor('0.0.0', platform, arch) ? '尚未检查更新。' : '当前平台或架构暂不支持自动更新。',
    errorCode: '',
    automaticChecksEnabled: Boolean(isPackaged),
  }

  function publish(next) {
    state = { ...state, ...next }
    onState({ ...state })
    return { ...state }
  }

  function runOwned(work) {
    if (operation) return operation
    operationController = new AbortController()
    operation = work(operationController.signal).finally(() => {
      operation = null
      operationController = null
    })
    return operation
  }

  async function check() {
    if (disposed) return { ...state }
    if (!artifactNameFor('0.0.0', platform, arch)) return publish({ phase: 'unsupported' })
    return runOwned(async (ownerSignal) => {
      publish({ phase: 'checking', progress: 0, message: '正在检查稳定更新…', errorCode: '' })
      const timed = combinedAbortSignal(ownerSignal, METADATA_TIMEOUT_MS)
      try {
        const manifest = await fetchBoundedJson(fetchImpl, STABLE_MANIFEST_URL, MAX_MANIFEST_BYTES, timed.signal)
        const nextCandidate = validateManifest(manifest, platform, arch)
        if (!nextCandidate.supported) return publish({ phase: 'unsupported', message: '当前平台或架构暂不支持自动更新。' })
        if (compareStableVersions(appVersion, nextCandidate.version) >= 0) {
          candidate = null
          verifiedFile = ''
          return publish({ phase: 'up-to-date', availableVersion: '', message: `当前已是最新版本 v${appVersion}。` })
        }
        const releaseUrl = `${GITHUB_API_ORIGIN}/repos/${GITHUB_REPOSITORY}/releases/tags/${nextCandidate.tag}`
        const release = await fetchBoundedJson(fetchImpl, releaseUrl, MAX_RELEASE_BYTES, timed.signal)
        candidate = validateGitHubRelease(release, nextCandidate)
        verifiedFile = ''
        return publish({
          phase: 'available',
          availableVersion: candidate.version,
          progress: 0,
          message: `发现新版本 v${candidate.version}。下载后仍需你确认打开安装包。`,
        })
      } catch (error) {
        const safe = publicError(error)
        return publish({ phase: safe.code === 'cancelled' ? 'idle' : 'error', message: safe.message, errorCode: safe.code })
      } finally {
        timed.dispose()
      }
    })
  }

  async function download() {
    if (disposed) return { ...state }
    if (!candidate) return publish({ phase: 'error', message: '请先检查并确认有可用更新。', errorCode: 'no-candidate' })
    return runOwned(async (ownerSignal) => {
      const versionDirectory = path.join(tempRoot, 'aivo-updates', candidate.version)
      const partialPath = path.join(versionDirectory, `${candidate.artifact.name}.part`)
      const finalPath = path.join(versionDirectory, candidate.artifact.name)
      publish({ phase: 'downloading', progress: 0, message: `正在下载 v${candidate.version}…`, errorCode: '' })
      const timed = combinedAbortSignal(ownerSignal, DOWNLOAD_TIMEOUT_MS)
      let handle = null
      try {
        await fs.promises.mkdir(versionDirectory, { recursive: true, mode: 0o700 })
        await fs.promises.rm(partialPath, { force: true })
        await fs.promises.rm(finalPath, { force: true })
        const response = await fetchImpl(candidate.artifact.url, { redirect: 'error', signal: timed.signal })
        if (!response.ok || !response.body) throw new UpdaterError('download-unavailable', '无法下载更新包。')
        const contentLength = Number(response.headers.get('content-length') || 0)
        if (contentLength && contentLength !== candidate.artifact.size) {
          throw new UpdaterError('download-size-mismatch', '更新包响应大小与发布记录不一致。')
        }
        handle = await fs.promises.open(partialPath, 'wx', 0o600)
        const reader = response.body.getReader()
        const hash = crypto.createHash('sha256')
        let received = 0
        let lastProgress = -1
        while (true) {
          const { done, value } = await reader.read()
          if (done) break
          received += value.byteLength
          if (received > candidate.artifact.size || received > MAX_PACKAGE_BYTES) {
            await reader.cancel()
            throw new UpdaterError('download-too-large', '更新包超过发布记录的大小。')
          }
          hash.update(value)
          await handle.write(value)
          const progress = Math.min(99, Math.floor((received / candidate.artifact.size) * 100))
          if (progress !== lastProgress) {
            lastProgress = progress
            publish({ progress })
          }
        }
        await handle.sync()
        await handle.close()
        handle = null
        const digest = hash.digest('hex')
        if (received !== candidate.artifact.size || digest !== candidate.artifact.sha256) {
          throw new UpdaterError('download-integrity-mismatch', '更新包未通过大小或 SHA-256 校验。')
        }
        await fs.promises.rename(partialPath, finalPath)
        if (platform === 'linux') await fs.promises.chmod(finalPath, 0o700)
        verifiedFile = finalPath
        return publish({ phase: 'ready', progress: 100, message: `v${candidate.version} 已验证，可以打开更新包。` })
      } catch (error) {
        if (handle) await handle.close().catch(() => {})
        await fs.promises.rm(partialPath, { force: true }).catch(() => {})
        await fs.promises.rm(finalPath, { force: true }).catch(() => {})
        verifiedFile = ''
        const safe = publicError(error)
        return publish({ phase: safe.code === 'cancelled' ? 'available' : 'error', progress: 0, message: safe.message, errorCode: safe.code })
      } finally {
        timed.dispose()
      }
    })
  }

  async function install() {
    if (!candidate || !verifiedFile || state.phase !== 'ready') {
      return publish({ phase: 'error', message: '没有已验证的更新包可供打开。', errorCode: 'not-ready' })
    }
    try {
      const info = await fs.promises.stat(verifiedFile)
      if (!info.isFile() || info.size !== candidate.artifact.size || await sha256File(verifiedFile) !== candidate.artifact.sha256) {
        throw new UpdaterError('install-integrity-mismatch', '更新包在打开前未通过完整性复核。')
      }
      if (platform === 'linux') {
        shell.showItemInFolder(verifiedFile)
        return publish({ message: '已在文件管理器中显示验证后的 AppImage，请按你的安装方式完成替换。' })
      }
      const errorMessage = await shell.openPath(verifiedFile)
      if (errorMessage) throw new UpdaterError('installer-open-failed', '操作系统未能打开更新安装包。')
      return publish({ message: '已打开验证后的更新包，请按操作系统提示完成安装。' })
    } catch (error) {
      const safe = publicError(error)
      return publish({ phase: 'error', message: safe.message, errorCode: safe.code })
    }
  }

  function cancel() {
    operationController?.abort(new DOMException('Cancelled', 'AbortError'))
    return { ...state }
  }

  function dispose() {
    disposed = true
    cancel()
  }

  return {
    getState: () => ({ ...state }),
    check,
    download,
    install,
    cancel,
    dispose,
  }
}

module.exports = {
  GITHUB_REPOSITORY,
  MAX_PACKAGE_BYTES,
  R2_ORIGIN,
  STABLE_MANIFEST_URL,
  UpdaterError,
  artifactNameFor,
  compareStableVersions,
  createDesktopUpdater,
  validateGitHubRelease,
  validateManifest,
}
