import { derived } from 'svelte/store';
import { corridorEditor } from './corridor.js';

// Core data stores
export const junctions = corridorEditor.junctions;
export const desiredSpeed = corridorEditor.desiredSpeed;

// Optimization direction: 'forward' or 'bidirectional'
export const optimizationDirection = corridorEditor.direction;

// Desired intensity (vehicles per hour)
export const desiredIntensity = corridorEditor.desiredIntensity;

// Desired flow (vehicles per second)
export const desiredFlow = derived(desiredIntensity, $desiredIntensity => $desiredIntensity / 3600);
