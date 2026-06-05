<script>
  import { onMount } from 'svelte';

  export let junctions = [];
  export let showOffsets = false;

  let container;
  let width = 400;
  let height = 400;
  let sized = false;

  function updateSize() {
    if (container) {
      width = container.clientWidth || 400;
      height = container.clientHeight || 400;
    }
  }

  onMount(() => {
    updateSize();
    sized = true;
    window.addEventListener('resize', updateSize);
    return () => window.removeEventListener('resize', updateSize);
  });

  const SIGNAL_BAR_W = 100;
  const ROAD_W = 16;

  const colorMap = {
    RED: '#dc2626',
    YELLOW: '#fbbf24',
    GREEN: '#16a34a',
    GREENPRIORITY: '#15803d'
  };

  function getColor(c) {
    return colorMap[c] || '#6b7280';
  }

  function getTotalDuration(junction) {
    return junction.cycle.reduce((sum, phase) => {
      return sum + (phase.signal_groups?.[0]?.signals ?? []).reduce((s, sig) => s + sig.duration, 0);
    }, 0);
  }

  function getSignals(junction) {
    return junction.cycle.flatMap(phase => phase.signal_groups?.[0]?.signals ?? []);
  }

  function positiveModulo(v, m) {
    return ((v % m) + m) % m;
  }

  // Index of the active signal at t=0, accounting for offset
  function getActiveIdx(junction) {
    const signals = getSignals(junction);
    const T = getTotalDuration(junction);
    if (T === 0) return 0;
    const tInCycle = positiveModulo(-(junction.offset || 0), T);
    let acc = 0;
    for (let i = 0; i < signals.length; i++) {
      acc += signals[i].duration;
      if (tInCycle < acc) return i;
    }
    return 0;
  }

  // Horizontal signal-bar segments (proportional to duration)
  function getSegments(junction) {
    const signals = getSignals(junction);
    const T = getTotalDuration(junction);
    if (T === 0) return [];
    let x = 0;
    return signals.map((sig, i) => {
      const w = (sig.duration / T) * SIGNAL_BAR_W;
      const seg = { x, w, fill: getColor(sig.color), i };
      x += w;
      return seg;
    });
  }

  // Layout constants (reactive so showOffsets can affect right margin)
  $: ml = 110; // left margin (room for label + distance)
  $: mr = showOffsets ? 64 : 16;
  $: mt = 24;
  $: mb = showOffsets ? 36 : 24;
  $: chartH = height - mt - mb;

  $: maxY = junctions.length > 0
    ? Math.max(...junctions.map(j => j.point.y))
    : 100;

  // Y coordinate in SVG for a given distance value
  function yPos(dist) {
    return mt + (dist / (maxY + 80)) * chartH;
  }

  // X of road center line
  $: roadX = ml;
</script>

<div bind:this={container} class="w-full h-full relative">
  {#if junctions.length === 0}
    <div class="flex items-center justify-center h-full text-gray-400 text-sm">
      No junctions to display
    </div>
  {:else if sized}
    <div class="absolute inset-0 overflow-hidden">
    <svg {width} {height} style="display:block">

      <!-- Road -->
      <rect
        x={roadX - ROAD_W / 2}
        y={mt}
        width={ROAD_W}
        height={chartH}
        rx="4"
        fill="#e2e8f0"
        stroke="#cbd5e1"
        stroke-width="1"
      />
      <!-- Center dashed line -->
      <line
        x1={roadX} y1={mt}
        x2={roadX} y2={mt + chartH}
        stroke="#94a3b8"
        stroke-width="1"
        stroke-dasharray="8,6"
      />

      {#each junctions as junction}
        {@const y = yPos(junction.point.y)}
        {@const segments = getSegments(junction)}
        {@const activeIdx = getActiveIdx(junction)}

        <!-- Crossbar on road -->
        <line
          x1={roadX - ROAD_W / 2 - 4}
          x2={roadX + ROAD_W / 2 + 4}
          y1={y} y2={y}
          stroke="#475569" stroke-width="2"
        />

        <!-- Junction label -->
        <text
          x={roadX - ROAD_W / 2 - 8}
          y={y - 3}
          text-anchor="end"
          font-size="11"
          font-weight="600"
          fill="#1e293b"
        >{junction.label || `J${junction.id}`}</text>

        <!-- Distance -->
        <text
          x={roadX - ROAD_W / 2 - 8}
          y={y + 10}
          text-anchor="end"
          font-size="9"
          fill="#94a3b8"
        >{junction.point.y} m</text>

        <!-- Signal bar -->
        <g transform={`translate(${roadX + ROAD_W / 2 + 8}, ${y - 6})`}>
          {#each segments as seg}
            <rect
              x={seg.x} y={0}
              width={seg.w} height={12}
              fill={seg.fill}
              opacity={seg.i === activeIdx ? 1 : 0.3}
              rx="1"
            />
            {#if seg.i === activeIdx}
              <rect
                x={seg.x} y={0}
                width={seg.w} height={12}
                fill="none"
                stroke="white"
                stroke-width="1.5"
                rx="1"
              />
            {/if}
          {/each}
          <rect
            x={0} y={0}
            width={SIGNAL_BAR_W} height={12}
            fill="none"
            stroke="#94a3b8"
            stroke-width="0.5"
            rx="1"
          />
        </g>

        <!-- Offset label -->
        {#if showOffsets}
          <text
            x={roadX + ROAD_W / 2 + 8 + SIGNAL_BAR_W + 6}
            y={y}
            dy="0.35em"
            font-size="10"
            font-weight="700"
            fill="#4B0082"
          >+{junction.offset}s</text>
        {/if}
      {/each}

      <!-- Legend (results panel only) -->
      {#if showOffsets}
        <g transform={`translate(${roadX + ROAD_W / 2 + 8}, ${mt + chartH + 10})`}>
          <rect x={0} y={0} width={10} height={8} fill="#475569" rx="1" />
          <rect x={0} y={0} width={10} height={8} fill="none" stroke="white" stroke-width="1.5" rx="1" />
          <text x={14} y={7} font-size="9" fill="#64748b">= active signal at t=0</text>
        </g>
      {/if}

    </svg>
    </div>
  {/if}
</div>
