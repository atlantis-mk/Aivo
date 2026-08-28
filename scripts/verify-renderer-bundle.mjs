#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const localReferencePattern = /\b(?:href|src)=(['"])(.*?)\1/g
const allowedConnectSources = new Set([
  "'self'",
  'http://127.0.0.1:*',
  'ws://127.0.0.1:*',
])

function isExternalReference(reference) {
  return /^(?:[a-z][a-z\d+.-]*:|\/\/|#)/i.test(reference)
}

export function verifyRendererBundle(outputDirectory) {
  const indexPath = path.join(outputDirectory, 'index.html')
  if (!fs.existsSync(indexPath)) {
    throw new Error(`Renderer entry point is missing at ${indexPath}`)
  }

  const html = fs.readFileSync(indexPath, 'utf8')
  verifyRendererConnectCSP(html)
  const references = [...html.matchAll(localReferencePattern)].map((match) => match[2])
  if (references.length === 0) {
    throw new Error('Renderer entry point contains no local asset references.')
  }

  for (const reference of references) {
    if (isExternalReference(reference)) continue
    if (reference.startsWith('/')) {
      throw new Error(`Renderer asset must be relative for Electron file loading: ${reference}`)
    }

    const pathname = decodeURIComponent(reference.split(/[?#]/, 1)[0])
    const assetPath = path.resolve(outputDirectory, pathname)
    const relativePath = path.relative(outputDirectory, assetPath)
    if (relativePath.startsWith('..') || path.isAbsolute(relativePath)) {
      throw new Error(`Renderer asset escapes the build directory: ${reference}`)
    }
    if (!fs.existsSync(assetPath)) {
      throw new Error(`Renderer asset is missing from the build output: ${reference}`)
    }
  }

  return references
}

export function verifyRendererConnectCSP(html) {
  const meta = [...html.matchAll(/<meta\b[^>]*>/gi)]
    .map((match) => match[0])
    .find((tag) => /\bhttp-equiv\s*=\s*(['"])Content-Security-Policy\1/i.test(tag))
  if (!meta) {
    throw new Error('Renderer Content Security Policy meta tag is missing.')
  }
  const content = meta.match(/\bcontent\s*=\s*(['"])(.*?)\1/i)?.[2] ?? ''
  const connectDirective = content
    .split(';')
    .map((directive) => directive.trim().split(/\s+/))
    .find(([name]) => name === 'connect-src')
  if (!connectDirective) {
    throw new Error('Renderer Content Security Policy connect-src directive is missing.')
  }
  const sources = connectDirective.slice(1)
  if (
    sources.length !== allowedConnectSources.size ||
    sources.some((source) => !allowedConnectSources.has(source))
  ) {
    throw new Error(
      "Renderer connect-src must allow only 'self' and dynamic HTTP/WebSocket ports on 127.0.0.1.",
    )
  }
}

if (fileURLToPath(import.meta.url) === path.resolve(process.argv[1] ?? '')) {
  try {
    const outputDirectory = process.argv[2]
      ? path.resolve(process.argv[2])
      : path.resolve(process.cwd(), 'dist')
    const references = verifyRendererBundle(outputDirectory)
    console.log(`Renderer bundle verification passed for ${references.length} asset references.`)
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error))
    process.exit(1)
  }
}
