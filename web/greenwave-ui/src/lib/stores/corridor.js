import { derived, get, writable } from 'svelte/store';
import { inputToSimpleCorridor } from '../utils/simple-corridor.js';
import { validateImportedConfig } from '../utils/export-import.js';
import { readProgram } from '../utils/junction-program.js';

export const CORRIDOR_STORAGE_KEY = 'greenwave.corridor.v1';

function emptyDocument() {
  return {
    format: 'greenwave-corridor', version: 1,
    input: { junctions: [], desiredSpeed: 40, desiredIntensity: 1800, direction: 'forward' },
    positions: {},
  };
}

function checkInput(input) {
  const validation = validateImportedConfig(input);
  if (!validation.isValid) throw new Error(validation.errors.join(' '));
}

function parseDocument(text) {
  const value = JSON.parse(text);
  if (value?.format !== 'greenwave-corridor' || value.version !== 1) throw new Error('Unsupported corridor project.');
  checkInput(value.input);
  if (!value.positions || typeof value.positions !== 'object' || Array.isArray(value.positions)) throw new Error('Invalid canvas positions.');
  const nodeIds = new Set(value.input.junctions.map(node => String(node.id)));
  for (const [id, position] of Object.entries(value.positions)) {
    if (!nodeIds.has(id) || !Number.isFinite(position?.x) || !Number.isFinite(position?.y)) throw new Error('Invalid canvas position.');
  }
  return { ...emptyDocument(), input: { ...emptyDocument().input, ...value.input }, positions: value.positions };
}

function calculationData(document) {
  const { junctions, desiredSpeed, direction, groupIds } = document.input;
  return JSON.stringify({ junctions, desiredSpeed, direction, groupIds });
}

