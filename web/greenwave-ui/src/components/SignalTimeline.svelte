<script>
  import { createEventDispatcher } from 'svelte';
  import { programStateAt, signalColor } from '$lib/utils/junction-program.js';

  export let timeline = { duration: 0, segments: [] };
  export let time = 0;
  export let offset = 0;
  export let groupId = 0;
  const dispatch = createEventDispatcher();
  $: state = programStateAt(timeline, time, offset);
</script>

<div class="space-y-2" role="group" aria-label={`Signal timeline for group ${groupId}`}>
  <div class="flex flex-wrap items-center justify-between gap-1 text-xs text-gray-600">
    <span class="font-semibold">G{groupId} · {timeline.duration} s cycle</span>
    <span>Program time: {state?.position.toFixed(1) ?? '0'} s</span>
  </div>
  <div class="relative">
    <div class="flex h-10 overflow-hidden rounded border border-gray-200">
      {#each timeline.segments as segment (`${segment.phaseId}:${segment.signalIndex}`)}
        <button type="button" class="min-w-0 overflow-hidden border-r border-white/50 px-1 text-xs font-semibold text-white focus-visible:z-10 focus-visible:outline-2 focus-visible:outline-blue-700"
          style:background={signalColor(segment.color)} style:width={`${segment.duration / timeline.duration * 100}%`}
          aria-label={`Phase ${segment.phaseIndex + 1}, signal ${segment.signalIndex + 1}: ${segment.color}, ${segment.duration} seconds`}
          title={`Phase ${segment.phaseIndex + 1}: ${segment.color}, ${segment.duration} s. Click to edit.`}
          on:click={() => dispatch('select', segment)}>{segment.duration}s</button>
      {/each}
    </div>
    {#if state}
      <span class="pointer-events-none absolute -top-1 bottom-0 w-0.5 bg-slate-900" style:left={`${state.position / timeline.duration * 100}%`} aria-hidden="true"></span>
    {/if}
  </div>
  <div class="flex justify-between text-xs text-gray-500"><span>0 s</span><span>{timeline.duration} s</span></div>
</div>
