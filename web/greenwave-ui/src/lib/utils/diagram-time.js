import { calculateTotalDuration } from './junction-helpers.js';

// Each row uses departure time at the route origin: departure = local time - travel.
export function departureTimeFrame(junctions, speedKmh, reverse = false) {
	if (!junctions.length || !Number.isFinite(speedKmh) || speedKmh <= 0) return null;
	const originIndex = reverse ? junctions.length - 1 : 0;
	const duration = calculateTotalDuration(junctions[originIndex]);
	if (!Number.isFinite(duration) || duration <= 0) return null;
	const travelTimes = Array(junctions.length).fill(0);
	const step = reverse ? -1 : 1;
	for (let index = originIndex + step; index >= 0 && index < junctions.length; index += step) {
		const from = junctions[index - step].point;
		const to = junctions[index].point;
		const travel = Math.hypot(to.x - from.x, to.y - from.y) / (speedKmh / 3.6);
		travelTimes[index] = travelTimes[index - step] + travel;
		if (!Number.isFinite(travelTimes[index])) return null;
	}
	return { originIndex, duration, travelTimes };
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
