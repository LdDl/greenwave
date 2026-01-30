<script>
  import { createEventDispatcher } from 'svelte';

  export let label = 'Menu';
  export let items = []; // { id, label, disabled?, danger? }

  const dispatch = createEventDispatcher();

  let isOpen = false;

  function toggle() {
    isOpen = !isOpen;
  }

  function close() {
    isOpen = false;
  }

  function handleSelect(item) {
    if (item.disabled) return;
    dispatch('select', item);
    close();
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') close();
  }

  function handleClickOutside(e) {
    if (isOpen) close();
  }
</script>

<svelte:window on:click={handleClickOutside} on:keydown={handleKeydown} />

<div class="relative inline-block">
  <button
    type="button"
    on:click|stopPropagation={toggle}
    class="px-3 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300 text-sm flex items-center gap-1"
    aria-expanded={isOpen}
    aria-haspopup="menu"
  >
    {label}
    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
    </svg>
  </button>

  {#if isOpen}
    <div
      class="absolute right-0 mt-1 w-48 bg-white border border-gray-200 rounded-md shadow-lg z-50"
      role="menu"
      tabindex="-1"
      on:click|stopPropagation={() => {}}
      on:keydown|stopPropagation={(e) => e.key === 'Escape' && close()}
    >
      {#each items as item}
        {#if item.separator}
          <div class="border-t border-gray-200 my-1" role="separator"></div>
        {:else}
          <button
            type="button"
            role="menuitem"
            class="w-full text-left px-4 py-2 text-sm hover:bg-gray-100 disabled:text-gray-400 disabled:cursor-not-allowed"
            class:text-red-600={item.danger}
            class:hover:bg-red-50={item.danger}
            disabled={item.disabled}
            on:click={() => handleSelect(item)}
          >
            {item.label}
          </button>
        {/if}
      {/each}
    </div>
  {/if}
</div>
