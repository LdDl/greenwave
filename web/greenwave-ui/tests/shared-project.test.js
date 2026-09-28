import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { get } from 'svelte/store';
import { createCorridorEditor } from '../src/lib/stores/corridor.js';
import { DEMO_DATA } from '../src/lib/utils/demo-input.js';
import { demoProject, intersectionTopology } from '../src/lib/utils/network-project.js';
import {
	SHARED_STORAGE_KEY,
	LEGACY_CORRIDOR_KEY,
	LEGACY_NETWORK_KEY,
	fromInput,
	fromLegacyNetwork,
	parseSharedProject,
	corridorProjection,
	graphView,
	routeMovements,
	programMovementConflicts
} from '../src/lib/utils/shared-project.js';
import { programTimeline, programStateAt } from '../src/lib/utils/junction-program.js';
const example = JSON.parse(
	readFileSync(new URL('./fixtures/bidirectional-turns.json', import.meta.url), 'utf8')
);
function memoryStorage(entries = []) {
	const values = new Map(entries);
	return {
		getItem: (key) => values.get(key) ?? null,
		setItem: (key, value) => values.set(key, value)
	};
}
function legacyProject() {
	const project = demoProject();
	const movements = intersectionTopology(1, project.nodes, project.edges).movements;
	project.intersections[1] = {
		macro_node_id: 1,
		groups: [
			{ id: 7, greenDuration: 20 },
			{ id: 9, greenDuration: 10 }
		],
		movState: { [movements[0].id]: { groupIds: [7, 9] }, [movements[1].id]: { groupIds: [9] } }
	};
	return project;
}
function branchedEditor() {
	const editor = createCorridorEditor();
	editor.replaceInput(DEMO_DATA);
	const id = editor.addNode(350, 350);
	editor.addRoad(1, id, 1, 1, 125);
	return { editor, id };
}

test('saved corridor migrates with node zero, all programs and layout, retaining both old storage keys', () => {
	const legacy = {
		format: 'greenwave-corridor',
		version: 1,
		input: example,
		positions: { 0: { x: 800, y: 700 } }
	};
	const network = JSON.stringify(legacyProject());
	const storage = memoryStorage([
		[LEGACY_CORRIDOR_KEY, JSON.stringify(legacy)],
		[LEGACY_NETWORK_KEY, network]
	]);
	const editor = createCorridorEditor();
	editor.connectStorage(storage);
	assert.deepEqual(editor.inputSnapshot(), {
		junctions: example.junctions,
		desiredSpeed: example.desiredSpeed,
		desiredIntensity: example.desiredIntensity,
		direction: example.direction,
		groupIds: example.groupIds,
		reverseGroupIds: example.reverseGroupIds
	});
	assert.deepEqual(editor.snapshot().positions[0], { x: 800, y: 700 });
	assert.equal(storage.getItem(LEGACY_NETWORK_KEY), network);
	assert.equal(storage.getItem(LEGACY_CORRIDOR_KEY), JSON.stringify(legacy));
	assert.equal(get(editor.legacyNetworkAvailable), true);
	assert.equal(JSON.parse(storage.getItem(SHARED_STORAGE_KEY)).format, 'greenwave-project');
	const restored = createCorridorEditor();
	restored.connectStorage(storage);
	assert.deepEqual(restored.snapshot(), editor.snapshot());
});

test('standalone stages migrate to permanent movement groups without inventing lengths or transitions', () => {
	const old = legacyProject();
	const project = fromLegacyNetwork(old);
	const node = project.junctions.find((node) => node.id === 1);
	assert.deepEqual(
		node.cycle.map((phase) => phase.id),
		[7, 9]
	);
	for (const [movement, assignment] of Object.entries(old.intersections[1].movState)) {
		const timeline = programTimeline(node, project.movements[1][movement]);
		for (const [time, stage] of [
			[5, 7],
			[25, 9]
		])
			assert.equal(
				programStateAt(timeline, time).segment.color,
				assignment.groupIds.includes(stage) ? 'GREEN' : 'RED'
			);
	}
	assert.ok(project.roads.every((road) => road.lengthMeters === null));
	assert.deepEqual(project.corridors[0].nodeIds, []);
	assert.deepEqual(project.migration.source, old);
	assert.deepEqual(parseSharedProject(JSON.stringify(project)), project);
	const storage = memoryStorage([[LEGACY_NETWORK_KEY, JSON.stringify(old)]]);
	const editor = createCorridorEditor();
	editor.connectStorage(storage);
	assert.deepEqual(editor.snapshot(), project);
	assert.equal(storage.getItem(LEGACY_NETWORK_KEY), JSON.stringify(old));
});

