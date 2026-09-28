<script>
  import { createEventDispatcher } from 'svelte';
  import { fade, scale } from 'svelte/transition';

  import { MAX_LANES } from '$lib/utils/network-project.js';
  import { modalFocus } from '$lib/utils/modal-focus.js';

  export let fromLabel = '';
  export let toLabel = '';

  const dispatch = createEventDispatcher();

  let twoWay = true;
  let lanesFwd = 1;
  let lanesBack = 1;

  $: valid = Number.isInteger(lanesFwd) && lanesFwd >= 1 && lanesFwd <= MAX_LANES &&
    (!twoWay || (Number.isInteger(lanesBack) && lanesBack >= 1 && lanesBack <= MAX_LANES));

  function save() {
    if (!valid) return;
    dispatch('save', { lanesFwd, lanesBack: twoWay ? lanesBack : 0 });
  }

  function handleBackdrop(e) {
    if (e.target === e.currentTarget) dispatch('close');
  }
</script>

<div
  use:modalFocus
  aria-labelledby="road-modal-title"
  transition:fade={{ duration: 150 }}
  class="fixed inset-0 flex items-center justify-center z-50"
  style="background-color: rgba(0,0,0,0.5)"
  on:click={handleBackdrop}
  on:keydown={e => { if (e.key === 'Escape') { e.stopPropagation(); dispatch('close'); } }}
  role="dialog"
  aria-modal="true"
  tabindex="-1"
>
  <form on:submit|preventDefault={save}
    transition:scale={{ start: 0.96, duration: 150 }}
    class="bg-white rounded-lg shadow-xl w-80 mx-4 p-6"
  >
    <h3 id="road-modal-title" class="text-base font-semibold mb-3">New Road Segment</h3>
    <div class="flex items-center gap-2 mb-5">
      <span class="px-2 py-0.5 rounded-md text-sm font-medium bg-green-50 text-green-700 border border-green-200">{fromLabel}</span>
      <span class="text-gray-400 text-sm">→</span>
      <span class="px-2 py-0.5 rounded-md text-sm font-medium bg-red-50 text-red-600 border border-red-200">{toLabel}</span>
    </div>

    <!-- Direction toggle -->
    <p class="text-xs font-medium text-gray-500 mb-1.5">Direction</p>
    <div class="inline-flex w-full bg-gray-100 rounded-lg p-0.5 gap-0.5 text-sm mb-5">
      <button
        type="button"
        class="flex-1 py-1.5 rounded-md font-medium transition-all"
        class:bg-white={!twoWay}
        class:shadow-sm={!twoWay}
        class:text-gray-900={!twoWay}
        class:text-gray-500={twoWay}
        on:click={() => twoWay = false}
      >One-way →</button>
      <button
        type="button"
        class="flex-1 py-1.5 rounded-md font-medium transition-all"
        class:bg-white={twoWay}
        class:shadow-sm={twoWay}
        class:text-gray-900={twoWay}
        class:text-gray-500={!twoWay}
        on:click={() => twoWay = true}
      >Two-way ↔</button>
    </div>

    <!-- Lane counts -->
    <div class="space-y-3">
      <div>
        <label for="rse-lanes-fwd" class="text-xs text-gray-500 block mb-1">
          Lanes <span class="text-green-600 font-semibold">→</span>
          &nbsp;<span class="text-gray-400">({fromLabel} → {toLabel})</span>
        </label>
        <input
          id="rse-lanes-fwd"
          type="number"
          min="1" max={MAX_LANES} step="1"
          bind:value={lanesFwd}
          class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
        />
      </div>

      {#if twoWay}
        <div>
          <label for="rse-lanes-back" class="text-xs text-gray-500 block mb-1">
            Lanes <span class="text-red-500 font-semibold">←</span>
            &nbsp;<span class="text-gray-400">({toLabel} → {fromLabel})</span>
          </label>
          <input
            id="rse-lanes-back"
            type="number"
            min="1" max={MAX_LANES} step="1"
            bind:value={lanesBack}
            class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
          />
        </div>
      {/if}
    </div>

    {#if !valid}<p class="mt-3 text-xs text-red-600" role="alert">Lane counts must be whole numbers from 1 to {MAX_LANES}.</p>{/if}
    <div class="flex justify-end gap-2 mt-6">
      <button
        type="button"
        class="px-4 py-2 bg-gray-200 text-gray-700 rounded-md text-sm hover:bg-gray-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
        on:click={() => dispatch('close')}
      >Cancel</button>
      <button
        type="submit"
        class="px-4 py-2 bg-blue-500 text-white rounded-md text-sm hover:bg-blue-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
        disabled={!valid}
      >Add road</button>
    </div>
  </form>
</div>
