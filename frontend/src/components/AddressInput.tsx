import React, { useState, useRef, useEffect, useCallback } from 'react';
import { suggest, geocode } from '../api';
import { GeocodeResponse, Location } from '../types';

interface AddressInputProps {
  onLocationSelect: (location: GeocodeResponse) => void;
  provider: '2gis';
}

export const AddressInput: React.FC<AddressInputProps> = ({ onLocationSelect, provider }) => {
  const [query, setQuery] = useState('');
  const [suggestions, setSuggestions] = useState<Location[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>();
  const wrapperRef = useRef<HTMLDivElement>(null);

  const doSearch = useCallback(async (q: string) => {
    if (q.length < 3) {
      setError('Введите минимум 3 символа');
      return;
    }

    setLoading(true);
    setError(null);
    setShowSuggestions(false);
    try {
      const result = await geocode(q, provider);
      onLocationSelect(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Адрес не найден');
    } finally {
      setLoading(false);
    }
  }, [onLocationSelect, provider]);

  useEffect(() => {
    if (query.length < 3) {
      setSuggestions([]);
      setShowSuggestions(false);
      return;
    }

    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(async () => {
      try {
        const results = await suggest(query);
        setSuggestions(results);
        setShowSuggestions(results.length > 0);
      } catch {
        setSuggestions([]);
        setShowSuggestions(false);
      }
    }, 300);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [query]);

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setShowSuggestions(false);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    doSearch(query);
  };

  const handleSuggestionClick = (loc: Location) => {
    setQuery(loc.label);
    setShowSuggestions(false);
    setError(null);
    onLocationSelect({ lat: loc.lat, lon: loc.lon, label: loc.label, provider: '2gis' });
  };

  return (
    <div ref={wrapperRef} className="address-input-wrapper">
      <form onSubmit={handleSubmit} className="address-input">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Введите адрес (например: Москва, Тверская 1)"
          className="address-input-field"
        />
        <button type="submit" className="search-button" disabled={loading}>
          {loading ? '...' : '🔍'}
        </button>
      </form>
      {showSuggestions && (
        <ul className="suggestions-dropdown">
          {suggestions.map((s, i) => (
            <li key={i} onClick={() => handleSuggestionClick(s)} className="suggestion-item">
              {s.label}
            </li>
          ))}
        </ul>
      )}
      {loading && <div className="loading">Поиск...</div>}
      {error && <div className="error">{error}</div>}
    </div>
  );
};
