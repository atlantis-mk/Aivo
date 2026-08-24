#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

function exactlyOne(entries, predicate, label) {
  const matches = entries.filter(predicate)
  if (matches.length !== 1) {
    throw new Error(`Expected exactly one ${label}, found ${matches.length}.`)
  }
  return matches[0]
}

export function selectNativeArtifacts(input, platform = process.platform) {
  const entries = fs.readdirSync(input).sort()
  if (platform === 'darwin') {
    return {
      dmg: path.join(input, exactlyOne(entries, (entry) => entry.endsWith('.dmg'), 'macOS DMG')),
      zip: path.join(input, exactlyOne(entries, (entry) => entry.endsWith('.zip'), 'macOS ZIP')),
    }
  }
  if (platform === 'win32') {
    return {
      installer: path.join(input, exactlyOne(entries, (entry) => entry.endsWith('.exe'), 'Windows installer')),
    }
  }
  if (platform === 'linux') {
    return {
      appImage: path.join(input, exactlyOne(entries, (entry) => entry.endsWith('.AppImage'), 'Linux AppImage')),
    }
  }
  throw new Error(`Unsupported native package platform: ${platform}`)
}

function run(command, args, options = {}) {
  const result = spawnSync(command, args, { stdio: 'inherit', shell: false, ...options })
  if (result.error) throw result.error
  if (result.status !== 0) {
    throw new Error(`${command} exited with status ${result.status ?? 'unknown'}.`)
  }
}

function findExecutable(root, expectedName) {
  const pending = [root]
  while (pending.length > 0) {
    const directory = pending.pop()
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      const candidate = path.join(directory, entry.name)
      if (entry.isDirectory()) pending.push(candidate)
      if (entry.isFile() && entry.name.toLowerCase() === expectedName.toLowerCase()) return candidate
    }
  }
  return ''
}

function removeTemporary(directory) {
  fs.rmSync(directory, {
    recursive: true,
    force: true,
    maxRetries: process.platform === 'win32' ? 10 : 2,
    retryDelay: 500,
  })
}

function verifyMac(input) {
  const { dmg, zip } = selectNativeArtifacts(input, 'darwin')
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'aivo-native-mac-'))
  const mountPoint = path.join(temporary, 'mounted')
  fs.mkdirSync(mountPoint)
  let mounted = false
  try {
    run('hdiutil', ['verify', dmg])
    run('hdiutil', ['attach', dmg, '-nobrowse', '-readonly', '-mountpoint', mountPoint])
    mounted = true
    if (!fs.existsSync(path.join(mountPoint, 'Aivo.app'))) {
      throw new Error('Mounted DMG does not contain Aivo.app.')
    }
    run('unzip', ['-t', zip])
  } finally {
    if (mounted) run('hdiutil', ['detach', mountPoint])
    removeTemporary(temporary)
  }
}

function verifyWindows(input) {
  const { installer } = selectNativeArtifacts(input, 'win32')
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'aivo-native-windows-'))
  const installDirectory = path.join(temporary, 'installed')
  try {
    run(installer, ['/S', `/D=${installDirectory}`])
    const executable = findExecutable(installDirectory, 'aivo.exe')
    if (!executable) throw new Error('NSIS installation did not produce aivo.exe.')
  } finally {
    removeTemporary(temporary)
  }
}

function verifyLinux(input) {
  const { appImage } = selectNativeArtifacts(input, 'linux')
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'aivo-native-linux-'))
  try {
    fs.chmodSync(appImage, 0o755)
    run(appImage, ['--appimage-extract'], { cwd: temporary })
    if (!fs.existsSync(path.join(temporary, 'squashfs-root', 'AppRun'))) {
      throw new Error('AppImage extraction did not produce AppRun.')
    }
  } finally {
    removeTemporary(temporary)
  }
}

export function verifyNativePackage(input, platform = process.platform) {
  if (platform === 'darwin') return verifyMac(input)
  if (platform === 'win32') return verifyWindows(input)
  if (platform === 'linux') return verifyLinux(input)
  throw new Error(`Unsupported native package platform: ${platform}`)
}

function parseInput(argv) {
  const index = argv.indexOf('--input')
  if (index === -1 || !argv[index + 1]) throw new Error('usage: verify-native-package.mjs --input <directory>')
  return path.resolve(argv[index + 1])
}

if (fileURLToPath(import.meta.url) === path.resolve(process.argv[1] ?? '')) {
  try {
    const input = parseInput(process.argv.slice(2))
    verifyNativePackage(input)
    console.log(`Native package acceptance passed for ${process.platform}/${process.arch}.`)
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error))
    process.exit(1)
  }
}
