<script>
	import { onMount } from 'svelte';
	import { createEventDispatcher } from 'svelte';

	import * as d3 from 'd3';
	import { diagramLayout } from '$lib/utils/diagram-layout.js';
	import {
		commonCycleDuration,
		foldWaveBand,
		diagramTimeWindow,
		visiblePeriodicIntervals,
		waveTimeExtent
	} from '$lib/utils/diagram-time.js';
	import { corridorGroupId, corridorTimelineRows } from '$lib/utils/junction-program.js';

	export let junctions = [];
	export let groupIds = {};
	export let reverseGroupIds = {};
	export let direction = 'forward';
	export let wavesAreOutdated = { isOutdated: false, reason: null };
	export let interactive = false;
	export let greenWaves = [];
	export let throughWaves = [];
	export let reverseGreenWaves = [];
	export let reverseThroughWaves = [];
	export let showWaves = false;
	export let showOffsets = false;
	export let overview = false;
	export let zoom = 1;
	export let timeZoom = 1;
	export let timeExtentOverride = 0;
	export let timeMode = 'cycle';

	const dispatch = createEventDispatcher();

	let svg;
	let ruler;
	let container;
	let width = 700;
	let height = 400;
	let isDragging = false;
	let windowStart = 0;
	$: timeExtent = Math.max(
		timeExtentOverride,
		waveTimeExtent(
			junctions,
			showWaves ? [...greenWaves, ...reverseGreenWaves] : [],
			showWaves ? [...throughWaves, ...reverseThroughWaves] : []
		)
	);
	$: commonCycle = commonCycleDuration(junctions);
	$: withinCycle = timeMode === 'cycle' && commonCycle > 0;
	$: timeWindow = withinCycle
		? { start: 0, end: commonCycle, span: commonCycle, maximumStart: 0 }
		: diagramTimeWindow(junctions, timeExtent, windowStart);

	const margin = { top: 62, right: 70, bottom: 46, left: 65 };
	let distanceMinimum = 0;
	let distanceMaximum = 100;
	let coincident = false;
	let chartWidth = width - margin.left - margin.right;
	let chartHeight = height - margin.top - margin.bottom;

	// Helper function to calculate total duration for a junction.
	// Uses the first signal group as the reference (all groups are synchronized).
	function calculateTotalDuration(junction) {
		return junction.cycle.reduce((total, phase) => {
			const signals = phase.signal_groups[0].signals;
			return (
				total +
				signals.reduce((phaseTotal, signal) => {
					return phaseTotal + signal.duration;
				}, 0)
			);
		}, 0);
	}

	function updateChart() {
		// Don't update chart while dragging (keep yScale stable)
		if (isDragging) return;

		if (!svg || !junctions.length) return;

		const g = d3.select(svg);
		g.selectAll('*').remove();

		// Create main group with margins
		const chart = g.append('g').attr('transform', `translate(${margin.left},${margin.top})`);

		// Calculate total durations for all junctions
		const junctionsWithDuration = junctions.map((junction) => ({
			...junction,
			total_duration: calculateTotalDuration(junction),
			group_id: corridorGroupId(junction, groupIds),
			timelineRows: corridorTimelineRows(junction, groupIds, reverseGroupIds, direction).map(
				(row) => ({ ...row, offset: row.offset * 1.8 })
			)
		}));

		const xScale = d3
			.scaleLinear()
			.domain([timeWindow.start, timeWindow.end])
			.range([0, chartWidth]);

		const yScale = d3
			.scaleLinear()
			.domain([distanceMinimum, distanceMaximum])
			.range([chartHeight, 0]);

		const rulerChart = d3.select(ruler);
		rulerChart.selectAll('*').remove();
		rulerChart
			.append('g')
			.attr('transform', `translate(${margin.left},0)`)
			.call(d3.axisBottom(xScale).ticks(Math.max(4, chartWidth / 90)));
		rulerChart
			.append('text')
			.attr('x', 8)
			.attr('y', 16)
			.attr('fill', '#64748b')
			.attr('font-size', 10)
			.text(withinCycle ? 'Cycle (s)' : 'Time (s)');
		chart
			.append('g')
			.attr('class', 'time-grid')
			.selectAll('line')
			.data(xScale.ticks())
			.join('line')
			.attr('x1', xScale)
			.attr('x2', xScale)
			.attr('y1', 0)
			.attr('y2', chartHeight)
			.attr('stroke', '#e9eef3')
			.attr('stroke-dasharray', '3,4');

		// Draw axes
		chart
			.append('g')
			.attr('transform', `translate(0,${chartHeight})`)
			.call(d3.axisBottom(xScale))
			.append('text')
			.attr('x', chartWidth / 2)
			.attr('y', 35)
			.attr('fill', 'black')
			.style('text-anchor', 'middle')
			.text(withinCycle ? 'Time within cycle (seconds)' : 'Elapsed time (seconds)');

		chart
			.append('g')
			.call(d3.axisLeft(yScale).ticks(Math.min(40, Math.max(4, chartHeight / 80))))
			.append('text')
			.attr('transform', 'rotate(-90)')
			.attr('y', -45)
			.attr('x', -chartHeight / 2)
			.attr('fill', 'black')
			.style('text-anchor', 'middle')
			.text('Distance (meters)');

		// Draw waves if enabled
		if (showWaves) {
			const waveChart = chart
				.append('svg')
				.attr('y', -20)
				.attr('width', chartWidth)
				.attr('height', chartHeight + 40)
				.attr('viewBox', `0 -20 ${chartWidth} ${chartHeight + 40}`)
				.attr('overflow', 'hidden');
			drawDirectionWaves(
				waveChart,
				junctionsWithDuration,
				xScale,
				yScale,
				greenWaves,
				throughWaves
			);
			drawDirectionWaves(
				waveChart,
				junctionsWithDuration,
				xScale,
				yScale,
				reverseGreenWaves,
				reverseThroughWaves,
				true
			);
		}

		// Draw phases
		drawPhases(chart, junctionsWithDuration, xScale, yScale);

		// Draw signal timelines
		drawSignalTimelines(chart, junctionsWithDuration, xScale, yScale);

		// Draw junctions
		const junctionGroups = chart
			.selectAll('.junction')
			.data(junctionsWithDuration)
			.enter()
			.append('g')
			.attr('class', 'junction')
			.attr('transform', (d) => `translate(0, ${yScale(d.point.y)})`)
			.style('cursor', interactive ? 'move' : 'default');

		// Make junctions draggable if interactive
		if (interactive) {
			junctionGroups.call(
				d3
					.drag()
					.on('start', function (event, d) {
						console.log('🎯 Drag start:', d.label, 'at distance:', d.point.y);
						d3.select(this).style('opacity', 0.8);
						isDragging = true;
						dispatch('dragStart');
					})
					.on('drag', function (event, d) {
						// Use the simple working approach with event.y
						let newDistance = Math.round(
							yScale.invert(Math.max(0, Math.min(chartHeight, event.y)))
						);
						const index = junctions.findIndex((junction) => junction.id === d.id);
						const ordered = junctions.every(
							(junction, i) => i === 0 || junction.point.y >= junctions[i - 1].point.y
						);
						if (ordered) {
							const lower = index > 0 ? junctions[index - 1].point.y : 0;
							const upper = index < junctions.length - 1 ? junctions[index + 1].point.y : Infinity;
							newDistance = Math.min(upper, Math.max(lower, newDistance));
						}
						const newY = yScale(newDistance);

						console.log('🔄 Dragging:', d.label, 'new distance:', Math.round(newDistance));

						// Update position immediately for visual feedback
						d3.select(this).attr('transform', `translate(0, ${newY})`);

						// Update signal lines for this junction in real-time
						updateSignalLinesForJunction(d.id, newY, junctionsWithDuration, xScale, yScale, chart);

						// Emit an event to notify the parent about the updated position
						dispatch('updateJunction', { id: d.id, newDistance: Math.round(newDistance) });
					})
					.on('end', function (event, d) {
						console.log('✅ Drag end:', d.label, 'final distance:', d.point.y);
						console.log(
							'📊 All junctions now:',
							junctions.map((j) => `${j.label}: ${j.point.y}m`)
						);

						if (wavesAreOutdated.isOutdated) {
							console.log("⚠️ Green waves are now outdated - click 'Extract Waves' to recalculate");
						}

						d3.select(this).style('opacity', 1);
						isDragging = false;
						dispatch('dragEnd');
						// Force complete redraw after drag ends
						setTimeout(() => updateChart(), 10);
					})
			);
		}

		// Draw junction labels with duration
		const junctionLabels = junctionGroups
			.append('text')
			.attr('x', 0)
			.attr('y', (d) => (d.timelineRows.length > 1 ? -47 : -32))
			.attr('text-anchor', 'start')
			.attr('font-size', '12px')
			.attr('font-weight', '600')
			.attr('fill', '#333')
			.text(
				(d) =>
					`${d.label || `J${d.id}`}, ${d.total_duration}s${d.cycle[0].signal_groups.length > 1 ? (d.group_id === null ? ', choose group' : `, G${d.group_id}`) : ''}`
			);

		// Draw offset labels at the right end of X axis (results panel only)
		if (showOffsets) {
			junctionGroups
				.append('text')
				.attr('x', chartWidth + 4)
				.attr('y', 0)
				.attr('dy', '0.35em')
				.attr('text-anchor', 'start')
				.attr('font-size', '10px')
				.attr('font-weight', 'bold')
				.attr('fill', '#4B0082')
				.text((d) => `+${d.offset}s`);
		}

		// Make labels clickable in interactive mode
		if (interactive) {
			junctionLabels
				.style('cursor', 'pointer')
				.on('mouseover', function () {
					d3.select(this).attr('fill', '#4B0082');
				})
				.on('mouseout', function () {
					d3.select(this).attr('fill', '#333');
				})
				.on('click', (event, d) => {
					event.stopPropagation();
					dispatch('editJunction', { junction: d });
				});
		}

		// Draw junction circles
		const junctionCircles = junctionGroups
			.append('circle')
			.attr('cx', 0)
			.attr('cy', 0)
			.attr('r', 6)
			.attr('fill', '#D8BFD8')
			.attr('stroke', '#4B0082')
			.attr('stroke-width', 2);

		// Make circles clickable in interactive mode
		if (interactive) {
			junctionCircles
				.style('cursor', 'pointer')
				.on('mouseover', function () {
					d3.select(this).attr('r', 8).attr('fill', '#C8A2C8');
				})
				.on('mouseout', function () {
					d3.select(this).attr('r', 6).attr('fill', '#D8BFD8');
				})
				.on('click', (event, d) => {
					event.stopPropagation();
					dispatch('editJunction', { junction: d });
				});
		}
	}

	function updateSignalLinesForJunction(
		junctionId,
		newY,
		junctionsWithDuration,
		xScale,
		yScale,
		chart
	) {
		const junction = junctionsWithDuration.find((j) => j.id === junctionId);
		if (!junction) return;
		chart.selectAll(`.signal-row-${junctionId}`).remove();
		drawJunctionTimelines(chart, junction, xScale, newY);
	}

	function drawSignalTimelines(chart, junctionsWithDuration, xScale, yScale) {
		junctionsWithDuration.forEach((junction) =>
			drawJunctionTimelines(chart, junction, xScale, yScale(junction.point.y))
		);
	}

	function drawJunctionTimelines(chart, junction, xScale, y) {
		for (const row of junction.timelineRows) {
			const container = chart
				.append('g')
				.attr('class', `signal-row-${junction.id}`)
				.attr('data-group-id', row.groupId);
			let currentTime = junction.offset ?? 0;
			for (const phase of junction.cycle) {
				const group = phase.signal_groups.find((group) => group.id === row.groupId);
				const signals = group?.signals ?? [];
				for (const signal of signals) {
					if (signal.duration > 0) {
						const parts = visiblePeriodicIntervals(
							currentTime,
							signal.duration,
							junction.total_duration,
							...xScale.domain()
						);
						for (const [from, to] of parts) {
							if (from === to) continue;
							const line = container
								.append('line')
								.attr('class', `signal-line-${junction.id}`)
								.attr('x1', xScale(from))
								.attr('x2', xScale(to))
								.attr('y1', y + row.offset)
								.attr('y2', y + row.offset)
								.attr('stroke', getSignalColor(signal.color))
								.attr('stroke-width', 6);
							line.append('title').text(`${row.label}: ${signal.color}, ${signal.duration} s`);
							if (interactive) {
								line
									.style('cursor', 'pointer')
									.on('mouseover', function () {
										d3.select(this).attr('stroke-width', 10);
									})
									.on('mouseout', function () {
										d3.select(this).attr('stroke-width', 6);
									})
									.on('click', () => handleSignalClick(junction, phase, signal));
							}
						}
					}
					currentTime += signal.duration;
				}
			}
			if (direction === 'bidirectional') {
				container
					.append('text')
					.attr('x', chartWidth)
					.attr('y', y + row.offset - 3)
					.attr('text-anchor', 'end')
					.attr('font-size', 10)
					.attr('fill', '#334155')
					.attr('stroke', 'white')
					.attr('stroke-width', 3)
					.attr('paint-order', 'stroke')
					.style('pointer-events', 'none')
					.text(row.groupId === null ? 'Choose group' : row.label);
			}
		}
	}

	function drawPhases(chart, junctionsWithDuration, xScale, yScale) {
		for (const junction of junctionsWithDuration) {
			let currentTime = junction.offset ?? 0;
			const y = yScale(junction.point.y) - (junction.timelineRows.length > 1 ? 20 : 0);
			junction.cycle.forEach((phase, phaseIdx) => {
				const duration = phase.signal_groups[0].signals.reduce(
					(sum, signal) => sum + signal.duration,
					0
				);
				const intervals = visiblePeriodicIntervals(
					currentTime,
					duration,
					junction.total_duration,
					...xScale.domain()
				);
				for (const [start, end] of intervals) {
					chart
						.append('rect')
						.attr('x', xScale(start))
						.attr('width', xScale(end) - xScale(start))
						.attr('y', y - 10)
						.attr('height', 5)
						.attr('fill', phaseIdx % 2 === 0 ? '#4B0082' : '#18B7CC')
						.attr('fill-opacity', 0.5)
						.attr('stroke', 'black')
						.attr('stroke-width', 1);
					if (xScale(end) - xScale(start) >= 35) {
						chart
							.append('text')
							.attr('x', xScale((start + end) / 2))
							.attr('y', y - 15)
							.attr('text-anchor', 'middle')
							.attr('font-size', '10px')
							.attr('fill', '#4B0082')
							.text(`Phase ${phaseIdx + 1}`);
					}
				}
				currentTime += duration;
			});
		}
	}

	function drawDirectionWaves(chart, nodes, xScale, yScale, segments, through, reverse = false) {
		const route = reverse ? [...nodes].reverse() : nodes;
		const rowIndex = reverse ? -1 : 0;
		const directionLabel = reverse ? 'Reverse' : 'Forward';
		const outdated = wavesAreOutdated.isOutdated;
		const segmentColor = reverse ? '#4FC3F7' : '#57B844';
		const throughColor = reverse ? '#1976D2' : '#541FE4';
		const yPositions = route.map(
			(node) => yScale(node.point.y) + node.timelineRows.at(rowIndex).offset
		);

		for (const wave of through) {
			for (let index = 0; index < wave.intervals.length - 1 && index < route.length - 1; index++) {
				drawWaveBand(
					chart,
					wave.intervals[index],
					wave.intervals[index + 1],
					yPositions[index],
					yPositions[index + 1],
					xScale,
					{
						color: throughColor,
						opacity: outdated ? 0.1 : 0.2,
						through: true,
						directionLabel
					}
				);
			}
		}
		segments.forEach((waves, index) => {
			if (index >= route.length - 1) return;
			const fromY = yPositions[index];
			const toY = yPositions[index + 1];
			for (const wave of waves) {
				drawWaveBand(chart, wave.interval_jun_one, wave.interval_jun_two, fromY, toY, xScale, {
					color: segmentColor,
					opacity: outdated ? 0.15 : 0.3,
					through: false,
					directionLabel
				});
			}
			if (withinCycle && waves.length > 0) {
				const first = waves[0];
				const folded = foldWaveBand(first.interval_jun_one, first.interval_jun_two, commonCycle);
				if (folded.skippedCycles > 0) {
					const label = chart
						.append('text')
						.attr('class', 'cycle-count')
						.attr('data-direction', directionLabel)
						.attr('x', reverse ? chartWidth - 12 : 12)
						.attr('y', (fromY + toY) / 2 - 4)
						.attr('text-anchor', reverse ? 'end' : 'start')
						.attr('font-size', 11)
						.attr('font-weight', 600)
						.attr('fill', reverse ? '#0369a1' : '#166534')
						.attr('stroke', 'white')
						.attr('stroke-width', 4)
						.attr('paint-order', 'stroke')
						.text(`+${folded.skippedCycles} ${folded.skippedCycles === 1 ? 'cycle' : 'cycles'}`);
					label
						.append('title')
						.text(
							`${directionLabel}: ${route[index].label || route[index].id} → ${route[index + 1].label || route[index + 1].id}. Travel ${formatSeconds(first.travel_time)} s = ${folded.skippedCycles} × ${commonCycle} s + ${formatSeconds(folded.remainder)} s. Whole travel cycles are omitted from the band slope.`
						);
				}
			}
		});
	}

	function formatSeconds(value) {
		return Number(value.toFixed(2)).toLocaleString('en-US');
	}

	function drawWaveBand(chart, from, to, fromY, toY, xScale, style) {
		const geometry = withinCycle
			? foldWaveBand(from, to, commonCycle)
			: {
					polygons: [
						[
							[from.start, 0],
							[to.start, 1],
							[to.end, 1],
							[from.end, 0]
						]
					],
					wraps: []
				};
		const outdated = wavesAreOutdated.isOutdated;
		const stroke = outdated ? '#ff6b35' : style.color;
		const tooltip = `${style.directionLabel}${style.through ? ' connected wave' : ''}: ${formatSeconds(from.end - from.start)} s band. Departure ${formatSeconds(from.start)} to ${formatSeconds(from.end)} s. Arrival ${formatSeconds(to.start)} to ${formatSeconds(to.end)} s.`;
		for (const polygon of geometry.polygons) {
			const points = polygon.map(([time, progress]) => [
				xScale(time),
				fromY + progress * (toY - fromY)
			]);
			const band = chart
				.append('polygon')
				.attr('class', style.through ? 'through-band' : 'segment-band')
				.attr('data-direction', style.directionLabel)
				.attr('points', points.map((point) => point.join(',')).join(' '))
				.attr('fill', style.color)
				.attr('fill-opacity', style.opacity)
				.attr('stroke', stroke)
				.attr('stroke-width', outdated ? 1 : 0.5)
				.attr('stroke-opacity', outdated ? 0.6 : 0.8)
				.attr('stroke-dasharray', outdated ? '3,3' : 'none');
			band.append('title').text(tooltip);
		}
		if (!style.through) {
			for (const progress of geometry.wraps) {
				const y = fromY + progress * (toY - fromY);
				for (const x of [1, chartWidth - 7]) {
					const arrow = chart
						.append('path')
						.attr('class', 'cycle-wrap')
						.attr('d', `M${x},${y - 4} L${x + 5},${y} L${x},${y + 4}`)
						.attr('fill', 'none')
						.attr('stroke', stroke)
						.attr('stroke-width', 2);
					arrow.append('title').text('Continues at the opposite edge of this cycle.');
				}
			}
		}
	}

	function getSignalColor(color) {
		const colorMap = {
			RED: '#dc2626',
			YELLOW: '#fbbf24',
			GREEN: '#16a34a',
			GREENPRIORITY: '#15803d'
		};
		return colorMap[color] || '#000000';
	}

	// Update chart when ANY relevant state changes
	$: if (svg && junctions.length > 0) {
		updateChart();
	}

	// Also reactive to all prop changes
	$: (groupIds,
		reverseGroupIds,
		direction,
		greenWaves,
		throughWaves,
		reverseGreenWaves,
		reverseThroughWaves,
		showWaves,
		wavesAreOutdated,
		timeWindow,
		withinCycle,
		updateChart());

	function updateSize() {
		if (!container || isDragging) return;
		const layout = diagramLayout(junctions, container.clientWidth, container.clientHeight, {
			overview,
			zoom,
			timeZoom
		});
		width = layout.width;
		height = layout.height;
		distanceMinimum = layout.minimum;
		distanceMaximum = layout.maximum;
		coincident = layout.coincident;
		chartWidth = width - margin.left - margin.right;
		chartHeight = height - margin.top - margin.bottom;
		updateChart();
	}
	$: (junctions, overview, zoom, timeZoom, updateSize());

	export function getViewport() {
		return {
			timeStart: windowStart,
			vertical: container
				? container.scrollTop / Math.max(1, container.scrollHeight - container.clientHeight)
				: 0,
			horizontal: container
				? container.scrollLeft / Math.max(1, container.scrollWidth - container.clientWidth)
				: 0
		};
	}

	export function scrollToViewport({ vertical, horizontal, timeStart }) {
		if (Number.isFinite(timeStart)) windowStart = timeStart;
		if (!container) return;
		const top = vertical * (container.scrollHeight - container.clientHeight);
		const left = horizontal * (container.scrollWidth - container.clientWidth);
		if (Math.abs(container.scrollTop - top) > 1) container.scrollTop = top;
		if (Math.abs(container.scrollLeft - left) > 1) container.scrollLeft = left;
	}

	function changeTimeWindow(value) {
		windowStart = diagramTimeWindow(junctions, timeExtent, value).start;
		dispatch('viewportScroll', getViewport());
	}

	function handleSignalClick(junction, phase, signal) {
		dispatch('editSignal', { junction, phase, signal });
	}

	onMount(() => {
		updateSize();
		const observer = new ResizeObserver(updateSize);
		observer.observe(container);
		return () => observer.disconnect();
	});
