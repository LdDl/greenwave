import test from 'node:test';
import assert from 'node:assert/strict';
import { get } from 'svelte/store';
import { createCorridorEditor, CORRIDOR_STORAGE_KEY } from '../src/lib/stores/corridor.js';
import { DEMO_DATA } from '../src/lib/utils/demo-input.js';

function memoryStorage(initial = []) {
  const values = new Map(initial);
  return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value) };
}

test('loading the actual demo creates its graph without reading or overwriting the standalone network', () => {
  const storage = memoryStorage([['greenwave.network.v1', '{legacy network untouched}']]);
  const editor = createCorridorEditor();
  editor.connectStorage(storage);
  editor.replaceInput(DEMO_DATA);
  assert.deepEqual(get(editor.junctions), DEMO_DATA.junctions);
  assert.deepEqual(get(editor.graph).nodes.map(node => node.id), [0, 1, 2, 3]);
  assert.deepEqual(get(editor.graph).edges.map(road => road.lengthMeters), [200, 250, 150]);
  assert.equal(storage.getItem('greenwave.network.v1'), '{legacy network untouched}');
  assert.equal(JSON.parse(storage.getItem(CORRIDOR_STORAGE_KEY)).input.junctions.length, 4);
});

test('diagram and graph commands update the same input, including node zero, with shared undo', () => {
  const editor = createCorridorEditor();
  editor.replaceInput(DEMO_DATA);
  editor.junctions.update(nodes => { nodes[0].label = 'Changed from diagram'; return nodes; });
  assert.equal(get(editor.graph).nodes[0].label, 'Changed from diagram');
  const edited = structuredClone(get(editor.junctions)[0]);
  edited.offset = 13;
  edited.cycle[0].signal_groups[0].signals[0].duration = 31;
  editor.saveJunction(edited);
  assert.equal(get(editor.junctions)[0].offset, 13);
  editor.undo();
  assert.equal(get(editor.junctions)[0].offset, 0);
  assert.equal(get(editor.junctions)[0].label, 'Changed from diagram');
  editor.undo();
  assert.equal(get(editor.graph).nodes[0].label, 'Junction 1');
  editor.redo();
  editor.redo();
  assert.equal(get(editor.junctions)[0].cycle[0].signal_groups[0].signals[0].duration, 31);
  assert.equal(DEMO_DATA.junctions[0].label, 'Junction 1');
  assert.equal(DEMO_DATA.junctions[0].cycle[0].signal_groups[0].signals[0].duration, 30);
});

test('layout drag is one undo action and does not invalidate calculation data', () => {
  const storage = memoryStorage();
  const editor = createCorridorEditor();
  editor.connectStorage(storage);
  editor.replaceInput(DEMO_DATA);
  const original = editor.snapshot();
  const revision = get(editor.calculationRevision);
  editor.canvas.begin();
  for (let i = 0; i < 10; i++) editor.canvas.change(view => { view.nodes[0].x = 500 + i; });
  assert.deepEqual(JSON.parse(storage.getItem(CORRIDOR_STORAGE_KEY)).positions, {});
  editor.canvas.finish();
  assert.equal(get(editor.graph).nodes[0].x, 509);
  assert.equal(get(editor.calculationRevision), revision);
  assert.deepEqual(get(editor.junctions), original.input.junctions);
  editor.undo();
  assert.deepEqual(editor.snapshot(), original);
  editor.redo();
  assert.equal(get(editor.graph).nodes[0].x, 509);
});

test('road length updates accumulated distances and flags calculations as stale, then undoes together', () => {
  const editor = createCorridorEditor();
  editor.replaceInput(DEMO_DATA);
  const revision = get(editor.calculationRevision);
  editor.setRoadLength(1, 300);
  assert.deepEqual(get(editor.junctions).map(node => node.point.y), [0, 300, 550, 700]);
  assert.deepEqual(get(editor.graph).edges.map(road => road.lengthMeters), [300, 250, 150]);
  assert.equal(get(editor.calculationRevision), revision + 1);
  editor.undo();
  assert.deepEqual(get(editor.junctions), DEMO_DATA.junctions);
  assert.throws(() => editor.setRoadLength(1, -1));
  assert.throws(() => editor.setRoadLength(999, 10));
  assert.deepEqual(get(editor.junctions), DEMO_DATA.junctions);
});

test('direction changes use the same nodes and layout, intensity changes do not invalidate waves', () => {
  const editor = createCorridorEditor();
  editor.replaceInput(DEMO_DATA);
  editor.canvas.change(view => { view.nodes[1].y = 200; });
  const nodes = get(editor.graph).nodes;
  const revision = get(editor.calculationRevision);
  editor.desiredIntensity.set(0);
  assert.equal(get(editor.calculationRevision), revision);
  editor.direction.set('bidirectional');
  assert.equal(get(editor.calculationRevision), revision + 1);
  assert.deepEqual(get(editor.graph).nodes, nodes);
  assert.ok(get(editor.graph).edges.every(road => road.lanes_back === 1));
});

