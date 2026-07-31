import { useState, useMemo, useCallback } from 'react';
import { AddressInput } from './components/AddressInput';
import { FuelFilter } from './components/FuelFilter';
import { RadiusSelector } from './components/RadiusSelector';
import { StationList } from './components/StationList';
import { MapRenderer2Gis } from './map/MapRenderer2Gis';
import { getStations } from './api';
import { Station, Location, GeocodeResponse } from './types';
import './App.css';

function App() {
  const [location, setLocation] = useState<Location | null>(null);
  const [stations, setStations] = useState<Station[]>([]);
  const [selectedFuel, setSelectedFuel] = useState('');
  const [radius, setRadius] = useState(3000);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const provider: '2gis' = '2gis';
  const defaultCenter = useMemo(() => ({ lat: 55.751244, lon: 37.618423, label: 'Москва' }), []);
  const mapCenter = location || defaultCenter;

  const handleLocationSelect = useCallback(async (geoResponse: GeocodeResponse) => {
    const newLocation = {
      lat: geoResponse.lat,
      lon: geoResponse.lon,
      label: geoResponse.label,
    };
    setLocation(newLocation);
    setError(null);
    setLoading(true);

    try {
      const fetchedStations = await getStations(newLocation.lat, newLocation.lon, radius, selectedFuel, provider);
      setStations(fetchedStations);
    } catch (err) {
      if (err instanceof Error && err.message === 'Request cancelled') {
        return;
      }
      setError(err instanceof Error ? err.message : 'Ошибка загрузки АЗС');
      setStations([]);
    } finally {
      setLoading(false);
    }
  }, [selectedFuel, provider, radius]);

  const handleFuelChange = useCallback(async (fuel: string) => {
    setSelectedFuel(fuel);
    if (!location) return;

    setLoading(true);
    setError(null);
    try {
      const fetchedStations = await getStations(location.lat, location.lon, radius, fuel, provider);
      setStations(fetchedStations);
    } catch (err) {
      if (err instanceof Error && err.message === 'Request cancelled') {
        return;
      }
      setError(err instanceof Error ? err.message : 'Ошибка загрузки АЗС');
      setStations([]);
    } finally {
      setLoading(false);
    }
  }, [location, provider, radius]);

  const handleRadiusChange = useCallback(async (newRadius: number) => {
    setRadius(newRadius);
    if (!location) return;

    setLoading(true);
    setError(null);
    try {
      const fetchedStations = await getStations(location.lat, location.lon, newRadius, selectedFuel, provider);
      setStations(fetchedStations);
    } catch (err) {
      if (err instanceof Error && err.message === 'Request cancelled') {
        return;
      }
      setError(err instanceof Error ? err.message : 'Ошибка загрузки АЗС');
      setStations([]);
    } finally {
      setLoading(false);
    }
  }, [location, selectedFuel, provider]);

  return (
    <div className="app">
      <header className="app-header">
        <h1>АЗС Поиск</h1>
      </header>

      <div className="controls">
        <AddressInput onLocationSelect={handleLocationSelect} provider={provider} />
        <FuelFilter selectedFuel={selectedFuel} onFuelChange={handleFuelChange} />
        <RadiusSelector radius={radius} onRadiusChange={handleRadiusChange} />
      </div>

      <div className="main-content">
        <div className="map-section">
          <MapRenderer2Gis
            center={mapCenter}
            stations={stations}
          />
        </div>

        <div className="stations-section">
          {loading && <div className="loading-overlay">Загрузка...</div>}
          {error && <div className="error-overlay">{error}</div>}
          <StationList stations={stations} onStationSelect={(_station: Station) => {}} />
        </div>
      </div>

      <div className="disclaimer">
        Данные о наличии топлива предоставлены 2ГИС. Очередь — приблизительная оценка.
      </div>
    </div>
  );
}

export default App;
