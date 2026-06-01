# Frontend Handoff — полная документация API

**Дата:** 2026-06-01  
**Бэкенд:** `https://chat.iq-home.kz`  
**Supabase:** `https://supabase.iq-home.kz`

---

## Что изменилось — срочные исправления

| # | Что сломано | Файл | Исправление |
|---|---|---|---|
| 1 | `updateQuantity` шлёт дельту через `POST /cart` | `useCartStore` | `PUT /cart/{id} { quantity: абсолютное }` |
| 2 | `removeItem` шлёт `DELETE /cart` с телом | `useCartStore` | `DELETE /cart/{id}` без тела |
| 3 | Детали заказа — n8n формат | `orders/[id]/page.tsx` | `GET /orders/{id}` |
| 4 | `userApi.ts` не поддерживает `PUT` метод | `userApi.ts` | Добавить `PUT` в `UserApiMethod` |

### Исправление 1 + 4 — изменение количества (PUT)

```ts
// userApi.ts — добавить PUT:
const BASE_URLS: Record<'GET' | 'POST' | 'PUT' | 'DELETE', string | undefined> = {
  GET:    process.env.NEXT_PUBLIC_API_USER_GET,
  POST:   process.env.NEXT_PUBLIC_API_USER_POST,
  PUT:    process.env.NEXT_PUBLIC_API_USER_POST, // тот же базовый URL
  DELETE: process.env.NEXT_PUBLIC_API_USER_DELETE,
};

// useCartStore — updateQuantity:
// ❌ Было:
await userApiRequest("/cart", "POST", { productId: id, quantity: delta });
// ✅ Стало:
const newQty = Math.max(1, currentItem.quantity + delta);
await userApiRequest(`/cart/${id}`, "PUT", { quantity: newQty });
```

### Исправление 2 — удаление из корзины

```ts
// ❌ Было:
await userApiRequest("/cart", "DELETE", { productId: id });
// ✅ Стало:
await userApiRequest(`/cart/${id}`, "DELETE");
```

### Исправление 3 — детали заказа

```ts
// ❌ Было:
await userApiRequest(`/order_details?id=${params.id}`, "GET");
// ✅ Стало:
await userApiRequest(`/orders/${params.id}`, "GET");
```

---

## Локализация

Все эндпоинты каталога принимают `?lang=` или заголовок `Accept-Language`.

```
GET  /api/filters?lang=kk
POST /api/products/search?lang=en
GET  /api/products/42?lang=kk
```

Поддерживаемые значения: `ru` (по умолчанию), `kk`, `en`.

Ответ возвращает тот же JSON-формат, но с переведёнными строками.

---

## GET /api/filters

Справочники для фильтрации и конфигуратора. **Не требует авторизации.**

```
GET https://chat.iq-home.kz/api/filters?lang=ru
```

**Ответ:**
```json
{
  "types": ["Выключатель", "Диммер", "Розетка", "Рамка", "Переключатель"],
  "brands": [
    { "id": 37, "name": "JASMART" }
  ],
  "colors": [
    { "id": 32, "name": "Алюминий" },
    { "id": 33, "name": "Антрацит" }
  ],
  "series": [
    { "id": 197, "name": "FD-серия",  "brand_id": 37 },
    { "id": 198, "name": "FS-серия",  "brand_id": 37 },
    { "id": 199, "name": "G-Classic", "brand_id": 37 },
    { "id": 200, "name": "G-Flex",    "brand_id": 37 },
    { "id": 201, "name": "G-Glass",   "brand_id": 37 },
    { "id": 202, "name": "G-Metal",   "brand_id": 37 }
  ],
  "min_price": 500,
  "max_price": 85000
}
```

> **Важно для конфигуратора:** `series[].brand_id` — добавлено. Используй для фильтрации серий по выбранному бренду в `TopSelectors`.

---

## POST /api/products/search

Поиск товаров. **Не требует авторизации.**

```
POST https://chat.iq-home.kz/api/products/search?lang=ru
Content-Type: application/json
```

**Тело запроса** (все поля опциональны):
```json
{
  "search":   "выключатель одноклавишный",
  "brandId":  37,
  "colorId":  32,
  "seriesId": 197,
  "type":     "Выключатель",
  "minPrice": 1000,
  "maxPrice": 15000,
  "sortBy":   "relevance",
  "limit":    12,
  "page":     1
}
```

`sortBy`: `"relevance"` | `"price_asc"` | `"price_desc"`

