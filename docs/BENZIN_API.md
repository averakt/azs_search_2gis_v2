# Benzin API 2GIS

## Статус

**Фаза 0 — завершено**

API не требует авторизации для read-only запросов. Проверено 2026-07-31.

См. docs/ADR-001-benzin-auth.md

## Источники данных

### Benzin API

**URL:** `https://benzin.api.2gis.ru/`

**Статус:** Работает без авторизации (проверено 2026-07-31)

**Авторизация:** Не требуется для read-only endpoints

**Ожидаемые параметры:**
- `lat`, `lon` — координаты
- `radius` — радиус поиска (метры)
- `fuel` — фильтр по топливу

**Ожидаемые фильтры:**
- `benzin_ai_92`
- `benzin_ai_95`
- `benzin_ai_98`
- `benzin_ai_100`
- `benzin_dt` (ДТ)
- `benzin_gas` (газ)
- `benzin_from_25_to_50` (очередь)

## План перехвата

1. Открыть `benzin.2gis.ru` в браузере
2. Открыть DevTools → Network
3. Найти запросы к `benzin.api.2gis.ru`
4. Зафиксировать:
   - Точный path URL
   - Query параметры
   - Заголовки (особенно Authorization)
   - Структуру ответа

5. Воспроизвести запрос через `curl` или Go
6. Документировать результат

## Скрипт для воспроизведения (TODO)

```bash
# После получения токена:
curl -H "Authorization: Bearer <TOKEN>" \
     "https://benzin.api.2gis.ru/stations?lat=55.751244&lon=37.618423&radius=3000"
```

## Структура ответа (ожидаемая)

```json
{
  "stations": [
    {
      "id": "string",
      "name": "string",
      "brand": "string",
      "address": "string",
      "lat": 0,
      "lon": 0,
      "fuels": [
        {"type": "АИ-95", "avail": "yes|no|unknown", "price": 0}
      ],
      "queue": {"level": "none|small|medium|large"},
      "limits": {"max_liters": 0, "can_jerrycan": false},
      "updated_at": 0
    }
  ]
}
```

## Риски

- Токен может быть origin-bound (привязан к домену)
- Токен может быть привязан к User-Agent/IP
- Методы API могут измениться

## Митигация

- Headless fallback через chromedp
- Кэширование ответов
- Изоляция клиента в `internal/benzin`

---

**Обновлено:** 2026-07-30
**Статус:** Ожидает перехвата (Фаза 0)
