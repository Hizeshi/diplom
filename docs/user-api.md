# User API — эндпоинты для фронтенда

Базовый URL: `https://chat.iq-home.kz/api/user`  
Все запросы требуют заголовка:
```
Authorization: Bearer <supabase_access_token>
```

---

## Корзина

### GET /api/user/cart
Получить корзину пользователя.

**Ответ:**
```json
{
  "data": [
    {
      "id": 1,
      "product_id": 42,
      "name": "Выключатель FD",
      "price": 3500,
      "quantity": 2,
      "image_url": "https://..."
    }
  ]
}
```

---

### POST /api/user/cart
Добавить товар в корзину (или установить количество если уже есть).

**Тело:**
```json
{ "productId": 42, "quantity": 1 }
```

> ⚠️ **Важно:** quantity — это **абсолютное** значение, не дельта.  
> Если товар уже есть в корзине, quantity перезапишет текущее значение.  
> Для изменения количества (+1 / -1) используйте `PUT /api/user/cart/{productId}`.

**Ответ:** `{ "success": true }`

---

### PUT /api/user/cart/{productId}  ← **НОВЫЙ**
Изменить количество существующего товара в корзине.

```
PUT /api/user/cart/42
```

**Тело:**
```json
{ "quantity": 3 }
```

> quantity — абсолютное значение (не дельта).  
> Например, если было 2 и нажали «+», считать `newQty = old + 1` на фронте и слать `{ "quantity": 3 }`.

**Ответ:** `{ "success": true }`

**Как исправить `updateQuantity` в `useCartStore`:**
```ts
// Было (неправильно):
await userApiRequest("/cart", "POST", { productId: id, quantity: delta });

// Стало (правильно):
const newQty = Math.max(1, item.quantity + delta);
await userApiRequest(`/cart/${id}`, "PUT", { quantity: newQty });
```

---

### DELETE /api/user/cart/{productId}
Удалить товар из корзины.

```
DELETE /api/user/cart/42
```

> ⚠️ **Важно:** productId передаётся в **URL**, не в теле запроса.

**Как исправить `removeItem` в `useCartStore`:**
```ts
// Было (неправильно):
await userApiRequest("/cart", "DELETE", { productId: id });

// Стало (правильно):
await userApiRequest(`/cart/${id}`, "DELETE");
```

**Ответ:** `204 No Content`

---

### GET /api/user/cart/compatibility
Проверить совместимость товаров в корзине через ИИ.

**Ответ:**
```json
{
  "compatible": false,
  "item_count": 3,
  "issues": [
    {
      "type": "incompatible",
      "product_ids": [10, 20],
      "message": "Рамка G-серия не подходит к механизму FD — серии физически несовместимы."
    },
    {
      "type": "warning",
      "product_ids": [30],
      "message": "Диммер рассчитан на резистивную нагрузку. Уточните тип ламп."
    }
  ]
}
```

- `type: "incompatible"` — блокировать оформление заказа
- `type: "warning"` — показывать предупреждение, заказ не блокировать
- При `500` — показывать корзину без блока совместимости

---

## Избранное

### GET /api/user/favorites
**Ответ:**
```json
{
  "data": [
    { "product_id": 42, "name": "Выключатель FD", "price": 3500, "image_url": "..." }
  ]
}
```

---

### POST /api/user/favorites
Toggle: добавить если нет, убрать если есть.

**Тело:**
```json
{ "productId": 42 }
```

**Ответ:** `{ "success": true }`

---

### DELETE /api/user/favorites/{productId}
Явно убрать из избранного (без toggle).

```
DELETE /api/user/favorites/42
```

**Ответ:** `204 No Content`

---

## История просмотров

### GET /api/user/history
**Ответ:**
```json
{
  "data": [
    { "product_id": 42, "name": "...", "price": 3500, "image_url": "...", "viewed_at": "2026-05-27T..." }
  ]
}
```

---

### POST /api/user/history
Записать просмотр товара.

**Тело** — принимается оба варианта:
```json
{ "product_id": 42 }
```
или
```json
{ "productId": 42 }
```

**Ответ:** `{ "success": true }`

---

### GET /api/user/recommendations
Рекомендации на основе истории просмотров (похожие товары).

**Ответ:**
```json
{
  "data": [
    { "id": 55, "name": "...", "price": 2800, "image_url": "..." }
  ]
}
```

---

## Заказы

### GET /api/user/orders
Список заказов пользователя.

