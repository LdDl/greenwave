import { writable, derived } from 'svelte/store';
import { junctions, desiredSpeed, desiredFlow } from './core';
import { calculateTotalDuration } from '$lib/utils/junction-helpers.js';
import { signalsInvalidated, resetSignalsInvalidation } from './signals';
import { inputInvalidationReasons, isInputInvalidated } from '$lib/stores/invalidation';

// Wave-related stores (forward direction)
export const originalGreenWaves = writable([]);
export const originalThroughWaves = writable([]);
// Wave-related stores (reverse direction, for bidirectional mode)
export const originalReverseGreenWaves = writable([]);
export const originalReverseThroughWaves = writable([]);
export const showGreenWaves = writable(false);
export const waveCalculationPositions = writable([]);
export const lastCalculatedSpeed = writable(null);

// Derived store to check if waves are outdated
export const wavesAreOutdated = derived(
  [isInputInvalidated, inputInvalidationReasons],
  ([$isInputInvalidated, $inputInvalidationReasons]) => ({
    isOutdated: $isInputInvalidated,
    reason: $isInputInvalidated
      ? `Input data is outdated due to: ${$inputInvalidationReasons.join(", ")}`
      : null
  })
);

// Function to store current junction positions when waves are calculated
export function storeWaveCalculationPositions(junctionsList, currentSpeed) {
  waveCalculationPositions.set(
    junctionsList.map(j => ({ id: j.id, y: j.point.y }))
  );
  lastCalculatedSpeed.set(currentSpeed);
  resetSignalsInvalidation();
}

// Actual flow for input data plot (vehicles per second)
export const actualFlow = derived(
  [originalThroughWaves, junctions, desiredFlow],
  ([$originalThroughWaves, $junctions, $desiredFlow]) => {
    if ($originalThroughWaves.length === 0 || $junctions.length === 0) return 0;

    // Filter waves with depth equal to the number of junctions
    const validWaves = $originalThroughWaves.filter(wave => wave.depth === $junctions.length);

    // Calculate total bandwidth of valid waves
    const totalBandwidth = validWaves.reduce((total, wave) => total + wave.bandwidth, 0);

    // Assume total cycle length is the same for all junctions
    const totalCycleLength = calculateTotalDuration($junctions[0]) || 0;

    if (totalCycleLength === 0) return 0;

    // Calculate actual flow
    return (totalBandwidth / totalCycleLength) * $desiredFlow;
  }
);

// Actual intensity for input data plot (vehicles per hour)
export const actualIntensity = derived(actualFlow, $actualFlow => $actualFlow * 3600);

// Actual flow for reverse direction (vehicles per second)
export const actualReverseFlow = derived(
  [originalReverseThroughWaves, junctions, desiredFlow],
  ([$originalReverseThroughWaves, $junctions, $desiredFlow]) => {
    if ($originalReverseThroughWaves.length === 0 || $junctions.length === 0) return 0;

    // Filter waves with depth equal to the number of junctions
    const validWaves = $originalReverseThroughWaves.filter(wave => wave.depth === $junctions.length);

    // Calculate total bandwidth of valid waves
    const totalBandwidth = validWaves.reduce((total, wave) => total + wave.bandwidth, 0);

    // Assume total cycle length is the same for all junctions
    const totalCycleLength = calculateTotalDuration($junctions[0]) || 0;

    if (totalCycleLength === 0) return 0;

    // Calculate actual flow
    return (totalBandwidth / totalCycleLength) * $desiredFlow;
  }
);

// Actual intensity for reverse direction (vehicles per hour)
export const actualReverseIntensity = derived(actualReverseFlow, $actualReverseFlow => $actualReverseFlow * 3600);
