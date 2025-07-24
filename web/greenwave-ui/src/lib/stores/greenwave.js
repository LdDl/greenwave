import { writable, derived } from 'svelte/store';
import { junctions, desiredSpeed } from './core';

// Wave-related stores
export const originalGreenWaves = writable([]);
export const originalThroughWaves = writable([]);
export const showGreenWaves = writable(false);
export const waveCalculationPositions = writable([]);
export const lastCalculatedSpeed = writable(null);

// Derived store to check if waves are outdated
export const wavesAreOutdated = derived(
  [junctions, desiredSpeed, originalGreenWaves, originalThroughWaves, waveCalculationPositions, lastCalculatedSpeed],
  ([$junctions, $desiredSpeed, $originalGreenWaves, $originalThroughWaves, $waveCalculationPositions, $lastCalculatedSpeed]) => {
    // No waves = not outdated
    if ($originalGreenWaves.length === 0 && $originalThroughWaves.length === 0) {
      return false;
    }

    // Different number of junctions = outdated
    if ($junctions.length !== $waveCalculationPositions.length) {
      return { isOutdated: true, reason: "Junction configuration changed" };
    }

    // Compare speed
    const speedChanged = $desiredSpeed !== $lastCalculatedSpeed;

    // Compare junction positions
    const positionsChanged = $junctions.some(junction => {
      const storedPosition = $waveCalculationPositions.find(pos => pos.id === junction.id);
      return !storedPosition || storedPosition.y !== junction.point.y;
    });

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

// Function to store current junction positions when waves are calculated
export function storeWaveCalculationPositions(junctionsList, currentSpeed) {
  waveCalculationPositions.set(
    junctionsList.map(j => ({ id: j.id, y: j.point.y }))
  );
  lastCalculatedSpeed.set(currentSpeed);
}