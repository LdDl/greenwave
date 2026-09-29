import test from 'node:test';
import assert from 'node:assert/strict';
import { diagramLayout } from '../src/lib/utils/diagram-layout.js';

const nodes = (distances) => distances.map((y) => ({ point: { x: 0, y } }));

test('long corridors retain readable neighboring signals and proportional travel distances', () => {
	const junctions = nodes([0, 200, 400, 700, 900, 1100, 1300, 1500, 1700, 1900, 2200, 2400]);
	const layout = diagramLayout(junctions, 1100, 500);
	const pixelsPerMeter = (layout.height - 108) / (layout.maximum - layout.minimum);
	assert.ok(layout.height > 500);
	assert.ok(200 * pixelsPerMeter >= 108);
	assert.equal((400 * pixelsPerMeter) / (200 * pixelsPerMeter), 2);
});

test('overview is explicit, time zoom does not change distance scale, and detail zoom increases spacing', () => {
	const junctions = nodes([0, 100, 200, 1000]);
	const detail = diagramLayout(junctions, 900, 400);
	const overview = diagramLayout(junctions, 900, 400, { overview: true });
	const time = diagramLayout(junctions, 900, 400, { timeZoom: 2 });
	const zoomed = diagramLayout(junctions, 900, 400, { zoom: 2 });
	assert.equal(overview.height, 400);
	assert.ok(detail.height > overview.height);
	assert.equal(time.height, detail.height);
	assert.equal(time.width, detail.width * 2);
	assert.ok(zoomed.height > detail.height);
});

test('empty, single, coincident and negative-origin inputs produce finite canvases', () => {
	for (const distances of [[], [0], [100], [0, 0, 0], [-50, 0, 100], [0, 0.001, 2000]]) {
		const layout = diagramLayout(nodes(distances), 300, 200);
		assert.ok(Number.isFinite(layout.width) && layout.width >= 560);
		assert.ok(Number.isFinite(layout.height) && layout.height <= 24000);
		assert.ok(layout.maximum > layout.minimum);
		assert.equal(layout.coincident, new Set(distances).size !== distances.length);
	}
});
