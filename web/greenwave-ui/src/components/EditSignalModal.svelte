<script>
	import { createEventDispatcher } from 'svelte';
	import { fade, scale } from 'svelte/transition';
	import { modalFocus } from '$lib/utils/modal-focus.js';

	export let signal;
	const dispatch = createEventDispatcher();
	let validationError = '';

	function saveSignal() {
		if (!Number.isFinite(signal.duration) || signal.duration <= 0) {
			validationError = 'Enter a positive signal duration.';
			return;
		}
		dispatch('save', { signal });
	}

	function handleBackdropClick(event) {
		if (event.target === event.currentTarget) {
			dispatch('close');
		}
	}
</script>

{#if signal}
	<div
		transition:fade={{ duration: 150 }}
		class="fixed inset-0 z-50 flex items-center justify-center"
		style="background-color: rgba(0, 0, 0, 0.6);"
		on:click={handleBackdropClick}
		on:keydown={(e) => e.key === 'Escape' && dispatch('close')}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
		use:modalFocus
	>
		<div
			transition:scale={{ start: 0.96, duration: 150 }}
			class="mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-lg"
		>
			<h3 class="mb-4 text-lg font-medium">Edit Signal</h3>
			{#if validationError}<p role="alert" class="mb-3 text-sm text-red-700">
					{validationError}
				</p>{/if}
			<div class="space-y-4">
				<div class="flex items-center gap-4">
					<label for="signal-color" class="w-24 shrink-0 text-sm font-medium">Color:</label>
					<select
						id="signal-color"
						bind:value={signal.color}
						class="min-w-0 flex-1 rounded-md border px-3 py-2"
					>
						<option value="GREEN">Green</option>
						<option value="RED">Red</option>
						<option value="YELLOW">Yellow</option>
					</select>
				</div>

				<div class="flex items-center gap-4">
					<label for="signal-duration" class="w-24 shrink-0 text-sm font-medium"
						>Duration (s):</label
					>
					<input
						id="signal-duration"
						type="number"
						bind:value={signal.duration}
						class="min-w-0 flex-1 rounded-md border px-3 py-2"
						min="1"
					/>
				</div>
			</div>

			<div class="mt-6 flex justify-between">
				<button
					class="rounded-md bg-blue-500 px-4 py-2 text-white hover:bg-blue-600 focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:outline-none"
					on:click={saveSignal}
				>
					Save
				</button>
				<button
					class="rounded-md bg-gray-200 px-4 py-2 text-gray-700 hover:bg-gray-300 focus-visible:ring-2 focus-visible:ring-gray-400 focus-visible:outline-none"
					on:click={() => dispatch('close')}
				>
					Cancel
				</button>
			</div>
		</div>
	</div>
{/if}
