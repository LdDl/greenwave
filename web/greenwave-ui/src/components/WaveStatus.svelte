<script>
  import { waveStatus } from '$lib/utils/wave-status.js';

  export let direction = 'Forward';
  export let junctionCount = 0;
  export let waves = [];
  export let throughWaves = [];
  export let calculated = false;
  export let outdated = false;
  export let notice = '';

  $: status = waveStatus(junctionCount, waves, throughWaves, calculated, outdated);
</script>

<div role="status" aria-label={`${direction} wave status`} class="rounded border px-3 py-2 text-xs leading-relaxed"
  class:border-green-200={status.kind === 'success'} class:bg-green-50={status.kind === 'success'} class:text-green-800={status.kind === 'success'}
  class:border-amber-200={status.kind === 'warning'} class:bg-amber-50={status.kind === 'warning'} class:text-amber-800={status.kind === 'warning'}
  class:border-gray-200={status.kind === 'pending'} class:bg-gray-50={status.kind === 'pending'} class:text-gray-600={status.kind === 'pending'}>
  <p><strong>{direction}:</strong> {status.message}</p>
  {#if notice}<p class="mt-1">{notice}</p>{/if}
</div>
