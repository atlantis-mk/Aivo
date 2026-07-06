import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const rootDir = process.cwd()
const outputDir = path.join(rootDir, 'build', 'aivo-core')
const executable = process.platform === 'win32' ? 'aivo-core.exe' : 'aivo-core'
const outputPath = path.join(outputDir, executable)

fs.mkdirSync(outputDir, { recursive: true })

const result = spawnSync('go', ['build', '-o', outputPath, './cmd/aivo-core'], {
  cwd: path.join(rootDir, 'core'),
  stdio: 'inherit',
  shell: false,
})

if (result.status !== 0) {
  process.exit(result.status ?? 1)
}

if (process.platform !== 'win32') {
  fs.chmodSync(outputPath, 0o755)
}

console.log(`Built ${outputPath}`)
