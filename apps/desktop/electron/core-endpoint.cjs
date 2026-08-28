const CORE_READY_PREFIX = 'AIVO_CORE_READY '
const CORE_URL_ARGUMENT_PREFIX = '--aivo-core-url='
const MAX_CORE_READY_LINE_BYTES = 512

function normalizePackagedCoreURL(value) {
  if (typeof value !== 'string' || value.length === 0 || value.length > 256) {
    throw new Error('Packaged Core endpoint is missing or too long')
  }

  let parsed
  try {
    parsed = new URL(value)
  } catch {
    throw new Error('Packaged Core endpoint is not a valid URL')
  }

  const port = Number(parsed.port)
  if (
    parsed.protocol !== 'http:' ||
    parsed.hostname !== '127.0.0.1' ||
    !Number.isInteger(port) ||
    port < 1 ||
    port > 65535 ||
    parsed.username ||
    parsed.password ||
    parsed.pathname !== '/' ||
    parsed.search ||
    parsed.hash
  ) {
    throw new Error('Packaged Core endpoint must be an exact non-zero IPv4 loopback HTTP origin')
  }

  return parsed.origin
}

function parseCoreReadyLine(line) {
  if (typeof line !== 'string' || !line.startsWith(CORE_READY_PREFIX)) {
    return null
  }
  if (Buffer.byteLength(line, 'utf8') > MAX_CORE_READY_LINE_BYTES) {
    throw new Error('Packaged Core readiness record is too large')
  }

  let record
  try {
    record = JSON.parse(line.slice(CORE_READY_PREFIX.length))
  } catch {
    throw new Error('Packaged Core readiness record is invalid JSON')
  }
  if (
    !record ||
    typeof record !== 'object' ||
    Array.isArray(record) ||
    record.version !== 1 ||
    typeof record.url !== 'string' ||
    Object.keys(record).some((key) => key !== 'version' && key !== 'url')
  ) {
    throw new Error('Packaged Core readiness record has an unsupported shape')
  }
  return normalizePackagedCoreURL(record.url)
}

function coreURLArgument(coreUrl) {
  return `${CORE_URL_ARGUMENT_PREFIX}${normalizePackagedCoreURL(coreUrl)}`
}

function packagedCoreURLFromArguments(argv) {
  const matches = Array.isArray(argv)
    ? argv.filter((value) => typeof value === 'string' && value.startsWith(CORE_URL_ARGUMENT_PREFIX))
    : []
  if (matches.length === 0) return ''
  if (matches.length !== 1) {
    throw new Error('Packaged Core endpoint argument must appear exactly once')
  }
  return normalizePackagedCoreURL(matches[0].slice(CORE_URL_ARGUMENT_PREFIX.length))
}

module.exports = {
  CORE_READY_PREFIX,
  CORE_URL_ARGUMENT_PREFIX,
  MAX_CORE_READY_LINE_BYTES,
  coreURLArgument,
  normalizePackagedCoreURL,
  packagedCoreURLFromArguments,
  parseCoreReadyLine,
}
