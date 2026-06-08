<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { networkNodes, networkEdges, nextNodeId } from '$lib/stores/network.js';

  // 'select' | 'node' | 'edge'
  export let mode = 'select';
  // Highlight connected nodes when an edge is selected (node IDs or null)
  export let highlightFrom = null;
  export let highlightTo   = null;

  const dispatch = createEventDispatcher();

  let container;
  let svgEl;
  let width = 800;
  let height = 600;
  let sized = false;

  // Pan / zoom
  let tx = 0;
  let ty = 0;
  let scale = 1;

  // Pan drag state
  let isPanning = false;
  let panStart = null;

  // Node drag state
  let dragNodeId = null;
  let dragOffset = { x: 0, y: 0 };

  // Edge drawing state
  let edgeFromId = null;
  let hoverNodeId = null; // for red-preview on destination while drawing

  // Track whether pointer moved since mousedown (to distinguish click from drag)
  let pointerMoved = false;

  // Live mouse position in canvas coords (for edge preview)
  let mouseCanvas = { x: 0, y: 0 };

  function updateSize() {
    if (container) {
      width = container.clientWidth || 800;
      height = container.clientHeight || 600;
    }
  }

  onMount(() => {
    updateSize();
    sized = true;
    window.addEventListener('resize', updateSize);
    return () => window.removeEventListener('resize', updateSize);
  });

  // Convert window coords to canvas coords
  function toCanvas(clientX, clientY) {
    const rect = svgEl.getBoundingClientRect();
    return {
      x: (clientX - rect.left - tx) / scale,
      y: (clientY - rect.top - ty) / scale,
    };
  }

  // Window-level mouse events (for smooth drag even if cursor leaves SVG)

  function onWindowMousemove(e) {
    const pos = toCanvas(e.clientX, e.clientY);
    mouseCanvas = pos;

    if (isPanning && panStart) {
      pointerMoved = true;
      tx = e.clientX - panStart.x;
      ty = e.clientY - panStart.y;
    }

    if (dragNodeId !== null) {
      pointerMoved = true;
      networkNodes.update(ns =>
        ns.map(n =>
          n.id === dragNodeId
            ? { ...n, x: pos.x - dragOffset.x, y: pos.y - dragOffset.y }
            : n
        )
      );
    }
  }

  function onWindowMouseup() {
    isPanning = false;
    dragNodeId = null;
    panStart = null;
  }

  // SVG background

  function onBgMousedown(e) {
    if (e.button !== 0) return;
    pointerMoved = false;
    if (mode === 'select') {
      isPanning = true;
      panStart = { x: e.clientX - tx, y: e.clientY - ty };
    }
  }

  function onBgClick(e) {
    if (pointerMoved) return;
    if (mode === 'node') {
      const pos = toCanvas(e.clientX, e.clientY);
      const id = nextNodeId();
      networkNodes.update(ns => [
        ...ns,
        { id, x: pos.x, y: pos.y, label: `J${id}` },
      ]);
      dispatch('nodeAdded', { id });
    } else if (mode === 'edge') {
      edgeFromId = null; // cancel pending edge
    } else if (mode === 'select') {
      dispatch('deselect');
    }
  }

  // Node events

  function onNodeMousedown(e, node) {
    e.stopPropagation();
    if (e.button !== 0) return;
    pointerMoved = false;
    if (mode === 'select') {
      const pos = toCanvas(e.clientX, e.clientY);
      dragNodeId = node.id;
      dragOffset = { x: pos.x - node.x, y: pos.y - node.y };
    }
  }

  function onNodeClick(e, node) {
    e.stopPropagation();
    if (pointerMoved) return;
    if (mode === 'node') return; // clicking node in node-mode does nothing
    if (mode === 'edge') {
      if (edgeFromId === null) {
        edgeFromId = node.id;
      } else if (edgeFromId !== node.id) {
        // Avoid duplicate edges
        const dup = $networkEdges.find(
          ed =>
            (ed.from === edgeFromId && ed.to === node.id) ||
            (ed.from === node.id && ed.to === edgeFromId)
        );
        if (!dup) {
          dispatch('edgePending', { fromId: edgeFromId, toId: node.id });
        }
        edgeFromId = null;
      }
    } else if (mode === 'select') {
      dispatch('selectNode', { node });
    }
  }

  // Edge events

  function onEdgeClick(e, edge) {
    e.stopPropagation();
    if (mode === 'select') {
      dispatch('selectEdge', { edge });
    }
  }

  // Scroll to zoom

  function onWheel(e) {
    e.preventDefault();
    const factor = e.deltaY < 0 ? 1.15 : 1 / 1.15;
    const rect = svgEl.getBoundingClientRect();
    const mx = e.clientX - rect.left;
    const my = e.clientY - rect.top;
    tx = mx + (tx - mx) * factor;
    ty = my + (ty - my) * factor;
    scale = Math.min(Math.max(scale * factor, 0.1), 8);
  }

  // Helpers

  function getNode(id, ns) {
    return ns.find(n => n.id === id);
  }

  // Point on quadratic bezier at parameter t
  function bezierPt(p0x, p0y, cpx, cpy, p1x, p1y, t) {
    const mt = 1 - t;
    return { x: mt*mt*p0x + 2*mt*t*cpx + t*t*p1x,
             y: mt*mt*p0y + 2*mt*t*cpy + t*t*p1y };
  }

  // Unit tangent on quadratic bezier at t
  function bezierDir(p0x, p0y, cpx, cpy, p1x, p1y, t) {
    const mt = 1 - t;
    const dx = 2*mt*(cpx - p0x) + 2*t*(p1x - cpx);
    const dy = 2*mt*(cpy - p0y) + 2*t*(p1y - cpy);
    const len = Math.sqrt(dx*dx + dy*dy) || 1;
    return { x: dx/len, y: dy/len };
  }

  // Arrow offset so arrow tip sits at node edge, not center
  const NODE_R = 14;
  function edgeEndpoints(a, b) {
    const dx = b.x - a.x;
    const dy = b.y - a.y;
    const len = Math.sqrt(dx * dx + dy * dy) || 1;
    const ox = (dx / len) * (NODE_R / scale);
    const oy = (dy / len) * (NODE_R / scale);
    return {
      x1: a.x + ox, y1: a.y + oy,
      x2: b.x - ox, y2: b.y - oy,
    };
  }

  $: transform = `translate(${tx},${ty}) scale(${scale})`;
  $: edgeFromNode = edgeFromId != null ? $networkNodes.find(n => n.id === edgeFromId) : null;

  // Cursor per mode
  $: canvasCursor =
    mode === 'node' ? 'crosshair' :
    mode === 'edge' ? 'crosshair' :
    isPanning ? 'grabbing' : 'grab';
