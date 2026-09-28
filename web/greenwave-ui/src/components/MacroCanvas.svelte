<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { networkNodes, networkEdges, networkEditor } from '$lib/stores/network.js';

  // 'select' | 'node' | 'edge'
  export let mode = 'select';
  export let disabled = false;
  export let selectedNodeId = null;
  export let selectedEdgeId = null;
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
  let hoverNodeId = null;
  let pointerStart = null;
  let inside = false;

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
    const observer = new ResizeObserver(updateSize);
    observer.observe(container);
    return () => { observer.disconnect(); networkEditor.finish(); };
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
    if (disabled || !svgEl) return;
    const pos = toCanvas(e.clientX, e.clientY);
    mouseCanvas = pos;
    if (pointerStart && Math.hypot(e.clientX - pointerStart.x, e.clientY - pointerStart.y) > 4) pointerMoved = true;
    if (!pointerMoved) return;
    if (isPanning && panStart) {
      tx = e.clientX - panStart.x;
      ty = e.clientY - panStart.y;
    }
    if (dragNodeId !== null) {
      networkEditor.change(p => {
        const node = p.nodes.find(n => n.id === dragNodeId);
        if (node) { node.x = pos.x - dragOffset.x; node.y = pos.y - dragOffset.y; }
      });
    }
  }

  function onWindowMouseup() {
    if (dragNodeId !== null) networkEditor.finish();
    isPanning = false;
    dragNodeId = null;
    panStart = null;
    pointerStart = null;
  }

  function onBgMousedown(e) {
    if (disabled || e.button !== 0) return;
    pointerMoved = false;
    pointerStart = { x: e.clientX, y: e.clientY };
    if (mode === 'select') {
      isPanning = true;
      panStart = { x: e.clientX - tx, y: e.clientY - ty };
    }
  }

  function onBgClick(e) {
    if (disabled || pointerMoved) return;
    if (mode === 'node') {
      const pos = toCanvas(e.clientX, e.clientY);
      const id = networkEditor.addNode(pos.x, pos.y);
      dispatch('nodeAdded', { id });
    } else if (mode === 'edge') edgeFromId = null;
    else dispatch('deselect');
  }

  function onNodeMousedown(e, node) {
    e.stopPropagation();
    if (disabled || e.button !== 0) return;
    pointerMoved = false;
    pointerStart = { x: e.clientX, y: e.clientY };
    if (mode === 'select') {
      const pos = toCanvas(e.clientX, e.clientY);
      dragNodeId = node.id;
      dragOffset = { x: pos.x - node.x, y: pos.y - node.y };
      networkEditor.begin();
    }
  }

  function onNodeClick(e, node) {
    e.stopPropagation();
    if (disabled || pointerMoved || mode === 'node') return;
    if (mode === 'edge') {
      if (edgeFromId === null) edgeFromId = node.id;
      else if (edgeFromId !== node.id) {
        const duplicate = $networkEdges.some(edge =>
          (edge.from === edgeFromId && edge.to === node.id) || (edge.from === node.id && edge.to === edgeFromId));
        if (duplicate) dispatch('notice', 'These junctions already have a road. Select it to change its lanes or direction.');
        else dispatch('edgePending', { fromId: edgeFromId, toId: node.id });
        edgeFromId = null;
      }
    } else dispatch('selectNode', { node });
  }

  function onEdgeClick(e, edge) {
    e.stopPropagation();
    if (!disabled && !pointerMoved && mode === 'select') dispatch('selectEdge', { edge });
  }

  function activateNode(event, node) {
    if (disabled) return;
    const directions = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1] };
    if (mode === 'select' && directions[event.key]) {
      event.preventDefault();
      event.stopPropagation();
      const [dx, dy] = directions[event.key];
      const distance = (event.shiftKey ? 50 : 10) / scale;
      networkEditor.change(p => {
        const moving = p.nodes.find(n => n.id === node.id);
        moving.x += dx * distance;
        moving.y += dy * distance;
      });
      return;
    }
    if (event.key !== 'Enter' && event.key !== ' ') return;
    event.preventDefault();
    pointerMoved = false;
    onNodeClick(event, node);
  }

  function activateEdge(event, edge) {
    if (event.key !== 'Enter' && event.key !== ' ') return;
    event.preventDefault();
    pointerMoved = false;
    onEdgeClick(event, edge);
  }

  function zoomAt(factor, mx = width / 2, my = height / 2) {
    const next = Math.min(Math.max(scale * factor, 0.1), 8);
    const ratio = next / scale;
    tx = mx + (tx - mx) * ratio;
    ty = my + (ty - my) * ratio;
    scale = next;
  }

  function onWheel(e) {
    if (disabled) return;
    e.preventDefault();
    const rect = svgEl.getBoundingClientRect();
    zoomAt(e.deltaY < 0 ? 1.15 : 1 / 1.15, e.clientX - rect.left, e.clientY - rect.top);
  }

  export function cancelDrawing() { edgeFromId = null; onWindowMouseup(); }

  export function fit() {
    updateSize();
    if (!$networkNodes.length) { tx = 0; ty = 0; scale = 1; return; }
    const xs = $networkNodes.map(n => n.x);
    const ys = $networkNodes.map(n => n.y);
    const minX = Math.min(...xs), maxX = Math.max(...xs);
    const minY = Math.min(...ys), maxY = Math.max(...ys);
    scale = Math.max(0.1, Math.min(2, Math.max(1, width - 120) / Math.max(100, maxX - minX), Math.max(1, height - 120) / Math.max(100, maxY - minY)));
    tx = width / 2 - (minX + maxX) * scale / 2;
    ty = height / 2 - (minY + maxY) * scale / 2;
  }

  function canvasKeys(event) {
    if (disabled || event.target !== svgEl) return;
    if (event.key === 'Enter' && mode === 'node') {
      networkEditor.addNode((width / 2 - tx) / scale, (height / 2 - ty) / scale);
    } else if (event.key === '+' || event.key === '=') zoomAt(1.15);
    else if (event.key === '-') zoomAt(1 / 1.15);
    else if (event.key === 'ArrowLeft') tx += 30;
    else if (event.key === 'ArrowRight') tx -= 30;
    else if (event.key === 'ArrowUp') ty += 30;
    else if (event.key === 'ArrowDown') ty -= 30;
    else return;
    event.preventDefault();
  }

  $: if (mode !== 'edge' || disabled) edgeFromId = null;

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

