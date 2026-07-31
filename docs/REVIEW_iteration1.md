# Code Review: azs_search_2gis_v2

> Ревью проведено по скиллу `review` (чеклисты Correctness / Maintainability / Security / Performance / Architecture alignment).
> Документ предназначен для передачи модели-имплементатору. Каждое замечание содержит `file:line`, проблему и ожидаемое действие.
> Дата: 2026-07-31.

---

## 0. Гейты (tests/lint/build)

Скилл `review` запускается **после** зелёных гейтов. Текущее состояние:

| Гейт | Команда | Статус | Примечание |
|---|---|---|---|
| Backend tests | `go test ./...` | ✅ green | все пакеты ok |
| go vet | `go vet ./...` | ✅ green | |
| gofmt | `gofmt -l .` | ❌ **RED** | 15 файлов не отформатированы (см. §0.1) |
| Frontend lint | `npm run lint` | ❌ **RED** | ESLint-конфиг отсутствует (см. §0.2) |
| Frontend build | `npm run build` (`tsc && vite build`) | ✅ green | TS strict-проверка проходит |

### 0.1 gofmt — RED

`gofmt -l .` в `backend/` выдаёт 15 файлов, требующих форматирования:

```
internal/benzin/client.go
internal/benzin/client_test.go
internal/config/config.go
internal/config/config_test.go
internal/geocode/twogis.go
internal/geocode/twogis_test.go
internal/headless/client.go
internal/httpapi/handler_test.go
internal/httpapi/provider_twogis.go
internal/httpapi/provider_twogis_test.go
internal/model/types.go
internal/osm/overpass.go
internal/osm/overpass_test.go
internal/stations/search.go
internal/stations/search_test.go
```

**Действие:** выполнить `gofmt -w .` в `backend/` и закоммитить. Makefile `lint` гоняет `gofmt -d .` (diff) — тоже RED.

### 0.2 ESLint — RED

`npm run lint` падает: `ESLint couldn't find a configuration file`. В `frontend/package.json` заявлены `eslint-plugin-react-hooks` и `eslint-plugin-react-refresh`, но файла `.eslintrc.*` / `eslint.config.*` нет.

**Действие:** добавить ESLint-конфиг (рекомендуется `eslint.config.js` flat-config для ESLint 8.57+ с плагинами `react-hooks`, `react-refresh`, `@typescript-eslint`). После — прогнать `npm run lint` до зелёного.

---

## 1. Critical

### C1. Секреты в репозитории и Docker-образе

- **`frontend/.env:1-2`** — реальные ключи `VITE_2GIS_MAPGL_KEY=7a769b43-…`, `VITE_YANDEX_MAP_KEY=39bf6daa-…`.
- **`.gitignore:4`** — игнорирует только корневой `.env`; `frontend/.env` **не** игнорируется → попадёт в VCS.
- **`frontend/.dockerignore`** — исключает только `node_modules`, `dist`, `*.local`; `.env` не исключён.
- **`frontend/Dockerfile:8`** — `COPY . .` копирует `.env` в build-context → ключи зашиваются в образ.
- **`.env:2` (корень)** — тот же ключ `7a769b43-…` используется как серверный `TWOGIS_API_KEY`. Если это сервер-рестриктед ключ 2GIS catalog API — он утёк в публичный JS-бандл (Vite `VITE_*` инлайнит значения в бандл).

**Ожидаемое:**
1. Добавить `frontend/.env` в `.gitignore` и в `frontend/.dockerignore`.
2. Удалить `frontend/.env` из репо (предоставить `frontend/.env.example`).
3. Разделить серверный ключ (`TWOGIS_API_KEY`, только бэкенд) и клиентский map-ключ (`VITE_2GIS_MAPGL_KEY`, публичный) — они должны быть разными ключами с разными scope.
4. Ротировать утёкшие ключи в партнёрском кабинете 2GIS/Yandex.
5. `VITE_*` по определению публичны — убедиться, что frontend-ключи имеют web-domain restriction, а не server-only.

### C2. Stored-XSS в попапе маркера карты

- **`src/map/MapRenderer2Gis.tsx:28`** — `marker.bindPopup(\`<b>${station.brand || station.name}</b><br>${station.address}\`)` вставляет `brand`/`name`/`address` как сырой HTML без экранирования.
- Источник `station` — и 2gis-, и osm-станции (тот же компонент рендерит оба источника). OSM-теги (`name`, `addr:full`) редактируются любым пользователем OSM → злоумышленник помещает `<img src=x onerror="…">` в имя АЗС → JS выполняется в браузере посетника.

