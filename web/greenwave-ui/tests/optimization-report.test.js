import test from 'node:test';
import assert from 'node:assert/strict';
import { applyOffsetsToJunctions } from '../src/lib/utils/junction-helpers.js';
import { calculationWindow, optimizationReport } from '../src/lib/utils/optimization-report.js';
import { rectangularDemoProject } from '../src/lib/utils/demo-network.js';
import { corridorProjection } from '../src/lib/utils/shared-project.js';

const { input } = corridorProjection(rectangularDemoProject());

test('result offsets reproduce the integer plan evaluated by the API without changing programs', () => {
	const junctions = structuredClone(input.junctions.slice(0, 3));
	const original = structuredClone(junctions);
	const result = applyOffsetsToJunctions(junctions, [0, 38.94889, -2.9]);
	assert.deepEqual(
		result.map((junction) => junction.offset),
		[0, 38, -2]
	);
	assert.deepEqual(junctions, original);
	result[1].cycle[0].signal_groups[0].signals[0].duration = 999;
	assert.deepEqual(junctions, original);
});

test('missing and malformed optimizer offsets cannot silently zero the input', () => {
	for (const offsets of [undefined, null, [], [0], [0, 1, 2], [0, NaN], [0, Infinity], [0, '12']]) {
		assert.throws(
			() => applyOffsetsToJunctions(input.junctions.slice(0, 2), offsets),
			/invalid offsets/
		);
	}
});

test('a partial improvement is reported even when both full-corridor bands remain zero', () => {
	const result = applyOffsetsToJunctions(
		input.junctions,
		input.junctions.map((junction, index) => index * 9)
	);
	const report = optimizationReport(
		input.junctions,
		result,
		{ through_green_waves: [{ depth: 3, bandwidth: 2 }] },
		{
			through_green_waves: [{ depth: 5, bandwidth: 38 }],
			reverse_green_waves: [[{ band_width: 6 }]]
		},
		'bidirectional'
	);
	assert.equal(report.changed, 7);
	assert.equal(report.outcome, 'improved');
	assert.deepEqual(
		report.directions.map((row) => [row.before.fullBand, row.after.fullBand]),
		[
			[0, 0],
			[0, 0]
		]
	);
	assert.deepEqual(
		report.directions.map((row) => [row.before.depth, row.after.depth]),
		[
			[3, 5],
			[0, 2]
		]
	);
	assert.equal(report.directions[0].after.longestBand, 38);
	assert.equal(report.directions[1].after.longestBand, 6);
	assert.equal(report.afterScore, 38 * (5 / 8) ** 2);
	result[1].offset = 500;
	assert.equal(report.offsets[1].after, 9);
});

test('both directions and unchanged or worse runs are distinguished', () => {
	const before = {
		through_green_waves: [{ depth: 8, bandwidth: 10 }],
		reverse_through_green_waves: [
			{ depth: 8, bandwidth: 15 },
			{ depth: 8, bandwidth: 5 }
		]
	};
	const same = optimizationReport(
		input.junctions,
		input.junctions,
		before,
		before,
		'bidirectional'
	);
	assert.equal(same.changed, 0);
	assert.equal(same.outcome, 'unchanged');
	assert.equal(same.afterScore, 30);
	assert.equal(same.directions[1].after.fullBand, 20);
	const worse = optimizationReport(input.junctions, input.junctions, before, {}, 'forward');
	assert.equal(worse.outcome, 'worse');
	assert.equal(worse.directions.length, 1);
});

test('one-cycle limitation is detected from travel time, not from offsets or diagram positions', () => {
	assert.deepEqual(calculationWindow(input.junctions, input.desiredSpeed), {
		travel: 162,
		cycle: 120,
		exceedsCycle: true
	});
	assert.equal(calculationWindow(input.junctions, 54).exceedsCycle, true);
	assert.equal(calculationWindow(input.junctions, 72).exceedsCycle, false);
	const translated = input.junctions.map((junction) => ({
		...junction,
		offset: 0,
		point: { x: junction.point.x + 123, y: junction.point.y + 456 }
	}));
	assert.deepEqual(calculationWindow(translated, 40), calculationWindow(input.junctions, 40));
	assert.equal(calculationWindow([], 40), null);
	assert.equal(calculationWindow(input.junctions, 0), null);
});
