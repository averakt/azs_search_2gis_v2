# План реализации: azs_search_2gis_v2

> Документ для передачи модели-имплементатору. Содержит цель, зафиксированные решения, опорные факты ресёрча, архитектуру, спецификации бэка/фронта, Docker, пофазовый план, критерии приёмки и риски. Самодостаточен — отдельный ресёрч не требуется.

---

## 0. Контекст и цель

Веб-приложение: пользователь вводит адрес → на карте показываются ближайшие АЗС с **наличием топлива, ценами и оценкой очереди**. Данные реальные (не замокированные). Упаковка в Docker, `docker compose up` → приложение работает. Поиск «как в Яндекс.Картах, так и в 2ГИС» = переключаемый движок карты. MVP — Москва.

Тип проекта: пет-проект, ToS-серая зона использования неофициального API принята осознанно.

---

## 1. Зафиксированные решения (от заказчика)

| Решение | Выбор |
|---|---|
| Источник наличия/цены/очереди | Серверный reverse-engineer внутреннего API 2ГИС `benzin.api.2gis.ru` (авторизация — JWE-токен от `passepartout.2gis.com`), с опциональным фолбэком на headless-перехват |
| Backend | **Go** |
| Frontend | **React + Vite** |
| Охват MVP | Москва (`region_id=32`) |
| Кэш | In-memory в процессе, TTL ~10 мин (Redis откладывается) |
| Модель-имплементатор | Qwen 3.5 Plus (по `PROJECT.md`) |

---

## 2. Опорные факты из ресёрча (передано имплементатору — экономит дни)

### 2.1. Живой источник данных — главный

- Базовый URL: `https://benzin.api.2gis.ru/` (config-ключ `benzApiUrl`, флаг `featureBenzin:true`).
- Авторизация: JWE-токен (алгоритм `RSA-OAEP-256` + `A256GCM`), получается с `https://passepartout.2gis.com` (React Query key `passepartout/getToken`). На `benzin.2gis.ru` токен гидрируется в `__REACT_QUERY_STATE__`.
- Фильтры (query-параметры API): `benzin_ai_92`, `benzin_ai_95`, `benzin_ai_98`, `benzin_ai_100`, `benzin_dt` (ДТ), `benzin_gas` (газ), `benzin_from_25_to_50` (очередь до 50 авто).
- **Точный sub-path в HTML отсутствует** — собирается в JS-бандлах. Кандидатные бандлы: `app.1a42148734cd1710dd59.js`, `core.76b30fc0f81e73bed1b1.js`, `catalogMarkers.75c56522020823661161.js`, `map3d.ad6cbb29e380f2b90da6.js`. Точный path снимается живым перехватом (см. Фазу 0).
- Источник данных 2ГИС: обезличенные транзакции Сбера + краудсорсинг пользователей; охват ~29 000 АЗС по РФ.
- Живые значения цены/очереди в HTML `benzin.2gis.ru` отсутствуют (`price`=0, `queue`=0 в `initialState`) — грузятся рантайм из `benzin.api.2gis.ru`.

### 2.2. Официальные API 2ГИС (имя/адрес/часы/телефон — БЕЗ наличия/цены)

- Geocoder: `GET https://catalog.api.2gis.com/3.0/items/geocode?q=...&fields=items.point&key=KEY`
- Reverse geocode: `GET https://catalog.api.2gis.com/3.0/items/geocode?lat=&lon=&fields=items.point&key=KEY`
- Places/поиск по рубрике «Заправочные станции»: rubric id `18547` (alias `zapravochnye_stancii`); есть «Онлайн-заправки 2ГИС» id `112667`.
- Топливные теги рубрикатора: `petrol_station_fuel_unleaded_92/95/98/100`, `…_diesel`, `…_eurodiesel` (ДТ+), `…_lpg` (пропан), `…_cng` (метан). Примеры id: `70000201006748159` (92), `70000201006748161` (95), `70000201006748165` (ДТ), `70000201006748168` (100), `70000201006748172` (ДТ+).
- Документация: `docs.2gis.com`.
- Ключ: получить собственный в партнёрском кабинете 2ГИС. **Демо-ключ `c7f1a769-c8a5-4636-b14d-d8c987808a12` — это собственный web-key 2ГИС, переиспользовать нельзя.**

### 2.3. Карты

