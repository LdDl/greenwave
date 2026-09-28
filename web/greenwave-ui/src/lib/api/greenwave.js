import { calculationGroupIds } from '../utils/junction-program.js';
import { apiRequest } from './base.js';

export async function extractGreenWaves(junctions, desiredSpeedKmh, direction = 'forward', groupSelections = {}, reverseGroupSelections = {}) {
  const groupIds = calculationGroupIds(junctions, groupSelections);
  return await apiRequest('/extract', {
    method: 'POST',
    body: JSON.stringify({
      junctions,
      desired_speed_kmh: desiredSpeedKmh,
      direction: direction,
      group_ids: groupIds,
      ...(direction === 'bidirectional' ? { reverse_group_ids: calculationGroupIds(junctions, { ...groupSelections, ...reverseGroupSelections }) } : {})
    })
  });
}
