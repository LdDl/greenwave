<script>
	import { onDestroy } from 'svelte';
	import { corridorEditor, corridorProject } from '$lib/stores/corridor.js';
	import { pressureResult } from '$lib/stores/pressure.js';
	import { MP_API_URL } from '$lib/api/base.js';
	import { buildMPRequest, mpScenario, mpTopology } from '$lib/utils/mp-network.js';
	import { activeMPScenario, mpScenarios } from '$lib/utils/mp-scenario.js';
	import MPTurningMovements from '../../components/MPTurningMovements.svelte';

	let tab = 'traffic';
	let busy = false;
	let error = '';
	let abort;
	let invalidTraffic = new Set();
	$: scenarios = mpScenarios($corridorProject);
	$: selectedScenario = activeMPScenario($corridorProject);
	$: entryRoads = topology.roads.filter((road) => road.entry);
	$: exitRoads = topology.roads.filter((road) => road.exit);
	$: excludedRoads = $corridorProject.roads.filter(
		(road) =>
			scenario.boundaryNodeIds.includes(road.from) && scenario.boundaryNodeIds.includes(road.to)
	);
	$: scenario = mpScenario($corridorProject);
	$: topology = mpTopology($corridorProject, scenario);
	$: signature = JSON.stringify($corridorProject);
	$: stale = $pressureResult && $pressureResult.signature !== signature;
	$: checks = validate($corridorProject);
	$: comparisons = $pressureResult
		? [
				{ label: 'Original program', color: '#64748b', data: $pressureResult.response.baseline },
				{ label: 'Standard MP', color: '#0284c7', data: $pressureResult.response.standard_mp },
				{ label: 'Smoothing-MP', color: '#7c3aed', data: $pressureResult.response.evaluation },
				{
					label: 'Returned fixed program',
					color: '#16a34a',
					data: $pressureResult.response.proposal_evaluation
				}
			]
		: [];
	$: maxQueue = Math.max(
		1,
		...comparisons.flatMap((row) =>
			row.data.samples.map((sample) => sample.queue_veh + sample.boundary_backlog_veh)
		)
	);

	function validate(project) {
		try {
			buildMPRequest(project);
			return '';
		} catch (cause) {
			return cause.message;
		}
	}
	function save(next) {
		try {
			corridorEditor.setMPScenario(next);
			error = '';
		} catch (cause) {
			error = cause.message;
		}
	}
	function editScenario(action, event) {
		try {
			action();
			error = '';
		} catch (cause) {
			error = cause.message;
			if (event) event.currentTarget.value = selectedScenario.name;
		}
	}
	function trafficValidity(id, event) {
		const next = new Set(invalidTraffic);
		if (event.detail.invalid) next.add(id);
		else next.delete(id);
		invalidTraffic = next;
	}
	function changeTraffic(event) {
		const { map, values } = event.detail;
		save({ ...scenario, [map]: { ...scenario[map], ...values } });
	}
	function number(key, event) {
		if (event.currentTarget.value === '' || !event.currentTarget.validity.valid) {
			event.currentTarget.value = String(scenario[key]);
			error = 'Enter a value within the displayed limits. The previous value was restored.';
			return;
		}
		save({ ...scenario, [key]: Number(event.currentTarget.value) });
	}
	function value(map, key, event, scale = 1) {
		if (event.currentTarget.value === '' || !event.currentTarget.validity.valid) {
			event.currentTarget.value = event.currentTarget.defaultValue;
			error = 'Enter a valid non-negative value. The previous value was restored.';
			return;
		}
		save({
			...scenario,
			[map]: { ...scenario[map], [key]: Number(event.currentTarget.value) / scale }
		});
	}
	function toggle(key, id) {
		save({
			...scenario,
			[key]: scenario[key].includes(id)
				? scenario[key].filter((item) => item !== id)
				: [...scenario[key], id]
		});
	}
	async function run() {
		error = '';
		try {
			const project = corridorEditor.snapshot();
			const request = buildMPRequest(project);
			const snapshot = JSON.stringify(project);
			const labels = mpTopology(project).movements.map((item) => ({
				id: item.id,
				label: `${project.junctions.find((node) => node.id === item.nodeId).label}: ${item.label}`
			}));
			abort = new AbortController();
			busy = true;
			const response = await fetch(MP_API_URL, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(request),
				signal: abort.signal
			});
			const result = await response.json();
			if (!response.ok) throw new Error(result.Error ?? `Simulation failed (${response.status}).`);
			if (result.model_version !== 2)
				throw new Error(
					'This server uses an older MP model. Restart the backend with the updated code.'
				);
			pressureResult.set({
				signature: snapshot,
				scenarioName: activeMPScenario(project).name,
				response: result,
				labels,
				applied: false
			});
			tab = 'results';
		} catch (cause) {
			if (cause.name !== 'AbortError') error = cause.message;
		} finally {
			busy = false;
		}
	}
	function apply() {
		try {
			corridorEditor.applyMPPrograms($pressureResult.response.proposal, $pressureResult.signature);
			pressureResult.update((result) => ({ ...result, applied: true }));
			error = '';
		} catch (cause) {
			error = cause.message;
		}
	}
	const fmt = (value) => new Intl.NumberFormat('en', { maximumFractionDigits: 1 }).format(value);
	function points(data) {
		return data.samples
			.map(
				(sample) =>
					`${50 + (sample.time_s / data.duration_s) * 900},${230 - ((sample.queue_veh + sample.boundary_backlog_veh) / maxQueue) * 200}`
			)
			.join(' ');
	}
	onDestroy(() => abort?.abort());
