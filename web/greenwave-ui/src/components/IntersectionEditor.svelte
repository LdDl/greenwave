<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { fade, scale } from 'svelte/transition';
  import { networkNodes, networkEdges, intersectionConfigs, networkEditor } from '$lib/stores/network.js';

  import { intersectionTopology, groupConflictPairs, MAX_GREEN } from '$lib/utils/network-project.js';
  import { modalFocus } from '$lib/utils/modal-focus.js';

  export let nodeId = null;

  const dispatch = createEventDispatcher();

  // Visual constants
  // px per lane
  const LANE_WIDTH = 13;
  // minimum junction circle radius
  const MIN_JUNCTION_RADIUS = 56;
  // road stub length beyond junction edge
  const STUB_LENGTH = 90;
  // control point pull toward center
  const BEZIER_TENSION = 65;
  // arrowhead triangle side
  const ARROW_SIZE = 9;
  // lane-count badge circle
  const BADGE_RADIUS = 10;
  // label distance beyond stub outer end
  const LABEL_OFFSET = 22;
  // white stop line stroke
  const STOP_LINE_WIDTH = 2.5;
  // square viewport
  const SVG_SIZE = 480;
  const CENTER = SVG_SIZE / 2;

  const GROUP_COLORS = ['#3b82f6', '#f97316', '#8b5cf6', '#14b8a6', '#ec4899'];
  const ROAD_COLOR = '#334155';
  // slightly lighter for inbound band
  const IN_TINT = '#3b4f6b';
  // slightly darker for outbound band
  const OUT_TINT = '#2d3d50';

  // State
  let node = null;
  let stubs = [];
  let movements = [];
  let movState = {};
  let groups = [];
  let selMovId = null;
  let showAllPills = false;
  let original = '';
  let discard = false;
  let saveError = '';
  let conflictSelection = [];
  let past = [];
  let future = [];

  function snapshot() { return JSON.stringify({ groups, movState }); }
  function remember() { past = [...past.slice(-99), snapshot()]; future = []; discard = false; }
  function restore(value) { const draft = JSON.parse(value); groups = draft.groups; movState = draft.movState; }
  function undoDraft() { if (!past.length) return; future = [...future, snapshot()]; restore(past[past.length - 1]); past = past.slice(0, -1); }
  function redoDraft() { if (!future.length) return; past = [...past, snapshot()]; restore(future[future.length - 1]); future = future.slice(0, -1); }
  $: dirty = JSON.stringify({ groups, movState }) !== original;
  $: invalidGreen = groups.some(g => !Number.isInteger(g.greenDuration) || g.greenDuration < 1 || g.greenDuration > MAX_GREEN);
  $: forbiddenCount = movements.filter(m => !movState[m.id]?.groupIds.length).length;
  $: conflictCount = Object.values(groupConflicts).reduce((sum, pairs) => sum + pairs.length, 0);

  function requestClose() {
    if (dirty) discard = true;
    else dispatch('close');
  }
  function dialogKeys(event) {
    if (event.key === 'Escape') { event.stopPropagation(); event.preventDefault(); if (discard) discard = false; else requestClose(); }
    if (event.target.closest('input, select, textarea')) return;
    if ((event.ctrlKey || event.metaKey) && ['z', 'y'].includes(event.key.toLowerCase())) {
      event.preventDefault(); event.stopPropagation();
      if (event.shiftKey || event.key.toLowerCase() === 'y') redoDraft(); else undoDraft();
    }
  }
  function beforeUnload(event) { if (dirty) { event.preventDefault(); event.returnValue = ''; } }


  // Zoom & pan (viewBox-based: SVG fills entire panel)
  const ZOOM_MIN = 0.3;
  const ZOOM_MAX = 3.0;
  const ZOOM_STEP = 0.15;
  let zoom = 1;
  let panX = 0;
  let panY = 0;
  let dragging = false;
  let dragStart = { x: 0, y: 0 };
  let panStart = { x: 0, y: 0 };
  let svgEl = null;

  const PAN_STEP = 30;
  function zoomIn() {
    zoom = Math.min(ZOOM_MAX, zoom + ZOOM_STEP);
  }
  function zoomOut() {
    zoom = Math.max(ZOOM_MIN, zoom - ZOOM_STEP);
  }
  function zoomReset() {
    zoom = 1;
    panX = 0;
    panY = 0;
  }
  function panLeft() {
    panX += PAN_STEP / zoom;
  }
  function panRight() {
    panX -= PAN_STEP / zoom;
  }
  function panUp() {
    panY += PAN_STEP / zoom;
  }
  function panDown() {
    panY -= PAN_STEP / zoom;
  }

  function handleWheel(e) {
    e.preventDefault();
    if (e.deltaY < 0) zoomIn(); else zoomOut();
  }

  // Pan converts screen-px delta to SVG-units delta (depends on current zoom & element size)
  function screenToSvgScale() {
    if (!svgEl) return 1;
    const rect = svgEl.getBoundingClientRect();
    return (SVG_SIZE / zoom) / Math.min(rect.width, rect.height);
  }

  const DRAG_THRESHOLD = 4;
  let pointerDown = false;
  let pointerMoved = false;

  function onPointerDown(e) {
    if (e.button !== 0) return;
    pointerDown = true;
    pointerMoved = false;
    dragStart = { x: e.clientX, y: e.clientY };
    panStart = { x: panX, y: panY };
  }
  function onPointerMove(e) {
    if (!pointerDown) return;
    const dx = e.clientX - dragStart.x;
    const dy = e.clientY - dragStart.y;
    if (!pointerMoved && Math.abs(dx) + Math.abs(dy) > DRAG_THRESHOLD) {
      pointerMoved = true;
      dragging = true;
      svgEl.setPointerCapture(e.pointerId);
    }
    if (dragging) {
      const s = screenToSvgScale();
      panX = panStart.x + dx * s;
      panY = panStart.y + dy * s;
    }
  }
  function onPointerUp() {
    pointerDown = false;
    dragging = false;
  }

  function onSvgClick() {
    if (pointerMoved) return;
    // Arc hit areas use stopPropagation, so any click reaching here is on background
    selMovId = null;
  }

  // Reactive viewBox: zoom shrinks the visible area, pan shifts it
  $: vbSize = SVG_SIZE / zoom;
  $: vbX = (SVG_SIZE - vbSize) / 2 - panX;
  $: vbY = (SVG_SIZE - vbSize) / 2 - panY;
  $: viewBox = `${f(vbX)} ${f(vbY)} ${f(vbSize)} ${f(vbSize)}`;

  onMount(() => {
    node = $networkNodes.find(n => n.id === nodeId) ?? null;
    if (node) init();
  });

  // Build stubs & movements from graph
  function init() {
    const topology = intersectionTopology(nodeId, $networkNodes, $networkEdges);
    stubs = topology.stubs;
    movements = topology.movements;
    const saved = $intersectionConfigs[nodeId];
    groups = saved ? JSON.parse(JSON.stringify(saved.groups)) : [{ id: 0, greenDuration: 30 }];
    movState = Object.fromEntries(movements.map(m => [m.id, { groupIds: [...(saved?.movState[m.id]?.groupIds ?? [])] }]));
    selMovId = null;
    original = snapshot();
    discard = false;
    past = [];
    future = [];
    conflictSelection = [];
  }

  // Geometry helpers

  function dir(angle) {
    return { dx: Math.cos(angle), dy: Math.sin(angle) };
  }

  // Unit perpendicular (left normal in SVG y-down coords).
  // Right-hand traffic: +perp = outbound side, -perp = inbound side.
  function perp(angle) {
    return { nx: -Math.sin(angle), ny: Math.cos(angle) };
  }

  function radialPt(angle, r) {
    return { x: CENTER + r * Math.cos(angle), y: CENTER + r * Math.sin(angle) };
  }

  function offsetPt(pt, angle, amount) {
    const { nx, ny } = perp(angle);
    return { x: pt.x + amount * nx, y: pt.y + amount * ny };
  }

  function halfWidth(stub) {
    return Math.max(1, stub.lanesIn + stub.lanesOut) * LANE_WIDTH / 2;
  }

  function f(v) { return (+v).toFixed(1); }

  // Minimum angular gap between two stubs at given radius
  const STUB_GAP_PX = 6;
  function minAngularGap(stubA, stubB, R) {
    const need = halfWidth(stubA) + halfWidth(stubB) + STUB_GAP_PX;
    return 2 * Math.asin(Math.min(1, need / (2 * R)));
  }

  // Adaptive junction radius
  const maxJunctionRadius = SVG_SIZE / 2 - STUB_LENGTH - LABEL_OFFSET - 10;
  $: junctionRadius = (() => {
    if (stubs.length < 2) return MIN_JUNCTION_RADIUS;
    const sorted = [...stubs].sort((a, b) => a.angle - b.angle);
    let bestR = MIN_JUNCTION_RADIUS;
    for (let i = 0; i < sorted.length; i++) {
      const a = sorted[i];
      const b = sorted[(i + 1) % sorted.length];
      let delta = b.angle - a.angle;
      if (delta <= 0) delta += Math.PI * 2;
      const sinHalf = Math.sin(delta / 2);
      if (sinHalf < 0.01) continue;
      const needed = (halfWidth(a) + halfWidth(b) + STUB_GAP_PX) / (2 * sinHalf);
      if (needed > bestR) bestR = needed;
    }
    return Math.min(bestR, maxJunctionRadius);
  })();

  // Spread stub angles so no pair overlaps
  $: displayStubs = (() => {
    if (stubs.length < 2) return stubs.map(s => ({ ...s, displayAngle: s.angle }));
    const R = junctionRadius;
    const sorted = [...stubs]
      .map(s => ({ ...s, displayAngle: s.angle }))
      .sort((a, b) => a.angle - b.angle);

    for (let pass = 0; pass < 5; pass++) {
      let changed = false;
      for (let i = 0; i < sorted.length; i++) {
        const a = sorted[i];
        const b = sorted[(i + 1) % sorted.length];
        let gap = b.displayAngle - a.displayAngle;
        if (gap <= 0) gap += Math.PI * 2;
        const minGap = minAngularGap(a, b, R);
        if (gap < minGap) {
          const push = (minGap - gap) / 2;
          a.displayAngle -= push;
          b.displayAngle += push;
          changed = true;
        }
      }
      if (!changed) break;
    }

    for (const s of sorted) {
      while (s.displayAngle > Math.PI) s.displayAngle -= Math.PI * 2;
      while (s.displayAngle < -Math.PI) s.displayAngle += Math.PI * 2;
    }
    return sorted;
  })();

  // Precomputed stub geometry
  $: stubGeo = displayStubs.map(s => {
    const a = s.displayAngle;
    const hw = halfWidth(s);
    const twoWay = s.lanesIn > 0 && s.lanesOut > 0;
    const leftOffset = twoWay ? -s.lanesIn * LANE_WIDTH : -hw;
    const rightOffset = twoWay ? s.lanesOut * LANE_WIDTH : hw;
    const inner = radialPt(a, junctionRadius);
    const outer = radialPt(a, junctionRadius + STUB_LENGTH);

    const corners = [
      offsetPt(inner, a, rightOffset),
      offsetPt(inner, a, leftOffset),
      offsetPt(outer, a, leftOffset),
      offsetPt(outer, a, rightOffset),
    ];
    const roadPath = corners.map((c, i) =>
      `${i === 0 ? 'M' : 'L'} ${f(c.x)},${f(c.y)}`
    ).join(' ') + ' Z';

    const lanesI = Math.max(1, s.lanesIn);
    const lanesO = Math.max(1, s.lanesOut);

    let inBandPath = null;
    let outBandPath = null;

    if (twoWay) {
      const inBand = [
        offsetPt(inner, a, 0),
        offsetPt(inner, a, leftOffset),
        offsetPt(outer, a, leftOffset),
        offsetPt(outer, a, 0),
      ];
      inBandPath = inBand.map((c, i) =>
        `${i === 0 ? 'M' : 'L'} ${f(c.x)},${f(c.y)}`
      ).join(' ') + ' Z';

      const outBand = [
        offsetPt(inner, a, 0),
        offsetPt(inner, a, rightOffset),
        offsetPt(outer, a, rightOffset),
        offsetPt(outer, a, 0),
      ];
      outBandPath = outBand.map((c, i) =>
        `${i === 0 ? 'M' : 'L'} ${f(c.x)},${f(c.y)}`
      ).join(' ') + ' Z';
    }

    const laneDividers = [];
    if (twoWay) {
      for (let i = 1; i < lanesI; i++) {
        const off = -(i * LANE_WIDTH);
        const p1 = offsetPt(inner, a, off);
        const p2 = offsetPt(outer, a, off);
        laneDividers.push({ x1: f(p1.x), y1: f(p1.y), x2: f(p2.x), y2: f(p2.y) });
      }
      for (let i = 1; i < lanesO; i++) {
        const off = i * LANE_WIDTH;
        const p1 = offsetPt(inner, a, off);
        const p2 = offsetPt(outer, a, off);
        laneDividers.push({ x1: f(p1.x), y1: f(p1.y), x2: f(p2.x), y2: f(p2.y) });
      }
    } else {
      const totalLanes = Math.max(lanesI, lanesO);
      for (let i = 1; i < totalLanes; i++) {
        const off = -hw + i * LANE_WIDTH;
        const p1 = offsetPt(inner, a, off);
        const p2 = offsetPt(outer, a, off);
        laneDividers.push({ x1: f(p1.x), y1: f(p1.y), x2: f(p2.x), y2: f(p2.y) });
      }
    }

    const divider = twoWay ? {
      x1: f(inner.x), y1: f(inner.y),
      x2: f(outer.x), y2: f(outer.y),
    } : null;

    const stopL = offsetPt(inner, a, twoWay ? 0 : rightOffset);
    const stopR = offsetPt(inner, a, leftOffset);
    const stopLine = {
      x1: f(stopL.x), y1: f(stopL.y),
      x2: f(stopR.x), y2: f(stopR.y),
    };

    const badge = radialPt(a, junctionRadius + STUB_LENGTH * 0.55);
    const labelPos = radialPt(a, junctionRadius + STUB_LENGTH + LABEL_OFFSET);

    let dirArrow = null;
    if (!twoWay) {
      const midR = junctionRadius + STUB_LENGTH * 0.35;
      const mid = radialPt(a, midR);
      const { dx, dy } = dir(a);
      const sign = s.lanesIn > 0 ? -1 : 1;
      const aLen = 8;
      const aW = 5;
      dirArrow = {
        x1: f(mid.x - sign * dx * aLen + aW * (-dy)),
        y1: f(mid.y - sign * dy * aLen + aW * dx),
        x2: f(mid.x + sign * dx * aLen),
        y2: f(mid.y + sign * dy * aLen),
        x3: f(mid.x - sign * dx * aLen - aW * (-dy)),
        y3: f(mid.y - sign * dy * aLen - aW * dx),
      };
    }

    return {
      ...s, hw, twoWay, roadPath, inBandPath, outBandPath,
      laneDividers, divider, stopLine, badge, labelPos, dirArrow,
    };
  });

  // Precomputed movement geometry
  function portPt(stub, inbound) {
    const da = stub.displayAngle ?? stub.angle;
    const inner = radialPt(da, junctionRadius);
    const lanesI = Math.max(1, stub.lanesIn);
    const lanesO = Math.max(1, stub.lanesOut);
    const halfIn = lanesI * LANE_WIDTH / 2;
    const halfOut = lanesO * LANE_WIDTH / 2;
    const offset = stub.lanesIn > 0 && stub.lanesOut > 0 ? (inbound ? -halfIn : halfOut) : 0;
    return offsetPt(inner, da, offset);
  }

  $: _ms = movState;
  $: movGeo = movements.map(mov => {
    const ms = _ms;
    const si = displayStubs.find(s => s.edgeId === mov.inEdgeId);
    const so = displayStubs.find(s => s.edgeId === mov.outEdgeId);
    if (!si || !so) return null;

    const p0 = portPt(si, true);
    const p1 = portPt(so, false);

    const pull = Math.min(BEZIER_TENSION, junctionRadius * 0.7);
    const incoming = dir(si.displayAngle ?? si.angle);
    const outgoing = dir(so.displayAngle ?? so.angle);
    const cp0 = { x: p0.x - incoming.dx * pull, y: p0.y - incoming.dy * pull };
    const cp1 = { x: p1.x - outgoing.dx * pull, y: p1.y - outgoing.dy * pull };

    const path = `M ${f(p0.x)},${f(p0.y)} C ${f(cp0.x)},${f(cp0.y)} ${f(cp1.x)},${f(cp1.y)} ${f(p1.x)},${f(p1.y)}`;

    const { dx, dy } = dir(so.displayAngle ?? so.angle);
    const tip  = { x: p1.x + dx * ARROW_SIZE, y: p1.y + dy * ARROW_SIZE };
    const left = { x: p1.x - dy * ARROW_SIZE * 0.5, y: p1.y + dx * ARROW_SIZE * 0.5 };
    const right= { x: p1.x + dy * ARROW_SIZE * 0.5, y: p1.y - dx * ARROW_SIZE * 0.5 };
    const arrow = `${f(tip.x)},${f(tip.y)} ${f(left.x)},${f(left.y)} ${f(right.x)},${f(right.y)}`;

    const gids = ms[mov.id]?.groupIds || [];
    const forbidden = gids.length === 0;
    const color = forbidden ? '#94a3b8' : gColor(gids[0]);

    return { ...mov, path, arrow, gids, color, forbidden };
  }).filter(Boolean);

  // Actions
  let hovMovId = null;
  let hovGroupId = null;

  function selectArc(movId) {
    if (pointerMoved) return;
    conflictSelection = [];
    selMovId = selMovId === movId ? null : movId;
  }

  function toggleGroup(movId, gid) {
    remember();
    const cur = movState[movId]?.groupIds || [];
    const has = cur.includes(gid);
    const next = has ? cur.filter(id => id !== gid) : [...cur, gid];
    movState = { ...movState, [movId]: { groupIds: next } };
  }

  function forbidMov(movId) {
    remember();
    movState = { ...movState, [movId]: { groupIds: [] } };
  }

  function addGroup() {
    remember();
    const maxId = groups.length ? Math.max(...groups.map(g => g.id)) : -1;
    groups = [...groups, { id: maxId + 1, greenDuration: 30 }];
  }

  function removeGroup(gid) {
    if (groups.length <= 1) return;
    remember();
    groups = groups.filter(g => g.id !== gid);
    const ns = { ...movState };
    for (const m of movements) {
      ns[m.id] = { groupIds: (ns[m.id]?.groupIds || []).filter(id => id !== gid) };
    }
    movState = ns;
  }

  function updGreen(gid, val) {
    remember();
    groups = groups.map(g => g.id === gid ? { ...g, greenDuration: Number(val) } : g);
  }

  function save() {
    if (invalidGreen) return;
    const config = {
      macro_node_id: nodeId,
      groups: groups.map(g => ({ id: g.id, greenDuration: g.greenDuration })),
      movState: { ...movState },
    };
    try {
      networkEditor.saveIntersection(nodeId, config);
      dispatch('close');
    } catch (error) { saveError = error.message; }
  }

  function handleBackdrop(e) {
    if (e.target === e.currentTarget) requestClose();
  }

  $: selMov = selMovId ? movements.find(m => m.id === selMovId) : null;
  $: if (!selMovId) showAllPills = false;
  $: pillMaxVisible = showAllPills ? groups.length : 4;
  $: pillVisible = groups.slice(0, pillMaxVisible);
  $: pillHiddenCount = groups.length - pillMaxVisible;

  function gColor(id) { return GROUP_COLORS[id % GROUP_COLORS.length]; }

  // Reactive map: groupId -> assigned movements
  $: assignedByGroup = (() => {
    const ms = _ms;
    const result = {};
    for (const g of groups) {
      result[g.id] = movements.filter(m => (ms[m.id]?.groupIds || []).includes(g.id));
    }
    return result;
  })();

  // Determine turn type from angle difference
  function turnType(mov) {
    const si = displayStubs.find(s => s.edgeId === mov.inEdgeId);
    const so = displayStubs.find(s => s.edgeId === mov.outEdgeId);
    if (!si || !so) return '';
    const inFacing = (si.displayAngle ?? si.angle) + Math.PI;
    const outDir = so.displayAngle ?? so.angle;
    let diff = outDir - inFacing;
    while (diff > Math.PI) diff -= 2 * Math.PI;
    while (diff <= -Math.PI) diff += 2 * Math.PI;
    const deg = Math.abs(diff * 180 / Math.PI);
    if (deg < 30) return 'straight';
    if (diff > 0) return 'right';
    return 'left';
  }

  const TURN_ICONS = { left: '\u2B9C', straight: '\u2B9D', right: '\u2B9E' };

  $: groupConflicts = groupConflictPairs(movements, movState, groups, stubs);

  function selectConflict(pair) {
    conflictSelection = pair.map(m => m.id);
    selMovId = pair[0].id;
  }
