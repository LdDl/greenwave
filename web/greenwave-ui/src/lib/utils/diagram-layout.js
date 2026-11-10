// Keep a linear distance scale while reserving readable space for nearby junctions.
export function diagramLayout(
	junctions,
	viewportWidth,
	viewportHeight,
	{ overview = false, zoom = 1, timeZoom = 1 } = {}
) {
	const positions = [...new Set(junctions.map((node) => node.point.y))].sort((a, b) => a - b);
	const minimum = Math.min(0, positions[0] ?? 0);
	const maximum = Math.max(minimum + 50, (positions.at(-1) ?? 0) + 50);
	const gaps = positions.slice(1).map((value, index) => value - positions[index]);
	const nearest = gaps.length ? Math.min(...gaps) : maximum - minimum;
	const readableHeight = ((maximum - minimum) / Math.max(1, nearest)) * 108 * zoom + 108;
	return {
		width: Math.max(560, viewportWidth) * timeZoom,
		height: overview
			? Math.max(250, viewportHeight)
			: Math.max(viewportHeight, Math.min(24000, readableHeight)),
		minimum,
		maximum,
		coincident: positions.length !== junctions.length
	};
}
