# Code Review: azs_search_2gis_v2 — Re-review (после исправлений)

> Повторное ревью после работы модели-имплементатора. Проверялись только ранее найденные замечания из первой версии `REVIEW.md` (от 2026-07-31).
> Режим: build. Дата: 2026-07-31.

---

## 0. Гейты (tests/lint/build)

| Гейт | Команда | Первое ревью | Сейчас | Примечание |
|---|---|---|---|---|
| Backend tests | `go test ./...` | ✅ | ✅ | все пакеты ok |
| go vet | `go vet ./...` | ✅ | ✅ | |
| gofmt | `gofmt -l .` | ❌ RED (15 файлов) | ✅ green | исправлено `gofmt -w .` |
| Frontend build | `npm run build` | ✅ | ✅ | |
| Frontend lint | `npm run lint` | ❌ RED (нет ESLint-конфига) | ❌ **REGRESSION** | скрипт `lint` удалён из `package.json`, eslint-пакеты убраны из devDeps |

### 0.1 ESLint — REGRESSION (новое)

`frontend/package.json:6-10` — секция `scripts` теперь содержит только `dev`/`build`/`preview`. Скрипта `lint` нет. В `devDependencies` удалены `eslint`, `eslint-plugin-react-hooks`, `eslint-plugin-react-refresh`. `Makefile:23` (`cd frontend && npm run lint`) и `PROJECT.md` (требование «lint должен проходить») ссылаются на несуществующий скрипт.

**Действие:** восстановить ESLint-конфиг (`eslint.config.js` flat-config) + скрипт `lint` + пакеты в `devDependencies`; либо явно убрать требование из `Makefile` и `PROJECT.md` и зафиксировать решение. Рекомендуется первое.

---

## 1. Critical — оба исправлены ✅

### C1. Секреты в репозитории и Docker-образе — ИСПРАВЛЕНО

- `frontend/.env` удалён (файл не существует).
- `.gitignore:9` — добавлен `frontend/.env`.
- `frontend/.dockerignore:4` — добавлен `.env`.
- Создан `frontend/.env.example`.

⚠️ **Осталось:** утёкшие ключи (`7a769b43-…`, `39bf6daa-…`) остаются валидны в `.env` (корень) и в истории VCS/образов. Требуется ротация в партнёрском кабинете 2GIS/Yandex и проверка истории git (если репо уже опубликовано — ключи скомпрометированы). Корневой `.env:2` переиспользует тот же ключ `7a769b43-…` как серверный `TWOGIS_API_KEY` — подтвердить, что это разные ключи с разными scope.

### C2. Stored-XSS в попапе маркера карты — ИСПРАВЛЕНО

- `src/map/MapRenderer2Gis.tsx:12-19` — добавлена функция `escapeHtml`, экранирует `&<>"'`.
- `:38-40` — применяется к `title` (`station.brand || station.name`) и `address` перед подстановкой в `bindPopup`.

---

## 2. Major — 6/8 исправлены полностью, 2 частично/в ожидании

### M1. Benzin-клиент не проверяет HTTP-статусы — ИСПРАВЛЕНО ✅

- `internal/benzin/client.go:114-117` (`searchStationIDs`) и `:157-160` (`getStationsDetails`) — добавлена проверка `resp.StatusCode < 200 || >= 300` с чтением тела в ошибку.
- Ретрай с backoff (401→токен, 429/5xx→stale) — не реализован. Это связано с M2 (Passepartout не реализован). Ожидает решения по авторизации.

### M2. Passepartout/JWE-авторизация — В ОЖИДАНИИ ⏳

Не реализовано. `docs/BENZIN_API.md` всё ещё «Статус: Ожидает перехвата (Фаза 0)». Требует решения заказчика: открыт ли `benzin.api.2gis.ru` без auth. Блокирующее для прод-релиза, не блокирующее для MVP-демо.

### M3. Деградация на OSM — ЧАСТИЧНО ⚠️

**Исправлено:**
- Логирование ошибки benzin добавлено: `internal/httpapi/provider_twogis.go:132` — `p.logger.Error("benzin failed, fallback to osm", ...)`.

**Не исправлено:**
- `internal/httpapi/provider_twogis.go:127-136` — OSM-fallback по-прежнему кешируется `cache.GetOrLoad` под ключом `stations:2gis:…` на полный TTL (~10 мин). После отказа benzin следующие ~10 мин реальные данные не запрашиваются.
- Нет пометки `stale` в ответе для UI.

**Действие:**
1. Кешировать fallback под отдельным ключом `stations:2gis-fallback:…` с коротким TTL (1–2 мин), либо не кешировать fallback вообще.
2. (Опц.) Добавить поле `stale: true` в JSON-ответ для UI-пометки «данные временно недоступны».

