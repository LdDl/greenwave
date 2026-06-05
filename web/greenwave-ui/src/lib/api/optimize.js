import { apiRequest } from './base.js';

export async function optimizeOffsets(junctions, desiredSpeedKmh, optimizerType = 'genetic', optimizerParams = {}, direction = 'forward') {
  // Derive group_ids from the first signal group of each junction.
  // Since the UI operates with a single group per junction, we always use group 0.
  const groupIds = Object.fromEntries(junctions.map(j => [j.id, j.cycle[0].signal_groups[0].id]));
  return await apiRequest('/optimize', {
    method: 'POST',
    body: JSON.stringify({
      junctions,
      desired_speed_kmh: desiredSpeedKmh,
      optimizer_type: optimizerType,
      optimizer_params: optimizerParams,
      direction: direction,
      group_ids: groupIds
    })
  });
}