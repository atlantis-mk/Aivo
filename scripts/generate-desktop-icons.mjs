import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'
import zlib from 'node:zlib'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const rootDir = path.resolve(scriptDir, '..')
const desktopDir = path.join(rootDir, 'apps', 'desktop')
const sourceIconPath = path.join(desktopDir, 'assets', 'app-icon.png')
const iconDir = path.join(desktopDir, 'build')
const iconsetDir = path.join(iconDir, 'icon.iconset')
const faviconPath = path.join(desktopDir, 'public', 'favicon.png')
const baseSize = 1024
const pngSignature = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])

fs.mkdirSync(iconDir, { recursive: true })

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
    pngSignature,
    pngChunk('IHDR', ihdr),
    pngChunk('IDAT', zlib.deflateSync(rows, { level: 9 })),
    pngChunk('IEND', Buffer.alloc(0)),
  ])
}

function paethPredictor(left, above, upperLeft) {
  const prediction = left + above - upperLeft
  const leftDistance = Math.abs(prediction - left)
  const aboveDistance = Math.abs(prediction - above)
  const upperLeftDistance = Math.abs(prediction - upperLeft)
  if (leftDistance <= aboveDistance && leftDistance <= upperLeftDistance) return left
  if (aboveDistance <= upperLeftDistance) return above
  return upperLeft
}

function decodePng(png) {
  if (!png.subarray(0, pngSignature.length).equals(pngSignature)) {
    throw new Error(`App icon source is not a PNG: ${sourceIconPath}`)
  }

  let width
  let height
  let bitDepth
  let colorType
  let interlaceMethod
  const imageData = []
  for (let offset = pngSignature.length; offset < png.length; ) {
    const length = png.readUInt32BE(offset)
    const type = png.toString('ascii', offset + 4, offset + 8)
    const data = png.subarray(offset + 8, offset + 8 + length)
    if (type === 'IHDR') {
      width = data.readUInt32BE(0)
      height = data.readUInt32BE(4)
      bitDepth = data[8]
      colorType = data[9]
      interlaceMethod = data[12]
    } else if (type === 'IDAT') {
      imageData.push(data)
    } else if (type === 'IEND') {
      break
    }
    offset += length + 12
  }

  if (!width || !height || bitDepth !== 8 || ![2, 6].includes(colorType) || interlaceMethod !== 0) {
    throw new Error('App icon source must be a non-interlaced 8-bit RGB or RGBA PNG')
  }

  const bytesPerPixel = colorType === 6 ? 4 : 3
  const stride = width * bytesPerPixel
  const inflated = zlib.inflateSync(Buffer.concat(imageData))
  const expectedLength = (stride + 1) * height
  if (inflated.length !== expectedLength) {
    throw new Error(`Unexpected PNG data length: expected ${expectedLength}, received ${inflated.length}`)
  }

  const raw = Buffer.alloc(stride * height)
  for (let y = 0; y < height; y += 1) {
    const inputRowOffset = y * (stride + 1)
    const outputRowOffset = y * stride
    const filter = inflated[inputRowOffset]
    for (let x = 0; x < stride; x += 1) {
      const value = inflated[inputRowOffset + x + 1]
      const left = x >= bytesPerPixel ? raw[outputRowOffset + x - bytesPerPixel] : 0
      const above = y > 0 ? raw[outputRowOffset + x - stride] : 0
      const upperLeft = y > 0 && x >= bytesPerPixel ? raw[outputRowOffset + x - stride - bytesPerPixel] : 0
      let reconstructed
      switch (filter) {
        case 0:
          reconstructed = value
          break
        case 1:
          reconstructed = value + left
          break
        case 2:
          reconstructed = value + above
          break
        case 3:
          reconstructed = value + Math.floor((left + above) / 2)
          break
        case 4:
          reconstructed = value + paethPredictor(left, above, upperLeft)
          break
        default:
          throw new Error(`Unsupported PNG filter type: ${filter}`)
      }
      raw[outputRowOffset + x] = reconstructed & 0xff
    }
  }

  const rgba = Buffer.alloc(width * height * 4)
  for (let pixel = 0; pixel < width * height; pixel += 1) {
    const sourceOffset = pixel * bytesPerPixel
    const targetOffset = pixel * 4
    rgba[targetOffset] = raw[sourceOffset]
    rgba[targetOffset + 1] = raw[sourceOffset + 1]
    rgba[targetOffset + 2] = raw[sourceOffset + 2]
    rgba[targetOffset + 3] = colorType === 6 ? raw[sourceOffset + 3] : 255
  }

  return { width, height, rgba }
}