**Ожидаемое:** экранировать `&<>"'` перед подстановкой, либо строить popup через DOM API (`document.createElement` + `textContent`), не через строку-шаблон. Завести утилиту `escapeHtml(s: string)` в `src/map/` и применить ко всем трём полям.

---

## 2. Major

### M1. Benzin-клиент не проверяет HTTP-статусы ответов

- **`internal/benzin/client.go:108-122`** (`searchStationIDs`) и **`:146-160`** (`getStationsDetails`): после `c.httpClient.Do(req)` сразу `io.ReadAll` + `json.Unmarshal`, без проверки `resp.StatusCode`.
- Любой 401/403/429/5xx или HTML-страница ошибки → невнятный `parse search response` / пустой список, реальная причина скрыта; ретрая нет.

**Ожидаемое:** после `Do` проверять `if resp.StatusCode < 200 || resp.StatusCode >= 300 { return nil, fmt.Errorf("benzin %s: status %d, body=%s", url, resp.StatusCode, snippet) }`. Реализовать ретрай с backoff (3 попытки) по плану §4.6: на 401 — перевыпуск токена (см. M2), на 429/5xx — возврат stale-кэша с пометкой.

### M2. Passepartout/JWE-авторизация не реализована (Фаза 0 не закрыта)

- **`internal/benzin/client.go`** целиком — запросы к `benzin.api.2gis.ru/api/v1/stations/search` идут **без** заголовка `Authorization`. План §4.6 требует JWE-токен от `passepartout.2gis.com`.
- **`internal/config/config.go:54`** — `PassepartoutURL` загружается, но нигде не используется.
- **`docs/BENZIN_API.md:5,23,98`** — «Статус: Ожидает перехвата (Фаза 0)» — блокирующая Фаза 0 не завершена.

**Ожидаемое:** подтвердить у заказчика статус `benzin.api.2gis.ru` — открыт ли он без auth, или нужен токен. Если нужен — реализовать `getToken` (кэш токена с TTL < срока жизни) + `Authorization: Bearer <JWE>`. Если открыт — обновить `docs/BENZIN_API.md` и `IMPLEMENTATION_PLAN.md` §4.6 с пометкой «auth не требуется (проверено ДД.ММ.ГГГГ)». Решение зафиксировать в `docs/ADR-001-benzin-auth.md`.

### M3. Деградация на OSM маскирует ошибку и кеширует fallback как «2gis»

- **`internal/httpapi/provider_twogis.go:88-92`**:
  ```go
  stations, err := p.benzin.GetStations(ctx, loc.Lat, loc.Lon, radius, fuel)
  if err != nil {
      return p.osm.GetStationsWithRetry(ctx, loc.Lat, loc.Lon, radius, fuel, 2)
  }
  return stations, nil
  ```
  - Ошибка `benzin` **не логируется** (проглатывается).
  - Результат OSM кешируется `cache.GetOrLoad` под тем же ключом `stations:2gis:…` на весь TTL (~10 мин) → после отказа benzin следующие ~10 мин реальные данные не запрашиваются, хотя сервис мог восстановиться. План §4.6 требует пометки `stale`.

**Ожидаемое:**
1. Логировать ошибку benzin: `p.logger.Error("benzin failed, fallback to osm", ...)` (пробросить logger в provider).
2. Не кешировать fallback под ключом `2gis`: либо кешировать под отдельным ключом `stations:2gis-fallback:…` с коротким TTL (1–2 мин), либо помечать результат `Source="osm-overpass"` и не класть в кеш вообще.
3. Вернуть признак stale в ответ (поле `stale: true` в JSON) для UI-пометки «данные временно недоступны».

### M4. Параметр `provider` в `/api/geocode` игнорируется

- **`internal/httpapi/handler.go:44-53,55`** — `providerName` читается из query, но `h.provider.Geocode(ctx, q)` зовётся без него; всегда используется `TwoGisGeocoder`. `?provider=yandex` бесполезен.
- **`internal/geocode/yandex.go`** — геокодер собран, но мёртв (не подключён в `main.go`, не вызывается).
- README §API описывает `provider=2gis|yandex` — расходится с реализацией.

