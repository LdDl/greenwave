import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { waveStatus, reverseGroupNotice } from '../src/lib/utils/wave-status.js';
import { DEMO_DATA } from '../src/lib/utils/demo-input.js';

const example = JSON.parse(readFileSync(new URL('./fixtures/bidirectional-turns.json', import.meta.url), 'utf8'));

test('empty responses distinguish not calculated, stale, no waves and partial coordination', () => {
  assert.equal(waveStatus(4, [], [], false).kind, 'pending');
  assert.match(waveStatus(4, [], [], true, true).message, /Recalculate/);
  assert.equal(waveStatus(4, [[], [], []], [], true).kind, 'warning');
  assert.match(waveStatus(4, [[], [], []], [], true).message, /No green wave/);
  assert.match(waveStatus(4, [[{ band_width: 12 }]], [], true).message, /No full-corridor wave/);
  assert.match(waveStatus(4, [], [{ depth: 3, bandwidth: 12 }], true).message, /No full-corridor wave/);
  assert.equal(waveStatus(1, [], [], true).kind, 'pending');
});

test('full-corridor status sums only waves that pass every junction', () => {
  const status = waveStatus(4, [], [{ depth: 3, bandwidth: 40 }, { depth: 4, bandwidth: 10 }, { depth: 4, bandwidth: 8.5 }], true);
  assert.equal(status.kind, 'success');
  assert.match(status.message, /18.5 s band/);
});

test('legacy demo explicitly reports that reverse inherits the forward group', () => {
  assert.match(reverseGroupNotice(DEMO_DATA.junctions), /forward group at every junction/);
  assert.equal(reverseGroupNotice(example.junctions, example.groupIds, example.reverseGroupIds), '');
  assert.match(reverseGroupNotice(example.junctions, example.groupIds, { ...example.reverseGroupIds, 0: 10 }), /1 of 4/);
});

test('missing reverse selections and reverse groups without green identify the affected junction', () => {
  assert.match(reverseGroupNotice(example.junctions), /Choose a reverse group.*Pushkin Street/);
  const junctions = structuredClone(example.junctions);
  for (const phase of junctions[0].cycle) {
    for (const signal of phase.signal_groups.find(group => group.id === 20).signals) signal.color = 'RED';
  }
  assert.match(reverseGroupNotice(junctions, example.groupIds, example.reverseGroupIds), /no green signal: Pushkin Street/);
});
