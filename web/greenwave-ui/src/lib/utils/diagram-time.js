import { calculateTotalDuration } from './junction-helpers.js';

export function commonCycleDuration(junctions) {
	if (!junctions.length) return 0;
	const cycle = calculateTotalDuration(junctions[0]);
	if (!Number.isFinite(cycle) || cycle <= 0) return 0;
	return junctions.every((junction) => Math.abs(calculateTotalDuration(junction) - cycle) < 1e-8)
		? cycle
		: 0;
}

export function waveTimeExtent(junctions, segments = [], through = []) {
	let end = Math.max(1, ...junctions.map(calculateTotalDuration));
	for (const segment of segments) {
		for (const wave of segment) {
			end = Math.max(end, wave.interval_jun_one.end, wave.interval_jun_two.end);
		}
	}
	for (const wave of through) {
		for (const interval of wave.intervals) end = Math.max(end, interval.end);
	}
	return end;
}

export function diagramTimeWindow(junctions, extent, requestedStart = 0) {
	const cycle = junctions.length ? calculateTotalDuration(junctions[0]) : 1;
	const span = Math.min(extent, 3 * Math.max(1, cycle));
	const maximumStart = Math.max(0, extent - span);
	const start = Math.max(0, Math.min(maximumStart, Number(requestedStart) || 0));
	return { start, end: start + span, span, maximumStart };
}

// Return only repetitions intersecting the visible window, even far from time zero.
export function visiblePeriodicIntervals(start, duration, cycle, from, to) {
	if (duration <= 0 || cycle <= 0 || to <= from) return [];
	const phase = ((((start % cycle) - (from % cycle)) % cycle) + cycle) % cycle;
	const result = [];
	for (let relative = phase - cycle; relative < to - from; relative += cycle) {
		const left = Math.max(0, relative);
		const right = Math.min(to - from, relative + duration);
		if (right > left) result.push([from + left, from + right]);
	}
	return result;
}

function clipAtTime(polygon, boundary, keepRight) {
	const result = [];
	for (let i = 0; i < polygon.length; i++) {
		const from = polygon[i];
		const to = polygon[(i + 1) % polygon.length];
		const fromInside = keepRight ? from[0] >= boundary : from[0] <= boundary;
		const toInside = keepRight ? to[0] >= boundary : to[0] <= boundary;
		if (fromInside) result.push(from);
		if (fromInside !== toInside) {
			const fraction = (boundary - from[0]) / (to[0] - from[0]);
			result.push([boundary, from[1] + fraction * (to[1] - from[1])]);
		}
	}
	return result;
}

function polygonArea(polygon) {
	let area = 0;
	for (let i = 0; i < polygon.length; i++) {
		const from = polygon[i];
		const to = polygon[(i + 1) % polygon.length];
		area += from[0] * to[1] - to[0] * from[1];
	}
	return Math.abs(area / 2);
}

// Horizontal coordinates are seconds within a common cycle. Vertical coordinates
// are progress from the upstream junction (0) to the downstream junction (1).
// Whole travel cycles are represented by a count, not by repeated geometry.
export function foldWaveBand(from, to, cycle) {
	const travel = to.start - from.start;
	const values = [from.start, from.end, to.start, to.end, cycle, travel];
	if (
		!values.every(Number.isFinite) ||
		cycle <= 0 ||
		travel < 0 ||
		from.end <= from.start ||
		to.end <= to.start
	)
		return { polygons: [], wraps: [], skippedCycles: 0, remainder: 0 };
	let skippedCycles = Math.floor(travel / cycle);
	let remainder = travel % cycle;
	if (cycle - remainder < 1e-8) {
		skippedCycles++;
		remainder = 0;
	}
	const start = ((from.start % cycle) + cycle) % cycle;
	const polygon = [
		[start, 0],
		[start + remainder, 1],
		[start + remainder + to.end - to.start, 1],
		[start + from.end - from.start, 0]
	];
	const maximum = Math.max(...polygon.map(([time]) => time));
	const polygons = [];
	const wraps = [];
	for (let repetition = 0; repetition * cycle < maximum; repetition++) {
		const left = repetition * cycle;
		const right = left + cycle;
		const afterLeft = clipAtTime(polygon, left, true);
		const clipped = clipAtTime(afterLeft, right, false);
		if (polygonArea(clipped) > 1e-8) {
			polygons.push(clipped.map(([time, progress]) => [time - left, progress]));
		}
		if (right < maximum - 1e-8) {
			const crossings = clipped.filter(([time]) => Math.abs(time - right) < 1e-8);
			if (crossings.length > 1) {
				const progress = crossings.map(([, progress]) => progress);
				wraps.push((Math.min(...progress) + Math.max(...progress)) / 2);
			}
		}
	}
	return { polygons, wraps, skippedCycles, remainder };
}
