import { GeocodeResponse, Location, StationsResponse, Station } from '../types';

const API_BASE = '/api';

const abortControllers = new Map<string, AbortController>();

const cancelRequest = (key: string) => {
  const controller = abortControllers.get(key);
  if (controller) {
    controller.abort();
    abortControllers.delete(key);
  }
};

const getController = (key: string): AbortController => {
  cancelRequest(key);
  const controller = new AbortController();
  abortControllers.set(key, controller);
  return controller;
};

const safeJsonParse = async (response: Response): Promise<any> => {
  const contentType = response.headers.get('content-type');
  if (contentType && contentType.includes('application/json')) {
    try {
      return await response.json();
    } catch {
      throw new Error(`HTTP ${response.status}`);
    }
  }
  throw new Error(`HTTP ${response.status}`);
};

export async function geocode(query: string, provider: '2gis' = '2gis'): Promise<GeocodeResponse> {
  const url = `${API_BASE}/geocode?q=${encodeURIComponent(query)}&provider=${provider}`;
  const controller = getController(`geocode:${query}:${provider}`);
  
  try {
    const response = await fetch(url, { signal: controller.signal });
    if (!response.ok) {
      const error = await safeJsonParse(response);
      throw new Error(error.error || 'Geocoding failed');
    }
    return response.json();
  } catch (err) {
    if (err instanceof Error && err.name === 'AbortError') {
      throw new Error('Request cancelled');
    }
    throw err;
  }
}

export async function suggest(query: string, limit: number = 5): Promise<Location[]> {
  const url = `${API_BASE}/suggest?q=${encodeURIComponent(query)}&limit=${limit}`;
  const controller = getController(`suggest:${query}:${limit}`);
  
  try {
    const response = await fetch(url, { signal: controller.signal });
    if (!response.ok) {
      const error = await safeJsonParse(response);
      throw new Error(error.error || 'Suggest failed');
    }
    return response.json();
  } catch (err) {
    if (err instanceof Error && err.name === 'AbortError') {
      throw new Error('Request cancelled');
    }
    throw err;
  }
}

export async function getStations(
  lat: number,
  lon: number,
  radius: number = 3000,
  fuel: string = '',
  provider: '2gis' = '2gis'
): Promise<Station[]> {
  const params = new URLSearchParams({
    lat: lat.toString(),
    lon: lon.toString(),
    radius: radius.toString(),
    provider,
  });
  if (fuel) {
    params.set('fuel', fuel);
  }
  const url = `${API_BASE}/stations?${params.toString()}`;
  const controller = getController(`stations:${lat}:${lon}:${radius}:${fuel}:${provider}`);
  
  try {
    const response = await fetch(url, { signal: controller.signal });
    if (!response.ok) {
      const error = await safeJsonParse(response);
      throw new Error(error.error || 'Failed to fetch stations');
    }
    const data: StationsResponse = await response.json();
    return data.stations;
  } catch (err) {
    if (err instanceof Error && err.name === 'AbortError') {
      throw new Error('Request cancelled');
    }
    throw err;
  }
}
