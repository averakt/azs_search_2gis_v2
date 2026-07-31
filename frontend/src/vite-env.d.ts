/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_2GIS_MAPGL_KEY: string
  readonly VITE_YANDEX_MAP_KEY: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
