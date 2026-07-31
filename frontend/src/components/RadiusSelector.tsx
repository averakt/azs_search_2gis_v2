import React from 'react';

interface RadiusSelectorProps {
  radius: number;
  onRadiusChange: (radius: number) => void;
}

const RADII = [
  { value: 1000, label: '1 км' },
  { value: 3000, label: '3 км' },
  { value: 5000, label: '5 км' },
  { value: 10000, label: '10 км' },
];

export const RadiusSelector: React.FC<RadiusSelectorProps> = ({ radius, onRadiusChange }) => {
  return (
    <div className="radius-selector">
      <label className="radius-label">Радиус поиска:</label>
      <div className="radius-buttons">
        {RADII.map((r) => (
          <button
            key={r.value}
            className={radius === r.value ? 'active' : ''}
            onClick={() => onRadiusChange(r.value)}
          >
            {r.label}
          </button>
        ))}
      </div>
    </div>
  );
};
