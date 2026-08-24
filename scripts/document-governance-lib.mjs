import fs from 'node:fs';
import path from 'node:path';

export const WORK_PROFILES = new Set(['light', 'controlled']);
export const SPEC_DELTAS = new Set(['none', 'clarification', 'behavior_change']);

export function isWorkV2(change) {
  return change.schema === '2';
}

export function walkFiles(directory) {
  if (!fs.existsSync(directory)) return [];
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const target = path.join(directory, entry.name);
    return entry.isDirectory() ? walkFiles(target) : [target.split(path.sep).join('/')];
  });
}

function parseScalar(encoded, file, line) {
  if (encoded === 'null') return null;
  if (encoded === '[]') return [];
  if (/^"(?:[^"\\]|\\.)*"$/.test(encoded)) return JSON.parse(encoded);
  throw new Error(`${file}:${line}: values must be quoted strings, null, or lists`);
}

export function parseChangeYamlText(text, file = '<change.yaml>') {
  const result = {};
  let activeList = null;
  for (const [index, raw] of text.split(/\r?\n/).entries()) {
    if (!raw.trim() || raw.trimStart().startsWith('#')) continue;
    const listItem = raw.match(/^  - (.+)$/);
    if (listItem) {
      if (!activeList || !Array.isArray(result[activeList])) throw new Error(`${file}:${index + 1}: orphan list item`);
      result[activeList].push(parseScalar(listItem[1], file, index + 1));
      continue;
    }
    const field = raw.match(/^([a-z][a-z0-9_]*):(?: (.*))?$/);
    if (!field) throw new Error(`${file}:${index + 1}: unsupported YAML syntax`);
    const [, key, encoded = ''] = field;
    if (key in result) throw new Error(`${file}:${index + 1}: duplicate key ${key}`);
    if (!encoded) {
      result[key] = [];
      activeList = key;
    } else {
      result[key] = parseScalar(encoded, file, index + 1);
      activeList = Array.isArray(result[key]) ? key : null;
    }
  }
  return result;
}

export function readChanges(root, { includeTemplate = false } = {}) {
  return walkFiles(path.join(root, 'changes'))
    .filter((file) => file.endsWith('/change.yaml') || file.endsWith('changes/_template/change.yaml'))
    .filter((file) => includeTemplate || !file.endsWith('changes/_template/change.yaml'))
    .sort()
    .map((absoluteFile) => {
      const file = path.relative(root, absoluteFile).split(path.sep).join('/');
      return { file, change: parseChangeYamlText(fs.readFileSync(absoluteFile, 'utf8'), file) };
    });
}

export function replaceYamlScalar(text, field, value) {
  const pattern = new RegExp(`^${field}: (?:"(?:[^"\\\\]|\\\\.)*"|null)$`, 'm');
  if (!pattern.test(text)) throw new Error(`change.yaml: missing scalar field ${field}`);
  const encoded = value === null ? 'null' : JSON.stringify(value);
  return text.replace(pattern, `${field}: ${encoded}`);
}

export function currentSpecRevision(root) {
  const text = fs.readFileSync(path.join(root, 'docs', '00-spec-index.md'), 'utf8');
  const match = text.match(/^- Specification revision: `([^`]+)`$/m);
  if (!match) throw new Error('docs/00-spec-index.md: missing specification revision');
  return match[1];
}

export function extractRequirements(text) {
  const headings = [...text.matchAll(/^### ((?:REQ|NFR)-[A-Z]+-\d+) (.+)$/gm)];
  return headings.map((heading, index) => {
    const bodyStart = heading.index + heading[0].length;
    const bodyEnd = headings[index + 1]?.index ?? text.length;
    const body = text.slice(bodyStart, bodyEnd);
    const tests = [...new Set([...body.matchAll(/`((?:AT|CT)-[A-Z-]+-\d+)`/g)].map((match) => match[1]))];
    return { id: heading[1], title: heading[2], tests };
  });
}

function slugify(value) {
  return value.toLowerCase().replace(/[^a-z0-9\s-]/g, '').trim().replace(/\s+/g, '-');
}

function workLink(work, suffix = '') {
  return `[${work.change.id}](../changes/${work.change.id}/)${suffix}`;
}

export function generateTraceabilityMarkdown({ requirementText, changes, archivedIds }) {
  const requirements = extractRequirements(requirementText);
  const rows = requirements.map((requirement) => {
    const routed = changes.filter(({ change }) => (change.requirements ?? []).includes(requirement.id));
    const adrs = [...new Set(routed.flatMap(({ change }) => change.adrs ?? []))].sort();
    const active = routed
      .filter(({ change }) => !archivedIds.has(change.id) && !(isWorkV2(change) && ['Done', 'Rejected'].includes(change.status)))
      .sort((left, right) => left.change.id.localeCompare(right.change.id))
      .map((work) => workLink(work, ` (${work.change.status})`));
    const completed = routed
      .filter(({ change }) => archivedIds.has(change.id) || (isWorkV2(change) && change.status === 'Done'))
      .sort((left, right) => left.change.id.localeCompare(right.change.id))
      .map((work) => workLink(work));
    const requirementLink = `[${requirement.id}](03-functional-requirements.md#${slugify(`${requirement.id} ${requirement.title}`)})`;
    return `| ${requirementLink} | ${requirement.tests.join(', ') || '-'} | ${adrs.join(', ') || '-'} | ${active.join('<br>') || '-'} | ${completed.join('<br>') || '-'} |`;
  });

  return `# Aivo requirement traceability

> Generated by \`pnpm docs:trace\`. Do not edit this file manually.

Requirements and stable Test IDs are owned by \`docs/03-functional-requirements.md\`. Work routing is owned by each \`change.yaml\`; Git owns schema-v2 completed history and \`changes/archive.json\` identifies legacy sealed evidence.

| Requirement | Test IDs | Routed ADRs | Active Work | Completed Work |
| --- | --- | --- | --- | --- |
${rows.join('\n')}
`;
}

export function expectedTraceability(root) {
  const requirementText = fs.readFileSync(path.join(root, 'docs', '03-functional-requirements.md'), 'utf8');
  const archive = JSON.parse(fs.readFileSync(path.join(root, 'changes', 'archive.json'), 'utf8'));
  return generateTraceabilityMarkdown({
    requirementText,
    changes: readChanges(root),
    archivedIds: new Set(archive.archives.map((entry) => entry.work_id)),
  });
}
