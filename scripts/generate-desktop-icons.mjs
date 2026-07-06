import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import zlib from 'node:zlib'

const rootDir = process.cwd()
const iconDir = path.join(rootDir, 'apps', 'desktop', 'build')
const iconsetDir = path.join(iconDir, 'icon.iconset')
const baseSize = 1024

fs.mkdirSync(iconDir, { recursive: true })

function pngChunk(type, data) {
  const typeBuffer = Buffer.from(type)
  const crcInput = Buffer.concat([typeBuffer, data])
  const chunk = Buffer.alloc(12 + data.length)
  chunk.writeUInt32BE(data.length, 0)
  typeBuffer.copy(chunk, 4)
  data.copy(chunk, 8)
  chunk.writeUInt32BE(crc32(crcInput), 8 + data.length)
  return chunk
}

function crc32(buffer) {
  let crc = 0xffffffff
  for (const byte of buffer) {
    crc ^= byte
    for (let index = 0; index < 8; index += 1) {
      crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0)
    }
  }
  return (crc ^ 0xffffffff) >>> 0
}

function encodePng(width, height, rgba) {
  const rows = Buffer.alloc((width * 4 + 1) * height)
  for (let y = 0; y < height; y += 1) {
    const rowOffset = y * (width * 4 + 1)
    rows[rowOffset] = 0
    rgba.copy(rows, rowOffset + 1, y * width * 4, (y + 1) * width * 4)
  }
  const ihdr = Buffer.alloc(13)
  ihdr.writeUInt32BE(width, 0)
  ihdr.writeUInt32BE(height, 4)
  ihdr[8] = 8
  ihdr[9] = 6
  ihdr[10] = 0
  ihdr[11] = 0
  ihdr[12] = 0
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    pngChunk('IHDR', ihdr),
    pngChunk('IDAT', zlib.deflateSync(rows, { level: 9 })),
    pngChunk('IEND', Buffer.alloc(0)),
  ])
}

function pointInPolygon(x, y, points) {
  let inside = false
  for (let i = 0, j = points.length - 1; i < points.length; j = i, i += 1) {
    const xi = points[i][0]
    const yi = points[i][1]
    const xj = points[j][0]
    const yj = points[j][1]
    const intersects = yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi
    if (intersects) {
      inside = !inside
    }
  }
  return inside
}

function roundedRectCoverage(x, y, size, radius) {
  const min = 0
  const max = size
  if (x < min || x > max || y < min || y > max) return 0
  const cx = x < radius ? radius : x > max - radius ? max - radius : x
  const cy = y < radius ? radius : y > max - radius ? max - radius : y
  const distance = Math.hypot(x - cx, y - cy)
  return distance <= radius ? 1 : 0
}

function blendPixel(buffer, offset, source, alpha) {
  const sourceAlpha = (source[3] / 255) * alpha
  const destAlpha = buffer[offset + 3] / 255
  const outAlpha = sourceAlpha + destAlpha * (1 - sourceAlpha)
  if (outAlpha <= 0) return
  for (let channel = 0; channel < 3; channel += 1) {
    const src = source[channel] / 255
    const dst = buffer[offset + channel] / 255
    buffer[offset + channel] = Math.round(((src * sourceAlpha + dst * destAlpha * (1 - sourceAlpha)) / outAlpha) * 255)
  }
  buffer[offset + 3] = Math.round(outAlpha * 255)
}

function lerp(a, b, amount) {
  return a + (b - a) * amount
}

function gradientColor(x, y, size) {
  const amount = Math.min(1, Math.max(0, (x * 0.72 + y * 0.28) / size))
  return [
    Math.round(lerp(126, 71, amount)),
    Math.round(lerp(20, 191, amount)),
    Math.round(lerp(255, 255, amount)),
    255,
  ]
}

