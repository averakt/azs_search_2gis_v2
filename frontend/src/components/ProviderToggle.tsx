import React from 'react';

export const ProviderToggle: React.FC = () => {
  return (
    <div className="provider-toggle">
      <button
        type="button"
        className="active"
        title="2GIS API (требует ключ, есть данные о топливе)"
      >
        2ГИС
      </button>
      <span style={{
        padding: '8px 16px',
        border: '1px solid rgba(255,255,255,0.3)',
        background: 'rgba(255,255,255,0.2)',
        color: 'rgba(255,255,255,0.5)',
        borderRadius: '4px',
        fontSize: '14px',
        cursor: 'not-allowed',
        opacity: 0.5,
        display: 'inline-block',
        userSelect: 'none',
        marginLeft: '8px'
      }}>
        Яндекс
      </span>
    </div>
  );
};
