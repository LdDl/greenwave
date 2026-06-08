import { writable } from 'svelte/store';

// Macro-level graph nodes (intersections)
// { id: number, x: number, y: number, label: string }
export const networkNodes = writable([]);

// Macro-level graph edges (road segments)
// { id: number, from: number, to: number, lanes_fwd: number, lanes_back: number }
export const networkEdges = writable([]);

// Counters (kept outside stores so they survive store resets)
let _nodeId = 1;
let _edgeId = 1;

export function nextNodeId() { return _nodeId++; }
export function nextEdgeId() { return _edgeId++; }

// Intersection configs keyed by macro node ID
// { [nodeId]: { groups: [{id, greenDuration}], movState: {[movId]: {groupId: number|null}} } }
export const intersectionConfigs = writable({});

export function resetNetwork() {
  networkNodes.set([]);
  networkEdges.set([]);
  intersectionConfigs.set({});
  _nodeId = 1;
  _edgeId = 1;
}
