declare global {
  interface Window {
    DG?: any;
  }
}

let loadPromise: Promise<void> | null = null;

export function ensureDGLoaded(key: string): Promise<void> {
  if (loadPromise) return loadPromise;

  loadPromise = new Promise<void>((resolve, reject) => {
    if (window.DG) {
      window.DG.then(() => resolve());
      return;
    }

    const script = document.createElement('script');
    script.src = `https://maps.api.2gis.ru/2.0/loader.js?pkg=full&key=${key}`;
    script.async = true;
    script.onload = () => {
      if (window.DG) {
        window.DG.then(() => resolve());
      } else {
        reject(new Error('2GIS DG loaded but window.DG not found'));
      }
    };
    script.onerror = () => reject(new Error('Failed to load 2GIS DG script'));
    document.head.appendChild(script);
  });

  return loadPromise;
}
