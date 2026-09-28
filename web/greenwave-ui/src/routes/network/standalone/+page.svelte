<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  onMount(() => goto('/network', { replaceState: true }));
</script>

<p class="p-4">The network and corridor editors now share one project. <a href="/network" class="text-blue-600 underline">Open network editor</a>.</p>
