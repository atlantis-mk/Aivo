import assert from 'node:assert/strict'
import test from 'node:test'

import { desktopRouterHistoryMode } from '../src/lib/router-history.ts'

test('packaged file renderer uses hash routing instead of the app bundle path', () => {
  assert.equal(desktopRouterHistoryMode('file:'), 'hash')
})

test('development server keeps browser routing', () => {
  assert.equal(desktopRouterHistoryMode('http:'), 'browser')
  assert.equal(desktopRouterHistoryMode('https:'), 'browser')
})
