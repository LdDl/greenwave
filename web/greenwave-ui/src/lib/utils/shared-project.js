import {
	parseProject as parseLegacyNetwork,
	intersectionTopology,
	movementsConflict,
	MAX_LANES
} from './network-project.js';
import { validateImportedConfig } from './export-import.js';
import { corridorGroupId, reverseCorridorGroupId, programTimeline } from './junction-program.js';
import { validateMPScenario } from './mp-scenario.js';

export const SHARED_FORMAT = 'greenwave-project';
export const SHARED_STORAGE_KEY = 'greenwave.project.v1';
export const LEGACY_CORRIDOR_KEY = 'greenwave.corridor.v1';
export const LEGACY_NETWORK_KEY = 'greenwave.network.v1';

const copy = (value) => structuredClone(value);
const record = (value) => value !== null && typeof value === 'object' && !Array.isArray(value);
const validId = (value) => Number.isSafeInteger(value) && value >= 0;
function requireValue(value, message) {
	if (!value) throw new Error(message);
}
export function checkInput(input) {
	const validation = validateImportedConfig(input);
	requireValue(validation.isValid, validation.errors.join(' '));
}
export function defaultJunction(id, label = `Junction ${id + 1}`) {
	return {
		id,
		label,
		offset: 0,
		cycle: [
			{
				id: 0,
				signal_groups: [
					{
						id: 0,
						signals: [
							{ color: 'GREEN', duration: 30 },
							{ color: 'RED', duration: 20 }
						]
					}
				]
			}
		]
	};
}
export function emptyCorridor(id = 0, name = 'Corridor 1') {
	return {
		id,
		name,
		nodeIds: [],
		desiredSpeed: 40,
		desiredIntensity: 1800,
		direction: 'forward',
		origin: { x: 0, y: 0 },
		automatic: false
	};
}
export function emptySharedProject() {
	return {
		format: SHARED_FORMAT,
		version: 1,
		name: 'Untitled network',
		junctions: [],
		positions: {},
		roads: [],
		movements: {},
		corridors: [emptyCorridor()],
		activeCorridorId: 0
	};
}
export function activeCorridor(project) {
	return project.corridors.find((route) => route.id === project.activeCorridorId);
}
export function roadBetween(project, from, to) {
	return project.roads.find(
		(road) => (road.from === from && road.to === to) || (road.to === from && road.from === to)
	);
}
export function graphView(project) {
	return {
		error: '',
		nodes: project.junctions.map((node) => ({
			id: node.id,
			label: node.label,
			...project.positions[node.id]
		})),
		edges: copy(project.roads)
	};
}
export function routeMovements(project, nodeId, route = activeCorridor(project)) {
	const index = route.nodeIds.indexOf(nodeId);
	if (index <= 0 || index >= route.nodeIds.length - 1) return {};
	const incoming = roadBetween(project, route.nodeIds[index - 1], nodeId);
	const outgoing = roadBetween(project, nodeId, route.nodeIds[index + 1]);
	if (!incoming || !outgoing) return {};
	const graph = graphView(project);
	const movements = intersectionTopology(nodeId, graph.nodes, graph.edges).movements;
	return {
		forward: movements.find((m) => m.inEdgeId === incoming.id && m.outEdgeId === outgoing.id)?.id,
		reverse: movements.find((m) => m.inEdgeId === outgoing.id && m.outEdgeId === incoming.id)?.id
	};
}
export function assignRouteGroups(project, route = activeCorridor(project)) {
	for (const id of route.nodeIds) {
		const node = project.junctions.find((node) => node.id === id);
		if (!node) continue;
		const moves = routeMovements(project, id, route);
		const assignments = (project.movements[id] ??= {});
		if (moves.forward) assignments[moves.forward] = corridorGroupId(node, route.groupIds);
		if (moves.reverse)
			assignments[moves.reverse] = reverseCorridorGroupId(
				node,
				route.groupIds,
				route.reverseGroupIds
			);
	}
}

