# Исправления по bags-2.md

## Что было исправлено

---

### 1. POST /api/user/history — исправлено

**Причина:** поле в теле запроса называлось `productId` (camelCase), а фронтенд отправлял `product_id` (snake_case). Данные не читались, передавался ID=0.

**Актуальное тело запроса:**
```json
{
  "product_id": 382
}
```

**Ответ `200`:**
```json
{ "success": true }
```

Если товар не существует — тоже возвращает `200` (без ошибки, запись просто игнорируется).

---

### 2. POST /v1/chat/media — исправлено

**Причина:** Apache Tika (парсер документов) не запущена на сервере. При `message_type=document` бэкенд падал с 500.

**Что изменено:** если Tika недоступна, вместо 500 бот отвечает пользователю с просьбой прислать текст вручную. `voice` и `photo` работали и раньше — они используют OpenAI, а не Tika.

**Актуальные значения `message_type`:**

| Значение | Обработка |
|---|---|
| `voice` | Транскрипция через OpenAI Whisper |
| `photo` | Анализ через OpenAI Vision |
| `document` | Извлечение текста через Tika (если недоступна — graceful fallback) |

---

### 3. POST /v1/telegram/webhook — закрыт

**Причина:** `TELEGRAM_WEBHOOK_SECRET` был пустой строкой, проверка не выполнялась.

**Что изменено:**
- Задан секрет
- Проверка заголовка `X-Telegram-Bot-Api-Secret-Token` теперь обязательна
- Вебхук зарегистрирован в Telegram

**Поведение:**

| Запрос | Ответ |
|---|---|
| Без `X-Telegram-Bot-Api-Secret-Token` | `401 Unauthorized` |
| С неверным секретом | `401 Unauthorized` |
| С верным секретом | `200 OK` |

> Фронтенд этот endpoint не вызывает — он только для Telegram.

---

### 4. GET /api/admin/metadata — исправлено

**Было:** возвращал фильтры каталога (типы, бренды, цвета) — это была ошибка реализации.

**Стало:** возвращает реальную статистику системы.

**Ответ `200`:**
```json
{
  "products_total":  514,
  "users_total":     8,
  "orders_total":    1,
  "orders_revenue":  5932,
  "knowledge_count": 181
}
```

---

### 5. POST /v1/products/images — теперь возвращает ID

**Было:**
```json
{ "uploaded": 1 }
```

**Стало:**
```json
{
  "uploaded": 1,
  "items": [
    {
      "id": 800,
      "product_id": 382,
      "image_url": "https://supabase.iq-home.kz/storage/v1/object/public/product-images/382/382_front.jpg",
      "bucket": "product-images",
      "object_path": "382/382_front.jpg",
      "display_order": 0
    }
  ]
}
```

Теперь можно сразу удалить загруженный файл через `DELETE /v1/products/images/item?id={id}`.

---

## Актуальный контракт — уточнения по api.md

### GET /api/products/{id}

Поле называется `name`, а не `name_raw`:

```json
{
  "id": 42,
  "name": "Люстра Maytoni Cascade",
  "article": "MOD123",
  "price": 45000,
  "stock": 3,
  "product_type": "Люстра",
  "description": "Описание...",
  "configurator_type": "",
  "brand":  { "id": 2, "name": "Maytoni" },
  "color":  { "id": 3, "name": "Золото" },
  "series": { "id": 4, "name": "Cascade" },
  "images": [{ "id": 1, "url": "https://...", "path": "42/file.jpg" }]
}
```

---

### POST /api/products/search

Актуальное тело запроса:

```json
{
  "search":   "люстра для гостиной",
  "brandId":  3,
  "colorId":  5,
  "seriesId": null,
  "type":     "Люстра",
  "minPrice": 10000,
  "maxPrice": 100000,
  "sortBy":   "relevance",
  "limit":    20,
  "page":     1
}
```

`sortBy`: `"relevance"` / `"price_asc"` / `"price_desc"`. Все поля опциональны.

Актуальный ответ — поле `total` (не `count`):

```json
{
  "items": [
    {
      "id": 42,
      "name": "Люстра Maytoni Cascade",
      "article": "MOD123",
      "price": 45000,
      "score": 0.87,
      "type": "Люстра",
      "configurator_type": "",
      "brand": "Maytoni",
      "color": "Золото",
      "series": "Cascade",
      "images": [{ "id": 1, "url": "https://...", "path": "42/file.jpg" }]
    }
  ],
  "total": 510
}
```

---

### Роль пользователя

Фактическое значение поля `role` — `"customer"`, а не `"user"`:

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "full_name": "Иван Иванов",
  "phone": "+77001234567",
  "avatar_url": "https://...",
  "role": "customer"
}
```

---

### Кодировка русских символов

Если в ответах API видны кракозябры (`??N????>N?...`) — это проблема терминала (PowerShell не отображает UTF-8 по умолчанию).

Через браузер, Postman или curl в нормальном окружении — всё читается корректно. Данные в базе и ответы бэкенда в UTF-8.

---

## Что нужно от фронтенда

1. **Убрать `NEXT_PUBLIC_CHAT_INTERNAL_TOKEN`** — токен не должен быть публичной переменной. Все вызовы `/v1/*` должны идти только через server-side код (Next.js API routes, server actions).

2. **Перевыпустить скомпрометированные токены:**
   - Supabase access/refresh token пользователя — invalidate сессию
   - `CHAT_INTERNAL_TOKEN` / `NEXT_PUBLIC_CHAT_INTERNAL_TOKEN` — заменить значение
   - Admin credentials (`Aurora` / `Gaming`) — заменить если используются в production

3. **Обновить `product_id` в запросах history** — было `productId`, теперь `product_id`.
