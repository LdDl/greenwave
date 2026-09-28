<script>
  import { onMount, tick } from 'svelte';
  import MacroCanvas from '../../components/MacroCanvas.svelte';
  import EditJunctionModal from '../../components/EditJunctionModal.svelte';
  import { corridorEditor, corridorGraph, corridorNodes, corridorEdges, corridorHistory, corridorPersistence } from '$lib/stores/corridor.js';
  import { junctions, desiredSpeed, desiredIntensity, desiredFlow, optimizationDirection } from '$lib/stores/core.js';
  import { resetToDemo } from '$lib/stores';
  import { calculateTotalDuration } from '$lib/utils/junction-helpers.js';
  import { exportToJSON, prepareInputExport } from '$lib/utils/export-import.js';

  let canvas;
  let mounted = false;
  let selectedNodeId = null;
  let selectedEdgeId = null;
  let editing = null;
  let isNew = false;
  let error = '';
  const button = 'rounded-md border border-gray-200 bg-white px-3 py-2 text-sm hover:bg-gray-50 disabled:opacity-40 focus-visible:outline-2 focus-visible:outline-blue-500';

  onMount(() => {
    mounted = true;
    tick().then(() => canvas?.fit());
  });

  $: selectedJunction = $junctions.find(node => node.id === selectedNodeId);
  $: selectedEdge = $corridorEdges.find(edge => edge.id === selectedEdgeId);

  function openJunction(id) {
    const junction = $junctions.find(node => node.id === id);
    if (!junction) return;
    selectedNodeId = id;
    selectedEdgeId = null;
    editing = structuredClone(junction);
    isNew = false;
  }

  function addJunction() {
    const id = Math.max(-1, ...$junctions.map(node => node.id)) + 1;
    editing = {
      id, label: `Junction ${id + 1}`, offset: 0,
      point: { x: $junctions[0]?.point.x ?? 0, y: Math.max(-100, ...$junctions.map(node => node.point.y)) + 150 },
      cycle: [{ id: 0, signal_groups: [{ id: 0, signals: [{ color: 'GREEN', duration: 30 }, { color: 'RED', duration: 20 }] }] }],
    };
    isNew = true;
  }

  function saveJunction(event) {
    try {
      corridorEditor.saveJunction(event.detail.junction, event.detail.isNew);
      selectedNodeId = event.detail.junction.id;
      editing = null;
      error = '';
    } catch (cause) { error = cause.message; }
  }

  function removeJunction(event) {
    corridorEditor.removeJunction(event.detail.junction.id);
    editing = null;
    selectedNodeId = null;
  }

  function setLength(event) {
    try {
      if (!event.target.value.trim()) throw new Error('Enter the road length in meters.');
      corridorEditor.setRoadLength(selectedEdgeId, Number(event.target.value));
      error = '';
    } catch (cause) {
      error = cause.message;
      event.target.value = selectedEdge.lengthMeters;
    }
  }

  function changeHistory(redo = false) {
    if (redo) corridorEditor.redo(); else corridorEditor.undo();
    error = '';
  }

  function handleKeydown(event) {
    if (editing || event.defaultPrevented || event.target.closest('input, textarea, select, [contenteditable="true"]')) return;
    const key = event.key.toLowerCase();
    if ((event.ctrlKey || event.metaKey) && (key === 'z' || key === 'y')) {
      event.preventDefault();
      changeHistory(key === 'y' || event.shiftKey);
    } else if (!event.ctrlKey && !event.metaKey && !event.altKey && key === 'f') canvas?.fit();
  }

  function exportInput() {
    exportToJSON(prepareInputExport($junctions, $desiredSpeed, $desiredIntensity, $optimizationDirection), 'greenwave-input.json');
  }

  async function loadDemo() {
    resetToDemo();
    await tick();
    canvas?.fit();
  }
</script>

<svelte:head><title>Input Corridor | Greenwave</title></svelte:head>
<svelte:window on:keydown={handleKeydown} />

