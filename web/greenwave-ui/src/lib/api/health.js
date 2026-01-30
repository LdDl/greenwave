import { apiRequest } from './base.js';

export async function healthCheck() {
  return await apiRequest('/health');
}