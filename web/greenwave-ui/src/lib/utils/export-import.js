/**
 * Export configuration to JSON file
 * @param {Object} data - Configuration data to export
 * @param {string} filename - Name of the file to download
 */
export function exportToJSON(data, filename = 'greenwave-config.json') {
  const json = JSON.stringify(data, null, 2);
  const blob = new Blob([json], { type: 'application/json' });
  const url = URL.createObjectURL(blob);

  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * Import configuration from JSON file
 * @returns {Promise<Object>} Parsed configuration data
 */
export function importFromJSON() {
  return new Promise((resolve, reject) => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json,application/json';

    input.onchange = async (e) => {
      const file = e.target.files[0];
      if (!file) {
        reject(new Error('No file selected'));
        return;
      }

      try {
        const text = await file.text();
        const data = JSON.parse(text);
        resolve(data);
      } catch (err) {
        reject(new Error(`Failed to parse JSON: ${err.message}`));
      }
    };

    input.click();
  });
}

/**
 * Validate imported configuration
 * @param {Object} data - Imported data to validate
 * @returns {Object} { isValid: boolean, errors: string[] }
 */
export function validateImportedConfig(data) {
  const errors = [];

  if (!data) {
    errors.push('Empty configuration');
    return { isValid: false, errors };
  }

  // Check junctions
  if (!Array.isArray(data.junctions)) {
    errors.push('Missing or invalid "junctions" array');
  } else {
    data.junctions.forEach((j, i) => {
      if (typeof j.id === 'undefined') errors.push(`Junction ${i}: missing "id"`);
      if (!j.label) errors.push(`Junction ${i}: missing "label"`);
      if (!j.point || typeof j.point.y === 'undefined') errors.push(`Junction ${i}: missing "point.y"`);
      if (!Array.isArray(j.cycle) || j.cycle.length === 0) {
        errors.push(`Junction ${i}: missing or empty "cycle"`);
      } else {
        j.cycle.forEach((phase, pi) => {
          if (!Array.isArray(phase.signal_groups) || phase.signal_groups.length === 0) {
            errors.push(`Junction ${i}, Phase ${pi}: missing or empty "signal_groups"`);
          }
        });
      }
    });
  }

  // Check desiredSpeed
  if (typeof data.desiredSpeed !== 'number' || data.desiredSpeed <= 0) {
    errors.push('Missing or invalid "desiredSpeed" (must be positive number)');
  }

  return {
    isValid: errors.length === 0,
    errors
  };
}

/**
 * Prepare input configuration for export
 * @param {Array} junctions - Junctions array
 * @param {number} desiredSpeed - Desired speed in km/h
 * @param {number} desiredIntensity - Desired intensity in vehicles/hour
 * @param {string} direction - Optimization direction ('forward' or 'bidirectional')
 * @returns {Object} Export-ready configuration
 */
export function prepareInputExport(junctions, desiredSpeed, desiredIntensity, direction = 'forward') {
  return {
    version: 1,
    type: 'input',
    exportedAt: new Date().toISOString(),
    desiredSpeed,
    desiredIntensity,
    direction,
    junctions: junctions.map(j => ({
      id: j.id,
      label: j.label,
      offset: j.offset || 0,
      point: { x: j.point.x, y: j.point.y },
      cycle: j.cycle.map(phase => ({
        id: phase.id,
        signal_groups: phase.signal_groups.map(sg => ({
          id: sg.id,
          signals: sg.signals.map(s => ({
            duration: s.duration,
            color: s.color
          }))
        }))
      }))
    }))
  };
}

/**
 * Prepare output (optimized) configuration for export
 * @param {Array} optimizedJunctions - Optimized junctions with offsets
 * @param {number} desiredSpeed - Desired speed in km/h
 * @param {number} desiredIntensity - Desired intensity in vehicles/hour
 * @param {string} direction - Optimization direction ('forward' or 'bidirectional')
 * @returns {Object} Export-ready configuration
 */
export function prepareOutputExport(optimizedJunctions, desiredSpeed, desiredIntensity, direction = 'forward') {
  return {
    version: 1,
    type: 'output',
    exportedAt: new Date().toISOString(),
    desiredSpeed,
    desiredIntensity,
    direction,
    junctions: optimizedJunctions.map(j => ({
      id: j.id,
      label: j.label,
      offset: j.offset || 0,
      point: { x: j.point.x, y: j.point.y },
      cycle: j.cycle.map(phase => ({
        id: phase.id,
        signal_groups: phase.signal_groups.map(sg => ({
          id: sg.id,
          signals: sg.signals.map(s => ({
            duration: s.duration,
            color: s.color
          }))
        }))
      }))
    }))
  };
}
