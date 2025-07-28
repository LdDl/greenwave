import { writable } from 'svelte/store';

// Store to track if signals have been invalidated
export const signalsInvalidated = writable(false);

// Store to track if results have been invalidated
export const resultsInvalidated = writable(false);

// Helper to invalidate signals
export function invalidateSignals() {
  signalsInvalidated.set(true);
  resultsInvalidated.set(true); // Results should also be invalidated when signals change
}

// Helper to reset signal invalidation
export function resetSignalsInvalidation() {
  signalsInvalidated.set(false);
}

// Helper to reset results invalidation
export function resetResultsInvalidation() {
  resultsInvalidated.set(false);
}

// Helper to check if signals are invalidated (optional, for debugging)
export function areSignalsInvalidated() {
  let invalidated;
  signalsInvalidated.subscribe(value => (invalidated = value))();
  return invalidated;
}