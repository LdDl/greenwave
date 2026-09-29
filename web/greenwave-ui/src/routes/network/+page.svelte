<script>
	import { onMount, tick } from 'svelte';
	import MacroCanvas from '../../components/MacroCanvas.svelte';
	import EditJunctionModal from '../../components/EditJunctionModal.svelte';
	import EditRoadSegmentModal from '../../components/EditRoadSegmentModal.svelte';
	import {
		corridorEditor,
		corridorProject,
		corridorNodes,
		corridorEdges,
		corridorGraph,
		corridorIssue,
		selectedCorridor,
		corridorGroupIds,
		corridorReverseGroupIds
	} from '$lib/stores/corridor.js';
	import {
		junctions,
		desiredSpeed,
		desiredIntensity,
		desiredFlow,
		optimizationDirection
	} from '$lib/stores/core.js';
	import { roadBetween } from '$lib/utils/shared-project.js';
	import { MAX_LANES } from '$lib/utils/network-project.js';
	import { corridorsOpen, workspaceDialogOpen, projectOpened } from '$lib/stores/workspace.js';
	import { corridorGroupId } from '$lib/utils/junction-program.js';

	const button =
		'rounded-md border border-gray-200 bg-white px-3 py-1.5 text-sm hover:bg-gray-50 disabled:opacity-40 focus-visible:outline-2 focus-visible:outline-blue-500';
	let canvas;
	let mounted = false;
	let mode = 'select';
	let selectedNodeId = null;
	let selectedEdgeId = null;
	let editing = null;
	let pendingEdge = null;
	let inspectorTab = 'route';
	let inspectorOpen = true;
	let error = '';
	let notice = '';

	onMount(() => {
		mounted = true;
		tick().then(() => canvas?.fit());
	});
	$: selectedNode = $corridorProject.junctions.find((node) => node.id === selectedNodeId);
	$: selectedEdge = $corridorEdges.find((edge) => edge.id === selectedEdgeId);
	$: activeRoadIds = $selectedCorridor.nodeIds
		.slice(1)
		.map((id, index) => roadBetween($corridorProject, $selectedCorridor.nodeIds[index], id)?.id)
		.filter((id) => id !== undefined);
	$: modalOpen = editing !== null || pendingEdge !== null || $workspaceDialogOpen;
	$: pendingFrom = $corridorNodes.find((node) => node.id === pendingEdge?.fromId);
	$: pendingTo = $corridorNodes.find((node) => node.id === pendingEdge?.toId);

	$: if (mounted && $projectOpened) resetView();
	async function resetView() {
		deselect();
		editing = null;
		pendingEdge = null;
		error = '';
		notice = '';
		await tick();
		canvas?.fit();
	}

	function attempt(action) {
		try {
			action();
			error = '';
		} catch (cause) {
			error = cause.message;
		}
	}
	function deselect() {
		selectedNodeId = null;
		selectedEdgeId = null;
	}
	function openJunction(id) {
		const node = $corridorProject.junctions.find((node) => node.id === id);
		if (node) {
			editing = structuredClone(
				$junctions.find((node) => node.id === id) ?? { ...node, point: { x: 0, y: 0 } }
			);
			error = '';
		}
	}
	function saveJunction(event) {
		attempt(() => {
			const detail = event.detail;
			corridorEditor.saveJunction(
				detail.junction,
				false,
				detail.groupId,
				detail.reverseGroupId,
				detail.assignments
			);
			editing = null;
		});
	}
	function removeSelected() {
		if (selectedNodeId === null && selectedEdgeId === null) return;
		corridorEditor.remove(selectedNodeId, selectedEdgeId);
		deselect();
		notice = 'Selection removed from the network. Undo restores it and its corridor references.';
	}
	function changeHistory(redo = false) {
		if (redo) corridorEditor.redo();
		else corridorEditor.undo();
		error = '';
		notice = '';
		deselect();
		canvas?.cancelDrawing();
	}
	function handleKeydown(event) {
		if (
			modalOpen ||
			event.defaultPrevented ||
			event.target.closest('input, textarea, select, [contenteditable="true"]')
		)
			return;
		const key = event.key.toLowerCase();
		if ((event.ctrlKey || event.metaKey) && (key === 'z' || key === 'y')) {
			event.preventDefault();
			changeHistory(key === 'y' || event.shiftKey);
			return;
		}
		if (event.ctrlKey || event.metaKey || event.altKey) return;
		if (key === 'delete' || key === 'backspace') {
			event.preventDefault();
			removeSelected();
		}
		if (key === 'escape') {
			deselect();
			canvas?.cancelDrawing();
			mode = 'select';
		}
		if (key === 'v') mode = 'select';
		if (key === 'n') mode = 'node';
		if (key === 'r') mode = 'edge';
		if (key === 'f') canvas?.fit();
	}
	function setRoute(ids) {
		attempt(() => corridorEditor.setRoute(ids));
	}
	function moveStop(index, delta) {
		const ids = [...$selectedCorridor.nodeIds];
		[ids[index], ids[index + delta]] = [ids[index + delta], ids[index]];
		setRoute(ids);
	}
	function addRoad(event) {
		attempt(() => {
			const { lanesFwd, lanesBack, lengthMeters } = event.detail;
			const id = corridorEditor.addRoad(
				pendingEdge.fromId,
				pendingEdge.toId,
				lanesFwd,
				lanesBack,
				lengthMeters
			);
			if (id === null) throw new Error('These junctions already have a road.');
			pendingEdge = null;
			mode = 'select';
			selectedEdgeId = id;
			selectedNodeId = null;
		});
	}
	function updateRoad(key, event) {
		const value = event.target.value.trim();
		attempt(() => {
			if (!value) throw new Error('Enter a value.');
			corridorEditor.setRoad(selectedEdgeId, { [key]: Number(value) });
		});
		event.target.value = selectedEdge[key] ?? '';
	}
