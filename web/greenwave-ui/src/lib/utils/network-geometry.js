export const MAX_LANES = 12;

export function intersectionTopology(nodeId, nodes, edges) {
	const node = nodes.find((n) => n.id === nodeId);
	if (!node) return { stubs: [], movements: [] };
	const stubs = edges
		.filter((e) => e.from === nodeId || e.to === nodeId)
		.map((e) => {
			const other = nodes.find((n) => n.id === (e.from === nodeId ? e.to : e.from));
			if (!other) return null;
			const atStart = e.from === nodeId;
			const lanesIn = atStart ? e.lanes_back : e.lanes_fwd;
			const lanesOut = atStart ? e.lanes_fwd : e.lanes_back;
			return {
				edgeId: e.id,
				angle: Math.atan2(other.y - node.y, other.x - node.x),
				label: other.label,
				lanesIn,
				lanesOut,
				hasInbound: lanesIn > 0,
				hasOutbound: lanesOut > 0,
				inDir: atStart ? 'back' : 'fwd',
				outDir: atStart ? 'fwd' : 'back'
			};
		})
		.filter(Boolean);
	const movements = [];
	for (const src of stubs.filter((s) => s.hasInbound)) {
		for (const dst of stubs.filter((s) => s.hasOutbound)) {
			if (src.edgeId === dst.edgeId) continue;
			movements.push({
				id: `${src.edgeId}_${src.inDir}_${dst.edgeId}_${dst.outDir}`,
				inEdgeId: src.edgeId,
				inDir: src.inDir,
				outEdgeId: dst.edgeId,
				outDir: dst.outDir,
				inLabel: src.label,
				outLabel: dst.label
			});
		}
	}
	return { stubs, movements };
}

// Separate inbound and outbound ports preserve the circular order for right-hand traffic.
export function movementsConflict(a, b, stubs) {
	if (a.id === b.id || a.inEdgeId === b.inEdgeId) return false;
	if (a.outEdgeId === b.outEdgeId) return true;
	const sorted = [...stubs].sort((x, y) => x.angle - y.angle || x.edgeId - y.edgeId);
	const port = (edgeId, inbound) =>
		sorted.findIndex((s) => s.edgeId === edgeId) * 2 + (inbound ? 0 : 1);
	const start = port(a.inEdgeId, true);
	const end = port(a.outEdgeId, false);
	const between = (x) => (start < end ? x > start && x < end : x > start || x < end);
	return between(port(b.inEdgeId, true)) !== between(port(b.outEdgeId, false));
}