<div class="flex min-h-screen flex-col bg-gray-100 p-2 sm:p-4">
  <div class="flex min-h-[calc(100dvh-2rem)] flex-1 flex-col overflow-hidden rounded-xl bg-white shadow-md">
    <header class="flex flex-wrap items-center gap-3 border-b border-gray-200 p-3">
      <a href="/" class="text-sm text-blue-600 hover:underline">← Diagram</a>
      <h1 class="font-semibold">Input corridor</h1>
      <span role="status" class="mr-auto text-xs" class:text-red-700={$corridorPersistence.state === 'error'} class:text-gray-500={$corridorPersistence.state !== 'error'}>{$corridorPersistence.message}</span>
      <button class={button} on:click={exportInput}>Export input</button>
      <a href="/network/standalone" class="text-sm text-gray-500 hover:underline">Standalone network</a>
    </header>

    <div class="flex flex-wrap items-center gap-2 border-b border-gray-200 px-3 py-2">
      <button class={button} on:click={addJunction}>+ Add Junction</button>
      <button class={button} disabled={!$corridorHistory.undo} on:click={() => changeHistory()}>Undo</button>
      <button class={button} disabled={!$corridorHistory.redo} on:click={() => changeHistory(true)}>Redo</button>
      <button class={button} on:click={() => canvas?.fit()}>Fit</button>
      <span class="ml-auto text-xs text-gray-500">{$junctions.length} junctions · {$corridorEdges.length} roads</span>
    </div>
    <p class="border-b border-gray-200 px-3 py-2 text-xs text-gray-500">The same junctions and signal programs as "Input configuration". Move nodes to arrange the view; select a road to change its length.</p>
    {#if error || $corridorGraph.error}
      <p role="alert" class="bg-red-50 px-4 py-3 text-sm text-red-700">{error || $corridorGraph.error} <a href="/" class="underline">Open diagram</a> to review the input configuration.</p>
    {/if}

    <div class="flex flex-1 flex-col md:flex-row">
      <div class="relative h-[50dvh] min-h-[300px] min-w-0 flex-1 md:h-auto">
        {#if mounted}
          <MacroCanvas bind:this={canvas} networkNodes={corridorNodes} networkEdges={corridorEdges} networkEditor={corridorEditor.canvas}
            interactionHint="Double-click a junction to edit its signal program"
            mode="select" disabled={editing !== null || !!$corridorGraph.error} {selectedNodeId} {selectedEdgeId}
            highlightFrom={selectedEdge?.from ?? null} highlightTo={selectedEdge?.to ?? null}
            on:selectNode={event => { selectedNodeId = event.detail.node.id; selectedEdgeId = null; }}
            on:selectEdge={event => { selectedEdgeId = event.detail.edge.id; selectedNodeId = null; }}
            on:deselect={() => { selectedNodeId = null; selectedEdgeId = null; }}
            on:openIntersection={event => openJunction(event.detail.nodeId)} />
        {/if}
        {#if !$junctions.length}
          <div class="pointer-events-none absolute inset-0 flex items-center justify-center p-6 text-center">
            <div class="rounded-xl bg-white/90 p-6 shadow-sm">
              <h2 class="font-semibold">No junctions configured</h2>
              <p class="mt-2 text-sm text-gray-500">Add a junction here or load the same demo used by the diagram.</p>
              <button class="pointer-events-auto mt-4 rounded bg-blue-600 px-4 py-2 text-sm text-white" on:click={loadDemo}>Load Demo Data</button>
            </div>
          </div>
        {/if}
      </div>

      <aside class="w-full shrink-0 space-y-5 border-t border-gray-200 p-4 md:w-80 md:border-t-0 md:border-l" aria-label="Corridor properties">
        {#if selectedJunction}
          <section>
            <h2 class="font-semibold">{selectedJunction.label}</h2>
            <p class="mt-2 text-sm text-gray-500">Distance: {selectedJunction.point.y} m · Cycle: {calculateTotalDuration(selectedJunction)} s</p>
            <p class="mt-1 text-sm text-gray-500">Group: G{selectedJunction.cycle[0].signal_groups[0].id} · Offset: {selectedJunction.offset ?? 0} s</p>
            <button class="mt-3 rounded bg-blue-600 px-3 py-2 text-sm text-white" on:click={() => openJunction(selectedJunction.id)}>Edit junction and program</button>
          </section>
        {:else if selectedEdge}
          <section>
            <h2 class="font-semibold">Road segment</h2>
            <p class="mt-2 text-sm text-gray-500">{$junctions.find(node => node.id === selectedEdge.from)?.label} → {$junctions.find(node => node.id === selectedEdge.to)?.label}</p>
            <label for="corridor-road-length" class="mt-3 block text-sm">Length (m)</label>
            <input id="corridor-road-length" type="number" min="0" step="any" value={selectedEdge.lengthMeters} on:change={setLength} class="mt-1 w-full rounded border border-gray-300 px-2 py-1.5" />
            <p class="mt-2 text-xs text-gray-500">Changing this length shifts all following junctions along the corridor.</p>
          </section>
        {:else}
          <p class="text-sm text-gray-500">Select a junction to edit its program or a road to edit its length. Double-click a junction to open its editor.</p>
        {/if}

        <section class="space-y-2">
          <h2 class="text-sm font-semibold">Calculation settings</h2>
          <label for="corridor-direction" class="block text-xs">Optimization direction</label>
          <select id="corridor-direction" bind:value={$optimizationDirection} class="w-full rounded border border-gray-300 px-2 py-1.5 text-sm">
            <option value="forward">Forward</option><option value="bidirectional">Bidirectional</option>
          </select>
          <label for="corridor-speed" class="block text-xs">Desired speed (km/h)</label>
          <input id="corridor-speed" type="number" min="10" max="100" bind:value={$desiredSpeed} class="w-full rounded border border-gray-300 px-2 py-1.5 text-sm" />
          <label for="corridor-intensity" class="block text-xs">Desired intensity (veh/h)</label>
          <input id="corridor-intensity" type="number" min="0" bind:value={$desiredIntensity} class="w-full rounded border border-gray-300 px-2 py-1.5 text-sm" />
          <label for="corridor-flow" class="block text-xs">Desired flow (veh/s)</label>
          <input id="corridor-flow" type="number" min="0" step="0.1" value={$desiredFlow} on:input={event => desiredIntensity.set(event.target.value === '' ? undefined : Number(event.target.value) * 3600)} class="w-full rounded border border-gray-300 px-2 py-1.5 text-sm" />
          <p class="text-xs text-gray-500">Extract waves and view calculated indicators on the <a href="/" class="text-blue-600 underline">diagram</a>.</p>
        </section>

        <section>
          <h2 class="mb-2 text-sm font-semibold">Junctions</h2>
          <div class="space-y-1">
            {#each $junctions as node (node.id)}
              <button class="block w-full rounded px-2 py-1.5 text-left text-sm hover:bg-gray-100" class:bg-blue-50={selectedNodeId === node.id} on:click={() => { selectedNodeId = node.id; selectedEdgeId = null; }}>{node.label} · {node.point.y} m</button>
            {/each}
          </div>
        </section>
      </aside>
    </div>
  </div>
</div>

{#if editing}
  <EditJunctionModal junction={editing} {isNew} on:save={saveJunction} on:delete={removeJunction} on:close={() => editing = null} />
{/if}