</script>

<svelte:window on:beforeunload={() => corridorEditor.finish()} />

<svelte:head><title>Max pressure | Greenwave</title></svelte:head>

<div class="mp-workspace">
	<header>
		<div>
			<h1>Max pressure</h1>
			<p>Compare signal control across the shared network under the same traffic demand.</p>
		</div>
		<div class="actions">
			{#if busy}<button class="ui-button" on:click={() => abort?.abort()}>Cancel</button>{/if}
			<button
				class="ui-button ui-accent"
				on:click={run}
				disabled={busy || !!checks || invalidTraffic.size > 0}
			>
				{busy ? 'Simulating…' : 'Run comparison'}</button
			>
		</div>
	</header>
	<nav class="tabs" aria-label="Max pressure sections">
		<button class:active={tab === 'traffic'} on:click={() => (tab = 'traffic')}
			>Traffic and control</button
		>
		<button
			class:active={tab === 'results'}
			disabled={!$pressureResult}
			on:click={() => (tab = 'results')}>Results</button
		>
		<a href="/network">Edit network and programs</a>
	</nav>
	<div class="mp-scroll">
		{#if error}<p class="notice danger" role="alert">{error}</p>{/if}
		{#if checks}<p class="notice warning">{checks}</p>{/if}
		{#if tab === 'traffic'}
			<section aria-label="Traffic scenario">
				<h2>Traffic scenario</h2>
				<p class="hint">
					Compare morning, evening or other traffic assumptions on the same network. Each scenario
					stores its own boundaries, demand, turning shares, queues and control settings. Roads,
					signal groups and fixed programs are shared; applying a program changes them for every
					scenario.
				</p>
				<div class="scenario-controls">
					<label
						>Active scenario<select
							value={selectedScenario.id}
							on:change={(event) =>
								editScenario(() =>
									corridorEditor.selectMPScenario(Number(event.currentTarget.value))
								)}
							>{#each scenarios as item (item.id)}<option value={item.id}>{item.name}</option
								>{/each}</select
						></label
					>
					<label
						>Scenario name<input
							type="text"
							maxlength="120"
							value={selectedScenario.name}
							on:change={(event) =>
								editScenario(
									() => corridorEditor.renameMPScenario(event.currentTarget.value),
									event
								)}
						/></label
					>
					<button
						class="ui-button"
						on:click={() => editScenario(() => corridorEditor.duplicateMPScenario())}
						>Duplicate scenario</button
					>
					<button
						class="ui-button delete-scenario"
						disabled={scenarios.length < 2}
						on:click={() => editScenario(() => corridorEditor.removeMPScenario())}
						>Delete scenario</button
					>
				</div>
				<p class="hint">
					Changes are saved with the project. Undo restores edits and deleted scenarios.
				</p>
			</section>
			<section>
				<h2>Control settings</h2>
				<div class="settings">
					<label
						>Duration, s<input
							type="number"
							min="1"
							max="14400"
							step="1"
							value={scenario.simTime}
							on:change={(event) => number('simTime', event)}
						/></label
					>
					<label
						>Time step, s<input
							type="number"
							min="0.1"
							max="60"
							step="0.1"
							value={scenario.deltaT}
							on:change={(event) => number('deltaT', event)}
						/></label
					>
					<label
						>Coordination weight, α<input
							type="number"
							min="0"
							max="1"
							step="0.05"
							value={scenario.alpha}
							on:change={(event) => number('alpha', event)}
						/></label
					>
					<label
						>Minimum green, s<input
							type="number"
							min="0"
							max="3600"
							value={scenario.minGreenS}
							on:change={(event) => number('minGreenS', event)}
						/></label
					>
					<label
						>Maximum green, s<input
							type="number"
							min="0"
							max="3600"
							value={scenario.maxGreenS}
							on:change={(event) => number('maxGreenS', event)}
						/></label
					>
					<label
						>Clearance, s<input
							type="number"
							min="0"
							max="3600"
							value={scenario.clearanceS}
							on:change={(event) => number('clearanceS', event)}
						/></label
					>
					<label
						>Queue model<select
							value={scenario.storageModel}
							aria-describedby="queue-model-hint"
							on:change={(event) => save({ ...scenario, storageModel: event.currentTarget.value })}
							><option value="point_queue">Point queues</option><option value="finite_storage"
								>Finite road storage</option
							></select
						></label
					>
				</div>
				<p class="hint">
					α = 0 uses ordinary MP. Timing bounds and clearance constrain adaptive control; fixed
					programs retain their own yellow and red intervals. Zero disables a timing bound. Minimum
					green and clearance round up to the time step; maximum green rounds down.
				</p>
				<p class="hint" id="queue-model-hint" aria-live="polite" aria-atomic="true">
					<strong
						>Picked queue model: {scenario.storageModel === 'finite_storage'
							? 'Finite road storage'
							: 'Point queues'}.</strong
					>
					{#if scenario.storageModel === 'finite_storage'}
						Available space depends on road length and lane count. A full road holds traffic
						upstream; arrivals that cannot enter the network wait outside and still count toward
						waiting time. Use this to compare control when congestion blocks other roads.
					{:else}
						Queues can grow without a road-space limit and do not block incoming traffic. Use this
						to compare signal control without the effects of roads filling up.
					{/if}
				</p>
			</section>
			<section>
				<h2>Network boundaries</h2>
				<p class="hint">
					Select external entry and exit nodes. A road from a boundary into the network gets an
					Entry demand field; traffic reaching a boundary leaves the model. Boundary signals are not
					simulated. To include an end intersection's traffic lights, add an approach road and use
					its outer endpoint as the boundary.
				</p>
				<div class="choices">
					{#each $corridorProject.junctions as node (node.id)}<label
							><input
								type="checkbox"
								checked={scenario.boundaryNodeIds.includes(node.id)}
								on:change={() => toggle('boundaryNodeIds', node.id)}
							/>{node.label}</label
						>{/each}
				</div>
				<p class="hint" aria-live="polite">
					{scenario.boundaryNodeIds.length} boundary nodes · {topology.intersections.length} controlled
					intersections · {entryRoads.length} entries · {exitRoads.length} exits.
				</p>
				{#if exitRoads.length}<details class="boundary-exits">
						<summary>Where traffic leaves the network</summary>
						<ul>
							{#each exitRoads as road (road.key)}<li>{road.label}</li>{/each}
						</ul>
					</details>{/if}
				{#if excludedRoads.length}<p class="notice warning">
						{excludedRoads.length} roads connect two boundary nodes and are outside this simulation:
						{excludedRoads
							.map(
								(road) =>
									`${$corridorProject.junctions.find((node) => node.id === road.from).label} ↔ ${$corridorProject.junctions.find((node) => node.id === road.to).label}`
							)
							.join(', ')}.
					</p>{/if}
				<h3>Coordinated corridors</h3>
				<p class="hint">
					These routes receive coordination priority in Smoothing-MP; the entire network is still
					simulated. Both directions are included for bidirectional corridors. Each path needs at
					least four nodes with assigned through movements. At α = 0, coordination priority is
					disabled.
				</p>
				<div class="choices">
					{#each $corridorProject.corridors as route (route.id)}<label
							><input
								type="checkbox"
								checked={scenario.coordinatedCorridorIds.includes(route.id)}
								on:change={() => toggle('coordinatedCorridorIds', route.id)}
							/>{route.name} ({route.nodeIds.length} nodes)</label
						>{/each}
				</div>
			</section>
			<section>
				<h2>Entry demand</h2>
				<p class="hint">
					Demand is the number of vehicles wanting to enter per hour: 600 veh/h means an average of
					10 per minute. Only roads from the selected boundaries into the network appear below.
					Internal flows follow the turning shares automatically. These values are independent of
					corridor intensity. New entries default to 300 veh/h; review them for your network. Stored
					rates are kept when you deselect and reselect a boundary.
				</p>
				<div class="entries">
					{#each entryRoads as road (road.key)}<label
							>{road.label}<input
								aria-label={`Demand ${road.label}`}
								type="number"
								min="0"
								max="100000"
								value={road.rate}
								on:change={(event) => value('entryRates', road.key, event)}
							/></label
						>{:else}<p>No entries selected. A closed network needs initial queues below.</p>{/each}
				</div>
			</section>
			<section>
				<h2>Turning movements</h2>
				<p class="hint">
					Only movements assigned to signal groups in Network carry traffic. Shares must total 100%
					for each incoming road, including movements that currently have red. Saturation is a
					movement's service rate while green; account for shared lanes when setting it.
				</p>
				{#each topology.intersections as intersection (intersection.node.id)}
					<MPTurningMovements
						{intersection}
						on:change={changeTraffic}
						on:validity={(event) => trafficValidity(intersection.node.id, event)}
					/>
				{/each}
			</section>
		{:else if $pressureResult}
			<p class="notice">
				Last comparison: {$pressureResult.scenarioName ?? 'Default scenario'}. Results describe this
				scenario at the time of the run.
			</p>
			{#if $pressureResult.applied}<p class="notice">
					The evaluated fixed programs were applied to the shared project. Undo restores the
					previous programs. These results describe the input before application.
				</p>{:else if stale}<p class="notice warning">
					The project has changed. These results describe the previous input; run again before
					applying programs.
				</p>{/if}
			{#each $pressureResult.response.warnings as warning (warning)}<p class="notice">
					{warning}
				</p>{/each}
			<section>
				<h2>Controller comparison</h2>
				<p class="hint">
					Waiting includes queues outside full entries. Adaptive MP changes stages online; the
					returned fixed program is a separate, replayed candidate, not the adaptive signal history.
				</p>
				<div class="table-scroll">
					<table>
						<thead
							><tr
								><th>Controller</th><th>Waiting, veh·s</th><th>Departed, veh</th><th
									>In network, veh</th
								><th>Outside, veh</th><th>Balance error, veh</th></tr
							></thead
						><tbody
							>{#each comparisons as row (row.label)}<tr
									><th scope="row"
										><span class="dot" style={`background:${row.color}`}></span>{row.label}</th
									><td>{fmt(row.data.total_delay_veh_s)}</td><td
										>{fmt(row.data.vehicle_balance.departed_veh)}</td
									><td>{fmt(row.data.vehicle_balance.remaining_veh)}</td><td
										>{fmt(row.data.vehicle_balance.boundary_backlog_veh)}</td
									><td>{row.data.vehicle_balance.conservation_error_veh.toExponential(1)}</td></tr
								>{/each}</tbody
						>
					</table>
				</div>
				<button
					class="ui-button ui-accent"
					disabled={busy ||
						stale ||
						$pressureResult.applied ||
						!$pressureResult.response.proposal_accepted}
					on:click={apply}>Apply evaluated fixed programs</button
				>
				<p class="hint">
					Application requires lower waiting time and at least as many departures as the original
					program. Cycle lengths, offsets and group assignments are preserved.
				</p>
			</section>
			<section>
				<h2>Total queue over time</h2>
				<p class="hint">
					Vehicles in the network plus the boundary backlog. Colors match the comparison table.
				</p>
				<svg
					class="chart"
					viewBox="0 0 980 270"
					role="img"
					aria-label="Total queue over simulation time"
					><path d="M50 25V230H950" fill="none" stroke="#64748b" /><text x="45" y="20"
						>{fmt(maxQueue)} veh</text
					><text x="32" y="235">0</text><text x="50" y="255">0 s</text><text x="880" y="255"
						>{fmt(comparisons[0].data.duration_s)} s</text
					>{#each comparisons as row (row.label)}<polyline
							points={points(row.data)}
							fill="none"
							stroke={row.color}
							stroke-width="2"
						/>{/each}</svg
				>
			</section>
			<section>
				<h2>Movement queues under smoothing</h2>
				<div class="table-scroll">
					<table>
						<thead
							><tr
								><th>Movement</th><th>Average, veh</th><th>Maximum, veh</th><th>Final, veh</th></tr
							></thead
						><tbody
							>{#each $pressureResult.response.evaluation.per_movement as stats (stats.id)}<tr
									><th scope="row"
										>{$pressureResult.labels.find((item) => item.id === stats.id)?.label ??
											stats.id}</th
									><td>{fmt(stats.avg_queue_veh)}</td><td>{fmt(stats.max_queue_veh)}</td><td
										>{fmt(stats.final_queue_veh)}</td
									></tr
								>{/each}</tbody
						>
					</table>
				</div>
			</section>
		{/if}
		<p class="hint model-note">
			This queueing model does not simulate road travel times or moving vehicles. Finite storage and
			practical timing constraints are extensions. Coordination favors movements whose upstream
			signal was green on the previous step; it does not predict arriving platoons.
		</p>
	</div>
</div>

<style>
	.mp-workspace {
		height: 100%;
		min-height: 0;
		display: flex;
		flex-direction: column;
		color: #1e293b;
		background: #f8fafc;
	}
	header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 16px;
		padding: 16px 24px 10px;
	}
	h1 {
		font-size: 22px;
		font-weight: 650;
		margin: 0;
	}
	h2 {
		font-size: 18px;
		font-weight: 600;
		margin: 0 0 10px;
	}
	h3 {
		margin: 20px 0 6px;
		font-weight: 600;
	}
	header p,
	.hint {
		font-size: 13px;
		color: #52637a;
		margin: 6px 0 14px;
		line-height: 1.55;
	}
	.actions,
	.tabs,
	.choices {
		display: flex;
		gap: 10px;
		flex-wrap: wrap;
		align-items: center;
	}
	.tabs {
		padding: 0 24px 10px;
		border-bottom: 1px solid #d7e0ea;
	}
	.tabs button,
	.tabs a {
		padding: 7px 12px;
		border-radius: 6px;
		font-size: 13px;
	}
	.tabs a {
		color: #0369a1;
	}
	.tabs .active {
		background: #d7e0ea;
	}
	button:disabled {
		opacity: 0.45;
		cursor: not-allowed;
	}
	.mp-scroll {
		flex: 1;
		min-height: 0;
		overflow: auto;
		padding: 18px 24px 30px;
	}
	section {
		margin: 0 auto 18px;
		padding: 20px;
		border: 1px solid #d7e0ea;
		border-radius: 10px;
		background: #ffffff;
		max-width: 1500px;
	}
	.scenario-controls {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 12px;
	}
	.scenario-controls label {
		flex: 1 1 200px;
		min-width: 0;
	}
	.delete-scenario {
		color: #b91c1c;
	}
	.boundary-exits {
		margin-bottom: 16px;
		font-size: 13px;
	}
	.boundary-exits ul {
		margin-top: 8px;
		padding-left: 20px;
		list-style: disc;
	}
	.settings {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
		gap: 14px;
	}
	label {
		font-size: 13px;
		display: flex;
		flex-direction: column;
		gap: 7px;
	}
	input[type='text'],
	input[type='number'],
	select {
		width: 100%;
		min-width: 80px;
		padding: 7px 10px;
		background: #f8fafc;
		border: 1px solid #c0cedd;
		border-radius: 5px;
		color: #1e293b;
		font-variant-numeric: tabular-nums;
	}
	input:focus,
	select:focus {
		outline: 2px solid #0d9488;
		outline-offset: 1px;
	}
	.choices label {
		flex-direction: row;
		align-items: center;
		padding: 6px 10px;
		border: 1px solid #c0cedd;
		border-radius: 6px;
	}
	.entries {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 15px;
	}
	.notice {
		max-width: 1500px;
		margin: 0 auto 14px;
		padding: 12px 16px;
		border-radius: 7px;
		background: #e0f2fe;
		font-size: 13px;
		line-height: 1.5;
	}
	.warning {
		background: #fef3c7;
		color: #92400e;
	}
	.danger {
		background: #fee2e2;
		color: #991b1b;
	}
	details {
		margin-top: 12px;
		border-top: 1px solid #d7e0ea;
		padding-top: 10px;
	}
	summary {
		cursor: pointer;
		font-weight: 600;
	}
	.table-scroll {
		overflow: auto;
		margin: 12px 0 18px;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 13px;
	}
	th,
	td {
		text-align: right;
		padding: 10px 12px;
		border-bottom: 1px solid #d7e0ea;
		white-space: nowrap;
		font-variant-numeric: tabular-nums;
	}
	th:first-child {
		text-align: left;
	}
	tbody th {
		font-weight: 400;
	}
	thead {
		color: #52637a;
	}
	.dot {
		display: inline-block;
		width: 9px;
		height: 9px;
		border-radius: 50%;
		margin-right: 8px;
	}
	.chart {
		width: 100%;
		max-height: 360px;
	}
	.chart text {
		fill: #52637a;
		font-size: 12px;
	}
	.model-note {
		max-width: 1500px;
		margin: auto;
	}
	@media (max-width: 650px) {
		header {
			padding: 12px;
			align-items: flex-start;
		}
		header p {
			display: none;
		}
		.tabs {
			padding: 0 12px 8px;
			gap: 3px;
		}
		.mp-scroll {
			padding: 12px;
		}
		section {
			padding: 12px;
		}
		.settings {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
</style>
