import React from 'react';
import { Station } from '../types';
import { StationCard } from './StationCard';

interface StationListProps {
  stations: Station[];
  onStationSelect: (station: Station) => void;
}

export const StationList: React.FC<StationListProps> = ({ stations, onStationSelect }) => {
  if (stations.length === 0) {
    return <div className="station-list-empty">АЗС не найдены</div>;
  }

  return (
    <div className="station-list">
      {stations.map((station) => (
        <StationCard
          key={station.id}
          station={station}
          onSelect={onStationSelect}
        />
      ))}
    </div>
  );
};