**Ожидаемое:** либо реализовать диспетчер геокодеров по `provider` (мапа `map[string]Geocoder` в `TwoGisProvider`), либо убрать `provider` из контракта `/api/geocode` и из README. Рекомендуется первое — это часть критерия приёмки №3.

### M5. Переключатель карты 2ГИС↔Яндекс и выбор станции не реализованы

- **`src/App.tsx:7`** — импортирован только `MapRenderer2Gis`; `src/map/MapRendererYandex.tsx` — мёртвый код.
- **`src/App.tsx:100,111-114`** — `ProviderToggle` меняет стейт `provider`, но `MapRenderer2Gis` монтируется всегда; рендерер не выбирается по `provider`.
- **`src/App.tsx:120`** — `StationList onStationSelect={() => {}}` (no-op); `onStationClick` в `MapRenderer2Gis` не прокинут (план §5.2 `flyTo(station)` не реализован).

**Ожидаемое:**
1. В `App.tsx` рендерить `MapRenderer2Gis` или `MapRendererYandex` по `provider` (или через общий интерфейс `MapRenderer` из плана §5.2).
2. Пробросить `onStationSelect` из `StationList` → `App` → `onStationClick` в `MapRenderer2Gis`; реализовать `flyTo(station)` (прокси к `mapInstance.flyTo`/`panTo`).
3. Это блокирует критерий приёмки §8 №3 и №5.

### M6. MapRendererYandex: маркеры не добавляются на карту + пересоздание карты

- **`src/map/MapRendererYandex.tsx:72`** — `new YMapMarker([station.lon, station.lat], markerElement)` создаётся, но **не добавляется** на карту (нет вызова добавления). Маркеры не отрисуются.
- **`src/map/MapRendererYandex.tsx:20-89`** — `useEffect` с deps `[center, stations]` пересоздаёт всю карту (`destroy` + `new YMap`) на каждое изменение `stations` → фликер, утечка, потеря состояния. Инициализация и рендер маркеров должны быть в разных эффектах.

**Ожидаемое:** добавить маркер на карту (API `ymaps3`: `map.addChild(new YMapMarker(...))`). Разделить эффекты: `[]` — загрузка SDK + инициализация карты; `[center]` — `setCenter`; `[stations]` — diff маркеров. После M5 это станет актуальным.

### M7. Некорректные `UpdatedAt` и `CanJerrycan` в данных АЗС

- **`internal/benzin/client.go:251`** — `CanJerrycan: true` захардкожено всегда `true` (нет источника данных).
- **`internal/benzin/client.go:253`** — `UpdatedAt: time.Now()` = время **запроса**, а не свежесть данных API. UI показывает «Обновлено: HH:MM» момента запроса — вводит в заблуждение (план §9 «свежесть ~30 мин»).
- **`internal/osm/overpass.go:188`** — та же проблема `time.Now()`.

**Ожидаемое:** брать `updated_at` из ответа benzin API (если поле есть); иначе — `time.Time{}` и во фронте не показывать «Обновлено», либо показывать «свежесть неизвестна». `CanJerrycan` — из данных API или `false` (не `true` по умолчанию).

### M8. Frontend API-клиент: маскировка ошибок и гонки запросов

- **`src/api/index.ts:9,19,44`** — на ошибочном пути `await response.json()` бросит, если тело не JSON (например, HTML 502 от nginx) → маскирует исходную HTTP-ошибку.
- **`src/api/index.ts`** — нет `AbortController`: быстрая смена фильтра/радиуса/провайдера → гонки; поздний ответ перетирает свежий state (видно в `App.tsx` `handleFuelChange`/`handleRadiusChange`/`handleProviderChange`).

**Ожидаемое:**
1. В error-ветке проверять `Content-Type: application/json` перед `.json()`, иначе кидать `new Error(\`HTTP \${response.status}\`)`.
2. Передавать `signal` в `fetch`; в `App.tsx` хранить `AbortController` в ref и обрывать предыдущий запрос при новом; игнорировать `AbortError`.

---

## 3. Minor

### m1. Мёртвый код / неиспользуемые поля
- **`internal/benzin/client.go:21`** — `mu sync.RWMutex` объявлен, не используется.
- **`internal/provider/provider.go`** — интерфейс-дубликат `httpapi.Provider`, не импортируется.
- **`internal/headless/client.go` + `backend/Dockerfile.headless`** — headless-фолбэк (план §4.6/§9) не подключён в `main.go`; пакет не используется, образ не нужен. Либо реализовать, либо удалить.

