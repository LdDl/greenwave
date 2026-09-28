import { derived, get } from 'svelte/store';
import { originalGreenWaves } from './greenwave';
import { isResultsInvalidated, isInputInvalidated, resultsInvalidationReasons, inputInvalidationReasons } from './invalidation-state.js';
export { isResultsInvalidated, isInputInvalidated, resultsInvalidationReasons, inputInvalidationReasons } from './invalidation-state.js';

// Helper: Invalidate results and input data
export function invalidateAll(reason) {
  isResultsInvalidated.set(true);

  // Only invalidate input data if green waves have been extracted
  if (get(originalGreenWaves).length > 0) {
    isInputInvalidated.set(true);

    // Add reason to input invalidation reasons
    inputInvalidationReasons.update(reasons => {
      if (!reasons.includes(reason)) {
        return [...reasons, reason];
      }
      return reasons;
    });
  }

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
