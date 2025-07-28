import { writable, derived } from 'svelte/store';
import { junctions, desiredSpeed, desiredFlow } from './core';
import { calculateTotalDuration } from '$lib/utils/junction-helpers.js';
import { signalsInvalidated, resetSignalsInvalidation } from './signals';

// Wave-related stores
export const originalGreenWaves = writable([]);
export const originalThroughWaves = writable([]);
export const showGreenWaves = writable(false);
export const waveCalculationPositions = writable([]);
export const lastCalculatedSpeed = writable(null);

// Derived store to check if waves are outdated
export const wavesAreOutdated = derived(
  [signalsInvalidated, junctions, desiredSpeed, originalGreenWaves, originalThroughWaves, waveCalculationPositions, lastCalculatedSpeed],
  ([$signalsInvalidated, $junctions, $desiredSpeed, $originalGreenWaves, $originalThroughWaves, $waveCalculationPositions, $lastCalculatedSpeed]) => {
    const reasons = [];

    // No waves = not outdated
    if ($originalGreenWaves.length === 0 && $originalThroughWaves.length === 0) {
      return { isOutdated: false, reason: null };
    }

    // Check for signal changes
    if ($signalsInvalidated) {
      reasons.push("signal changes");
    }

    // Check for junction configuration changes
    if ($junctions.length !== $waveCalculationPositions.length) {
      reasons.push("junction configuration changed");
    }

    // Check for speed changes
    if ($desiredSpeed !== $lastCalculatedSpeed) {
      reasons.push("desired speed changed");
    }

    // Check for junction position changes
    const positionsChanged = $junctions.some(junction => {
      const storedPosition = $waveCalculationPositions.find(pos => pos.id === junction.id);
      return !storedPosition || storedPosition.y !== junction.point.y;
    });
    if (positionsChanged) {
      reasons.push("junction positions changed");
    }

    // If there are reasons, waves are outdated
    if (reasons.length > 0) {
      return { isOutdated: true, reason: `Waves outdated due to: ${reasons.join(", ")}` };
    }

    return { isOutdated: false, reason: null };
  }
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
