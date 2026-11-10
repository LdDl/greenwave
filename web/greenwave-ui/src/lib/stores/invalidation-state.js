import { writable } from 'svelte/store';

export const isResultsInvalidated = writable(false);
export const isInputInvalidated = writable(false);
export const resultsInvalidationReasons = writable([]);
export const inputInvalidationReasons = writable([]);