export function corridorProjection(project, route = activeCorridor(project)) {
	const errors = [];
	let distance = route.origin.y;
	const junctions = route.nodeIds
		.map((id, index) => {
			const node = project.junctions.find((node) => node.id === id);
			if (!node) {
				errors.push(`Corridor references missing junction ${id}.`);
				return null;
			}
			if (index > 0) {
				const from = route.nodeIds[index - 1];
				const road = roadBetween(project, from, id);
				if (!road)
					errors.push(`Connect junctions ${from} and ${id}, or change the corridor route.`);
				else {
					if (road.lengthMeters === null)
						errors.push(`Set the length of road ${road.id} before calculation.`);
					else distance += road.lengthMeters;
					const forward = road.from === from ? road.lanes_fwd : road.lanes_back;
					const reverse = road.from === from ? road.lanes_back : road.lanes_fwd;
					if (!forward) errors.push(`Road ${road.id} does not allow travel along the corridor.`);
					if (route.direction === 'bidirectional' && !reverse)
						errors.push(`Road ${road.id} does not allow reverse travel.`);
				}
			}
			const moves = routeMovements(project, id, route);
			for (const [direction, movementId, groupId] of [
				['forward', moves.forward, corridorGroupId(node, route.groupIds)],
				[
					'reverse',
					moves.reverse,
					reverseCorridorGroupId(node, route.groupIds, route.reverseGroupIds)
				]
			]) {
				if (direction === 'reverse' && route.direction !== 'bidirectional') continue;
				if (movementId && project.movements[id]?.[movementId] !== groupId)
					errors.push(
						`${node.label}: assign the ${direction} corridor movement to its selected group.`
					);
			}
			return { ...copy(node), point: { x: route.origin.x, y: distance } };
		})
		.filter(Boolean);
	const { desiredSpeed, desiredIntensity, direction, groupIds, reverseGroupIds } = route;
	return {
		input: {
			junctions,
			desiredSpeed,
			desiredIntensity,
			direction,
			...(groupIds === undefined ? {} : { groupIds: copy(groupIds) }),
			...(reverseGroupIds === undefined ? {} : { reverseGroupIds: copy(reverseGroupIds) })
		},
		error: [...new Set(errors)].join(' ')
	};
}

export function fromInput(input, positions = {}) {
	const normalized = { desiredSpeed: 40, desiredIntensity: 1800, direction: 'forward', ...input };
	checkInput(normalized);
	const project = emptySharedProject();
	const route = activeCorridor(project);
	Object.assign(route, {
		desiredSpeed: normalized.desiredSpeed,
		desiredIntensity: normalized.desiredIntensity,
		direction: normalized.direction,
		automatic: true,
		origin: copy(normalized.junctions[0]?.point ?? { x: 0, y: 0 })
	});
	for (const key of ['groupIds', 'reverseGroupIds'])
		if (normalized[key] !== undefined) route[key] = copy(normalized[key]);
	project.junctions = normalized.junctions.map(({ point, ...node }, index) => {
		project.positions[node.id] = copy(positions[node.id] ?? { x: 100 + index * 180, y: 100 });
		route.nodeIds.push(node.id);
		if (index > 0) {
			const previous = normalized.junctions[index - 1];
			project.roads.push({
				id: index,
				from: previous.id,
				to: node.id,
				lengthMeters: Math.hypot(point.x - previous.point.x, point.y - previous.point.y),
				lanes_fwd: 1,
				lanes_back: route.direction === 'bidirectional' ? 1 : 0
			});
		}
		return copy(node);
	});
	assignRouteGroups(project);
	return project;
}

