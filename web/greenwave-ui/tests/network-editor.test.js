import test from 'node:test';
import assert from 'node:assert/strict';
import { get } from 'svelte/store';
import { createNetworkEditor, STORAGE_KEY } from '../src/lib/stores/network.js';
import { demoProject, parseProject, intersectionTopology, movementsConflict } from '../src/lib/utils/network-project.js';

function configuredProject() {
  const project = demoProject();
  const { movements } = intersectionTopology(1, project.nodes, project.edges);
  project.intersections[1] = {
    macro_node_id: 1,
    groups: [{ id: 0, greenDuration: 30 }, { id: 1, greenDuration: 20 }],
    movState: Object.fromEntries(movements.map(m => [m.id, { groupIds: [0, 1] }])),
  };
  return project;
}

function memoryStorage(initial = null) {
  const values = new Map(initial === null ? [] : [[STORAGE_KEY, initial]]);
  return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value) };
}

test('project round trip preserves geometry, durations, multiple assignments and forbidden movements', () => {
  const project = configuredProject();
  project.intersections[1].movState['1_back_2_fwd'].groupIds = [];
  assert.deepEqual(parseProject(JSON.stringify(project)), project);
});

test('import rejects unknown versions, duplicate IDs, invalid directions and dangling references', () => {
  const cases = [
    p => { p.version = 99; },
    p => { p.nodes[1].id = p.nodes[0].id; },
    p => { p.nodes[0].x = NaN; },
    p => { p.edges[0].to = 999; },
    p => { p.edges[0].to = p.edges[0].from; },
    p => { p.edges[0].lanes_fwd = 0; },
    p => { p.edges[0].lanes_back = 1.5; },
    p => { p.edges.push({ ...p.edges[0], id: 100 }); },
    p => { p.intersections[1].groups[0].greenDuration = 0; },
    p => { p.intersections[1].movState['1_back_2_fwd'].groupIds = [99]; },
    p => { p.intersections[1].movState['999_back_2_fwd'] = { groupIds: [0] }; },
    p => { p.intersections[1].macro_node_id = 2; },
  ];
  for (const mutate of cases) {
    const project = configuredProject();
    mutate(project);
    assert.throws(() => parseProject(project));
  }
});

test('invalid import does not change project or history', () => {
  const editor = createNetworkEditor();
  editor.replace(configuredProject());
  const before = editor.snapshot();
  const broken = editor.snapshot();
  broken.edges[0].from = 999;
  assert.throws(() => editor.replace(broken));
  assert.deepEqual(editor.snapshot(), before);
  editor.undo();
  assert.equal(editor.snapshot().nodes.length, 0);
});

test('deletion cleans connected roads and assignments; undo restores the entire document', () => {
  const editor = createNetworkEditor();
  const original = configuredProject();
  editor.replace(original);
  editor.remove(2, null);
  const deleted = editor.snapshot();
  assert.equal(deleted.nodes.length, 4);
  assert.equal(deleted.edges.length, 3);
  assert.equal(Object.keys(deleted.intersections[1].movState).length, 6);
  assert.ok(!Object.keys(deleted.intersections[1].movState).some(id => id.startsWith('1_') || id.includes('_1_')));
  editor.undo();
  assert.deepEqual(editor.snapshot(), original);
  editor.redo();
  assert.deepEqual(editor.snapshot(), deleted);
  editor.remove(1, null);
  assert.deepEqual(editor.snapshot().intersections, {});
});

test('removing a direction clears only unavailable movements, new directions start forbidden', () => {
  const editor = createNetworkEditor();
  editor.replace(configuredProject());
  editor.change(p => { p.edges[0].lanes_back = 0; });
  assert.ok(!Object.keys(editor.snapshot().intersections[1].movState).some(id => id.startsWith('1_back')));
  editor.change(p => { p.edges[0].lanes_back = 1; });
  const state = editor.snapshot().intersections[1].movState;
  assert.deepEqual(state['1_back_2_fwd'].groupIds, []);
  assert.deepEqual(state['2_back_3_fwd'].groupIds, [0, 1]);
});

