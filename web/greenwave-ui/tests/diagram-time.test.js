import test from 'node:test';
import assert from 'node:assert/strict';
import {
	diagramTimeWindow,
	commonCycleDuration,
	foldWaveBand,
	visiblePeriodicIntervals,
	waveTimeExtent
} from '../src/lib/utils/diagram-time.js';

const junctions = [
	{ cycle: [{ signal_groups: [{ signals: [{ duration: 40 }, { duration: 60 }] }] }] }
];

test('arrival times beyond the first cycle determine the diagram extent', () => {
	const end = waveTimeExtent(
		junctions,
		[[{ interval_jun_one: { end: 149 }, interval_jun_two: { end: 233.96 } }]],
		[{ intervals: [{ end: 40 }, { end: 149 }, { end: 233.96 }] }]
	);
	assert.equal(end, 233.96);
	assert.deepEqual(diagramTimeWindow(junctions, end), {
		start: 0,
		end: 233.96,
		span: 233.96,
		maximumStart: 0
	});
});

test('cycle folding is available only for a shared positive program duration', () => {
	assert.equal(commonCycleDuration(junctions), 100);
	assert.equal(commonCycleDuration([]), 0);
	const other = structuredClone(junctions[0]);
	other.cycle[0].signal_groups[0].signals[0].duration += 1;
	assert.equal(commonCycleDuration([...junctions, other]), 0);
});

function area(polygon) {
	return Math.abs(
		polygon.reduce((sum, point, index) => {
			const next = polygon[(index + 1) % polygon.length];
			return sum + point[0] * next[1] - next[0] * point[1];
		}, 0) / 2
	);
}

function polygonContains(polygon, x, y) {
	let inside = false;
	for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
		const a = polygon[i];
		const b = polygon[j];
		if (a[1] > y !== b[1] > y && x < ((b[0] - a[0]) * (y - a[1])) / (b[1] - a[1]) + a[0]) {
			inside = !inside;
		}
	}
	return inside;
}

test('folded bands preserve their width and periodic position across the edges', () => {
	for (const start of [0, 12, 65, 80]) {
		for (const width of [5, 18, 85]) {
			for (const travel of [0, 9, 50, 84, 85, 720, 900, 900009]) {
				const folded = foldWaveBand(
					{ start, end: start + width },
					{ start: start + travel, end: start + travel + width },
					85
				);
				assert.ok(folded.polygons.length > 0 && folded.polygons.length <= 3);
				assert.ok(
					Math.abs(folded.polygons.reduce((sum, polygon) => sum + area(polygon), 0) - width) < 1e-8
				);
				assert.equal(folded.skippedCycles, Math.floor(travel / 85));
				for (const polygon of folded.polygons) {
					for (const [time, progress] of polygon) {
						assert.ok(time >= -1e-8 && time <= 85 + 1e-8);
						assert.ok(progress >= -1e-8 && progress <= 1 + 1e-8);
					}
				}
				for (let progress = 0.035; progress < 1; progress += 0.1) {
					for (let time = 0.131; time < 85; time += 2.5) {
						const phase = (((time - start - (travel % 85) * progress) % 85) + 85) % 85;
						const expected = phase < width;
						const containing = folded.polygons.filter((polygon) =>
							polygonContains(polygon, time, progress)
						);
						assert.ok(containing.length <= 1);
						assert.equal(containing.length === 1, expected);
					}
				}
			}
		}
	}
});

test('nine thousand omitted cycles produce the same geometry as nine seconds', () => {
	const from = { start: 0, end: 40 };
	const short = foldWaveBand(from, { start: 9, end: 49 }, 100);
	const slow = foldWaveBand(from, { start: 900009, end: 900049 }, 100);
	assert.deepEqual(slow.polygons, short.polygons);
	assert.deepEqual(slow.wraps, short.wraps);
	assert.equal(slow.skippedCycles, 9000);
});

test('the supplied 1 km/h wave wraps its second leg and retains its actual travel counts', () => {
	const one = foldWaveBand({ start: 12, end: 30 }, { start: 732, end: 750 }, 85);
	const two = foldWaveBand({ start: 732, end: 750 }, { start: 1632, end: 1650 }, 85);
	const three = foldWaveBand({ start: 1632, end: 1650 }, { start: 2172, end: 2190 }, 85);
	assert.deepEqual([one.skippedCycles, two.skippedCycles, three.skippedCycles], [8, 10, 6]);
	assert.equal(one.polygons.length, 1);
	assert.equal(two.polygons.length, 2);
	assert.equal(two.wraps.length, 1);
	assert.equal(three.polygons.length, 1);
});

test('very slow travel uses a bounded window and allows a direct jump to the last arrival', () => {
	const extent = 1800058;
	const first = diagramTimeWindow(junctions, extent);
	const last = diagramTimeWindow(junctions, extent, extent);
	assert.equal(first.span, 300);
	assert.equal(last.span, 300);
	assert.equal(last.start, extent - 300);
	assert.equal(last.end, extent);
	assert.equal(diagramTimeWindow(junctions, extent, -10).start, 0);
	const intervals = visiblePeriodicIntervals(18, 40, 100, last.start, last.end);
	assert.equal(intervals.length, 3);
	assert.deepEqual(intervals.at(-1), [1800018, 1800058]);
});

test('phase and signal repetitions cover the visible window without gaps or duplicates', () => {
	for (const from of [0, 83, 900083]) {
		const green = visiblePeriodicIntervals(94, 40, 100, from, from + 300);
		const red = visiblePeriodicIntervals(134, 60, 100, from, from + 300);
		for (let time = from + 0.5; time < from + 300; time += 1) {
			const greenCount = green.filter(([start, end]) => time >= start && time < end).length;
			const redCount = red.filter(([start, end]) => time >= start && time < end).length;
			assert.equal(greenCount + redCount, 1);
			assert.equal(greenCount === 1, (((time - 94) % 100) + 100) % 100 < 40);
		}
	}
	assert.deepEqual(visiblePeriodicIntervals(0, 0, 100, 0, 300), []);
	assert.deepEqual(visiblePeriodicIntervals(0, 10, 0, 0, 300), []);
});
