import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { expectedTraceability, parseChangeYamlText } from './document-governance-lib.mjs';
import { createWork, finishWork, startWork } from './work-lifecycle-lib.mjs';

function fixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'aivo-work-lifecycle-'));
  fs.mkdirSync(path.join(root, 'docs'), { recursive: true });
  fs.mkdirSync(path.join(root, 'changes'), { recursive: true });
  fs.writeFileSync(path.join(root, 'docs', '00-spec-index.md'), '- Specification revision: `0.1.4-active`\n');
  fs.writeFileSync(path.join(root, 'docs', '03-functional-requirements.md'), '### REQ-TEST-001 Test behavior\n\nAcceptance: `AT-TEST-001`.\n');
  fs.writeFileSync(path.join(root, 'changes', 'archive.json'), '{"version":1,"archives":[]}\n');
  fs.writeFileSync(path.join(root, 'docs', '08-traceability.md'), expectedTraceability(root));
  return root;
}

test('creates one-file schema-v2 Work with only durable coordination fields', () => {
  const root = fixture();
  createWork({
    root,
    workId: 'CHG-2026-901-minimal-test',
    title: 'Preserve a decision',
    type: 'feature',
    goal: 'Carry an unresolved product decision into another task',
  });
  const directory = path.join(root, 'changes', 'CHG-2026-901-minimal-test');
  assert.deepEqual(fs.readdirSync(directory), ['change.yaml']);
  const yaml = parseChangeYamlText(fs.readFileSync(path.join(directory, 'change.yaml'), 'utf8'));
  assert.equal(yaml.schema, '2');
  assert.equal(yaml.status, 'Draft');
  assert.equal(yaml.spec_revision, '0.1.4-active');
  assert.equal('profile' in yaml, false);
  assert.equal('spec_delta' in yaml, false);
  assert.throws(() => createWork({ root, workId: 'CHG-2026-902-no-goal', title: 'No goal', type: 'feature', goal: '' }), /goal is required/);
});

test('starts schema-v2 Work with one explicit Draft-to-Active transition', () => {
  const root = fixture();
  createWork({ root, workId: 'CHG-2026-903-start', title: 'Start', type: 'governance', goal: 'Preserve an approved boundary change' });
  const result = startWork({ root, workId: 'CHG-2026-903-start' });
  assert.equal(result.status, 'Active');
  assert.match(fs.readFileSync(path.join(root, 'changes', 'CHG-2026-903-start', 'change.yaml'), 'utf8'), /status: "Active"/);
});

test('finishes schema-v2 Work without evidence files or archive hashes', () => {
  const root = fixture();
  createWork({ root, workId: 'CHG-2026-904-finish', title: 'Finish', type: 'governance', goal: 'Complete a governance boundary change' });
  startWork({ root, workId: 'CHG-2026-904-finish' });
  fs.writeFileSync(path.join(root, 'docs', '08-traceability.md'), expectedTraceability(root));
  const calls = [];
  const result = finishWork({
    root,
    workId: 'CHG-2026-904-finish',
    runCheck: (name) => { calls.push(name); return true; },
  });
  assert.equal(result.status, 'Done');
  assert.deepEqual(calls, ['docs:check', 'docs:check']);
  const directory = path.join(root, 'changes', 'CHG-2026-904-finish');
  assert.deepEqual(fs.readdirSync(directory), ['change.yaml']);
  assert.match(fs.readFileSync(path.join(directory, 'change.yaml'), 'utf8'), /status: "Done"/);
  assert.equal(JSON.parse(fs.readFileSync(path.join(root, 'changes', 'archive.json'), 'utf8')).archives.length, 0);
});

test('rolls back schema-v2 status and Traceability after post-completion failure', () => {
  const root = fixture();
  createWork({ root, workId: 'CHG-2026-905-rollback', title: 'Rollback', type: 'governance', goal: 'Exercise rollback' });
  startWork({ root, workId: 'CHG-2026-905-rollback' });
  fs.writeFileSync(path.join(root, 'docs', '08-traceability.md'), expectedTraceability(root));
  let call = 0;
  assert.throws(() => finishWork({
    root,
    workId: 'CHG-2026-905-rollback',
    runCheck: () => ++call === 1,
  }), /post-completion/);
  assert.match(fs.readFileSync(path.join(root, 'changes', 'CHG-2026-905-rollback', 'change.yaml'), 'utf8'), /status: "Active"/);
  assert.equal(JSON.parse(fs.readFileSync(path.join(root, 'changes', 'archive.json'), 'utf8')).archives.length, 0);
});

test('keeps the legacy completion and archive path compatible', () => {
  const root = fixture();
  const workId = 'CHG-2026-906-legacy';
  const directory = path.join(root, 'changes', workId);
  fs.mkdirSync(directory);
  fs.writeFileSync(path.join(directory, 'change.yaml'), `id: "${workId}"
title: "Legacy"
type: "governance"
profile: "controlled"
spec_delta: "behavior_change"
status: "Implementing"
spec_revision: "0.1.3-active"
target_release: null
requirements: []
tests: []
adrs: []
context_refs: []
related_changes: []
affected_surfaces: []
affected_formats: []
affected_versions: []
security_impact: "none"
migration: "none"
supersedes: []
`);
  fs.writeFileSync(path.join(root, 'docs', '08-traceability.md'), expectedTraceability(root));
  finishWork({
    root,
    workId,
    runCheck: () => true,
    now: () => new Date('2026-08-25T00:00:00.000Z'),
    commit: 'abc123',
  });
  assert.match(fs.readFileSync(path.join(directory, 'change.yaml'), 'utf8'), /status: "Verified"/);
  assert.equal(fs.existsSync(path.join(directory, 'verification.json')), true);
  assert.equal(JSON.parse(fs.readFileSync(path.join(root, 'changes', 'archive.json'), 'utf8')).archives.length, 1);
});
