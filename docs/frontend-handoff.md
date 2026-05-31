# Передача фронтенду — актуальное состояние бэкенда

**Дата:** 2026-05-31  
**Бэкенд:** `https://chat.iq-home.kz`  
**Supabase:** `https://supabase.iq-home.kz`

---

## Срочно исправить на фронте

Эти баги сейчас ломают функциональность:

| # | Файл | Проблема | Исправление |
|---|---|---|---|
| 1 | `useCartStore` | `updateQuantity` шлёт дельту через `POST /cart` | Слать абсолютное значение через `PUT /cart/{id}` |
| 2 | `useCartStore` | `removeItem` шлёт `DELETE /cart` с телом `{productId}` | Слать `DELETE /cart/{id}` без тела |
| 3 | `orders/[id]/page.tsx` | `GET /order_details?id=X` — n8n формат | Заменить на `GET /orders/{id}` |

### Исправление 1 — изменение количества

```ts
// ❌ Было:
await userApiRequest("/cart", "POST", { productId: id, quantity: delta });

// ✅ Стало (нужно добавить PUT в userApi.ts):
const newQty = Math.max(1, currentItem.quantity + delta);
await userApiRequest(`/cart/${id}`, "PUT", { quantity: newQty });
```

> `userApi.ts` не поддерживает метод `PUT` — нужно добавить его в тип `UserApiMethod` и в `BASE_URLS`.

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

## Локализация (новое)

Все публичные эндпоинты каталога поддерживают параметр `?lang=`:

```
GET /api/filters?lang=kk
GET /api/products/42?lang=en
POST /api/products/search?lang=kk
```

Поддерживаемые значения: `ru` (по умолчанию), `kk`, `en`.

Также принимается заголовок `Accept-Language: kk`.

Ответ возвращает тот же формат JSON, но с переведёнными значениями полей `name`, `description`, `brand`, `color`, `series`.

**Как передавать язык из фронта:** добавь `?lang=${currentLocale}` к запросам каталога.

---

## Чат — теперь требует авторизацию

`POST /api/chat` и `GET /api/chat/history` теперь закрыты за Supabase JWT.

```
Authorization: Bearer <supabase_access_token>
```

Неавторизованный запрос → `401 Unauthorized`.  
На фронте: показывать кнопку «Войти» вместо чата для гостей.

---

## Checkout — изменения

### Поля запроса

```json
{
  "full_name": "Иван Иванов",
  "phone": "+77001234567",
  "address": "г. Алматы, ул. Абая 1",
  "paymentMethod": "card"
}
```

> Поле называется `full_name` (не `name`).  
> `paymentMethod`: `"card"` | `"cash"` | `"kaspi"` | `"other"`.

### Ответ при оплате картой / Kaspi

```json
{
  "success": true,
  "orderId": 101,
  "paymentUrl": "https://l-xor-pay.vercel.app/?orderId=101&amount=27357"
}
```

> Ключи ответа — **camelCase**: `orderId`, `paymentUrl` (не `order_id`, не `payment_url`).

### Ответ при наличных / other

```json
{
  "success": true,
  "orderId": 101,
  "message": "Заказ принят. Менеджер свяжется с вами."
}
```

---

## Платёжная система — вебхук

Платёжный сайт `l-xor-pay.vercel.app` должен отправлять вебхук на:

```
POST https://chat.iq-home.kz/api/payment/notify
```

Тело:
```json
{ "orderId": 101, "status": "success" }
```

`status`: `"success"` или `"failed"`.

Редиректы после оплаты (настроить в платёжном приложении):
- Успех: `https://iq-home.kz/orders/{orderId}`
- Отказ: `https://iq-home.kz/cart`

---

## Полная таблица эндпоинтов пользователя

Базовый URL: `https://chat.iq-home.kz/api/user`  
Все запросы: `Authorization: Bearer <supabase_access_token>`

### Корзина

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | `/cart` | — | `{ data: [...] }` |
| POST | `/cart` | `{ productId, quantity }` | `{ success: true }` |
| PUT | `/cart/{productId}` | `{ quantity }` | `{ success: true }` |
| DELETE | `/cart/{productId}` | — | `204` |
| GET | `/cart/compatibility` | — | `{ compatible, issues[] }` |

