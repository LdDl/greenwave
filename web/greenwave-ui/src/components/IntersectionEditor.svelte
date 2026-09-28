<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import IntersectionDiagram from './IntersectionDiagram.svelte';
  import { fade, scale } from 'svelte/transition';
  import { networkNodes, networkEdges, intersectionConfigs, networkEditor } from '$lib/stores/network.js';

  import { intersectionTopology, groupConflictPairs, MAX_GREEN } from '$lib/utils/network-project.js';
  import { modalFocus } from '$lib/utils/modal-focus.js';

  export let nodeId = null;

  const dispatch = createEventDispatcher();

  const GROUP_COLORS = ['#3b82f6', '#f97316', '#8b5cf6', '#14b8a6', '#ec4899'];

  // State
  let node = null;
  let stubs = [];
  let movements = [];
  let movState = {};
  let groups = [];
  let selMovId = null;
  let showAllPills = false;
  let original = '';
  let discard = false;
  let saveError = '';
  let conflictSelection = [];
  let past = [];
  let future = [];

  function snapshot() { return JSON.stringify({ groups, movState }); }
  function remember() { past = [...past.slice(-99), snapshot()]; future = []; discard = false; }
  function restore(value) { const draft = JSON.parse(value); groups = draft.groups; movState = draft.movState; }
  function undoDraft() { if (!past.length) return; future = [...future, snapshot()]; restore(past[past.length - 1]); past = past.slice(0, -1); }
  function redoDraft() { if (!future.length) return; past = [...past, snapshot()]; restore(future[future.length - 1]); future = future.slice(0, -1); }
  $: dirty = JSON.stringify({ groups, movState }) !== original;
  $: invalidGreen = groups.some(g => !Number.isInteger(g.greenDuration) || g.greenDuration < 1 || g.greenDuration > MAX_GREEN);
  $: forbiddenCount = movements.filter(m => !movState[m.id]?.groupIds.length).length;
  $: conflictCount = Object.values(groupConflicts).reduce((sum, pairs) => sum + pairs.length, 0);

  function requestClose() {
    if (dirty) discard = true;
    else dispatch('close');
  }
  function dialogKeys(event) {
    if (event.key === 'Escape') { event.stopPropagation(); event.preventDefault(); if (discard) discard = false; else requestClose(); }
    if (event.target.closest('input, select, textarea')) return;
    if ((event.ctrlKey || event.metaKey) && ['z', 'y'].includes(event.key.toLowerCase())) {
      event.preventDefault(); event.stopPropagation();
      if (event.shiftKey || event.key.toLowerCase() === 'y') redoDraft(); else undoDraft();
    }
  }
  function beforeUnload(event) { if (dirty) { event.preventDefault(); event.returnValue = ''; } }


  onMount(() => {
    node = $networkNodes.find(n => n.id === nodeId) ?? null;
    if (node) init();
  });

  // Build stubs & movements from graph
  function init() {
    const topology = intersectionTopology(nodeId, $networkNodes, $networkEdges);
    stubs = topology.stubs;
    movements = topology.movements;
    const saved = $intersectionConfigs[nodeId];
    groups = saved ? JSON.parse(JSON.stringify(saved.groups)) : [{ id: 0, greenDuration: 30 }];
    movState = Object.fromEntries(movements.map(m => [m.id, { groupIds: [...(saved?.movState[m.id]?.groupIds ?? [])] }]));
    selMovId = null;
    original = snapshot();
    discard = false;
    past = [];
    future = [];
    conflictSelection = [];
  }

  let hovGroupId = null;

  function toggleGroup(movId, gid) {
    remember();
    const cur = movState[movId]?.groupIds || [];
    const has = cur.includes(gid);
    const next = has ? cur.filter(id => id !== gid) : [...cur, gid];
    movState = { ...movState, [movId]: { groupIds: next } };
  }

  function forbidMov(movId) {
    remember();
    movState = { ...movState, [movId]: { groupIds: [] } };
  }

  function addGroup() {
    remember();
    const maxId = groups.length ? Math.max(...groups.map(g => g.id)) : -1;
    groups = [...groups, { id: maxId + 1, greenDuration: 30 }];
  }

  function removeGroup(gid) {
    if (groups.length <= 1) return;
    remember();
    groups = groups.filter(g => g.id !== gid);
    const ns = { ...movState };
    for (const m of movements) {
      ns[m.id] = { groupIds: (ns[m.id]?.groupIds || []).filter(id => id !== gid) };
    }
    movState = ns;
  }

  function updGreen(gid, val) {
    remember();
    groups = groups.map(g => g.id === gid ? { ...g, greenDuration: Number(val) } : g);
  }

  function save() {
    if (invalidGreen) return;
    const config = {
      macro_node_id: nodeId,
      groups: groups.map(g => ({ id: g.id, greenDuration: g.greenDuration })),
      movState: { ...movState },
    };
    try {
      networkEditor.saveIntersection(nodeId, config);
      dispatch('close');
    } catch (error) { saveError = error.message; }
  }

  function handleBackdrop(e) {
    if (e.target === e.currentTarget) requestClose();
  }

  $: selMov = selMovId ? movements.find(m => m.id === selMovId) : null;
  $: if (!selMovId) showAllPills = false;
  $: pillMaxVisible = showAllPills ? groups.length : 4;
  $: pillVisible = groups.slice(0, pillMaxVisible);
  $: pillHiddenCount = groups.length - pillMaxVisible;

  function gColor(id) { return GROUP_COLORS[id % GROUP_COLORS.length]; }

  // Reactive map: groupId -> assigned movements
  $: assignedByGroup = (() => {
    const ms = movState;
    const result = {};
    for (const g of groups) {
      result[g.id] = movements.filter(m => (ms[m.id]?.groupIds || []).includes(g.id));
    }
    return result;
  })();

  // Determine turn type from angle difference
  function turnType(mov) {
    const si = stubs.find(s => s.edgeId === mov.inEdgeId);
    const so = stubs.find(s => s.edgeId === mov.outEdgeId);
    if (!si || !so) return '';
    const inFacing = (si.displayAngle ?? si.angle) + Math.PI;
    const outDir = so.displayAngle ?? so.angle;
    let diff = outDir - inFacing;
    while (diff > Math.PI) diff -= 2 * Math.PI;
    while (diff <= -Math.PI) diff += 2 * Math.PI;
    const deg = Math.abs(diff * 180 / Math.PI);
    if (deg < 30) return 'straight';
    if (diff > 0) return 'right';
    return 'left';
  }

  const TURN_ICONS = { left: '\u2B9C', straight: '\u2B9D', right: '\u2B9E' };

  $: groupConflicts = groupConflictPairs(movements, movState, groups, stubs);

  function selectConflict(pair) {
    conflictSelection = pair.map(m => m.id);
    selMovId = pair[0].id;
  }
