import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { get } from 'svelte/store';
import { readProgram, programTimeline, programStateAt, reverseCorridorGroupId, corridorTimelineRows, removeProgramGroup, calculationGroupIds } from '../src/lib/utils/junction-program.js';
import { inputToSimpleCorridor, simpleCorridorToInput } from '../src/lib/utils/simple-corridor.js';
import { createCorridorEditor } from '../src/lib/stores/corridor.js';
import { prepareInputExport, prepareOutputExport, validateImportedConfig } from '../src/lib/utils/export-import.js';
import { DEMO_DATA } from '../src/lib/utils/demo-input.js';

const example = JSON.parse(readFileSync(new URL('./fixtures/bidirectional-turns.json', import.meta.url), 'utf8'));

test('example has aligned 120-second programs and a turn-only green while both through groups are red', () => {
  assert.equal(validateImportedConfig(example).isValid, true);
  assert.deepEqual(calculationGroupIds(example.junctions, example.groupIds), { 0: 10, 1: 10, 2: 10, 3: 10 });
  assert.deepEqual(calculationGroupIds(example.junctions, example.reverseGroupIds), { 0: 20, 1: 20, 2: 20, 3: 20 });
  for (const junction of example.junctions) {
    assert.deepEqual(readProgram(junction), { groupIds: [10, 20, 30, 40], phaseDurations: [85, 15, 20], duration: 120 });
    for (let time = 85; time < 95; time += 0.5) {
      for (const [id, color] of [[10, 'RED'], [20, 'RED'], [30, 'GREEN'], [40, 'RED']]) {
        assert.equal(programStateAt(programTimeline(junction, id), time + junction.offset, junction.offset).segment.color, color);
      }
    }
  }
});

test('independent reverse selections round trip through graph and both JSON export types', () => {
  for (const prepare of [prepareInputExport, prepareOutputExport]) {
    const file = prepare(example.junctions, example.desiredSpeed, 1200, 'bidirectional', example.groupIds, example.reverseGroupIds);
    const graph = inputToSimpleCorridor(file);
    assert.deepEqual(graph.corridor.stops[0], { nodeId: 0, forwardGroupId: 10, reverseGroupId: 20 });
    const input = simpleCorridorToInput(graph);
    assert.deepEqual(input.junctions, example.junctions);
    assert.deepEqual(input.groupIds, example.groupIds);
    assert.deepEqual(input.reverseGroupIds, example.reverseGroupIds);
    assert.equal(validateImportedConfig(input).isValid, true);
    file.reverseGroupIds[0] = 30;
    assert.equal(example.reverseGroupIds[0], 20);
  }
});

test('the cross street uses gaps in through traffic without overlapping through or turn signals', () => {
  for (const junction of example.junctions) {
    const crossStreet = programTimeline(junction, 40);
    assert.ok(crossStreet.segments.some(signal => signal.phaseIndex === 0 && signal.color === 'GREEN'));
    for (let time = 0; time < crossStreet.duration; time += 0.5) {
      const crossColor = programStateAt(crossStreet, time).segment.color;
      if (crossColor === 'GREEN' || crossColor === 'YELLOW') {
        for (const id of [10, 20, 30]) assert.equal(programStateAt(programTimeline(junction, id), time).segment.color, 'RED');
      }
    }
  }
});

test('reverse selection defaults per junction and explicit zero is not treated as missing', () => {
  const junction = example.junctions[0];
  assert.equal(reverseCorridorGroupId(junction, example.groupIds), 10);
  assert.equal(reverseCorridorGroupId(junction, example.groupIds, { 0: 20 }), 20);
  assert.equal(reverseCorridorGroupId(junction, example.groupIds, { 0: 0 }), 0);
  assert.equal(corridorTimelineRows(junction, example.groupIds, example.reverseGroupIds, 'forward').length, 1);
  assert.equal(corridorTimelineRows(junction, example.groupIds, {}, 'bidirectional').length, 1);
  assert.deepEqual(corridorTimelineRows(junction, example.groupIds, example.reverseGroupIds, 'bidirectional').map(row => row.groupId), [10, 20]);
  assert.throws(() => removeProgramGroup(junction, 20, 10, 20), /Choose another corridor group/);
});

test('reverse-only edits are atomic, invalidate calculations, survive reload and are pruned with the junction', () => {
  const values = new Map();
  const storage = { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value) };
  const editor = createCorridorEditor();
  editor.connectStorage(storage);
  editor.replaceInput(example);
  const before = editor.snapshot();
  const revision = get(editor.calculationRevision);
  editor.saveJunction(example.junctions[0], false, 10, 30);
  assert.equal(get(editor.calculationRevision), revision + 1);
  assert.equal(get(editor.reverseGroupIds)[0], 30);
  editor.undo();
  assert.deepEqual(editor.snapshot(), before);
  editor.redo();
  const restored = createCorridorEditor();
  restored.connectStorage(storage);
  assert.deepEqual(restored.snapshot(), editor.snapshot());
  editor.direction.set('forward');
  assert.equal(get(editor.reverseGroupIds)[0], 30);
  editor.saveJunction(example.junctions[0], false, 10, null);
  assert.equal(Object.hasOwn(get(editor.reverseGroupIds), 0), false);
  editor.removeJunction(1);
  assert.equal(Object.hasOwn(get(editor.reverseGroupIds), 1), false);
  editor.undo();
  assert.equal(get(editor.reverseGroupIds)[1], 20);
  editor.replaceInput(DEMO_DATA);
  assert.deepEqual(get(editor.reverseGroupIds), {});
});

test('bad reverse references fail without changing saved input or history', () => {
  const editor = createCorridorEditor();
  editor.replaceInput(example);
  const before = editor.snapshot();
  for (const reverseGroupIds of [{ 0: 99 }, { 99: 10 }, { 0: '20' }, null, []]) {
    assert.throws(() => editor.replaceInput({ ...example, reverseGroupIds }), /group/);
    assert.deepEqual(editor.snapshot(), before);
  }
  assert.throws(() => editor.saveJunction(example.junctions[0], false, 10, 99), /group/);
  assert.deepEqual(editor.snapshot(), before);
});