**Ответ:**
```json
{
  "data": [
    {
      "id": 7,
      "status": "pending",
      "total_amount": 15000,
      "payment_method": "card",
      "payment_status": "pending",
      "full_name": "Иван Иванов",
      "phone": "+7 701 123 45 67",
      "address": "Алматы, ул. Абая 1",
      "created_at": "2026-05-27T12:00:00Z"
    }
  ]
}
```

---

### GET /api/user/orders/{id}
Детали конкретного заказа.

```
GET /api/user/orders/7
```

> ⚠️ **Важно:** id в **URL**, не query-параметр.

**Как исправить в `orders/[id]/page.tsx`:**
```ts
// Было (неправильно — n8n-формат):
await userApiRequest(`/order_details?id=${params.id}`, 'GET');

// Стало (правильно):
await userApiRequest(`/orders/${params.id}`, 'GET');
```

**Ответ:**
```json
{
  "id": 7,
  "status": "confirmed",
  "total_amount": 15000,
  "payment_method": "card",
  "payment_status": "paid",
  "full_name": "Иван Иванов",
  "phone": "+7 701 123 45 67",
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

---

## Оформление заказа

### POST /api/user/checkout

**Тело:**
```json
{
  "name": "Иван Иванов",
  "phone": "+7 701 123 45 67",
  "address": "Алматы, ул. Абая 1",
  "payment_method": "card"
}
```

`payment_method`: `"card"` | `"cash"` | `"other"`

**Ответ при `"card"`:**
```json
{
  "success": true,
  "order_id": 7,
  "payment_url": "https://payment.example.com?order_id=7"
}
```

**Ответ при `"cash"` / `"other"`:**
```json
{
  "success": true,
  "order_id": 7,
  "message": "Заказ принят. Менеджер свяжется с вами."
}
```

После успешного checkout корзина очищается автоматически.

---

## Профиль и сессия

### GET /api/user/session
Получить данные профиля + активную чат-сессию.

**Ответ:**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "full_name": "Иван Иванов",
  "phone": "+7 701 123 45 67",
  "avatar_url": "https://...",
  "role": "customer"
}
```

---

## Аватар

### POST /api/user/avatar
Привязать URL аватара к профилю (после самостоятельной загрузки в Supabase Storage).

**Тело:**
```json
{
  "avatar_url": "https://supabase.iq-home.kz/storage/v1/object/public/avatars/...",
  "avatar_path": "user-id/filename.jpg"
}
```

**Ответ:** `{ "success": true }`

---

### DELETE /api/user/avatar
Удалить аватар (очищает запись в БД, файл из Storage удаляет сервер).

**Ответ:** `204 No Content`

---

## Коды ошибок

| HTTP | Причина |
|------|---------|
| `200` | Успех |
| `204` | Успех без тела (DELETE) |
| `400` | Неверные параметры запроса |
| `401` | Нет или невалидный JWT |
| `404` | Ресурс не найден |
| `500` | Ошибка сервера |

---

## Сводная таблица — что изменилось по сравнению с n8n

| Операция | Было (n8n) | Стало (Go backend) | Статус |
|---|---|---|---|
| Получить корзину | `GET /cart` | `GET /api/user/cart` | ✅ то же |
| Добавить в корзину | `POST /cart { productId, quantity }` | `POST /api/user/cart { productId, quantity }` | ✅ то же |
| **Изменить количество** | `POST /cart { productId, quantity: delta }` | `PUT /api/user/cart/{productId} { quantity: abs }` | ⚠️ **исправить** |
| **Удалить из корзины** | `DELETE /cart { productId: id }` (body) | `DELETE /api/user/cart/{id}` (URL param) | ⚠️ **исправить** |
| Проверка совместимости | — | `GET /api/user/cart/compatibility` | 🆕 новый |
| Избранное toggle | `POST /favorites { productId }` | `POST /api/user/favorites { productId }` | ✅ то же |
| Добавить в историю | `POST /history { productId }` | `POST /api/user/history { productId }` | ✅ исправлено на бэке |
| **Детали заказа** | `GET /order_details?id=X` | `GET /api/user/orders/{id}` | ⚠️ **исправить** |
| Checkout | `POST /checkout` | `POST /api/user/checkout` | ✅ то же |
| Аватар | `POST /avatar { avatar_url, avatar_path }` | `POST /api/user/avatar { avatar_url, avatar_path }` | ✅ то же |
