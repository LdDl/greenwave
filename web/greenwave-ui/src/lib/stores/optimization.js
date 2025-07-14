import { writable } from 'svelte/store';

export const optimizedGreenWaves = writable([]);
export const optimizedThroughWaves = writable([]);
export const optimizedOffsets = writable([]);
export const optimizationHistory = writable([]);
export const isOptimizing = writable(false);