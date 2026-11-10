<script>
	import { createEventDispatcher, onDestroy } from 'svelte';
	import { corridorEditor } from '$lib/stores/corridor.js';
	import IntersectionDiagram from './IntersectionDiagram.svelte';

	export let intersection;
	const dispatch = createEventDispatcher();
	let showDiagram = false;
	let selMovId = null;
	let editingShare = false;
	let fieldError = '';
	$: selected = intersection.movements.find((movement) => movement.movementId === selMovId);
	$: selectedApproach = intersection.approaches.find(
		(approach) => approach.road.id === selected?.incoming.id
	);
	$: invalidApproaches = intersection.approaches.filter((approach) => !approach.valid);
	$: invalidMovements = invalidApproaches.flatMap((approach) =>
		approach.movements.map((movement) => movement.movementId)
	);
	$: movementState = Object.fromEntries(
		intersection.movements.map((movement) => [movement.movementId, { groupIds: [movement.group] }])
	);
	$: movementLabels = Object.fromEntries(
		intersection.movements.map((movement) => [movement.movementId, `${percent(movement.ratio)}%`])
	);
	const percent = (ratio) => Number((ratio * 100).toFixed(6));

	function beginShare() {
		corridorEditor.begin();
		editingShare = true;
	}
	function change(map, movement, event, scale = 1) {
		const input = event.currentTarget;
		if (input.value === '' || !input.validity.valid) {
			fieldError =
				map === 'turningRatios'
					? 'Enter a turn share from 0 to 100%.'
					: 'Enter a value within the displayed limits.';
			if (map === 'turningRatios') dispatch('validity', { invalid: true });
			else input.value = String(map === 'saturationFlows' ? movement.saturation : movement.queue);
			return;
		}
		fieldError = '';
		dispatch('validity', { invalid: false });
		dispatch('change', { map, values: { [movement.key]: Number(input.value) / scale } });
	}
	function finishShare(event, movement) {
		if (event.currentTarget.value === '' || !event.currentTarget.validity.valid) {
			event.currentTarget.value = String(percent(movement.ratio));
			fieldError = 'Invalid turn share. The last valid value was restored.';
		}
		dispatch('validity', { invalid: false });
		corridorEditor.finish();
		editingShare = false;
	}
	function select(movement) {
		showDiagram = true;
		selMovId = movement.movementId;
	}
	function equalShares(approach) {
		dispatch('change', {
			map: 'turningRatios',
			values: Object.fromEntries(
				approach.movements.map((movement) => [movement.key, 1 / approach.movements.length])
			)
		});
	}
	onDestroy(() => {
		if (editingShare) corridorEditor.finish();
		dispatch('validity', { invalid: false });
	});
</script>

