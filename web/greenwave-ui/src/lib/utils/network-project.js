export const PROJECT_FORMAT = 'greenwave-network';
export const PROJECT_VERSION = 1;
export const MAX_LANES = 12;
export const MAX_GREEN = 300;

export function emptyProject() {
  return { format: PROJECT_FORMAT, version: PROJECT_VERSION, name: 'Untitled network', nodes: [], edges: [], intersections: {} };
}

export function cloneProject(value) {
  return JSON.parse(JSON.stringify(value));
}

export function intersectionTopology(nodeId, nodes, edges) {
  const node = nodes.find(n => n.id === nodeId);
  if (!node) return { stubs: [], movements: [] };
  const stubs = edges.filter(e => e.from === nodeId || e.to === nodeId).map(e => {
    const other = nodes.find(n => n.id === (e.from === nodeId ? e.to : e.from));
    if (!other) return null;
    const atStart = e.from === nodeId;
    const lanesIn = atStart ? e.lanes_back : e.lanes_fwd;
    const lanesOut = atStart ? e.lanes_fwd : e.lanes_back;
    return {
      edgeId: e.id, angle: Math.atan2(other.y - node.y, other.x - node.x), label: other.label,
      lanesIn, lanesOut, hasInbound: lanesIn > 0, hasOutbound: lanesOut > 0,
      inDir: atStart ? 'back' : 'fwd', outDir: atStart ? 'fwd' : 'back',
    };
  }).filter(Boolean);
  const movements = [];
  for (const src of stubs.filter(s => s.hasInbound)) {
    for (const dst of stubs.filter(s => s.hasOutbound)) {
      if (src.edgeId === dst.edgeId) continue;
      movements.push({
        id: `${src.edgeId}_${src.inDir}_${dst.edgeId}_${dst.outDir}`,
        inEdgeId: src.edgeId, inDir: src.inDir, outEdgeId: dst.edgeId, outDir: dst.outDir,
        inLabel: src.label, outLabel: dst.label,
      });
    }
  }
  return { stubs, movements };
}

export function reconcileIntersections(project) {
  const next = {};
  for (const node of project.nodes) {
    const saved = project.intersections[node.id];
    if (!saved) continue;
    const groupIds = new Set(saved.groups.map(g => g.id));
    const { movements } = intersectionTopology(node.id, project.nodes, project.edges);
    const movState = {};
    for (const movement of movements) {
      const state = saved.movState[movement.id];
      movState[movement.id] = { groupIds: [...new Set(state?.groupIds ?? [])].filter(id => groupIds.has(id)) };
    }
    next[node.id] = { macro_node_id: node.id, groups: saved.groups, movState };
  }
  project.intersections = next;
  return project;
}

function requireValue(condition, message) {
  if (!condition) throw new Error(message);
}