test('reload restores programs, settings and canvas layout together', () => {
  const storage = memoryStorage();
  const first = createCorridorEditor();
  first.connectStorage(storage);
  first.replaceInput({ ...DEMO_DATA, direction: 'bidirectional', desiredIntensity: 0 });
  first.canvas.change(view => { view.nodes[1].x = 999; });
  const second = createCorridorEditor();
  second.connectStorage(storage);
  assert.deepEqual(second.snapshot(), first.snapshot());
  assert.equal(get(second.desiredIntensity), 0);
  assert.deepEqual(get(second.history), { undo: false, redo: false });
});

test('import is one atomic undo step and invalid import leaves input, history and storage intact', () => {
  const storage = memoryStorage();
  const editor = createCorridorEditor();
  editor.connectStorage(storage);
  const original = editor.snapshot();
  editor.replaceInput({ ...DEMO_DATA, desiredSpeed: 55, desiredIntensity: 0, direction: 'bidirectional' });
  const loaded = editor.snapshot();
  const saved = storage.getItem(CORRIDOR_STORAGE_KEY);
  assert.throws(() => editor.replaceInput({ ...DEMO_DATA, junctions: [{ id: 0 }] }));
  assert.deepEqual(editor.snapshot(), loaded);
  assert.equal(storage.getItem(CORRIDOR_STORAGE_KEY), saved);
  editor.undo();
  assert.deepEqual(editor.snapshot(), original);
  editor.redo();
  assert.deepEqual(editor.snapshot(), loaded);
});

test('unsupported program timing keeps all imported groups and shows no stale graph', () => {
  const editor = createCorridorEditor();
  editor.replaceInput(DEMO_DATA);
  const input = structuredClone(DEMO_DATA);
  for (const phase of input.junctions[0].cycle) phase.signal_groups.push({ ...structuredClone(phase.signal_groups[0]), id: 5 });
  input.junctions[0].cycle[0].signal_groups[1].signals[0].duration += 1;
  editor.replaceInput(input);
  assert.deepEqual(get(editor.junctions), input.junctions);
  assert.match(get(editor.graph).error, /group durations must match/);
  assert.deepEqual(get(editor.graph).nodes, []);
  editor.undo();
  assert.equal(get(editor.graph).nodes.length, 4);
});

test('corrupt storage is preserved through a visit until explicit replacement or edit', () => {
  const storage = memoryStorage([[CORRIDOR_STORAGE_KEY, '{broken']]);
  const editor = createCorridorEditor();
  const disconnect = editor.connectStorage(storage);
  assert.equal(get(editor.persistence).state, 'error');
  disconnect();
  editor.connectStorage(storage);
  assert.equal(storage.getItem(CORRIDOR_STORAGE_KEY), '{broken');
  editor.replaceInput({ junctions: [], desiredSpeed: 40 });
  assert.equal(JSON.parse(storage.getItem(CORRIDOR_STORAGE_KEY)).format, 'greenwave-corridor');
});

test('new input before storage initialization wins over an older saved corridor', () => {
  const stored = createCorridorEditor();
  stored.replaceInput(DEMO_DATA);
  const storage = memoryStorage([[CORRIDOR_STORAGE_KEY, JSON.stringify(stored.snapshot())]]);
  const editor = createCorridorEditor();
  editor.replaceInput({ junctions: [], desiredSpeed: 45 });
  editor.connectStorage(storage);
  assert.deepEqual(get(editor.junctions), []);
  assert.equal(get(editor.desiredSpeed), 45);
});

test('storage failures and temporarily invalid settings do not destroy the last valid saved input', () => {
  const editor = createCorridorEditor();
  editor.connectStorage({ getItem: () => null, setItem: () => { throw new Error('Quota exceeded'); } });
  editor.replaceInput(DEMO_DATA);
  assert.equal(get(editor.graph).nodes.length, 4);
  assert.equal(get(editor.persistence).state, 'error');

  const storage = memoryStorage();
  const second = createCorridorEditor();
  second.connectStorage(storage);
  second.replaceInput(DEMO_DATA);
  const saved = storage.getItem(CORRIDOR_STORAGE_KEY);
  second.desiredSpeed.set(undefined);
  assert.equal(storage.getItem(CORRIDOR_STORAGE_KEY), saved);
  assert.equal(get(second.persistence).state, 'error');
  second.desiredSpeed.set(50);
  assert.equal(get(second.persistence).state, 'saved');
  assert.equal(JSON.parse(storage.getItem(CORRIDOR_STORAGE_KEY)).input.desiredSpeed, 50);
});
