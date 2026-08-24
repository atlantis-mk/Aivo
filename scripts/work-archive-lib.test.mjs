import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { archiveWork, hashFile, validateArchiveManifest } from './work-archive-lib.mjs';

function fixture(status = 'Verified') {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'aivo-work-archive-'));
  const workId = 'CHG-2026-999-test-archive';
  fs.mkdirSync(path.join(root, 'changes', workId), { recursive: true });
  fs.writeFileSync(path.join(root, 'changes', workId, 'change.yaml'), `id: "${workId}"\nstatus: "${status}"\n`);
  fs.writeFileSync(path.join(root, 'changes', workId, 'change.md'), '# Evidence\n');
  return { root, workId, change: { id: workId, status, file: `changes/${workId}/change.yaml` } };
}

test('archives completed Work and detects later content changes', () => {
  const { root, workId, change } = fixture();
  archiveWork({ root, workId, status: change.status, archivedAt: '2026-07-31T00:00:00.000Z' });
  assert.deepEqual(validateArchiveManifest({ root, changes: [change], baseRef: null }), []);

  fs.appendFileSync(path.join(root, 'changes', workId, 'change.md'), 'tampered\n');
  assert.match(validateArchiveManifest({ root, changes: [change], baseRef: null }).join('\n'), /archived file changed/);
});

test('refuses to archive incomplete or already archived Work', () => {
  const pending = fixture('Implementing');
  assert.throws(() => archiveWork({ root: pending.root, workId: pending.workId, status: pending.change.status }), /not complete/);

  const complete = fixture();
  archiveWork({ root: complete.root, workId: complete.workId, status: complete.change.status });
  assert.throws(() => archiveWork({ root: complete.root, workId: complete.workId, status: complete.change.status }), /already archived/);
});

test('rejects rehashing an existing archive relative to the Git baseline', () => {
  const { root, workId, change } = fixture();
  archiveWork({ root, workId, status: change.status, archivedAt: '2026-07-31T00:00:00.000Z' });
  execFileSync('git', ['init'], { cwd: root, stdio: 'ignore' });
  execFileSync('git', ['config', 'user.email', 'archive-test@aivo.invalid'], { cwd: root });
  execFileSync('git', ['config', 'user.name', 'Aivo Archive Test'], { cwd: root });
  execFileSync('git', ['add', '.'], { cwd: root });
  execFileSync('git', ['commit', '-m', 'archive baseline'], { cwd: root, stdio: 'ignore' });

  const markdown = path.join(root, 'changes', workId, 'change.md');
  fs.appendFileSync(markdown, 'rewritten\n');
  const manifestPath = path.join(root, 'changes', 'archive.json');
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  manifest.archives[0].files[`changes/${workId}/change.md`] = hashFile(markdown);
  fs.writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);

  assert.match(validateArchiveManifest({ root, changes: [change], baseRef: 'HEAD' }).join('\n'), /existing archive .* is immutable/);
});

test('allows an initial archive registry when the trusted baseline predates it', () => {
  const { root, workId, change } = fixture();
  execFileSync('git', ['init'], { cwd: root, stdio: 'ignore' });
  execFileSync('git', ['config', 'user.email', 'archive-test@aivo.invalid'], { cwd: root });
  execFileSync('git', ['config', 'user.name', 'Aivo Archive Test'], { cwd: root });
  execFileSync('git', ['add', '.'], { cwd: root });
  execFileSync('git', ['commit', '-m', 'pre-archive baseline'], { cwd: root, stdio: 'ignore' });

  archiveWork({ root, workId, status: change.status, archivedAt: '2026-07-31T00:00:00.000Z' });
  assert.deepEqual(validateArchiveManifest({ root, changes: [change], baseRef: 'HEAD' }), []);
});
