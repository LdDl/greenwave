import { writable, derived } from 'svelte/store';

// Core data stores
export const junctions = writable([]);
export const desiredSpeed = writable(40.0);

// Desired intensity (vehicles per hour)
export const desiredIntensity = writable(1800);

// Desired flow (vehicles per second)
export const desiredFlow = derived(desiredIntensity, $desiredIntensity => $desiredIntensity / 3600);