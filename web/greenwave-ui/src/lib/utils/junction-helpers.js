// lib/utils/junction-helpers.js

// Calculate total duration for a junction.
// Uses the first signal group as the reference (all groups are synchronized).
export function calculateTotalDuration(junction) {
  return junction.cycle.reduce((total, phase) => {
    const signals = phase.signal_groups[0].signals;
    return total + signals.reduce((phaseTotal, signal) => {
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

// Apply offsets to junctions
// Returns a new array of deep copied junctions with offsets applied
export function applyOffsetsToJunctions(junctions, offsets) {
  return junctions.map((junction, index) => {
    const updatedJunction = JSON.parse(JSON.stringify(junction)); // Deep copy
    // Round offsets to nearest integer
    const roundedOffset = Math.round(offsets[index] || 0);
    updatedJunction.offset = roundedOffset;
    return updatedJunction;
  });
}