**Ответ:**
```json
{
  "items": [
    {
      "id": 42,
      "name": "Выключатель JASMART FD одноклавишный, белый матовый",
      "article": "FD-SW-01-WM",
      "price": 3500,
      "score": 0.91,
      "type": "Выключатель",
      "configurator_type": "key_1",
      "brand": "JASMART",
      "color": "Белый матовый",
      "series": "FD-серия",
      "images": [
        { "id": 1, "url": "https://supabase.iq-home.kz/storage/v1/object/public/...", "path": "42/img.jpg" }
      ]
    }
  ],
  "total": 128
}
```

**Конфигуратор использует:**
```ts
// VisualBuilder.tsx делает этот запрос:
getAllItems({ seriesId: selectedSeriesId, limit: 150 })
// Что равно:
POST /api/products/search { seriesId: X, limit: 150 }
// Затем фильтрует по:
item.configurator_type === 'frame_1'  // рамки
item.configurator_type === 'key_1'    // клавиши
```

`configurator_type` возможные значения: `"frame_1"`, `"key_1"`, `"not_applicable"`, `""` (пусто).

---

## GET /api/products/{id}

Карточка товара. **Не требует авторизации.**

```
GET https://chat.iq-home.kz/api/products/42?lang=kk
```

**Ответ:**
```json
{
  "id": 42,
  "name": "Выключатель JASMART FD одноклавишный, белый матовый",
  "article": "FD-SW-01-WM",
  "price": 3500,
  "stock": 15,
  "product_type": "Выключатель",
  "description": "Одноклавишный выключатель серии FD...",
  "configurator_type": "key_1",
  "brand":  { "id": 37,  "name": "JASMART" },
  "color":  { "id": 32,  "name": "Белый матовый" },
  "series": { "id": 197, "name": "FD-серия" },
  "images": [
    { "id": 1, "url": "https://...", "path": "42/img.jpg" }
  ],
  "model_url": "https://..."
}
```

`model_url` — ссылка на 3D-модель, может быть пустой строкой.

---

## Пользовательские эндпоинты (требует JWT)

```
Authorization: Bearer <supabase_access_token>
```

### Корзина

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | `/api/user/cart` | — | `{ data: CartItem[] }` |
| POST | `/api/user/cart` | `{ productId, quantity }` | `{ success: true }` |
| PUT | `/api/user/cart/{productId}` | `{ quantity }` | `{ success: true }` |
| DELETE | `/api/user/cart/{productId}` | — | `204` |
| GET | `/api/user/cart/compatibility` | — | `CompatibilityResult` |

**CartItem:**
```json
{
  "id": 1,
  "product_id": 42,
  "name": "Выключатель FD",
  "price": 3500,
  "quantity": 2,
  "image_url": "https://..."
}
```

**PUT quantity** — абсолютное значение, не дельта:
```ts
// При нажатии «+»:
const newQty = currentItem.quantity + 1;
await userApiRequest(`/cart/${productId}`, "PUT", { quantity: newQty });
```

**CompatibilityResult:**
```json
{
  "compatible": false,
  "item_count": 3,
  "issues": [
    {
      "type": "incompatible",
      "product_ids": [10, 20],
      "message": "Рамка G-серия не подходит к механизму FD."
    },
    {
      "type": "warning",
      "product_ids": [30],
      "message": "Диммер: уточните тип нагрузки."
    }
  ]
}
```

- `type: "incompatible"` → блокировать кнопку «Оформить заказ»
- `type: "warning"` → показывать предупреждение, не блокировать
- При ошибке 500 → игнорировать, показывать корзину как обычно

---

### Избранное

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | `/api/user/favorites` | — | `{ data: FavoriteItem[] }` |
| POST | `/api/user/favorites` | `{ productId }` | `{ success: true }` — toggle |
| DELETE | `/api/user/favorites/{productId}` | — | `204` |

**FavoriteItem:**
```json
{ "product_id": 42, "name": "Выключатель FD", "price": 3500, "image_url": "..." }
```

---

### История и рекомендации

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | `/api/user/history` | — | `{ data: HistoryItem[] }` |
| POST | `/api/user/history` | `{ productId }` или `{ product_id }` | `{ success: true }` |
| GET | `/api/user/recommendations` | — | `{ data: RecommendedItem[] }` |

**HistoryItem:**
```json
{ "product_id": 42, "name": "...", "price": 3500, "image_url": "...", "viewed_at": "2026-05-27T12:00:00Z" }
```

---

### Заказы