function record(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function integer(value, minimum = 0, maximum = Number.MAX_SAFE_INTEGER - 1) {
  return Number.isSafeInteger(value) && value >= minimum && value <= maximum;
}

export function parseProject(input) {
  const data = typeof input === 'string' ? JSON.parse(input) : input;
  requireValue(record(data) && data.format === PROJECT_FORMAT, 'Choose a Greenwave network project JSON file.');
  requireValue(data.version === PROJECT_VERSION, `Unsupported project version: ${data.version}.`);
  requireValue(typeof data.name === 'string' && data.name.trim().length > 0 && data.name.length <= 120, 'Project name must contain 1 to 120 characters.');
  requireValue(Array.isArray(data.nodes) && Array.isArray(data.edges) && record(data.intersections), 'Project nodes, roads or intersections are missing.');
  const nodeIds = new Set();
  const nodes = data.nodes.map(n => {
    requireValue(record(n) && integer(n.id, 1) && !nodeIds.has(n.id), 'Junction IDs must be unique positive integers.');
    requireValue(Number.isFinite(n.x) && Number.isFinite(n.y), `Junction ${n.id} has invalid coordinates.`);
    requireValue(typeof n.label === 'string' && n.label.trim().length > 0 && n.label.length <= 120, `Junction ${n.id} needs a label of 1 to 120 characters.`);
    nodeIds.add(n.id);
    return { id: n.id, x: n.x, y: n.y, label: n.label.trim() };
  });
  const edgeIds = new Set();
  const pairs = new Set();
  const edges = data.edges.map(e => {
    requireValue(record(e) && integer(e.id, 1) && !edgeIds.has(e.id), 'Road IDs must be unique positive integers.');
    requireValue(nodeIds.has(e.from) && nodeIds.has(e.to) && e.from !== e.to, `Road ${e.id} must connect two existing, different junctions.`);
    requireValue(integer(e.lanes_fwd, 1, MAX_LANES) && integer(e.lanes_back, 0, MAX_LANES), `Road ${e.id} has invalid lane counts (1 to ${MAX_LANES} forward, 0 to ${MAX_LANES} back).`);
    const pair = [e.from, e.to].sort((a, b) => a - b).join(':');
    requireValue(!pairs.has(pair), 'Only one road segment is allowed between a pair of junctions.');
    pairs.add(pair);
    edgeIds.add(e.id);
    return { id: e.id, from: e.from, to: e.to, lanes_fwd: e.lanes_fwd, lanes_back: e.lanes_back };
  });
  const intersections = {};
  for (const [key, config] of Object.entries(data.intersections)) {
    const id = Number(key);
    requireValue(String(id) === key && nodeIds.has(id) && record(config), 'Intersection configuration references an unknown junction.');
    requireValue(config.macro_node_id === id, `Intersection ${id} has a mismatched junction ID.`);
    requireValue(Array.isArray(config.groups) && config.groups.length > 0 && record(config.movState), `Intersection ${id} needs groups and movement assignments.`);
    const groupIds = new Set();
    const groups = config.groups.map(g => {
      requireValue(record(g) && integer(g.id) && !groupIds.has(g.id), `Intersection ${id} has invalid or duplicate group IDs.`);
      requireValue(integer(g.greenDuration, 1, MAX_GREEN), `Green duration must be a whole number from 1 to ${MAX_GREEN} seconds.`);
      groupIds.add(g.id);
      return { id: g.id, greenDuration: g.greenDuration };
    });
    const validMovements = new Set(intersectionTopology(id, nodes, edges).movements.map(m => m.id));
    const movState = {};
    for (const [movement, state] of Object.entries(config.movState)) {
      requireValue(validMovements.has(movement) && record(state), `Intersection ${id} references a movement that does not exist.`);
      const ids = state.groupIds ?? (state.groupId != null ? [state.groupId] : []);
      requireValue(Array.isArray(ids) && ids.every(gid => groupIds.has(gid)), `Movement ${movement} references an unknown group.`);
      movState[movement] = { groupIds: [...new Set(ids)] };
    }
    intersections[id] = { macro_node_id: id, groups, movState };
  }
  return reconcileIntersections({ format: PROJECT_FORMAT, version: PROJECT_VERSION, name: data.name.trim(), nodes, edges, intersections });
}

// Separate inbound and outbound ports preserve the circular order for right-hand traffic.
export function movementsConflict(a, b, stubs) {
  if (a.id === b.id || a.inEdgeId === b.inEdgeId) return false;
  if (a.outEdgeId === b.outEdgeId) return true;
  const sorted = [...stubs].sort((x, y) => x.angle - y.angle || x.edgeId - y.edgeId);
  const port = (edgeId, inbound) => sorted.findIndex(s => s.edgeId === edgeId) * 2 + (inbound ? 0 : 1);
  const start = port(a.inEdgeId, true);
  const end = port(a.outEdgeId, false);
  const between = x => start < end ? x > start && x < end : x > start || x < end;
  return between(port(b.inEdgeId, true)) !== between(port(b.outEdgeId, false));
}

export function groupConflictPairs(movements, movState, groups, stubs) {
  return Object.fromEntries(groups.map(group => {
    const assigned = movements.filter(m => movState[m.id]?.groupIds.includes(group.id));
    const pairs = [];
    for (let i = 0; i < assigned.length; i++) {
      for (let j = i + 1; j < assigned.length; j++) {
        if (movementsConflict(assigned[i], assigned[j], stubs)) pairs.push([assigned[i], assigned[j]]);
      }
    }
    return [group.id, pairs];
  }));
}

export function demoProject() {
  const project = emptyProject();
  project.name = 'Four-way junction';
  project.nodes = [
    { id: 1, x: 360, y: 260, label: 'Center' },
    { id: 2, x: 360, y: 70, label: 'North' },
    { id: 3, x: 600, y: 260, label: 'East' },
    { id: 4, x: 360, y: 450, label: 'South' },
    { id: 5, x: 120, y: 260, label: 'West' },
  ];
  project.edges = [2, 3, 4, 5].map((to, index) => ({ id: index + 1, from: 1, to, lanes_fwd: 1, lanes_back: 1 }));
  return project;
}
