import test from 'node:test';
import assert from 'node:assert/strict';
import {
	diagramTimeWindow,
	departureTimeFrame,
	visiblePeriodicIntervals,
	waveTimeExtent
} from '../src/lib/utils/diagram-time.js';
import { DEMO_DATA } from '../src/lib/utils/demo-input.js';
import { calculateTotalDuration } from '../src/lib/utils/junction-helpers.js';

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

test('departure frames follow travel order and the origin cycle in either direction', () => {
	const nodes = structuredClone(DEMO_DATA.junctions);
	nodes.at(-1).cycle[0].signal_groups[0].signals[0].duration += 10;
	assert.deepEqual(departureTimeFrame(nodes, 1), {
		originIndex: 0,
		duration: 85,
		travelTimes: [0, 720, 1620, 2160]
	});
	assert.deepEqual(departureTimeFrame(nodes, 1, true), {
		originIndex: 3,
		duration: 95,
		travelTimes: [2160, 1440, 540, 0]
	});
});

test('the supplied through wave projects to the same departure interval at every junction', () => {
	const frame = departureTimeFrame(DEMO_DATA.junctions, 1);
	const intervals = [
		[12, 30],
		[732, 750],
		[1632, 1650],
		[2172, 2190]
	];
	const projected = intervals.map((interval, index) =>
		interval.map((time) => time - frame.travelTimes[index])
	);
	assert.deepEqual(projected, [
		[12, 30],
		[12, 30],
		[12, 30],
		[12, 30]
	]);
});

function greenAt(junction, time) {
	const cycle = calculateTotalDuration(junction);
	let elapsed = (((time - junction.offset) % cycle) + cycle) % cycle;
	for (const phase of junction.cycle) {
		for (const signal of phase.signal_groups[0].signals) {
			if (elapsed < signal.duration) return signal.color === 'GREEN';
			elapsed -= signal.duration;
		}
	}
	return false;
}

test('shifted signal rows match actual arrivals, including unequal cycles and very slow travel', () => {
	const nodes = structuredClone(DEMO_DATA.junctions);
	const offsets = [0, 15, 40, 65];
	nodes.forEach((node, index) => {
		node.offset = offsets[index];
	});
	nodes[2].cycle[0].signal_groups[0].signals[0].duration += 10;
	const original = structuredClone(nodes);
	for (const reverse of [false, true]) {
		for (const speed of [1, 0.001, 17.3]) {
			const frame = departureTimeFrame(nodes, speed, reverse);
			for (const [index, junction] of nodes.entries()) {
				const cycle = calculateTotalDuration(junction);
				let start = junction.offset - (frame.travelTimes[index] % cycle);
				const green = [];
				for (const phase of junction.cycle) {
					for (const signal of phase.signal_groups[0].signals) {
						if (signal.color === 'GREEN') {
							green.push(
								...visiblePeriodicIntervals(start, signal.duration, cycle, 0, frame.duration)
							);
						}
						start += signal.duration;
					}
				}
				assert.ok(green.length <= 4);
				for (let departure = 0.137; departure < frame.duration; departure += 0.71) {
					const active = green.some(([start, end]) => departure >= start && departure < end);
					assert.equal(active, greenAt(junction, departure + frame.travelTimes[index]));
				}
			}
		}
	}
	assert.deepEqual(nodes, original);
});

test('departure geometry uses cumulative path lengths and rejects invalid travel times', () => {
	const nodes = structuredClone(DEMO_DATA.junctions.slice(0, 3));
	nodes[0].point = { x: 0, y: 0 };
	nodes[1].point = { x: 3, y: 4 };
	nodes[2].point = { x: 0, y: 0 };
	assert.deepEqual(departureTimeFrame(nodes, 3.6).travelTimes, [0, 5, 10]);
	assert.deepEqual(departureTimeFrame(nodes, 3.6, true).travelTimes, [10, 5, 0]);
	for (const speed of [0, -1, NaN, Infinity, Number.MIN_VALUE]) {
		assert.equal(departureTimeFrame(nodes, speed), null);
	}
	assert.equal(departureTimeFrame([], 1), null);
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