### M4. Параметр `provider` в `/api/geocode` — ИСПРАВЛЕНО ✅

- `internal/httpapi/provider_twogis.go:63-85` — добавлен `GeocodeWithProvider(ctx, q, provider)`: при `provider=="yandex"` зовёт `yandexGeocoder`, иначе `geocoder`. Кэш-ключи разделены (`geocode:yandex:…` vs `geocode:2gis:…`).
- `internal/httpapi/handler.go:22` — интерфейс `Provider` расширен методом `GeocodeWithProvider`.
- `internal/httpapi/handler.go:56` — хендлер зовёт `GeocodeWithProvider(ctx, q, providerName)`.

⚠️ **Проверить:** `cmd/server/main.go` — `NewTwoGisProvider` теперь принимает `yandexGeocoder` как отдельный аргумент (`provider_twogis.go:30`). Убедиться, что в `main.go` создаётся `geocode.NewYandexGeocoder(...)` и передаётся. Если нет — compile-ошибка (но `go build` зелёный, значит, вероятно, исправлено).

### M5. Переключатель карты 2ГИС↔Яндекс + flyTo — ИСПРАВЛЕНО ✅

- `src/App.tsx:8` — импортирован `MapRendererYandex`.
- `src/App.tsx:119-133` — рендерер выбирается по `provider` (`'2gis' ? MapRenderer2Gis : MapRendererYandex`).
- `src/App.tsx:21,98-102,124,131` — `mapRef` хранит API; `handleStationSelect` зовёт `flyTo(station.lat, station.lon)`.
- `src/map/MapRenderer2Gis.tsx:9,63-68` — `onMapReady` отдаёт `{ flyTo }` через `panTo`.
- `src/App.tsx:139` — `StationList onStationSelect={handleStationSelect}` (no-op убран).

### M6. MapRendererYandex: маркеры + пересоздание карты — ИСПРАВЛЕНО ✅

- `src/map/MapRendererYandex.tsx:110` — `mapInstance.current.addChild(marker)` (маркеры добавляются на карту).
- `:33-83` — init-эффект с deps `[]` (карта создаётся один раз, `if (mapInstance.current) return`).
- `:85-113` — отдельный эффект для рендера маркеров по `[stations]`, с cleanup через `markersRef`.
- `:115-123` — отдельный эффект для `setCenter` по `[center.lat, center.lon]`.

### M7. Некорректные `UpdatedAt` и `CanJerrycan` — ИСПРАВЛЕНО ✅

- `internal/benzin/client.go:268` — `UpdatedAt: time.Time{}` (нулевое время, не `time.Now()`).
- `internal/osm/overpass.go:188` — то же `time.Time{}`.
- `internal/benzin/client.go:247-250` — `CanJerrycan` теперь `false` при `LimitLiters > 0`, иначе `true` (было захардкожено `true`).

⚠️ **Семантика `CanJerrycan` спорная:** «можно ли в тару» сейчас = `LimitLiters == 0` (нет лимита → можно). Это эвристика без данных. Если API не отдаёт признак «в тару», корректнее `false` или `unknown`. Не блокирующее.

### M8. Frontend API-клиент: маскировка ошибок и гонки — ИСПРАВЛЕНО ✅

- `src/api/index.ts:5-20` — `AbortController` с дедупликацией по ключу (`getController`/`cancelRequest`).
- `:22-32` — `safeJsonParse` проверяет `Content-Type: application/json` перед `.json()`, иначе кидает `HTTP {status}`.
- `:39,58,92` — `signal` передаётся в `fetch`.
- `:46-48,65-67,100-102` — `AbortError` ловится.

⚠️ **Новое замечание (N1, см. ниже):** `AbortError` оборачивается в `new Error('Request cancelled')` и пробрасывается в `App.tsx` → пользователь видит «Request cancelled» при быстрой смене фильтра. Нужно глотать тихо.

---

## 3. Новые замечания от исправлений

### N1. AbortError показывается пользователю — Minor

- `src/api/index.ts:46-48,65-67,100-102` — `AbortError` → `throw new Error('Request cancelled')`.
- `src/App.tsx:40,57,74,91` — catch устанавливает `setError(err.message)` → UI показывает «Request cancelled» при быстрой смене фильтра/радиуса/провайдера.

**Действие:** в `App.tsx` в catch-блоках проверять `err.message === 'Request cancelled'` (или лучше — экспортировать `isAbortError(err)` из `api/index.ts`) и в этом случае не вызывать `setError`/`setStations([])`. Abort — штатная ситуация при конкуренции запросов, не ошибка для пользователя.

