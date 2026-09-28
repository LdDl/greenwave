import test from 'node:test';
import assert from 'node:assert/strict';
import { programGroupIds, programTimeline, programStateAt } from '../src/lib/utils/junction-program.js';

function junction() {
  return {
    id: 0,
    offset: 5,
    cycle: [
      { id: 0, signal_groups: [{ id: 8, signals: [
        { color: 'GREEN', duration: 20, min_duration: 10, max_duration: 30 },
        { color: 'YELLOW', duration: 5 },
      ] }] },
      { id: 4, signal_groups: [{ id: 8, signals: [{ color: 'RED', duration: 15 }] }] },
    ],
  };
}

test('timeline retains phase and signal identities, duration bounds and the source program', () => {
  const input = junction();
  const before = structuredClone(input);
  const timeline = programTimeline(input, 8);
  assert.deepEqual(programGroupIds(input), [8]);
  assert.equal(timeline.duration, 40);
  assert.deepEqual(timeline.segments.map(s => [s.phaseId, s.phaseIndex, s.groupId, s.signalIndex, s.start, s.end]), [
    [0, 0, 8, 0, 0, 20], [0, 0, 8, 1, 20, 25], [4, 1, 8, 0, 25, 40],
  ]);
  assert.equal(timeline.segments[0].min_duration, 10);
  assert.equal(timeline.segments[0].max_duration, 30);
  timeline.segments[0].duration = 99;
  assert.deepEqual(input, before);
});

test('preview observes offsets, exact signal boundaries and wrapping in either direction', () => {
  const input = junction();
  const timeline = programTimeline(input, 8);
  const cases = [
    { time: 0, offset: 5, position: 35, color: 'RED' },
    { time: 5, offset: 5, position: 0, color: 'GREEN' },
    { time: 25, offset: 5, position: 20, color: 'YELLOW' },
    { time: 30, offset: 5, position: 25, color: 'RED' },
    { time: 45, offset: 5, position: 0, color: 'GREEN' },
    { time: 0, offset: -20, position: 20, color: 'YELLOW' },
    { time: -10, offset: 5, position: 25, color: 'RED' },
    { time: 10, offset: 85, position: 5, color: 'GREEN' },
  ];
  for (const { time, offset, position, color } of cases) {
    const state = programStateAt(timeline, time, offset);
    assert.equal(state.position, position);
    assert.equal(state.segment.color, color);
  }
});

test('selecting a group does not merge other groups or confuse zero with an absent ID', () => {
  const input = junction();
  input.cycle[0].signal_groups.unshift({ id: 0, signals: [{ color: 'RED', duration: 25 }] });
  input.cycle[1].signal_groups.unshift({ id: 0, signals: [{ color: 'GREEN', duration: 15 }] });
  assert.deepEqual(programGroupIds(input), [0, 8]);
  assert.deepEqual(programTimeline(input, 0).segments.map(s => s.color), ['RED', 'GREEN']);
  assert.deepEqual(programTimeline(input, 8).segments.map(s => s.color), ['GREEN', 'YELLOW', 'RED']);
  input.cycle[1].signal_groups.pop();
  assert.throws(() => programTimeline(input, 8), /G8 is missing in phase 2/);
});

test('incomplete drafts fail predictably and an empty program has no active signal', () => {
  for (const duration of [undefined, NaN, Infinity, 0, -1]) {
    const input = junction();
    input.cycle[0].signal_groups[0].signals[0].duration = duration;
    assert.throws(() => programTimeline(input, 8), /positive signal durations/);
  }
  assert.deepEqual(programGroupIds(null), []);
  assert.deepEqual(programTimeline(null, 0), { duration: 0, segments: [] });
  assert.equal(programStateAt(programTimeline(null, 0), 0), null);
  const timeline = programTimeline(junction(), 8);
  assert.equal(programStateAt(timeline, NaN), null);
  assert.equal(programStateAt(timeline, 1, NaN), null);
});

test('fractional signal boundaries use the new signal without shifting the preview time', () => {
  const input = junction();
  input.cycle = [{ id: 0, signal_groups: [{ id: 8, signals: [
    { color: 'RED', duration: 0.1 }, { color: 'GREEN', duration: 0.3 },
  ] }] }];
  const state = programStateAt(programTimeline(input, 8), 0.1);
  assert.equal(state.position, 0.1);
  assert.equal(state.segment.color, 'GREEN');
});
