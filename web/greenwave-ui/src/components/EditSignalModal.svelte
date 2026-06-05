<script>
  import { createEventDispatcher } from 'svelte';
  import { fade, scale } from 'svelte/transition';

  export let signal;
  const dispatch = createEventDispatcher();

  function saveSignal() {
    dispatch('save', { signal });
  }

  function handleBackdropClick(event) {
    if (event.target === event.currentTarget) {
      dispatch('close');
    }
  }
</script>

{#if signal}
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
      class="bg-white rounded-lg shadow-lg max-w-md w-full mx-4 p-6"
    >
      <h3 class="text-lg font-medium mb-4">Edit Signal</h3>
      <div class="space-y-4">
        <div class="flex items-center gap-4">
          <label for="signal-color" class="text-sm font-medium w-24 shrink-0">Color:</label>
          <select id="signal-color" bind:value={signal.color} class="flex-1 px-3 py-2 border rounded-md min-w-0">
            <option value="GREEN">Green</option>
            <option value="RED">Red</option>
            <option value="YELLOW">Yellow</option>
          </select>
        </div>

        <div class="flex items-center gap-4">
          <label for="signal-duration" class="text-sm font-medium w-24 shrink-0">Duration (s):</label>
          <input
            id="signal-duration"
            type="number"
            bind:value={signal.duration}
            class="flex-1 px-3 py-2 border rounded-md min-w-0"
            min="1"
          />
        </div>
      </div>

      <div class="flex justify-between mt-6">
        <button
          class="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
          on:click={saveSignal}
        >
          Save
        </button>
        <button
          class="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
          on:click={() => dispatch('close')}
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
{/if}
