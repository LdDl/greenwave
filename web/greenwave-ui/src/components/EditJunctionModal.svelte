<script>
  import { createEventDispatcher } from 'svelte';

  export let junction = null;
  export let isNew = false;

  const dispatch = createEventDispatcher();

  // Local copy for editing
  let editedJunction = null;

  $: if (junction) {
    // Deep clone the junction to avoid mutating the original
    editedJunction = JSON.parse(JSON.stringify(junction));
  }

  function handleBackdropClick(event) {
    if (event.target === event.currentTarget) {
      dispatch('close');
    }
  }

  function saveJunction() {
    if (!editedJunction) return;

    // Validate: at least one phase with at least one signal
    if (editedJunction.cycle.length === 0) {
      alert('Junction must have at least one phase');
      return;
    }

    for (const phase of editedJunction.cycle) {
      if (phase.signals.length === 0) {
        alert('Each phase must have at least one signal');
        return;
      }
    }

    dispatch('save', { junction: editedJunction, isNew });
  }

  function deleteJunction() {
    if (confirm(`Are you sure you want to delete "${editedJunction.label}"?`)) {
      dispatch('delete', { junction: editedJunction });
    }
  }

  function addPhase() {
    const newPhaseId = Math.max(0, ...editedJunction.cycle.map(p => p.id)) + 1;
    editedJunction.cycle = [
      ...editedJunction.cycle,
      {
        id: newPhaseId,
        signals: [
          { duration: 30, color: 'GREEN' },
          { duration: 20, color: 'RED' }
        ]
      }
    ];
  }

  function removePhase(phaseIndex) {
    if (editedJunction.cycle.length <= 1) {
      alert('Junction must have at least one phase');
      return;
    }
    editedJunction.cycle = editedJunction.cycle.filter((_, i) => i !== phaseIndex);
  }

  function addSignal(phaseIndex) {
    editedJunction.cycle[phaseIndex].signals = [
      ...editedJunction.cycle[phaseIndex].signals,
      { duration: 10, color: 'GREEN' }
    ];
  }

  function removeSignal(phaseIndex, signalIndex) {
    if (editedJunction.cycle[phaseIndex].signals.length <= 1) {
      alert('Phase must have at least one signal');
      return;
    }
    editedJunction.cycle[phaseIndex].signals = editedJunction.cycle[phaseIndex].signals.filter((_, i) => i !== signalIndex);
  }
</script>

{#if editedJunction}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="fixed inset-0 flex items-center justify-center z-50"
    style="background-color: rgba(0, 0, 0, 0.75);"
    on:click={handleBackdropClick}
    on:keydown={(e) => e.key === 'Escape' && dispatch('close')}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div class="bg-white rounded-lg shadow-lg max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto">
      <h3 class="text-lg font-medium mb-4">
        {isNew ? 'Create New Junction' : `Edit ${editedJunction.label}`}
      </h3>

      <!-- Junction basic info -->
      <div class="space-y-4 mb-6">
        <div class="flex items-center gap-4">
          <label for="junction-label" class="text-sm font-medium w-32">Label:</label>
          <input
            id="junction-label"
            type="text"
            bind:value={editedJunction.label}
            class="flex-1 px-3 py-2 border rounded-md"
            placeholder="Junction name"
          />
        </div>

        <div class="flex items-center gap-4">
          <label for="junction-distance" class="text-sm font-medium w-32">Distance (m):</label>
          <input
            id="junction-distance"
            type="number"
            bind:value={editedJunction.point.y}
            class="flex-1 px-3 py-2 border rounded-md"
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
            class="px-3 py-1 bg-green-500 text-white text-sm rounded hover:bg-green-600"
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
                  class="px-2 py-1 bg-red-100 text-red-600 text-xs rounded hover:bg-red-200"
                  on:click={() => removePhase(phaseIndex)}
                  title="Remove phase"
                >
                  Remove
                </button>
              </div>

              <div class="space-y-2">
                {#each phase.signals as signal, signalIndex}
                  <div class="flex items-center gap-2">
                    <span class="text-xs text-gray-500 w-16">Signal {signalIndex + 1}</span>
                    <select
                      bind:value={signal.color}
                      class="px-2 py-1 border rounded-md text-sm"
                    >
                      <option value="GREEN">Green</option>
                      <option value="RED">Red</option>
                      <option value="YELLOW">Yellow</option>
                    </select>
                    <input
                      type="number"
                      bind:value={signal.duration}
                      class="w-20 px-2 py-1 border rounded-md text-sm"
                      min="1"
                      placeholder="sec"
                    />
                    <span class="text-xs text-gray-400">sec</span>
                    <button
                      class="px-2 py-1 text-red-500 hover:text-red-700 text-xs"
                      on:click={() => removeSignal(phaseIndex, signalIndex)}
                      title="Remove signal"
                    >
                      ✕
                    </button>
                  </div>
                {/each}
              </div>

              <button
                class="mt-2 px-2 py-1 bg-gray-200 text-xs rounded hover:bg-gray-300"
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
            <button
              class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600"
              on:click={deleteJunction}
            >
              Delete Junction
            </button>
          {/if}
        </div>
        <div class="flex gap-2">
          <button
            class="px-4 py-2 bg-gray-300 text-gray-700 rounded hover:bg-gray-400"
            on:click={() => dispatch('close')}
          >
            Cancel
          </button>
          <button
            class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
            on:click={saveJunction}
          >
            {isNew ? 'Create' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
