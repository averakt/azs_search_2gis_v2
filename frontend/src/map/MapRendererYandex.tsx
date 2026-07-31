import React, { useEffect, useRef } from 'react';
import { Station, Location } from '../types';

declare global {
  interface Window {
    ymaps3: any;
  }
}

interface MapRendererYandexProps {
  center: Location;
  stations: Station[];
  onStationClick?: (station: Station) => void;
  onMapReady?: (api: { flyTo: (lat: number, lon: number) => void }) => void;
}

export const MapRendererYandex: React.FC<MapRendererYandexProps> = ({ center, stations, onStationClick, onMapReady }) => {
  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstance = useRef<any>(null);
  const markersRef = useRef<any[]>([]);
  const stationsRef = useRef(stations);
  const onStationClickRef = useRef(onStationClick);

  stationsRef.current = stations;
  onStationClickRef.current = onStationClick;

  useEffect(() => {
    if (!mapRef.current) return;

    const loadMap = async () => {
      if (!window.ymaps3) {
        const script = document.createElement('script');
        script.src = `https://api-maps.yandex.ru/v3/?apikey=${import.meta.env.VITE_YANDEX_MAP_KEY}&lang=ru_RU`;
        script.async = true;
        script.onload = initMap;
        document.head.appendChild(script);
      } else {
        await initMap();
      }
    };

    const initMap = async () => {
      if (mapInstance.current) return;

      await window.ymaps3.ready;
      const { YMap } = window.ymaps3;

      mapInstance.current = new YMap(mapRef.current, {
        location: {
          center: [center.lon, center.lat],
          zoom: 14,
        },
      });

      if (onMapReady) {
        onMapReady({
          flyTo: (lat: number, lon: number) => {
            mapInstance.current?.update({
              location: {
                center: [lon, lat],
                zoom: 14,
              }
            });
          }
        });
      }
    };

    loadMap();

    return () => {
      if (mapInstance.current) {
        mapInstance.current.destroy();
        mapInstance.current = null;
      }
    };
  }, []);

  useEffect(() => {
    if (!mapInstance.current) return;

    markersRef.current.forEach((m) => m.remove());
    markersRef.current = [];

    const { YMapMarker } = window.ymaps3;

    stationsRef.current.forEach((station) => {
      const markerElement = document.createElement('div');
      markerElement.className = 'yandex-marker';
      markerElement.style.cssText = `
        width: 20px;
        height: 20px;
        border-radius: 50%;
        background-color: ${getMarkerColor(station)};
        border: 2px solid white;
        cursor: pointer;
      `;

      if (onStationClickRef.current) {
        markerElement.addEventListener('click', () => onStationClickRef.current?.(station));
      }

      const marker = new YMapMarker([station.lon, station.lat], markerElement);
      mapInstance.current.addChild(marker);
      markersRef.current.push(marker);
    });
  }, [stations]);

  useEffect(() => {
    if (!mapInstance.current) return;
    mapInstance.current.update({
      location: {
        center: [center.lon, center.lat],
        zoom: 14,
      }
    });
  }, [center.lat, center.lon]);

  const getMarkerColor = (station: Station): string => {
    if (station.fuels.length === 0) return '#9e9e9e';
    const hasFuel = station.fuels.some(f => f.avail === 'yes');
    return hasFuel ? '#4caf50' : '#f44336';
  };

  return <div ref={mapRef} className="map-container" />;
};
