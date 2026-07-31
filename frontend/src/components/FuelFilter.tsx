import React from 'react';

const FUEL_TYPES = [
  { value: '92', label: 'АИ-92' },
  { value: '95', label: 'АИ-95' },
  { value: '98', label: 'АИ-98' },
  { value: '100', label: 'АИ-100' },
  { value: 'dt', label: 'ДТ' },
  { value: 'gas', label: 'Газ' },
];

interface FuelFilterProps {
  selectedFuel: string;
  onFuelChange: (fuel: string) => void;
}

export const FuelFilter: React.FC<FuelFilterProps> = ({ selectedFuel, onFuelChange }) => {
  return (
    <div className="fuel-filter">
      <button
        className={selectedFuel === '' ? 'active' : ''}
        onClick={() => onFuelChange('')}
      >
        Все
      </button>
      {FUEL_TYPES.map((fuel) => (
        <button
          key={fuel.value}
          className={selectedFuel === fuel.value ? 'active' : ''}
          onClick={() => onFuelChange(fuel.value)}
        >
          {fuel.label}
        </button>
      ))}
    </div>
  );
};