### m2. Дублирование маппинга топлива (DRY)
- **`internal/benzin/client.go:258-275`** (`mapFuelType`), **`:277-300`** (`hasFuelType`), **`internal/stations/search.go:58-79`** (`matchFuelType`) — три копии маппинга фильтр↔тип.
- Дополнительно **`internal/httpapi/provider_twogis.go:99`** — `stations.FilterByFuel` дублирует фильтр, уже выполненный в **`internal/benzin/client.go:186`** (`mapStation` отбрасывает несовпадения). Двойная фильтрация на каждом запросе.

**Ожидаемое:** вынести таблицу маппинга в `internal/fuel` пакет; убрать дублирующую фильтрацию в `provider_twogis.go` (benzin уже отфильтровал; OSM-фильтр оставить).

### m3. Производительность: bubble sort
- **`internal/stations/search.go:29-35`** — `SortByDistance` O(n²). Для MVP достаточно; при росте числа станций заменить на `sort.Slice`.

### m4. Nominatim Usage Policy
- **`internal/osm/nominatim.go:53`** — User-Agent `azs_search_2gis_v2/1.0` без контактных данных; rate-limit на бэке отсутствует. Nominatim требует ≤1 req/s и контакт в UA.
- **Ожидаемое:** добавить контакт (email/URL) в UA; добавить бакенд-семафор (≤1 req/s) на suggest; фронт-дебаунс 300 мс уже есть — этого мало на параллельных клиентах.

### m5. Слабые TypeScript-типы
- **`src/types/index.ts:18,24`** — `avail: string`, `level: string` → union-типы `'yes'|'no'|'unknown'` и `'none'|'small'|'medium'|'large'|'very_large'|'unknown'` (`StationCard.tsx` всё равно свитчит по литералам).
- **`src/map/dg.ts:3`** (`DG?: any`), **`src/map/MapRenderer2Gis.tsx:13-14`** (`useRef<any>`), **`src/map/MapRendererYandex.tsx:12`** (`ymaps3: any`) — потеря типобезопасности. Завести минимальные типы для SDK.

### m6. UX-несогласованность геокодирования
- **`src/components/AddressInput.tsx:28`** — `geocode(q,'2gis')` требует `TWOGIS_API_KEY`; без ключа Enter падает с ошибкой.
- **`src/components/AddressInput.tsx:78`** — клик по подсказке (Nominatim) работает без ключа.
- **Ожидаемое:** либо сделать fallback geocode на Nominatim при отсутствии 2GIS-ключа, либо явно сообщать пользователю «геокодирование требует настройки ключа».

### m7. nginx: кеширование статики и security-заголовки
- **`frontend/nginx.conf:9`** — `Cache-Control: no-store` на `location /` применяется и к хешированным `/assets/*` → статика не кешируется (перф).
- Нет CSP, `X-Content-Type-Options: nosniff`, `X-Frame-Options`, `Referrer-Policy`.
- **Ожидаемое:** `no-store` — только для `index.html`; для `/assets/*` — `immutable, max-age=31536000`. Добавить security-заголовки.

### m8. Валидация входных параметров
- **`internal/httpapi/handler.go:107-122`** — нет валидации диапазона `lat/lon` (`[-90,90]`/`[-180,180]`) и верхнего предела `radius` → мусор уходит в upstream. `fuel` не валидируется (unknown-фильтр в `benzin.hasFuelType` молча возвращает `true` = нет фильтрации).

### m9. Прочее
- **`src/components/AddressInput.tsx:98`**, **`src/components/StationCard.tsx:51`** — `key={i}`/`key={idx}` (индекс как key) — использовать стабильный id (`station.id`, `fuel.type`).
- **`src/map/MapRenderer2Gis.tsx:36-63`** — init `useEffect` с deps `[]` захватывает стейл `onStationClick` (латентный баг; сейчас нейтрализован тем, что `onStationClick` не передаётся). После M5 — актуально.
- **`cmd/server/main.go:30-32`** — `slog` logger создаётся, но как default не устанавливается; chi `middleware.Logger` использует stdlib `log` — мелкая несогласованность логирования.

---