</script>

<svelte:window on:mousemove={onWindowMousemove} on:mouseup={onWindowMouseup} />

<div bind:this={container} class="absolute inset-0" style="cursor:{canvasCursor}">
  {#if sized}
    <svg
      bind:this={svgEl}
      {width}
      {height}
      style="display:block; user-select:none"
      on:mousedown={onBgMousedown}
      on:click={onBgClick}
      on:wheel|nonpassive={onWheel}
    >
      <!-- Background -->
      <rect {width} {height} fill="#f8fafc" />

      <!-- Dot grid (moves with canvas) -->
      <defs>
        <pattern
          id="dot-grid"
          width={40}
          height={40}
          patternUnits="userSpaceOnUse"
          patternTransform={transform}
        >
          <circle cx="0" cy="0" r="1.5" fill="#cbd5e1" />
        </pattern>
      </defs>
      <rect {width} {height} fill="url(#dot-grid)" />

      <g {transform}>
        <!-- Edges -->
        {#each $networkEdges as edge}
          {@const a = getNode(edge.from, $networkNodes)}
          {@const b = getNode(edge.to, $networkNodes)}
          {#if a && b}
            {@const ep  = edgeEndpoints(a, b)}
            {@const dx  = ep.x2 - ep.x1}
            {@const dy  = ep.y2 - ep.y1}
            {@const len = Math.sqrt(dx*dx + dy*dy) || 1}
            {@const ux  = dx/len}
            {@const uy  = dy/len}
            {@const two = edge.lanes_back > 0}
            <!-- Bend amount in canvas units — curves bow apart for two-way roads -->
            {@const bend = two ? 22 : 0}
            {@const mx = (ep.x1 + ep.x2) / 2}
            {@const my = (ep.y1 + ep.y2) / 2}
            <!-- Control points: perpendicular to edge, opposite sides -->
            {@const cfx = mx - uy * bend}  {@const cfy = my + ux * bend}
            {@const cbx = mx + uy * bend}  {@const cby = my - ux * bend}
            <!-- SVG path strings -->
            {@const fwdD  = two ? `M ${ep.x1},${ep.y1} Q ${cfx},${cfy} ${ep.x2},${ep.y2}` : `M ${ep.x1},${ep.y1} L ${ep.x2},${ep.y2}`}
            {@const backD = two ? `M ${ep.x2},${ep.y2} Q ${cbx},${cby} ${ep.x1},${ep.y1}` : null}
            <!-- Arrow positions & directions on fwd path at t=0.65 -->
            {@const fAP  = two ? bezierPt(ep.x1,ep.y1,cfx,cfy,ep.x2,ep.y2,0.65)  : { x: ep.x1+dx*0.65, y: ep.y1+dy*0.65 }}
            {@const fAD  = two ? bezierDir(ep.x1,ep.y1,cfx,cfy,ep.x2,ep.y2,0.65) : { x: ux, y: uy }}
            <!-- Arrow on back path at t=0.65 (back goes b -> a) -->
            {@const bAP  = two ? bezierPt(ep.x2,ep.y2,cbx,cby,ep.x1,ep.y1,0.65)  : null}
            {@const bAD  = two ? bezierDir(ep.x2,ep.y2,cbx,cby,ep.x1,ep.y1,0.65) : null}
            {@const as   = 6}

            <!-- Hit areas (both arcs select the same edge) -->
            <path d={fwdD} stroke="transparent" stroke-width={14/scale} fill="none"
              style="cursor:pointer" on:click={e => onEdgeClick(e, edge)} />
            {#if two && backD}
              <path d={backD} stroke="transparent" stroke-width={14/scale} fill="none"
                style="cursor:pointer" on:click={e => onEdgeClick(e, edge)} />
            {/if}

            <!-- Forward path -->
            <path d={fwdD} stroke="#475569" stroke-width={2/scale} fill="none" pointer-events="none" />
            <!-- Forward arrowhead -->
            <polygon
              points={`${fAP.x+fAD.x*as},${fAP.y+fAD.y*as} ${fAP.x-fAD.x*as-fAD.y*as*0.5},${fAP.y-fAD.y*as+fAD.x*as*0.5} ${fAP.x-fAD.x*as+fAD.y*as*0.5},${fAP.y-fAD.y*as-fAD.x*as*0.5}`}
              fill="#475569" pointer-events="none"
            />
            <!-- Forward lane label offset to the curve side -->
            <text
              x={two ? cfx : mx} y={two ? cfy - 7 : my - 8/scale}
              text-anchor="middle" font-size={9/scale} fill="#475569" pointer-events="none"
            >→{edge.lanes_fwd}</text>

            <!-- Backward path + arrow + label (two-way only) -->
            {#if two && backD && bAP && bAD}
              <path d={backD} stroke="#94a3b8" stroke-width={2/scale} fill="none" pointer-events="none" />
              <polygon
                points={`${bAP.x+bAD.x*as},${bAP.y+bAD.y*as} ${bAP.x-bAD.x*as-bAD.y*as*0.5},${bAP.y-bAD.y*as+bAD.x*as*0.5} ${bAP.x-bAD.x*as+bAD.y*as*0.5},${bAP.y-bAD.y*as-bAD.x*as*0.5}`}
                fill="#94a3b8" pointer-events="none"
              />
              <text
                x={cbx} y={cby - 7}
                text-anchor="middle" font-size={9/scale} fill="#94a3b8" pointer-events="none"
              >←{edge.lanes_back}</text>
            {/if}
          {/if}
        {/each}

        <!-- Preview edge while drawing -->
        {#if edgeFromNode && mode === 'edge'}
          <line
            x1={edgeFromNode.x} y1={edgeFromNode.y}
            x2={mouseCanvas.x} y2={mouseCanvas.y}
            stroke="#3b82f6"
            stroke-width={2 / scale}
            stroke-dasharray={`${6 / scale},${4 / scale}`}
            pointer-events="none"
          />
        {/if}

        <!-- Ghost node preview in node mode -->
        {#if mode === 'node'}
          <circle
            cx={mouseCanvas.x} cy={mouseCanvas.y}
            r={NODE_R / scale}
            fill="#dbeafe"
            stroke="#3b82f6"
            stroke-width={2 / scale}
            stroke-dasharray={`${4/scale},${3/scale}`}
            opacity="0.75"
            pointer-events="none"
          />
        {/if}

        <!-- Nodes -->
        {#each $networkNodes as node}
          {@const isEdgeSrc  = edgeFromId === node.id}
          {@const isDstHover = mode === 'edge' && edgeFromId != null && hoverNodeId === node.id && node.id !== edgeFromId}
          {@const isHlFrom   = highlightFrom === node.id}
          {@const isHlTo     = highlightTo   === node.id}
          {@const nodeFill   = isEdgeSrc || isHlFrom ? '#dcfce7' : isDstHover || isHlTo ? '#fee2e2' : '#ffffff'}
          {@const nodeStroke = isEdgeSrc || isHlFrom ? '#16a34a' : isDstHover || isHlTo ? '#ef4444' : '#475569'}
          {@const labelColor = isEdgeSrc || isHlFrom ? '#15803d' : isDstHover || isHlTo ? '#dc2626' : '#1e293b'}
          <!-- svelte-ignore a11y_interactive_supports_focus -->
          <g
            role="button"
            style="cursor:pointer"
            on:mousedown={e => onNodeMousedown(e, node)}
            on:click={e => onNodeClick(e, node)}
            on:dblclick={e => { e.stopPropagation(); if (mode === 'select') dispatch('openIntersection', { nodeId: node.id }); }}
            on:mouseenter={() => hoverNodeId = node.id}
            on:mouseleave={() => hoverNodeId = null}
          >
            <circle
              cx={node.x} cy={node.y}
              r={NODE_R / scale}
              fill={nodeFill}
              stroke={nodeStroke}
              stroke-width={2 / scale}
            />
            <text
              x={node.x} y={node.y}
              text-anchor="middle"
              dominant-baseline="central"
              font-size={11 / scale}
              font-weight="600"
              fill={labelColor}
              pointer-events="none"
            >{node.label}</text>
          </g>
        {/each}
      </g>
    </svg>

    <!-- Mode hint overlay -->
    {#if mode === 'select' && $networkNodes.length > 0}
      <div class="absolute bottom-3 left-1/2 -translate-x-1/2 bg-gray-700/60 text-white text-xs px-3 py-1.5 rounded-full pointer-events-none shadow">
        Double-click a junction to configure its intersection
      </div>
    {:else if mode === 'edge' && edgeFromId === null}
      <div class="absolute bottom-3 left-1/2 -translate-x-1/2 bg-blue-600 text-white text-xs px-3 py-1.5 rounded-full pointer-events-none shadow">
        Click a junction to start a road
      </div>
    {:else if mode === 'edge' && edgeFromId !== null}
      <div class="absolute bottom-3 left-1/2 -translate-x-1/2 bg-blue-600 text-white text-xs px-3 py-1.5 rounded-full pointer-events-none shadow">
        Click another junction to finish — or background to cancel
      </div>
    {:else if mode === 'node'}
      <div class="absolute bottom-3 left-1/2 -translate-x-1/2 bg-blue-600 text-white text-xs px-3 py-1.5 rounded-full pointer-events-none shadow">
        Click on the canvas to place a junction
      </div>
    {/if}
  {/if}
</div>
