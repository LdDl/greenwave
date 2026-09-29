import { graphView, parseSharedProject, programMovementConflicts } from './shared-project.js';
import { intersectionTopology } from './network-project.js';
import { validateMPScenario } from './mp-scenario.js';

const roadKey = (id, direction) => `${id}:${direction}`;
const requireValue = (ok, message) => {
	if (!ok) throw new Error(message);
};
const ordered = (items) => [...items].sort((a, b) => a.id - b.id);

export function mpScenario(project) {
	const leaves = project.junctions.filter(
		(node) =>
			project.roads.filter((road) => road.from === node.id || road.to === node.id).length === 1
	);
	return {
		alpha: 0.5,
		deltaT: 1,
		simTime: 600,
		storageModel: 'point_queue',
		minGreenS: 5,
		maxGreenS: 60,
		clearanceS: 5,
		boundaryNodeIds: leaves.map((node) => node.id),
		coordinatedCorridorIds: project.corridors
			.filter((route) => route.id === project.activeCorridorId && route.nodeIds.length >= 4)
			.map((route) => route.id),
		entryRates: {},
		turningRatios: {},
		saturationFlows: {},
		initialMovementQueues: {},
		...project.mpScenario
	};
}

export function mpTopology(project, scenario = mpScenario(project)) {
	const boundaries = new Set(scenario.boundaryNodeIds);
	const nodes = new Map(project.junctions.map((node) => [node.id, node]));
	const roads = [];
	for (const road of ordered(project.roads)) {
		for (const [dir, from, to, lanes] of [
			['fwd', road.from, road.to, road.lanes_fwd],
			['back', road.to, road.from, road.lanes_back]
		]) {
			if (!lanes || (boundaries.has(from) && boundaries.has(to))) continue;
			roads.push({
				id: roads.length + 1,
				key: roadKey(road.id, dir),
				from,
				to,
				lanes,
				length: road.lengthMeters,
				label: `${nodes.get(from).label} → ${nodes.get(to).label}`,
				entry: boundaries.has(from),
				exit: boundaries.has(to)
			});
		}
	}
	const roadMap = new Map(roads.map((road) => [road.key, road]));
	const graph = graphView(project);
	const intersections = [];
	const movements = [];
	const conflicts = [];
	for (const node of ordered(project.junctions)) {
		if (boundaries.has(node.id)) continue;
		const topology = intersectionTopology(node.id, graph.nodes, graph.edges);
		const assignments = project.movements[node.id] ?? {};
		const assigned = [];
		for (const movement of [...topology.movements].sort((a, b) =>
			a.id.localeCompare(b.id, 'en', { numeric: true })
		)) {
			const group = assignments[movement.id];
			if (!Number.isInteger(group)) continue;
			const incoming = roadMap.get(roadKey(movement.inEdgeId, movement.inDir));
			const outgoing = roadMap.get(roadKey(movement.outEdgeId, movement.outDir));
			if (!incoming || !outgoing) continue;
			const item = {
				id: roads.length + movements.length + 1,
				key: `${node.id}:${movement.id}`,
				nodeId: node.id,
				group,
				incoming,
				outgoing,
				label: `${movement.inLabel} → ${movement.outLabel}`
			};
			assigned.push(item);
			movements.push(item);
		}
		intersections.push({ node, movements: assigned });
		if (programMovementConflicts(node, topology, assignments).length) conflicts.push(node.label);
	}
	for (const movement of movements) {
		const count = movements.filter((item) => item.incoming.id === movement.incoming.id).length;
		movement.ratio = scenario.turningRatios[movement.key] ?? 1 / count;
		movement.saturation = scenario.saturationFlows[movement.key] ?? 1800;
		movement.queue = scenario.initialMovementQueues[movement.key] ?? 0;
	}
	for (const road of roads) road.rate = scenario.entryRates[road.key] ?? 300;
	return { roads, movements, intersections, conflicts };
}