<details open class="intersection">
	<summary
		>{intersection.node.label}<span>{intersection.movements.length} movements</span
		>{#if invalidApproaches.length}<span class="invalid"
				>{invalidApproaches.length} incomplete approaches</span
			>{/if}</summary
	>
	<div class="diagram-toggle">
		<button
			class="ui-button"
			aria-expanded={showDiagram}
			on:click={() => (showDiagram = !showDiagram)}
			>{showDiagram ? 'Hide diagram' : 'Edit on diagram'}</button
		>
		<span>Select an arrow or a movement name to edit its traffic.</span>
	</div>
	{#if fieldError}<p class="field-error" role="alert">{fieldError}</p>{/if}
	{#if showDiagram}
		<div class="diagram-editor">
			<div class="diagram">
				<IntersectionDiagram
					stubs={intersection.diagram.stubs}
					movements={intersection.diagram.movements}
					movState={movementState}
					bind:selMovId
					conflictSelection={invalidMovements}
					{movementLabels}
				/>
			</div>
			<div class="movement-inspector">
				{#if selected}
					<h3>{selected.label}</h3>
					<p>Signal group {selected.group} · {intersection.node.label}</p>
					<label
						>Turn share, %<input
							aria-label={`Diagram turn share ${intersection.node.label} ${selected.label}`}
							type="number"
							min="0"
							max="100"
							step="any"
							value={percent(selected.ratio)}
							aria-invalid={!selectedApproach.valid}
							on:focus={beginShare}
							on:input={(event) => change('turningRatios', selected, event, 100)}
							on:blur={(event) => finishShare(event, selected)}
						/></label
					>
					<label
						>Saturation, veh/h<input
							aria-label={`Diagram saturation ${intersection.node.label} ${selected.label}`}
							type="number"
							min="1"
							max="100000"
							value={selected.saturation}
							on:change={(event) => change('saturationFlows', selected, event)}
						/></label
					>
					<label
						>Initial queue, veh<input
							aria-label={`Diagram initial queue ${intersection.node.label} ${selected.label}`}
							type="number"
							min="0"
							max="1000000000"
							step="any"
							value={selected.queue}
							on:change={(event) => change('initialMovementQueues', selected, event)}
						/></label
					>
					<p class:invalid={!selectedApproach.valid} aria-live="polite">
						Incoming road: {selected.incoming.label}. Total {selectedApproach.percent}% / 100%. {selectedApproach.message}
					</p>
				{:else if selMovId !== null}
					<h3>Unassigned movement</h3>
					<p>
						This arrow carries no traffic. Assign it to a signal group in Network before setting its
						share.
					</p>
					<a href="/network">Edit movement assignments</a>
				{:else}
					<h3>Select a movement</h3>
					<p>
						Click an arrow to set its share, saturation and initial queue. The diagram and table
						edit the same scenario.
					</p>
				{/if}
				<p>
					Red arrows belong to an incoming road whose shares do not total 100%. Dashed arrows have
					no assigned signal group. Colors here do not show the current signal state.
				</p>
			</div>
		</div>
	{/if}
	<div class="table-scroll">
		<table class="movements-table" aria-label={`Turning movements at ${intersection.node.label}`}>
			<colgroup
				><col /><col class="group-column" /><col class="share-column" /><col
					class="capacity-column"
				/><col class="queue-column" /></colgroup
			>
			<thead
				><tr
					><th>From → to</th><th>Group</th><th>Turn share, %</th><th>Saturation, veh/h</th><th
						>Initial queue, veh</th
					></tr
				></thead
			>
			{#each intersection.approaches as approach (approach.road.key)}
				<tbody>
					<tr class="approach" class:invalid={!approach.valid}
						><th colspan="5" scope="rowgroup"
							><div class="approach-heading">
								<span>From {approach.road.label}</span><strong aria-live="polite"
									>{approach.percent}% / 100%</strong
								>{#if approach.movements.length > 1}<button
										class="ui-button"
										on:click={() => equalShares(approach)}>Distribute equally</button
									>{/if}
							</div>
							<span class="total-message">{approach.message}</span></th
						></tr
					>
					{#each approach.movements as movement (movement.key)}
						<tr class:selected={showDiagram && selMovId === movement.movementId}>
							<th scope="row"
								><button
									class="movement-name"
									on:click={() => select(movement)}
									aria-label={`Edit ${intersection.node.label} ${movement.label} on diagram`}
									>{movement.label}</button
								></th
							><td>{movement.group}</td>
							<td
								><input
									aria-label={`Turn share ${intersection.node.label} ${movement.label}`}
									aria-invalid={!approach.valid}
									type="number"
									min="0"
									max="100"
									step="any"
									value={percent(movement.ratio)}
									on:focus={beginShare}
									on:input={(event) => change('turningRatios', movement, event, 100)}
									on:blur={(event) => finishShare(event, movement)}
								/></td
							>
							<td
								><input
									aria-label={`Saturation ${intersection.node.label} ${movement.label}`}
									type="number"
									min="1"
									max="100000"
									value={movement.saturation}
									on:change={(event) => change('saturationFlows', movement, event)}
								/></td
							>
							<td
								><input
									aria-label={`Initial queue ${intersection.node.label} ${movement.label}`}
									type="number"
									min="0"
									max="1000000000"
									step="any"
									value={movement.queue}
									on:change={(event) => change('initialMovementQueues', movement, event)}
								/></td
							>
						</tr>
					{/each}
				</tbody>
			{/each}
		</table>
	</div>
</details>

<style>
	.intersection {
		margin-top: 16px;
		border-top: 1px solid #d7e0ea;
		padding-top: 12px;
		min-width: 0;
	}
	summary {
		cursor: pointer;
		font-weight: 600;
	}
	summary span {
		margin-left: 12px;
		font-size: 12px;
		font-weight: 400;
		color: #52637a;
	}
	.diagram-toggle {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
		margin: 12px 0;
		font-size: 13px;
		color: #52637a;
	}
	.diagram-editor {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(240px, 320px);
		border: 1px solid #d7e0ea;
		border-radius: 8px;
		overflow: hidden;
	}
	.diagram {
		display: flex;
		min-height: 400px;
		min-width: 0;
	}
	.movement-inspector {
		padding: 16px;
		min-width: 0;
		background: #f8fafc;
	}
	h3 {
		font-size: 15px;
		font-weight: 600;
		overflow-wrap: anywhere;
	}
	p {
		font-size: 13px;
		line-height: 1.5;
		margin: 8px 0 14px;
		color: #52637a;
	}
	.field-error,
	.invalid {
		color: #b91c1c;
	}
	a,
	.movement-name {
		color: #0369a1;
		text-align: left;
		text-decoration: underline;
		text-underline-offset: 3px;
	}
	label {
		display: flex;
		flex-direction: column;
		gap: 5px;
		font-size: 13px;
		margin: 12px 0;
	}
	input {
		width: 100%;
		min-width: 0;
		padding: 7px 10px;
		background: #f8fafc;
		border: 1px solid #c0cedd;
		border-radius: 5px;
		color: #1e293b;
		font-variant-numeric: tabular-nums;
	}
	input[aria-invalid='true'] {
		border-color: #dc2626;
		background: #fff7f7;
	}
	input:focus {
		outline: 2px solid #0d9488;
		outline-offset: 1px;
	}
	.table-scroll {
		overflow: auto;
		margin: 12px 0 18px;
	}
	table {
		width: 100%;
		table-layout: fixed;
		min-width: 760px;
		border-collapse: collapse;
		font-size: 13px;
	}
	.group-column {
		width: 80px;
	}
	.share-column {
		width: 144px;
	}
	.capacity-column,
	.queue-column {
		width: 172px;
	}
	th,
	td {
		text-align: right;
		padding: 10px 12px;
		border-bottom: 1px solid #d7e0ea;
		font-variant-numeric: tabular-nums;
	}
	th:first-child {
		text-align: left;
	}
	tbody th {
		font-weight: 400;
		overflow-wrap: anywhere;
	}
	table input {
		max-width: 140px;
		text-align: right;
	}
	thead {
		color: #52637a;
	}
	.approach {
		background: #f1f5f9;
	}
	.approach.invalid {
		background: #fff1f2;
	}
	.approach-heading {
		display: flex;
		align-items: center;
		gap: 16px;
	}
	.approach-heading strong {
		margin-left: auto;
		white-space: nowrap;
	}
	.approach-heading button {
		font-size: 12px;
		color: #334155;
	}
	.total-message {
		display: block;
		font-size: 12px;
		margin-top: 4px;
	}
	.selected {
		background: #f0fdfa;
	}
	@media (max-width: 850px) {
		.diagram-editor {
			grid-template-columns: minmax(0, 1fr);
		}
		.diagram {
			min-height: 340px;
		}
	}
</style>
