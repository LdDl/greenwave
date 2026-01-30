import { apiRequest } from './base.js';

export async function optimizeOffsets(junctions, desiredSpeedKmh, optimizerType = 'genetic', optimizerParams = {}, direction = 'forward') {
  return await apiRequest('/optimize', {
    method: 'POST',
    body: JSON.stringify({
      junctions,
      desired_speed_kmh: desiredSpeedKmh,
      optimizer_type: optimizerType,
      optimizer_params: optimizerParams,
      direction: direction
    })
  });
}