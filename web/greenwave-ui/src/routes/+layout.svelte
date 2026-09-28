<script>
	import '../app.css';
	import { onMount } from 'svelte';
	import { corridorEditor } from '$lib/stores/corridor.js';
	import { invalidateAll } from '$lib/stores/invalidation';

	let { children } = $props();

	onMount(() => {
		let storage = null;
		try { storage = window.localStorage; } catch { storage = null; }
		const disconnect = corridorEditor.connectStorage(storage);
		let first = true;
		const unsubscribe = corridorEditor.calculationRevision.subscribe(() => {
			if (first) { first = false; return; }
			invalidateAll('corridor configuration changed');
		});
		return () => { unsubscribe(); disconnect(); };
	});
</script>

{@render children()}
