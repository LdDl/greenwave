<script>
  import { createEventDispatcher, tick } from 'svelte';
  import { fade, scale } from 'svelte/transition';
  import { modalFocus } from '$lib/utils/modal-focus.js';
  import { validateImportedConfig } from '$lib/utils/export-import.js';
  import { intersectionTopology } from '$lib/utils/network-project.js';
  import { programGroupIds, programTimeline, programStateAt, signalColor, readProgram, addProgramGroup, removeProgramGroup } from '$lib/utils/junction-program.js';
  import IntersectionDiagram from './IntersectionDiagram.svelte';
  import SignalTimeline from './SignalTimeline.svelte';

  export let junction = null;
  export let isNew = false;
  export let corridorGroup = undefined;
  export let initialSignal = null;
  export let graph = { nodes: [], edges: [], error: '' };

  const dispatch = createEventDispatcher();

  // Local copy for editing
  let editedJunction = null;

  // Inline validation error  replaces native alert()
  let validationError = null;

  // Inline delete confirmation  replaces native confirm()
  let showDeleteConfirm = false;
  let activeTab = 'movements';
  let original = '';
  let discard = false;
  let selMovId = null;
  let previewTime = 0;
  let selectedSignalKey = null;
  let groupId = null;
  let selectedGroupId = null;
  let removingGroupId = null;
  let dialogElement;
  let discardButton;
  let returnFocus;

  $: groupIds = programGroupIds(editedJunction);
  $: topology = graph.error || isNew ? { stubs: [], movements: [] } : intersectionTopology(editedJunction?.id, graph.nodes, graph.edges);
  $: movementState = Object.fromEntries(topology.movements.map(movement => [movement.id, { groupIds: selectedGroupId === null ? [] : [selectedGroupId] }]));
  $: previews = groupIds.map(id => ({ id, ...readTimeline(editedJunction, id) }));
  $: preview = previews.find(preview => preview.id === groupId) ?? { timeline: { duration: 0, segments: [] }, error: '' };
  $: programError = validateProgram(editedJunction);
  $: signalState = programStateAt(preview.timeline, previewTime, editedJunction?.offset ?? 0);
  $: colors = Object.fromEntries(previews.map(preview => [preview.id, signalColor(programStateAt(preview.timeline, previewTime, editedJunction?.offset ?? 0)?.segment.color)]));
  $: selectedMovement = topology.movements.find(movement => movement.id === selMovId);
  $: dirty = editedJunction !== null && JSON.stringify({ junction: editedJunction, groupId: selectedGroupId }) !== original;

  function validateProgram(junction) {
    if (!junction) return '';
    try { readProgram(junction); return ''; }
    catch (cause) { return cause.message; }
  }

  function readTimeline(junction, id) {
    try { return { timeline: programTimeline(junction, id), error: '' }; }
    catch (cause) { return { timeline: { duration: 0, segments: [] }, error: cause.message }; }
  }

  async function requestClose() {
    if (!dirty) { dispatch('close'); return; }
    returnFocus = document.activeElement;
    discard = true;
    await tick();
    discardButton?.focus();
  }

  async function keepEditing() {
    discard = false;
    await tick();
    if (returnFocus?.isConnected) returnFocus.focus();
    else dialogElement?.focus();
  }

  function dialogKeys(event) {
    if (event.key !== 'Escape') return;
    event.stopPropagation();
    event.preventDefault();
    if (discard) keepEditing();
    else requestClose();
  }

  function beforeUnload(event) {
    if (dirty) { event.preventDefault(); event.returnValue = ''; }
  }

  async function selectSignal(event) {
    const segment = event.detail;
    groupId = segment.groupId;
    selectedSignalKey = `${segment.groupId}:${segment.phaseId}:${segment.signalIndex}`;
    const duration = previews.find(preview => preview.id === segment.groupId).timeline.duration;
    previewTime = ((segment.start + (editedJunction.offset ?? 0)) % duration + duration) % duration;
    activeTab = 'program';
    await tick();
    document.getElementById(`program-duration-${segment.phaseId}-${segment.signalIndex}`)?.focus();
  }


  $: if (junction) {
    const draft = { ...structuredClone(junction), offset: junction.offset ?? 0 };
    editedJunction = draft;
    const ids = programGroupIds(draft);
    const selected = corridorGroup === undefined ? (ids.length === 1 ? ids[0] : null) : corridorGroup;
    selectedGroupId = selected;
    groupId = selected ?? ids[0];
    original = JSON.stringify({ junction: draft, groupId: selected });
    removingGroupId = null;
    activeTab = isNew ? 'program' : 'movements';
    previewTime = 0;
    selMovId = null;
    selectedSignalKey = null;
    if (initialSignal) {
      groupId = initialSignal.groupId;
      selectedSignalKey = `${initialSignal.groupId}:${initialSignal.phaseId}:${initialSignal.signalIndex}`;
      activeTab = 'program';
      tick().then(() => document.getElementById(`program-duration-${initialSignal.phaseId}-${initialSignal.signalIndex}`)?.focus());
    }
    discard = false;
    validationError = null;

    showDeleteConfirm = false;
  }

  function handleBackdropClick(event) {
    if (event.target === event.currentTarget) {
      requestClose();
    }
  }

  function saveJunction() {
    if (!editedJunction) return;
    validationError = null;

    const validation = validateImportedConfig({ junctions: [editedJunction], desiredSpeed: 40 });
    if (!validation.isValid) {
      validationError = validation.errors.join(' ');
      return;
    }

    if (editedJunction.cycle.length === 0) {
      validationError = 'Junction must have at least one phase.';
      return;
    }

    for (const phase of editedJunction.cycle) {
      if (phase.signal_groups[0].signals.length === 0) {
        validationError = 'Each phase must have at least one signal.';
        return;
      }
    }

    if (programError) { validationError = programError; return; }
    dispatch('save', { junction: editedJunction, isNew, groupId: selectedGroupId });
  }

  function confirmDelete() {
    dispatch('delete', { junction: editedJunction });
  }

  function addPhase() {
    validationError = null;
    const newPhaseId = Math.max(0, ...editedJunction.cycle.map(p => p.id)) + 1;
    editedJunction.cycle = [
      ...editedJunction.cycle,
      {
        id: newPhaseId,
        signal_groups: groupIds.map(id => ({ id, signals: id === groupId
          ? [{ duration: 30, color: 'GREEN' }, { duration: 20, color: 'RED' }]
          : [{ duration: 50, color: 'RED' }] }))
      }
    ];
  }

  function removePhase(phaseIndex) {
    if (editedJunction.cycle.length <= 1) {
      validationError = 'Junction must have at least one phase.';
      return;
    }
    validationError = null;
    editedJunction.cycle = editedJunction.cycle.filter((_, i) => i !== phaseIndex);
  }

  function addSignal(phaseIndex) {
    validationError = null;
    const sg = editedJunction.cycle[phaseIndex].signal_groups.find(group => group.id === groupId);
    sg.signals = [...sg.signals, { duration: 10, color: 'GREEN' }];
    editedJunction.cycle = [...editedJunction.cycle];
  }

  function removeSignal(phaseIndex, signalIndex) {
    const sg = editedJunction.cycle[phaseIndex].signal_groups.find(group => group.id === groupId);
    if (sg.signals.length <= 1) {
      validationError = 'Phase must have at least one signal.';
      return;
    }
    validationError = null;
    sg.signals = sg.signals.filter((_, i) => i !== signalIndex);
    editedJunction.cycle = [...editedJunction.cycle];
  }
  function addGroup() {
    try {
      const result = addProgramGroup(editedJunction);
      editedJunction = result.junction;
      groupId = result.groupId;
      validationError = null;
    } catch (cause) { validationError = cause.message; }
  }

  function removeGroup() {
    try {
      editedJunction = removeProgramGroup(editedJunction, removingGroupId, selectedGroupId);
      if (groupId === removingGroupId) groupId = programGroupIds(editedJunction)[0];
      removingGroupId = null;
      validationError = null;
    } catch (cause) { validationError = cause.message; }
  }