test('a branch and its signal program survive route selection, diagram editing and project round trip', () => {
	const { editor, id } = branchedEditor();
	const original = editor.snapshot();
	editor.setRoute([0, 1, id]);
	assert.deepEqual(
		get(editor.junctions).map((node) => node.point.y),
		[0, 200, 325]
	);
	assert.match(get(editor.routeIssue), /assign the forward/);
	const junction = structuredClone(get(editor.junctions)[1]);
	const assignments = { ...editor.snapshot().movements[1], [editor.routeMovements(1).forward]: 0 };
	editor.saveJunction(junction, false, 0, null, assignments);
	assert.equal(get(editor.routeIssue), '');
	junction.offset = 13;
	editor.saveJunction(junction, false, 0, null, assignments);
	assert.equal(editor.snapshot().junctions.find((node) => node.id === 1).offset, 13);
	assert.deepEqual(
		editor.snapshot().junctions.find((node) => node.id === 3),
		original.junctions.find((node) => node.id === 3)
	);
	const exported = parseSharedProject(JSON.stringify(editor.snapshot()));
	assert.equal(exported.junctions.length, 5);
	assert.equal(exported.roads.length, 4);
	assert.deepEqual(corridorProjection(exported).input, editor.inputSnapshot());
});

test('multiple corridors share programs but retain independent routes, settings and group selections', () => {
	const { editor, id } = branchedEditor();
	const first = editor.snapshot().activeCorridorId;
	editor.addCorridor('Branch route');
	editor.setRoute([1, id]);
	editor.desiredSpeed.set(55);
	editor.desiredIntensity.set(0);
	const second = editor.snapshot().activeCorridorId;
	const junction = structuredClone(get(editor.junctions)[0]);
	junction.offset = 17;
	editor.saveJunction(junction);
	const before = editor.snapshot();
	editor.applyOffsets(
		[
			{ id: 1, offset: 4 },
			{ id, offset: 9 }
		],
		get(editor.calculationRevision)
	);
	assert.equal(
		editor.snapshot().junctions.find((node) => node.id === 2).offset,
		before.junctions.find((node) => node.id === 2).offset
	);
	editor.selectCorridor(first);
	assert.equal(get(editor.desiredSpeed), 40);
	assert.equal(get(editor.junctions)[1].offset, 4);
	editor.selectCorridor(second);
	assert.equal(get(editor.desiredSpeed), 55);
	assert.equal(get(editor.desiredIntensity), 0);
	assert.equal(get(editor.junctions)[0].offset, 4);
	editor.undo();
	assert.equal(editor.snapshot().activeCorridorId, first);
});

test('removing a stop keeps network objects; deleting roads or junctions invalidates paths and is undoable', () => {
	const { editor, id } = branchedEditor();
	const original = editor.snapshot();
	editor.setRoute([0, 1, 2]);
	assert.equal(editor.snapshot().junctions.length, 5);
	assert.equal(editor.snapshot().roads.length, 4);
	editor.remove(null, 1);
	assert.match(get(editor.routeIssue), /Connect junctions/);
	editor.undo();
	assert.equal(get(editor.routeIssue), '');
	editor.addCorridor('Second');
	editor.setRoute([1, id]);
	editor.removeJunction(1);
	assert.ok(editor.snapshot().corridors.every((route) => !route.nodeIds.includes(1)));
	assert.ok(editor.snapshot().roads.every((road) => road.from !== 1 && road.to !== 1));
	assert.ok(!editor.snapshot().movements[1]);
	editor.undo();
	assert.equal(editor.snapshot().junctions.length, original.junctions.length);
});

test('manual networks never invent reverse lanes, and missing physical lengths block the route', () => {
	const editor = createCorridorEditor();
	editor.replace(fromLegacyNetwork(legacyProject()));
	editor.setRoute([2, 1, 3]);
	assert.match(get(editor.routeIssue), /Set the length/);
	editor.setRoadLength(1, 90);
	editor.setRoadLength(2, 120);
	assert.deepEqual(
		get(editor.junctions).map((node) => node.point.y),
		[0, 90, 210]
	);
	editor.setRoad(1, { lanes_back: 0 });
	assert.match(get(editor.routeIssue), /does not allow travel/);
	editor.direction.set('bidirectional');
	assert.equal(editor.snapshot().roads.find((road) => road.id === 1).lanes_back, 0);
	editor.reverseRoad(1);
	assert.match(get(editor.routeIssue), /does not allow reverse/);
});

