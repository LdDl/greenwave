import test from 'node:test';
import assert from 'node:assert/strict';
import { get } from 'svelte/store';
import { createCorridorEditor } from '../src/lib/stores/corridor.js';
import { rectangularDemoProject } from '../src/lib/utils/demo-network.js';
import { DEMO_DATA } from '../src/lib/utils/demo-input.js';
import {
	fromInput,
	parseSharedProject,
	SHARED_STORAGE_KEY
} from '../src/lib/utils/shared-project.js';
import { buildMPRequest, mpTopology, mpScenario } from '../src/lib/utils/mp-network.js';
import { activeMPScenario, mpScenarios } from '../src/lib/utils/mp-scenario.js';

test('turn totals identify the incomplete incoming road without mixing approaches or intersections', () => {
	const project = rectangularDemoProject();
	const initial = mpTopology(project);
	const intersection = initial.intersections.find((item) =>
		item.approaches.some((approach) => approach.movements.length > 1)
	);
	const approach = intersection.approaches.find((item) => item.movements.length > 1);
	const movement = approach.movements[0];
	project.mpScenario = {
		...mpScenario(project),
		turningRatios: { [movement.key]: movement.ratio - 0.1 }
	};
	const changed = mpTopology(project);
	const invalid = changed.intersections
		.flatMap((item) => item.approaches)
		.filter((item) => !item.valid);
	assert.equal(invalid.length, 1);
	assert.equal(invalid[0].road.key, approach.road.key);
	assert.equal(invalid[0].percent, 90);
	assert.equal(invalid[0].message, '10% of incoming traffic is unassigned.');
	assert.throws(() => buildMPRequest(project), /100%/);
	project.mpScenario.turningRatios[movement.key] = movement.ratio + 0.1;
	const excessive = mpTopology(project)
		.intersections.flatMap((item) => item.approaches)
		.find((item) => !item.valid);
	assert.equal(excessive.percent, 110);
	assert.equal(excessive.message, '10% too much traffic is assigned.');
	for (const item of approach.movements)
		delete project.movements[intersection.node.id][item.movementId];
	const missing = mpTopology(project)
		.intersections.flatMap((item) => item.approaches)
		.find((item) => item.road.key === approach.road.key);
	assert.equal(missing.percent, 0);
	assert.equal(missing.valid, false);
	assert.match(missing.message, /No assigned outgoing movement/);
});

test('named scenarios isolate demand and boundaries, preserve old input and survive storage and history', () => {
	const values = new Map();
	const storage = {
		getItem: (key) => values.get(key) ?? null,
		setItem: (key, value) => values.set(key, value)
	};
	const editor = createCorridorEditor();
	editor.connectStorage(storage);
	editor.replace(rectangularDemoProject());
	const original = editor.snapshot();
	const originalRequest = buildMPRequest(original);
	editor.duplicateMPScenario();
	assert.equal(mpScenarios(editor.snapshot()).length, 2);
	assert.deepEqual(buildMPRequest(editor.snapshot()), originalRequest);
	editor.renameMPScenario('Evening');
	const eveningId = activeMPScenario(editor.snapshot()).id;
	const entry = mpTopology(editor.snapshot()).roads.find((road) => road.entry);
	editor.setMPScenario({
		...mpScenario(editor.snapshot()),
		entryRates: { [entry.key]: 950 },
		alpha: 0
	});
	assert.equal(buildMPRequest(editor.snapshot()).demand.rates[entry.id], 950);
	editor.selectMPScenario(0);
	assert.deepEqual(buildMPRequest(editor.snapshot()), originalRequest);
	editor.renameMPScenario('Morning');
	assert.throws(() => editor.renameMPScenario('Evening'), /distinct MP scenario names/);
	assert.throws(() => editor.selectMPScenario(9000), /existing MP scenario/);
	editor.selectMPScenario(eveningId);
	const beforeBoundaryChange = editor.snapshot();
	editor.setMPScenario({ ...mpScenario(beforeBoundaryChange), boundaryNodeIds: [] });
	assert.equal(mpTopology(editor.snapshot()).roads.filter((road) => road.entry).length, 0);
	assert.equal(mpTopology(editor.snapshot()).intersections.length, original.junctions.length);
	editor.undo();
	assert.deepEqual(editor.snapshot(), beforeBoundaryChange);
	assert.equal(
		mpTopology(editor.snapshot()).roads.find((road) => road.key === entry.key).rate,
		950
	);
	assert.deepEqual(editor.snapshot().junctions, original.junctions);
	assert.deepEqual(editor.snapshot().roads, original.roads);
	assert.deepEqual(editor.snapshot().corridors, original.corridors);
	const reloaded = createCorridorEditor();
	reloaded.connectStorage(storage);
	assert.deepEqual(reloaded.snapshot(), editor.snapshot());
	assert.deepEqual(parseSharedProject(JSON.stringify(editor.snapshot())), editor.snapshot());
	editor.removeMPScenario();
	assert.equal(activeMPScenario(editor.snapshot()).name, 'Morning');
	assert.deepEqual(buildMPRequest(editor.snapshot()), originalRequest);
	assert.throws(() => editor.removeMPScenario(), /at least one/);
	editor.undo();
	assert.deepEqual(editor.snapshot(), beforeBoundaryChange);
	editor.redo();
	assert.equal(mpScenarios(editor.snapshot()).length, 1);
});