</script>

<svelte:window on:beforeunload={beforeUnload} />

{#if nodeId != null && node}
<div
  transition:fade={{ duration: 150 }}
  class="fixed inset-0 z-50 flex items-center justify-center"
  style="background-color: rgba(0,0,0,0.55)"
  on:click={handleBackdrop}
  on:keydown={dialogKeys}
  use:modalFocus
  role="dialog" aria-modal="true" aria-labelledby="intersection-title" tabindex="-1"
>
  <div
    transition:scale={{ start: 0.97, duration: 150 }}
    class="bg-white rounded-xl shadow-xl w-full max-w-6xl mx-4 flex flex-col overflow-hidden"
    style="height: 92vh"
  >
    <!-- Header -->
    <div class="flex items-center gap-3 px-6 py-4 border-b border-gray-200 shrink-0">
      <svg class="w-4 h-4 text-gray-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M8 9l4-4 4 4m0 6l-4 4-4-4"/>
      </svg>
      <h2 id="intersection-title" class="text-sm font-semibold text-gray-800">{node.label}: Intersection</h2>
      <span class="hidden md:block text-xs text-gray-400">Click arc to select · assign group or mark forbidden</span>
      <button
        class="ml-auto p-1.5 text-gray-400 hover:text-gray-600 rounded-md hover:bg-gray-100
               focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
        on:click={requestClose}
        aria-label="Close"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
        </svg>
      </button>
    </div>

    <!-- Body -->
    <div class="flex flex-1 min-h-0 flex-col md:flex-row">

      <!-- Left: SVG road diagram -->
      <IntersectionDiagram {stubs} {movements} {movState} bind:selMovId {hovGroupId} {conflictSelection}
        on:select={() => conflictSelection = []} />

      <!-- Right panel -->
      <div class="w-full md:w-80 max-h-[45vh] md:max-h-none shrink-0 overflow-y-auto md:overflow-visible border-l border-gray-200 bg-gray-50/40 flex flex-col min-h-0" role="region" aria-label="Intersection settings">

        <!-- Fixed top section -->
        <div class="p-4 pb-0 space-y-4 shrink-0">
          <label for="movement-select" class="block text-xs font-medium text-gray-600">Movements ({movements.length})</label>
          <select id="movement-select" class="w-full rounded border border-gray-300 bg-white p-2 text-sm" value={selMovId ?? ''}
            on:change={e => { selMovId = e.target.value || null; conflictSelection = []; }}>
            <option value="">Select a movement</option>
            {#each movements as movement (movement.id)}
              <option value={movement.id}>{movement.inLabel} → {movement.outLabel}{!movState[movement.id]?.groupIds.length ? ' (forbidden)' : ''}</option>
            {/each}
          </select>
          {#if !movements.length}<p class="text-xs text-gray-500">No through or turning movements. Connect another road with an available incoming or outgoing direction.</p>{/if}
          <!-- Selected movement -->
          <div class="rounded-lg border border-gray-200 bg-white p-3 shrink-0 flex flex-col"
          >
            {#if selMov}
              {@const gids = movState[selMov.id]?.groupIds || []}
              {@const turn = turnType(selMov)}
              <div class="space-y-2.5">
                <div class="flex items-center justify-between">
                  <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Selected movement</p>
                  {#if turn}
                    <span class="text-xs text-gray-400 capitalize">{TURN_ICONS[turn]} {turn}</span>
                  {/if}
                </div>
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="px-2 py-0.5 rounded-md text-xs font-semibold bg-blue-50 text-blue-700 border border-blue-200">
                    {selMov.inLabel}
                  </span>
                  <svg class="w-3 h-3 text-gray-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                  </svg>
                  <span class="px-2 py-0.5 rounded-md text-xs font-semibold bg-orange-50 text-orange-700 border border-orange-200">
                    {selMov.outLabel}
                  </span>
                </div>
                <div class="space-y-1.5">
                  <button
                    on:click={() => forbidMov(selMov.id)}
                    class="px-2 py-0.5 text-xs rounded-md font-medium border transition-colors"
                    style="{gids.length === 0
                      ? 'background:#fef2f2; color:#dc2626; border-color:#fecaca'
                      : 'background:white; color:#9ca3af; border-color:#e5e7eb'}"
                  >Forbidden</button>
                  <div class="flex flex-wrap gap-1">
                    {#each pillVisible as g (g.id)}
                      {@const active = gids.includes(g.id)}
                      <button
                        on:click={() => toggleGroup(selMov.id, g.id)}
                        class="px-2 py-0.5 text-xs rounded-md font-medium border transition-colors"
                        style="{active
                          ? `background:${gColor(g.id)}; color:white; border-color:${gColor(g.id)}`
                          : `color:${gColor(g.id)}; border-color:${gColor(g.id)}44; background:white`}"
                      >G{g.id}</button>
                    {/each}
                    {#if pillHiddenCount > 0}
                      <button
                        on:click={() => showAllPills = true}
                        class="px-2 py-0.5 text-xs rounded-md font-medium border border-gray-200 text-gray-400 hover:text-gray-600 hover:bg-gray-50"
                      >+{pillHiddenCount} more</button>
                    {/if}
                    {#if showAllPills && groups.length > 4}
                      <button
                        on:click={() => showAllPills = false}
                        class="px-2 py-0.5 text-xs rounded-md font-medium border border-gray-200 text-gray-400 hover:text-gray-600 hover:bg-gray-50"
                      >less</button>
                    {/if}
                  </div>
                </div>
              </div>
            {:else}
              <div class="h-[104px] flex items-center justify-center">
                <p class="text-xs text-gray-400 italic leading-snug text-center">
                  Click any arc to select it,<br/>
                  then assign a group or mark forbidden.
                </p>
              </div>
            {/if}
          </div>

          <!-- Signal groups header -->
          <div class="flex items-center justify-between">
            <p class="text-xs font-semibold text-gray-400 uppercase tracking-wide">Movement groups</p>
            <button
              on:click={addGroup}
              class="text-xs px-2 py-1 rounded-md bg-white border border-gray-200 hover:bg-gray-100
                     text-gray-600 font-medium focus-visible:outline-none focus-visible:ring-2
                     focus-visible:ring-gray-400"
            >+ Add</button>
          </div>
        </div>

        <!-- Scrollable groups list -->
        <div class="shrink-0 md:flex-1 md:shrink md:min-h-0 md:overflow-y-auto px-4 py-2">
          {#if groups.length === 0}
            <p class="text-xs text-gray-400 text-center py-6 leading-snug">
              Add a group to assign movements
            </p>
          {:else}
            <div class="space-y-3">
              {#each groups as g (g.id)}
                {@const assigned = assignedByGroup[g.id] || []}
                <div class="rounded-lg border bg-white p-3 space-y-2.5"
                  style="border-color: {gColor(g.id)}55"
                  class:ring-2={selMovId && (movState[selMovId]?.groupIds || []).includes(g.id)}
                  class:ring-blue-400={selMovId && (movState[selMovId]?.groupIds || []).includes(g.id)}
                  on:pointerenter={() => hovGroupId = g.id}
                  on:pointerleave={() => { if (hovGroupId === g.id) hovGroupId = null; }}
                >
                  <div class="flex items-center gap-2 h-6">
                    <div class="w-3 h-3 rounded-full shrink-0" style="background:{gColor(g.id)}"></div>
                    <span class="text-sm font-semibold text-gray-700">Group {g.id}</span>
                    <div class="ml-auto">
                      {#if selMovId}
                        {@const inGroup = (movState[selMovId]?.groupIds || []).includes(g.id)}
                        <button
                          on:click={() => toggleGroup(selMovId, g.id)}
                          class="px-2 py-0.5 text-xs rounded-md font-medium border"
                          style="{inGroup
                            ? `background:${gColor(g.id)}; color:white; border-color:${gColor(g.id)}`
                            : `color:${gColor(g.id)}; border-color:${gColor(g.id)}55`}"
                        >{inGroup ? 'Remove' : 'Assign'}</button>
                      {/if}
                      {#if groups.length > 1}
                        <button
                          on:click={() => removeGroup(g.id)}
                          class="p-0.5 text-gray-400 hover:text-red-500 rounded
                                 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400"
                          title="Remove group" aria-label={`Remove group ${g.id}`}
                        >
                          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                              d="M6 18L18 6M6 6l12 12"/>
                          </svg>
                        </button>
                      {/if}
                    </div>
                  </div>

                  <div class="flex items-center gap-2">
                    <span class="text-xs text-gray-500 shrink-0">Green</span>
                    <input
                      type="number" min="1" max={MAX_GREEN} step="1" aria-label={`Green duration for group ${g.id}`}
                      value={g.greenDuration}
                      on:change={e => updGreen(g.id, e.target.value)}
                      class="w-16 px-2 py-1 text-xs border border-gray-200 rounded-md bg-white
                             focus:outline-none focus:ring-2 focus:ring-blue-400"
                    />
                    <span class="text-xs text-gray-400">s</span>
                  </div>

                  <div class="text-xs text-gray-500 leading-snug">
                    {#if assigned.length === 0}
                      <span class="italic text-gray-400">No movements assigned</span>
                    {:else}
                      {#each assigned as movement (movement.id)}
                        <button class="block w-full rounded px-1 py-0.5 text-left hover:bg-blue-50" on:click={() => { selMovId = movement.id; conflictSelection = []; }}>
                          {TURN_ICONS[turnType(movement)] || ''} {movement.inLabel} → {movement.outLabel}
                        </button>
                      {/each}
                    {/if}
                  </div>
                  {#if groupConflicts[g.id]?.length}
                    <p class="text-xs text-amber-600 flex items-center gap-1">
                      <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"/>
                      </svg>
                      {groupConflicts[g.id].length} possible conflicts
                    </p>
                    <div class="space-y-1">
                      {#each groupConflicts[g.id] as pair (`${pair[0].id}:${pair[1].id}`)}
                        <button class="block w-full rounded bg-amber-50 p-1.5 text-left text-xs text-amber-800 hover:bg-amber-100" on:click={() => selectConflict(pair)}>
                          {pair[0].inLabel} → {pair[0].outLabel} / {pair[1].inLabel} → {pair[1].outLabel}
                        </button>
                      {/each}
                    </div>
                  {:else if assigned.length}
                    <p class="text-xs text-green-600 flex items-center gap-1">
                      <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                      </svg>
                      No conflicts detected
                    </p>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Fixed bottom hint -->
        <div class="p-4 pt-2 shrink-0 border-t border-gray-100">
          <p class="text-xs text-gray-400 leading-snug">
            A group allows its movements together. A movement may belong to several groups. Dashed movements are forbidden. Conflict checks use approach geometry and shared exits.
          </p>
        </div>
      </div>
    </div>

    <div class="border-t border-gray-200 px-4 py-3 shrink-0">
      {#if invalidGreen || saveError}<p class="mb-2 text-sm text-red-600" role="alert">{saveError || `Green duration must be a whole number from 1 to ${MAX_GREEN} seconds.`}</p>{/if}
      {#if discard}
        <div class="flex flex-wrap items-center justify-end gap-2" role="alert">
          <p class="mr-auto text-sm text-amber-800">Discard unsaved intersection changes?</p>
          <button class="rounded border px-3 py-2 text-sm" on:click={() => discard = false}>Keep editing</button>
          <button class="rounded bg-red-600 px-3 py-2 text-sm text-white" on:click={() => dispatch('close')}>Discard changes</button>
        </div>
      {:else}
        <div class="flex flex-wrap items-center gap-2">
          <button class="rounded border px-3 py-2 text-sm disabled:opacity-40" disabled={!past.length} on:click={undoDraft}>Undo</button>
          <button class="rounded border px-3 py-2 text-sm disabled:opacity-40" disabled={!future.length} on:click={redoDraft}>Redo</button>
          <p class="mr-auto text-xs text-gray-500">{forbiddenCount} forbidden · {conflictCount} possible conflicts{dirty ? ' · Unsaved changes' : ''}</p>
          <button class="rounded bg-gray-100 px-3 py-2 text-sm" on:click={requestClose}>Cancel</button>
          <button class="rounded bg-blue-600 px-3 py-2 text-sm text-white disabled:opacity-40" disabled={invalidGreen} on:click={save}>Save configuration</button>
        </div>
      {/if}
    </div>
  </div>
</div>
{/if}