export function fromLegacyNetwork(value) {
	const legacy = parseLegacyNetwork(value);
	const project = emptySharedProject();
	project.name = legacy.name;
	project.roads = legacy.edges.map((edge) => ({ ...edge, lengthMeters: null }));
	project.junctions = legacy.nodes.map((node) => {
		project.positions[node.id] = { x: node.x, y: node.y };
		const config = legacy.intersections[node.id];
		if (!config) return defaultJunction(node.id, node.label);
		const assigned = Object.entries(config.movState).filter(([, state]) => state.groupIds.length);
		const groups = assigned.length
			? assigned.map(([movement, state], id) => ({ id, movement, stages: state.groupIds }))
			: [{ id: 0, movement: null, stages: [] }];
		project.movements[node.id] = Object.fromEntries(
			groups.filter((group) => group.movement).map((group) => [group.movement, group.id])
		);
		return {
			id: node.id,
			label: node.label,
			offset: 0,
			cycle: config.groups.map((stage) => ({
				id: stage.id,
				signal_groups: groups.map((group) => ({
					id: group.id,
					signals: [
						{
							color: group.stages.includes(stage.id) ? 'GREEN' : 'RED',
							duration: stage.greenDuration
						}
					]
				}))
			}))
		};
	});
	project.migration = {
		source: copy(typeof value === 'string' ? JSON.parse(value) : value),
		notes: [
			'Imported movement stages were converted into phases, with a permanent signal group for each assigned movement.',
			'Road lengths were not stored. Set them before calculation. No yellow or clearance intervals were invented.'
		]
	};
	return project;
}

export function reconcileMovements(project) {
	const graph = graphView(project);
	for (const node of project.junctions) {
		const valid = new Set(
			intersectionTopology(node.id, graph.nodes, graph.edges).movements.map((m) => m.id)
		);
		if (project.movements[node.id])
			project.movements[node.id] = Object.fromEntries(
				Object.entries(project.movements[node.id]).filter(([id]) => valid.has(id))
			);
	}
	const ids = new Set(project.junctions.map((node) => String(node.id)));
	for (const map of [project.positions, project.movements])
		for (const id of Object.keys(map)) if (!ids.has(id)) delete map[id];
}

