<script>
  import MacroCanvas from '../../components/MacroCanvas.svelte';
  import EditRoadSegmentModal from '../../components/EditRoadSegmentModal.svelte';
  import IntersectionEditor from '../../components/IntersectionEditor.svelte';
  import { networkNodes, networkEdges, nextEdgeId } from '$lib/stores/network.js';

  let mode = 'select'; // 'select' | 'node' | 'edge'

  let selectedNodeId = null;
  let selectedEdgeId = null;

  // Always reflect latest store values
  $: selectedNode = selectedNodeId != null
    ? $networkNodes.find(n => n.id === selectedNodeId) ?? null
    : null;
  $: selectedEdge = selectedEdgeId != null
    ? $networkEdges.find(e => e.id === selectedEdgeId) ?? null
    : null;

  // Intersection editor

  let intersectionNodeId = null;

  function handleOpenIntersection({ detail }) {
    intersectionNodeId = detail.nodeId;
  }

  // Pending edge (modal)

  let pendingEdge = null; // { fromId, toId }

  $: pendingFrom = pendingEdge ? $networkNodes.find(n => n.id === pendingEdge.fromId) : null;
  $: pendingTo   = pendingEdge ? $networkNodes.find(n => n.id === pendingEdge.toId)   : null;

  function handleEdgePending({ detail }) {
    pendingEdge = { fromId: detail.fromId, toId: detail.toId };
  }

  function handleEdgeSave({ detail }) {
    if (!pendingEdge) return;
    const id = nextEdgeId();
    networkEdges.update(es => [
      ...es,
      { id, from: pendingEdge.fromId, to: pendingEdge.toId,
        lanes_fwd: detail.lanesFwd, lanes_back: detail.lanesBack },
    ]);
    pendingEdge = null;
  }

  function handleEdgeModalClose() {
    pendingEdge = null;
  }

  // Clear selection when leaving select mode
  $: if (mode !== 'select') {
    selectedNodeId = null;
    selectedEdgeId = null;
  }

  // Selection handlers

  function handleSelectNode({ detail }) {
    selectedNodeId = detail.node.id;
    selectedEdgeId = null;
  }

  function handleSelectEdge({ detail }) {
    selectedEdgeId = detail.edge.id;
    selectedNodeId = null;
  }

  function handleDeselect() {
    selectedNodeId = null;
    selectedEdgeId = null;
  }

  // Delete

  function deleteSelected() {
    if (selectedNodeId != null) {
      networkNodes.update(ns => ns.filter(n => n.id !== selectedNodeId));
      networkEdges.update(es => es.filter(e => e.from !== selectedNodeId && e.to !== selectedNodeId));
      selectedNodeId = null;
    } else if (selectedEdgeId != null) {
      networkEdges.update(es => es.filter(e => e.id !== selectedEdgeId));
      selectedEdgeId = null;
    }
  }

  function handleKeydown(e) {
    if (pendingEdge || intersectionNodeId != null) return; // modal is open — block shortcuts
    if (e.target.matches('input, textarea, select')) return;
    if (e.key === 'Delete' || e.key === 'Backspace') deleteSelected();
    if (e.key === 'Escape') { handleDeselect(); mode = 'select'; }
  }

  // Edit helpers

  function updateNodeLabel(value) {
    networkNodes.update(ns =>
      ns.map(n => n.id === selectedNodeId ? { ...n, label: value } : n)
    );
  }

  function updateEdgeLanesFwd(value) {
    const v = Math.max(1, parseInt(value) || 1);
    networkEdges.update(es =>
      es.map(e => e.id === selectedEdgeId ? { ...e, lanes_fwd: v } : e)
    );
  }

  function updateEdgeLanesBack(value) {
    const v = Math.max(0, parseInt(value) || 0);
    networkEdges.update(es =>
      es.map(e => e.id === selectedEdgeId ? { ...e, lanes_back: v } : e)
    );
  }

  // Stats
  $: nodeCount = $networkNodes.length;
  $: edgeCount = $networkEdges.length;
  $: hasSelection = selectedNodeId != null || selectedEdgeId != null;
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="min-h-screen bg-gray-100 p-4 flex flex-col">

  <div class="flex-1 bg-white rounded-xl shadow-md flex flex-col overflow-hidden min-h-[500px]">

  <!-- Toolbar -->
  <div class="border-b border-gray-200 px-4 h-12 flex items-center gap-3 shrink-0 rounded-t-xl">

    <a href="/" class="text-sm text-gray-400 hover:text-gray-600 flex items-center gap-1">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
      </svg>
      Greenwave
    </a>

    <div class="w-px h-5 bg-gray-200"></div>
    <h1 class="text-sm font-semibold text-gray-700 mr-2">Network Editor</h1>

    <!-- Mode switcher -->
    <div class="inline-flex bg-gray-100 rounded-lg p-0.5 gap-0.5 text-sm">
      <button
        class="px-3 py-1 rounded-md font-medium transition-all"
        class:bg-white={mode === 'select'} class:shadow-sm={mode === 'select'}
        class:text-gray-900={mode === 'select'} class:text-gray-500={mode !== 'select'}
        on:click={() => mode = 'select'}
      >Select</button>
      <button
        class="px-3 py-1 rounded-md font-medium transition-all"
        class:bg-white={mode === 'node'} class:shadow-sm={mode === 'node'}
        class:text-gray-900={mode === 'node'} class:text-gray-500={mode !== 'node'}
        on:click={() => mode = 'node'}
      >+ Junction</button>
      <button
        class="px-3 py-1 rounded-md font-medium transition-all"
        class:bg-white={mode === 'edge'} class:shadow-sm={mode === 'edge'}
        class:text-gray-900={mode === 'edge'} class:text-gray-500={mode !== 'edge'}
        on:click={() => mode = 'edge'}
      >+ Road segment</button>
    </div>

    <div class="w-px h-5 bg-gray-200"></div>

    <button
      class="px-3 py-1 text-sm rounded-md transition-all"
      class:bg-red-50={hasSelection} class:text-red-600={hasSelection}
      class:hover:bg-red-100={hasSelection} class:text-gray-300={!hasSelection}
      class:cursor-not-allowed={!hasSelection}
      disabled={!hasSelection}
      on:click={deleteSelected}
    >Delete</button>

    <div class="ml-auto text-xs text-gray-400">
      {nodeCount} junction{nodeCount !== 1 ? 's' : ''}
      &nbsp;·&nbsp;
      {edgeCount} road{edgeCount !== 1 ? 's' : ''}
    </div>
  </div>

  <!-- Main area -->
  <div class="flex-1 flex min-h-0">

    <!-- Canvas: relative + overflow-hidden so absolute child fills it correctly -->
    <div class="flex-1 min-w-0 relative overflow-hidden">
      <MacroCanvas
        {mode}
        highlightFrom={selectedEdge ? selectedEdge.from : null}
        highlightTo={selectedEdge ? selectedEdge.to : null}
        on:selectNode={handleSelectNode}
        on:selectEdge={handleSelectEdge}
        on:deselect={handleDeselect}
        on:edgePending={handleEdgePending}
        on:openIntersection={handleOpenIntersection}
      />
    </div>

    <!-- Side panel -->
    <div class="w-60 bg-white border-l border-gray-200 flex flex-col shrink-0 overflow-y-auto">
      {#if selectedNode}
        <div class="p-4 space-y-4">
          <p class="text-xs font-semibold text-gray-400 uppercase tracking-wide">Junction</p>
          <div>
            <label for="node-label" class="text-xs text-gray-500 block mb-1">Label</label>
            <input
              id="node-label"
              type="text"
              value={selectedNode.label}
              on:input={e => updateNodeLabel(e.target.value)}
              class="w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>
          <div class="text-xs text-gray-400 space-y-0.5">
            <div>x: {Math.round(selectedNode.x)}</div>
            <div>y: {Math.round(selectedNode.y)}</div>
          </div>
        </div>

      {:else if selectedEdge}
        {@const fromNode = $networkNodes.find(n => n.id === selectedEdge.from)}
        {@const toNode   = $networkNodes.find(n => n.id === selectedEdge.to)}
        {@const isTwoWay = selectedEdge.lanes_back > 0}
        <div class="p-4 space-y-4">
          <p class="text-xs font-semibold text-gray-400 uppercase tracking-wide">Road Segment</p>
          {#if fromNode && toNode}
            <p class="text-xs text-gray-500">{fromNode.label} ↔ {toNode.label}</p>
          {/if}

          <!-- One-way / Two-way -->
          <div class="inline-flex w-full bg-gray-100 rounded-lg p-0.5 gap-0.5 text-xs">
            <button
              class="flex-1 py-1.5 rounded-md font-medium transition-all"
              class:bg-white={!isTwoWay} class:shadow-sm={!isTwoWay}
              class:text-gray-900={!isTwoWay} class:text-gray-500={isTwoWay}
              on:click={() => updateEdgeLanesBack(0)}
            >One-way →</button>
            <button
              class="flex-1 py-1.5 rounded-md font-medium transition-all"
              class:bg-white={isTwoWay} class:shadow-sm={isTwoWay}
              class:text-gray-900={isTwoWay} class:text-gray-500={!isTwoWay}
              on:click={() => { if (!isTwoWay) updateEdgeLanesBack(1); }}
            >Two-way ↔</button>
          </div>

          <div>
            <label for="lanes-fwd" class="text-xs text-gray-500 block mb-1">
              Lanes → {fromNode && toNode ? `(${fromNode.label}→${toNode.label})` : ''}
            </label>
            <input id="lanes-fwd" type="number" min="1"
              value={selectedEdge.lanes_fwd}
              on:input={e => updateEdgeLanesFwd(e.target.value)}
              class="w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>

          {#if isTwoWay}
            <div>
              <label for="lanes-back" class="text-xs text-gray-500 block mb-1">
                Lanes ← {fromNode && toNode ? `(${toNode.label}→${fromNode.label})` : ''}
              </label>
              <input id="lanes-back" type="number" min="1"
                value={selectedEdge.lanes_back}
                on:input={e => updateEdgeLanesBack(e.target.value)}
                class="w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
            </div>
          {/if}

          <div class="pt-2 border-t border-gray-100 text-xs text-gray-400">
            Meso road links: {selectedEdge.lanes_fwd + selectedEdge.lanes_back}
          </div>
        </div>

      {:else}
        <div class="flex-1 flex flex-col items-center justify-center gap-2 p-6 text-center text-gray-400">
          <svg class="w-8 h-8 opacity-40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
              d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6-3V7m6 13l4.553 2.276A1 1 0 0021 21.382V10.618a1 1 0 00-.553-.894L15 7m0 13V7m0 0L9 7" />
          </svg>
          <p class="text-xs leading-snug">Select a junction or road to edit properties</p>
          <p class="text-xs text-gray-300 mt-2">Del — delete selected<br/>Esc — deselect<br/>Double-click junction — edit intersection</p>
        </div>
      {/if}
    </div>
  </div>
  </div>
</div>

<!-- Road segment config modal -->
{#if pendingEdge && pendingFrom && pendingTo}
  <EditRoadSegmentModal
    fromLabel={pendingFrom.label}
    toLabel={pendingTo.label}
    on:save={handleEdgeSave}
    on:close={handleEdgeModalClose}
  />
{/if}

<!-- Intersection editor -->
<IntersectionEditor
  nodeId={intersectionNodeId}
  on:close={() => intersectionNodeId = null}
/>
