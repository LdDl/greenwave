<script>
  import { createEventDispatcher } from 'svelte';

  export let signal;
  const dispatch = createEventDispatcher();

  function saveSignal() {
    dispatch('save', { signal }); // Emit the updated signal
  }

  function handleBackdropClick(event) {
    // Close the modal when clicking outside
    if (event.target === event.currentTarget) {
      dispatch('close');
    }
  }
</script>

{#if signal}
  <!-- Modal backdrop -->
  <div 
    class="fixed inset-0 flex items-center justify-center z-50"
    style="background-color: rgba(0, 0, 0, 0.75);"
    on:click={handleBackdropClick}
    role="dialog"
    aria-modal="true"
  >
    <!-- Modal content -->
    <div class="bg-white rounded-lg shadow-lg max-w-md w-full mx-4 p-6">
      <h3 class="text-lg font-medium mb-4">Edit Signal</h3>
      <div class="space-y-4">
        <!-- Color input -->
        <div class="flex items-center gap-4">
          <label class="text-sm font-medium w-24">Color:</label>
          <select bind:value={signal.color} class="flex-1 px-3 py-2 border rounded-md">
            <option value="GREEN">Green</option>
            <option value="RED">Red</option>
            <option value="YELLOW">Yellow</option>
          </select>
        </div>

        <!-- Duration input -->
        <div class="flex items-center gap-4">
          <label class="text-sm font-medium w-24">Duration (s):</label>
          <input 
            type="number" 
            bind:value={signal.duration} 
            class="flex-1 px-3 py-2 border rounded-md"
            min="1"
          />
        </div>
      </div>

      <div class="flex justify-between mt-6">
        <button 
          class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
          on:click={saveSignal}
        >
          Save
        </button>
        <button 
          class="px-4 py-2 bg-gray-300 text-gray-700 rounded hover:bg-gray-400"
          on:click={() => dispatch('close')}
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    background: white;
    padding: 20px;
    border-radius: 8px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    z-index: 1000;
    width: 400px;
    max-width: 90%;
  }
</style>