- **2GIS MapGL JS** (CDN `mapgl.2gis.com`), режим 3D. Тайлы `tile0–3.maps.2gis.com/v2`. Стили `styles.api.2gis.com`, ids: `d4f7ed0c-2289-4979-b43b-e47eceb5c134` (realty), `760af037-7450-48cd-ae98-49dc13902a05` (gray), `eb10e2c3-3c28-4b81-b74b-859c9c4cf47e` (custom). Векторные тайлы `jakartaVectorTilesKey: 10153539-2026-4a0c-b7a3-52ddb3fed411`.
- **Yandex JS API v3** (`yandex.ru/dev/jsapi30/doc/ru/ref/`). Требует собственный ключ.

### 2.4. Альтернативы и что НЕ использовать

- **OSM Overpass** (`amenity=fuel`) — фолбэк-источник координат АЗС, бесплатно, без ключа, GeoJSON. Наличия/цен нет.
- **Яндекс.Заправки / Tanker** (`app.tanker.yandex.net`) — gated, по форумам падает в 404 даже с выданным ключом. **Не использовать.**

---

## 3. Структура репозитория

```
azs_search_2gis_v2/
├─ backend/
│  ├─ cmd/server/main.go            # точка входа, конфиг, http server
│  ├─ internal/
│  │  ├─ config/                    # env-конфиг
│  │  ├─ httpapi/                   # хендлеры (chi): geocode, stations, health
│  │  ├─ provider/                  # interface Provider + TwoGis, Yandex
│  │  ├─ geocode/                   # геокодирование (2gis/yandex)
│  │  ├─ stations/                  # поиск АЗС в радиусе + обогащение
│  │  ├─ benzin/                    # клиент benzin.api.2gis.ru + passepartout
│  │  ├─ cache/                     # in-memory TTL (go-cache) + singleflight
│  │  ├─ headless/                  # chromedp-перехват (фолбэк, опц.)
│  │  └─ model/                     # доменные типы (Station, Fuel, Queue…)
│  ├─ go.mod / go.sum
│  └─ Dockerfile (multi-stage → distroless)
├─ frontend/
│  ├─ src/
│  │  ├─ main.tsx, App.tsx
│  │  ├─ api/                       # клиент к /api/*
│  │  ├─ components/                # AddressInput, MapView, StationCard, ProviderToggle, FuelFilter
│  │  ├─ map/                       # MapRenderer2Gis.tsx, MapRendererYandex.tsx, общий интерфейс
│  │  └─ types/
│  ├─ vite.config.ts, package.json, tsconfig.json
│  └─ Dockerfile (build → nginx serve)
├─ docs/
│  └─ BENZIN_API.md                 # результат Фазы 0: точный endpoint, авторизация, схема ответа
├─ docker-compose.yml
├─ .env.example
├─ Makefile                         # make dev / make test / make lint / make build
├─ PROJECT.md (есть)
└─ README.md (инструкция запуска)
```

---

## 4. Backend — спецификация (Go)

### 4.1. Зависимости

- `github.com/go-chi/chi/v5` — роутер.
- `github.com/patrickmn/go-cache` — TTL-кэш in-memory.
- `golang.org/x/sync/singleflight` — дедупликация параллельных одинаковых запросов.
- `github.com/joho/godotenv` — env.
- `log/slog` — логи (stdlib).
- HTTP-клиент — stdlib `net/http` с явными таймаутами (5–15 с).
- `github.com/chromedp/chromedp` — **только** если включается headless-фолбэк (увеличивает образ, нужен chromium в контейнере).
- Go 1.22+.

### 4.2. Конфиг (`.env`)

```
PORT=8080
TWOGIS_API_KEY=           # собственный ключ 2GIS (партнёрский кабинет)
YANDEX_API_KEY=           # собственный ключ Yandex JS API / geocoder
BENZIN_BASE_URL=https://benzin.api.2gis.ru/
PASSEPARTOUT_URL=https://passepartout.2gis.com
CACHE_TTL=10m
HTTP_TIMEOUT=10s
DEFAULT_RADIUS=3000
ENABLE_HEADLESS_FALLBACK=false
LOG_LEVEL=info
```

### 4.3. Доменные типы (`internal/model`)

```go
type Station struct {
    ID, Name, Brand, Address string
    Lat, Lon                 float64
    Distance                 float64 // м, от точки пользователя
    Fuels                    []Fuel
    Queue                    Queue
    Limits                   *Limits
    UpdatedAt                time.Time
    Source                   string // "2gis-benzin" | "2gis-catalog" | "overpass"
}

type Fuel struct {
    Type     string  // "АИ-92"|"АИ-95"|"АИ-98"|"АИ-100"|"ДТ"|"ДТ+"|"Пропан"|"Метан"
    Avail    string  // "yes"|"no"|"unknown"
    Price    float64 // руб/л, 0 если нет
    Currency string
}

type Queue struct {
    Level       string // "none"|"small"|"medium"|"large"
    EstWaitMin  int    // эвристическая оценка минут
}

type Limits struct {
    MaxLiters   int
    CanJerrycan bool
}
```

