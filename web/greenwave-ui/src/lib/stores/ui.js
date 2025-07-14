import { writable } from 'svelte/store';

export const isLoading = writable(false);
export const error = writable(null);
export const toasts = writable([]);