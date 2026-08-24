import { spawnSync } from 'node:child_process'
import process from 'node:process'

const target = process.argv[2] || 'package'
const forwardedArgs = process.argv.slice(3)

const buildCore = spawnSync('npm', ['run', 'build:core'], {
  stdio: 'inherit',
  shell: process.platform === 'win32',
})

if (buildCore.status !== 0) {
  process.exit(buildCore.status ?? 1)
}

const packageDesktop = spawnSync('npm', ['--workspace', '@aivo/desktop', 'run', target, '--', ...forwardedArgs], {
  stdio: 'inherit',
  shell: process.platform === 'win32',
})

process.exit(packageDesktop.status ?? 1)