</script>

<svelte:window on:beforeunload={beforeUnload} />

{#if nodeId != null && node}
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
  transition:fade={{ duration: 150 }}
  class="fixed inset-0 z-50 flex items-center justify-center"
  style="background-color: rgba(0,0,0,0.55)"
  on:click={handleBackdrop}
  on:keydown={dialogKeys}
  use:modalFocus
  role="dialog" aria-modal="true" aria-labelledby="intersection-title" tabindex="-1"
>
  <div
    transition:scale={{ start: 0.97, duration: 150 }}
    class="bg-white rounded-xl shadow-xl w-full max-w-6xl mx-4 flex flex-col overflow-hidden"
    style="height: 92vh"
  >
    <!-- Header -->
    <div class="flex items-center gap-3 px-6 py-4 border-b border-gray-200 shrink-0">
      <svg class="w-4 h-4 text-gray-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M8 9l4-4 4 4m0 6l-4 4-4-4"/>
      </svg>
      <h2 id="intersection-title" class="text-sm font-semibold text-gray-800">{node.label}: Intersection</h2>
      <span class="hidden md:block text-xs text-gray-400">Click arc to select · assign group or mark forbidden</span>
      <button
        class="ml-auto p-1.5 text-gray-400 hover:text-gray-600 rounded-md hover:bg-gray-100
               focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
        on:click={requestClose}
        aria-label="Close"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
        </svg>
      </button>
    </div>

    <!-- Body -->
    <div class="flex flex-1 min-h-0 flex-col md:flex-row">

      <!-- Left: SVG road diagram -->
      <div class="flex-1 min-h-[220px] relative overflow-hidden" style="background:#e2e8f0">
        {#if stubs.length === 0}
          <div class="absolute inset-0 flex flex-col items-center justify-center gap-3 text-gray-400">
            <svg class="w-10 h-10 opacity-25" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6"/>
            </svg>
            <p class="text-xs text-center leading-snug max-w-[180px]">
              No roads connected.<br/>Draw road segments to this junction first.
            </p>
          </div>
        {:else}
          <!-- Zoom & pan toolbar -->
          {@const tbtn = "w-7 h-7 flex items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"}
          <div class="absolute top-3 left-3 z-10 flex flex-col gap-1.5">
            <div class="flex items-center gap-0.5 bg-white/90 rounded-lg shadow-sm border border-gray-200 px-1 py-0.5">
              <button on:click={zoomOut} title="Zoom out" class="{tbtn} text-base font-bold">-</button>
              <button on:click={zoomReset} title="Reset"
                class="px-1.5 h-7 flex items-center justify-center rounded-md text-xs text-gray-500 hover:bg-gray-100 hover:text-gray-700 tabular-nums
                       focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400 font-medium min-w-[3rem]"
              >{Math.round(zoom * 100)}%</button>
              <button on:click={zoomIn} title="Zoom in" class="{tbtn} text-base font-bold">+</button>
            </div>
            <div class="flex items-center justify-center gap-0.5 bg-white/90 rounded-lg shadow-sm border border-gray-200 px-0.5 py-0.5">
              <button on:click={panLeft} title="Pan left" aria-label="Pan left" class={tbtn}>
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7"/></svg>
              </button>
              <div class="flex flex-col gap-0.5">
                <button on:click={panUp} title="Pan up" aria-label="Pan up" class={tbtn}>
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7"/></svg>
                </button>
                <button on:click={panDown} title="Pan down" aria-label="Pan down" class={tbtn}>
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7"/></svg>
                </button>
              </div>
              <button on:click={panRight} title="Pan right" aria-label="Pan right" class={tbtn}>
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7"/></svg>
              </button>
            </div>
          </div>

          <svg
            bind:this={svgEl}
            viewBox={viewBox}
            preserveAspectRatio="xMidYMid meet"
            class="absolute inset-0 w-full h-full select-none"
            style="cursor: {dragging ? 'grabbing' : 'grab'}; touch-action:none"
            role="application" aria-label="Intersection movements" tabindex="-1"
            on:keydown={e => { if (e.key === 'Escape') selMovId = null; }}
            on:wheel|nonpassive={handleWheel}
            on:pointerdown={onPointerDown}
            on:pointermove={onPointerMove}
            on:pointerup={onPointerUp}
            on:pointercancel={onPointerUp}
            on:click={onSvgClick}
          >

            <!-- Junction box -->
            <circle cx={CENTER} cy={CENTER} r={junctionRadius + 2} fill={ROAD_COLOR}/>

            <!-- Road stubs -->
            {#each stubGeo as sg (sg.edgeId)}
              {#if sg.twoWay}
                <path d={sg.inBandPath} fill={IN_TINT}/>
                <path d={sg.outBandPath} fill={OUT_TINT}/>
              {:else}
                <path d={sg.roadPath} fill={sg.lanesIn > 0 ? IN_TINT : OUT_TINT}/>
              {/if}

              {#each sg.laneDividers as ld, index (index)}
                <line x1={ld.x1} y1={ld.y1} x2={ld.x2} y2={ld.y2}
                  stroke="white" stroke-width="0.8"
                  stroke-dasharray="4,3" opacity="0.6"
                  pointer-events="none"/>
              {/each}

              {#if sg.divider}
                <line x1={sg.divider.x1} y1={sg.divider.y1}
                      x2={sg.divider.x2} y2={sg.divider.y2}
                  stroke="#fbbf24" stroke-width="1.5"
                  stroke-dasharray="7,5" opacity="0.7"
                  pointer-events="none"/>
              {/if}

              {#if sg.dirArrow}
                <polyline
                  points="{sg.dirArrow.x1},{sg.dirArrow.y1} {sg.dirArrow.x2},{sg.dirArrow.y2} {sg.dirArrow.x3},{sg.dirArrow.y3}"
                  fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                  opacity="0.7" pointer-events="none"/>
              {/if}

              {#if sg.lanesIn > 0}
              <line x1={sg.stopLine.x1} y1={sg.stopLine.y1}
                    x2={sg.stopLine.x2} y2={sg.stopLine.y2}
                stroke="white" stroke-width={STOP_LINE_WIDTH} opacity="0.85"
                pointer-events="none"/>
              {/if}

              <circle cx={sg.badge.x} cy={sg.badge.y} r={BADGE_RADIUS}
                fill="white" stroke="#94a3b8" stroke-width="1"/>
              <text x={sg.badge.x} y={sg.badge.y}
                text-anchor="middle" dominant-baseline="central"
                font-size="9" font-weight="700" fill="#334155"
              >{#if sg.twoWay}{sg.lanesIn}+{sg.lanesOut}{:else}{sg.lanesIn || sg.lanesOut}{/if}</text>

              <text x={sg.labelPos.x} y={sg.labelPos.y}
                text-anchor="middle" dominant-baseline="central"
                font-size="12" font-weight="700" fill="#1e293b"
              >{sg.label}</text>
            {/each}

            <!-- Movement arcs: hit areas -->
            {#each movGeo as mg (mg.id)}
              <path d={mg.path} fill="none" stroke="transparent" stroke-width="16"
                style="cursor:pointer" role="button" tabindex="0" aria-label={`${mg.inLabel} to ${mg.outLabel}`} aria-pressed={selMovId === mg.id}
                on:keydown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pointerMoved = false; selectArc(mg.id); } }}
                on:click|stopPropagation={() => selectArc(mg.id)}
                on:pointerenter={() => hovMovId = mg.id}
                on:pointerleave={() => { if (hovMovId === mg.id) hovMovId = null; }}/>
            {/each}

            <!-- Movement arcs: visible -->
            {#each movGeo as mg (mg.id)}
              {#if conflictSelection.includes(mg.id) || selMovId === mg.id || hovMovId === mg.id || (hovGroupId !== null && mg.gids.includes(hovGroupId))}
                <path d={mg.path} fill="none"
                  stroke={conflictSelection.includes(mg.id) ? '#dc2626' : mg.color} stroke-width="12"
                  opacity={selMovId === mg.id ? 0.18 : 0.1}
                  pointer-events="none"/>
              {/if}
              <path d={mg.path} fill="none"
                stroke={conflictSelection.includes(mg.id) ? '#dc2626' : mg.color}
                stroke-width={selMovId === mg.id ? 4.5 : 3}
                stroke-dasharray={mg.forbidden ? '5,4' : 'none'}
                opacity={mg.forbidden ? 0.45 : 1}
                stroke-linecap="round"
                pointer-events="none"/>
              <polygon points={mg.arrow} fill={mg.color}
                opacity={mg.forbidden ? 0.45 : 1} pointer-events="none"/>
            {/each}

            <circle cx={CENTER} cy={CENTER} r="3" fill="white" opacity="0.4"/>
          </svg>
        {/if}
      </div>

      <!-- Right panel -->
      <div class="w-full md:w-80 max-h-[45vh] md:max-h-none shrink-0 overflow-y-auto md:overflow-visible border-l border-gray-200 bg-gray-50/40 flex flex-col min-h-0" role="region" aria-label="Intersection settings">

        <!-- Fixed top section -->
        <div class="p-4 pb-0 space-y-4 shrink-0">
          <label for="movement-select" class="block text-xs font-medium text-gray-600">Movements ({movements.length})</label>
          <select id="movement-select" class="w-full rounded border border-gray-300 bg-white p-2 text-sm" value={selMovId ?? ''}
            on:change={e => { selMovId = e.target.value || null; conflictSelection = []; }}>
            <option value="">Select a movement</option>
            {#each movements as movement (movement.id)}
              <option value={movement.id}>{movement.inLabel} → {movement.outLabel}{!movState[movement.id]?.groupIds.length ? ' (forbidden)' : ''}</option>
            {/each}
          </select>
          {#if !movements.length}<p class="text-xs text-gray-500">No through or turning movements. Connect another road with an available incoming or outgoing direction.</p>{/if}
          <!-- Selected movement -->
          <div class="rounded-lg border border-gray-200 bg-white p-3 shrink-0 flex flex-col"
          >
            {#if selMov}
              {@const gids = movState[selMov.id]?.groupIds || []}
              {@const turn = turnType(selMov)}
              <div class="space-y-2.5">
                <div class="flex items-center justify-between">
                  <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Selected movement</p>
                  {#if turn}
                    <span class="text-xs text-gray-400 capitalize">{TURN_ICONS[turn]} {turn}</span>
                  {/if}
                </div>
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="px-2 py-0.5 rounded-md text-xs font-semibold bg-blue-50 text-blue-700 border border-blue-200">
                    {selMov.inLabel}
                  </span>
                  <svg class="w-3 h-3 text-gray-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                  </svg>
                  <span class="px-2 py-0.5 rounded-md text-xs font-semibold bg-orange-50 text-orange-700 border border-orange-200">
                    {selMov.outLabel}
                  </span>
                </div>
                <div class="space-y-1.5">
                  <button
                    on:click={() => forbidMov(selMov.id)}
                    class="px-2 py-0.5 text-xs rounded-md font-medium border transition-colors"
                    style="{gids.length === 0
                      ? 'background:#fef2f2; color:#dc2626; border-color:#fecaca'
                      : 'background:white; color:#9ca3af; border-color:#e5e7eb'}"
                  >Forbidden</button>
                  <div class="flex flex-wrap gap-1">
                    {#each pillVisible as g (g.id)}
                      {@const active = gids.includes(g.id)}
                      <button
                        on:click={() => toggleGroup(selMov.id, g.id)}
                        class="px-2 py-0.5 text-xs rounded-md font-medium border transition-colors"
                        style="{active
                          ? `background:${gColor(g.id)}; color:white; border-color:${gColor(g.id)}`
                          : `color:${gColor(g.id)}; border-color:${gColor(g.id)}44; background:white`}"
                      >G{g.id}</button>
                    {/each}
                    {#if pillHiddenCount > 0}
                      <button
                        on:click={() => showAllPills = true}
                        class="px-2 py-0.5 text-xs rounded-md font-medium border border-gray-200 text-gray-400 hover:text-gray-600 hover:bg-gray-50"
                      >+{pillHiddenCount} more</button>
                    {/if}
                    {#if showAllPills && groups.length > 4}
                      <button
                        on:click={() => showAllPills = false}
                        class="px-2 py-0.5 text-xs rounded-md font-medium border border-gray-200 text-gray-400 hover:text-gray-600 hover:bg-gray-50"
                      >less</button>
                    {/if}
                  </div>
                </div>
              </div>
            {:else}
              <div class="h-[104px] flex items-center justify-center">
                <p class="text-xs text-gray-400 italic leading-snug text-center">
                  Click any arc to select it,<br/>
                  then assign a group or mark forbidden.
                </p>
              </div>
            {/if}
          </div>

          <!-- Signal groups header -->
          <div class="flex items-center justify-between">
            <p class="text-xs font-semibold text-gray-400 uppercase tracking-wide">Movement groups</p>
            <button
              on:click={addGroup}
              class="text-xs px-2 py-1 rounded-md bg-white border border-gray-200 hover:bg-gray-100
                     text-gray-600 font-medium focus-visible:outline-none focus-visible:ring-2
                     focus-visible:ring-gray-400"
            >+ Add</button>
          </div>
        </div>

        <!-- Scrollable groups list -->
        <div class="shrink-0 md:flex-1 md:shrink md:min-h-0 md:overflow-y-auto px-4 py-2">
          {#if groups.length === 0}
            <p class="text-xs text-gray-400 text-center py-6 leading-snug">
              Add a group to assign movements
            </p>
          {:else}
            <div class="space-y-3">
              {#each groups as g (g.id)}
                {@const assigned = assignedByGroup[g.id] || []}
                <div class="rounded-lg border bg-white p-3 space-y-2.5"
                  style="border-color: {gColor(g.id)}55"
                  class:ring-2={selMovId && (movState[selMovId]?.groupIds || []).includes(g.id)}
                  class:ring-blue-400={selMovId && (movState[selMovId]?.groupIds || []).includes(g.id)}
                  on:pointerenter={() => hovGroupId = g.id}
                  on:pointerleave={() => { if (hovGroupId === g.id) hovGroupId = null; }}
                >
                  <div class="flex items-center gap-2 h-6">
                    <div class="w-3 h-3 rounded-full shrink-0" style="background:{gColor(g.id)}"></div>
                    <span class="text-sm font-semibold text-gray-700">Group {g.id}</span>
                    <div class="ml-auto">
                      {#if selMovId}
                        {@const inGroup = (movState[selMovId]?.groupIds || []).includes(g.id)}
                        <button
                          on:click={() => toggleGroup(selMovId, g.id)}
                          class="px-2 py-0.5 text-xs rounded-md font-medium border"
                          style="{inGroup
                            ? `background:${gColor(g.id)}; color:white; border-color:${gColor(g.id)}`
                            : `color:${gColor(g.id)}; border-color:${gColor(g.id)}55`}"
                        >{inGroup ? 'Remove' : 'Assign'}</button>
                      {/if}
                      {#if groups.length > 1}
                        <button
                          on:click={() => removeGroup(g.id)}
                          class="p-0.5 text-gray-400 hover:text-red-500 rounded
                                 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400"
                          title="Remove group" aria-label={`Remove group ${g.id}`}
                        >
                          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                              d="M6 18L18 6M6 6l12 12"/>
                          </svg>
                        </button>
                      {/if}
                    </div>
                  </div>

                  <div class="flex items-center gap-2">
                    <span class="text-xs text-gray-500 shrink-0">Green</span>
                    <input
                      type="number" min="1" max={MAX_GREEN} step="1" aria-label={`Green duration for group ${g.id}`}
                      value={g.greenDuration}
                      on:change={e => updGreen(g.id, e.target.value)}
                      class="w-16 px-2 py-1 text-xs border border-gray-200 rounded-md bg-white
                             focus:outline-none focus:ring-2 focus:ring-blue-400"
                    />
                    <span class="text-xs text-gray-400">s</span>
                  </div>

                  <div class="text-xs text-gray-500 leading-snug">
                    {#if assigned.length === 0}
                      <span class="italic text-gray-400">No movements assigned</span>
                    {:else}
                      {#each assigned as movement (movement.id)}
                        <button class="block w-full rounded px-1 py-0.5 text-left hover:bg-blue-50" on:click={() => { selMovId = movement.id; conflictSelection = []; }}>
                          {TURN_ICONS[turnType(movement)] || ''} {movement.inLabel} → {movement.outLabel}
                        </button>
                      {/each}
                    {/if}
                  </div>
                  {#if groupConflicts[g.id]?.length}
                    <p class="text-xs text-amber-600 flex items-center gap-1">
                      <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"/>
                      </svg>
                      {groupConflicts[g.id].length} possible conflicts
                    </p>
                    <div class="space-y-1">
                      {#each groupConflicts[g.id] as pair (`${pair[0].id}:${pair[1].id}`)}
                        <button class="block w-full rounded bg-amber-50 p-1.5 text-left text-xs text-amber-800 hover:bg-amber-100" on:click={() => selectConflict(pair)}>
                          {pair[0].inLabel} → {pair[0].outLabel} / {pair[1].inLabel} → {pair[1].outLabel}
                        </button>
                      {/each}
                    </div>
                  {:else if assigned.length}
                    <p class="text-xs text-green-600 flex items-center gap-1">
                      <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                      </svg>
                      No conflicts detected
                    </p>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Fixed bottom hint -->
        <div class="p-4 pt-2 shrink-0 border-t border-gray-100">
          <p class="text-xs text-gray-400 leading-snug">
            A group allows its movements together. A movement may belong to several groups. Dashed movements are forbidden. Conflict checks use approach geometry and shared exits.
          </p>
        </div>
      </div>
    </div>

    <div class="border-t border-gray-200 px-4 py-3 shrink-0">
      {#if invalidGreen || saveError}<p class="mb-2 text-sm text-red-600" role="alert">{saveError || `Green duration must be a whole number from 1 to ${MAX_GREEN} seconds.`}</p>{/if}
      {#if discard}
        <div class="flex flex-wrap items-center justify-end gap-2" role="alert">
          <p class="mr-auto text-sm text-amber-800">Discard unsaved intersection changes?</p>
          <button class="rounded border px-3 py-2 text-sm" on:click={() => discard = false}>Keep editing</button>
          <button class="rounded bg-red-600 px-3 py-2 text-sm text-white" on:click={() => dispatch('close')}>Discard changes</button>
        </div>
      {:else}
        <div class="flex flex-wrap items-center gap-2">
          <button class="rounded border px-3 py-2 text-sm disabled:opacity-40" disabled={!past.length} on:click={undoDraft}>Undo</button>
          <button class="rounded border px-3 py-2 text-sm disabled:opacity-40" disabled={!future.length} on:click={redoDraft}>Redo</button>
          <p class="mr-auto text-xs text-gray-500">{forbiddenCount} forbidden · {conflictCount} possible conflicts{dirty ? ' · Unsaved changes' : ''}</p>
          <button class="rounded bg-gray-100 px-3 py-2 text-sm" on:click={requestClose}>Cancel</button>
          <button class="rounded bg-blue-600 px-3 py-2 text-sm text-white disabled:opacity-40" disabled={invalidGreen} on:click={save}>Save configuration</button>
        </div>
      {/if}
    </div>
  </div>
</div>
{/if}
