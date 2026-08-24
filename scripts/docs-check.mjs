import fs from 'node:fs';
import path from 'node:path';
import { validateArchiveManifest } from './work-archive-lib.mjs';
import { expectedTraceability, isWorkV2, parseChangeYamlText, SPEC_DELTAS, WORK_PROFILES } from './document-governance-lib.mjs';

const ROOT = process.cwd();
const DOC_DIRS = ['docs', 'specs', 'adr', 'changes', 'releases'];
const LEGACY_STATUSES = new Set(['Draft', 'Accepted', 'Implementing', 'Verified', 'Released', 'Rejected']);
const V2_STATUSES = new Set(['Draft', 'Active', 'Done', 'Rejected']);
const ALLOWED_TYPES = new Set(['feature', 'bug', 'security', 'dependency', 'migration', 'technical_debt', 'governance']);
const CURRENT_SPEC_REVISION = '0.1.4-active';
const SPEC_REVISION_PATTERN = /^0\.1\.(\d+)-active$/;

function walk(directory) {
  if (!fs.existsSync(directory)) return [];
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const target = path.join(directory, entry.name);
    return entry.isDirectory() ? walk(target) : [target.split(path.sep).join('/')];
  });
}

function setDifference(left, right) {
  return [...left].filter((value) => !right.has(value));
}

function parseChangeYaml(file) {
  return parseChangeYamlText(fs.readFileSync(file, 'utf8'), file);
}

