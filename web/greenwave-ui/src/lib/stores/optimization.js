import { writable, derived } from 'svelte/store';
import { junctions, desiredSpeed, desiredFlow } from './core';
import { calculateTotalDuration } from '$lib/utils/junction-helpers.js';
import { resultsInvalidated } from './signals';

export const optimizedGreenWaves = writable([]);
export const optimizedThroughWaves = writable([]);
export const optimizedOffsets = writable([]);
export const optimizedJunctions = writable([]);
export const optimizationHistory = writable([]);
export const isOptimizing = writable(false);

// Track the state of junctions and speed when optimization was last performed
export const optimizedWaveCalculationPositions = writable([]);
export const optimizedLastCalculatedSpeed = writable(null);

// Derived store to check if optimized results are outdated
export const optimizedResultsAreOutdated = derived(
  [resultsInvalidated, junctions, desiredSpeed, optimizedWaveCalculationPositions, optimizedLastCalculatedSpeed],
  ([$resultsInvalidated, $junctions, $desiredSpeed, $optimizedWaveCalculationPositions, $optimizedLastCalculatedSpeed]) => {
    const reasons = [];

    // No optimized results = not outdated
    if ($optimizedWaveCalculationPositions.length === 0 || $optimizedLastCalculatedSpeed === null) {
      return { isOutdated: false, reason: null };
    }

    // Check for results invalidation
    if ($resultsInvalidated) {
      reasons.push("signal changes");
    }

    // Check for junction configuration changes
    if ($junctions.length !== $optimizedWaveCalculationPositions.length) {
      reasons.push("junction configuration changed");
    }

    // Check for speed changes
    if ($desiredSpeed !== $optimizedLastCalculatedSpeed) {
      reasons.push("desired speed changed");
    }

    // Check for junction position changes
    const positionsChanged = $junctions.some(junction => {
      const storedPosition = $optimizedWaveCalculationPositions.find(pos => pos.id === junction.id);
      return !storedPosition || storedPosition.y !== junction.point.y;
    });
    if (positionsChanged) {
      reasons.push("junction positions changed");
    }

    // If there are reasons, results are outdated
    if (reasons.length > 0) {
      return { isOutdated: true, reason: `Results outdated due to: ${reasons.join(", ")}` };
    }

    return { isOutdated: false, reason: null };
  }
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