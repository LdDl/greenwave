// lib/stores/index.js
import { writable } from 'svelte/store';
import { corridorEditor } from './corridor.js';
import { DEMO_DATA } from '../utils/demo-input.js';
import { originalGreenWaves, originalThroughWaves, originalReverseGreenWaves, originalReverseThroughWaves, showGreenWaves, waveCalculationPositions, lastCalculatedSpeed } from './greenwave';
import { optimizedGroupIds, optimizedReverseGroupIds, optimizedDirection, optimizedJunctions, optimizedOffsets, optimizedGreenWaves, optimizedThroughWaves, optimizedReverseGreenWaves, optimizedReverseThroughWaves, optimizedWaveCalculationPositions, optimizedLastCalculatedSpeed, optimizedInputRevision } from './optimization';
import { validateInput, validateResults } from './invalidation';
import { optimizedCorridorName, analysisMode } from './workspace.js';
import { lastOptimizationReport } from './optimization';

// UI state stores
export const isLoading = writable(false);
export const error = writable(null);

export { DEMO_DATA } from '../utils/demo-input.js';

// Reset function to restore demo data and clear API results
export function resetToDemo() {
  corridorEditor.replaceInput(DEMO_DATA);
  clearCalculatedData();
}

export function resetToEmpty() {
  corridorEditor.replaceInput({ junctions: [], desiredSpeed: 40 });
  clearCalculatedData();
}

export function clearCalculatedData() {
  lastOptimizationReport.set(null);
  analysisMode.set('input');
  // Clear API results
  originalGreenWaves.set([]);
  originalThroughWaves.set([]);
  originalReverseGreenWaves.set([]);
  originalReverseThroughWaves.set([]);
  showGreenWaves.set(false);

  // Clear wave calculation positions
  waveCalculationPositions.set([]);
  // Clear last calculated speed
  lastCalculatedSpeed.set(null);
  optimizedJunctions.set([]);
  optimizedCorridorName.set('');
  optimizedGroupIds.set({});
  optimizedReverseGroupIds.set({});
  optimizedDirection.set('forward');
  optimizedOffsets.set([]);
  optimizedGreenWaves.set([]);
  optimizedThroughWaves.set([]);
  optimizedReverseGreenWaves.set([]);
  optimizedReverseThroughWaves.set([]);
  optimizedWaveCalculationPositions.set([]);
  optimizedLastCalculatedSpeed.set(null);
  optimizedInputRevision.set(null);
  validateInput();
  validateResults();

  // Clear UI state
  isLoading.set(false);
  error.set(null);
}