function validateExplicitPaths(markdownFiles) {
  const missing = [];
  const rootPrefixes = /^(docs|specs|adr|changes|releases|openspec|apps|core|scripts|\.github)\//;
  for (const file of markdownFiles) {
    const text = fs.readFileSync(file, 'utf8');
    for (const match of text.matchAll(/`([^`\n]+)`/g)) {
      const token = match[1].replace(/[：，。；、,.;:]$/, '');
      if (/^(https?:|N\/A$)|[<>{}*]|^releases\/vX|^\/api\//.test(token)) continue;
      let candidate = null;
      if (token.startsWith('../') || token.startsWith('./')) candidate = path.resolve(path.dirname(file), token.split('#')[0]);
      else if (rootPrefixes.test(token)) candidate = path.resolve(ROOT, token.split('#')[0]);
      else if (file === 'docs/00-spec-index.md' && /^\d{2}-.*\.md$/.test(token)) candidate = path.resolve(ROOT, 'docs', token);
      else if ((file === 'releases/README.md' || file === 'changes/README.md') && token.startsWith('_template')) candidate = path.resolve(path.dirname(file), token);
      if (candidate && !fs.existsSync(candidate)) missing.push(`${file}: ${token}`);
    }
  }
  return missing;
}

function validateContextRef(reference, file) {
  const separator = reference.indexOf('#');
  const referencedPath = separator === -1 ? reference : reference.slice(0, separator);
  const selector = separator === -1 ? null : reference.slice(separator + 1);
  if (!referencedPath || path.isAbsolute(referencedPath) || referencedPath.split('/').includes('..')) {
    return [`${file}: context_refs must use a repository-root relative path: ${reference}`];
  }
  if (/^changes\/(?!_template\/)/.test(referencedPath)) {
    return [`${file}: other Work Packages must use related_changes, not context_refs: ${reference}`];
  }
  const target = path.resolve(ROOT, referencedPath);
  if (!fs.existsSync(target) || !fs.statSync(target).isFile()) {
    return [`${file}: context_refs path does not exist: ${reference}`];
  }
  if (selector === '') return [`${file}: context_refs selector is empty: ${reference}`];
  if (selector && !fs.readFileSync(target, 'utf8').includes(selector)) {
    return [`${file}: context_refs selector not found: ${reference}`];
  }
  return [];
}

const markdownFiles = ['AGENTS.md', ...DOC_DIRS.flatMap(walk)
  .filter((file) => file.endsWith('.md') && !file.startsWith('docs/legacy/'))];
const requirementText = fs.readFileSync('docs/03-functional-requirements.md', 'utf8');
const traceabilityText = fs.readFileSync('docs/08-traceability.md', 'utf8');
const requirements = new Set([...requirementText.matchAll(/^### ((?:REQ|NFR)-[A-Z]+-\d+)/gm)].map((match) => match[1]));
const tracedRequirements = new Set([...traceabilityText.matchAll(/^\| \[((?:REQ|NFR)-[A-Z]+-\d+)\]/gm)].map((match) => match[1]));
const tests = new Set([...requirementText.matchAll(/`((?:AT|CT)-[A-Z-]+-\d+)`/g)].map((match) => match[1]));
const tracedTests = new Set([...traceabilityText.matchAll(/(?:AT|CT)-[A-Z-]+-\d+/g)].map((match) => match[0]));

const packageJson = JSON.parse(fs.readFileSync('package.json', 'utf8'));
const documentedCommands = new Set();
for (const file of markdownFiles) {
  for (const match of fs.readFileSync(file, 'utf8').matchAll(/pnpm ([a-z][a-z0-9:.-]+)/g)) documentedCommands.add(match[1]);
}
const absentCommands = [...documentedCommands].filter((command) => command !== 'install' && !(command in packageJson.scripts));

const adrIds = new Set(walk('adr')
  .filter((file) => /^\d{4}-.*\.md$/.test(path.basename(file)))
  .map((file) => `ADR-${path.basename(file).slice(0, 4)}`));

const yamlFiles = walk('changes').filter((file) => file.endsWith('.yaml')).sort();
const parsedChanges = [];
const yamlErrors = [];
for (const file of yamlFiles) {
  try {
    parsedChanges.push([file, parseChangeYaml(file)]);
  } catch (error) {
    yamlErrors.push(error.message);
  }
}

const templateEntry = parsedChanges.find(([file]) => file === 'changes/_template/change.yaml');
if (!templateEntry) yamlErrors.push('changes/_template/change.yaml: missing template');
const v2Keys = Object.keys(templateEntry?.[1] ?? {}).sort();
const legacyKeys = new Set([
  'id', 'title', 'type', 'profile', 'spec_delta', 'status', 'spec_revision', 'target_release',
  'requirements', 'tests', 'adrs', 'context_refs', 'related_changes', 'affected_surfaces',
  'affected_formats', 'affected_versions', 'security_impact', 'migration', 'supersedes',
]);
const archivedIds = new Set(JSON.parse(fs.readFileSync('changes/archive.json', 'utf8')).archives.map((entry) => entry.work_id));
const changeIds = new Map();
for (const [file, change] of parsedChanges) {
  if (file === 'changes/_template/change.yaml') continue;
  if (changeIds.has(change.id)) yamlErrors.push(`${file}: duplicate Work ID ${change.id}`);
  else changeIds.set(change.id, file);
}

for (const [file, change] of parsedChanges) {
  const keys = Object.keys(change).sort();
  const historicalArchive = file !== 'changes/_template/change.yaml' && archivedIds.has(change.id);
  const v2 = isWorkV2(change);
  if (v2) {
    const missingKeys = v2Keys.filter((key) => !keys.includes(key));
    const unknownKeys = keys.filter((key) => !v2Keys.includes(key));
    if (missingKeys.length > 0) yamlErrors.push(`${file}: missing schema-v2 fields ${missingKeys.join(', ')}`);
    if (unknownKeys.length > 0) yamlErrors.push(`${file}: unknown schema-v2 fields ${unknownKeys.join(', ')}`);
    if (!V2_STATUSES.has(change.status)) yamlErrors.push(`${file}: invalid schema-v2 status ${change.status}`);
    if (typeof change.goal !== 'string' || !change.goal.trim()) yamlErrors.push(`${file}: schema-v2 goal is required`);
  } else {
    const unknownKeys = keys.filter((key) => !legacyKeys.has(key));
    if (unknownKeys.length > 0) yamlErrors.push(`${file}: unknown legacy fields ${unknownKeys.join(', ')}`);
    if (!LEGACY_STATUSES.has(change.status)) yamlErrors.push(`${file}: invalid legacy status ${change.status}`);
  }
  if (!ALLOWED_TYPES.has(change.type)) yamlErrors.push(`${file}: invalid type ${change.type}`);
  if (!v2 && !historicalArchive) {
    if (!WORK_PROFILES.has(change.profile)) yamlErrors.push(`${file}: invalid profile ${change.profile}`);
    if (!SPEC_DELTAS.has(change.spec_delta)) yamlErrors.push(`${file}: invalid spec_delta ${change.spec_delta}`);
    if (change.spec_delta === 'behavior_change' && change.profile !== 'controlled') {
      yamlErrors.push(`${file}: behavior_change requires profile controlled`);
    }
    if (change.profile === 'light' && ['security', 'dependency', 'migration', 'governance'].includes(change.type)) {
      yamlErrors.push(`${file}: type ${change.type} requires profile controlled`);
    }
  }
  if (file !== 'changes/_template/change.yaml') {
    const revision = SPEC_REVISION_PATTERN.exec(change.spec_revision ?? '');
    const currentRevision = SPEC_REVISION_PATTERN.exec(CURRENT_SPEC_REVISION);
    if (!revision || !currentRevision || Number(revision[1]) > Number(currentRevision[1])) {
      yamlErrors.push(`${file}: spec_revision must be a known revision no newer than ${CURRENT_SPEC_REVISION}`);
    }
  }
  const listFields = v2
    ? ['requirements', 'tests', 'adrs', 'context_refs', 'related_changes', 'boundaries', 'risks', 'next']
    : ['requirements', 'tests', 'adrs', 'context_refs', 'related_changes', 'affected_surfaces', 'affected_formats', 'affected_versions', 'supersedes'];
  for (const field of listFields) {
    if (!Array.isArray(change[field])) yamlErrors.push(`${file}: ${field} must be a list`);
  }
  for (const requirement of change.requirements ?? []) if (!requirements.has(requirement)) yamlErrors.push(`${file}: unknown Requirement ${requirement}`);
  for (const testId of change.tests ?? []) if (!tests.has(testId)) yamlErrors.push(`${file}: unknown Test ${testId}`);
  for (const adr of change.adrs ?? []) if (!/^ADR-\d{4}$/.test(adr) || !adrIds.has(adr)) yamlErrors.push(`${file}: unknown ADR ${adr}`);
  for (const reference of change.context_refs ?? []) yamlErrors.push(...validateContextRef(reference, file));
  if (file === 'changes/_template/change.yaml') continue;
  for (const relatedId of change.related_changes ?? []) {
    if (relatedId === change.id) yamlErrors.push(`${file}: related_changes cannot reference itself`);
    else if (!changeIds.has(relatedId)) yamlErrors.push(`${file}: unknown related Work ${relatedId}`);
  }
  for (const supersededId of change.supersedes ?? []) {
    if (supersededId === change.id) yamlErrors.push(`${file}: supersedes cannot reference itself`);
    else if (!changeIds.has(supersededId)) yamlErrors.push(`${file}: unknown superseded Work ${supersededId}`);
  }
}

const duplicateStateViews = [];
const specIndexText = fs.readFileSync('docs/00-spec-index.md', 'utf8');
if (/^## Active Work$/m.test(specIndexText)) duplicateStateViews.push('docs/00-spec-index.md: duplicated active Work status list');
if (/^## Active Work$/m.test(traceabilityText)) duplicateStateViews.push('docs/08-traceability.md: duplicated active Work status table');

const governanceLinks = [
  ['AGENTS.md', 'The canonical threshold and minimal lifecycle are owned only by `docs/09-document-governance.md`.'],
  ['docs/09-document-governance.md', 'This file is the canonical owner of Aivo documentation governance.'],
  ['changes/_template/change.yaml', 'schema: "2"'],
];
const governanceErrors = [];
for (const [file, marker] of governanceLinks) {
  if (!fs.readFileSync(file, 'utf8').includes(marker)) governanceErrors.push(`${file}: missing governance marker ${marker}`);
}

const releaseErrors = [];
const doneV2Ids = new Set(parsedChanges
  .filter(([, change]) => isWorkV2(change) && change.status === 'Done')
  .map(([, change]) => change.id));
for (const file of walk('releases').filter((candidate) => /\/v[^/]+\.md$/.test(candidate))) {
  for (const match of fs.readFileSync(file, 'utf8').matchAll(/\b(?:CHG|BUG|SEC|DEP|MIG)-\d{4}-\d{3}[a-z0-9-]*\b/g)) {
    if (!archivedIds.has(match[0]) && !doneV2Ids.has(match[0])) {
      releaseErrors.push(`${file}: Release references Work that is neither schema-v2 Done nor legacy sealed: ${match[0]}`);
    }
  }
}

const errors = {
  missing_paths: validateExplicitPaths(markdownFiles),
  requirements_not_traced: setDifference(requirements, tracedRequirements),
  trace_without_requirement: setDifference(tracedRequirements, requirements),
  tests_not_traced: setDifference(tests, tracedTests),
  trace_tests_without_requirement: setDifference(tracedTests, tests),
  stale_traceability: traceabilityText === expectedTraceability(ROOT) ? [] : ['docs/08-traceability.md: run pnpm docs:trace'],
  absent_commands: absentCommands,
  yaml_errors: yamlErrors,
  duplicate_work_state_views: duplicateStateViews,
  governance_markers: governanceErrors,
  release_references: releaseErrors,
  work_archives: validateArchiveManifest({
    root: ROOT,
    changes: parsedChanges
      .filter(([file]) => file !== 'changes/_template/change.yaml')
      .map(([file, change]) => ({ file, id: change.id, status: change.status })),
    baseRef: process.env.AIVO_ARCHIVE_BASE_REF || 'HEAD',
  }),
};

if (Object.values(errors).some((values) => values.length > 0)) {
  console.error(JSON.stringify(errors, null, 2));
  process.exit(1);
}

console.log(`docs:check passed (${markdownFiles.length} Markdown, ${yamlFiles.length} YAML, ${requirements.size} requirements, ${tests.size} test IDs, ${adrIds.size} ADRs, ${changeIds.size} Work Packages, ${archivedIds.size} archived Work Packages)`);
