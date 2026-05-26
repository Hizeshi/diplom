# Исправления по bags-1.md

## Что было исправлено

---

### 1. GET /api/products/{id} — больше не падает 500

**Причина:** поля `product_type`, `description`, `configurator_type`, `price`, `stock` в базе допускают NULL. Go-драйвер падал при попытке записать NULL в обычную строку.

**Результат:** теперь возвращает `200` с данными товара.

---

### 2. POST /api/products/search — больше не падает 500

**Причина:** та же проблема с NULL-полями + неверный тип данных для параметров `brandId`, `colorId`, `seriesId` (функция в базе ожидает `integer`, а передавался `bigint`).

**Результат:** теперь возвращает `200 { "items": [...], "count": N }`.

---

### 3. GET /api/user/favorites — больше не падает 500

**Причина:** поле `price` у товара в базе допускает NULL, а скан шёл в `float64`.

**Результат:** теперь возвращает `200 { "data": [...] }`.

---

### 4. POST /api/user/favorites — больше не падает 500

**Причина:** попытка добавить в избранное товар с несуществующим ID нарушала FK-ограничение базы.

**Результат:** теперь работает корректно. Если товар не существует — просто ничего не происходит.

---

### 5. POST /api/user/history — больше не падает 500

**Причина:** та же — FK-violation при несуществующем `product_id`.

**Результат:** теперь возвращает `200 { "success": true }`. Если товар не существует — вызов игнорируется без ошибки.

---

### 6. GET /api/user/session — теперь возвращает профиль пользователя

**Было:**
```json
{
  "session_id": "user:67e4641c-...",
  "is_human_mode": false
}
```

**Стало:**
```json
{
  "id": "67e4641c-2896-4956-b69f-090083a50175",
  "email": "user@example.com",
  "full_name": "Иван Иванов",
  "phone": "+77001234567",
  "avatar_url": "https://supabase.iq-home.kz/storage/v1/object/public/avatars/...",
  "role": "customer"
}
```

---

### 7. Все /api/user/* — поле `output` переименовано в `data`

**Было:** `{ "output": [...] }`

**Стало:** `{ "data": [...] }`

Затронутые эндпоинты:
- `GET /api/user/cart`
- `GET /api/user/favorites`
- `GET /api/user/history`
- `GET /api/user/recommendations`
- `GET /api/user/orders`

---

### 8. GET /api/chat/history — поле `messages` переименовано в `data`

**Было:** `{ "messages": [...] }`

**Стало:** `{ "data": [...] }`

---

### 9. Пустые списки — теперь `[]` вместо `null`

Когда у пользователя нет заказов, истории, избранного и т.д., раньше возвращалось `{ "data": null }`. Теперь всегда возвращается `{ "data": [] }`.

---

### 10. POST /v1/quotes — больше не падает 500

**Причина:** не существовало бакета `quotes` в Supabase Storage.

**Результат:** бакет создан, эндпоинт возвращает `200 { "url": "https://..." }`.

---

### 11. POST /v1/products/vectorize — больше не зависает

**Было:** запрос висел минутами и завершался timeout.

**Стало:** сервер сразу отвечает `202 { "status": "started" }`, а векторизация выполняется в фоне. Повторно вызывать не нужно — достаточно одного запроса.

---

### 12. POST /v1/telegram/webhook — исправлена архитектура

**Было:** вебхук находился за `X-Internal-Token`, из-за чего Telegram физически не мог прислать сообщения (он не знает внутренний токен).

**Стало:** вебхук вынесен в отдельный публичный маршрут, защищён только `X-Telegram-Bot-Api-Secret-Token` (стандарт Telegram). Теперь Telegram реально может слать сообщения.

---

## Актуальные ответы эндпоинтов

### POST /api/user/checkout

**Тело запроса:**
```json
{
  "name": "Иван Иванов",
  "phone": "+77001234567",
  "address": "г. Алматы, ул. Абая 1",
  "paymentMethod": "card"
}
```

**Ответ `200`:**
```json
{
  "success": true,
  "orderId": 101,
  "paymentUrl": "https://...",
  "message": "Заказ принят. Менеджер свяжется с вами."
}
```

> `paymentUrl` — только при `paymentMethod: "card"`. `message` — только при `"cash"`.

---

### POST /v1/chat/media

Поле называется **`message_type`** (не `type`):

| Поле | Тип | Обязательное |
|---|---|---|
| `session_id` | string | да |
| `message_type` | string (`"voice"` / `"photo"` / `"document"`) | да |
| `file` | file | да |
| `user_id` | string | нет |
| `platform` | string | нет |

---

### GET /api/filters

Поле `types` — **массив строк**, не объектов:

```json
{
  "types": ["Люстра", "Розетки электрические", "Рамки"],
  "brands": [{ "id": 2, "name": "Maytoni" }],
  "colors": [{ "id": 3, "name": "Золото" }],
  "series": [{ "id": 4, "name": "Cascade" }],
  "min_price": 5000,
  "max_price": 250000
}
```

---

## Admin API

**BasicAuth** для `/api/admin/*`:

```
Username: Aurora
Password: Gaming
```

---

## Что требуется от фронтенда

1. **Убрать `NEXT_PUBLIC_CHAT_INTERNAL_TOKEN`** — внутренний токен не должен быть публичной переменной окружения. Фронтенд не должен напрямую вызывать `/v1/*`. Все `/v1/*` вызовы — только с сервера (Next.js API routes или server actions).

2. **Перевыпустить токены** — в ходе тестирования в чат были отправлены:
   - Supabase access/refresh token пользователя
   - `CHAT_INTERNAL_TOKEN`
   - `NEXT_PUBLIC_CHAT_INTERNAL_TOKEN`

   Эти токены нужно считать скомпрометированными и перевыпустить.
