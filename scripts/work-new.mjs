#!/usr/bin/env node

import { createWork } from './work-lifecycle-lib.mjs';

const args = process.argv.slice(2).filter((argument) => argument !== '--');
const workId = args[0];
function option(name) {
  const index = args.indexOf(name);
  return index === -1 ? null : args[index + 1];
}

if (!workId || !option('--title') || !option('--type') || !option('--goal')) {
  console.error('usage: pnpm work:new -- <WORK-ID> --title <title> --type <type> --goal <cross-task-or-controlled-boundary>');
  process.exit(2);
}

try {
  createWork({
    root: process.cwd(),
    workId,
    title: option('--title'),
    type: option('--type'),
    goal: option('--goal'),
  });
  console.log(`created changes/${workId}`);
} catch (error) {
  console.error(error.message);
  process.exit(1);
}