**Структура элемента корзины:**
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

**Структура ответа совместимости:**
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
- `incompatible` → блокировать кнопку «Оформить заказ»
- `warning` → показывать предупреждение, не блокировать
- При ошибке 500 → показывать корзину без проверки совместимости

---

### Избранное

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | `/favorites` | — | `{ data: [...] }` |
| POST | `/favorites` | `{ productId }` | `{ success: true }` — toggle |
| DELETE | `/favorites/{productId}` | — | `204` |

---

### История и рекомендации

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | `/history` | — | `{ data: [...] }` |
| POST | `/history` | `{ productId }` или `{ product_id }` | `{ success: true }` |
| GET | `/recommendations` | — | `{ data: [...] }` |

---

### Заказы

| Метод | Путь | Ответ |
|---|---|---|
| GET | `/orders` | `{ data: [...] }` |
| GET | `/orders/{id}` | объект заказа с items[] |

**Структура элемента в списке заказов:**
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

**Структура детального заказа:**
```json
{
  "id": 7,
  "status": "confirmed",
  "payment_status": "paid",
  "total_amount": 15000,
  "full_name": "Иван Иванов",
  "phone": "+77011234567",
  "address": "Алматы, ул. Абая 1",
  "created_at": "2026-05-27T12:00:00Z",
  "items": [
    {
      "product_id": 42,
      "product_name": "Выключатель FD",
      "price_at_purchase": 3500,
      "quantity": 2
    }
  ]
}
```

**Статусы заказа:** `new` | `confirmed` | `processing` | `cancelled`  
**Статусы оплаты:** `pending` | `success` | `failed`

---

### Профиль, аватар, сессия

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | `/session` | — | профиль пользователя |
| POST | `/avatar` | `{ avatar_url, avatar_path }` | `{ success: true }` |
| DELETE | `/avatar` | — | `204` |

**Профиль:**
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

## Публичные эндпоинты каталога

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/filters` | Справочники фильтров |
| POST | `/api/products/search` | Поиск и пагинация |
| GET | `/api/products/{id}` | Карточка товара |

Все поддерживают `?lang=ru|kk|en`.

**Тело поиска:**
```json
{
  "search": "выключатель",
  "brandId": null,
  "colorId": null,
  "seriesId": null,
  "type": null,
  "minPrice": null,
  "maxPrice": null,
  "sortBy": "relevance",
  "limit": 12,
  "page": 1
}
```

**Ответ поиска:**
```json
{
  "items": [
    {
      "id": 42,
      "name": "Выключатель JASMART FD",
      "article": "FD-001",
      "price": 3500,
      "type": "Выключатель",
      "configurator_type": "",
      "brand": "JASMART",
      "color": "Белый матовый",
      "series": "FD",
      "images": [{ "id": 1, "url": "https://...", "path": "42/img.jpg" }]
    }
  ],
  "total": 128
}
```

**Ответ фильтров:**
```json
{
  "types": ["Выключатель", "Розетка", "Рамка"],
  "brands": [{ "id": 37, "name": "JASMART" }],
  "colors": [{ "id": 1, "name": "Белый матовый" }],
  "series": [{ "id": 2, "name": "FD" }],
  "min_price": 500,
  "max_price": 85000
}
```

---

## Коды ошибок

| Код | Причина |
|---|---|
| `400` | Неверные параметры / пустое обязательное поле |
| `401` | Нет JWT или невалидный |
| `403` | Нет доступа к ресурсу |
| `404` | Ресурс не найден |
| `500` | Ошибка сервера — показывать пользователю нейтральное сообщение |

Формат ошибки:
```json
{ "error": "описание ошибки" }
```

---

## CORS

Разрешённые origins:
- `https://iq-home.kz`
- `https://www.iq-home.kz`
- `https://l-xor-pay.vercel.app`

Если фронт работает с другого домена — сообщить, добавим.
