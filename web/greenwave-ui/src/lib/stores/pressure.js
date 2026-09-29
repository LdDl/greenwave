import { writable } from 'svelte/store';

// Results survive navigation; project settings are saved by the shared project store.
export const pressureResult = writable(null);
