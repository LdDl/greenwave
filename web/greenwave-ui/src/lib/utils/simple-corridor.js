// This in-memory model is the first step towards shared editor data.
// It does not read or migrate saved greenwave-network version 1 projects.

function requireValue(condition, message) {
  if (!condition) throw new Error(message);
}

function requireId(id, name) {
  requireValue(Number.isSafeInteger(id) && id >= 0, `${name} must be a non-negative integer.`);
}

function readSettings({ desiredSpeed, desiredIntensity = 1800, direction = 'forward' }) {
  requireValue(Number.isFinite(desiredSpeed) && desiredSpeed > 0, 'Desired speed must be positive.');
  requireValue(Number.isFinite(desiredIntensity) && desiredIntensity >= 0, 'Desired intensity must be non-negative.');
  requireValue(direction === 'forward' || direction === 'bidirectional', 'Unknown calculation direction.');
  return { desiredSpeed, desiredIntensity, direction };
}

function readGroupId(cycle, nodeId) {
  const context = `Junction ${nodeId}`;
  requireValue(Array.isArray(cycle) && cycle.length > 0, `${context}: a program needs at least one phase.`);
  const phaseIds = new Set();
  let groupId;
  for (const phase of cycle) {
    requireId(phase?.id, `${context} phase ID`);
    requireValue(!phaseIds.has(phase.id), `${context}: duplicate phase ID ${phase.id}.`);
    phaseIds.add(phase.id);
    requireValue(Array.isArray(phase.signal_groups) && phase.signal_groups.length === 1,
      `${context}: this adapter supports exactly one signal group per phase; multiple groups need explicit selection.`);
    const group = phase.signal_groups[0];
    requireId(group?.id, `${context} signal group ID`);
    if (groupId === undefined) groupId = group.id;
    requireValue(group.id === groupId, `${context}: every phase must refer to the same permanent signal group.`);
    requireValue(Array.isArray(group.signals) && group.signals.length > 0, `${context}: a group needs signals in every phase.`);
    for (const signal of group.signals) {
      requireValue(Number.isFinite(signal?.duration) && signal.duration > 0, `${context}: signal duration must be positive.`);
      requireValue(typeof signal.color === 'string' && signal.color.length > 0, `${context}: signal color is missing.`);
    }
  }
  return groupId;
}

function indexById(items, name) {
  requireValue(Array.isArray(items), `${name} must be an array.`);
  const result = new Map();
  for (const item of items) {
    requireId(item?.id, `${name} ID`);
    requireValue(!result.has(item.id), `${name}: duplicate ID ${item.id}.`);
    result.set(item.id, item);
  }
  return result;
}

/**
 * Convert a linear Input configuration into nodes, roads, programs and one corridor.
 * Layout coordinates are arbitrary; road lengths and the corridor origin retain meters.
 * Existing cycle data is kept intact until the shared timeline editor is introduced.
 */
export function inputToSimpleCorridor(input) {
  requireValue(input && typeof input === 'object', 'Input configuration is missing.');
  const settings = readSettings(input);
  indexById(input.junctions, 'Junctions');
  const origin = structuredClone(input.junctions[0]?.point ?? { x: 0, y: 0 });
  const nodes = [];
  const roads = [];
  const stops = [];

  for (const [index, junction] of input.junctions.entries()) {
    requireValue(typeof junction.label === 'string' && junction.label.trim().length > 0, `Junction ${junction.id}: label is missing.`);
    requireValue(Number.isFinite(junction.point?.x) && Number.isFinite(junction.point?.y), `Junction ${junction.id}: coordinates must be finite.`);
    requireValue(junction.point.x === origin.x, 'Only a straight corridor with a common point.x is supported; coordinates were not changed.');
    const groupId = readGroupId(junction.cycle, junction.id);
    const offset = junction.offset ?? 0;
    requireValue(Number.isFinite(offset), `Junction ${junction.id}: offset must be finite.`);
    nodes.push({
      id: junction.id,
      label: junction.label,
      position: { x: 100 + index * 180, y: 100 },
      signalGroups: [{ id: groupId, label: `G${groupId}` }],
      program: { offset, cycle: structuredClone(junction.cycle) },
    });
    stops.push({ nodeId: junction.id, forwardGroupId: groupId, reverseGroupId: groupId });
    if (index > 0) {
      const previous = input.junctions[index - 1];
      const lengthMeters = junction.point.y - previous.point.y;
      requireValue(Number.isFinite(lengthMeters) && lengthMeters >= 0,
        'Junctions must follow increasing distance; route order was not changed.');
      roads.push({
        id: index,
        from: previous.id,
        to: junction.id,
        lengthMeters,
        forwardLanes: 1,
        reverseLanes: settings.direction === 'bidirectional' ? 1 : 0,
      });
    }
  }

  return {
    settings,
    nodes,
    roads,
    corridor: { origin, stops, roadIds: roads.map(road => road.id) },
  };
}