export function buildMPRequest(project) {
	parseSharedProject(project);
	const scenario = mpScenario(project);
	validateMPScenario(scenario);
	const { roads, movements, intersections, conflicts } = mpTopology(project, scenario);
	requireValue(
		intersections.length && movements.length,
		'Assign movements to signal groups in Network. Boundary nodes supply and remove traffic; their programs are not simulated.'
	);
	requireValue(
		!conflicts.length,
		`Resolve overlapping conflicting movements in Network: ${conflicts.join(', ')}.`
	);
	requireValue(
		scenario.alpha <= 1 &&
			scenario.deltaT >= 0.1 &&
			scenario.deltaT <= 60 &&
			scenario.simTime > 0 &&
			scenario.simTime <= 14400,
		'Use alpha 0..1, step 0.1..60 s and duration 1..14400 s.'
	);
	requireValue(
		scenario.maxGreenS === 0 ||
			Math.ceil(scenario.minGreenS / scenario.deltaT) <=
				Math.floor(scenario.maxGreenS / scenario.deltaT),
		'Maximum green must accommodate minimum green at the selected time step.'
	);
	for (const road of roads) {
		requireValue(
			road.length > 0 && road.length <= 100000,
			`Set a positive physical road length in Network: ${road.label}.`
		);
		const outgoing = movements.filter((movement) => movement.incoming.id === road.id);
		requireValue(
			road.exit || outgoing.length,
			`Assign an outgoing movement at ${project.junctions.find((node) => node.id === road.to).label} for ${road.label}, or mark that node as a boundary.`
		);
		requireValue(
			road.entry || movements.some((movement) => movement.outgoing.id === road.id),
			`No assigned movement feeds ${road.label}. Assign one in Network or mark its start as a boundary.`
		);
		if (outgoing.length) {
			const total = outgoing.reduce((sum, movement) => sum + movement.ratio, 0);
			requireValue(
				Math.abs(total - 1) < 1e-8,
				`Turning shares for ${road.label} must add up to 100%.`
			);
			for (const movement of outgoing) movement.ratio /= total;
		}
		requireValue(!road.entry || road.rate <= 100000, `Entry demand is too large: ${road.label}.`);
	}
	for (const item of intersections)
		requireValue(
			item.movements.length,
			`Assign movements at ${item.node.label} or mark it as a boundary.`
		);
	for (const movement of movements)
		requireValue(
			movement.ratio <= 1 &&
				Number.isInteger(movement.saturation) &&
				movement.saturation > 0 &&
				movement.saturation <= 100000 &&
				movement.queue <= 1e9,
			`Invalid turning share, saturation or queue for ${movement.label}.`
		);
	const corridors = [];
	for (const id of scenario.coordinatedCorridorIds) {
		const route = project.corridors.find((item) => item.id === id);
		requireValue(route, 'A coordinated corridor was deleted. Update the MP corridor selection.');
		const directions = [route.nodeIds];
		if (route.direction === 'bidirectional') directions.push([...route.nodeIds].reverse());
		for (const nodes of directions) {
			requireValue(
				nodes.length >= 4,
				`${route.name}: coordination needs at least three directed roads (four nodes).`
			);
			const path = nodes
				.slice(1)
				.map((to, index) => roads.find((road) => road.from === nodes[index] && road.to === to)?.id);
			requireValue(
				path.every((id) => id !== undefined) &&
					path
						.slice(1)
						.every((to, index) =>
							movements.some(
								(movement) => movement.incoming.id === path[index] && movement.outgoing.id === to
							)
						),
				`${route.name}: assign every through movement along the selected direction; interior nodes cannot be boundaries.`
			);
			corridors.push(path);
		}
	}
	requireValue(
		scenario.alpha === 0 || corridors.length,
		'Select a coordinated corridor, or set alpha to 0 for ordinary MaxPressure.'
	);
	return {
		network: {
			links: [
				...roads.map((road) => ({
					id: road.id,
					source_node: road.id * 2,
					target_node: road.id * 2 + 1,
					is_connection: false,
					length_meters: road.length,
					lanes: road.lanes,
					capacity: 1800 * road.lanes
				})),
				...movements.map((movement) => ({
					id: movement.id,
					source_node: movement.incoming.id * 2 + 1,
					target_node: movement.outgoing.id * 2,
					macro_node: movement.nodeId,
					is_connection: true,
					movement_meso_link_income: movement.incoming.id,
					movement_meso_link_outcome: movement.outgoing.id,
					capacity: movement.saturation
				}))
			]
		},
		intersections: intersections.map(({ node, movements }) => ({
			macro_node_id: node.id,
			junction: { ...structuredClone(node), point: { x: 0, y: 0 } },
			group_connectors: Object.fromEntries(
				[...new Set(movements.map((item) => item.group))].map((group) => [
					group,
					movements.filter((item) => item.group === group).map((item) => item.id)
				])
			),
			min_green_s: scenario.minGreenS,
			max_green_s: scenario.maxGreenS,
			clearance_s: scenario.clearanceS
		})),
		coordinated_corridors: corridors,
		turning_ratios: Object.fromEntries(movements.map((item) => [item.id, item.ratio])),
		initial_movement_queues: Object.fromEntries(movements.map((item) => [item.id, item.queue])),
		demand: {
			type: 'constant',
			rates: Object.fromEntries(
				roads.filter((road) => road.entry).map((road) => [road.id, road.rate])
			)
		},
		drain: { type: 'auto' },
		config: {
			delta_t: scenario.deltaT,
			sim_time: scenario.simTime,
			alpha: scenario.alpha,
			storage_model: scenario.storageModel
		}
	};
}