test('off-route edits and canvas moves preserve calculation revision, active route edits invalidate it', () => {
	const { editor, id } = branchedEditor();
	let revision = get(editor.calculationRevision);
	const node = editor.snapshot().junctions.find((node) => node.id === id);
	editor.saveJunction({ ...node, point: { x: 0, y: 0 }, offset: 19 });
	editor.canvas.change((view) => {
		view.nodes[0].x += 300;
	});
	assert.equal(get(editor.calculationRevision), revision);
	editor.setRoadLength(1, 220);
	assert.equal(get(editor.calculationRevision), ++revision);
	editor.addCorridor('Other');
	assert.equal(get(editor.calculationRevision), ++revision);
});

test('project import rejects bad references atomically and keeps saved data unchanged', () => {
	const { editor } = branchedEditor();
	const storage = memoryStorage();
	editor.connectStorage(storage);
	const original = editor.snapshot();
	const saved = storage.getItem(SHARED_STORAGE_KEY);
	const changes = [
		(p) => {
			p.roads[0].from = 99;
		},
		(p) => {
			p.positions[0].x = NaN;
		},
		(p) => {
			p.corridors[0].nodeIds = [0, 0];
		},
		(p) => {
			p.movements[0] = { bad: 0 };
		},
		(p) => {
			p.corridors[0].groupIds = { 0: 99 };
		},
		(p) => {
			p.activeCorridorId = 99;
		}
	];
	for (const mutate of changes) {
		const broken = structuredClone(original);
		mutate(broken);
		assert.throws(() => editor.replace(broken));
		assert.deepEqual(editor.snapshot(), original);
		assert.equal(storage.getItem(SHARED_STORAGE_KEY), saved);
	}
});

test('a corrupt new project is preserved and never silently replaced with an older saved project', () => {
	const storage = memoryStorage([
		[SHARED_STORAGE_KEY, '{broken'],
		[LEGACY_NETWORK_KEY, JSON.stringify(legacyProject())]
	]);
	const editor = createCorridorEditor();
	editor.connectStorage(storage);
	assert.equal(get(editor.persistence).state, 'error');
	assert.equal(storage.getItem(SHARED_STORAGE_KEY), '{broken');
	assert.equal(editor.snapshot().junctions.length, 0);
});

test('removing a group used by another corridor is rejected without modifying the project', () => {
	const editor = createCorridorEditor();
	editor.replaceInput(example);
	editor.addCorridor('Other');
	editor.setRoute([0, 1]);
	editor.groupIds.set({ 0: 30, 1: 30 });
	const before = editor.snapshot();
	const junction = structuredClone(get(editor.junctions)[0]);
	for (const phase of junction.cycle)
		phase.signal_groups = phase.signal_groups.filter((group) => group.id !== 10);
	assert.throws(() => editor.saveJunction(junction, false, 30, null), /does not exist/);
	assert.deepEqual(editor.snapshot(), before);
});

test('conflicts include distinct permanent groups whose green intervals overlap', () => {
	const legacy = legacyProject();
	const project = fromLegacyNetwork(legacy);
	const graph = graphView(project);
	const topology = intersectionTopology(1, graph.nodes, graph.edges);
	const junction = project.junctions.find((node) => node.id === 1);
	const conflicting = topology.movements.filter((move) => move.outEdgeId === 2);
	const assignments = { [conflicting[0].id]: 0, [conflicting[1].id]: 1 };
	assert.ok(programMovementConflicts(junction, topology, assignments).length > 0);
	const copy = structuredClone(junction);
	for (const phase of copy.cycle)
		for (const signal of phase.signal_groups.find((group) => group.id === 1).signals)
			signal.color = 'RED';
	assert.equal(programMovementConflicts(copy, topology, assignments).length, 0);
});

test('input import generates actual road distances, independent of the network canvas', () => {
	const input = structuredClone(DEMO_DATA);
	input.junctions[1].point = { x: 120, y: 160 };
	const project = fromInput(input);
	assert.equal(project.roads[0].lengthMeters, 200);
	project.positions[1] = { x: -1000, y: 5000 };
	assert.equal(corridorProjection(project).input.junctions[1].point.y, 200);
	assert.deepEqual(routeMovements(fromInput(DEMO_DATA), 1), {
		forward: '1_fwd_2_fwd',
		reverse: undefined
	});
});
