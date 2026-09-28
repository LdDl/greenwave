import { intersectionTopology } from './network-project.js';
import { emptySharedProject, graphView, routeMovements } from './shared-project.js';

const layout = {
	a: [0, 320],
	b: [160, 320],
	c: [320, 320],
	d: [320, 0],
	e: [480, 320],
	f: [640, 320],
	g: [640, 0],
	h: [960, 320],
	i: [640, 640],
	j: [320, 640],
	k: [1280, 320],
	l: [960, 0],
	m: [960, 640],
	n: [1280, 0],
	o: [1280, 640],
	p: [1440, 320],
	q: [480, 0],
	r: [800, 0],
	s: [1120, 0],
	t: [480, 640],
	u: [800, 640],
	v: [1120, 640]
};

const streets = [
	{ name: 'Pushkin Street', nodes: 'a b c e f h k p', lanes: 2 },
	{ name: 'Nekrasov Street', nodes: 'd q g r l s n', lanes: 1 },
	{ name: 'Tolstoy Street', nodes: 'j t i u m v o', lanes: 1 },
	{ name: 'Krylov Street', nodes: 'd c j', lanes: 2 },
	{ name: 'Gogol Street', nodes: 'g f i', lanes: 2 },
	{ name: 'Chekhov Street', nodes: 'l h m', lanes: 2 },
	{ name: 'Turgenev Street', nodes: 'n k o', lanes: 2 }
];

const groups = [
	{ id: 10, label: 'Eastbound through / right', phase: 0 },
	{ id: 20, label: 'Westbound through / right', phase: 0 },
	{ id: 11, label: 'Eastbound protected left', phase: 1 },
	{ id: 21, label: 'Westbound protected left', phase: 1 },
	{ id: 30, label: 'Northbound through / right', phase: 2 },
	{ id: 40, label: 'Southbound through / right', phase: 2 },
	{ id: 31, label: 'Northbound protected left', phase: 3 },
	{ id: 41, label: 'Southbound protected left', phase: 3 }
];

function approachGroup(dx, dy) {
	return dx > 0 ? 10 : dx < 0 ? 20 : dy < 0 ? 30 : 40;
}

function signalProgram(groupIds) {
	return [40, 10, 40, 10].map((green, index) => ({
		id: index + 1,
		signal_groups: groups
			.filter((group) => groupIds.has(group.id))
			.map((group) => ({
				id: group.id,
				label: group.label,
				signals:
					group.phase === index
						? [
								{ color: 'GREEN', duration: green },
								{ color: 'YELLOW', duration: 3 },
								{ color: 'RED', duration: 2 }
							]
						: [{ color: 'RED', duration: green + 5 }]
			}))
	}));
}

export function rectangularDemoProject() {
	const project = emptySharedProject();
	project.name = 'Writers Quarter';
	const ids = Object.fromEntries(Object.keys(layout).map((letter, index) => [letter, index]));
	project.junctions = Object.entries(layout).map(([letter, [x, y]], index) => {
		project.positions[ids[letter]] = { x, y };
		return { id: ids[letter], label: letter.toUpperCase(), offset: (index * 37) % 120, cycle: [] };
	});
	for (const street of streets) {
		const path = street.nodes.split(' ');
		for (let index = 1; index < path.length; index++) {
			const from = ids[path[index - 1]],
				to = ids[path[index]];
			const a = project.positions[from],
				b = project.positions[to];
			project.roads.push({
				id: project.roads.length + 1,
				name: street.name,
				from,
				to,
				lanes_fwd: street.lanes,
				lanes_back: street.lanes,
				lengthMeters: a.y === b.y ? Math.abs(a.x - b.x) * 1.25 : 300
			});
		}
	}
	const graph = graphView(project);
	for (const node of project.junctions) {
		const center = project.positions[node.id];
		const otherEnd = (roadId) => {
			const road = project.roads.find((road) => road.id === roadId);
			return project.positions[road.from === node.id ? road.to : road.from];
		};
		const topology = intersectionTopology(node.id, graph.nodes, graph.edges);
		const assignments = {};
		for (const movement of topology.movements) {
			const from = otherEnd(movement.inEdgeId),
				to = otherEnd(movement.outEdgeId);
			const dx = center.x - from.x,
				dy = center.y - from.y;
			// Canvas Y grows downward, so a negative cross product is a left turn.
			const left = dx * (to.y - center.y) - dy * (to.x - center.x) < 0;
			assignments[movement.id] = approachGroup(dx, dy) + (left ? 1 : 0);
		}
		project.movements[node.id] = assignments;
		const usedGroups = new Set(Object.values(assignments));
		// Endpoints keep both travel groups for the corridor's external continuation.
		if (!usedGroups.size) {
			usedGroups.add(10);
			usedGroups.add(20);
		}
		node.cycle = signalProgram(usedGroups);
	}
	const routes = [
		['Pushkin Street: main', 'a b c e f h k p', 40, 1500],
		['Nekrasov Street: north bypass', 'a b c d q g r l s n k p', 40, 900],
		['Tolstoy Street: south bypass', 'a b c j t i u m v o k p', 40, 900],
		['Krylov Street: north to south', 'd c j', 36, 1100],
		['Gogol Street: north to south', 'g f i', 36, 1100]
	];
	project.corridors = routes.map(([name, letters, desiredSpeed, desiredIntensity], id) => {
		const route = {
			id,
			name,
			nodeIds: letters.split(' ').map((letter) => ids[letter]),
			desiredSpeed,
			desiredIntensity,
			direction: 'bidirectional',
			origin: { x: 0, y: 0 },
			automatic: false,
			groupIds: {},
			reverseGroupIds: {}
		};
		for (const [index, nodeId] of route.nodeIds.entries()) {
			const movements = routeMovements(project, nodeId, route);
			const from = project.positions[route.nodeIds[Math.max(0, index - 1)]];
			const to = project.positions[route.nodeIds[Math.min(route.nodeIds.length - 1, index + 1)]];
			const forward = approachGroup(to.x - from.x, to.y - from.y);
			const reverse = approachGroup(from.x - to.x, from.y - to.y);
			route.groupIds[nodeId] = project.movements[nodeId][movements.forward] ?? forward;
			route.reverseGroupIds[nodeId] = project.movements[nodeId][movements.reverse] ?? reverse;
			// Endpoints can coordinate an external approach absent from the drawn topology.
			const junction = project.junctions.find((junction) => junction.id === nodeId);
			const selected = new Set(junction.cycle[0].signal_groups.map((group) => group.id));
			selected.add(route.groupIds[nodeId]);
			selected.add(route.reverseGroupIds[nodeId]);
			junction.cycle = signalProgram(selected);
		}
		return route;
	});
	return project;
}