export function mpProgramUpdate(project, proposals) {
	requireValue(
		Array.isArray(proposals) && proposals.length > 0,
		'The MP result contains no programs.'
	);
	const controlled = new Set(mpTopology(project).intersections.map((item) => item.node.id));
	requireValue(
		proposals.length === controlled.size &&
			proposals.every((item) => controlled.has(item.macro_node_id)),
		'The MP result must include every simulated intersection exactly once.'
	);
	const next = structuredClone(project);
	const seen = new Set();
	for (const proposal of proposals) {
		const node = next.junctions.find((node) => node.id === proposal.macro_node_id);
		requireValue(
			node && !seen.has(node.id) && proposal.junction?.id === node.id,
			'The MP result references an invalid junction.'
		);
		seen.add(node.id);
		const shape = (cycle) =>
			cycle.map((phase) => [
				phase.id,
				phase.signal_groups.map((group) => [group.id, group.signals.map((signal) => signal.color)])
			]);
		requireValue(
			JSON.stringify(shape(node.cycle)) === JSON.stringify(shape(proposal.junction.cycle)),
			'The MP result changed the phase or group structure.'
		);
		const duration = (cycle) =>
			cycle.reduce(
				(sum, phase) =>
					sum + phase.signal_groups[0].signals.reduce((sum, signal) => sum + signal.duration, 0),
				0
			);
		requireValue(
			duration(node.cycle) === duration(proposal.junction.cycle) &&
				(node.offset ?? 0) === proposal.junction.offset,
			'The MP result changed the cycle or offset.'
		);
		// Apply durations only; the project owns labels, explicit bounds and metadata.
		for (let pi = 0; pi < node.cycle.length; pi++)
			for (let gi = 0; gi < node.cycle[pi].signal_groups.length; gi++)
				for (let si = 0; si < node.cycle[pi].signal_groups[gi].signals.length; si++) {
					const signal = node.cycle[pi].signal_groups[gi].signals[si];
					const seconds = proposal.junction.cycle[pi].signal_groups[gi].signals[si].duration;
					requireValue(
						Number.isSafeInteger(seconds) &&
							seconds > 0 &&
							seconds >= (signal.min_duration ?? 0) &&
							seconds <= (signal.max_duration ?? Infinity),
						'The MP result violates a signal duration bound.'
					);
					requireValue(
						!['YELLOW', 'REDYELLOW'].includes(signal.color) || seconds === signal.duration,
						'The MP result changed a clearance signal.'
					);
					signal.duration = seconds;
				}
		for (const phase of node.cycle) {
			const totals = phase.signal_groups.map((group) =>
				group.signals.reduce((sum, signal) => sum + signal.duration, 0)
			);
			requireValue(
				totals.every((seconds) => seconds === totals[0]),
				'The MP result has inconsistent group durations.'
			);
		}
	}
	return parseSharedProject(next);
}
