import React from 'react';
import { Station } from '../types';

interface StationCardProps {
  station: Station;
  onSelect: (station: Station) => void;
}

export const StationCard: React.FC<StationCardProps> = ({ station, onSelect }) => {
  const getQueueLabel = (level: string) => {
    switch (level) {
      case 'none': return 'Нет';
      case 'small': return 'Маленькая';
      case 'medium': return 'Средняя';
      case 'large': return 'Большая';
      case 'very_large': return 'Очень большая';
      default: return 'Неизвестно';
    }
  };

  const getQueueColor = (level: string) => {
    switch (level) {
      case 'small': return '#4caf50';
      case 'medium': return '#ff9800';
      case 'large': return '#f44336';
      case 'very_large': return '#b71c1c';
      default: return '#9e9e9e';
    }
  };

  const getAvailColor = (avail: string) => {
    switch (avail) {
      case 'yes': return '#4caf50';
      case 'no': return '#f44336';
      default: return '#9e9e9e';
    }
  };

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  };

  return (
    <div className="station-card" onClick={() => onSelect(station)}>
      <div className="station-header">
        <h3>{station.brand || station.name}</h3>
        <span className="distance">{Math.round(station.distance)} м</span>
      </div>
      <div className="station-address">{station.address}</div>
      <div className="station-fuels">
        {station.fuels.map((fuel, idx) => (
          <span
            key={idx}
            className="fuel-badge"
            style={{ backgroundColor: getAvailColor(fuel.avail) }}
          >
            {fuel.type}: {fuel.price > 0 ? `${fuel.price} ₽` : 'нет цены'}
          </span>
        ))}
      </div>
      {station.queue.level && station.queue.level !== 'unknown' && (
        <div className="station-queue">
          Очередь: <span style={{ color: getQueueColor(station.queue.level) }}>
            {getQueueLabel(station.queue.level)} (~{station.queue.est_wait_min} мин)
          </span>
        </div>
      )}
      {station.limits && (
        <div className="station-limits">
          {station.limits.max_liters > 0 && (
            <span>Лимит: {station.limits.max_liters}л</span>
          )}
          {station.limits.can_jerrycan && <span>В тару ✓</span>}
        </div>
      )}
      <div className="station-updated">
        Обновлено: {formatTime(station.updated_at)}
      </div>
    </div>
  );
};
