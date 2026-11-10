import { defaultJunction, emptySharedProject } from '../../src/lib/utils/shared-project.js';
import { intersectionTopology } from '../../src/lib/utils/network-geometry.js';

export function fourWayProject() {
	const project = emptySharedProject();
	const nodes = [
		{ id: 1, x: 360, y: 260, label: 'Center' },
		{ id: 2, x: 360, y: 70, label: 'North' },
		{ id: 3, x: 600, y: 260, label: 'East' },
		{ id: 4, x: 360, y: 450, label: 'South' },
		{ id: 5, x: 120, y: 260, label: 'West' }
	];
	project.junctions = nodes.map((node) => defaultJunction(node.id, node.label));
	project.positions = Object.fromEntries(nodes.map(({ id, x, y }) => [id, { x, y }]));
	project.roads = [2, 3, 4, 5].map((to, index) => ({
		id: index + 1,
		from: 1,
		to,
		lanes_fwd: 1,
		lanes_back: 1,
		lengthMeters: 200
	}));
	project.movements[1] = Object.fromEntries(
		intersectionTopology(1, nodes, project.roads).movements.map((movement) => [movement.id, 0])
	);
	return project;
}
