export interface Station {
  id: string;
  name: string;
  brand: string;
  address: string;
  lat: number;
  lon: number;
  distance: number;
  fuels: Fuel[];
  queue: Queue;
  limits?: Limits;
  updated_at: string;
  source: string;
}

export interface Fuel {
  type: string;
  avail: string;
  price: number;
  currency: string;
}

export interface Queue {
  level: string;
  est_wait_min: number;
}

export interface Limits {
  max_liters: number;
  can_jerrycan: boolean;
}

export interface Location {
  lat: number;
  lon: number;
  label: string;
}

export interface GeocodeResponse {
  lat: number;
  lon: number;
  label: string;
  provider: string;
}

export interface StationsResponse {
  stations: Station[];
}
