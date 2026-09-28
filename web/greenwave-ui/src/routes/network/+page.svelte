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
		corridorHistory,
		corridorPersistence,
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
	import {
		emptySharedProject,
		parseSharedProject,
		fromLegacyNetwork,
		roadBetween
	} from '$lib/utils/shared-project.js';
	import { demoProject, MAX_LANES } from '$lib/utils/network-project.js';
	import { exportToJSON, prepareInputExport } from '$lib/utils/export-import.js';
	import { corridorGroupId } from '$lib/utils/junction-program.js';
	import { modalFocus } from '$lib/utils/modal-focus.js';

	const legacyNetworkAvailable = corridorEditor.legacyNetworkAvailable;
	const button =
		'rounded-md border border-gray-200 bg-white px-3 py-1.5 text-sm hover:bg-gray-50 disabled:opacity-40 focus-visible:outline-2 focus-visible:outline-blue-500';
	let canvas;
	let mounted = false;
	let mode = 'select';
	let selectedNodeId = null;
	let selectedEdgeId = null;
	let editing = null;
	let pendingEdge = null;
	let replacement = null;
	let importInput;
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
	$: modalOpen = editing !== null || pendingEdge !== null || replacement !== null;
	$: pendingFrom = $corridorNodes.find((node) => node.id === pendingEdge?.fromId);
	$: pendingTo = $corridorNodes.find((node) => node.id === pendingEdge?.toId);

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
	function exportProject() {
		exportToJSON(corridorEditor.snapshot(), 'greenwave-project.json');
	}
	function exportInput() {
		if ($corridorIssue) {
			error = $corridorIssue;
			return;
		}
		exportToJSON(
			prepareInputExport(
				$junctions,
				$desiredSpeed,
				$desiredIntensity,
				$optimizationDirection,
				$corridorGroupIds,
				$corridorReverseGroupIds
			),
			'greenwave-input.json'
		);
	}
	async function applyReplacement(project) {
		corridorEditor.replace(project);
		replacement = null;
		error = '';
		notice = 'Project opened. Undo restores the previous project.';
		deselect();
		mode = 'select';
		await tick();
		canvas?.fit();
	}
	function requestReplacement(project) {
		if ($corridorProject.junctions.length) replacement = project;
		else applyReplacement(project);
	}
	async function importProject(event) {
		const file = event.target.files?.[0];
		event.target.value = '';
		if (!file) return;
		try {
			if (file.size > 5 * 1024 * 1024) throw new Error('Choose a file smaller than 5 MB.');
			requestReplacement(parseSharedProject(await file.text()));
		} catch (cause) {
			error = cause.message;
		}
	}
</script>

<svelte:head><title>Network and corridors | Greenwave</title></svelte:head>
<svelte:window on:keydown={handleKeydown} />

