// lib/utils/junction-helpers.js

// Calculate total duration for a junction
export function calculateTotalDuration(junction) {
  return junction.cycle.reduce((total, phase) => {
    return total + phase.signals.reduce((phaseTotal, signal) => {
      return phaseTotal + signal.duration;
    }, 0);
  }, 0);
}

// Prepare junctions for API call
export function prepareJunctionsForAPI(junctions) {
  return junctions.map(junction => ({
    ...junction,
  }));
}

// Validate that all junctions have the same total duration
export function validateJunctionCycles(junctions) {
  const durations = junctions.map(calculateTotalDuration);
  const firstDuration = durations[0];
  
  return {
    isValid: durations.every(duration => duration === firstDuration),
    durations: durations,
    commonDuration: firstDuration
  };
}