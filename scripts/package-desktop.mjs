import { spawnSync } from 'node:child_process'
import process from 'node:process'

const target = process.argv[2] || 'package'
const forwardedArgs = process.argv.slice(3)
const packageEnvironment = { ...process.env }
const optionalSigningVariables = [
  'CSC_LINK',
  'CSC_KEY_PASSWORD',
  'APPLE_API_KEY',
  'APPLE_API_KEY_ID',
  'APPLE_API_ISSUER',
  'APPLE_ID',
  'APPLE_APP_SPECIFIC_PASSWORD',
  'APPLE_TEAM_ID',
]

for (const name of optionalSigningVariables) {
  if (!packageEnvironment[name]?.trim()) {
    delete packageEnvironment[name]
  }
}
if (process.platform === 'darwin' && !packageEnvironment.CSC_LINK) {
  packageEnvironment.CSC_IDENTITY_AUTO_DISCOVERY = 'false'
}

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
  env: packageEnvironment,
})

process.exit(packageDesktop.status ?? 1)
