<script>
  import { onMount, tick } from 'svelte';
  import MacroCanvas from '../../../components/MacroCanvas.svelte';
  import EditRoadSegmentModal from '../../../components/EditRoadSegmentModal.svelte';
  import IntersectionEditor from '../../../components/IntersectionEditor.svelte';
  import { networkEditor, networkProject, networkNodes, networkEdges, intersectionConfigs, networkHistory, networkPersistence } from '$lib/stores/network.js';
  import { emptyProject, demoProject, parseProject, MAX_LANES, intersectionTopology, groupConflictPairs } from '$lib/utils/network-project.js';
  import { modalFocus } from '$lib/utils/modal-focus.js';

  let mode = 'select';
  let selectedNodeId = null;
  let selectedEdgeId = null;
  let intersectionNodeId = null;
  let pendingEdge = null;
  let replacement = null;
  let importInput;
  let canvas;
  let notice = '';
  let error = '';
  let importing = false;
  let mounted = false;
  const button = 'rounded-md border border-gray-200 bg-white px-3 py-1.5 text-sm hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed focus-visible:outline-2 focus-visible:outline-blue-500';

  onMount(() => {
    let storage = null;
    // Storage may be disabled by the browser.
    try { storage = window.localStorage; } catch { storage = null; }
    const disconnect = networkEditor.connectStorage(storage);
    mounted = true;
    tick().then(() => canvas?.fit());
    return disconnect;
  });

  $: selectedNode = $networkNodes.find(n => n.id === selectedNodeId) ?? null;
  $: selectedEdge = $networkEdges.find(e => e.id === selectedEdgeId) ?? null;
  $: pendingFrom = $networkNodes.find(n => n.id === pendingEdge?.fromId);
  $: pendingTo = $networkNodes.find(n => n.id === pendingEdge?.toId);
  $: modalOpen = pendingEdge !== null || intersectionNodeId !== null || replacement !== null || importing;
  $: if (mode !== 'select') { selectedNodeId = null; selectedEdgeId = null; }
  $: junctionStatus = Object.fromEntries($networkNodes.map(node => {
    const config = $intersectionConfigs[node.id];
    if (!config) return [node.id, 'Not configured'];
    const { stubs, movements } = intersectionTopology(node.id, $networkNodes, $networkEdges);
    const conflicts = Object.values(groupConflictPairs(movements, config.movState, config.groups, stubs)).flat().length;
    const enabled = movements.filter(m => config.movState[m.id]?.groupIds.length).length;
    return [node.id, conflicts ? `${conflicts} possible conflicts` : `${enabled}/${movements.length} movements allowed`];
  }));

  function deselect() { selectedNodeId = null; selectedEdgeId = null; }
  function selectNode(id) { mode = 'select'; selectedNodeId = id; selectedEdgeId = null; }
  function selectEdge(id) { selectedEdgeId = id; selectedNodeId = null; }

  function deleteSelected() {
    if (!selectedNode && !selectedEdge) return;
    networkEditor.remove(selectedNodeId, selectedEdgeId);
    deselect();
    notice = 'Selection deleted. Use Undo to restore it.';
  }

  function changeHistory(redo = false) {
    if (redo) networkEditor.redo(); else networkEditor.undo();
    deselect();
    canvas?.cancelDrawing();
    notice = '';
  }

  function handleKeydown(event) {
    if (modalOpen || event.defaultPrevented || event.target.closest('input, textarea, select, [contenteditable="true"]')) return;
    const key = event.key.toLowerCase();
    if ((event.ctrlKey || event.metaKey) && (key === 'z' || key === 'y')) {
      event.preventDefault();
      changeHistory(key === 'y' || event.shiftKey);
      return;
    }
    if (event.ctrlKey || event.metaKey || event.altKey) return;
    if (key === 'delete' || key === 'backspace') { event.preventDefault(); deleteSelected(); }
    if (key === 'escape') { deselect(); canvas?.cancelDrawing(); mode = 'select'; }
    if (key === 'v') mode = 'select';
    if (key === 'n') mode = 'node';
    if (key === 'r') mode = 'edge';
    if (key === 'f') canvas?.fit();
  }

  function handleEdgeSave({ detail }) {
    if (!pendingEdge) return;
    const id = networkEditor.addRoad(pendingEdge.fromId, pendingEdge.toId, detail.lanesFwd, detail.lanesBack);
    pendingEdge = null;
    if (id === null) error = 'This road already exists or one of its junctions was removed.';
  }

  function updateLabel(event) {
    const value = event.target.value.trim();
    if (!value) { event.target.value = selectedNode.label; return; }
    networkEditor.change(p => { p.nodes.find(n => n.id === selectedNodeId).label = value; });
  }

  function updateLanes(field, value) {
    const count = Number(value);
    const minimum = field === 'lanes_fwd' ? 1 : 0;
    if (!Number.isInteger(count) || count < minimum || count > MAX_LANES) {
      error = `Use a whole number from ${minimum} to ${MAX_LANES} for the lane count.`;
      return;
    }
    error = '';
    networkEditor.change(p => { p.edges.find(e => e.id === selectedEdgeId)[field] = count; });
    notice = 'Road updated. Movements removed by this change were cleared from intersection groups.';
  }

  function reverseEdge() {
    networkEditor.reverseRoad(selectedEdgeId);
    notice = 'Direction reversed. Review movement assignments at both ends. Undo restores the previous configuration.';
  }

  async function applyReplacement(project) {
    networkEditor.replace(project);
    replacement = null;
    error = '';
    notice = 'Project opened. The previous project is available through Undo.';
    deselect();
    mode = 'select';
    canvas?.cancelDrawing();
    await tick();
    canvas?.fit();
  }

  function requestReplacement(project) {
    if ($networkNodes.length || $networkEdges.length) replacement = project;
    else applyReplacement(project);
  }

  async function importProject(event) {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    importing = true;
    error = '';
    try {
      if (file.size > 5 * 1024 * 1024) throw new Error('Choose a project smaller than 5 MB.');
      const project = parseProject(await file.text());
      requestReplacement(project);
    } catch (cause) {
      error = `Import failed: ${cause.message}`;
    } finally { importing = false; }
  }

  function exportProject() {
    const project = networkEditor.snapshot();
    const blob = new Blob([JSON.stringify(project, null, 2) + '\n'], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${project.name.replace(/[^a-zA-Z0-9а-яА-ЯёЁ_-]+/g, '-').slice(0, 80) || 'network'}.json`;
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
    notice = 'Project exported, including all intersection configurations.';
  }
</script>

<svelte:head><title>Standalone Network | Greenwave</title></svelte:head>
<svelte:window on:keydown={handleKeydown} />

<div class="flex h-dvh min-h-[540px] flex-col bg-gray-100 p-2 sm:p-4">
  <div class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl bg-white shadow-md">
    <header class="flex flex-wrap items-center gap-2 border-b border-gray-200 p-3">
      <a href="/network" class="mr-2 text-sm text-gray-500 hover:text-blue-600">← Input corridor</a>
      <h1 class="text-sm font-semibold">Standalone network</h1>
      <label class="sr-only" for="project-name">Project name</label>
      <input id="project-name" aria-label="Project name" maxlength="120" value={$networkProject.name}
        on:change={e => { const name = e.target.value.trim(); if (name) networkEditor.change(p => { p.name = name; }); else e.target.value = $networkProject.name; }}
        class="min-w-0 w-44 rounded border border-transparent px-2 py-1 text-sm font-semibold hover:border-gray-200 focus:border-blue-400" />
      <span class="mr-auto text-xs text-gray-500" role="status">{$networkPersistence.state === 'saved' ? $networkPersistence.message : 'Changes are kept in memory'}</span>
      <button class={button} on:click={() => requestReplacement(emptyProject())}>New</button>
      <button class={button} on:click={() => requestReplacement(demoProject())}>Example</button>
      <input class="hidden" type="file" accept=".json,application/json" bind:this={importInput} on:change={importProject} aria-label="Import network project" />
      <button class={button} disabled={importing} on:click={() => importInput.click()}>Import</button>
      <button class={button} on:click={exportProject}>Export</button>
    </header>

    <p class="border-b border-gray-200 px-3 py-2 text-xs text-gray-500">This is a separate network project with movement groups. Open "Input corridor" to edit the junctions from the diagram.</p>
    <div class="flex flex-wrap items-center gap-2 border-b border-gray-200 px-3 py-2">
      <div class="inline-flex gap-1 rounded-lg bg-gray-100 p-1" aria-label="Editor tools">
        {#each [{ id: 'select', text: 'Select', key: 'V' }, { id: 'node', text: '+ Junction', key: 'N' }, { id: 'edge', text: '+ Road', key: 'R' }] as tool (tool.id)}
          <button class="rounded px-3 py-1 text-sm focus-visible:outline-2 focus-visible:outline-blue-500" class:bg-white={mode === tool.id} class:shadow-sm={mode === tool.id} aria-pressed={mode === tool.id} title={`${tool.text} (${tool.key})`} on:click={() => mode = tool.id}>{tool.text}</button>
        {/each}
      </div>
      <button class={button} disabled={!$networkHistory.undo} title="Undo (Ctrl/Cmd+Z)" on:click={() => changeHistory()}>Undo</button>
      <button class={button} disabled={!$networkHistory.redo} title="Redo (Ctrl/Cmd+Shift+Z)" on:click={() => changeHistory(true)}>Redo</button>
      <button class={button} disabled={!selectedNode && !selectedEdge} on:click={deleteSelected}>Delete</button>
      <span class="ml-auto text-xs text-gray-500">{$networkNodes.length} junctions · {$networkEdges.length} roads</span>
    </div>

    {#if error || $networkPersistence.state === 'error'}
      <div role="alert" class="flex items-center gap-3 bg-red-50 px-4 py-2 text-sm text-red-700">
        <span class="flex-1">{error || $networkPersistence.message}</span>
        {#if error}<button aria-label="Dismiss error" on:click={() => error = ''}>Close</button>{/if}
      </div>
    {:else if notice}
      <div role="status" class="flex items-center gap-3 bg-blue-50 px-4 py-2 text-xs text-blue-800"><span class="flex-1">{notice}</span><button aria-label="Dismiss message" on:click={() => notice = ''}>Close</button></div>
    {/if}

    <div class="flex min-h-0 flex-1 flex-col md:flex-row">
      <div class="relative min-h-[280px] min-w-0 flex-1 overflow-hidden">
        {#if mounted}
          <MacroCanvas bind:this={canvas} {mode} disabled={modalOpen} {selectedNodeId} {selectedEdgeId}
            highlightFrom={selectedEdge?.from ?? null} highlightTo={selectedEdge?.to ?? null}
            on:selectNode={e => selectNode(e.detail.node.id)} on:selectEdge={e => selectEdge(e.detail.edge.id)}
            on:deselect={deselect} on:edgePending={e => pendingEdge = e.detail}
            on:openIntersection={e => intersectionNodeId = e.detail.nodeId}
            on:notice={e => notice = e.detail} />
        {/if}
        {#if !$networkNodes.length}
          <div class="pointer-events-none absolute inset-0 flex items-center justify-center p-6 text-center">
            <div class="max-w-sm rounded-xl bg-white/90 p-6 shadow-sm">
              <h2 class="font-semibold text-gray-700">Build your road network</h2>
              <p class="mt-2 text-sm text-gray-500">Choose "+ Junction" and click the canvas. Connect junctions with "+ Road", then configure their movements.</p>
              <button class="pointer-events-auto mt-4 rounded bg-blue-600 px-4 py-2 text-sm text-white" on:click={() => requestReplacement(demoProject())}>Open example</button>
            </div>
          </div>
        {/if}
      </div>

      <aside class="max-h-[40vh] w-full shrink-0 overflow-y-auto border-t border-gray-200 bg-white p-4 md:max-h-none md:w-64 md:border-t-0 md:border-l" aria-label="Network properties">
        {#if selectedNode}
          <h2 class="mb-3 text-sm font-semibold">Junction</h2>
          <label for="node-label" class="block text-xs text-gray-500">Label</label>
          <input id="node-label" maxlength="120" value={selectedNode.label} on:change={updateLabel} class="mt-1 w-full rounded border border-gray-300 px-2 py-1.5 text-sm" />
          <p class="mt-3 text-xs text-gray-500">{junctionStatus[selectedNode.id]}</p>
          <button class="mt-3 w-full rounded bg-blue-600 px-3 py-2 text-sm text-white hover:bg-blue-700" on:click={() => intersectionNodeId = selectedNode.id}>Configure intersection</button>
          <p class="mt-3 text-xs text-gray-400">Canvas position: {Math.round(selectedNode.x)}, {Math.round(selectedNode.y)}</p>
        {:else if selectedEdge}
          {@const from = $networkNodes.find(n => n.id === selectedEdge.from)}
          {@const to = $networkNodes.find(n => n.id === selectedEdge.to)}
          <h2 class="mb-3 text-sm font-semibold">Road segment</h2>
          <p class="mb-3 text-xs text-gray-500">{from.label} {selectedEdge.lanes_back ? '↔' : '→'} {to.label}</p>
          <div class="mb-4 flex gap-1">
            <button class={button} aria-pressed={!selectedEdge.lanes_back} on:click={() => updateLanes('lanes_back', 0)}>One-way</button>
            <button class={button} aria-pressed={selectedEdge.lanes_back > 0} on:click={() => { if (!selectedEdge.lanes_back) updateLanes('lanes_back', 1); }}>Two-way</button>
          </div>
          <label for="lanes-fwd" class="block text-xs text-gray-500">Lanes: {from.label} → {to.label}</label>
          <input id="lanes-fwd" type="number" min="1" max={MAX_LANES} step="1" value={selectedEdge.lanes_fwd} on:change={e => updateLanes('lanes_fwd', e.target.value)} class="mt-1 mb-3 w-full rounded border border-gray-300 px-2 py-1.5 text-sm" />
          {#if selectedEdge.lanes_back}
            <label for="lanes-back" class="block text-xs text-gray-500">Lanes: {to.label} → {from.label}</label>
            <input id="lanes-back" type="number" min="1" max={MAX_LANES} step="1" value={selectedEdge.lanes_back} on:change={e => updateLanes('lanes_back', e.target.value)} class="mt-1 w-full rounded border border-gray-300 px-2 py-1.5 text-sm" />
          {:else}
            <button class={button} on:click={reverseEdge}>Reverse direction</button>
          {/if}
        {:else}
          <h2 class="mb-3 text-sm font-semibold">Junctions</h2>
          <p class="mb-3 text-xs text-gray-500">Select a road or junction to edit its properties.</p>
          <div class="space-y-1">
            {#each $networkNodes as node (node.id)}
              <button class="w-full rounded border border-gray-100 p-2 text-left hover:bg-blue-50" on:click={() => selectNode(node.id)}>
                <span class="block truncate text-sm">{node.label}</span><span class="block text-xs text-gray-400">{junctionStatus[node.id]}</span>
              </button>
            {/each}
          </div>
          <p class="mt-4 text-xs leading-relaxed text-gray-400">V: select · N: junction · R: road · F: fit<br />Delete: remove · Escape: cancel<br />Ctrl/Cmd+Z: undo · Shift+Z: redo</p>
        {/if}
      </aside>
    </div>
  </div>
</div>

{#if pendingEdge && pendingFrom && pendingTo}
  <EditRoadSegmentModal fromLabel={pendingFrom.label} toLabel={pendingTo.label} on:save={handleEdgeSave} on:close={() => pendingEdge = null} />
{/if}
{#if intersectionNodeId !== null}
  <IntersectionEditor nodeId={intersectionNodeId} on:close={() => intersectionNodeId = null} />
{/if}
{#if replacement}
  <div role="dialog" aria-modal="true" aria-labelledby="replace-title" tabindex="-1" use:modalFocus
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
    on:keydown={e => { if (e.key === 'Escape') { e.stopPropagation(); replacement = null; } }}>
    <div class="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
      <h2 id="replace-title" class="font-semibold">Open {replacement.name}?</h2>
      <p class="mt-3 text-sm text-gray-600">This replaces the current network. Export it first to keep a separate copy. You can also Undo the replacement during this session.</p>
      <div class="mt-5 flex justify-end gap-2">
        <button class={button} on:click={exportProject}>Export current</button>
        <button class={button} on:click={() => replacement = null}>Cancel</button>
        <button class="rounded bg-blue-600 px-3 py-1.5 text-sm text-white" on:click={() => applyReplacement(replacement)}>Open project</button>
      </div>
    </div>
  </div>
{/if}
