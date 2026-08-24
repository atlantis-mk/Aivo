#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const localReferencePattern = /\b(?:href|src)=(['"])(.*?)\1/g

function isExternalReference(reference) {
  return /^(?:[a-z][a-z\d+.-]*:|\/\/|#)/i.test(reference)
}

export function verifyRendererBundle(outputDirectory) {
  const indexPath = path.join(outputDirectory, 'index.html')
  if (!fs.existsSync(indexPath)) {
    throw new Error(`Renderer entry point is missing at ${indexPath}`)
  }

  const html = fs.readFileSync(indexPath, 'utf8')
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
