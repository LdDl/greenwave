import { writable, derived } from 'svelte/store';
import { originalGreenWaves } from './greenwave';

// Writable stores for invalidation flags
export const isResultsInvalidated = writable(false);
export const isInputInvalidated = writable(false);

// Separate stores for invalidation reasons
export const resultsInvalidationReasons = writable([]);
export const inputInvalidationReasons = writable([]);

// Helper: Invalidate results and input data
export function invalidateAll(reason) {
  isResultsInvalidated.set(true);

  // Only invalidate input data if green waves have been extracted
  originalGreenWaves.subscribe(greenWaves => {
    if (greenWaves.length > 0) {
      isInputInvalidated.set(true);

      // Add reason to input invalidation reasons
      inputInvalidationReasons.update(reasons => {
        if (!reasons.includes(reason)) {
          return [...reasons, reason];
        }
        return reasons;
      });
    }
  });

  // Add reason to results invalidation reasons
  resultsInvalidationReasons.update(reasons => {
    if (!reasons.includes(reason)) {
      return [...reasons, reason];
    }
    return reasons;
  });
}

// Helper: Invalidate results only
export function invalidateResults(reason) {
  isResultsInvalidated.set(true);

  // Add reason to results invalidation reasons
  resultsInvalidationReasons.update(reasons => {
    if (!reasons.includes(reason)) {
      return [...reasons, reason];
    }
    return reasons;
  });
}

// Helper: Validate results
export function validateResults() {
  isResultsInvalidated.set(false);
  resultsInvalidationReasons.set([]);
}

// Helper: Validate input data
export function validateInput() {
  isInputInvalidated.set(false);
  inputInvalidationReasons.set([]);
}

// Derived store: Check if anything is invalidated
export const isAnythingInvalidated = derived(
  [isResultsInvalidated, isInputInvalidated],
  ([$isResultsInvalidated, $isInputInvalidated]) => $isResultsInvalidated || $isInputInvalidated
);