test('reversing one-way roads resets affected assignments and undo restores them', () => {
  const editor = createNetworkEditor();
  editor.replace(configuredProject());
  editor.change(p => { p.edges[0].lanes_back = 0; });
  const original = editor.snapshot();
  editor.reverseRoad(1);
  const reversed = editor.snapshot();
  assert.equal(reversed.edges[0].from, 2);
  assert.deepEqual(reversed.intersections[1].movState['1_fwd_2_fwd'].groupIds, []);
  assert.deepEqual(reversed.intersections[1].movState['2_back_3_fwd'].groupIds, [0, 1]);
  editor.undo();
  assert.deepEqual(editor.snapshot(), original);
});

test('drag is one undo action and persists only the completed gesture', () => {
  const editor = createNetworkEditor();
  const storage = memoryStorage();
  editor.connectStorage(storage);
  editor.replace(demoProject());
  const original = editor.snapshot();
  editor.begin();
  for (let i = 1; i <= 20; i++) editor.change(p => { p.nodes[0].x = original.nodes[0].x + i; });
  assert.equal(JSON.parse(storage.getItem(STORAGE_KEY)).nodes[0].x, original.nodes[0].x);
  editor.finish();
  assert.equal(JSON.parse(storage.getItem(STORAGE_KEY)).nodes[0].x, original.nodes[0].x + 20);
  editor.undo();
  assert.deepEqual(editor.snapshot(), original);
  editor.redo();
  assert.equal(editor.snapshot().nodes[0].x, original.nodes[0].x + 20);
});

test('new action after undo drops redo and no-op edits do not create history', () => {
  const editor = createNetworkEditor();
  const id = editor.addNode(0, 0);
  editor.begin(); editor.finish();
  editor.change(() => {});
  editor.undo();
  assert.equal(editor.snapshot().nodes.length, 0);
  editor.addNode(10, 10);
  assert.equal(get(editor.history).redo, false);
  assert.equal(editor.snapshot().nodes[0].id, id);
});

test('reload restores complete configurations and newly added IDs remain unique', () => {
  const storage = memoryStorage();
  const first = createNetworkEditor();
  first.connectStorage(storage);
  first.replace(configuredProject());
  const second = createNetworkEditor();
  second.connectStorage(storage);
  assert.deepEqual(second.snapshot(), first.snapshot());
  assert.equal(second.addNode(0, 0), 6);
  assert.equal(second.addRoad(5, 6, 2, 0), 5);
  assert.equal(second.addRoad(6, 5, 1, 0), null);
});

test('corrupt saved data is not overwritten by opening the editor', () => {
  const storage = memoryStorage('{bad json');
  const editor = createNetworkEditor();
  const disconnect = editor.connectStorage(storage);
  assert.equal(storage.getItem(STORAGE_KEY), '{bad json');
  assert.equal(get(editor.persistence).state, 'error');
  disconnect();
  assert.equal(storage.getItem(STORAGE_KEY), '{bad json');
  editor.connectStorage(storage);
  editor.replace(editor.snapshot());
  assert.equal(JSON.parse(storage.getItem(STORAGE_KEY)).format, 'greenwave-network');
});

test('storage failure keeps edits usable and reports export fallback', () => {
  const editor = createNetworkEditor();
  editor.connectStorage({ getItem: () => null, setItem: () => { throw new Error('Quota exceeded'); } });
  editor.addNode(1, 2);
  assert.equal(editor.snapshot().nodes.length, 1);
  assert.equal(get(editor.persistence).state, 'error');
});

test('conflicts distinguish crossing, merging, opposing straight and separated right turns', () => {
  const project = demoProject();
  const { stubs, movements } = intersectionTopology(1, project.nodes, project.edges);
  const movement = (from, to) => movements.find(m => m.inEdgeId === from && m.outEdgeId === to);
  assert.equal(movementsConflict(movement(1, 3), movement(2, 4), stubs), true);
  assert.equal(movementsConflict(movement(1, 3), movement(3, 1), stubs), false);
  assert.equal(movementsConflict(movement(1, 4), movement(2, 1), stubs), false);
  assert.equal(movementsConflict(movement(1, 3), movement(2, 3), stubs), true);
  assert.equal(movementsConflict(movement(1, 3), movement(1, 4), stubs), false);
  assert.equal(movementsConflict(movement(1, 2), movement(3, 1), stubs), true);
});
