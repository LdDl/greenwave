import { derived, get, writable } from 'svelte/store';
import { readProgram } from '../utils/junction-program.js';
import { validateMPScenario } from '../utils/mp-scenario.js';
import { mpProgramUpdate } from '../utils/mp-network.js';
import {
	SHARED_STORAGE_KEY,
	LEGACY_CORRIDOR_KEY,
	LEGACY_NETWORK_KEY,
	emptySharedProject,
	emptyCorridor,
	defaultJunction,
	activeCorridor,
	corridorProjection,
	graphView,
	fromInput,
	parseSharedProject,
	reconcileMovements,
	roadBetween,
	routeMovements,
	assignRouteGroups,
	checkInput
} from '../utils/shared-project.js';

export const CORRIDOR_STORAGE_KEY = SHARED_STORAGE_KEY;
const copy = (value) => structuredClone(value);
const nextId = (items) => Math.max(-1, ...items.map((item) => item.id)) + 1;

function calculationData(document) {
	const { input, error } = corridorProjection(document);
	const { junctions, desiredSpeed, direction, groupIds, reverseGroupIds } = input;
	return JSON.stringify({
		active: document.activeCorridorId,
		junctions,
		desiredSpeed,
		direction,
		groupIds,
		reverseGroupIds,
		error
	});
}

function updateRouteJunctions(document, list) {
	const route = activeCorridor(document);
	const before = corridorProjection(document).input.junctions;
	checkInput({ ...corridorProjection(document).input, junctions: list });
	if (
		list.some(
			(node, index) =>
				node.point.x !== list[0].point.x || (index > 0 && node.point.y < list[index - 1].point.y)
		)
	)
		throw new Error('Keep corridor junctions in increasing distance order.');
	for (const node of list) {
		const program = copy(node);
		delete program.point;
		const existing = document.junctions.findIndex((item) => item.id === node.id);
		if (existing >= 0) document.junctions[existing] = program;
		else {
			document.junctions.push(program);
			document.positions[node.id] = { x: 100 + document.junctions.length * 180, y: 100 };
		}
	}
	route.nodeIds = list.map((node) => node.id);
	if (list.length) route.origin = copy(list[0].point);
	for (let index = 1; index < list.length; index++) {
		const previous = list[index - 1];
		const node = list[index];
		const road = roadBetween(document, previous.id, node.id);
		const beforeNode = before.find((item) => item.id === node.id);
		const beforePrevious = before.find((item) => item.id === previous.id);
		if (
			road &&
			(node.point.y !== beforeNode?.point.y || previous.point.y !== beforePrevious?.point.y)
		)
			road.lengthMeters = node.point.y - previous.point.y;
		else if (!road && (!beforeNode || !beforePrevious))
			document.roads.push({
				id: nextId(document.roads),
				from: previous.id,
				to: node.id,
				lengthMeters: node.point.y - previous.point.y,
				lanes_fwd: 1,
				lanes_back: route.direction === 'bidirectional' ? 1 : 0
			});
	}
	for (const key of ['groupIds', 'reverseGroupIds'])
		if (route[key])
			route[key] = Object.fromEntries(
				Object.entries(route[key]).filter(([id]) => route.nodeIds.includes(Number(id)))
			);
	if (route.automatic) assignRouteGroups(document);
}

