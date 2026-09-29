import { calculateTotalDuration } from './junction-helpers.js';

export function calculationWindow(junctions, speed) {
	if (junctions.length < 2 || !Number.isFinite(speed) || speed <= 0) return null;
	const cycle = calculateTotalDuration(junctions[0]);
	const distance = junctions.slice(1).reduce((sum, junction, index) => {
		const previous = junctions[index].point;
		return sum + Math.hypot(junction.point.x - previous.x, junction.point.y - previous.y);
	}, 0);
	const travel = distance / (speed / 3.6);
	return { travel, cycle, exceedsCycle: travel >= cycle };
}

function waveMetrics(response, prefix, count) {
	const segments = response[`${prefix}green_waves`] ?? [];
	const through = response[`${prefix}through_green_waves`] ?? [];
	const pairBands = segments
		.flat()
		.map((wave) => wave.band_width)
		.filter((band) => band > 0);
	const valid = through.filter((wave) => wave.bandwidth > 0);
	const depth = Math.max(pairBands.length ? 2 : 0, ...valid.map((wave) => wave.depth));
	return {
		fullBand: valid
			.filter((wave) => wave.depth === count)
			.reduce((sum, wave) => sum + wave.bandwidth, 0),
		depth,
		longestBand:
			depth === 2
				? Math.max(0, ...pairBands)
				: Math.max(
						0,
						...valid.filter((wave) => wave.depth === depth).map((wave) => wave.bandwidth)
					),
		// Match the server's objective, including its preference for deeper chains.
		score: valid.reduce((sum, wave) => sum + (wave.depth / count) ** 2 * wave.bandwidth, 0)
	};
}

export function optimizationReport(input, result, before, after, direction) {
	const offsets = input.map((junction, index) => ({
		id: junction.id,
		label: junction.label,
		before: junction.offset ?? 0,
		after: result[index].offset
	}));
	const directions = [
		['Forward', ''],
		...(direction === 'bidirectional' ? [['Reverse', 'reverse_']] : [])
	].map(([label, prefix]) => ({
		label,
		before: waveMetrics(before, prefix, input.length),
		after: waveMetrics(after, prefix, input.length)
	}));
	const beforeScore = directions.reduce((sum, row) => sum + row.before.score, 0);
	const afterScore = directions.reduce((sum, row) => sum + row.after.score, 0);
	return {
		offsets,
		directions,
		beforeScore,
		afterScore,
		changed: offsets.filter((row) => row.before !== row.after).length,
		outcome:
			afterScore > beforeScore + 1e-8
				? 'improved'
				: afterScore < beforeScore - 1e-8
					? 'worse'
					: 'unchanged'
	};
}
