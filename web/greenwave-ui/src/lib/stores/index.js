// lib/stores/index.js
import { writable, derived } from 'svelte/store';

// Core data stores
export const junctions = writable([]);
export const desiredSpeed = writable(40.0);

// API result stores
export const originalGreenWaves = writable([]);
export const originalThroughWaves = writable([]);
export const showGreenWaves = writable(false);

// Track junction positions when waves were calculated
export const waveCalculationPositions = writable([]);

// Derived store to check if waves are outdated
export const wavesAreOutdated = derived(
  [junctions, originalGreenWaves, originalThroughWaves, waveCalculationPositions],
  ([$junctions, $originalGreenWaves, $originalThroughWaves, $waveCalculationPositions]) => {
    // No waves = not outdated
    if ($originalGreenWaves.length === 0 && $originalThroughWaves.length === 0) {
      return false;
    }
    
    // No stored positions = not outdated (first time)
    if ($waveCalculationPositions.length === 0) {
      return false;
    }
    
    // Different number of junctions = outdated
    if ($junctions.length !== $waveCalculationPositions.length) {
      return true;
    }
    
    // Compare positions
    return $junctions.some(junction => {
      const storedPosition = $waveCalculationPositions.find(pos => pos.id === junction.id);
      return !storedPosition || storedPosition.y !== junction.point.y;
    });
  }
);

// UI state stores
export const isLoading = writable(false);
export const error = writable(null);

// Function to store current junction positions when waves are calculated
export function storeWaveCalculationPositions(junctionsList) {
  waveCalculationPositions.set(
    junctionsList.map(j => ({ id: j.id, y: j.point.y }))
  );
}

export const DEMO_DATA = {
  junctions: [
    {
      id: 0,
      label: "Junction 1",
      cycle: [
        {
          id: 0,
          signals: [
            { duration: 30, color: "GREEN" },
            { duration: 20, color: "RED" }
          ]
        },
        {
          id: 1,
          signals: [
            { duration: 20, color: "GREEN" },
            { duration: 15, color: "RED" }
          ]
        }
      ],
      offset: 0,
      point: { x: 0, y: 0 }
    },
    {
      id: 1,
      label: "Junction 2",
      cycle: [
        {
          id: 10,
          signals: [
            { duration: 20, color: "RED" },
            { duration: 35, color: "GREEN" },
            { duration: 5, color: "YELLOW" }
          ]
        },
        {
          id: 11,
          signals: [
            { duration: 10, color: "RED" },
            { duration: 10, color: "GREEN" },
            { duration: 5, color: "YELLOW" }
          ]
        }
      ],
      offset: 0,
      point: { x: 0, y: 200 }
    },
    {
      id: 2,
      label: "Junction 3",
      cycle: [
        {
          id: 20,
          signals: [
            { duration: 45, color: "RED" },
            { duration: 10, color: "GREEN" }
          ]
        },
        {
          id: 21,
          signals: [
            { duration: 7, color: "RED" },
            { duration: 18, color: "GREEN" },
            { duration: 5, color: "YELLOW" }
          ]
        }
      ],
      offset: 0,
      point: { x: 0, y: 450 }
    },
    {
      id: 3,
      label: "Junction 4",
      cycle: [
        {
          id: 20,
          signals: [
            { duration: 40, color: "RED" },
            { duration: 15, color: "GREEN" }
          ]
        },
        {
          id: 21,
          signals: [
            { duration: 10, color: "RED" },
            { duration: 20, color: "GREEN" }
          ]
        }
      ],
      offset: 0,
      point: { x: 0, y: 600 }
    }
  ],
  desiredSpeed: 40.0
};

// Reset function to restore demo data and clear API results
export function resetToDemo() {
  junctions.set([...DEMO_DATA.junctions]); // Deep copy to avoid mutation
  desiredSpeed.set(DEMO_DATA.desiredSpeed);
  
  // Clear API results
  originalGreenWaves.set([]);
  originalThroughWaves.set([]);
  showGreenWaves.set(false);
  
  // Clear wave calculation positions
  waveCalculationPositions.set([]);
  
  // Clear UI state
  isLoading.set(false);
  error.set(null);
}

export function resetToEmpty() {
  junctions.set([]);
  desiredSpeed.set(40.0);
  
  // Clear API results
  originalGreenWaves.set([]);
  originalThroughWaves.set([]);
  showGreenWaves.set(false);
  
  // Clear wave calculation positions
  waveCalculationPositions.set([]);
  
  // Clear UI state
  isLoading.set(false);
  error.set(null);
}