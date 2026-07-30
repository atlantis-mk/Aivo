#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import { archiveWork, COMPLETED_STATUSES, readArchiveManifest } from './work-archive-lib.mjs';

const ROOT = process.cwd();
const args = process.argv.slice(2).filter((argument) => argument !== '--');
const archiveAll = args.includes('--all');
const dateIndex = args.indexOf('--date');
const archivedAt = dateIndex === -1 ? new Date().toISOString() : args[dateIndex + 1];
const positional = args.filter((argument, index) => argument !== '--all'
  && argument !== '--date'
  && (dateIndex === -1 || index !== dateIndex + 1));

if ((!archiveAll && positional.length !== 1) || (archiveAll && positional.length !== 0) || (dateIndex !== -1 && !archivedAt)) {
  console.error('usage: pnpm work:archive -- <WORK-ID> [--date <ISO-8601>] | --all [--date <ISO-8601>]');
  process.exit(2);
}

function readStatus(workId) {
  const yamlPath = path.join(ROOT, 'changes', workId, 'change.yaml');
  if (!fs.existsSync(yamlPath)) throw new Error(`Work YAML does not exist: changes/${workId}/change.yaml`);
  const match = fs.readFileSync(yamlPath, 'utf8').match(/^status: "([^"]+)"$/m);
  if (!match) throw new Error(`${yamlPath}: missing quoted status`);
  return match[1];
}

let workIds = positional;
if (archiveAll) {
  const existing = new Set(readArchiveManifest(ROOT).archives.map((entry) => entry.work_id));
  workIds = fs.readdirSync(path.join(ROOT, 'changes'), { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && entry.name !== '_template')
    .map((entry) => entry.name)
    .filter((workId) => COMPLETED_STATUSES.has(readStatus(workId)) && !existing.has(workId))
    .sort();
}

try {
  for (const workId of workIds) {
    const status = readStatus(workId);
    archiveWork({ root: ROOT, workId, status, archivedAt });
    console.log(`archived ${workId} (${status})`);
  }
  if (workIds.length === 0) console.log('no completed Work needs archiving');
} catch (error) {
  console.error(error.message);
  process.exit(1);
}
