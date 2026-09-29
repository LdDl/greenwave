<script>
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { corridorEditor, corridorProject } from '$lib/stores/corridor.js';
	import { corridorsOpen, corridorToEdit } from '$lib/stores/workspace.js';

	export let label = '+ Create corridor';

	function create() {
		const id = corridorEditor.addCorridor(
			`Corridor ${Math.max(...$corridorProject.corridors.map((route) => route.id)) + 2}`
		);
		corridorToEdit.set(id);
		corridorsOpen.set(false);
		if ($page.url.pathname !== '/network') goto('/network');
	}
</script>

<button
	class="ui-button ui-primary"
	title="Create a corridor and choose its route through the network"
	on:click={create}>{label}</button
>
