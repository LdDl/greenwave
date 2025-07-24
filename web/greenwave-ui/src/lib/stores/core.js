import { writable } from 'svelte/store';

// Core data stores
export const junctions = writable([]);
export const desiredSpeed = writable(40.0);