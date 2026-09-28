import { derived, get, writable } from 'svelte/store';
import { cloneProject, emptyProject, parseProject, reconcileIntersections } from '../utils/network-project.js';

export const STORAGE_KEY = 'greenwave.network.v1';

export function createNetworkEditor() {
  const project = writable(emptyProject());
  const history = writable({ undo: false, redo: false });
  const persistence = writable({ state: 'loading', message: '' });
  let past = [];
  let future = [];
  let transaction = null;
  let storage = null;
  let unsubscribe = null;
  let ready = false;

  function status() {
    history.set({ undo: past.length > 0, redo: future.length > 0 });
  }

  function persist(value) {
    if (!ready) return;
    try {
      storage.setItem(STORAGE_KEY, JSON.stringify(value));
      persistence.set({ state: 'saved', message: 'Saved in this browser' });
    } catch {
      persistence.set({ state: 'error', message: 'Browser save unavailable. Export your project to keep a copy.' });
    }
  }

  function remember(previous) {
    past = [...past.slice(-99), previous];
    future = [];
    status();
  }

  function change(mutator) {
    const previous = get(project);
    const next = cloneProject(previous);
    mutator(next);
    reconcileIntersections(next);
    if (JSON.stringify(previous) === JSON.stringify(next)) return;
    if (!transaction) remember(previous);
    project.set(next);
  }

  function begin() {
    if (!transaction) transaction = cloneProject(get(project));
  }

  function finish() {
    if (!transaction) return;
    if (JSON.stringify(transaction) !== JSON.stringify(get(project))) remember(transaction);
    transaction = null;
    persist(get(project));
  }

  function replace(value) {
    const next = parseProject(value);
    finish();
    const previous = get(project);
    if (JSON.stringify(previous) === JSON.stringify(next)) {
      ready = true;
      persist(next);
      return;
    }
    remember(previous);
    project.set(next);
  }

  function undo() {
    finish();
    if (!past.length) return;
    future.push(get(project));
    project.set(past.pop());
    status();
  }

  function redo() {
    finish();
    if (!future.length) return;
    past.push(get(project));
    project.set(future.pop());
    status();
  }

  function connectStorage(target) {
    storage = target;
    if (!ready) {
      try {
        const saved = storage.getItem(STORAGE_KEY);
        if (saved !== null) project.set(parseProject(saved));
        ready = true;
      } catch {
        persistence.set({ state: 'error', message: 'Saved project could not be opened. Import a backup or start a new project.' });
        // Preserve an unreadable saved project until the user changes the document.
      }
    }
    unsubscribe?.();
    let first = true;
    unsubscribe = project.subscribe(value => {
      if (!ready && first) { first = false; return; }
      first = false;
      ready = true;
      if (!transaction) persist(value);
    });
    return () => { finish(); unsubscribe?.(); unsubscribe = null; };
  }

  return {
    project: { subscribe: project.subscribe }, history, persistence, change, begin, finish, replace, undo, redo, connectStorage,
    snapshot: () => cloneProject(get(project)),
    addNode(x, y) {
      const id = Math.max(0, ...get(project).nodes.map(n => n.id)) + 1;
      change(p => p.nodes.push({ id, x, y, label: `J${id}` }));
      return id;
    },
    addRoad(from, to, lanesFwd, lanesBack) {
      const value = get(project);
      if (from === to || !value.nodes.some(n => n.id === from) || !value.nodes.some(n => n.id === to)) return null;
      if (value.edges.some(e => (e.from === from && e.to === to) || (e.from === to && e.to === from))) return null;
      const id = Math.max(0, ...value.edges.map(e => e.id)) + 1;
      const next = cloneProject(value);
      next.edges.push({ id, from, to, lanes_fwd: lanesFwd, lanes_back: lanesBack });
      replace(next);
      return id;
    },
    remove(nodeId, edgeId) {
      change(p => {
        if (nodeId != null) {
          p.nodes = p.nodes.filter(n => n.id !== nodeId);
          p.edges = p.edges.filter(e => e.from !== nodeId && e.to !== nodeId);
        } else p.edges = p.edges.filter(e => e.id !== edgeId);
      });
    },
    reverseRoad(id) {
      change(p => {
        const edge = p.edges.find(e => e.id === id);
        if (!edge || edge.lanes_back > 0) return;
        [edge.from, edge.to] = [edge.to, edge.from];
        // Reversing a one-way road changes its physical movements at both ends.
        for (const config of Object.values(p.intersections)) {
          for (const movement of Object.keys(config.movState)) {
            const [incoming, , outgoing] = movement.split('_');
            if (Number(incoming) === id || Number(outgoing) === id) delete config.movState[movement];
          }
        }
      });
    },
    saveIntersection(nodeId, config) {
      const next = cloneProject(get(project));
      next.intersections[nodeId] = config;
      replace(next);
    },
  };
}

export const networkEditor = createNetworkEditor();
export const networkProject = networkEditor.project;
export const networkHistory = networkEditor.history;
export const networkPersistence = networkEditor.persistence;
export const networkNodes = derived(networkProject, p => p.nodes);
export const networkEdges = derived(networkProject, p => p.edges);
export const intersectionConfigs = derived(networkProject, p => p.intersections);
