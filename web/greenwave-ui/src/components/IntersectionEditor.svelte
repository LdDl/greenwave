<script>
  import { createEventDispatcher } from 'svelte';
  import { fade, scale } from 'svelte/transition';
  import { networkNodes, networkEdges, intersectionConfigs } from '$lib/stores/network.js';

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
  const GROUP_BG = ['#eff6ff', '#fff7ed', '#f5f3ff', '#f0fdf9', '#fdf2f8'];
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
    return (SVG_SIZE / zoom) / rect.width;
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

  function onSvgClick(e) {
    if (pointerMoved) return;
    // Only deselect if click was on SVG background (not on an arc hit area)
    if (e.target === svgEl || e.target.tagName === 'rect' || e.target.tagName === 'circle') {
      selMovId = null;
    }
  }

  // Reactive viewBox: zoom shrinks the visible area, pan shifts it
  $: vbSize = SVG_SIZE / zoom;
  $: vbX = (SVG_SIZE - vbSize) / 2 - panX;
  $: vbY = (SVG_SIZE - vbSize) / 2 - panY;
  $: viewBox = `${f(vbX)} ${f(vbY)} ${f(vbSize)} ${f(vbSize)}`;

  // Re-init when nodeId changes OR when edge data changes (e.g. lanes, reverse)
  $: if (nodeId != null && $networkEdges) {
    node = $networkNodes.find(n => n.id === nodeId) ?? null;
    if (node) {
      init();
      zoom = 1;
      panX = 0;
      panY = 0;
    }
  }

  // Build stubs & movements from graph
  function init() {
    const ns = $networkNodes;
    const es = $networkEdges;
    const sm = {};

    for (const e of es) {
      if (e.from !== nodeId && e.to !== nodeId) continue;
      const otherId = e.from === nodeId ? e.to : e.from;
      const other = ns.find(n => n.id === otherId);
      if (!other) continue;

      const angle = Math.atan2(other.y - node.y, other.x - node.x);
      if (!sm[e.id]) {
        sm[e.id] = {
          edgeId: e.id, angle, label: other.label ?? `J${otherId}`,
          lanesIn: 0, lanesOut: 0,
          hasInbound: false, inDir: null,
          hasOutbound: false, outDir: null,
        };
      }
      const s = sm[e.id];

      // Inbound = traffic arriving at this junction; Outbound = traffic leaving
      if (e.to === nodeId && e.lanes_fwd  > 0) { s.hasInbound  = true; s.inDir  = 'fwd';  s.lanesIn  = e.lanes_fwd;  }
      if (e.to === nodeId && e.lanes_back > 0) { s.hasOutbound = true; s.outDir = 'back'; s.lanesOut = e.lanes_back; }
      if (e.from === nodeId && e.lanes_fwd  > 0) { s.hasOutbound = true; s.outDir = 'fwd';  s.lanesOut = e.lanes_fwd;  }
      if (e.from === nodeId && e.lanes_back > 0) { s.hasInbound  = true; s.inDir  = 'back'; s.lanesIn  = e.lanes_back; }
    }
    stubs = Object.values(sm).filter(s => s.lanesIn > 0 || s.lanesOut > 0);

    const inStubs  = stubs.filter(s => s.hasInbound);
    const outStubs = stubs.filter(s => s.hasOutbound);
    movements = [];
    for (const src of inStubs) {
      for (const dst of outStubs) {
        if (src.edgeId === dst.edgeId) continue; // no U-turn on same edge
        movements.push({
          id: `${src.edgeId}_${src.inDir}_${dst.edgeId}_${dst.outDir}`,
          inEdgeId: src.edgeId, inDir: src.inDir,
          outEdgeId: dst.edgeId, outDir: dst.outDir,
          inLabel: src.label, outLabel: dst.label,
        });
      }
    }

    const saved = $intersectionConfigs[nodeId];
    if (saved) {
      groups = JSON.parse(JSON.stringify(saved.groups));
      movState = {};
      for (const m of movements) {
        movState[m.id] = saved.movState?.[m.id]
          ? { ...saved.movState[m.id] }
          : { groupId: groups[0]?.id ?? null };
      }
    } else {
      groups = [{ id: 0, greenDuration: 30 }];
      movState = Object.fromEntries(movements.map(m => [m.id, { groupId: 0 }]));
    }
    selMovId = null;
  }

  // Geometry helpers

  /** Unit direction vector pointing away from junction along stub */
  function dir(angle) {
    return { dx: Math.cos(angle), dy: Math.sin(angle) };
  }

  // Unit perpendicular (left normal in SVG y-down coords).
  // Right-hand traffic: +perp = outbound side, -perp = inbound side.
  function perp(angle) {
    return { nx: -Math.sin(angle), ny: Math.cos(angle) };
  }

  /** Point at distance `r` from center along stub angle */
  function radialPt(angle, r) {
    return { x: CENTER + r * Math.cos(angle), y: CENTER + r * Math.sin(angle) };
  }

  /** Offset a point by `amount` along perpendicular */
  function offsetPt(pt, angle, amount) {
    const { nx, ny } = perp(angle);
    return { x: pt.x + amount * nx, y: pt.y + amount * ny };
  }

  /** Total half-width of stub road (in pixels) */
  function halfWidth(stub) {
    return Math.max(1, stub.lanesIn + stub.lanesOut) * LANE_WIDTH / 2;
  }

  /** Format number to 1 decimal for SVG attributes */
  function f(v) { return (+v).toFixed(1); }

  // Minimum angular gap between two stubs at given radius
  const STUB_GAP_PX = 6;
  function minAngularGap(stubA, stubB, R) {
    const need = halfWidth(stubA) + halfWidth(stubB) + STUB_GAP_PX;
    return 2 * Math.asin(Math.min(1, need / (2 * R)));
  }

  // Step 1: adaptive junction radius (grow R so stubs fit, capped by viewport)
  $: maxJunctionRadius = SVG_SIZE / 2 - STUB_LENGTH - LABEL_OFFSET - 10;
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

  // Step 2: spread stub angles so no pair overlaps at junctionRadius.
  // Preserves angular order; pushes apart only where needed.
  $: displayStubs = (() => {
    if (stubs.length < 2) return stubs.map(s => ({ ...s, displayAngle: s.angle }));
    const R = junctionRadius;
    const sorted = [...stubs]
      .map(s => ({ ...s, displayAngle: s.angle }))
      .sort((a, b) => a.angle - b.angle);

    // Iterative relaxation: push overlapping neighbours apart (few passes suffice)
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

    // Normalize angles to [-PI, PI]
    for (const s of sorted) {
      while (s.displayAngle > Math.PI) s.displayAngle -= Math.PI * 2;
      while (s.displayAngle < -Math.PI) s.displayAngle += Math.PI * 2;
    }
    return sorted;
  })();

  // Precomputed stub geometry (uses displayAngle for layout, keeps original angle for data)
  $: stubGeo = displayStubs.map(s => {
    const a = s.displayAngle;
    const hw = halfWidth(s);
    const inner = radialPt(a, junctionRadius);
    const outer = radialPt(a, junctionRadius + STUB_LENGTH);

    // Four corners of the road rectangle
    const corners = [
      offsetPt(inner, a, +hw),
      offsetPt(inner, a, -hw),
      offsetPt(outer, a, -hw),
      offsetPt(outer, a, +hw),
    ];
    const roadPath = corners.map((c, i) =>
      `${i === 0 ? 'M' : 'L'} ${f(c.x)},${f(c.y)}`
    ).join(' ') + ' Z';

    const twoWay = s.lanesIn > 0 && s.lanesOut > 0;
    const lanesI = Math.max(1, s.lanesIn);
    const lanesO = Math.max(1, s.lanesOut);

    // For one-way roads, the single band covers full width
    // For two-way, split into inbound (perp-) and outbound (perp+) halves
    let inBandPath = null;
    let outBandPath = null;

    if (twoWay) {
      const inBand = [
        offsetPt(inner, a, 0),
        offsetPt(inner, a, -hw),
        offsetPt(outer, a, -hw),
        offsetPt(outer, a, 0),
      ];
      inBandPath = inBand.map((c, i) =>
        `${i === 0 ? 'M' : 'L'} ${f(c.x)},${f(c.y)}`
      ).join(' ') + ' Z';

      const outBand = [
        offsetPt(inner, a, 0),
        offsetPt(inner, a, +hw),
        offsetPt(outer, a, +hw),
        offsetPt(outer, a, 0),
      ];
      outBandPath = outBand.map((c, i) =>
        `${i === 0 ? 'M' : 'L'} ${f(c.x)},${f(c.y)}`
      ).join(' ') + ' Z';
    }

    // Lane divider lines (white dashed, between lanes of same direction)
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
      // One-way: dividers span full road width from edge
      const totalLanes = Math.max(lanesI, lanesO);
      for (let i = 1; i < totalLanes; i++) {
        const off = -hw + i * LANE_WIDTH;
        const p1 = offsetPt(inner, a, off);
        const p2 = offsetPt(outer, a, off);
        laneDividers.push({ x1: f(p1.x), y1: f(p1.y), x2: f(p2.x), y2: f(p2.y) });
      }
    }

    // Center divider (yellow, only for two-way roads)
    const divider = twoWay ? {
      x1: f(inner.x), y1: f(inner.y),
      x2: f(outer.x), y2: f(outer.y),
    } : null;

    // Stop line at junction edge
    const stopL = offsetPt(inner, a, +hw);
    const stopR = offsetPt(inner, a, -hw);
    const stopLine = {
      x1: f(stopL.x), y1: f(stopL.y),
      x2: f(stopR.x), y2: f(stopR.y),
    };

    // Badge position (midway along stub)
    const badge = radialPt(a, junctionRadius + STUB_LENGTH * 0.55);

    // Label position (beyond outer end)
    const labelPos = radialPt(a, junctionRadius + STUB_LENGTH + LABEL_OFFSET);

    // One-way direction arrow (chevron at mid-stub)
    let dirArrow = null;
    if (!twoWay) {
      const midR = junctionRadius + STUB_LENGTH * 0.35;
      const mid = radialPt(a, midR);
      const { dx, dy } = dir(a);
      // Inbound = toward junction (opposite of stub direction)
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

  /** Port point where an arc starts (inbound) or ends (outbound) on a stub.
   *  Inbound port: center of the inbound lanes band at junction edge.
   *  Outbound port: center of the outbound lanes band at junction edge. */
  function portPt(stub, inbound) {
    const da = stub.displayAngle ?? stub.angle;
    const inner = radialPt(da, junctionRadius);
    const lanesI = Math.max(1, stub.lanesIn);
    const lanesO = Math.max(1, stub.lanesOut);
    const halfIn = lanesI * LANE_WIDTH / 2;
    const halfOut = lanesO * LANE_WIDTH / 2;
    // Right-hand traffic: inbound on -perp side, outbound on +perp side
    const offset = inbound ? -(halfIn / 2) : halfOut / 2;
    return offsetPt(inner, da, offset);
  }

  $: _ms = movState;
  $: movGeo = movements.map(mov => {
    const ms = _ms;
    const si = displayStubs.find(s => s.edgeId === mov.inEdgeId);
    const so = displayStubs.find(s => s.edgeId === mov.outEdgeId);
    if (!si || !so) return null;

    // Arc: inbound port (traffic arriving) -> outbound port (traffic leaving)
    const p0 = portPt(si, true);
    const p1 = portPt(so, false);

    // Control points: pull toward junction center by BEZIER_TENSION
    const toCenter0 = { x: CENTER - p0.x, y: CENTER - p0.y };
    const len0 = Math.sqrt(toCenter0.x ** 2 + toCenter0.y ** 2) || 1;
    const cp0 = {
      x: p0.x + BEZIER_TENSION * toCenter0.x / len0,
      y: p0.y + BEZIER_TENSION * toCenter0.y / len0,
    };

    const toCenter1 = { x: CENTER - p1.x, y: CENTER - p1.y };
    const len1 = Math.sqrt(toCenter1.x ** 2 + toCenter1.y ** 2) || 1;
    const cp1 = {
      x: p1.x + BEZIER_TENSION * toCenter1.x / len1,
      y: p1.y + BEZIER_TENSION * toCenter1.y / len1,
    };

    const path = `M ${f(p0.x)},${f(p0.y)} C ${f(cp0.x)},${f(cp0.y)} ${f(cp1.x)},${f(cp1.y)} ${f(p1.x)},${f(p1.y)}`;

    // Arrowhead at outbound port, pointing along stub direction (away from junction)
    const { dx, dy } = dir(so.displayAngle ?? so.angle);
    const tip  = { x: p1.x + dx * ARROW_SIZE, y: p1.y + dy * ARROW_SIZE };
    const left = { x: p1.x - dy * ARROW_SIZE * 0.5, y: p1.y + dx * ARROW_SIZE * 0.5 };
    const right= { x: p1.x + dy * ARROW_SIZE * 0.5, y: p1.y - dx * ARROW_SIZE * 0.5 };
    const arrow = `${f(tip.x)},${f(tip.y)} ${f(left.x)},${f(left.y)} ${f(right.x)},${f(right.y)}`;

    const gid = ms[mov.id]?.groupId ?? null;
    const color = gid !== null ? gColor(gid) : '#94a3b8';
    const forbidden = gid === null;

    return { ...mov, path, arrow, gid, color, forbidden };
  }).filter(Boolean);

  // Actions

  let hovMovId = null;
  let hovGroupId = null;

  function selectArc(movId) {
    selMovId = selMovId === movId ? null : movId;
  }

  function setGroup(movId, raw) {
    movState = { ...movState, [movId]: { groupId: raw === '' ? null : parseInt(raw) } };
  }

  function addGroup() {
    const maxId = groups.length ? Math.max(...groups.map(g => g.id)) : -1;
    groups = [...groups, { id: maxId + 1, greenDuration: 30 }];
  }

  function removeGroup(gid) {
    if (groups.length <= 1) return;
    groups = groups.filter(g => g.id !== gid);
    const ns = { ...movState };
    for (const m of movements) {
      if (ns[m.id]?.groupId === gid) ns[m.id] = { groupId: null };
    }
    movState = ns;
  }

  function updGreen(gid, val) {
    groups = groups.map(g => g.id === gid ? { ...g, greenDuration: Math.max(1, parseInt(val) || 1) } : g);
  }

  function save() {
    const config = {
      macro_node_id: nodeId,
      groups: groups.map(g => ({ id: g.id, greenDuration: g.greenDuration })),
      movState: { ...movState },
    };
    intersectionConfigs.update(c => ({ ...c, [nodeId]: config }));
    console.log(`[IntersectionEditor] ${node.label} config:`, JSON.stringify(config, null, 2));
    dispatch('close');
  }

  function handleBackdrop(e) {
    if (e.target === e.currentTarget) dispatch('close');
  }

  $: selMov = selMovId ? movements.find(m => m.id === selMovId) : null;

  function gColor(id) { return GROUP_COLORS[id % GROUP_COLORS.length]; }
  function gBg(id) { return GROUP_BG[id % GROUP_BG.length]; }

  // Reactive map: groupId → assigned movements
  $: assignedByGroup = (() => {
    const ms = _ms;
    const result = {};
    for (const g of groups) {
      result[g.id] = movements.filter(m => ms[m.id]?.groupId === g.id);
    }
    return result;
  })();

  // Determine turn type from angle difference between inbound and outbound stubs
  function turnType(mov) {
    const si = displayStubs.find(s => s.edgeId === mov.inEdgeId);
    const so = displayStubs.find(s => s.edgeId === mov.outEdgeId);
    if (!si || !so) return '';
    // Angle from inbound direction (toward junction) to outbound direction (away from junction)
    // Inbound arrives from si.displayAngle direction, so it "faces" si.displayAngle + PI
    const inFacing = (si.displayAngle ?? si.angle) + Math.PI;
    const outDir = so.displayAngle ?? so.angle;
    let diff = outDir - inFacing;
    // Normalize to (-PI, PI]
    while (diff > Math.PI) diff -= 2 * Math.PI;
    while (diff <= -Math.PI) diff += 2 * Math.PI;
    const deg = Math.abs(diff * 180 / Math.PI);
    if (deg < 30) return 'straight';
    if (diff > 0) return 'right';
    return 'left';
  }

  const TURN_ICONS = { left: '\u2B9C', straight: '\u2B9D', right: '\u2B9E' };

  // Check if two movements potentially conflict (cross paths).
  // Simplified heuristic: movements conflict if they come from different approaches
  // and their outbound approaches are "crossed" in angular order.
  function movementsConflict(m1, m2) {
    if (m1.inEdgeId === m2.inEdgeId && m1.outEdgeId === m2.outEdgeId) return false;
    if (m1.inEdgeId === m2.inEdgeId) return false; // same approach, no crossing
    if (m1.outEdgeId === m2.outEdgeId) return true; // merging into same exit = conflict
    // Check if paths cross: in1→out1 crosses in2→out2
    const si1 = displayStubs.find(s => s.edgeId === m1.inEdgeId);
    const so1 = displayStubs.find(s => s.edgeId === m1.outEdgeId);
    const si2 = displayStubs.find(s => s.edgeId === m2.inEdgeId);
    const so2 = displayStubs.find(s => s.edgeId === m2.outEdgeId);
    if (!si1 || !so1 || !si2 || !so2) return false;
    const a1 = si1.displayAngle ?? si1.angle;
    const b1 = so1.displayAngle ?? so1.angle;
    const a2 = si2.displayAngle ?? si2.angle;
    const b2 = so2.displayAngle ?? so2.angle;
    // Normalize angles and check if arcs (a1→b1) and (a2→b2) cross on the circle
    function normAngle(a) { return ((a % (2 * Math.PI)) + 2 * Math.PI) % (2 * Math.PI); }
    const na1 = normAngle(a1), nb1 = normAngle(b1);
    const na2 = normAngle(a2), nb2 = normAngle(b2);
    // Two chords cross if their endpoints alternate on the circle
    function between(x, lo, hi) {
      if (lo < hi) return x > lo && x < hi;
      return x > lo || x < hi;
    }
    const b1_between = between(nb1, na1, na2) !== between(nb1, na1, nb2);
    return between(na2, na1, nb1) !== between(nb2, na1, nb1);
  }

  // Per-group conflict warnings
  $: groupConflicts = (() => {
    const result = {};
    for (const g of groups) {
      const gMovs = assignedByGroup[g.id] || [];
      let hasConflict = false;
      for (let i = 0; i < gMovs.length && !hasConflict; i++) {
        for (let j = i + 1; j < gMovs.length && !hasConflict; j++) {
          if (movementsConflict(gMovs[i], gMovs[j])) hasConflict = true;
        }
      }
      result[g.id] = hasConflict;
    }
    return result;
  })();
</script>

{#if nodeId != null && node}
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
  transition:fade={{ duration: 150 }}
  class="fixed inset-0 z-50 flex items-center justify-center"
  style="background-color: rgba(0,0,0,0.55)"
  on:click={handleBackdrop}
  on:keydown={e => e.key === 'Escape' && dispatch('close')}
  role="dialog" aria-modal="true" tabindex="-1"
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
      <h2 class="text-sm font-semibold text-gray-800">{node.label} — Intersection</h2>
      <span class="text-xs text-gray-400">Click arc to select · assign group or mark forbidden</span>
      <button
        class="ml-auto p-1.5 text-gray-400 hover:text-gray-600 rounded-md hover:bg-gray-100
               focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
        on:click={() => dispatch('close')}
        aria-label="Close"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
        </svg>
      </button>
    </div>

    <!-- Body -->
    <div class="flex flex-1 min-h-0">

      <!-- Left: SVG road diagram -->
      <div class="flex-1 relative overflow-hidden" style="background:#e2e8f0">
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
              <button on:click={panLeft} title="Pan left" class={tbtn}>
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7"/></svg>
              </button>
              <div class="flex flex-col gap-0.5">
                <button on:click={panUp} title="Pan up" class={tbtn}>
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7"/></svg>
                </button>
                <button on:click={panDown} title="Pan down" class={tbtn}>
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7"/></svg>
                </button>
              </div>
              <button on:click={panRight} title="Pan right" class={tbtn}>
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7"/></svg>
              </button>
            </div>
          </div>

          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <svg
            bind:this={svgEl}
            viewBox={viewBox}
            preserveAspectRatio="xMidYMid meet"
            class="absolute inset-0 w-full h-full select-none"
            style="cursor: {dragging ? 'grabbing' : 'grab'}"
            on:wheel={handleWheel}
            on:pointerdown={onPointerDown}
            on:pointermove={onPointerMove}
            on:pointerup={onPointerUp}
            on:pointercancel={onPointerUp}
            on:click={onSvgClick}
          >

            <!-- Junction box (circle, seamless with road stubs) -->
            <circle cx={CENTER} cy={CENTER} r={junctionRadius + 2} fill={ROAD_COLOR}/>

            <!-- Road stubs -->
            {#each stubGeo as sg}
              {#if sg.twoWay}
                <!-- Two-way: separate in/out bands -->
                <path d={sg.inBandPath} fill={IN_TINT}/>
                <path d={sg.outBandPath} fill={OUT_TINT}/>
              {:else}
                <!-- One-way: single band, full width -->
                <path d={sg.roadPath} fill={sg.lanesIn > 0 ? IN_TINT : OUT_TINT}/>
              {/if}

              {#each sg.laneDividers as ld}
                <line x1={ld.x1} y1={ld.y1} x2={ld.x2} y2={ld.y2}
                  stroke="white" stroke-width="0.8"
                  stroke-dasharray="4,3" opacity="0.6"
                  pointer-events="none"/>
              {/each}

              <!-- Center divider (yellow dashed, only two-way) -->
              {#if sg.divider}
                <line x1={sg.divider.x1} y1={sg.divider.y1}
                      x2={sg.divider.x2} y2={sg.divider.y2}
                  stroke="#fbbf24" stroke-width="1.5"
                  stroke-dasharray="7,5" opacity="0.7"
                  pointer-events="none"/>
              {/if}

              <!-- One-way direction arrow -->
              {#if sg.dirArrow}
                <polyline
                  points="{sg.dirArrow.x1},{sg.dirArrow.y1} {sg.dirArrow.x2},{sg.dirArrow.y2} {sg.dirArrow.x3},{sg.dirArrow.y3}"
                  fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                  opacity="0.7" pointer-events="none"/>
              {/if}

              <!-- Stop line at junction edge -->
              <line x1={sg.stopLine.x1} y1={sg.stopLine.y1}
                    x2={sg.stopLine.x2} y2={sg.stopLine.y2}
                stroke="white" stroke-width={STOP_LINE_WIDTH} opacity="0.85"
                pointer-events="none"/>

              <!-- Lane count badge -->
              <circle cx={sg.badge.x} cy={sg.badge.y} r={BADGE_RADIUS}
                fill="white" stroke="#94a3b8" stroke-width="1"/>
              <text x={sg.badge.x} y={sg.badge.y}
                text-anchor="middle" dominant-baseline="central"
                font-size="9" font-weight="700" fill="#334155"
              >{#if sg.twoWay}{sg.lanesIn}+{sg.lanesOut}{:else}{sg.lanesIn || sg.lanesOut}{/if}</text>

              <!-- Node label beyond stub -->
              <text x={sg.labelPos.x} y={sg.labelPos.y}
                text-anchor="middle" dominant-baseline="central"
                font-size="12" font-weight="700" fill="#1e293b"
              >{sg.label}</text>
            {/each}

            <!-- Movement arcs: hit areas + hover glow -->
            {#each movGeo as mg}
              <path d={mg.path} fill="none" stroke="transparent" stroke-width="16"
                style="cursor:pointer"
                on:click|stopPropagation={() => selectArc(mg.id)}
                on:pointerenter={() => hovMovId = mg.id}
                on:pointerleave={() => { if (hovMovId === mg.id) hovMovId = null; }}/>
            {/each}

            <!-- Movement arcs: visible -->
            {#each movGeo as mg}
              <!-- Selection / hover / group-hover glow -->
              {#if selMovId === mg.id || hovMovId === mg.id || (hovGroupId !== null && mg.gid === hovGroupId)}
                <path d={mg.path} fill="none"
                  stroke={mg.color} stroke-width="12"
                  opacity={selMovId === mg.id ? 0.18 : 0.1}
                  pointer-events="none"/>
              {/if}
              <!-- Arc line -->
              <path d={mg.path} fill="none"
                stroke={mg.color}
                stroke-width={selMovId === mg.id ? 4.5 : 3}
                stroke-dasharray={mg.forbidden ? '5,4' : 'none'}
                opacity={mg.forbidden ? 0.45 : 1}
                stroke-linecap="round"
                pointer-events="none"/>
              <!-- Arrowhead (only for allowed movements) -->
              {#if !mg.forbidden}
                <polygon points={mg.arrow} fill={mg.color} pointer-events="none"/>
              {/if}
            {/each}

            <!-- Center reference dot -->
            <circle cx={CENTER} cy={CENTER} r="3" fill="white" opacity="0.4"/>
          </svg>
        {/if}
      </div>

      <!-- Right panel -->
      <div class="w-72 shrink-0 border-l border-gray-200 bg-gray-50/40 flex flex-col">

        <!-- Fixed top section -->
        <div class="p-4 pb-0 space-y-4 shrink-0">
        <!-- Selected movement (fixed height to avoid layout jumps) -->
        <div class="rounded-lg border border-gray-200 bg-white p-3 shrink-0 h-[106px] flex flex-col justify-center overflow-hidden">
          {#if selMov}
            {@const gid = movState[selMov.id]?.groupId ?? null}
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
              <div class="flex items-center gap-1.5">
                <select
                  value={gid !== null ? String(gid) : ''}
                  on:change={e => setGroup(selMov.id, e.target.value)}
                  class="flex-1 text-xs px-2 py-1.5 border border-gray-200 rounded-md bg-white
                         focus:outline-none focus:ring-2 focus:ring-blue-400 text-gray-700"
                >
                  <option value="">Forbidden</option>
                  {#each groups as g}
                    <option value={String(g.id)}>Group {g.id}</option>
                  {/each}
                </select>
                {#if gid !== null}
                  <button
                    on:click={() => setGroup(selMov.id, '')}
                    class="px-2 py-1.5 text-xs rounded-md border border-red-200 text-red-500 hover:bg-red-50
                           focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400"
                    title="Forbid this movement"
                  >
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"/>
                    </svg>
                  </button>
                {/if}
              </div>
            </div>
          {:else}
            <p class="text-xs text-gray-400 italic leading-snug text-center">
              Click any arc to select it,<br/>
              then assign a signal group or mark forbidden.
            </p>
          {/if}
        </div>

        <!-- Signal groups header -->
          <div class="flex items-center justify-between">
            <p class="text-xs font-semibold text-gray-400 uppercase tracking-wide">Signal Groups</p>
            {#if groups.length < GROUP_COLORS.length}
              <button
                on:click={addGroup}
                class="text-xs px-2 py-1 rounded-md bg-white border border-gray-200 hover:bg-gray-100
                       text-gray-600 font-medium focus-visible:outline-none focus-visible:ring-2
                       focus-visible:ring-gray-400"
              >+ Add</button>
            {/if}
          </div>
        </div>

        <!-- Scrollable groups list -->
        <div class="flex-1 min-h-0 overflow-y-auto px-4 py-2">
          {#if groups.length === 0}
            <p class="text-xs text-gray-400 text-center py-6 leading-snug">
              Add a group to assign movements
            </p>
          {:else}
            <div class="space-y-3">
              {#each groups as g (g.id)}
                {@const assigned = assignedByGroup[g.id] || []}
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div class="rounded-lg border bg-white p-3 space-y-2.5"
                  style="border-color: {gColor(g.id)}55"
                  class:ring-2={selMovId && movState[selMovId]?.groupId === g.id}
                  class:ring-blue-400={selMovId && movState[selMovId]?.groupId === g.id}
                  on:pointerenter={() => hovGroupId = g.id}
                  on:pointerleave={() => { if (hovGroupId === g.id) hovGroupId = null; }}
                >
                  <div class="flex items-center gap-2 h-6">
                    <div class="w-3 h-3 rounded-full shrink-0" style="background:{gColor(g.id)}"></div>
                    <span class="text-sm font-semibold text-gray-700">Group {g.id}</span>
                    <div class="ml-auto">
                      {#if selMovId && movState[selMovId]?.groupId !== g.id}
                        <button
                          on:click={() => setGroup(selMovId, String(g.id))}
                          class="px-2 py-0.5 text-xs rounded-md font-medium border"
                          style="color:{gColor(g.id)}; border-color:{gColor(g.id)}55"
                        >Assign</button>
                      {:else if groups.length > 1 && !selMovId}
                        <button
                          on:click={() => removeGroup(g.id)}
                          class="p-0.5 text-gray-400 hover:text-red-500 rounded
                                 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400"
                          title="Remove group"
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
                      type="number" min="1" max="300"
                      value={g.greenDuration}
                      on:input={e => updGreen(g.id, e.target.value)}
                      class="w-16 px-2 py-1 text-xs border border-gray-200 rounded-md bg-white
                             focus:outline-none focus:ring-2 focus:ring-blue-400"
                    />
                    <span class="text-xs text-gray-400">s</span>
                  </div>

                  <div class="text-xs text-gray-500 leading-snug">
                    {#if assigned.length === 0}
                      <span class="italic text-gray-400">No movements assigned</span>
                    {:else}
                      {assigned.slice(0, 3).map(m => `${TURN_ICONS[turnType(m)] || ''} ${m.inLabel}\u2192${m.outLabel}`).join(', ')}
                      {#if assigned.length > 3}
                        <span class="text-gray-400"> +{assigned.length - 3} more</span>
                      {/if}
                    {/if}
                  </div>
                  {#if groupConflicts[g.id]}
                    <p class="text-xs text-amber-600 flex items-center gap-1">
                      <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"/>
                      </svg>
                      Possible conflict: crossing paths
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
            Each group = one stage (green phase).<br/>
            Dashed arc = no meso connector generated.
          </p>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-end gap-2 px-6 py-4 border-t border-gray-200 shrink-0">
      <button
        on:click={() => dispatch('close')}
        class="px-4 py-2 bg-gray-100 text-gray-700 text-sm rounded-md hover:bg-gray-200
               focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
      >Cancel</button>
      <button
        on:click={save}
        class="px-4 py-2 bg-blue-500 text-white text-sm rounded-md hover:bg-blue-600
               focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
      >Save configuration</button>
    </div>
  </div>
</div>
{/if}