function resizeRgba(source, sourceWidth, sourceHeight, targetWidth, targetHeight) {
  if (sourceWidth === targetWidth && sourceHeight === targetHeight) return Buffer.from(source)

  const target = Buffer.alloc(targetWidth * targetHeight * 4)
  const scaleX = sourceWidth / targetWidth
  const scaleY = sourceHeight / targetHeight
  for (let targetY = 0; targetY < targetHeight; targetY += 1) {
    const sourceTop = targetY * scaleY
    const sourceBottom = (targetY + 1) * scaleY
    const firstSourceY = Math.floor(sourceTop)
    const lastSourceY = Math.ceil(sourceBottom)
    for (let targetX = 0; targetX < targetWidth; targetX += 1) {
      const sourceLeft = targetX * scaleX
      const sourceRight = (targetX + 1) * scaleX
      const firstSourceX = Math.floor(sourceLeft)
      const lastSourceX = Math.ceil(sourceRight)
      let alphaWeight = 0
      let totalWeight = 0
      let red = 0
      let green = 0
      let blue = 0

      for (let sourceY = firstSourceY; sourceY < lastSourceY; sourceY += 1) {
        const yWeight = Math.min(sourceBottom, sourceY + 1) - Math.max(sourceTop, sourceY)
        for (let sourceX = firstSourceX; sourceX < lastSourceX; sourceX += 1) {
          const xWeight = Math.min(sourceRight, sourceX + 1) - Math.max(sourceLeft, sourceX)
          const weight = xWeight * yWeight
          const sourceOffset = (sourceY * sourceWidth + sourceX) * 4
          const pixelAlphaWeight = weight * (source[sourceOffset + 3] / 255)
          red += source[sourceOffset] * pixelAlphaWeight
          green += source[sourceOffset + 1] * pixelAlphaWeight
          blue += source[sourceOffset + 2] * pixelAlphaWeight
          alphaWeight += pixelAlphaWeight
          totalWeight += weight
        }
      }

      const targetOffset = (targetY * targetWidth + targetX) * 4
      if (alphaWeight > 0) {
        target[targetOffset] = Math.round(red / alphaWeight)
        target[targetOffset + 1] = Math.round(green / alphaWeight)
        target[targetOffset + 2] = Math.round(blue / alphaWeight)
      }
      target[targetOffset + 3] = Math.round((alphaWeight / totalWeight) * 255)
    }
  }
  return target
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

const sourceIcon = decodePng(fs.readFileSync(sourceIconPath))
if (sourceIcon.width !== sourceIcon.height || sourceIcon.width < baseSize) {
  throw new Error(`App icon source must be square and at least ${baseSize}x${baseSize}`)
}

const baseRgba = resizeRgba(sourceIcon.rgba, sourceIcon.width, sourceIcon.height, baseSize, baseSize)
const pngSizes = [16, 32, 48, 64, 128, 256, 512, 1024]
const pngBySize = new Map()
for (const size of pngSizes) {
  const rgba = resizeRgba(baseRgba, baseSize, baseSize, size, size)
  pngBySize.set(size, encodePng(size, size, rgba))
}

fs.writeFileSync(path.join(iconDir, 'icon.png'), pngBySize.get(1024))
fs.writeFileSync(faviconPath, pngBySize.get(48))
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

console.log(`Generated desktop icons from ${path.relative(rootDir, sourceIconPath)}`)
