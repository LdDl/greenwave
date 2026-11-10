import test from 'node:test';
import assert from 'node:assert/strict';
import { createCorridorEditor } from '../src/lib/stores/corridor.js';
import { parseSharedProject, graphView } from '../src/lib/utils/shared-project.js';
import { intersectionTopology, movementsConflict } from '../src/lib/utils/network-geometry.js';
import { fourWayProject } from './fixtures/four-way-project.js';

test('shared project round trip preserves forbidden movements and rejects broken roads and assignments', () => {
  const project = fourWayProject();
  project.movements[1]['1_back_2_fwd'] = null;
  assert.deepEqual(parseSharedProject(JSON.stringify(project)), project);
  const cases = [
    p => { p.version = 99; },
    p => { p.junctions[1].id = p.junctions[0].id; },
    p => { p.positions[1].x = NaN; },
    p => { p.roads[0].to = 999; },
    p => { p.roads[0].to = p.roads[0].from; },
    p => { p.roads[0].lanes_fwd = 0; },
    p => { p.roads[0].lanes_back = 1.5; },
    p => { p.roads.push({ ...p.roads[0], id: 100 }); },
    p => { p.movements[1]['1_back_2_fwd'] = 99; },
    p => { p.movements[1]['999_back_2_fwd'] = 0; }
  ];
  for (const mutate of cases) {
    const broken = structuredClone(project);
    mutate(broken);
    assert.throws(() => parseSharedProject(broken));
  }
});

test('deleting a junction cleans incident roads and movements, and undo restores them', () => {
  const editor = createCorridorEditor();
  const original = fourWayProject();
  editor.replace(original);
  editor.removeJunction(2);
  const deleted = editor.snapshot();
  assert.equal(deleted.junctions.length, 4);
  assert.equal(deleted.roads.length, 3);
  assert.equal(Object.keys(deleted.movements[1]).length, 6);
  assert.ok(!Object.keys(deleted.movements[1]).some(id => id.startsWith('1_') || id.includes('_1_')));
  editor.undo();
  assert.deepEqual(editor.snapshot(), original);
  editor.redo();
  assert.deepEqual(editor.snapshot(), deleted);
  editor.removeJunction(1);
  assert.equal(editor.snapshot().movements[1], undefined);
});

test('road direction edits clear only unavailable assignments, and undo restores the full network', () => {
  const editor = createCorridorEditor();
  editor.replace(fourWayProject());
  editor.setRoad(1, { lanes_back: 0 });
  assert.ok(!Object.keys(editor.snapshot().movements[1]).some(id => id.startsWith('1_back')));
  editor.setRoad(1, { lanes_back: 1 });
  assert.equal(editor.snapshot().movements[1]['1_back_2_fwd'], undefined);
  assert.equal(editor.snapshot().movements[1]['2_back_3_fwd'], 0);
  editor.setRoad(1, { lanes_back: 0 });
  const before = editor.snapshot();
  editor.reverseRoad(1);
  assert.equal(editor.snapshot().roads[0].from, 2);
  assert.equal(editor.snapshot().movements[1]['1_fwd_2_fwd'], undefined);
  assert.equal(editor.snapshot().movements[1]['2_back_3_fwd'], 0);
  editor.undo();
  assert.deepEqual(editor.snapshot(), before);
});

test('conflicts distinguish crossing, merging, opposing straight and separated right turns', () => {
  const graph = graphView(fourWayProject());
  const { stubs, movements } = intersectionTopology(1, graph.nodes, graph.edges);
  assert.equal(movements.length, 12);
  const movement = (from, to) => movements.find(m => m.inEdgeId === from && m.outEdgeId === to);
  assert.equal(movementsConflict(movement(1, 3), movement(2, 4), stubs), true);
  assert.equal(movementsConflict(movement(1, 3), movement(3, 1), stubs), false);
  assert.equal(movementsConflict(movement(1, 4), movement(2, 1), stubs), false);
  assert.equal(movementsConflict(movement(1, 3), movement(2, 3), stubs), true);
  assert.equal(movementsConflict(movement(1, 3), movement(1, 4), stubs), false);
  assert.equal(movementsConflict(movement(1, 2), movement(3, 1), stubs), true);
});
