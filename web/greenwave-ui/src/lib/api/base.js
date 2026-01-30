// In dev mode: use VITE_API_URL or default to localhost:36000
// In production: use relative path (same origin)
const DEV_API_URL = import.meta.env.VITE_API_URL || 'http://localhost:36000';
export const API_BASE = import.meta.env.DEV ? `${DEV_API_URL}/api/greenwave` : '/api/greenwave';

export class APIError extends Error {
  constructor(message, status, response) {
    super(message);
    this.name = 'APIError';
    this.status = status;
    this.response = response;
  }
}

export async function apiRequest(endpoint, options = {}) {
  const url = `${API_BASE}${endpoint}`;
  const config = {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers
    },
    ...options
  };

  try {
    const response = await fetch(url, config);
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new APIError(
        errorData.Error || `HTTP ${response.status}`,
        response.status,
        errorData
      );
    }
    
    return await response.json();
  } catch (error) {
    if (error instanceof APIError) {
      throw error;
    }
    throw new APIError(`Network error: ${error.message}`, 0, null);
  }
}