function renderIcon(size) {
  const data = Buffer.alloc(size * size * 4)
  const scale = size / baseSize
  const samples = size < 128 ? 4 : 3
  const radius = 220 * scale
  const bolt = [
    [558, 68],
    [220, 514],
    [448, 514],
    [363, 956],
    [807, 373],
    [566, 373],
    [665, 68],
  ].map(([x, y]) => [x * scale, y * scale])
  const shadow = bolt.map(([x, y]) => [x + 18 * scale, y + 24 * scale])

  for (let y = 0; y < size; y += 1) {
    for (let x = 0; x < size; x += 1) {
      let backgroundCoverage = 0
      let shadowCoverage = 0
      let boltCoverage = 0
      for (let sy = 0; sy < samples; sy += 1) {
        for (let sx = 0; sx < samples; sx += 1) {
          const px = x + (sx + 0.5) / samples
          const py = y + (sy + 0.5) / samples
          backgroundCoverage += roundedRectCoverage(px, py, size, radius)
          if (pointInPolygon(px, py, shadow)) shadowCoverage += 1
          if (pointInPolygon(px, py, bolt)) boltCoverage += 1
        }
      }
      const totalSamples = samples * samples
      const offset = (y * size + x) * 4
      const bgAlpha = backgroundCoverage / totalSamples
      if (bgAlpha > 0) {
        blendPixel(data, offset, gradientColor(x, y, size), bgAlpha)
      }
      if (shadowCoverage > 0) {
        blendPixel(data, offset, [42, 15, 110, 110], (shadowCoverage / totalSamples) * bgAlpha)
      }
      if (boltCoverage > 0) {
        blendPixel(data, offset, [255, 255, 255, 255], (boltCoverage / totalSamples) * bgAlpha)
      }
    }
  }
  return encodePng(size, size, data)
}

function writePng(name, size) {
  const filePath = path.join(iconDir, name)
  fs.writeFileSync(filePath, renderIcon(size))
  return filePath
}

function writeIco(images) {
  const header = Buffer.alloc(6)
  header.writeUInt16LE(0, 0)
  header.writeUInt16LE(1, 2)
  header.writeUInt16LE(images.length, 4)
  let offset = 6 + images.length * 16
  const entries = []
  for (const image of images) {
    const entry = Buffer.alloc(16)
    entry[0] = image.size >= 256 ? 0 : image.size
    entry[1] = image.size >= 256 ? 0 : image.size
    entry[2] = 0
    entry[3] = 0
    entry.writeUInt16LE(1, 4)
    entry.writeUInt16LE(32, 6)
    entry.writeUInt32LE(image.data.length, 8)
    entry.writeUInt32LE(offset, 12)
    offset += image.data.length
    entries.push(entry)
  }
  fs.writeFileSync(path.join(iconDir, 'icon.ico'), Buffer.concat([header, ...entries, ...images.map((image) => image.data)]))
}

const pngSizes = [16, 32, 48, 64, 128, 256, 512, 1024]
const pngBySize = new Map()
for (const size of pngSizes) {
  pngBySize.set(size, renderIcon(size))
}
fs.writeFileSync(path.join(iconDir, 'icon.png'), pngBySize.get(1024))
writeIco([16, 32, 48, 64, 128, 256].map((size) => ({ size, data: pngBySize.get(size) })))

if (process.platform === 'darwin') {
  fs.rmSync(iconsetDir, { recursive: true, force: true })
  fs.mkdirSync(iconsetDir, { recursive: true })
  const iconsetSizes = [
    ['icon_16x16.png', 16],
    ['icon_16x16@2x.png', 32],
    ['icon_32x32.png', 32],
    ['icon_32x32@2x.png', 64],
    ['icon_128x128.png', 128],
    ['icon_128x128@2x.png', 256],
    ['icon_256x256.png', 256],
    ['icon_256x256@2x.png', 512],
    ['icon_512x512.png', 512],
    ['icon_512x512@2x.png', 1024],
  ]
  for (const [name, size] of iconsetSizes) {
    fs.writeFileSync(path.join(iconsetDir, name), pngBySize.get(size))
  }
  const result = spawnSync('iconutil', ['-c', 'icns', iconsetDir, '-o', path.join(iconDir, 'icon.icns')], {
    stdio: 'inherit',
  })
  if (result.status !== 0) {
    process.exit(result.status ?? 1)
  }
}

console.log(`Generated desktop icons in ${path.relative(rootDir, iconDir)}`)
