import { apiRequest } from './base.js';

export async function extractGreenWaves(junctions, desiredSpeedKmh) {
  return await apiRequest('/extract', {
    method: 'POST',
    body: JSON.stringify({
      junctions,
      desired_speed_kmh: desiredSpeedKmh
    })
  });
}