import React, { useEffect, useRef } from 'react';
import { Station, Location } from '../types';
import { ensureDGLoaded } from './dg';

interface MapRenderer2GisProps {
  center: Location;
  stations: Station[];
  onStationClick?: (station: Station) => void;
  onMapReady?: (api: { flyTo: (lat: number, lon: number) => void }) => void;
}

const escapeHtml = (str: string): string => {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
};

export const MapRenderer2Gis: React.FC<MapRenderer2GisProps> = ({ center, stations, onStationClick, onMapReady }) => {
  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstance = useRef<any>(null);
  const markersRef = useRef<any[]>([]);
  const centerRef = useRef(center);
  const stationsRef = useRef(stations);

  centerRef.current = center;
  stationsRef.current = stations;

  const renderMarkers = () => {
    if (!mapInstance.current) return;
    markersRef.current.forEach((m) => m.remove());
    markersRef.current = [];
    stationsRef.current.forEach((station) => {
      const marker = window.DG.marker([station.lat, station.lon]).addTo(mapInstance.current);
      if (station.brand || station.name) {
        const title = escapeHtml(station.brand || station.name);
        const address = escapeHtml(station.address);
        marker.bindPopup(`<b>${title}</b><br>${address}`);
      }
      if (onStationClick) {
        marker.on('click', () => onStationClick(station));
      }
      markersRef.current.push(marker);
    });
  };

  useEffect(() => {
    let destroyed = false;

    const init = async () => {
      try {
        await ensureDGLoaded(import.meta.env.VITE_2GIS_MAPGL_KEY);
        if (destroyed || !mapRef.current || mapInstance.current) return;

        mapInstance.current = window.DG.map(mapRef.current, {
          center: [centerRef.current.lat, centerRef.current.lon],
          zoom: 14,
        });
        renderMarkers();
        
        if (onMapReady) {
          onMapReady({
            flyTo: (lat: number, lon: number) => {
              mapInstance.current?.panTo([lat, lon], { animate: true });
            }
          });
        }
      } catch (err) {
        console.error('Failed to init 2GIS map:', err);
      }
    };

    init();

    return () => {
      destroyed = true;
      markersRef.current.forEach((m) => m.remove());
      markersRef.current = [];
      if (mapInstance.current) {
        mapInstance.current.remove();
        mapInstance.current = null;
      }
    };
  }, []);

  useEffect(() => {
    if (!mapInstance.current) return;
    mapInstance.current.panTo([center.lat, center.lon]);
  }, [center.lat, center.lon]);

  useEffect(() => {
    if (!mapInstance.current) return;
    renderMarkers();
  }, [stations]);

  return <div ref={mapRef} className="map-container" />;
};
