import test from 'node:test';
import assert from 'node:assert/strict';
import { fromInput, corridorProjection, parseSharedProject } from '../src/lib/utils/shared-project.js';
import { prepareInputExport, prepareOutputExport } from '../src/lib/utils/export-import.js';

function inputConfiguration(direction = 'forward') {
  return {
    desiredSpeed: 42.5,
    desiredIntensity: 0,
    direction,
    junctions: [0, 7, 21].map((id, index) => ({
      id,
      label: `Junction ${id}`,
      offset: [0, -5, 12][index],
      point: { x: 15, y: [50, 250, 600][index] },
      cycle: [
        {
          id: 0,
          signal_groups: [{ id: 0, signals: [
            { duration: 25, color: 'GREEN', min_duration: 10, max_duration: 40 },
            { duration: 3, color: 'YELLOW' },
            { duration: 12, color: 'RED' },
          ] }],
        },
        {
          id: 1,
          signal_groups: [{ id: 0, signals: [
            { duration: 10, color: 'GREEN', min_duration: 0, max_duration: 15 },
            { duration: 10, color: 'RED' },
          ] }],
        },
      ],
    })),
  };
}

test('input round trip preserves zero IDs, phase boundaries, bounds, offsets, origin and settings', () => {
  const input = inputConfiguration();
  const project = fromInput(input);
  assert.deepEqual(project.junctions.map(node => node.id), [0, 7, 21]);
  assert.deepEqual(project.roads.map(road => [road.from, road.to, road.lengthMeters]), [[0, 7, 200], [7, 21, 350]]);
  assert.deepEqual(corridorProjection(parseSharedProject(JSON.stringify(project))).input, input);
});

test('bidirectional input uses the same junctions, programs and permanent groups for both directions', () => {
  const input = inputConfiguration('bidirectional');
  for (const phase of input.junctions[1].cycle) phase.signal_groups[0].id = 8;
  const project = fromInput(input);
  assert.equal(project.junctions.length, 3);
  assert.equal(project.roads.length, 2);
  assert.ok(project.roads.every(road => road.lanes_fwd === 1 && road.lanes_back === 1));
  assert.equal(project.junctions[1].cycle[0].signal_groups[0].id, 8);
  assert.deepEqual(corridorProjection(parseSharedProject(project)).input, input);

  project.corridors[0].nodeIds.reverse();
  const reverse = corridorProjection(parseSharedProject(project)).input;
  assert.deepEqual(reverse.junctions.map(junction => junction.id), [21, 7, 0]);
  assert.deepEqual(reverse.junctions.map(junction => junction.point.y), [50, 400, 600]);
  for (const junction of reverse.junctions) {
    const original = input.junctions.find(item => item.id === junction.id);
    assert.deepEqual(junction.cycle, original.cycle);
    assert.equal(junction.offset, original.offset);
  }
});

test('canvas layout has no effect on calculation input', () => {
  const input = inputConfiguration();
  const project = fromInput(input);
  project.positions[0] = { x: 1000, y: -200 };
  project.positions[7] = { x: -500, y: 999 };
  assert.deepEqual(corridorProjection(parseSharedProject(project)).input, input);
});

test('editing road length and the common program updates input without mutating earlier snapshots', () => {
  const input = inputConfiguration();
  const before = structuredClone(input);
  const project = fromInput(input);
  project.roads[0].lengthMeters = 240;
  project.junctions[1].offset = 8;
  project.junctions[1].cycle[0].signal_groups[0].signals[0].duration = 30;
  const output = corridorProjection(parseSharedProject(project)).input;
  assert.deepEqual(output.junctions.map(junction => junction.point.y), [50, 290, 640]);
  assert.equal(output.junctions[1].offset, 8);
  assert.equal(output.junctions[1].cycle[0].signal_groups[0].signals[0].duration, 30);
  assert.deepEqual(input, before);
  output.junctions[1].cycle[0].signal_groups[0].signals[0].duration = 1;
  assert.equal(project.junctions[1].cycle[0].signal_groups[0].signals[0].duration, 30);
});

test('empty and single-junction inputs do not invent roads or lose programs', () => {
  for (const count of [0, 1]) {
    const input = inputConfiguration();
    input.junctions = input.junctions.slice(0, count);
    const project = fromInput(input);
    assert.deepEqual(project.roads, []);
    assert.deepEqual(corridorProjection(parseSharedProject(project)).input, input);
  }
});

test('optional input settings get defaults while an explicit zero intensity is retained', () => {
  const input = inputConfiguration();
  delete input.direction;
  delete input.desiredIntensity;
  const route = fromInput(input).corridors[0];
  assert.equal(route.desiredSpeed, 42.5);
  assert.equal(route.direction, 'forward');
  assert.equal(route.desiredIntensity, 1800);
  input.desiredIntensity = 0;
  assert.equal(fromInput(input).corridors[0].desiredIntensity, 0);
});

test('invalid API input fails without modifying the original programs', () => {
  const cases = [
    input => { input.junctions[1].id = 0; },
    input => { input.junctions[0].id = -1; },
    input => { input.junctions[0].cycle[0].signal_groups[0].signals[0].duration = 0; },
    input => { input.junctions[0].cycle[1].id = 0; },
    input => { input.junctions[0].point.y = Infinity; },
    input => { input.desiredSpeed = 0; },
    input => { input.desiredIntensity = -1; },
    input => { input.direction = 'reverse'; }
  ];
  for (const mutate of cases) {
    const input = inputConfiguration();
    mutate(input);
    const before = structuredClone(input);
    assert.throws(() => fromInput(input));
    assert.deepEqual(input, before);
  }
});

test('projection reports missing connections and prohibited travel without mutating the project', () => {
  const cases = [
    { change: project => { project.roads.pop(); }, error: /Connect junctions/ },
    { change: project => { project.roads[0].lanes_back = 0; }, error: /reverse travel/ },
    { change: project => { project.roads[0].lengthMeters = null; }, error: /Set the length/ }
  ];
  for (const { change, error } of cases) {
    const project = fromInput(inputConfiguration('bidirectional'));
    change(project);
    const before = structuredClone(project);
    assert.match(corridorProjection(project).error, error);
    assert.deepEqual(project, before);
  }
});

test('input and result exports preserve all signal groups, optional constraints and metadata', () => {
  const input = inputConfiguration('bidirectional');
  for (const phase of input.junctions[0].cycle) {
    phase.label = `Phase ${phase.id}`;
    phase.signal_groups.push({
      id: 12,
      signals: [{ duration: phase.id === 0 ? 40 : 20, color: 'RED', min_duration: 0, max_duration: 60 }],
    });
  }
  const before = structuredClone(input);
  for (const [prepareExport, type] of [[prepareInputExport, 'input'], [prepareOutputExport, 'output']]) {
    const exported = prepareExport(input.junctions, input.desiredSpeed, input.desiredIntensity, input.direction);
    const parsed = JSON.parse(JSON.stringify(exported));
    assert.equal(parsed.type, type);
    assert.equal(parsed.version, 1);
    assert.equal(parsed.desiredSpeed, input.desiredSpeed);
    assert.equal(parsed.desiredIntensity, 0);
    assert.equal(parsed.direction, 'bidirectional');
    assert.deepEqual(parsed.junctions, input.junctions);
    exported.junctions[0].cycle[0].signal_groups[0].signals[0].min_duration = 24;
    assert.deepEqual(input, before);
  }
});