<svelte:window on:pointermove={onWindowMousemove} on:pointerup={onWindowMouseup} on:pointercancel={onWindowMouseup} on:blur={onWindowMouseup} />

<div bind:this={container} class="absolute inset-0" style="cursor:{canvasCursor}">
  {#if sized}
    <!-- svelte-ignore a11y_no_noninteractive_tabindex a11y_no_noninteractive_element_interactions -->
    <svg
      bind:this={svgEl}
      {width}
      {height}
      style="display:block; user-select:none; touch-action:none"
      role="application" aria-label="Road network canvas" aria-describedby="canvas-keyboard-help" tabindex="0"
      on:keydown={canvasKeys}
      on:pointerenter={() => inside = true} on:pointerleave={() => inside = false}
      on:pointerdown={onBgMousedown}
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
        {#each $networkEdges as edge (edge.id)}
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
            <!-- Bend amount in canvas units - curves bow apart for two-way roads -->
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
            {@const as = 6 / scale}
            {@const edgeColor = selectedEdgeId === edge.id ? '#2563eb' : '#475569'}

            <!-- Hit areas (both arcs select the same edge) -->
            <path d={fwdD} stroke="transparent" stroke-width={14/scale} fill="none"
              style="cursor:pointer" role="button" tabindex="0" aria-label={`Road ${a.label} to ${b.label}`} aria-pressed={selectedEdgeId === edge.id}
              on:pointerdown={e => { e.stopPropagation(); pointerMoved = false; }} on:keydown={e => activateEdge(e, edge)} on:click={e => onEdgeClick(e, edge)} />
            {#if two && backD}
              <path d={backD} stroke="transparent" stroke-width={14/scale} fill="none"
                style="cursor:pointer" role="button" tabindex="-1" aria-label={`Road ${b.label} to ${a.label}`}
                on:pointerdown={e => { e.stopPropagation(); pointerMoved = false; }} on:keydown={e => activateEdge(e, edge)} on:click={e => onEdgeClick(e, edge)} />
            {/if}

            <!-- Forward path -->
            <path d={fwdD} stroke={edgeColor} stroke-width={(selectedEdgeId === edge.id ? 3 : 2)/scale} fill="none" pointer-events="none" />
            <!-- Forward arrowhead -->
            <polygon
              points={`${fAP.x+fAD.x*as},${fAP.y+fAD.y*as} ${fAP.x-fAD.x*as-fAD.y*as*0.5},${fAP.y-fAD.y*as+fAD.x*as*0.5} ${fAP.x-fAD.x*as+fAD.y*as*0.5},${fAP.y-fAD.y*as-fAD.x*as*0.5}`}
              fill={edgeColor} pointer-events="none"
            />
            <!-- Forward lane label offset to the curve side -->
            <text
              x={two ? cfx : mx} y={two ? cfy - 7 : my - 8/scale}
              text-anchor="middle" font-size={9/scale} fill={edgeColor} pointer-events="none"
            >→{edge.lanes_fwd}</text>

            <!-- Backward path + arrow + label (two-way only) -->
            {#if two && backD && bAP && bAD}
              <path d={backD} stroke={selectedEdgeId === edge.id ? '#2563eb' : '#94a3b8'} stroke-width={2/scale} fill="none" pointer-events="none" />
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
        {#if mode === 'node' && inside}
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
        {#each $networkNodes as node (node.id)}
          {@const isEdgeSrc  = edgeFromId === node.id}
          {@const isDstHover = mode === 'edge' && edgeFromId != null && hoverNodeId === node.id && node.id !== edgeFromId}
          {@const isHlFrom   = highlightFrom === node.id}
          {@const isHlTo     = highlightTo   === node.id}
          {@const nodeFill   = selectedNodeId === node.id ? '#dbeafe' : isEdgeSrc || isHlFrom ? '#dcfce7' : isDstHover || isHlTo ? '#fee2e2' : '#ffffff'}
          {@const nodeStroke = selectedNodeId === node.id ? '#2563eb' : isEdgeSrc || isHlFrom ? '#16a34a' : isDstHover || isHlTo ? '#ef4444' : '#475569'}
          {@const labelColor = isEdgeSrc || isHlFrom ? '#15803d' : isDstHover || isHlTo ? '#dc2626' : '#1e293b'}
          <g
            role="button" tabindex="0" aria-label={`Junction ${node.label}`} aria-pressed={selectedNodeId === node.id}
            on:keydown={e => activateNode(e, node)}
            style="cursor:pointer"
            on:pointerdown={e => onNodeMousedown(e, node)}
            on:click={e => onNodeClick(e, node)}
            on:dblclick={e => { e.stopPropagation(); if (!disabled && mode === 'select') dispatch('openIntersection', { nodeId: node.id }); }}
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

    <div class="absolute top-3 left-3 flex gap-1 rounded-lg bg-white p-1 shadow" aria-label="Canvas view">
      <button class="rounded px-3 py-1 hover:bg-gray-100" aria-label="Zoom out" disabled={disabled} on:click={() => zoomAt(1 / 1.15)}>-</button>
      <span class="self-center text-xs tabular-nums">{Math.round(scale * 100)}%</span>
      <button class="rounded px-3 py-1 hover:bg-gray-100" aria-label="Zoom in" disabled={disabled} on:click={() => zoomAt(1.15)}>+</button>
      <button class="rounded px-3 py-1 text-xs hover:bg-gray-100" disabled={disabled} on:click={fit}>Fit network</button>
    </div>

    <p id="canvas-keyboard-help" class="sr-only">Arrow keys pan the canvas. Plus and minus zoom. In Junction mode, Enter creates a junction at the center. Focus a junction and use arrows to move it, or Enter to select it. In Road mode, select two junctions with Enter to connect them.</p>

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
        Click another junction to finish, or background to cancel
      </div>
    {:else if mode === 'node'}
      <div class="absolute bottom-3 left-1/2 -translate-x-1/2 bg-blue-600 text-white text-xs px-3 py-1.5 rounded-full pointer-events-none shadow">
        Click on the canvas to place a junction
      </div>
    {/if}
  {/if}
</div>
