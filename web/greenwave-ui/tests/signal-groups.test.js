import test from 'node:test';
import assert from 'node:assert/strict';
import { get } from 'svelte/store';
import { DEMO_DATA } from '../src/lib/utils/demo-input.js';
import { readProgram, programTimeline, addProgramGroup, removeProgramGroup, corridorGroupId, calculationGroupIds, corridorProgramError } from '../src/lib/utils/junction-program.js';
import { inputToSimpleCorridor, simpleCorridorToInput } from '../src/lib/utils/simple-corridor.js';
import { createCorridorEditor, CORRIDOR_STORAGE_KEY } from '../src/lib/stores/corridor.js';
import { prepareInputExport, prepareOutputExport, validateImportedConfig } from '../src/lib/utils/export-import.js';

function multipleGroups() {
  const input = structuredClone(DEMO_DATA);
  input.junctions[1] = addProgramGroup(input.junctions[1]).junction;
  input.groupIds = { 1: 1 };
  return input;
}

test('new permanent groups are red across all phases and preserve existing programs and bounds', () => {
  const original = structuredClone(DEMO_DATA.junctions[1]);
  original.cycle[0].signal_groups[0].signals[1].min_duration = 15;
  for (const phase of original.cycle) phase.signal_groups[0].id = 8;
  const before = structuredClone(original);
  const result = addProgramGroup(original);
  assert.equal(result.groupId, 9);
  assert.deepEqual(readProgram(result.junction), { groupIds: [8, 9], phaseDurations: [60, 25], duration: 85 });
  assert.deepEqual(programTimeline(result.junction, 9).segments.map(s => [s.color, s.duration]), [['RED', 60], ['RED', 25]]);
  assert.deepEqual(removeProgramGroup(result.junction, 9, 8), before);
  assert.deepEqual(original, before);
  assert.throws(() => removeProgramGroup(result.junction, 8, 8), /Choose another corridor group/);
  assert.throws(() => removeProgramGroup(original, 8, null), /at least one/);
});

test('phase boundaries must agree even when different groups have the same total cycle', () => {
  const junction = multipleGroups().junctions[1];
  junction.cycle[0].signal_groups[1].signals[0].duration = 59;
  junction.cycle[1].signal_groups[1].signals[0].duration = 26;
  assert.throws(() => readProgram(junction), /Phase 1: group durations must match.*G0: 60 s.*G1: 59 s/);
  assert.throws(() => addProgramGroup(junction), /group durations must match/);
  junction.cycle[0].signal_groups[1].signals = [{ color: 'RED', duration: 5 }, { color: 'GREEN', duration: 55 }];
  junction.cycle[1].signal_groups[1].signals[0].duration = 25;
  assert.equal(readProgram(junction).duration, 85);
  assert.equal(programTimeline(junction, 1).segments[1].start, 5);
});

test('a single group is implicit; multiple groups require selection and never silently use the first', () => {
  const input = multipleGroups();
  assert.equal(corridorGroupId(input.junctions[0]), 0);
  assert.equal(corridorGroupId(input.junctions[1]), null);
  assert.throws(() => calculationGroupIds(input.junctions), /choose a corridor group/);
  assert.deepEqual(calculationGroupIds(input.junctions, input.groupIds), { 0: 0, 1: 1, 2: 0, 3: 0 });
  assert.deepEqual(calculationGroupIds(input.junctions, { 1: 0 }), { 0: 0, 1: 0, 2: 0, 3: 0 });
  assert.match(corridorProgramError(input.junctions, { 1: 99 }), /does not exist/);
  assert.equal(corridorProgramError(input.junctions, input.groupIds), '');
});

test('graph round trips retain all groups, timing, explicit selection and shared reverse stops', () => {
  const input = { ...multipleGroups(), desiredIntensity: 0, direction: 'bidirectional' };
  const project = inputToSimpleCorridor(input);
  assert.deepEqual(project.corridor.stops[1], { nodeId: 1, forwardGroupId: 1, reverseGroupId: 1 });
  assert.equal(project.nodes.length, 4);
  assert.deepEqual(project.nodes[1].signalGroups.map(group => group.id), [0, 1]);
  assert.deepEqual(simpleCorridorToInput(project), input);
  delete input.groupIds;
  const unresolved = inputToSimpleCorridor(input);
  assert.equal(unresolved.corridor.stops[1].forwardGroupId, null);
  assert.deepEqual(simpleCorridorToInput(unresolved), input);
});

test('program and corridor selection save as one undo step, survive reload and invalidate calculations', () => {
  const values = new Map();
  const storage = { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value) };
  const editor = createCorridorEditor();
  editor.connectStorage(storage);
  editor.replaceInput(DEMO_DATA);
  const before = editor.snapshot();
  const revision = get(editor.calculationRevision);
  const junction = multipleGroups().junctions[1];
  editor.saveJunction(junction, false, 1);
  assert.equal(get(editor.calculationRevision), revision + 1);
  assert.deepEqual(get(editor.groupIds), { 1: 1 });
  assert.equal(get(editor.graph).nodes.length, 4);
  const after = editor.snapshot();
  editor.undo();
  assert.deepEqual(editor.snapshot(), before);
  editor.redo();
  assert.deepEqual(editor.snapshot(), after);
  const restored = createCorridorEditor();
  restored.connectStorage(storage);
  assert.deepEqual(restored.snapshot(), after);
  editor.saveJunction(junction, false, 0);
  assert.equal(get(editor.calculationRevision), revision + 4);
  editor.removeJunction(1);
  assert.deepEqual(get(editor.groupIds), {});
  editor.undo();
  assert.deepEqual(get(editor.groupIds), { 1: 0 });
  const saved = storage.getItem(CORRIDOR_STORAGE_KEY);
  assert.throws(() => editor.saveJunction(removeProgramGroup(junction, 1, 0), false, 99), /Selected corridor group/);
  assert.equal(storage.getItem(CORRIDOR_STORAGE_KEY), saved);
});

test('input and result exports preserve selection independently; imports cannot retain stale selection', () => {
  const input = multipleGroups();
  for (const prepare of [prepareInputExport, prepareOutputExport]) {
    const file = prepare(input.junctions, 40, 0, 'bidirectional', input.groupIds);
    assert.equal(validateImportedConfig(file).isValid, true);
    assert.deepEqual(file.groupIds, { 1: 1 });
    assert.deepEqual(file.junctions, input.junctions);
    file.groupIds[1] = 0;
    assert.equal(input.groupIds[1], 1);
  }
  for (const groupIds of [{ 99: 0 }, { 1: 42 }, { 1: '1' }, [], null]) {
    assert.equal(validateImportedConfig({ ...input, groupIds }).isValid, false);
  }
  const editor = createCorridorEditor();
  editor.replaceInput(input);
  editor.replaceInput(DEMO_DATA);
  assert.deepEqual(get(editor.groupIds), {});
  editor.undo();
  assert.deepEqual(get(editor.groupIds), { 1: 1 });
});