### 4.4. Интерфейс провайдера

```go
type Provider interface {
    Geocode(ctx context.Context, q string) (Location, error)
    SearchStations(ctx context.Context, loc Location, radius int, fuel string) ([]Station, error)
}
```

- `TwoGis` — Geocoder/Places (официальные) + обогащение через `BenzinClient`.
- `Yandex` — геокодер Яндекса + поиск мест; наличие≈нет → `Avail="unknown"`, пометка в UI.

### 4.5. Эндпоинты (`internal/httpapi`)

- `GET /api/health` — liveness.
- `GET /api/geocode?q=&provider=2gis|yandex` → `{lat, lon, label, provider}`. Кэш по `q+provider`.
- `GET /api/stations?lat=&lon=&radius=&fuel=&provider=` → `[{Station}]` (JSON, GeoJSON-friendly), отсортировано по `Distance`. Кэш по `lat,lon,radius,fuel,provider` (округление lat/lon до ~3 знаков для hit-рейта).

### 4.6. BenzinClient (`internal/benzin`)

1. `getToken` → запрос к `passepartout.2gis.com` (точный метод/параметры — снять в Фазе 0). Кэш токена с TTL меньше срока жизни токена. Заголовок `Authorization: Bearer <JWE>`.
2. `getStations(bbox|point+radius, filters)` → `benzin.api.2gis.ru/<path>?…` (path — Фаза 0). Маппинг ответа → `model.Station`. Маппинг очереди: уровни/`benzin_from_25_to_50` → `Queue.Level` + эвристика `EstWaitMin` (например: small≈10, medium≈25, large≈45 мин — константы вынести в конфиг).
3. Ретрай с backoff (3 попытки): на 401 → перевыпуск токена; на 429/5xx → вернуть кэш с пометкой `stale`.

### 4.7. Деградация

Если `benzin` недоступен — вернуть АЗС из 2GIS Places/Overpass с `Avail="unknown"` и `Source="2gis-catalog"`. UI показывает «данные о наличии временно недоступны». Приложение не падает.

### 4.8. Кэш

`go-cache` с TTL 10 мин, `singleflight` на одинаковые ключи. Метрики попаданий в логи (slog).

---

## 5. Frontend — спецификация (React + Vite + TS)

### 5.1. Зависимости

- React 18, Vite 5, TypeScript.
- SDK карт грузятся динамически через `<script>` (не npm): 2GIS MapGL JS, Yandex JS API v3.
- Стили — минимальный CSS, без тяжёлого UI-фреймворка (опц. lightweight).

### 5.2. Компоненты

- `App` — состояние: `provider`, `location`, `stations`, `selectedFuel`, `loading`, `error`.
- `AddressInput` — инпут + debounce 300 мс → `GET /api/geocode`; выпадающий список адресов; выбор → установка `location`.
- `ProviderToggle` — переключатель «2ГИС» / «Яндекс» (влияет на рендерер карты; источник наличия — 2ГИС в обоих режимах).
- `MapContainer` — по `provider` монтирует `MapRenderer2Gis` или `MapRendererYandex`. Общий интерфейс:
  ```ts
  interface MapRenderer {
    setCenter(loc: Location): void;
    renderMarkers(stations: Station[]): void;
    flyTo(station: Station): void;
  }
  ```
- `StationCard` / `StationList` — список рядом: топливо (бейджи: есть/нет/нет данных), цена, очередь (цвет + «~X мин»), лимиты, «обновлено N мин назад», расстояние, «построить маршрут».
- `FuelFilter` — чипсы: АИ-92/95/98/100, ДТ, Газ → `selectedFuel` → параметр `fuel` в `/api/stations`.

### 5.3. Маркеры

Цвет = наличие (зелёный/красный/серый), размер = очередь (small/med/large). Клик → карточка + `flyTo`.

### 5.4. UX-пометки

- «Данные о наличии предоставлены 2ГИС».
- «обновлено в HH:MM».
- «очередь — приблизительная оценка».

---

## 6. Docker

`docker-compose.yml`:

- **backend**: образ из `backend/Dockerfile` (multi-stage: `golang:1.22-alpine` build → `gcr.io/distroless/static` или `alpine`), env из `.env`, порт 8080. `chromedp`/chromium — только при `ENABLE_HEADLESS_FALLBACK=true`.
- **frontend**: образ из `frontend/Dockerfile` (`node:20-alpine` build → `nginx:alpine` serve на 80), прокси `/api/*` → `backend:8080` через nginx.