export function createCorridorEditor() {
  let current = emptyDocument();
  let past = [];
  let future = [];
  let transaction = null;
  let storage = null;
  let connected = false;
  let initialized = false;
  let preserveSaved = false;
  const project = writable(current);
  const history = writable({ undo: false, redo: false });
  const persistence = writable({ state: 'memory', message: 'Changes are kept in memory' });
  const calculationRevision = writable(0);

  function status() {
    history.set({ undo: past.length > 0, redo: future.length > 0 });
  }

  function persist() {
    if (!connected || preserveSaved || transaction) return;
    try {
      checkInput(current.input);
    } catch (cause) {
      persistence.set({ state: 'error', message: `Corridor could not be saved. ${cause.message}` });
      return;
    }
    try {
      storage.setItem(CORRIDOR_STORAGE_KEY, JSON.stringify(current));
      persistence.set({ state: 'saved', message: 'Saved in this browser' });
    } catch {
      persistence.set({ state: 'error', message: 'Browser save unavailable. Export input to keep a copy.' });
    }
  }

  function publish(next) {
    const changedCalculation = calculationData(current) !== calculationData(next);
    current = next;
    project.set(current);
    if (changedCalculation) calculationRevision.update(value => value + 1);
    status();
    persist();
  }

  function change(mutator) {
    const next = structuredClone(current);
    mutator(next);
    const ids = new Set(next.input.junctions.map(node => String(node.id)));
    next.positions = Object.fromEntries(Object.entries(next.positions).filter(([id]) => ids.has(id)));
    if (next.input.groupIds) next.input.groupIds = Object.fromEntries(Object.entries(next.input.groupIds).filter(([id]) => ids.has(id)));
    if (JSON.stringify(next) === JSON.stringify(current)) return;
    initialized = true;
    preserveSaved = false;
    if (!transaction) {
      past = [...past.slice(-99), current];
      future = [];
    }
    publish(next);
  }

  function begin() {
    if (!transaction) transaction = current;
  }

  function finish() {
    if (!transaction) return;
    if (JSON.stringify(transaction) !== JSON.stringify(current)) {
      past = [...past.slice(-99), transaction];
      future = [];
    }
    transaction = null;
    status();
    persist();
  }

  function undo() {
    finish();
    if (!past.length) return;
    future.push(current);
    preserveSaved = false;
    publish(past.pop());
  }

  function redo() {
    finish();
    if (!future.length) return;
    past.push(current);
    preserveSaved = false;
    publish(future.pop());
  }

  function replaceInput(input) {
    const next = { ...current.input };
    for (const key of ['junctions', 'desiredSpeed', 'desiredIntensity', 'direction', 'groupIds']) {
      if (input[key] !== undefined) next[key] = structuredClone(input[key]);
    }
    if (input.junctions !== undefined && input.groupIds === undefined) delete next.groupIds;
    checkInput(next);
    finish();
    initialized = true;
    preserveSaved = false;
    change(document => { document.input = next; document.positions = {}; });
    persist();
  }

  // These fields are writable views of one document, not synchronized copies.
  function field(name, fallback) {
    const value = derived(project, document => document.input[name] ?? fallback);
    return {
      subscribe: value.subscribe,
      set: next => change(document => { document.input[name] = structuredClone(next); }),
      update: update => change(document => { document.input[name] = update(document.input[name]); }),
    };
  }

  const graph = derived(project, document => {
    try {
      const corridor = inputToSimpleCorridor(document.input);
      return {
        error: '',
        nodes: corridor.nodes.map(node => ({ id: node.id, label: node.label, ...(document.positions[node.id] ?? node.position) })),
        edges: corridor.roads.map(road => ({ id: road.id, from: road.from, to: road.to, lengthMeters: road.lengthMeters, lanes_fwd: road.forwardLanes, lanes_back: road.reverseLanes })),
      };
    } catch (cause) {
      return { error: cause.message, nodes: [], edges: [] };
    }
  });

  function connectStorage(target) {
    storage = target;
    if (!initialized) {
      initialized = true;
      try {
        const saved = storage.getItem(CORRIDOR_STORAGE_KEY);
        if (saved !== null) publish(parseDocument(saved));
      } catch {
        preserveSaved = true;
        persistence.set({ state: 'error', message: 'Saved corridor could not be opened. Import a backup or load demo data to replace it.' });
      }
    }
    connected = true;
    persist();
    return () => { finish(); connected = false; storage = null; };
  }

  return {
    project: { subscribe: project.subscribe }, history, persistence, calculationRevision, graph,
    junctions: field('junctions'), desiredSpeed: field('desiredSpeed'), desiredIntensity: field('desiredIntensity'), direction: field('direction'),
    groupIds: field('groupIds', {}),
    begin, finish, undo, redo, replaceInput, connectStorage,
    snapshot: () => structuredClone(current),
    canvas: {
      begin, finish,
      change: mutate => {
        const view = structuredClone(get(graph));
        if (view.error) return;
        mutate(view);
        change(document => {
          for (const node of view.nodes) document.positions[node.id] = { x: node.x, y: node.y };
        });
      },
    },
    saveJunction(junction, isNew = false, groupId) {
      change(document => {
        const program = readProgram(junction);
        if (groupId !== undefined && groupId !== null && !program.groupIds.includes(groupId)) throw new Error('Selected corridor group does not exist.');
        if (isNew) document.input.junctions.push(structuredClone(junction));
        else document.input.junctions = document.input.junctions.map(node => node.id === junction.id ? structuredClone(junction) : node);
        if (groupId !== undefined && (program.groupIds.length > 1 || document.input.groupIds)) {
          document.input.groupIds = { ...document.input.groupIds };
          if (groupId === null) delete document.input.groupIds[junction.id];
          else document.input.groupIds[junction.id] = groupId;
        }
        checkInput(document.input);
      });
    },
    removeJunction(id) {
      change(document => { document.input.junctions = document.input.junctions.filter(node => node.id !== id); });
    },
    setRoadLength(id, length) {
      if (!Number.isFinite(length) || length < 0) throw new Error('Road length must be a non-negative number.');
      const view = get(graph);
      const road = view.edges.find(edge => edge.id === id);
      if (!road) throw new Error('Road is no longer part of the corridor.');
      change(document => {
        const start = document.input.junctions.findIndex(node => node.id === road.to);
        const difference = length - road.lengthMeters;
        for (const node of document.input.junctions.slice(start)) node.point.y += difference;
        checkInput(document.input);
      });
    },
  };
}

export const corridorEditor = createCorridorEditor();
export const corridorProject = corridorEditor.project;
export const corridorGraph = corridorEditor.graph;
export const corridorHistory = corridorEditor.history;
export const corridorPersistence = corridorEditor.persistence;
export const corridorGroupIds = corridorEditor.groupIds;
export const corridorNodes = derived(corridorGraph, graph => graph.nodes);
export const corridorEdges = derived(corridorGraph, graph => graph.edges);
