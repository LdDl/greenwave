import { writable } from 'svelte/store';

export const junctions = writable([]);
export const selectedJunction = writable(null);
export const desiredSpeed = writable(40);