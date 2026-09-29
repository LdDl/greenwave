<script>
	import { corridorEditor, corridorProject, selectedCorridor } from '$lib/stores/corridor.js';
	import { corridorsOpen } from '$lib/stores/workspace.js';
	import { corridorProjection } from '$lib/utils/shared-project.js';
	import { modalFocus } from '$lib/utils/modal-focus.js';
	import { goto } from '$app/navigation';

	function select(id) {
		corridorEditor.selectCorridor(id);
		corridorsOpen.set(false);
	}
	function create() {
		corridorEditor.addCorridor(
			`Corridor ${Math.max(...$corridorProject.corridors.map((route) => route.id)) + 2}`
		);
		corridorsOpen.set(false);
		goto('/network');
	}
</script>

{#if $corridorsOpen}
	<div
		class="workspace-overlay corridor-overlay"
		role="dialog"
		aria-modal="true"
		aria-labelledby="corridor-picker-title"
		tabindex="-1"
		use:modalFocus
		on:keydown={(event) => {
			if (event.key === 'Escape') {
				event.stopPropagation();
				corridorsOpen.set(false);
			}
		}}
	>
		<section class="corridor-picker">
			<header>
				<div>
					<p class="ui-eyebrow">Routes through your network</p>
					<h2 id="corridor-picker-title">Corridors</h2>
				</div>
				<button
					class="ui-button"
					aria-label="Close corridors"
					on:click={() => corridorsOpen.set(false)}>×</button
				>
			</header>
			<p>
				A corridor is an ordered route through the same network. Select one to edit its input and
				calculate its green waves.
			</p>
			<div class="corridor-options">
				{#each $corridorProject.corridors as route (route.id)}
					{@const projection = corridorProjection($corridorProject, route)}
					{@const nodes = projection.input.junctions}
					<button
						class="corridor-option"
						aria-pressed={route.id === $selectedCorridor.id}
						on:click={() => select(route.id)}
					>
						<strong
							>{route.name}{#if route.id === $selectedCorridor.id}<span>Active</span>{/if}</strong
						>
						<small
							>{nodes.length} junctions · {Math.round(
								(nodes.at(-1)?.point.y ?? 0) - (nodes[0]?.point.y ?? 0)
							)} m · {route.direction === 'bidirectional' ? 'Both directions' : 'Forward'}</small
						>
						<span class="route-sequence"
							>{nodes.length
								? nodes.map((node) => node.label).join(' → ')
								: 'Choose junctions on the network to build this route.'}</span
						>
						{#if projection.error}<span class="ui-warning"
								>Route needs attention: {projection.error}</span
							>{/if}
					</button>
				{/each}
			</div>
			<footer>
				<button class="ui-button ui-primary" on:click={create}>+ Create corridor</button><a
					href="/network"
					class="ui-button"
					on:click={() => corridorsOpen.set(false)}>Edit route on network</a
				>
			</footer>
		</section>
	</div>
{/if}
