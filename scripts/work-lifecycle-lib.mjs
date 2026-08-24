import fs from 'node:fs';
import path from 'node:path';
import { archiveWork } from './work-archive-lib.mjs';
import {
  currentSpecRevision,
  expectedTraceability,
  isWorkV2,
  parseChangeYamlText,
  replaceYamlScalar,
  SPEC_DELTAS,
  WORK_PROFILES,
} from './document-governance-lib.mjs';

export const ALLOWED_CHECKS = new Set(['docs:check', 'scripts:test', 'test:core', 'lint', 'build']);
export const ALLOWED_TYPES = new Set(['feature', 'bug', 'security', 'dependency', 'migration', 'technical_debt', 'governance']);

function changePath(root, workId, name = 'change.yaml') {
  if (!/^[A-Z]+-\d{4}-\d{3}[a-z0-9-]*$/.test(workId)) throw new Error(`invalid Work ID: ${workId}`);
  return path.join(root, 'changes', workId, name);
}

export function validateProfile({ profile, specDelta, type }) {
  if (!WORK_PROFILES.has(profile)) throw new Error(`legacy profile must be one of: ${[...WORK_PROFILES].join(', ')}`);
  if (!SPEC_DELTAS.has(specDelta)) throw new Error(`legacy spec_delta must be one of: ${[...SPEC_DELTAS].join(', ')}`);
  if (!ALLOWED_TYPES.has(type)) throw new Error(`type must be one of: ${[...ALLOWED_TYPES].join(', ')}`);
  if (profile === 'light' && specDelta === 'behavior_change') throw new Error('legacy Light Work cannot use spec_delta behavior_change');
  if (profile === 'light' && ['security', 'dependency', 'migration', 'governance'].includes(type)) {
    throw new Error(`legacy Work type ${type} requires profile controlled`);
  }
}

export function createWork({ root, workId, title, type, goal }) {
  if (!ALLOWED_TYPES.has(type)) throw new Error(`type must be one of: ${[...ALLOWED_TYPES].join(', ')}`);
  if (!title?.trim()) throw new Error('title is required');
  if (!goal?.trim()) throw new Error('goal is required');
  const directory = path.dirname(changePath(root, workId));
  if (fs.existsSync(directory)) throw new Error(`Work directory already exists: changes/${workId}`);
  const yaml = `schema: "2"
id: ${JSON.stringify(workId)}
title: ${JSON.stringify(title.trim())}
type: ${JSON.stringify(type)}
status: "Draft"
spec_revision: ${JSON.stringify(currentSpecRevision(root))}
target_release: null
goal: ${JSON.stringify(goal.trim())}
requirements: []
tests: []
adrs: []
context_refs: []
related_changes: []
boundaries: []
risks: []
next: []
`;
  fs.mkdirSync(directory);
  fs.writeFileSync(path.join(directory, 'change.yaml'), yaml);
  return directory;
}

export function startWork({ root, workId, accept = false }) {
  const yamlPath = changePath(root, workId);
  const original = fs.readFileSync(yamlPath, 'utf8');
  const change = parseChangeYamlText(original, path.relative(root, yamlPath));
  let nextStatus;
  if (isWorkV2(change)) {
    if (change.status !== 'Draft') throw new Error(`${workId}: status ${change.status} cannot start`);
    nextStatus = 'Active';
  } else {
    validateProfile({ profile: change.profile, specDelta: change.spec_delta, type: change.type });
    if (!['Draft', 'Accepted'].includes(change.status)) throw new Error(`${workId}: status ${change.status} cannot start`);
    if (change.profile === 'controlled' && change.status === 'Draft' && !accept) {
      throw new Error(`${workId}: Draft legacy Controlled Work requires --accept`);
    }
    nextStatus = 'Implementing';
  }
  fs.writeFileSync(yamlPath, replaceYamlScalar(original, 'status', nextStatus));
  return { yamlPath, original, status: nextStatus };
}