<div class="flex min-h-screen flex-col bg-gray-100 p-2 sm:p-4">
	<div
		class="flex min-h-[calc(100dvh-2rem)] flex-1 flex-col overflow-hidden rounded-xl bg-white shadow-md"
	>
		<header class="flex flex-wrap items-center gap-3 border-b border-gray-200 p-3">
			<a href="/" class="text-sm text-blue-600 hover:underline">← Diagram</a>
			<h1 class="font-semibold">Network and corridors</h1>
			<span
				role="status"
				class="mr-auto text-xs"
				class:text-red-700={$corridorPersistence.state === 'error'}
				class:text-gray-500={$corridorPersistence.state !== 'error'}
				>{$corridorPersistence.message}</span
			>
			<button class={button} on:click={() => requestReplacement(emptySharedProject())}
				>New project</button
			>
			<button class={button} on:click={() => requestReplacement(fromLegacyNetwork(demoProject()))}
				>Example network</button
			>
			<button class={button} on:click={() => importInput.click()}>Import project</button>
			<button class={button} on:click={exportProject}>Export project</button>
			{#if $legacyNetworkAvailable}<button
					class={button}
					on:click={() => attempt(() => requestReplacement(corridorEditor.readLegacyNetwork()))}
					>Open saved network</button
				>{/if}
			<input
				bind:this={importInput}
				type="file"
				accept=".json,application/json"
				class="hidden"
				on:change={importProject}
				aria-label="Import network project"
			/>
		</header>
		<div class="flex flex-wrap items-center gap-2 border-b border-gray-200 px-3 py-2">
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
			<button class={button} disabled={!$corridorHistory.undo} on:click={() => changeHistory()}
				>Undo</button
			>
			<button class={button} disabled={!$corridorHistory.redo} on:click={() => changeHistory(true)}
				>Redo</button
			>
			<button class={button} on:click={() => canvas?.fit()}>Fit</button>
			<span class="ml-auto text-xs text-gray-500"
				>{$corridorNodes.length} junctions · {$corridorEdges.length} roads</span
			>
		</div>
		{#if error}<p role="alert" class="bg-red-50 px-4 py-2 text-sm text-red-700">{error}</p>{/if}
		{#if notice}<p role="status" class="bg-blue-50 px-4 py-2 text-sm text-blue-800">
				{notice}
			</p>{/if}
		{#if $corridorProject.migration}<p class="bg-amber-50 px-4 py-2 text-xs text-amber-900">
				{$corridorProject.migration.notes.join(' ')}
			</p>{/if}
		<div class="flex flex-1 flex-col lg:flex-row">
			<div class="relative h-[55dvh] min-h-[340px] min-w-0 flex-1 lg:h-auto">
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
							selectedEdgeId = null;
						}}
						on:selectEdge={(event) => {
							selectedEdgeId = event.detail.edge.id;
							selectedNodeId = null;
						}}
						on:nodeAdded={(event) => {
							mode = 'select';
							selectedNodeId = event.detail.id;
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
			<aside
				class="w-full shrink-0 space-y-5 border-t border-gray-200 p-4 lg:max-h-[calc(100dvh-12rem)] lg:w-96 lg:overflow-y-auto lg:border-t-0 lg:border-l"
				aria-label="Network and corridor properties"
			>
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
				<section class="space-y-2">
					<label for="active-corridor" class="block text-sm font-semibold">Active corridor</label>
					<div class="flex gap-2">
						<select
							id="active-corridor"
							class="min-w-0 flex-1 rounded border p-2 text-sm"
							value={$selectedCorridor.id}
							on:change={(event) => corridorEditor.selectCorridor(Number(event.target.value))}
						>
							{#each $corridorProject.corridors as route (route.id)}<option value={route.id}
									>{route.name}</option
								>{/each}
						</select>
						<button
							class={button}
							on:click={() =>
								corridorEditor.addCorridor(
									`Corridor ${Math.max(...$corridorProject.corridors.map((route) => route.id)) + 2}`
								)}>+ Corridor</button
						>
					</div>
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
						Select junctions on the network and append them in travel order. Removing a stop keeps
						its junction and roads in the network.
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
										setRoute($selectedCorridor.nodeIds.filter((nodeId) => nodeId !== id))}>×</button
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
						<button class={button} on:click={exportInput} disabled={!!$corridorIssue}
							>Export corridor input</button
						>
						<button
							class={button}
							disabled={$corridorProject.corridors.length === 1}
							on:click={() => corridorEditor.removeCorridor($selectedCorridor.id)}
							>Delete corridor</button
						>
					</div>
				</section>
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
					<label for="corridor-intensity" class="block text-xs">Desired intensity (veh/h)</label>
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
					<a href="/" class="inline-block text-sm text-blue-600 underline">Open corridor diagram</a>
				</section>
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
			</aside>
		</div>
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
{#if replacement}
	<div
		role="dialog"
		aria-modal="true"
		aria-labelledby="replace-title"
		tabindex="-1"
		use:modalFocus
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		on:keydown={(event) => {
			if (event.key === 'Escape') {
				event.stopPropagation();
				replacement = null;
			}
		}}
	>
		<div class="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl">
			<h2 id="replace-title" class="font-semibold">Open {replacement.name}?</h2>
			<p class="mt-3 text-sm">
				This replaces the current network and all its corridors. Export the current project to keep
				a separate copy. Undo can restore it during this session.
			</p>
			<p class="mt-2 text-sm">
				{replacement.junctions.length} junctions · {replacement.roads.length} roads · {replacement
					.corridors.length} corridors
			</p>
			{#if replacement.migration}<p class="mt-2 text-sm text-amber-800">
					{replacement.migration.notes.join(' ')} The original saved network is kept.
				</p>{/if}
			<div class="mt-5 flex flex-wrap justify-end gap-2">
				<button class={button} on:click={exportProject}>Export current</button><button
					class={button}
					on:click={() => (replacement = null)}>Cancel</button
				><button class={button} on:click={() => applyReplacement(replacement)}>Open project</button>
			</div>
		</div>
	</div>
{/if}