test('legacy scenario conversion retains every setting and malformed collections fail before replacement', () => {
	const project = rectangularDemoProject();
	project.mpScenario = { ...mpScenario(project), alpha: 0.7, simTime: 300 };
	const editor = createCorridorEditor();
	editor.replace(project);
	const before = editor.snapshot();
	editor.renameMPScenario('Imported demand');
	const converted = editor.snapshot();
	assert.equal(converted.mpScenario, undefined);
	assert.deepEqual(activeMPScenario(converted).settings, before.mpScenario);
	assert.deepEqual(buildMPRequest(converted), buildMPRequest(before));
	editor.undo();
	assert.deepEqual(editor.snapshot(), before);
	const malformed = [
		{ ...converted, mpScenarios: [] },
		{ ...converted, mpScenario: before.mpScenario },
		{ ...converted, activeMPScenarioId: 99 },
		{ ...converted, mpScenarios: [...converted.mpScenarios, converted.mpScenarios[0]] },
		{
			...converted,
			mpScenarios: [{ id: 0, name: 'Bad', settings: { turningRatios: { any: -1 } } }]
		},
		{ ...before, activeMPScenarioId: 0 }
	];
	for (const input of malformed) {
		assert.throws(() => editor.replace(input));
		assert.deepEqual(editor.snapshot(), before);
	}
});

test('directed network maps both corridor directions, all assigned turns and explicit boundaries', () => {
	const project = rectangularDemoProject();
	const before = structuredClone(project);
	const scenario = mpScenario(project);
	const topology = mpTopology(project);
	const request = buildMPRequest(project);
	assert.deepEqual(project, before);
	assert.equal(
		request.intersections.length,
		project.junctions.length - scenario.boundaryNodeIds.length
	);
	assert.equal(request.network.links.length, topology.roads.length + topology.movements.length);
	assert.equal(
		new Set(request.network.links.map((link) => link.id)).size,
		request.network.links.length
	);
	assert.equal(request.coordinated_corridors.length, 2);
	assert.notDeepEqual(
		request.coordinated_corridors[0],
		request.coordinated_corridors[1].toReversed()
	);
	for (const road of topology.roads) {
		const turns = topology.movements.filter((item) => item.incoming.id === road.id);
		if (turns.length)
			assert.ok(
				Math.abs(turns.reduce((sum, item) => sum + request.turning_ratios[item.id], 0) - 1) < 1e-12
			);
		assert.equal(request.demand.rates[road.id], road.entry ? 300 : undefined);
	}
	assert.equal(request.initial_queues, undefined);
	assert.equal(request.config.delta_t, 1);
});

test('traffic settings use stable topology keys and preserve independent corridor input', () => {
	const project = rectangularDemoProject();
	const topology = mpTopology(project);
	const entry = topology.roads.find((road) => road.entry);
	const movement = topology.movements[0];
	project.mpScenario = {
		...mpScenario(project),
		entryRates: { [entry.key]: 700 },
		initialMovementQueues: { [movement.key]: 12 }
	};
	const originalIntensity = project.corridors[0].desiredIntensity;
	project.roads.reverse();
	project.junctions.reverse();
	const request = buildMPRequest(project);
	assert.equal(request.demand.rates[entry.id], 700);
	assert.equal(request.initial_movement_queues[movement.id], 12);
	assert.equal(project.corridors[0].desiredIntensity, originalIntensity);
});

