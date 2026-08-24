#!/usr/bin/env node

import fs from 'node:fs';
import { spawnSync } from 'node:child_process';
import { startWork } from './work-lifecycle-lib.mjs';

const args = process.argv.slice(2).filter((argument) => argument !== '--');
const workId = args.find((argument) => !argument.startsWith('--'));
if (!workId) {
  console.error('usage: pnpm work:start -- <WORK-ID> [--accept-for-legacy-work]');
  process.exit(2);
}

let mutation;
try {
  mutation = startWork({ root: process.cwd(), workId, accept: args.includes('--accept-for-legacy-work') });
  const result = spawnSync('pnpm', ['docs:trace'], { cwd: process.cwd(), stdio: 'inherit' });
  if (result.status !== 0) throw new Error('pnpm docs:trace failed');
  console.log(`started ${workId}`);
} catch (error) {
  if (mutation) fs.writeFileSync(mutation.yamlPath, mutation.original);
  console.error(error.message);
  process.exit(1);
}