| Метод | Путь | Ответ |
|---|---|---|
| GET | `/api/user/orders` | `{ data: Order[] }` |
| GET | `/api/user/orders/{id}` | `OrderDetail` |

**Order:**
```json
{
  "id": 7,
  "status": "confirmed",
  "payment_status": "paid",
  "payment_method": "card",
  "total_amount": 15000,
  "full_name": "Иван Иванов",
  "phone": "+77011234567",
  "address": "Алматы, ул. Абая 1",
  "created_at": "2026-05-27T12:00:00Z"
}
```

**OrderDetail** — то же + `items[]`:
```json
{
  "items": [
    { "product_id": 42, "product_name": "Выключатель FD", "price_at_purchase": 3500, "quantity": 2 }
  ]
}
```

Статусы заказа: `new` | `confirmed` | `processing` | `cancelled`  
Статусы оплаты: `pending` | `success` | `failed`

---

### Checkout

```
POST /api/user/checkout
```

**Тело:**
```json
{
  "full_name":     "Иван Иванов",
  "phone":         "+77011234567",
  "address":       "г. Алматы, ул. Абая 1",
  "paymentMethod": "card"
}
```

`paymentMethod`: `"card"` | `"kaspi"` | `"cash"` | `"other"`

**Ответ при card / kaspi:**
```json
{
  "success":    true,
  "orderId":    101,
  "paymentUrl": "https://l-xor-pay.vercel.app/?orderId=101&amount=27357"
}
```

**Ответ при cash / other:**
```json
{
  "success": true,
  "orderId": 101,
  "message": "Заказ принят. Менеджер свяжется с вами."
}
```

> Ключи в ответе — **camelCase**: `orderId`, `paymentUrl`.  
> После успешного checkout корзина очищается автоматически.

---

### Профиль, аватар, сессия

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | `/api/user/session` | — | `Profile` |
| POST | `/api/user/avatar` | `{ avatar_url, avatar_path }` | `{ success: true }` |
| DELETE | `/api/user/avatar` | — | `204` |

**Profile:**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "full_name": "Иван Иванов",
  "phone": "+77011234567",
  "avatar_url": "https://...",
  "role": "customer"
}
```

---

## Чат

Требует авторизации (Supabase JWT). Для гостей показывать кнопку «Войти».

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/chat` | Отправить сообщение ИИ |
| GET | `/api/chat/history?sessionId={id}` | История чата |

**Тело POST /api/chat:**
```json
{
  "message":    "Посоветуй выключатель для спальни",
  "session_id": "любая-строка-uuid"
}
```

**Ответ:**
```json
{
  "answer": "Рекомендую выключатель JASMART FD...",
  "products": [
    { "id": 42, "name": "...", "price": 3500, "url": "https://iq-home.kz/products/42" }
  ]
}
```

---

## Платёжная система

Вебхук от `l-xor-pay.vercel.app`:
```
POST https://chat.iq-home.kz/api/payment/notify
{ "orderId": 101, "status": "success" }
```

Редиректы после оплаты (настроить в платёжном приложении):
- Успех → `https://iq-home.kz/orders/{orderId}`
- Отказ → `https://iq-home.kz/cart`

---

## Коды ошибок

```json
{ "error": "описание" }
```

| Код | Причина |
|---|---|
| `400` | Неверные параметры / пустое обязательное поле |
| `401` | Нет JWT или невалидный |
| `403` | Нет доступа к ресурсу |
| `404` | Ресурс не найден |
| `500` | Ошибка сервера |

---

## CORS — разрешённые origins

- `https://iq-home.kz`
- `https://www.iq-home.kz`
- `https://l-xor-pay.vercel.app`

---

## Переводы справочников — что нужно от тебя

Бренды, цвета и серии ещё не переведены. В репозитории лежит файл  
`docs/refs_to_translate.json` — 1 бренд, 25 цветов, 7 серий.

Формат для перевода и загрузки:
```json
{
  "brands": {
    "kk": [{ "id": 37, "name": "JASMART" }],
    "en": [{ "id": 37, "name": "JASMART" }]
  },
  "colors": {
    "kk": [{ "id": 32, "name": "Ақ матовый" }, ...],
    "en": [{ "id": 32, "name": "White matte" }, ...]
  },
  "series": {
    "kk": [{ "id": 197, "name": "FD-сериясы" }, ...],
    "en": [{ "id": 197, "name": "FD Series" }, ...]
  }
}
```

Загрузить: `python3 docs/upload_refs_translations.py refs_translated.json`