## 4. Architecture alignment (расхождения с `IMPLEMENTATION_PLAN.md`)

| Пункт плана | Реализация | Статус |
|---|---|---|
| §4.6 Passepartout JWE-токен | не реализован | **Major (M2)** |
| §4.6 ретрай: 401→токен, 429/5xx→stale | не реализован | **Major (M1, M3)** |
| §4.7 деградация с `Avail="unknown"`, `Source="2gis-catalog"` | деградация на OSM есть, но логирование/кеширование некорректны | **Major (M3)** |
| §5.2 переключаемый рендерер карты (2GIS↔Yandex) | не реализован | **Major (M5, M6)** |
| §5.2 `flyTo(station)` при выборе | не реализован | **Major (M5)** |
| §5.3 цвет маркера = наличие, размер = очередь | только маркеры 2GIS, без дифференциации | Minor |
| §6 headless-фолбэк | Dockerfile есть, код не подключён | Minor (m1) |
| §8 критерий №3 (переключатель карт) | не работает | **Major (M5)** |
| §8 критерий №5 (цена/очередь/лимиты/«обновлено») | частично; «обновлено» некорректно (M7) | **Major (M7)** |
| Фаза 0 (de-risk источника) | docs/BENZIN_API.md помечен «ожидает» | **Major (M2)** |

---

## 5. Positive

- Чистая слоистая Go-структура (`cmd/internal` split, доменные пакеты `model/benzin/cache/geocode/osm/httpapi`) — соответствует плану §3.
- Корректный graceful shutdown (`cmd/server/main.go:79-92`) с таймаутом и обработкой SIGINT/SIGTERM.
- `singleflight` + TTL-кэш реализованы по плану §4.8 (`internal/cache/cache.go`); кэш-ключ станций округляет координаты до `%.3f` (план §4.5).
- Деградация на OSM Overpass работает — приложение не падает при отказе benzin (`internal/httpapi/provider_twogis.go:89-91`).
- BBox с cos-широта-коррекцией (`internal/benzin/client.go:173-183`) и гаверсинус (`internal/stations/search.go:9-22`) — геометрия корректна.
- Покрытие тестами: 10 `*_test.go` (benzin, cache, config, geocode×2, httpapi×2, osm×2, stations) — `go test ./...` green.
- Multi-stage backend Dockerfile → `gcr.io/distroless/static-debian12` (малая поверхность атаки).
- ToS-дисклеймер в README и в UI (`src/App.tsx:124-128`).
- TypeScript `strict` включён; `tsc` проходит; `vite build` green.

---

## 6. Рекомендуемый порядок исправлений (для имплементатора)

1. **Гейты (§0):** `gofmt -w .` в backend; создать ESLint-конфиг в frontend → оба гейта зелёные.
2. **C1 (секреты):** `.gitignore`/`.dockerignore` + удалить `frontend/.env` + `.env.example` + ротация ключей.
3. **C2 (XSS):** `escapeHtml` + DOM-popup в `MapRenderer2Gis.tsx`.
4. **M1+M2 (benzin):** проверка статусов + решение по Passepartout-авторизации (ADR).
5. **M3 (деградация):** логирование + отдельный кеш-ключ для fallback.
6. **M4 (geocode provider):** диспетчер geocoder’ов по `provider`.
7. **M5+M6 (карты):** переключение рендерера по `provider` + `flyTo` + починка Yandex-маркеров.
8. **M7 (данные):** корректный `updated_at` + `CanJerrycan`.
9. **M8 (fetch):** `AbortController` + безопасный error-parsing.
10. Minor — по мере работы в затронутых файлах.

---

## 7. Вердикт

**Request changes.**

Блокирующее (Critical): C1 — секреты в репо/образе; C2 — stored-XSS.
Сильные Major: M1/M2 — отсутствие проверки статусов benzin + нереализованная Passepartout-авторизация (Фаза 0 не закрыта); M3 — маскирование ошибки деградации общим кешем; M5/M6 — нерабочий переключатель карт и выбор станции (критерии приёмки §8 №3/№5); M8 — гонки/маскировка ошибок во frontend fetch.
Гейты: gofmt и ESLint — RED; tests/vet/build — green.
Мёртвый код (`yandex`, `headless`, `provider`) — убрать или дописать.

Ревью можно повторить после исправления Critical + Major и перевода обоих RED-гейтов в зелёное состояние.
