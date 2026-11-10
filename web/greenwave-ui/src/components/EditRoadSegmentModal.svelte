<script>
	import { createEventDispatcher } from 'svelte';
	import { fade, scale } from 'svelte/transition';

	import { MAX_LANES } from '$lib/utils/network-geometry.js';
	import { modalFocus } from '$lib/utils/modal-focus.js';

	export let fromLabel = '';
	export let toLabel = '';
	export let includeLength = false;
	let lengthMeters = 150;

	const dispatch = createEventDispatcher();

	let twoWay = true;
	let lanesFwd = 1;
	let lanesBack = 1;

	$: valid =
		(!includeLength || (Number.isFinite(lengthMeters) && lengthMeters >= 0)) &&
		Number.isInteger(lanesFwd) &&
		lanesFwd >= 1 &&
		lanesFwd <= MAX_LANES &&
		(!twoWay || (Number.isInteger(lanesBack) && lanesBack >= 1 && lanesBack <= MAX_LANES));

	function save() {
		if (!valid) return;
		dispatch('save', {
			lanesFwd,
			lanesBack: twoWay ? lanesBack : 0,
			...(includeLength ? { lengthMeters } : {})
		});
	}

	function handleBackdrop(e) {
		if (e.target === e.currentTarget) dispatch('close');
	}
</script>

<div
	use:modalFocus
	aria-labelledby="road-modal-title"
	transition:fade={{ duration: 150 }}
	class="fixed inset-0 z-50 flex items-center justify-center"
	style="background-color: rgba(0,0,0,0.5)"
	on:click={handleBackdrop}
	on:keydown={(e) => {
		if (e.key === 'Escape') {
			e.stopPropagation();
			dispatch('close');
		}
	}}
	role="dialog"
	aria-modal="true"
	tabindex="-1"
>
	<form
		on:submit|preventDefault={save}
		transition:scale={{ start: 0.96, duration: 150 }}
		class="mx-4 w-80 rounded-lg bg-white p-6 shadow-xl"
	>
		<h3 id="road-modal-title" class="mb-3 text-base font-semibold">New Road Segment</h3>
		<div class="mb-5 flex items-center gap-2">
			<span
				class="rounded-md border border-green-200 bg-green-50 px-2 py-0.5 text-sm font-medium text-green-700"
				>{fromLabel}</span
			>
			<span class="text-sm text-gray-400">→</span>
			<span
				class="rounded-md border border-red-200 bg-red-50 px-2 py-0.5 text-sm font-medium text-red-600"
				>{toLabel}</span
			>
		</div>

		{#if includeLength}
			<label for="road-new-length" class="block text-xs text-gray-500">Length (m)</label>
			<input
				id="road-new-length"
				type="number"
				min="0"
				step="any"
				bind:value={lengthMeters}
				class="mt-1 mb-4 w-full rounded border border-gray-300 px-3 py-2 text-sm"
			/>
		{/if}

		<!-- Direction toggle -->
		<p class="mb-1.5 text-xs font-medium text-gray-500">Direction</p>
		<div class="mb-5 inline-flex w-full gap-0.5 rounded-lg bg-gray-100 p-0.5 text-sm">
			<button
				type="button"
				class="flex-1 rounded-md py-1.5 font-medium transition-all"
				class:bg-white={!twoWay}
				class:shadow-sm={!twoWay}
				class:text-gray-900={!twoWay}
				class:text-gray-500={twoWay}
				on:click={() => (twoWay = false)}>One-way →</button
			>
			<button
				type="button"
				class="flex-1 rounded-md py-1.5 font-medium transition-all"
				class:bg-white={twoWay}
				class:shadow-sm={twoWay}
				class:text-gray-900={twoWay}
				class:text-gray-500={!twoWay}
				on:click={() => (twoWay = true)}>Two-way ↔</button
			>
		</div>

		<!-- Lane counts -->
		<div class="space-y-3">
			<div>
				<label for="rse-lanes-fwd" class="mb-1 block text-xs text-gray-500">
					Lanes <span class="font-semibold text-green-600">→</span>
					&nbsp;<span class="text-gray-400">({fromLabel} → {toLabel})</span>
				</label>
				<input
					id="rse-lanes-fwd"
					type="number"
					min="1"
					max={MAX_LANES}
					step="1"
					bind:value={lanesFwd}
					class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-400 focus:outline-none"
				/>
			</div>

			{#if twoWay}
				<div>
					<label for="rse-lanes-back" class="mb-1 block text-xs text-gray-500">
						Lanes <span class="font-semibold text-red-500">←</span>
						&nbsp;<span class="text-gray-400">({toLabel} → {fromLabel})</span>
					</label>
					<input
						id="rse-lanes-back"
						type="number"
						min="1"
						max={MAX_LANES}
						step="1"
						bind:value={lanesBack}
						class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-400 focus:outline-none"
					/>
				</div>
			{/if}
		</div>

		{#if !valid}<p class="mt-3 text-xs text-red-600" role="alert">
				Lane counts must be whole numbers from 1 to {MAX_LANES}.
			</p>{/if}
		<div class="mt-6 flex justify-end gap-2">
			<button
				type="button"
				class="rounded-md bg-gray-200 px-4 py-2 text-sm text-gray-700 hover:bg-gray-300 focus-visible:ring-2 focus-visible:ring-gray-400 focus-visible:outline-none"
				on:click={() => dispatch('close')}>Cancel</button
			>
			<button
				type="submit"
				class="rounded-md bg-blue-500 px-4 py-2 text-sm text-white hover:bg-blue-600 focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:outline-none"
				disabled={!valid}>Add road</button
			>
		</div>
	</form>
</div>