export function createCorridorEditor() {
	let current = emptySharedProject();
	let past = [];
	let future = [];
	let transaction = null;
	let storage = null;
	let connected = false;
	let initialized = false;
	let preserveSaved = false;
	const project = writable(current);
	const history = writable({ undo: false, redo: false });
	const persistence = writable({ state: 'memory', message: 'Changes are kept in memory' });
	const calculationRevision = writable(0);
	const legacyNetworkAvailable = writable(false);
	const projection = derived(project, (document) => corridorProjection(document));
	const input = derived(projection, (value) => value.input);
	const routeIssue = derived(projection, (value) => value.error);
	const graph = derived(project, graphView);

	function status() {
		history.set({ undo: past.length > 0, redo: future.length > 0 });
	}
	function persist() {
		if (!connected || preserveSaved || transaction) return;
		try {
			parseSharedProject(current);
		} catch (cause) {
			persistence.set({ state: 'error', message: `Project could not be saved. ${cause.message}` });
			return;
		}
		try {
			storage.setItem(SHARED_STORAGE_KEY, JSON.stringify(current));
			persistence.set({ state: 'saved', message: 'Saved in this browser' });
		} catch {
			persistence.set({
				state: 'error',
				message: 'Browser save unavailable. Export project to keep a copy.'
			});
		}
	}
	function publish(next) {
		const changed = calculationData(current) !== calculationData(next);
		current = next;
		project.set(current);
		if (changed) calculationRevision.update((value) => value + 1);
		status();
		persist();
	}
	function change(mutator) {
		const next = copy(current);
		mutator(next);
		reconcileMovements(next);
		if (JSON.stringify(next) === JSON.stringify(current)) return;
		initialized = true;
		preserveSaved = false;
		if (!transaction) {
			past = [...past.slice(-99), current];
			future = [];
		}
		publish(next);
	}
	function begin() {
		if (!transaction) transaction = current;
	}
	function finish() {
		if (!transaction) return;
		if (JSON.stringify(transaction) !== JSON.stringify(current)) {
			past = [...past.slice(-99), transaction];
			future = [];
		}
		transaction = null;
		status();
		persist();
	}
	function undo() {
		finish();
		if (!past.length) return;
		future.push(current);
		preserveSaved = false;
		publish(past.pop());
	}
	function redo() {
		finish();
		if (!future.length) return;
		past.push(current);
		preserveSaved = false;
		publish(future.pop());
	}
	function replace(value) {
		const next = parseSharedProject(value);
		finish();
		initialized = true;
		preserveSaved = false;
		change((document) => {
			for (const key of Object.keys(document)) delete document[key];
			Object.assign(document, next);
		});
		persist();
	}
	function replaceInput(value) {
		const previous = get(input);
		const next = { ...previous, ...value };
		for (const key of ['groupIds', 'reverseGroupIds'])
			if (value.junctions !== undefined && value[key] === undefined) delete next[key];
		replace(fromInput(next));
	}
	function setField(name, value) {
		change((document) => {
			const route = activeCorridor(document);
			if (name === 'junctions') updateRouteJunctions(document, copy(value));
			else {
				route[name] = copy(value);
				if (name === 'direction' && route.automatic) {
					for (let index = 1; index < route.nodeIds.length; index++) {
						const road = roadBetween(document, route.nodeIds[index - 1], route.nodeIds[index]);
						if (road)
							road.lanes_back = value === 'bidirectional' ? Math.max(1, road.lanes_back) : 0;
					}
					assignRouteGroups(document);
				}
				if ((name === 'groupIds' || name === 'reverseGroupIds') && route.automatic)
					assignRouteGroups(document);
			}
		});
	}
	function field(name, fallback) {
		const value = derived(input, (value) => value[name] ?? fallback);
		return {
			subscribe: value.subscribe,
			set: (next) => setField(name, next),
			update: (mutate) => setField(name, mutate(copy(get(input)[name])))
		};
	}
	function connectStorage(target) {
		storage = target;
		try {
			legacyNetworkAvailable.set(storage.getItem(LEGACY_NETWORK_KEY) !== null);
		} catch {
			legacyNetworkAvailable.set(false);
		}
		if (!initialized) {
			initialized = true;
			try {
				const saved = storage.getItem(SHARED_STORAGE_KEY);
				const legacy =
					saved === null
						? (storage.getItem(LEGACY_CORRIDOR_KEY) ?? storage.getItem(LEGACY_NETWORK_KEY))
						: null;
				if (saved !== null || legacy !== null) publish(parseSharedProject(saved ?? legacy));
			} catch {
				preserveSaved = true;
				persistence.set({
					state: 'error',
					message:
						'Saved project could not be opened. Import a backup or start a new project. The saved data was kept.'
				});
			}
		}
		connected = true;
		persist();
		return () => {
			finish();
			connected = false;
			storage = null;
		};
	}
	function manual(document) {
		for (const route of document.corridors) route.automatic = false;
	}
	function setRoad(id, patch) {
		const next = copy(current);
		const road = next.roads.find((road) => road.id === id);
		if (!road) throw new Error('Road no longer exists.');
		Object.assign(road, patch);
		manual(next);
		reconcileMovements(next);
		parseSharedProject(next);
		change((document) => {
			document.roads = next.roads;
			document.corridors = next.corridors;
			document.movements = next.movements;
		});
	}
	const canvas = {
		begin,
		finish,
		change: (mutate) => {
			const view = copy(get(graph));
			mutate(view);
			change((document) => {
				for (const node of view.nodes) document.positions[node.id] = { x: node.x, y: node.y };
			});
		}
	};
	const editor = {
		project: { subscribe: project.subscribe },
		input,
		history,
		persistence,
		calculationRevision,
		graph,
		routeIssue,
		legacyNetworkAvailable,
		junctions: field('junctions', []),
		desiredSpeed: field('desiredSpeed'),
		desiredIntensity: field('desiredIntensity'),
		direction: field('direction'),
		groupIds: field('groupIds', {}),
		reverseGroupIds: field('reverseGroupIds', {}),
		begin,
		finish,
		undo,
		redo,
		replace,
		replaceInput,
		connectStorage,
		canvas,
		snapshot: () => copy(current),
		inputSnapshot: () => copy(get(input)),
		setMPScenario(scenario) {
			validateMPScenario(scenario);
			change((document) => {
				document.mpScenario = copy(scenario);
			});
		},
		applyMPPrograms(proposals, expectedProject) {
			if (JSON.stringify(current) !== expectedProject)
				throw new Error(
					'The project changed after simulation. Run MP again before applying programs.'
				);
			const next = mpProgramUpdate(current, proposals);
			finish();
			change((document) => Object.assign(document, next));
		},
		readLegacyNetwork() {
			const saved = storage?.getItem(LEGACY_NETWORK_KEY);
			if (saved === null || saved === undefined)
				throw new Error('No saved standalone network was found.');
			return parseSharedProject(saved);
		},
		rename(name) {
			if (!name.trim() || name.length > 120)
				throw new Error('Use a project name of 1 to 120 characters.');
			change((document) => {
				document.name = name.trim();
			});
		},
		applyOffsets(junctionOffsets, expectedRevision) {
			if (expectedRevision !== get(calculationRevision))
				throw new Error(
					'Input changed after optimization. Optimize again before applying offsets.'
				);
			const route = activeCorridor(current);
			if (
				!Array.isArray(junctionOffsets) ||
				junctionOffsets.length !== route.nodeIds.length ||
				junctionOffsets.length < 2
			)
				throw new Error('The result must contain an offset for every corridor junction.');
			const offsets = new Map();
			for (const node of junctionOffsets) {
				if (
					!route.nodeIds.includes(node?.id) ||
					offsets.has(node.id) ||
					!Number.isSafeInteger(node.offset)
				)
					throw new Error('The result contains an invalid junction or offset.');
				offsets.set(node.id, node.offset);
			}
			checkInput(get(input));
			if (get(input).junctions.every((node) => (node.offset ?? 0) === offsets.get(node.id)))
				return false;
			finish();
			change((document) => {
				for (const node of document.junctions)
					if (offsets.has(node.id)) node.offset = offsets.get(node.id);
			});
			return true;
		},
		saveJunction(junction, isNew = false, groupId, reverseGroupId, assignments) {
			const next = copy(current);
			const program = readProgram(junction);
			const route = activeCorridor(next);
			const member = route.nodeIds.includes(junction.id);
			if (member || isNew) {
				const nodes = corridorProjection(next).input.junctions;
				if (isNew) {
					if (next.junctions.some((node) => node.id === junction.id))
						throw new Error('Junction ID already exists.');
					nodes.push(copy(junction));
				} else nodes[nodes.findIndex((node) => node.id === junction.id)] = copy(junction);
				updateRouteJunctions(next, nodes);
				for (const [key, value] of [
					['groupIds', groupId],
					['reverseGroupIds', reverseGroupId]
				]) {
					if (
						value !== undefined &&
						(value !== null || route[key] || program.groupIds.length > 1)
					) {
						if (value !== null && !program.groupIds.includes(value))
							throw new Error('Selected corridor group does not exist.');
						route[key] = { ...route[key] };
						if (value === null) delete route[key][junction.id];
						else route[key][junction.id] = value;
					}
				}
			} else {
				const index = next.junctions.findIndex((node) => node.id === junction.id);
				if (index < 0) throw new Error('Junction no longer exists.');
				const node = copy(junction);
				delete node.point;
				next.junctions[index] = node;
			}
			if (assignments !== undefined) next.movements[junction.id] = copy(assignments);
			else if (route.automatic) assignRouteGroups(next);
			parseSharedProject(next);
			change((document) => Object.assign(document, next));
		},
		removeJunction(id) {
			editor.remove(id, null);
		},
		remove(nodeId, roadId) {
			change((document) => {
				manual(document);
				if (nodeId !== null) {
					document.junctions = document.junctions.filter((node) => node.id !== nodeId);
					document.roads = document.roads.filter(
						(road) => road.from !== nodeId && road.to !== nodeId
					);
					for (const route of document.corridors) {
						route.nodeIds = route.nodeIds.filter((id) => id !== nodeId);
						for (const key of ['groupIds', 'reverseGroupIds'])
							if (route[key]) delete route[key][nodeId];
					}
				} else document.roads = document.roads.filter((road) => road.id !== roadId);
			});
		},
		addNode(x, y) {
			const id = nextId(current.junctions);
			change((document) => {
				manual(document);
				document.junctions.push(defaultJunction(id));
				document.positions[id] = { x, y };
			});
			return id;
		},
		addRoad(from, to, lanesFwd, lanesBack, lengthMeters = null) {
			if (
				from === to ||
				!current.junctions.some((node) => node.id === from) ||
				!current.junctions.some((node) => node.id === to) ||
				roadBetween(current, from, to)
			)
				return null;
			const id = nextId(current.roads);
			const next = copy(current);
			next.roads.push({ id, from, to, lanes_fwd: lanesFwd, lanes_back: lanesBack, lengthMeters });
			manual(next);
			parseSharedProject(next);
			change((document) => {
				document.roads = next.roads;
				document.corridors = next.corridors;
			});
			return id;
		},
		setRoad,
		setRoadLength(id, length) {
			if (!Number.isFinite(length) || length < 0)
				throw new Error('Road length must be a non-negative number.');
			setRoad(id, { lengthMeters: length });
		},
		reverseRoad(id) {
			const road = current.roads.find((road) => road.id === id);
			if (!road || road.lanes_back > 0) return;
			setRoad(id, { from: road.to, to: road.from });
		},
		selectCorridor(id) {
			if (!current.corridors.some((route) => route.id === id))
				throw new Error('Corridor no longer exists.');
			change((document) => {
				document.activeCorridorId = id;
			});
		},
		addCorridor(name) {
			if (!name.trim() || name.length > 120)
				throw new Error('Use a corridor name of 1 to 120 characters.');
			const id = nextId(current.corridors);
			change((document) => {
				document.corridors.push(emptyCorridor(id, name.trim()));
				document.activeCorridorId = id;
			});
			return id;
		},
		renameCorridor(id, name) {
			if (!name.trim() || name.length > 120)
				throw new Error('Use a corridor name of 1 to 120 characters.');
			change((document) => {
				const route = document.corridors.find((route) => route.id === id);
				if (!route) throw new Error('Corridor no longer exists.');
				route.name = name.trim();
			});
		},
		removeCorridor(id) {
			if (current.corridors.length === 1) throw new Error('Keep at least one corridor.');
			change((document) => {
				document.corridors = document.corridors.filter((route) => route.id !== id);
				if (document.activeCorridorId === id) document.activeCorridorId = document.corridors[0].id;
			});
		},
		setRoute(nodeIds) {
			if (
				new Set(nodeIds).size !== nodeIds.length ||
				nodeIds.some((id) => !current.junctions.some((node) => node.id === id))
			)
				throw new Error('Choose distinct junctions from the network.');
			change((document) => {
				const route = activeCorridor(document);
				route.nodeIds = copy(nodeIds);
				route.automatic = false;
				for (const key of ['groupIds', 'reverseGroupIds'])
					if (route[key])
						route[key] = Object.fromEntries(
							Object.entries(route[key]).filter(([id]) => nodeIds.includes(Number(id)))
						);
			});
		},
		routeMovements(id) {
			return routeMovements(current, id);
		}
	};
	canvas.addNode = editor.addNode;
	return editor;
}

export const corridorEditor = createCorridorEditor();
export const corridorProject = corridorEditor.project;
export const corridorGraph = corridorEditor.graph;
export const corridorHistory = corridorEditor.history;
export const corridorPersistence = corridorEditor.persistence;
export const corridorGroupIds = corridorEditor.groupIds;
export const corridorReverseGroupIds = corridorEditor.reverseGroupIds;
export const corridorIssue = corridorEditor.routeIssue;
export const corridorNodes = derived(corridorGraph, (graph) => graph.nodes);
export const corridorEdges = derived(corridorGraph, (graph) => graph.edges);
export const selectedCorridor = derived(corridorProject, activeCorridor);
