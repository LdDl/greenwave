import { calculationGroupIds } from '../utils/junction-program.js';
import { apiRequest } from './base.js';

export async function optimizeOffsets(junctions, desiredSpeedKmh, optimizerType = 'genetic', optimizerParams = {}, direction = 'forward', groupSelections = {}) {
  const groupIds = calculationGroupIds(junctions, groupSelections);
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
