<script>
	import { fade, scale } from 'svelte/transition';

	export let show = false;
	export let title = 'Confirm Action';
	export let message = 'Are you sure you want to proceed?';
	export let confirmText = 'Confirm';
	export let cancelText = 'Cancel';
	export let onConfirm = () => {};
	export let onCancel = () => {};
	export let danger = false;

	function handleConfirm() {
		onConfirm();
		show = false;
	}

	function handleCancel() {
		onCancel();
		show = false;
	}

	function handleBackdropClick(event) {
		if (event.target === event.currentTarget) {
			handleCancel();
		}
	}
</script>

{#if show}
	<div
		transition:fade={{ duration: 150 }}
		class="fixed inset-0 z-50 flex items-center justify-center"
		style="background-color: rgba(0, 0, 0, 0.6);"
		on:click={handleBackdropClick}
		on:keydown={(e) => e.key === 'Escape' && handleCancel()}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div
			transition:scale={{ start: 0.96, duration: 150 }}
			class="mx-4 w-full max-w-md rounded-lg border border-gray-200 bg-white p-6 shadow-2xl"
		>
			<h3 class="mb-3 text-lg font-semibold text-gray-900">{title}</h3>
			<p class="mb-6 text-sm leading-relaxed text-gray-600">{message}</p>

			<div class="flex justify-end gap-3">
				<button
					on:click={handleCancel}
					class="rounded-md border border-gray-300 px-4 py-2 text-sm text-gray-700 transition-colors hover:bg-gray-50 focus-visible:ring-2 focus-visible:ring-gray-400 focus-visible:outline-none"
				>
					{cancelText}
				</button>
				<button
					on:click={handleConfirm}
					class="rounded-md px-4 py-2 text-sm text-white transition-colors focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:outline-none"
					class:bg-red-500={danger}
					class:hover:bg-red-600={danger}
					class:focus-visible:ring-red-400={danger}
					class:bg-blue-500={!danger}
					class:hover:bg-blue-600={!danger}
					class:focus-visible:ring-blue-400={!danger}
				>
					{confirmText}
				</button>
			</div>
		</div>
	</div>
{/if}