export function parseSharedProject(value) {
	const data = typeof value === 'string' ? JSON.parse(value) : copy(value);
	if (data?.format === 'greenwave-network') return fromLegacyNetwork(data);
	if (data?.format === 'greenwave-corridor' && data.version === 1) {
		requireValue(record(data.positions), 'Invalid saved canvas positions.');
		for (const [id, position] of Object.entries(data.positions))
			requireValue(
				data.input?.junctions?.some((node) => String(node.id) === id) &&
					Number.isFinite(position?.x) &&
					Number.isFinite(position?.y),
				'Invalid saved canvas position.'
			);
		return parseSharedProject(fromInput(data.input, data.positions));
	}
	if (Array.isArray(data?.junctions) && !data.format) return parseSharedProject(fromInput(data));
	requireValue(
		data?.format === SHARED_FORMAT && data.version === 1,
		'Choose a supported Greenwave project or input JSON file.'
	);
	requireValue(
		typeof data.name === 'string' && data.name.trim().length > 0 && data.name.length <= 120,
		'Project needs a name of 1 to 120 characters.'
	);
	requireValue(
		Array.isArray(data.junctions) &&
			Array.isArray(data.roads) &&
			record(data.positions) &&
			record(data.movements) &&
			Array.isArray(data.corridors) &&
			data.corridors.length > 0,
		'Project structure is incomplete.'
	);
	checkInput({
		junctions: data.junctions.map((node) => ({ ...node, point: { x: 0, y: 0 } })),
		desiredSpeed: 40
	});
	const ids = new Set(data.junctions.map((node) => node.id));
	requireValue(
		Object.keys(data.positions).length === ids.size,
		'Every junction needs one canvas position.'
	);
	for (const [id, position] of Object.entries(data.positions))
		requireValue(
			String(Number(id)) === id &&
				ids.has(Number(id)) &&
				Number.isFinite(position?.x) &&
				Number.isFinite(position?.y),
			'Invalid canvas position.'
		);
	const roadIds = new Set();
	const pairs = new Set();
	for (const road of data.roads) {
		requireValue(
			validId(road?.id) && !roadIds.has(road.id),
			'Road IDs must be unique non-negative integers.'
		);
		requireValue(
			ids.has(road.from) && ids.has(road.to) && road.from !== road.to,
			'Road must connect two existing junctions.'
		);
		const pair = [road.from, road.to].sort((a, b) => a - b).join(':');
		requireValue(!pairs.has(pair), 'Only one road is allowed between a pair of junctions.');
		requireValue(
			Number.isInteger(road.lanes_fwd) &&
				road.lanes_fwd >= 1 &&
				road.lanes_fwd <= MAX_LANES &&
				Number.isInteger(road.lanes_back) &&
				road.lanes_back >= 0 &&
				road.lanes_back <= MAX_LANES,
			'Invalid lane counts.'
		);
		requireValue(
			road.lengthMeters === null || (Number.isFinite(road.lengthMeters) && road.lengthMeters >= 0),
			'Road length must be non-negative or unset.'
		);
		roadIds.add(road.id);
		pairs.add(pair);
	}
	const corridorIds = new Set();
	for (const route of data.corridors) {
		requireValue(
			validId(route?.id) &&
				!corridorIds.has(route.id) &&
				typeof route.name === 'string' &&
				route.name.trim().length > 0 &&
				route.name.length <= 120,
			'Corridors need unique IDs and names.'
		);
		requireValue(
			Array.isArray(route.nodeIds) &&
				new Set(route.nodeIds).size === route.nodeIds.length &&
				route.nodeIds.every((id) => ids.has(id)),
			'Corridor must reference distinct existing junctions.'
		);
		requireValue(
			Number.isFinite(route.origin?.x) &&
				Number.isFinite(route.origin?.y) &&
				typeof route.automatic === 'boolean',
			'Invalid corridor origin or editing mode.'
		);
		checkInput({
			...route,
			junctions: route.nodeIds.map((id) => ({
				...data.junctions.find((node) => node.id === id),
				point: { x: 0, y: 0 }
			}))
		});
		corridorIds.add(route.id);
	}
	requireValue(corridorIds.has(data.activeCorridorId), 'Choose an existing active corridor.');
	if (data.mpScenario !== undefined) validateMPScenario(data.mpScenario);
	const graph = graphView(data);
	for (const [key, assignments] of Object.entries(data.movements)) {
		const node = data.junctions.find((node) => String(node.id) === key);
		requireValue(
			node && record(assignments),
			'Movement assignments reference an unknown junction.'
		);
		const valid = new Set(
			intersectionTopology(node.id, graph.nodes, graph.edges).movements.map((m) => m.id)
		);
		for (const [id, group] of Object.entries(assignments))
			requireValue(
				valid.has(id) &&
					(group === null ||
						node.cycle.every((phase) => phase.signal_groups.some((g) => g.id === group))),
				`Junction ${key}: movement references an unknown road or signal group.`
			);
	}
	return data;
}

export function programMovementConflicts(junction, topology, assignments) {
	let timelines;
	try {
		timelines = new Map(
			junction.cycle[0].signal_groups.map((group) => [
				group.id,
				programTimeline(junction, group.id).segments.filter((signal) =>
					['GREEN', 'GREENPRIORITY', 'YELLOW'].includes(signal.color)
				)
			])
		);
	} catch {
		return [];
	}
	const pairs = [];
	for (let i = 0; i < topology.movements.length; i++)
		for (let j = i + 1; j < topology.movements.length; j++) {
			const a = topology.movements[i],
				b = topology.movements[j];
			const one = timelines.get(assignments[a.id]) ?? [],
				two = timelines.get(assignments[b.id]) ?? [];
			if (
				movementsConflict(a, b, topology.stubs) &&
				one.some((x) => two.some((y) => Math.max(x.start, y.start) < Math.min(x.end, y.end)))
			)
				pairs.push([a, b]);
		}
	return pairs;
}
