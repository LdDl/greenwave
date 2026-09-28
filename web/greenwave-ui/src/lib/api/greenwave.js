import { calculationGroupIds } from '../utils/junction-program.js';
import { apiRequest } from './base.js';

export async function extractGreenWaves(junctions, desiredSpeedKmh, direction = 'forward', groupSelections = {}) {
  const groupIds = calculationGroupIds(junctions, groupSelections);
  return await apiRequest('/extract', {
    method: 'POST',
    body: JSON.stringify({
      junctions,
      desired_speed_kmh: desiredSpeedKmh,
      direction: direction,
      group_ids: groupIds
    })
  });
}