</script>

<svelte:head><title>Network and corridors | Greenwave</title></svelte:head>
<svelte:window on:keydown={handleKeydown} />

<div class="network-workspace">
	<div class="network-tools">
		<div class="network-heading">
			<p class="ui-eyebrow">Network editor</p>
			<h1>{$corridorProject.name}</h1>
		</div>
		<button class={button} aria-pressed={mode === 'select'} on:click={() => (mode = 'select')}
			>Select</button
		>
		<button class={button} aria-pressed={mode === 'node'} on:click={() => (mode = 'node')}
			>+ Junction</button
		>
		<button class={button} aria-pressed={mode === 'edge'} on:click={() => (mode = 'edge')}
			>+ Road</button
		>
		<button
			class={button}
			disabled={selectedNodeId === null && selectedEdgeId === null}
			on:click={removeSelected}>Delete</button
		>
		<button class={button} on:click={() => canvas?.fit()}>Fit</button>
		<span class="ml-auto text-xs text-gray-500"
			>{$corridorNodes.length} junctions · {$corridorEdges.length} roads</span
		>
		<button
			class="ui-button"
			aria-pressed={inspectorOpen}
			on:click={() => (inspectorOpen = !inspectorOpen)}
			>{inspectorOpen ? 'Hide properties' : 'Show properties'}</button
		>
	</div>
	{#if error}<p role="alert" class="bg-red-50 px-4 py-2 text-sm text-red-700">{error}</p>{/if}
	{#if notice}<p role="status" class="bg-blue-50 px-4 py-2 text-sm text-blue-800">
			{notice}
		</p>{/if}
	{#if $corridorProject.migration}<p class="bg-amber-50 px-4 py-2 text-xs text-amber-900">
			{$corridorProject.migration.notes.join(' ')}
		</p>{/if}
	<div class="network-body" class:inspector-hidden={!inspectorOpen}>
		<div class="network-canvas">
			{#if mounted}
				<MacroCanvas
					bind:this={canvas}
					networkNodes={corridorNodes}
					networkEdges={corridorEdges}
					networkEditor={corridorEditor.canvas}
					interactionHint="Draw roads, then select junctions in corridor order. Double-click a junction to edit movements and programs."
					activeNodeIds={$selectedCorridor.nodeIds}
					{activeRoadIds}
					{mode}
					disabled={modalOpen}
					{selectedNodeId}
					{selectedEdgeId}
					highlightFrom={selectedEdge?.from ?? null}
					highlightTo={selectedEdge?.to ?? null}
					on:selectNode={(event) => {
						selectedNodeId = event.detail.node.id;
						inspectorTab = 'selection';
						inspectorOpen = true;
						selectedEdgeId = null;
					}}
					on:selectEdge={(event) => {
						selectedEdgeId = event.detail.edge.id;
						inspectorTab = 'selection';
						inspectorOpen = true;
						selectedNodeId = null;
					}}
					on:nodeAdded={(event) => {
						mode = 'select';
						selectedNodeId = event.detail.id;
						inspectorTab = 'selection';
						inspectorOpen = true;
						selectedEdgeId = null;
					}}
					on:edgePending={(event) => {
						pendingEdge = event.detail;
						error = '';
					}}
					on:notice={(event) => (notice = event.detail)}
					on:deselect={deselect}
					on:openIntersection={(event) => openJunction(event.detail.nodeId)}
				/>
			{/if}
		</div>
		{#if inspectorOpen}
			<aside class="network-inspector" aria-label="Network and corridor properties">
				<div class="inspector-tabs" role="group" aria-label="Network properties">
					{#each ['route', 'selection', 'settings'] as tab (tab)}<button
							class:active={inspectorTab === tab}
							aria-pressed={inspectorTab === tab}
							on:click={() => (inspectorTab = tab)}
							>{tab === 'route'
								? 'Current route'
								: tab === 'selection'
									? 'Selection'
									: 'Settings'}</button
						>{/each}
				</div>
				<div class="network-inspector-content">
					{#if inspectorTab === 'settings'}
						<section>
							<label for="project-name" class="block text-xs font-medium">Project name</label>
							<input
								id="project-name"
								maxlength="120"
								class="mt-1 w-full rounded border p-2 text-sm"
								value={$corridorProject.name}
								on:change={(event) => attempt(() => corridorEditor.rename(event.target.value))}
							/>
						</section>
					{/if}
					{#if inspectorTab === 'selection'}
						{#if selectedNode}
							<section class="rounded border border-blue-200 bg-blue-50 p-3">
								<h2 class="font-semibold">{selectedNode.label}</h2>
								<p class="mt-1 text-xs text-gray-600">
									Offset: {selectedNode.offset ?? 0} s · {$selectedCorridor.nodeIds.includes(
										selectedNode.id
									)
										? 'In this corridor'
										: 'Outside this corridor'}
								</p>
								<div class="mt-3 flex flex-wrap gap-2">
									<button class={button} on:click={() => openJunction(selectedNode.id)}
										>Edit intersection</button
									>
									<button
										class={button}
										disabled={$selectedCorridor.nodeIds.includes(selectedNode.id)}
										on:click={() => setRoute([...$selectedCorridor.nodeIds, selectedNode.id])}
										>Append to corridor</button
									>
								</div>
							</section>
						{:else if selectedEdge}
							<section class="space-y-2 rounded border p-3">
								<h2 class="font-semibold">Road segment</h2>
								<p class="text-xs">
									{$corridorNodes.find((node) => node.id === selectedEdge.from)?.label} → {$corridorNodes.find(
										(node) => node.id === selectedEdge.to
									)?.label}
								</p>
								<label for="road-length" class="block text-xs">Length (m)</label>
								<input
									id="road-length"
									type="number"
									min="0"
									step="any"
									class="w-full rounded border p-2 text-sm"
									placeholder="Set physical length"
									value={selectedEdge.lengthMeters ?? ''}
									on:change={(event) => updateRoad('lengthMeters', event)}
								/>
								<label for="lanes-fwd" class="block text-xs">Forward lanes</label>
								<input
									id="lanes-fwd"
									type="number"
									min="1"
									max={MAX_LANES}
									class="w-full rounded border p-2 text-sm"
									value={selectedEdge.lanes_fwd}
									on:change={(event) => updateRoad('lanes_fwd', event)}
								/>
								<label for="lanes-back" class="block text-xs">Reverse lanes (0 for one-way)</label>
								<input
									id="lanes-back"
									type="number"
									min="0"
									max={MAX_LANES}
									class="w-full rounded border p-2 text-sm"
									value={selectedEdge.lanes_back}
									on:change={(event) => updateRoad('lanes_back', event)}
								/>
								{#if !selectedEdge.lanes_back}<button
										class={button}
										on:click={() => attempt(() => corridorEditor.reverseRoad(selectedEdge.id))}
										>Reverse direction</button
									>{/if}
							</section>
						{/if}
					{/if}
					{#if inspectorTab === 'route'}
						<section class="space-y-2">
							<div class="route-heading">
								<h2>{$selectedCorridor.name}</h2>
								<button class="ui-button" on:click={() => corridorsOpen.set(true)}
									>Change corridor</button
								>
							</div>
							<p class="text-xs text-gray-500">
								A route through the shared network. Edits here also update its coordination diagram.
							</p>
							<label for="corridor-name" class="block text-xs">Corridor name</label>
							<input
								id="corridor-name"
								maxlength="120"
								class="w-full rounded border p-2 text-sm"
								value={$selectedCorridor.name}
								on:change={(event) =>
									attempt(() =>
										corridorEditor.renameCorridor($selectedCorridor.id, event.target.value)
									)}
							/>
							<p class="text-xs text-gray-500">
								Select junctions on the network and append them in travel order. Removing a stop
								keeps its junction and roads in the network.
							</p>
							<ol class="space-y-1">
								{#each $selectedCorridor.nodeIds as id, index (id)}
									<li class="flex items-center gap-1 rounded border p-2 text-xs">
										<button
											class="mr-auto text-left text-blue-700"
											on:click={() => {
												selectedNodeId = id;
												selectedEdgeId = null;
											}}>{index + 1}. {$corridorNodes.find((node) => node.id === id)?.label}</button
										>
										<button
											aria-label={`Move stop ${index + 1} earlier`}
											class="rounded border px-2 py-1 disabled:opacity-30"
											disabled={index === 0}
											on:click={() => moveStop(index, -1)}>↑</button
										>
										<button
											aria-label={`Move stop ${index + 1} later`}
											class="rounded border px-2 py-1 disabled:opacity-30"
											disabled={index === $selectedCorridor.nodeIds.length - 1}
											on:click={() => moveStop(index, 1)}>↓</button
										>
										<button
											aria-label={`Remove stop ${index + 1} from corridor`}
											class="rounded border px-2 py-1 text-red-700"
											on:click={() =>
												setRoute($selectedCorridor.nodeIds.filter((nodeId) => nodeId !== id))}
											>×</button
										>
									</li>
								{/each}
							</ol>
							{#if !$selectedCorridor.nodeIds.length}<p class="text-sm text-gray-500">
									No junctions selected for this corridor.
								</p>{/if}
							{#if $corridorIssue}<p
									role="status"
									class="rounded bg-amber-50 p-2 text-xs text-amber-900"
								>
									{$corridorIssue}
								</p>{/if}
							<div class="flex flex-wrap gap-2">
								<button
									class={button}
									disabled={$corridorProject.corridors.length === 1}
									on:click={() => corridorEditor.removeCorridor($selectedCorridor.id)}
									>Delete corridor</button
								>
							</div>
						</section>
					{/if}
					{#if inspectorTab === 'settings'}
						<section class="space-y-2">
							<h2 class="text-sm font-semibold">Calculation settings</h2>
							<label for="corridor-direction" class="block text-xs">Optimization direction</label>
							<select
								id="corridor-direction"
								bind:value={$optimizationDirection}
								class="w-full rounded border p-2 text-sm"
								><option value="forward">Forward</option><option value="bidirectional"
									>Bidirectional</option
								></select
							>
							<label for="corridor-speed" class="block text-xs">Desired speed (km/h)</label>
							<input
								id="corridor-speed"
								type="number"
								min="10"
								max="100"
								bind:value={$desiredSpeed}
								class="w-full rounded border p-2 text-sm"
							/>
							<label for="corridor-intensity" class="block text-xs">Desired intensity (veh/h)</label
							>
							<input
								id="corridor-intensity"
								type="number"
								min="0"
								bind:value={$desiredIntensity}
								class="w-full rounded border p-2 text-sm"
							/>
							<label for="corridor-flow" class="block text-xs">Desired flow (veh/s)</label>
							<input
								id="corridor-flow"
								type="number"
								min="0"
								step="0.1"
								value={$desiredFlow}
								on:input={(event) =>
									desiredIntensity.set(
										event.target.value === '' ? undefined : Number(event.target.value) * 3600
									)}
								class="w-full rounded border p-2 text-sm"
							/>
							<a href="/" class="inline-block text-sm text-blue-600 underline"
								>Open corridor diagram</a
							>
						</section>
					{/if}
					{#if inspectorTab === 'selection'}
						{#if !selectedNode && !selectedEdge}<p class="selection-hint">
								Select a junction or road on the canvas to edit it, or choose a junction below.
							</p>{/if}
						<section>
							<h2 class="mb-2 text-sm font-semibold">All junctions</h2>
							{#each $corridorNodes as node (node.id)}<button
									class="block w-full rounded px-2 py-1.5 text-left text-sm hover:bg-blue-50"
									on:click={() => {
										selectedNodeId = node.id;
										selectedEdgeId = null;
										mode = 'select';
									}}>{node.label}</button
								>{/each}
						</section>
					{/if}
				</div>
			</aside>
		{/if}
	</div>
</div>

{#if editing}
	<EditJunctionModal
		junction={editing}
		graph={$corridorGraph}
		assignments={$corridorProject.movements[editing.id] ?? {}}
		inCorridor={$selectedCorridor.nodeIds.includes(editing.id)}
		corridorMovements={corridorEditor.routeMovements(editing.id)}
		corridorGroup={corridorGroupId(editing, $corridorGroupIds)}
		reverseCorridorGroup={$corridorReverseGroupIds[editing.id] ?? null}
		direction={$optimizationDirection}
		saveError={error}
		on:save={saveJunction}
		on:close={() => {
			editing = null;
			error = '';
		}}
		on:delete={(event) => {
			corridorEditor.removeJunction(event.detail.junction.id);
			editing = null;
			deselect();
		}}
	/>
{/if}
{#if pendingEdge && pendingFrom && pendingTo}
	<EditRoadSegmentModal
		fromLabel={pendingFrom.label}
		toLabel={pendingTo.label}
		includeLength={true}
		on:save={addRoad}
		on:close={() => (pendingEdge = null)}
	/>
{/if}

<style>
	.network-workspace {
		height: 100%;
		display: flex;
		flex-direction: column;
		min-height: 0;
	}
	.network-tools {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 7px;
		padding: 12px 20px;
		background: #fff;
		border-bottom: 1px solid #dce3ec;
		flex: none;
	}
	.network-heading {
		margin-right: 15px;
	}
	.network-heading h1 {
		font-size: 15px;
		font-weight: 650;
	}
	.network-tools :global(button) {
		font-size: 12px;
		padding: 7px 10px;
		border-color: #d7e0e8;
	}
	.network-tools :global(button[aria-pressed='true']) {
		background: #e6f6f2;
		color: #0f766e;
		border-color: #a7dacc;
	}
	.network-body {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 325px;
		flex: 1;
		min-height: 0;
	}
	.network-body.inspector-hidden {
		grid-template-columns: minmax(0, 1fr);
	}
	.network-canvas {
		position: relative;
		min-height: 0;
		min-width: 0;
	}
	.network-inspector {
		min-height: 0;
		display: flex;
		flex-direction: column;
		border-left: 1px solid #dce3ec;
		background: #fff;
	}
	.inspector-tabs {
		display: flex;
		padding: 12px 12px 0;
		border-bottom: 1px solid #e2e8f0;
		gap: 5px;
	}
	.inspector-tabs button {
		font-size: 12px;
		padding: 10px 8px;
		color: #64748b;
		border-bottom: 2px solid transparent;
	}
	.inspector-tabs button.active {
		border-color: #0d9488;
		color: #0f766e;
		font-weight: 600;
	}
	.network-inspector-content {
		overflow-y: auto;
		flex: 1;
		min-height: 0;
		padding: 16px;
	}
	.network-inspector-content section + section {
		margin-top: 20px;
	}
	.route-heading {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px;
		margin-bottom: 12px;
	}
	.route-heading h2 {
		font-size: 15px;
		font-weight: 650;
	}
	.selection-hint {
		padding: 12px;
		background: #f1f5f9;
		border-radius: 8px;
		font-size: 13px;
		color: #64748b;
		line-height: 1.5;
		margin-bottom: 20px;
	}
	@media (max-width: 800px) {
		.network-workspace {
			overflow-y: auto;
		}
		.network-body {
			display: flex;
			flex-direction: column;
			flex: none;
		}
		.network-canvas {
			height: 55dvh;
			min-height: 330px;
			flex: none;
		}
		.network-inspector {
			border-left: 0;
			border-top: 1px solid #dce3ec;
		}
		.network-inspector-content {
			overflow: visible;
		}
		.network-tools {
			padding: 10px 12px;
		}
		.network-heading {
			width: 100%;
		}
	}
</style>