test('incomplete turns, missing lengths, unsupported routes and missing assignments are actionable errors', () => {
	const project = rectangularDemoProject();
	const movement = mpTopology(project).movements[0];
	project.mpScenario = { ...mpScenario(project), turningRatios: { [movement.key]: 0.2 } };
	assert.throws(() => buildMPRequest(project), /100%/);
	project.mpScenario.turningRatios = {};
	project.roads[0].lengthMeters = null;
	assert.throws(() => buildMPRequest(project), /physical road length/);
	project.roads[0].lengthMeters = 200;
	project.corridors[0].nodeIds = project.corridors[0].nodeIds.slice(0, 3);
	for (const field of ['groupIds', 'reverseGroupIds'])
		project.corridors[0][field] = Object.fromEntries(
			Object.entries(project.corridors[0][field]).filter(([id]) =>
				project.corridors[0].nodeIds.includes(Number(id))
			)
		);
	assert.throws(() => buildMPRequest(project), /four nodes/);
	const simple = fromInput(DEMO_DATA);
	simple.movements = {};
	assert.throws(() => buildMPRequest(simple), /Assign movements/);
});

test('MP settings persist through reload, export/import and Undo without changing programs', () => {
	const values = new Map();
	const storage = {
		getItem: (key) => values.get(key) ?? null,
		setItem: (key, value) => values.set(key, value)
	};
	const editor = createCorridorEditor();
	editor.connectStorage(storage);
	editor.replace(rectangularDemoProject());
	const before = editor.snapshot();
	editor.setMPScenario({ ...mpScenario(before), alpha: 0.7, simTime: 300 });
	assert.deepEqual(editor.snapshot().junctions, before.junctions);
	const reloaded = createCorridorEditor();
	reloaded.connectStorage(storage);
	assert.equal(reloaded.snapshot().mpScenario.alpha, 0.7);
	assert.deepEqual(parseSharedProject(storage.getItem(SHARED_STORAGE_KEY)), editor.snapshot());
	editor.undo();
	assert.deepEqual(editor.snapshot(), before);
	editor.redo();
	assert.equal(editor.snapshot().mpScenario.alpha, 0.7);
	assert.throws(() => editor.setMPScenario({ entryRates: { '0:fwd': NaN } }), /Invalid MP values/);
});

test('applying MP timings preserves metadata, changes all views and is one undoable action', () => {
	const editor = createCorridorEditor();
	editor.replaceInput(DEMO_DATA);
	const before = editor.snapshot();
	const node = before.junctions[1];
	const updated = structuredClone(node);
	updated.cycle[0].signal_groups[0].signals[0].duration += 1;
	updated.cycle[0].signal_groups[0].signals[1].duration -= 1;
	updated.label = 'Server should not rename this';
	const revision = get(editor.calculationRevision);
	const proposals = mpTopology(before).intersections.map((item) => ({
		macro_node_id: item.node.id,
		junction: item.node.id === node.id ? updated : item.node
	}));
	editor.applyMPPrograms(proposals, JSON.stringify(before));
	assert.equal(editor.snapshot().junctions[1].label, node.label);
	assert.equal(
		editor.inputSnapshot().junctions[1].cycle[0].signal_groups[0].signals[0].duration,
		updated.cycle[0].signal_groups[0].signals[0].duration
	);
	assert.ok(get(editor.calculationRevision) > revision);
	editor.undo();
	assert.deepEqual(editor.snapshot(), before);
	editor.setMPScenario({ alpha: 0 });
	assert.throws(
		() =>
			editor.applyMPPrograms(
				[{ macro_node_id: node.id, junction: updated }],
				JSON.stringify(before)
			),
		/changed after simulation/
	);
});

test('invalid partial MP response cannot mutate the project', () => {
	const editor = createCorridorEditor();
	editor.replaceInput(DEMO_DATA);
	const before = editor.snapshot();
	const proposal = structuredClone(before.junctions[1]);
	proposal.cycle[0].signal_groups[0].signals[0].duration += 1;
	assert.throws(
		() =>
			editor.applyMPPrograms(
				[{ macro_node_id: proposal.id, junction: proposal }],
				JSON.stringify(before)
			),
		/every simulated intersection/
	);
	const proposals = mpTopology(before).intersections.map((item) => ({
		macro_node_id: item.node.id,
		junction: item.node.id === proposal.id ? proposal : item.node
	}));
	assert.throws(() => editor.applyMPPrograms(proposals, JSON.stringify(before)), /cycle or offset/);
	assert.deepEqual(editor.snapshot(), before);
});