/**
 * Project the selected simple corridor back to the existing input shape.
 * Moving a node on the canvas never changes calculation distances or programs.
 */
export function simpleCorridorToInput(project) {
  requireValue(project && typeof project === 'object', 'Corridor project is missing.');
  const settings = readSettings(project.settings ?? {});
  const nodes = indexById(project.nodes, 'Nodes');
  const roads = indexById(project.roads, 'Roads');
  const corridor = project.corridor;
  requireValue(Array.isArray(corridor?.stops) && Array.isArray(corridor?.roadIds), 'Corridor stops and road IDs are required.');
  requireValue(Number.isFinite(corridor.origin?.x) && Number.isFinite(corridor.origin?.y), 'Corridor origin must be finite.');
  requireValue(corridor.roadIds.length === Math.max(0, corridor.stops.length - 1), 'Each pair of stops needs one road.');
  const visited = new Set();
  let distance = corridor.origin.y;
  const junctions = corridor.stops.map((stop, index) => {
    requireId(stop?.nodeId, 'Corridor node ID');
    const node = nodes.get(stop.nodeId);
    requireValue(node, `Corridor references missing junction ${stop.nodeId}.`);
    requireValue(!visited.has(node.id), `Corridor repeats junction ${node.id}.`);
    visited.add(node.id);
    requireValue(typeof node.label === 'string' && node.label.trim().length > 0, `Junction ${node.id}: label is missing.`);
    const groupId = readGroupId(node.program?.cycle, node.id);
    requireValue(node.signalGroups?.length === 1 && node.signalGroups[0]?.id === groupId,
      `Junction ${node.id}: program must match its permanent signal group.`);
    requireValue(stop.forwardGroupId === groupId && stop.reverseGroupId === groupId,
      `Junction ${node.id}: selected group does not match its program.`);
    requireValue(Number.isFinite(node.program.offset), `Junction ${node.id}: offset must be finite.`);

    if (index > 0) {
      const roadId = corridor.roadIds[index - 1];
      const road = roads.get(roadId);
      requireValue(road, `Corridor references missing road ${roadId}.`);
      const from = corridor.stops[index - 1].nodeId;
      const followsRoad = road.from === from && road.to === node.id;
      const opposesRoad = road.to === from && road.from === node.id;
      requireValue(followsRoad || opposesRoad, `Road ${road.id} does not connect consecutive stops.`);
      requireValue(Number.isInteger(road.forwardLanes) && road.forwardLanes >= 0 && Number.isInteger(road.reverseLanes) && road.reverseLanes >= 0,
        `Road ${road.id}: lane counts must be non-negative integers.`);
      const lanes = followsRoad ? road.forwardLanes : road.reverseLanes;
      const reverseLanes = followsRoad ? road.reverseLanes : road.forwardLanes;
      requireValue(lanes > 0, `Road ${road.id} does not allow travel along the corridor.`);
      requireValue(settings.direction !== 'bidirectional' || reverseLanes > 0, `Road ${road.id} does not allow return travel.`);
      requireValue(Number.isFinite(road.lengthMeters) && road.lengthMeters >= 0, `Road ${road.id}: length must be non-negative.`);
      distance += road.lengthMeters;
      requireValue(Number.isFinite(distance), 'Corridor distance exceeds the supported range.');
    }

    return {
      id: node.id,
      label: node.label,
      offset: node.program.offset,
      point: { x: corridor.origin.x, y: distance },
      cycle: structuredClone(node.program.cycle),
    };
  });
  return { ...settings, junctions };
}
