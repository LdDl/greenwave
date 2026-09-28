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

  const validId = value => Number.isSafeInteger(value) && value >= 0;
  const junctionIds = new Set();
  if (!Array.isArray(data.junctions)) {
    errors.push('Missing or invalid "junctions" array');
  } else {
    data.junctions.forEach((j, i) => {
      if (!j || typeof j !== 'object') { errors.push(`Junction ${i}: invalid object`); return; }
      if (!validId(j.id) || junctionIds.has(j.id)) errors.push(`Junction ${i}: invalid or duplicate ID`);
      junctionIds.add(j.id);
      if (typeof j.label !== 'string' || !j.label.trim()) errors.push(`Junction ${i}: missing "label"`);
      if (!Number.isFinite(j.point?.x) || !Number.isFinite(j.point?.y)) errors.push(`Junction ${i}: coordinates must be finite`);
      if (j.offset !== undefined && !Number.isFinite(j.offset)) errors.push(`Junction ${i}: invalid offset`);
      if (!Array.isArray(j.cycle) || j.cycle.length === 0) {
        errors.push(`Junction ${i}: missing or empty "cycle"`);
      } else {
        const phaseIds = new Set();
        j.cycle.forEach((phase, pi) => {
          if (!validId(phase?.id) || phaseIds.has(phase.id)) errors.push(`Junction ${i}, phase ${pi}: invalid or duplicate ID`);
          phaseIds.add(phase?.id);
          if (!Array.isArray(phase?.signal_groups) || phase.signal_groups.length === 0) {
            errors.push(`Junction ${i}, Phase ${pi}: missing or empty "signal_groups"`);
          } else {
            const groupIds = new Set();
            for (const group of phase.signal_groups) {
              if (!validId(group?.id) || groupIds.has(group.id)) errors.push(`Junction ${i}, phase ${pi}: invalid or duplicate group ID`);
              groupIds.add(group?.id);
              if (!Array.isArray(group?.signals) || !group.signals.length) {
                errors.push(`Junction ${i}, phase ${pi}: missing signals`);
              } else if (group.signals.some(signal => !Number.isFinite(signal?.duration) || signal.duration <= 0 || typeof signal.color !== 'string' || !signal.color)) {
                errors.push(`Junction ${i}, phase ${pi}: invalid signal duration or color`);
              }
            }
          }
        });
      }
    });
  }

  // Check desiredSpeed
  if (!Number.isFinite(data.desiredSpeed) || data.desiredSpeed <= 0) {
    errors.push('Missing or invalid "desiredSpeed" (must be positive number)');
  }

  if (data.desiredIntensity !== undefined && (!Number.isFinite(data.desiredIntensity) || data.desiredIntensity < 0)) errors.push('Desired intensity must be non-negative');
  if (data.direction !== undefined && !['forward', 'bidirectional'].includes(data.direction)) errors.push('Invalid optimization direction');
  for (const key of ['groupIds', 'reverseGroupIds']) {
    if (data[key] === undefined) continue;
    if (!data[key] || typeof data[key] !== 'object' || Array.isArray(data[key])) {
      errors.push(`Invalid ${key} corridor group selections`);
    } else {
      for (const [id, groupId] of Object.entries(data[key])) {
        const junction = Array.isArray(data.junctions) ? data.junctions.find(junction => String(junction?.id) === id) : null;
        if (!junction || !validId(groupId) || !Array.isArray(junction.cycle) || !junction.cycle.every(phase => Array.isArray(phase?.signal_groups) && phase.signal_groups.some(group => group?.id === groupId))) {
          errors.push(`Junction ${id}: selected ${key} corridor group does not exist in every phase`);
        }
      }
    }
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
export function prepareInputExport(junctions, desiredSpeed, desiredIntensity, direction = 'forward', groupIds, reverseGroupIds) {
  return {
    version: 1,
    type: 'input',
    exportedAt: new Date().toISOString(),
    desiredSpeed,
    desiredIntensity,
    direction,
    ...(groupIds === undefined ? {} : { groupIds: structuredClone(groupIds) }),
    ...(reverseGroupIds === undefined ? {} : { reverseGroupIds: structuredClone(reverseGroupIds) }),
    junctions: copyJunctionsForExport(junctions)
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
export function prepareOutputExport(optimizedJunctions, desiredSpeed, desiredIntensity, direction = 'forward', groupIds, reverseGroupIds) {
  return {
    version: 1,
    type: 'output',
    exportedAt: new Date().toISOString(),
    desiredSpeed,
    desiredIntensity,
    direction,
    ...(groupIds === undefined ? {} : { groupIds: structuredClone(groupIds) }),
    ...(reverseGroupIds === undefined ? {} : { reverseGroupIds: structuredClone(reverseGroupIds) }),
    junctions: copyJunctionsForExport(optimizedJunctions)
  };
}

function copyJunctionsForExport(junctions) {
  // Keep every group and signal property, including optional duration constraints.
  return structuredClone(junctions).map(junction => ({ ...junction, offset: junction.offset ?? 0 }));
}