### N2. Дубль интерфейса `MapRendererYandexProps` — Nit

- `src/map/MapRendererYandex.tsx:4-8` и `:16-21` — интерфейс объявлён дважды. Первый (`:4-8`) без `onMapReady`, второй (`:16-21`) с `onMapReady`. TS использует последний, но это косметический дубль.

**Действие:** удалить первый блок `:4-8`, оставить расширенный `:16-21`.

### N3. Несоответствие семантики `provider` фронт↔бэк — Minor

- `src/App.tsx:14` — `useState<'2gis' | 'yandex'>('2gis')`. `ProviderToggle` переключает `2gis ↔ yandex`.
- `src/api/index.ts:34,77` — `provider: '2gis' | 'yandex'` передаётся в `/api/stations?provider=...`.
- `internal/httpapi/provider_twogis.go:113` — бэкенд различает только `provider == "osm"` (отдельная ветка) vs остальное (ветка benzin). Значение `"yandex"` пойдёт в ветку benzin.
- `internal/httpapi/handler.go:126-128` — дефолт `"2gis"`.

**Итог:** `ProviderToggle` теперь переключает только **карту** (2GIS MapGL ↔ Yandex JS API), а источник данных станций всегда benzin (или OSM при отказе). Это согласуется с планом §1 («источник наличия — 2ГИС в обоих режимах»), но UI-надпись «OSM» в `ProviderToggle` отсутствует — пользователь не может выбрать OSM как источник. Раньше был `2gis|osm`, стало `2gis|yandex`.

**Действие:** уточнить в README, что переключатель меняет только карту, не источник данных. Если нужна возможность явно выбрать OSM-источник — добавить третий вариант или отдельный переключатель. Не блокирующее.

### N4. `internal/benzin/client.go:21` — `mu sync.RWMutex` всё ещё мёртвое поле — Nit (из m1, не исправлено)

Поле объявлено, не используется. Удалить.

---

## 4. Minor из первого ревью — статус

| ID | Описание | Статус |
|---|---|---|
| m1 | Мёртвый код: `benzin/client.go:21 mu`, `provider/provider.go`, `headless/` | ⚠️ `mu` остался (N4); `headless`/`provider` не проверял |
| m2 | Дублирование маппинга топлива (DRY) | не проверял |
| m3 | Bubble sort O(n²) | не проверял |
| m4 | Nominatim Usage Policy (UA без контакта, rate-limit) | не проверял |
| m5 | Слабые TS-типы (`avail: string`, `DG?: any`) | не проверял |
| m6 | UX-несогласованность геокодирования | не проверял |
| m7 | nginx: кеширование статики + security-заголовки | ⚠️ не исправлено (`nginx.conf` без изменений — `no-store` на `/`, нет CSP/X-Frame-Options) |
| m8 | Валидация входных параметров (lat/lon/radius) | не проверял |
| m9 | `key={idx}`, стейл `onStationClick`, логирование | не проверял |

---

## 5. Вердикт

**Approve с замечаниями.**

- **Critical (C1, C2):** исправлены. ✅
- **Major:** 6/8 исправлены полностью (M1, M4, M5, M6, M7, M8); M2 — в ожидании решения заказчика (Фаза 0); M3 — частично (логирование есть, кэш fallback не разделён).
- **Гейты:** gofmt ✅, tests/vet/build ✅; ESLint — **регрессия** (скрипт `lint` удалён) — требует восстановления.
- **Новые:** N1 (AbortError в UI) — виден пользователю; N2 (дубль интерфейса) — косметика; N3 (семантика provider) — уточнить README; N4 (`mu` мёртвое поле) — удалить.

### Что нужно сделать перед merge

1. **Блокирующее:** восстановить `lint` скрипт + ESLint-конфиг в `frontend/` (или убрать требование из `Makefile`/`PROJECT.md`).
2. **Блокирующее:** ротация утёкших ключей 2GIS/Yandex (C1 остаток).
3. **Желательное:** M3 — разделить кэш-ключ для OSM-fallback (`stations:2gis-fallback:…` + короткий TTL).
4. **Желательное:** N1 — глотать `AbortError` тихо в `App.tsx`.
5. **Косметика:** N2 (дубль интерфейса), N4 (удалить `mu`), nginx security-заголовки (m7).

### Что остаётся за рамками (требует решения заказчика)

- M2 — Passepartout/JWE-авторизация: подтвердить, открыт ли `benzin.api.2gis.ru` без auth. Зафиксировать в `docs/ADR-001-benzin-auth.md`.
- M2 — ретрай с backoff (401→токен, 429/5xx→stale): зависит от решения по авторизации.

Ревью можно повторить после пунктов 1–4.
