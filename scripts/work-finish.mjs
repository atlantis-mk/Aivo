#!/usr/bin/env node

import { spawnSync } from 'node:child_process';
import { finishWork } from './work-lifecycle-lib.mjs';

const args = process.argv.slice(2).filter((argument) => argument !== '--');
const workId = args[0];
const checks = args.flatMap((argument, index) => argument === '--check' ? [args[index + 1]] : []).filter(Boolean);
if (!workId) {
  console.error('usage: pnpm work:finish -- <WORK-ID> [--check <docs:check|scripts:test|test:core|lint|build>]...');
  process.exit(2);
}

function runCheck(name) {
  return spawnSync('pnpm', [name], { cwd: process.cwd(), stdio: 'inherit' }).status === 0;
}

const commitResult = spawnSync('git', ['rev-parse', 'HEAD'], { cwd: process.cwd(), encoding: 'utf8' });
const commit = commitResult.status === 0 ? commitResult.stdout.trim() : null;
try {
  finishWork({ root: process.cwd(), workId, additionalChecks: checks, runCheck, commit });
  console.log(`completed ${workId}`);
} catch (error) {
  console.error(error.message);
  process.exit(1);
}
