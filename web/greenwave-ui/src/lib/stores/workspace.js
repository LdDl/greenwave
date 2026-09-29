import { writable, derived } from 'svelte/store';

export const corridorsOpen = writable(false);
export const corridorToEdit = writable(null);
export const projectDialogOpen = writable(false);
export const workspaceDialogOpen = derived(
	[corridorsOpen, projectDialogOpen],
	([routes, project]) => routes || project
);
export const projectOpened = writable(0);
export const analysisMode = writable('input');
export const analysisInspectorOpen = writable(true);
export const optimizedCorridorName = writable('');
