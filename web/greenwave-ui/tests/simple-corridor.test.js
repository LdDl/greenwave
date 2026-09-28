import test from 'node:test';
import assert from 'node:assert/strict';
import { inputToSimpleCorridor, simpleCorridorToInput } from '../src/lib/utils/simple-corridor.js';
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
  const project = inputToSimpleCorridor(input);
  assert.deepEqual(project.nodes.map(node => node.id), [0, 7, 21]);
  assert.deepEqual(project.roads.map(road => [road.from, road.to, road.lengthMeters]), [[0, 7, 200], [7, 21, 350]]);
  assert.deepEqual(simpleCorridorToInput(JSON.parse(JSON.stringify(project))), input);
});

test('bidirectional input uses the same junctions, programs and permanent groups for both directions', () => {
  const input = inputConfiguration('bidirectional');
  for (const phase of input.junctions[1].cycle) phase.signal_groups[0].id = 8;
  const project = inputToSimpleCorridor(input);
  assert.equal(project.nodes.length, 3);
  assert.equal(project.roads.length, 2);
  assert.ok(project.roads.every(road => road.forwardLanes === 1 && road.reverseLanes === 1));
  assert.deepEqual(project.nodes[1].signalGroups, [{ id: 8, label: 'G8' }]);
  assert.deepEqual(project.corridor.stops[1], { nodeId: 7, forwardGroupId: 8, reverseGroupId: 8 });
  assert.deepEqual(simpleCorridorToInput(project), input);

  project.corridor.stops.reverse();
  project.corridor.roadIds.reverse();
  const reverse = simpleCorridorToInput(project);
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
  const project = inputToSimpleCorridor(input);
  project.nodes[0].position = { x: 1000, y: -200 };
  project.nodes[1].position = { x: -500, y: 999 };
  assert.deepEqual(simpleCorridorToInput(project), input);
});

test('editing road length and the common program updates input without mutating earlier snapshots', () => {
  const input = inputConfiguration();
  const before = structuredClone(input);
  const project = inputToSimpleCorridor(input);
  project.roads[0].lengthMeters = 240;
  project.nodes[1].program.offset = 8;
  project.nodes[1].program.cycle[0].signal_groups[0].signals[0].duration = 30;
  const output = simpleCorridorToInput(project);
  assert.deepEqual(output.junctions.map(junction => junction.point.y), [50, 290, 640]);
  assert.equal(output.junctions[1].offset, 8);
  assert.equal(output.junctions[1].cycle[0].signal_groups[0].signals[0].duration, 30);
  assert.deepEqual(input, before);
  output.junctions[1].cycle[0].signal_groups[0].signals[0].duration = 1;
  assert.equal(project.nodes[1].program.cycle[0].signal_groups[0].signals[0].duration, 30);
});

test('empty and single-junction inputs do not invent roads or lose programs', () => {
  for (const count of [0, 1]) {
    const input = inputConfiguration();
    input.junctions = input.junctions.slice(0, count);
    const project = inputToSimpleCorridor(input);
    assert.deepEqual(project.roads, []);
    assert.deepEqual(simpleCorridorToInput(project), input);
  }
});

test('older input settings get defaults while an explicit zero intensity is retained', () => {
  const input = inputConfiguration();
  delete input.direction;
  delete input.desiredIntensity;
  assert.deepEqual(inputToSimpleCorridor(input).settings, { desiredSpeed: 42.5, direction: 'forward', desiredIntensity: 1800 });
  input.desiredIntensity = 0;
  assert.equal(inputToSimpleCorridor(input).settings.desiredIntensity, 0);
});

test('unsupported or malformed inputs fail without sorting, flattening groups or mutating the source', () => {
  const cases = [
    { change: input => { input.junctions[1].id = 0; }, error: /duplicate ID/ },
    { change: input => { input.junctions[0].id = -1; }, error: /non-negative integer/ },
    { change: input => { input.junctions[1].point.x = 40; }, error: /common point.x/ },
    { change: input => { input.junctions[1].point.y = 10; }, error: /route order was not changed/ },
    { change: input => { input.junctions[0].cycle[0].signal_groups.push({ id: 1, signals: [{ color: 'RED', duration: 40 }] }); }, error: /multiple groups need explicit selection/ },
    { change: input => { input.junctions[0].cycle[1].signal_groups[0].id = 1; }, error: /same permanent signal group/ },
    { change: input => { input.junctions[0].cycle[0].signal_groups[0].signals[0].duration = 0; }, error: /duration must be positive/ },
    { change: input => { input.junctions[0].cycle[1].id = 0; }, error: /duplicate phase ID/ },
    { change: input => { input.junctions[0].point.y = Infinity; }, error: /coordinates must be finite/ },
    { change: input => { input.desiredSpeed = 0; }, error: /speed must be positive/ },
    { change: input => { input.desiredIntensity = -1; }, error: /intensity must be non-negative/ },
    { change: input => { input.direction = 'reverse'; }, error: /Unknown calculation direction/ },
  ];
  for (const { change, error } of cases) {
    const input = inputConfiguration();
    change(input);
    const before = structuredClone(input);
    assert.throws(() => inputToSimpleCorridor(input), error);
    assert.deepEqual(input, before);
  }
});

test('projection refuses missing objects, mismatched groups and unavailable road directions', () => {
  const cases = [
    { change: project => { project.nodes.pop(); }, error: /missing junction/ },
    { change: project => { project.roads.pop(); }, error: /missing road/ },
    { change: project => { project.roads[0].to = 21; }, error: /consecutive stops/ },
    { change: project => { project.roads[0].forwardLanes = 0; }, error: /travel along the corridor/ },
    { change: project => { project.roads[0].reverseLanes = 0; }, error: /return travel/ },
    { change: project => { project.roads[0].lengthMeters = -1; }, error: /length must be non-negative/ },
    { change: project => { project.corridor.stops[1].reverseGroupId = 99; }, error: /selected group/ },
    { change: project => { project.nodes[0].signalGroups[0].id = 5; }, error: /permanent signal group/ },
    { change: project => { project.corridor.stops[1].nodeId = 0; }, error: /repeats junction/ },
  ];
  for (const { change, error } of cases) {
    const project = inputToSimpleCorridor(inputConfiguration('bidirectional'));
    change(project);
    const before = structuredClone(project);
    assert.throws(() => simpleCorridorToInput(project), error);
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