</script>

<svelte:window on:beforeunload={beforeUnload} />

{#if editedJunction}
  <div
    bind:this={dialogElement}
    transition:fade={{ duration: 150 }}
    class="fixed inset-0 flex items-center justify-center z-50"
    style="background-color: rgba(0, 0, 0, 0.6);"
    on:click={handleBackdropClick}
    on:keydown={dialogKeys}
    on:input={() => validationError = null}
    role="dialog"
    aria-modal="true"
    aria-labelledby="junction-editor-title"
    tabindex="-1"
    use:modalFocus
  >
    <div
      transition:scale={{ start: 0.96, duration: 150 }}
      class="bg-white rounded-xl shadow-xl max-w-6xl w-full mx-2 sm:mx-4 h-[92dvh] flex flex-col overflow-hidden"
    >
      <h3 id="junction-editor-title" class="px-4 py-3 text-lg font-medium border-b border-gray-200 shrink-0">
        {isNew ? 'Create New Junction' : `Edit ${editedJunction.label}`}
      </h3>

      <!-- Inline validation error. So no native alert() -->
      {#if validationError}
        <div class="mb-4 px-3 py-2 bg-red-50 border border-red-300 rounded-md text-sm text-red-700">
          {validationError}
        </div>
      {/if}

      <div class="flex-1 min-h-0 overflow-y-auto lg:flex lg:overflow-hidden">
        <section class="flex min-w-0 flex-col lg:flex-1 lg:min-h-0" aria-label="Junction diagram and timeline">
          <div class="flex h-[300px] shrink-0 lg:h-auto lg:flex-1 lg:min-h-[240px]">
            <IntersectionDiagram stubs={topology.stubs} movements={topology.movements} movState={movementState} bind:selMovId groupColors={colors}
              emptyMessage={isNew ? 'Save the new junction to connect it to the corridor.' : graph.error || 'No roads connected to this junction.'}
              on:select={() => activeTab = 'movements'} />
          </div>
          <div class="shrink-0 border-t border-gray-200 bg-white p-4 space-y-3">
            <div class="max-h-[220px] overflow-y-auto space-y-3">
              {#each previews as row (row.id)}
                {#if row.error}<p class="text-sm text-red-700" role="status">G{row.id}: {row.error}</p>
                {:else}<SignalTimeline timeline={row.timeline} time={previewTime} offset={editedJunction.offset ?? 0} groupId={row.id} on:select={selectSignal} />{/if}
              {/each}
            </div>
            {#if programError}<p class="text-sm text-amber-800" role="status">{programError} Adjust the groups before saving.</p>{/if}
            <label for="junction-preview-time" class="block text-xs text-gray-600">Signal preview at corridor time {Number(previewTime).toFixed(1)} s: <strong>{signalState?.segment.color ?? 'No signal'}</strong></label>
            <input id="junction-preview-time" type="range" min="0" max={preview.timeline.duration || 1} step="0.1" bind:value={previewTime} class="w-full" />
            <p class="text-xs text-gray-500">Click a colored interval to edit it. Offset: {editedJunction.offset ?? 0} s. Preview does not change the saved program.</p>
          </div>
        </section>
        <section class="border-t border-gray-200 lg:border-t-0 lg:border-l lg:w-[28rem] lg:shrink-0 lg:flex lg:flex-col lg:min-h-0" aria-label="Junction settings">
          <div class="flex flex-wrap gap-1 border-b border-gray-200 bg-gray-50 p-2" role="group" aria-label="Junction sections">
            {#each [{ id: 'movements', label: 'Movements' }, { id: 'groups', label: 'Groups' }, { id: 'program', label: 'Program' }] as tab (tab.id)}
              <button type="button" aria-pressed={activeTab === tab.id} class="rounded px-4 py-2 text-sm focus-visible:outline-2 focus-visible:outline-blue-500" class:bg-blue-600={activeTab === tab.id} class:text-white={activeTab === tab.id} on:click={() => activeTab = tab.id}>{tab.label}</button>
            {/each}
          </div>
          <div class="p-4 lg:overflow-y-auto lg:flex-1">
            {#if activeTab === 'movements'}
              <h4 class="mb-3 font-semibold">Movements ({topology.movements.length})</h4>
              {#if graph.error}
                <p class="text-sm text-amber-800">{graph.error} Your signal program remains available in "Program".</p>
              {:else if isNew}
                <p class="text-sm text-gray-500">Set the program, then create the junction to see its connected roads.</p>
              {:else if !topology.movements.length}
                <p class="text-sm text-gray-500">{topology.stubs.length === 1 ? 'This is an end of the corridor. The external approach is not drawn, so there is no complete through movement here.' : 'Connect roads to see through movements.'} The signal group and its program can still be edited.</p>
              {:else}
                <p class="mb-3 text-sm text-gray-500">Select an arc or a movement below. Through movements use the corridor group in both directions. Choose it in "Groups".</p>
                <div class="space-y-2">
                  {#each topology.movements as movement (movement.id)}
                    <button class="block w-full rounded border px-3 py-2 text-left text-sm hover:bg-blue-50" class:border-blue-600={selMovId === movement.id} class:bg-blue-50={selMovId === movement.id} aria-pressed={selMovId === movement.id} on:click={() => selMovId = movement.id}>{movement.inLabel} → {movement.outLabel} · {selectedGroupId === null ? 'Choose group' : `G${selectedGroupId}`}</button>
                  {/each}
                </div>
                {#if selectedMovement}<p class="mt-4 text-sm text-gray-600">Selected: {selectedMovement.inLabel} → {selectedMovement.outLabel}. Corridor group: {selectedGroupId === null ? 'Not selected' : `G${selectedGroupId}`}.</p>{/if}
              {/if}
              <button class="mt-4 rounded bg-blue-600 px-3 py-2 text-sm text-white" on:click={() => activeTab = 'program'}>Edit G{groupId} program</button>
            {/if}
            {#if activeTab === 'groups'}
              <div class="flex items-center justify-between gap-2 mb-3">
                <h4 class="font-semibold">Signal groups ({groupIds.length})</h4>
                <button class="rounded bg-green-600 px-3 py-2 text-sm text-white" on:click={addGroup}>+ Add group</button>
              </div>
              <p class="text-sm text-gray-600">Groups persist across every phase and share its duration. A new group starts red for the full program.</p>
              <label for="corridor-group" class="mt-4 mb-1 block text-sm font-medium">Corridor group</label>
              <select id="corridor-group" bind:value={selectedGroupId} class="w-full rounded border p-2 pr-8">
                <option value={null}>Choose a group</option>
                {#each groupIds as id (id)}<option value={id}>G{id}</option>{/each}
              </select>
              <p class="mt-1 mb-4 text-xs text-gray-500">Used by the diagram, calculations and through movements in both directions.</p>
              <div class="space-y-2">
                {#each groupIds as id (id)}
                  <div class="rounded border p-3" class:border-blue-500={groupId === id}>
                    <div class="flex items-center justify-between gap-2">
                      <button class="font-semibold text-blue-700" on:click={() => { groupId = id; activeTab = 'program'; }}>Edit G{id} program</button>
                      <button aria-label={`Remove group G${id}`} class="text-sm text-red-600 disabled:opacity-40" disabled={groupIds.length <= 1 || selectedGroupId === id} title={selectedGroupId === id ? 'Choose another corridor group before removing this group' : 'Remove this group from every phase'} on:click={() => removingGroupId = id}>Remove</button>
                    </div>
                    {#if selectedGroupId === id}<p class="mt-1 text-xs text-gray-500">Corridor group</p>{/if}
                  </div>
                {/each}
              </div>
              {#if removingGroupId !== null}
                <div class="mt-3 rounded border border-red-200 bg-red-50 p-3 text-sm">
                  <p>Remove G{removingGroupId} and its signals from every phase?</p>
                  <div class="mt-2 flex gap-3"><button class="text-red-700" on:click={removeGroup}>Remove group</button><button on:click={() => removingGroupId = null}>Keep group</button></div>
                </div>
              {/if}
            {/if}
            <div hidden={activeTab !== 'program'}>
              <label for="program-group" class="mb-1 block text-sm font-medium">Edit signal group</label>
              <select id="program-group" bind:value={groupId} class="mb-3 w-full rounded border p-2 pr-8">
                {#each groupIds as id (id)}<option value={id}>G{id}</option>{/each}
              </select>
              <p class="mb-4 text-sm text-gray-500">Program for G{groupId}. Changes appear in both the diagram and the network after Save.</p>
              <div class="space-y-4 mb-6">
                <div class="flex items-center gap-4">
                  <label for="junction-label" class="text-sm font-medium w-32 shrink-0">Label:</label>
                  <input
                    id="junction-label"
                    type="text"
                    bind:value={editedJunction.label}
                    class="flex-1 px-3 py-2 border rounded-md min-w-0"
                    placeholder="Junction name"
                  />
                </div>

                <div class="flex items-center gap-4">
                  <label for="junction-distance" class="text-sm font-medium w-32 shrink-0">Distance (m):</label>
                  <input
                    id="junction-distance"
                    type="number"
                    bind:value={editedJunction.point.y}
                    class="flex-1 px-3 py-2 border rounded-md min-w-0"
                    min="0"
                    step="10"
                    placeholder="Distance from start"
                  />
                </div>
                <div class="flex items-center gap-4">
                  <label for="junction-offset" class="text-sm font-medium w-32 shrink-0">Offset (s):</label>
                  <input id="junction-offset" type="number" bind:value={editedJunction.offset} class="flex-1 px-3 py-2 border rounded-md min-w-0" step="1" />
                </div>
              </div>

              <!-- Phases section -->
              <div class="border-t pt-4">
                <div class="flex justify-between items-center mb-4">
                  <h4 class="text-md font-medium">Phases ({editedJunction.cycle.length})</h4>
                  <button
                    class="px-3 py-1.5 bg-green-500 text-white text-sm rounded-md hover:bg-green-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green-400"
                    on:click={addPhase}
                  >
                    + Add Phase
                  </button>
                </div>

                <div class="space-y-4">
                  {#each editedJunction.cycle as phase, phaseIndex (phase.id)}
                    {@const phaseGroup = phase.signal_groups.find(group => group.id === groupId)}
                    <div class="border rounded-md p-4 bg-gray-50">
                      <div class="flex justify-between items-center mb-3">
                        <h5 class="text-sm font-medium">Phase {phaseIndex + 1}</h5>
                        <button
                          class="px-2 py-1 bg-red-100 text-red-600 text-xs rounded-md hover:bg-red-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400"
                          on:click={() => removePhase(phaseIndex)}
                          title="Remove phase"
                        >
                          Remove
                        </button>
                      </div>

                      <p class="mb-2 text-xs text-gray-500">{phase.signal_groups.map(group => `G${group.id}: ${group.signals.reduce((sum, signal) => sum + (Number(signal.duration) || 0), 0)} s`).join(' · ')}</p>
                      {#if !phaseGroup}<p class="text-sm text-amber-800">G{groupId} is missing in this phase.</p>{/if}
                      <div class="space-y-2">
                        {#each phaseGroup?.signals ?? [] as signal, signalIndex (signalIndex)}
                          <!-- flex-wrap so rows don't overflow on narrow modals (phones) -->
                          <div class="flex items-center gap-2 flex-wrap rounded p-1" class:bg-blue-100={selectedSignalKey === `${groupId}:${phase.id}:${signalIndex}`}>
                            <span class="text-xs text-gray-500 w-14 shrink-0">Signal {signalIndex + 1}</span>
                            <select
                              aria-label={`Phase ${phaseIndex + 1} signal ${signalIndex + 1} color`}
                              bind:value={signal.color}
                              class="pl-2 pr-8 py-1.5 border rounded-md text-sm min-w-0"
                            >
                              <option value="GREEN">Green</option>
                              <option value="RED">Red</option>
                              <option value="YELLOW">Yellow</option>
                            </select>
                            <input
                              type="number"
                              id={`program-duration-${phase.id}-${signalIndex}`}
                              aria-label={`Phase ${phaseIndex + 1} signal ${signalIndex + 1} duration`}
                              bind:value={signal.duration}
                              class="w-16 px-2 py-1.5 border rounded-md text-sm"
                              min="1"
                              placeholder="sec"
                            />
                            <span class="text-xs text-gray-400">s</span>
                            <button
                              class="px-2 py-1 text-red-500 hover:text-red-700 text-xs rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400"
                              on:click={() => removeSignal(phaseIndex, signalIndex)}
                              title="Remove signal"
                            >
                              ✕
                            </button>
                          </div>
                        {/each}
                      </div>

                      <button
                        class="mt-2 px-2 py-1.5 bg-gray-200 text-xs rounded-md hover:bg-gray-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
                        disabled={!phaseGroup}
                        on:click={() => addSignal(phaseIndex)}
                      >
                        + Add Signal
                      </button>
                    </div>
                  {/each}
                </div>
              </div>

            </div>
          </div>
        </section>
      </div>

      <!-- Action buttons -->
      <div class="flex flex-wrap items-center justify-between gap-3 p-4 border-t shrink-0">
        {#if discard}
          <p class="text-sm text-gray-600">Discard unsaved junction changes?</p>
          <div class="flex gap-2"><button bind:this={discardButton} class="rounded border px-3 py-2 text-sm" on:click={keepEditing}>Keep editing</button><button class="rounded bg-red-600 px-3 py-2 text-sm text-white" on:click={() => dispatch('close')}>Discard changes</button></div>
        {:else}
        <div>
          {#if !isNew}
            {#if showDeleteConfirm}
              <!-- Inline delete confirmation. Just replaces native confirm() (same, as alert())-->
              <div class="flex items-center gap-2">
                <span class="text-sm text-red-600">Delete "{editedJunction.label}"?</span>
                <button
                  class="px-3 py-1.5 bg-red-500 text-white text-sm rounded-md hover:bg-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400"
                  on:click={confirmDelete}
                >
                  Yes, delete
                </button>
                <button
                  class="px-3 py-1.5 bg-gray-200 text-gray-700 text-sm rounded-md hover:bg-gray-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
                  on:click={() => showDeleteConfirm = false}
                >
                  Cancel
                </button>
              </div>
            {:else}
              <button
                class="px-4 py-2 bg-red-500 text-white rounded-md hover:bg-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400"
                on:click={() => showDeleteConfirm = true}
              >
                Delete Junction
              </button>
            {/if}
          {/if}
        </div>
        <div class="flex flex-wrap items-center gap-2">
          {#if dirty}<span class="text-xs text-gray-500">Unsaved changes</span>{/if}
          <button
            class="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
            on:click={requestClose}
          >
            Cancel
          </button>
          <button
            class="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
            on:click={saveJunction}
          >
            {isNew ? 'Create' : 'Save'}
          </button>
        </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
