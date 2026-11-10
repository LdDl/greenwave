<script>
	import { createEventDispatcher } from 'svelte';

	export let label = 'Menu';
	// { id, label, disabled?, danger?, separator? }
	export let items = [];

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

	function handleClickOutside() {
		if (isOpen) close();
	}
</script>

<svelte:window on:click={handleClickOutside} on:keydown={handleKeydown} />

<div class="relative inline-block">
	<button
		type="button"
		on:click|stopPropagation={toggle}
		class="flex items-center gap-1 rounded bg-gray-200 px-3 py-2 text-sm text-gray-700 hover:bg-gray-300"
		aria-expanded={isOpen}
		aria-haspopup="menu"
	>
		{label}
		<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	{#if isOpen}
		<div
			class="absolute right-0 z-40 mt-1 w-48 rounded-md border border-gray-200 bg-white shadow-lg"
			role="menu"
			tabindex="-1"
			on:click|stopPropagation={() => {}}
			on:keydown|stopPropagation={(e) => e.key === 'Escape' && close()}
		>
			{#each items as item (item.id ?? item)}
				{#if item.separator}
					<div class="my-1 border-t border-gray-200" role="separator"></div>
				{:else}
					<button
						type="button"
						role="menuitem"
						class="w-full px-4 py-2 text-left text-sm hover:bg-gray-100 disabled:cursor-not-allowed disabled:text-gray-400"
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