</script>

<div class="diagram-wrapper" data-time-mode={withinCycle ? 'cycle' : 'absolute'}>
	{#if timeWindow.maximumStart > 0}
		<div class="time-navigation" aria-label="Diagram time window">
			<button
				title="Previous time window"
				disabled={timeWindow.start <= 0}
				on:click={() => changeTimeWindow(timeWindow.start - timeWindow.span)}>Earlier</button
			>
			<label
				>From <input
					aria-label="Time window start in seconds"
					type="number"
					min="0"
					max={timeWindow.maximumStart}
					step="any"
					value={timeWindow.start}
					on:change={(event) => changeTimeWindow(event.currentTarget.valueAsNumber)}
				/> s</label
			>
			<span>to {Number(timeWindow.end.toFixed(1))} s</span>
			<button
				title="Next time window"
				disabled={timeWindow.start >= timeWindow.maximumStart}
				on:click={() => changeTimeWindow(timeWindow.start + timeWindow.span)}>Later</button
			>
			<button
				title="Time window containing the last arrival"
				disabled={timeWindow.start >= timeWindow.maximumStart}
				on:click={() => changeTimeWindow(timeWindow.maximumStart)}>Last arrival</button
			>
		</div>
	{/if}
	<!-- Keyboard focus enables native scrolling of the diagram. -->
	<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
	<div
		bind:this={container}
		class="diagram-container"
		on:scroll={() => dispatch('viewportScroll', getViewport())}
		role="region"
		aria-label={interactive
			? 'Scrollable input time-distance diagram'
			: 'Scrollable result time-distance diagram'}
		tabindex="0"
	>
		{#if coincident}<p class="coincident-note">
				Some junctions have the same distance and their signals overlap. Edit their distances to
				separate them.
			</p>{/if}
		<svg bind:this={ruler} {width} height={26} class="diagram-ruler" aria-hidden="true"></svg>
		<svg
			bind:this={svg}
			{width}
			{height}
			style="display:block"
			aria-label={withinCycle
				? 'Time within cycle in seconds and distance in meters'
				: 'Elapsed time in seconds and distance in meters'}
		></svg>
	</div>
</div>

<style>
	.diagram-wrapper {
		height: 100%;
		min-height: 0;
		display: flex;
		flex-direction: column;
	}
	.time-navigation {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 6px;
		padding: 5px 8px;
		font-size: 11px;
		border-bottom: 1px solid #e2e8f0;
		background: #f8fafc;
	}
	.time-navigation label {
		display: flex;
		align-items: center;
		gap: 4px;
	}
	.time-navigation input {
		width: 100px;
		padding: 2px 4px;
		font-size: 11px;
		border: 1px solid #cbd5e1;
		border-radius: 4px;
	}
	.time-navigation button {
		padding: 3px 6px;
		border: 1px solid #cbd5e1;
		border-radius: 4px;
		background: white;
		cursor: pointer;
	}
	.time-navigation button:disabled {
		opacity: 0.45;
		cursor: default;
	}
	.diagram-container {
		width: 100%;
		height: 100%;
		min-height: 0;
		position: relative;
		overflow: auto;
		overscroll-behavior: contain;
		scrollbar-gutter: stable;
	}
	.diagram-ruler {
		position: sticky;
		top: 0;
		z-index: 2;
		display: block;
		background: #fff;
		border-bottom: 1px solid #e2e8f0;
	}
	.coincident-note {
		position: sticky;
		top: 0;
		left: 0;
		z-index: 1;
		font-size: 12px;
		color: #92400e;
		background: #fffbeb;
		padding: 8px 12px;
	}
	@media (max-width: 900px) {
		.diagram-container {
			overscroll-behavior: auto;
		}
	}
</style>