export function selectApplicableChecks(change, additional = []) {
  for (const check of additional) if (!ALLOWED_CHECKS.has(check)) throw new Error(`unsupported check: ${check}`);
  const surfaces = [...(change.boundaries ?? []), ...(change.affected_surfaces ?? [])].join(' ').toLowerCase();
  const checks = new Set(['docs:check']);
  if (/(documentation|governance|script|release|github|workflow)/.test(surfaces)) checks.add('scripts:test');
  if (/(core|domain|application|persistence|transport|migration)/.test(surfaces)) checks.add('test:core');
  if (/(desktop|renderer|electron|preload|ui)/.test(surfaces)) {
    checks.add('lint');
    checks.add('build');
  }
  for (const check of additional) checks.add(check);
  return [...checks];
}

function runChecks(workId, checks, runCheck) {
  const results = [];
  for (const name of checks) {
    const started = Date.now();
    const passed = runCheck(name);
    results.push({ name, status: passed ? 'passed' : 'failed', duration_ms: Date.now() - started });
    if (!passed) throw new Error(`${workId}: check failed: pnpm ${name}`);
  }
  return results;
}

function finishV2Work({ root, workId, yamlPath, originalYaml, checks, runCheck }) {
  const tracePath = path.join(root, 'docs', '08-traceability.md');
  const priorTrace = fs.readFileSync(tracePath);
  const results = runChecks(workId, checks, runCheck);
  try {
    fs.writeFileSync(yamlPath, replaceYamlScalar(originalYaml, 'status', 'Done'));
    fs.writeFileSync(tracePath, expectedTraceability(root));
    if (!runCheck('docs:check')) throw new Error(`${workId}: post-completion documentation validation failed`);
    return { version: 2, work_id: workId, status: 'Done', checks: results.map(({ name, status }) => ({ name, status })) };
  } catch (error) {
    fs.writeFileSync(yamlPath, originalYaml);
    fs.writeFileSync(tracePath, priorTrace);
    throw error;
  }
}

function finishLegacyWork({ root, workId, yamlPath, originalYaml, checks, runCheck, now, commit }) {
  const verificationPath = changePath(root, workId, 'verification.json');
  const archivePath = path.join(root, 'changes', 'archive.json');
  const tracePath = path.join(root, 'docs', '08-traceability.md');
  const results = runChecks(workId, checks, runCheck);
  const priorVerification = fs.existsSync(verificationPath) ? fs.readFileSync(verificationPath) : null;
  const priorArchive = fs.readFileSync(archivePath);
  const priorTrace = fs.readFileSync(tracePath);
  try {
    const evidence = {
      version: 1,
      work_id: workId,
      verified_at: now().toISOString(),
      commit,
      checks: results,
    };
    fs.writeFileSync(verificationPath, `${JSON.stringify(evidence, null, 2)}\n`);
    fs.writeFileSync(yamlPath, replaceYamlScalar(originalYaml, 'status', 'Verified'));
    archiveWork({ root, workId, status: 'Verified', archivedAt: evidence.verified_at });
    fs.writeFileSync(tracePath, expectedTraceability(root));
    if (!runCheck('docs:check')) throw new Error(`${workId}: post-seal documentation validation failed`);
    return evidence;
  } catch (error) {
    fs.writeFileSync(yamlPath, originalYaml);
    fs.writeFileSync(archivePath, priorArchive);
    fs.writeFileSync(tracePath, priorTrace);
    if (priorVerification === null) fs.rmSync(verificationPath, { force: true });
    else fs.writeFileSync(verificationPath, priorVerification);
    throw error;
  }
}

export function finishWork({ root, workId, additionalChecks = [], runCheck, now = () => new Date(), commit = null }) {
  const yamlPath = changePath(root, workId);
  const originalYaml = fs.readFileSync(yamlPath, 'utf8');
  const change = parseChangeYamlText(originalYaml, path.relative(root, yamlPath));
  const checks = selectApplicableChecks(change, additionalChecks);
  if (isWorkV2(change)) {
    if (change.status !== 'Active') throw new Error(`${workId}: only Active schema-v2 Work can finish`);
    return finishV2Work({ root, workId, yamlPath, originalYaml, checks, runCheck });
  }
  if (change.status !== 'Implementing') throw new Error(`${workId}: only Implementing legacy Work can finish`);
  validateProfile({ profile: change.profile, specDelta: change.spec_delta, type: change.type });
  return finishLegacyWork({ root, workId, yamlPath, originalYaml, checks, runCheck, now, commit });
}
