import { writable, derived } from 'svelte/store';
import { junctions, desiredSpeed, desiredFlow } from './core';
import { calculateTotalDuration } from '$lib/utils/junction-helpers.js';
import { resultsInvalidated } from './signals';
import { resultsInvalidationReasons, isResultsInvalidated } from '$lib/stores/invalidation';

// Optimized waves (forward direction)
export const optimizedGreenWaves = writable([]);
export const optimizedThroughWaves = writable([]);
// Optimized waves (reverse direction, for bidirectional mode)
export const optimizedReverseGreenWaves = writable([]);
export const optimizedReverseThroughWaves = writable([]);
export const optimizedOffsets = writable([]);
export const optimizedJunctions = writable([]);
export const optimizationHistory = writable([]);
export const isOptimizing = writable(false);

// Track the state of junctions and speed when optimization was last performed
export const optimizedWaveCalculationPositions = writable([]);
export const optimizedLastCalculatedSpeed = writable(null);

// Derived store to check if optimized results are outdated
export const optimizedResultsAreOutdated = derived(
  [isResultsInvalidated, resultsInvalidationReasons],
  ([$isResultsInvalidated, $resultsInvalidationReasons]) => ({
    isOutdated: $isResultsInvalidated,
    reason: $isResultsInvalidated
      ? `Results are outdated due to: ${$resultsInvalidationReasons.join(", ")}`
      : null
  })
);

// Actual flow (vehicles per second)
export const actualFlowOptimized = derived(
  [optimizedThroughWaves, junctions, desiredFlow],
  ([$optimizedThroughWaves, $junctions, $desiredFlow]) => {
    if ($optimizedThroughWaves.length === 0 || $junctions.length === 0) return 0;

    // Filter waves with depth equal to the number of junctions
    const validWaves = $optimizedThroughWaves.filter(wave => wave.depth === $junctions.length);

    // Calculate total bandwidth of valid waves
    const totalBandwidth = validWaves.reduce((total, wave) => total + wave.bandwidth, 0);

    // Assume total cycle length is the same for all junctions
    const totalCycleLength = calculateTotalDuration($junctions[0]) || 0;

    if (totalCycleLength === 0) return 0;

    // Calculate actual flow
    const actualFlow = (totalBandwidth / totalCycleLength) * $desiredFlow;
    return actualFlow;
  }
);

// Actual intensity (vehicles per hour)
export const actualIntensityOptimized = derived(
  actualFlowOptimized,
  $actualFlowOptimized => $actualFlowOptimized * 3600
);

// Actual flow for reverse direction (vehicles per second)
export const actualReverseFlowOptimized = derived(
  [optimizedReverseThroughWaves, junctions, desiredFlow],
  ([$optimizedReverseThroughWaves, $junctions, $desiredFlow]) => {
    if ($optimizedReverseThroughWaves.length === 0 || $junctions.length === 0) return 0;

    // Filter waves with depth equal to the number of junctions
    const validWaves = $optimizedReverseThroughWaves.filter(wave => wave.depth === $junctions.length);

    // Calculate total bandwidth of valid waves
    const totalBandwidth = validWaves.reduce((total, wave) => total + wave.bandwidth, 0);

    // Assume total cycle length is the same for all junctions
    const totalCycleLength = calculateTotalDuration($junctions[0]) || 0;

    if (totalCycleLength === 0) return 0;

    return (totalBandwidth / totalCycleLength) * $desiredFlow;
  }
);

// Actual intensity for reverse direction (vehicles per hour)
export const actualReverseIntensityOptimized = derived(
  actualReverseFlowOptimized,
  $actualReverseFlowOptimized => $actualReverseFlowOptimized * 3600
);