Цель: `docker compose up --build` → приложение открывается на `localhost:80`.

---

## 7. План реализации (фазы для имплементатора)

### Фаза 0 — Де-риск источника (ПЕРВАЯ, блокирующая)

Перехватить сеть `benzin.2gis.ru` (devtools или Playwright):

1. Зафиксировать запрос `passepartout/getToken`: метод, URL, query/body, заголовки, привязка к origin/UA/IP, TTL токена.
2. Зафиксировать запрос(ы) к `benzin.api.2gis.ru`: точный path, query, заголовок авторизации, форма ответа (структура полей станций/топлива/очереди/лимитов).
3. Воспроизвести запросы из Go-скрипта с другого IP/UA → подтвердить 200 и структуру ответа.
4. Решение: «серверный replay работает» / «нужен headless».
5. Результат: документ `docs/BENZIN_API.md` с примерами запрос/ответ и решением.

### Фаза 1 — Backend-скелет

Go-модуль, config, chi-роутер, `/api/health`, `/api/geocode` (2GIS, официально), in-memory кэш + singleflight, логи (slog), `Makefile`, `go vet`/`gofmt`.

### Фаза 2 — Поиск АЗС + BenzinClient

`/api/stations` через 2GIS Places (rubric `18547`) + обогащение из `benzin` (по результатам Фазы 0). Маппинг очереди, деградация, кэш. Эндпоинт возвращает JSON.

### Фаза 3 — Frontend MVP

Vite+React+TS, `AddressInput`+геокодер, `ProviderToggle`, 2GIS MapGL-рендерер, маркеры, список/карточки. Сначала один рендерер.

### Фаза 4 — Yandex-рендерер + фильтры топлива

Второй ренддер (Yandex JS API v3) через общий интерфейс; `FuelFilter`.

### Фаза 5 — Docker + полировка

`docker-compose`, Dockerfiles, nginx-прокси, `.env.example`, README с инструкцией запуска, пометки об источнике данных и ToS.

### Фаза 6 — Качество (по `PROJECT.md`)

- Тесты: `go test` (юниты маппинга/кэша/деградации; моки HTTP через `httptest`), фронт — базовые тесты компонентов.
- Линт: `golangci-lint`/`gofmt` (backend), `eslint` (frontend).
- `make build` зелёный.

---

## 8. Критерии приёмки

- `docker compose up` поднимает приложение, открывается на `localhost`.
- Вводишь московский адрес → карта центрируется, видны маркеры ближайших АЗС с реальным наличием/ценами/очередью (источник — 2ГИС benzin, не моки).
- Переключатель карты 2ГИС ↔ Яндекс работает; данные АЗС одни и те же.
- Фильтр по типу топлива работает.
- В карточке: цена, очередь (~мин), лимиты, «обновлено …».
- При недоступности benzin — АЗС показываются с «нет данных о наличии», приложение не падает.
- `go test`, линт, `make build` — зелёные.

---

## 9. Риски и митигация

| Риск | Митигация |
|---|---|
| `benzin`-токен origin/UA-bound | Фаза 0 решает; фолбэк headless (chromedp); пометка «данные 2ГИС» |
| Методы API меняются | Изоляция в `internal/benzin`, версионирование, кэш-стабилизация |
| ToS-серая зона | Пет-проект, явная атрибуция, не для коммерции |
| Демо-ключи с лимитами | Кэш + singleflight + rate-limit; получение собственных ключей 2GIS/Yandex |
| Очередь — не минуты, а уровни | Эвристическая оценка, явная пометка «приблизительно» |
| chromedp утяжеляет образ | Подключать только при `ENABLE_HEADLESS_FALLBACK=true` |
| Свежесть данных (~30 мин) | Метка `updatedAt` / «обновлено …» в UI |

---

## 10. Ссылки и источники

- 2ГИС «Карта бензина»: `benzin.2gis.ru` (запущен июль 2026, ~29 000 АЗС).
- 2GIS API docs: `docs.2gis.com` (Geocoder, Places, Categories).
- Yandex JS API v3: `yandex.ru/dev/jsapi30/doc/ru/ref/`.
- OSM Overpass: `overpass-api.de`, тег `amenity=fuel`.
- Ресёрч-новости: kod.ru, vc.ru, finance.mail.ru (запуск карты бензина 2ГИС, партнёрство со Сбером).

---

## 11. Рабочий процесс (по `PROJECT.md`)

- Architecture: GLM 5.2 (этот документ).
- Implementation: Qwen 3.5 Plus.
- Testing: DeepSeek V4 Flash Free.
- Review: GLM 5.2.

Перед Pull Request: tests проходят, lint проходит, build проходит, код прошёл review.
