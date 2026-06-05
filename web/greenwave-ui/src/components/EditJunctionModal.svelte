<script>
  import { createEventDispatcher } from 'svelte';
  import { fade, scale } from 'svelte/transition';

  export let junction = null;
  export let isNew = false;

  const dispatch = createEventDispatcher();

  // Local copy for editing
  let editedJunction = null;

  // Inline validation error  replaces native alert()
  let validationError = null;

  // Inline delete confirmation  replaces native confirm()
  let showDeleteConfirm = false;

  $: if (junction) {
    editedJunction = JSON.parse(JSON.stringify(junction));
    validationError = null;
    showDeleteConfirm = false;
  }

  function handleBackdropClick(event) {
    if (event.target === event.currentTarget) {
      dispatch('close');
    }
  }

  function saveJunction() {
    if (!editedJunction) return;
    validationError = null;

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

    dispatch('save', { junction: editedJunction, isNew });
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
        signal_groups: [{ id: 0, signals: [
          { duration: 30, color: 'GREEN' },
          { duration: 20, color: 'RED' }
        ]}]
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
    const sg = editedJunction.cycle[phaseIndex].signal_groups[0];
    sg.signals = [...sg.signals, { duration: 10, color: 'GREEN' }];
    editedJunction.cycle = [...editedJunction.cycle];
  }

  function removeSignal(phaseIndex, signalIndex) {
    const sg = editedJunction.cycle[phaseIndex].signal_groups[0];
    if (sg.signals.length <= 1) {
      validationError = 'Phase must have at least one signal.';
      return;
    }
    validationError = null;
    sg.signals = sg.signals.filter((_, i) => i !== signalIndex);
    editedJunction.cycle = [...editedJunction.cycle];
  }
</script>

{#if editedJunction}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    transition:fade={{ duration: 150 }}
    class="fixed inset-0 flex items-center justify-center z-50"
    style="background-color: rgba(0, 0, 0, 0.6);"
    on:click={handleBackdropClick}
    on:keydown={(e) => e.key === 'Escape' && dispatch('close')}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div
      transition:scale={{ start: 0.96, duration: 150 }}
      class="bg-white rounded-lg shadow-lg max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto"
    >
      <h3 class="text-lg font-medium mb-4">
        {isNew ? 'Create New Junction' : `Edit ${editedJunction.label}`}
      </h3>

      <!-- Inline validation error. So no native alert() -->
      {#if validationError}
        <div class="mb-4 px-3 py-2 bg-red-50 border border-red-300 rounded-md text-sm text-red-700">
          {validationError}
        </div>
      {/if}

      <!-- Junction basic info -->
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
          {#each editedJunction.cycle as phase, phaseIndex}
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

              <div class="space-y-2">
                {#each phase.signal_groups[0].signals as signal, signalIndex}
                  <!-- flex-wrap so rows don't overflow on narrow modals (phones) -->
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="text-xs text-gray-500 w-14 shrink-0">Signal {signalIndex + 1}</span>
                    <select
                      bind:value={signal.color}
                      class="px-2 py-1.5 border rounded-md text-sm min-w-0"
                    >
                      <option value="GREEN">Green</option>
                      <option value="RED">Red</option>
                      <option value="YELLOW">Yellow</option>
                    </select>
                    <input
                      type="number"
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
                on:click={() => addSignal(phaseIndex)}
              >
                + Add Signal
              </button>
            </div>
          {/each}
        </div>
      </div>

      <!-- Action buttons -->
      <div class="flex justify-between mt-6 pt-4 border-t">
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
        <div class="flex gap-2">
          <button
            class="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
            on:click={() => dispatch('close')}
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
      </div>
    </div>
  </div>
{/if}
