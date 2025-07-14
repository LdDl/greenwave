// lib/stores/index.js
import { writable } from 'svelte/store';

// Core data stores
export const junctions = writable([]);
export const desiredSpeed = writable(40.0);

// API result stores
export const originalGreenWaves = writable([]);
export const originalThroughWaves = writable([]);
export const showGreenWaves = writable(false);

// UI state stores
export const isLoading = writable(false);
export const error = writable(null);

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
  
  // Clear UI state
  isLoading.set(false);
  error.set(null);
}