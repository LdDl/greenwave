import { writable, derived } from 'svelte/store';
import { junctions, desiredSpeed, desiredFlow } from './core';
import { calculateTotalDuration } from '$lib/utils/junction-helpers.js';

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
  [junctions, desiredSpeed, optimizedWaveCalculationPositions, optimizedLastCalculatedSpeed],
  ([$junctions, $desiredSpeed, $optimizedWaveCalculationPositions, $optimizedLastCalculatedSpeed]) => {
    // No optimized results = not outdated
    if ($optimizedWaveCalculationPositions.length === 0 || $optimizedLastCalculatedSpeed === null) {
      return { isOutdated: false, reason: null };
    }

    // Different number of junctions = outdated
    if ($junctions.length !== $optimizedWaveCalculationPositions.length) {
      return { isOutdated: true, reason: "Junction configuration changed" };
    }

    // Compare junction positions
    const positionsChanged = $junctions.some(junction => {
      const storedPosition = $optimizedWaveCalculationPositions.find(pos => pos.id === junction.id);
      return !storedPosition || storedPosition.y !== junction.point.y;
    });

    // Compare speed
    const speedChanged = $desiredSpeed !== $optimizedLastCalculatedSpeed;

    // Determine reason
    if (positionsChanged && speedChanged) {
      return { isOutdated: true, reason: "Both junction positions and desired speed changed" };
    } else if (positionsChanged) {
      return { isOutdated: true, reason: "Junction positions changed" };
    } else if (speedChanged) {
      return { isOutdated: true, reason: "Desired speed changed" };
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