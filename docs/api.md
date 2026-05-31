# IQ-Home Backend — API Reference

## Базовый URL

```
https://chat.iq-home.kz
```

---

## Аутентификация

| Тип | Где используется | Как передавать |
|---|---|---|
| **Supabase JWT** | `/api/user/*` | `Authorization: Bearer <token>` |
| **BasicAuth** | `/api/admin/*` | `Authorization: Basic base64(user:pass)` |
| **X-Internal-Token** | `/v1/*` | `X-Internal-Token: <token>` |

### Формат ошибок

Все ошибки возвращают JSON:

```json
{ "error": "описание ошибки" }
```

| Код | Значение |
|---|---|
| `400` | Неверный запрос / отсутствуют обязательные поля |
| `401` | Не авторизован |
| `403` | Доступ запрещён |
| `404` | Не найдено |
| `500` | Внутренняя ошибка сервера |

---

## Health

### GET /health

Проверка работоспособности сервера.

**Ответ `200`:**
```json
{ "status": "ok" }
```

---

## Продукты (публичные)

### GET /api/filters

Получить все доступные фильтры для каталога.

**Ответ `200`:**
```json
{
  "types":     ["Люстра", "Розетки электрические"],
  "brands":    [{ "id": 2, "name": "Maytoni" }],
  "colors":    [{ "id": 3, "name": "Золото" }],
  "series":    [{ "id": 4, "name": "Cascade" }],
  "min_price": 5000,
  "max_price": 250000
}
```

> **Примечание:** `types` — массив строк (не объектов), так как типы товаров хранятся как текстовое поле.

---

### GET /api/products/{id}

Получить товар по ID.

**Параметры пути:** `id` — числовой ID товара.

**Ответ `200`:**
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
  "images": [{ "id": 1, "url": "https://...", "path": "382/file.jpg" }]
}
```

---

### POST /api/products/search

Семантический + полнотекстовый поиск по каталогу.

**Тело запроса:**
```json
{
  "search":   "золотая люстра для гостиной",
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

Все поля опциональны. Если `search` пустой — только фильтрация. `sortBy`: `"relevance"` / `"price_asc"` / `"price_desc"`.

**Ответ `200`:**
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
  "total": 128
}
```

---

## Чат (публичный)

### POST /api/chat

Отправить сообщение AI-ассистенту.

> Если пользователь авторизован — передать `Authorization: Bearer <token>` (опционально, для персонализации).

**Тело запроса:**
```json
{
  "message":    "Посоветуй люстру для спальни 20м²",
  "session_id": "uuid-или-любая-строка",
  "match_count": 5,
  "topic_filter": ""
}
```

| Поле | Обязательно | Описание |
|---|---|---|
| `message` | ✅ | Текст сообщения |
| `session_id` | ✅ | Идентификатор сессии (генерировать на клиенте) |
| `match_count` | ❌ | Кол-во товаров для поиска (по умолч. 5) |
| `topic_filter` | ❌ | Тема для фильтрации базы знаний |

**Ответ `200`:**
```json
{
  "answer": "Для спальни 20м² рекомендую...",
  "products": [
    {
      "id": 42,
      "name": "Люстра Maytoni",
      "price": 45000,
      "url": "https://iq-home.kz/products/42",
      "metadata": { "image": "https://..." }
    }
  ],
  "quote_url": ""
}
```

---

### GET /api/chat/history?sessionId={id}

Получить историю чата по session_id.

**Параметры запроса:** `sessionId` — строка.

**Ответ `200`:**
```json
{
  "data": [
    {
      "role": "customer",
      "content": "Привет",
      "time": "14:32"
    },
    {
      "role": "assistant",
      "content": "Здравствуйте! Чем могу помочь?",
      "time": "14:32"
    }
  ]
}
```

---

## Контакты (публичный)

### POST /api/contact

Отправить заявку с формы обратной связи.

**Тело запроса:**
```json
{
  "name":    "Иван Иванов",
  "email":   "ivan@example.com",
  "message": "Хочу уточнить условия доставки"
}
```

**Ответ `200`:**
```json
{ "success": true }
```

---

## Оплата

### POST /api/payment/notify

Вебхук от платёжного сайта `l-xor-pay.vercel.app` после завершения оплаты. Без HMAC-подписи.

**Тело запроса:**
```json
{
  "orderId": 101,
  "status":  "success"
}
```

`status`: `"success"` или `"failed"`.

**Ответ `200`:**
```json
{ "success": true }
```

> Этот эндпоинт нужно прописать в настройках платёжного сайта как webhook URL:
> `https://chat.iq-home.kz/api/payment/notify`

---

### POST /api/payment/webhook

Вебхук от платёжного провайдера с HMAC-SHA256 подписью (для будущей интеграции).  
Требует заголовок `X-Payment-Signature`.

**Тело запроса:**
```json
{
  "order_id": 101,
  "status":   "success",
  "tx_id":    "txn_abc123"
}
```

**Ответ `200`:**
```json
{ "success": true }
```

---

## Пользователь (требует Supabase JWT)

> Все маршруты ниже требуют заголовок `Authorization: Bearer <supabase_access_token>`

---

### GET /api/user/session

Получить данные текущего пользователя.

**Ответ `200`:**
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

### GET /api/user/cart

Получить содержимое корзины.

**Ответ `200`:**
```json
{
  "data": [
    {
      "product_id": 42,
      "name": "Люстра Maytoni",
      "price": 45000,
      "quantity": 2,
      "image": "https://..."
    }
  ]
}
```

---

### POST /api/user/cart

Добавить товар в корзину. Если товар уже есть — перезаписывает quantity.

**Тело запроса:**
```json
{
  "productId": 42,
  "quantity":  1
}
```

**Ответ `200`:**
```json
{ "success": true }
```

---

### PUT /api/user/cart/{productId}

Изменить количество товара в корзине. Принимает **абсолютное** значение, не дельту.

```
PUT /api/user/cart/42
```

**Тело запроса:**
```json
{ "quantity": 3 }
```

**Ответ `200`:**
```json
{ "success": true }
```

---

### DELETE /api/user/cart/{productId}

Удалить товар из корзины. ID товара передаётся в **URL**, не в теле.

```
DELETE /api/user/cart/42
```

**Ответ `204`** (без тела)

---

### GET /api/user/cart/compatibility

Проверить совместимость товаров в корзине через ИИ.

**Ответ `200`:**
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
- `type: "warning"` — показывать предупреждение, не блокировать
- При `500` — показывать корзину без блока совместимости

---

### GET /api/user/favorites

Получить список избранного.

**Ответ `200`:**
```json
{
  "data": [
    {
      "product_id": 42,
      "name": "Люстра Maytoni",
      "price": 45000,
      "image": "https://..."
    }
  ]
}
```

---

### POST /api/user/favorites

Переключить товар в избранном (добавить если нет, удалить если есть).

**Тело запроса:**
```json
{ "productId": 42 }
```

**Ответ `200`:**
```json
{ "success": true }
```

---

### DELETE /api/user/favorites/{productId}

Удалить конкретный товар из избранного.

**Ответ `204`** (без тела)

---

### GET /api/user/history

Получить историю просмотренных товаров.

**Ответ `200`:**
```json
{
  "data": [
    {
      "product_id": 42,
      "name": "Люстра Maytoni",
      "price": 45000,
      "image": "https://...",
      "viewed_at": "2024-05-01T12:00:00Z"
    }
  ]
}
```

---

### POST /api/user/history

Добавить товар в историю просмотров.

**Тело запроса** — принимается оба варианта:
```json
{ "product_id": 42 }
```
или
```json
{ "productId": 42 }
```

**Ответ `200`:**
```json
{ "success": true }
```

---

### GET /api/user/recommendations

Получить персонализированные рекомендации на основе истории просмотров.

**Ответ `200`:**
```json
{
  "data": [
    {
      "product_id": 55,
      "name": "Бра Odeon",
      "price": 12000,
      "image": "https://..."
    }
  ]
}
```

---

### GET /api/user/orders

Получить список заказов пользователя.

**Ответ `200`:**
```json
{
  "data": [
    {
      "id": 101,
      "status": "confirmed",
      "payment_status": "success",
      "payment_method": "card",
      "total_amount": 90000,
      "created_at": "2024-05-01T12:00:00Z"
    }
  ]
}
```

---

### GET /api/user/orders/{id}

Получить детальную информацию о заказе. ID передаётся в **URL**, не как query-параметр.

```
GET /api/user/orders/101
```

**Ответ `200`:**
```json
{
  "id": 101,
  "status": "confirmed",
  "payment_status": "success",
  "payment_method": "card",
  "total_amount": 90000,
  "full_name": "Иван Иванов",
  "phone": "+77001234567",
  "address": "г. Алматы, ул. Абая 1",
  "created_at": "2024-05-01T12:00:00Z",
  "items": [
    {
      "product_id": 42,
      "product_name": "Люстра Maytoni",
      "quantity": 2,
      "price_at_purchase": 45000
    }
  ]
}
```

---

### POST /api/user/checkout

Оформить заказ из корзины.

**Тело запроса:**
```json
{
  "full_name":     "Иван Иванов",
  "phone":         "+77001234567",
  "address":       "г. Алматы, ул. Абая 1",
  "paymentMethod": "card"
}
```

| Поле | Обязательно | Описание |
|---|---|---|
| `full_name` | ✅ | ФИО получателя |
| `phone` | ✅ | Телефон (мин. 7 символов) |
| `address` | ✅ | Адрес доставки (мин. 5 символов) |
| `paymentMethod` | ✅ | `"card"` / `"cash"` / `"kaspi"` / `"other"` |

**Ответ при `paymentMethod: "card"` или `"kaspi"`:**
```json
{
  "success":    true,
  "orderId":    101,
  "paymentUrl": "https://l-xor-pay.vercel.app/?orderId=101&amount=27357"
}
```

**Ответ при `paymentMethod: "cash"` или `"other"`:**
```json
{
  "success": true,
  "orderId": 101,
  "message": "Заказ принят. Менеджер свяжется с вами."
}
```

> После успешного checkout корзина очищается автоматически.  
> Ключи в ответе: **`orderId`** и **`paymentUrl`** (camelCase).

---

### POST /api/user/avatar

Загрузить аватар пользователя. Тип запроса: `multipart/form-data`.

| Поле формы | Тип | Описание |
|---|---|---|
| `avatar` | file | Изображение (JPG, PNG, WebP) |

**Ответ `200`:**
```json
{ "avatar_url": "https://..." }
```

---

### DELETE /api/user/avatar

Удалить аватар пользователя.

**Ответ `200`:**
```json
{ "success": true }
```

---

## Admin (требует BasicAuth)

> `Authorization: Basic base64("admin:password")`

---

### GET /api/admin/chats

Список всех чат-сессий.

**Ответ `200`:**
```json
{
  "data": [
    {
      "session_id": "abc-123",
      "is_human_mode": false,
      "last_active": "19.05 14:32",
      "last_message": "Какая цена на люстру?"
    }
  ]
}
```

---

### GET /api/admin/chats/{sessionId}

История сообщений конкретной сессии.

**Ответ `200`:**
```json
{
  "data": [
    {
      "role": "customer",
      "content": "Привет",
      "sender_type": "user",
      "created_at": "2024-05-01T12:00:00Z"
    }
  ]
}
```

---

### POST /api/admin/chats/{sessionId}/toggle

Переключить режим «живого оператора» для сессии.

**Тело запроса:**
```json
{ "is_human_mode": true }
```

**Ответ `200`:**
```json
{ "success": true }
```

---

### POST /api/admin/chats/{sessionId}/message

Отправить сообщение от имени менеджера в чат.

**Тело запроса:**
```json
{ "text": "Здравствуйте! Уточните, пожалуйста, ваш вопрос." }
```

**Ответ `200`:**
```json
{ "success": true }
```

---

### GET /api/admin/products?search=&limit=20&page=1

Список товаров с поиском и пагинацией.

**Query параметры:** `search` (строка), `limit` (по умолч. 20), `page` (по умолч. 1).

**Ответ `200`:**
```json
{
  "items": [
    {
      "id": 42,
      "name_raw": "Люстра Maytoni",
      "article": "MOD123",
      "price": 45000,
      "product_type": "Люстра",
      "brand": "Maytoni",
      "series": "Cascade",
      "color": "Золото",
      "configurator_type": null,
      "images": [{ "id": 1, "url": "https://...", "path": "42/image.jpg" }]
    }
  ],
  "total": 500,
  "page":  1,
  "limit": 20
}
```

---

### POST /api/admin/products

Создать новый товар.

**Тело запроса:**
```json
{
  "name_raw":     "Люстра Maytoni Cascade",
  "article":      "MOD123",
  "price":        45000,
  "stock":        5,
  "product_type": "Люстра",
  "description":  "Описание...",
  "brand_id":     2,
  "series_id":    4,
  "color_id":     3
}
```

**Ответ `201`:**
```json
{ "id": 43 }
```

---

### PUT /api/admin/products/{id}

Обновить товар (передаются только изменяемые поля).

**Тело запроса:** те же поля что и при создании, все опциональны.

**Ответ `200`:**
```json
{ "success": true }
```

---

### PUT /api/admin/products/{id}/configurator

Обновить тип конфигуратора для товара.

**Тело запроса:**
```json
{ "type": "lighting" }
```

**Ответ `200`:**
```json
{ "success": true }
```

---

### DELETE /api/admin/products/{id}

Мягкое удаление товара (soft delete).

**Ответ `204`** (без тела)

---

### POST /api/admin/products/scan-duplicates

Найти дублирующиеся товары по векторному сходству (> 99%).

**Ответ `200`:**
```json
{
  "data": [
    {
      "product_id_1": 42,
      "name_1": "Люстра Maytoni A",
      "product_id_2": 55,
      "name_2": "Люстра Maytoni B",
      "similarity": 0.995
    }
  ]
}
```

---

### GET /api/admin/users?search=&limit=20&page=1

Список пользователей.

**Ответ `200`:**
```json
{
  "items": [
    {
      "id": "uuid",
      "email": "user@example.com",
      "full_name": "Иван Иванов",
      "phone": "+77001234567",
      "role": "customer",
      "created_at": "2024-05-01T12:00:00Z"
    }
  ],
  "total": 100
}
```

---

### GET /api/admin/users/{id}

Детальная информация о пользователе: профиль + заказы + корзина + история.

**Ответ `200`:**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "full_name": "Иван Иванов",
  "phone": "+77001234567",
  "avatar_url": "https://...",
  "role": "customer",
  "orders": [
    { "id": 101, "total_amount": 90000, "status": "confirmed", "created_at": "..." }
  ],
  "cart": [
    { "product_id": 42, "name": "Люстра", "quantity": 1, "price": 45000 }
  ],
  "history": [
    { "product_id": 42, "name": "Люстра", "viewed_at": "..." }
  ]
}
```

---

### PUT /api/admin/users/{id}

Обновить профиль пользователя. Можно обновить email и роль за один запрос.

**Тело запроса:**
```json
{
  "full_name": "Новое Имя",
  "phone":     "+77009998877",
  "email":     "new@example.com",
  "role":      "admin"
}
```

`role`: `"user"` или `"admin"`.

**Ответ `200`:**
```json
{ "success": true }
```

---

### DELETE /api/admin/users/{id}

Полное удаление пользователя (из Supabase Auth + локальная БД).

**Ответ `204`** (без тела)

---

### DELETE /api/admin/users/{id}/cart

Очистить корзину пользователя.

**Ответ `204`** (без тела)

---

### DELETE /api/admin/users/{id}/history

Очистить историю просмотров пользователя.

**Ответ `204`** (без тела)

---

### GET /api/admin/orders

Список всех заказов.

**Ответ `200`:**
```json
{
  "data": [
    {
      "id": 101,
      "user_id": "uuid",
      "status": "confirmed",
      "payment_status": "success",
      "payment_method": "card",
      "total_amount": 90000,
      "full_name": "Иван Иванов",
      "phone": "+77001234567",
      "created_at": "2024-05-01T12:00:00Z"
    }
  ]
}
```

---

### GET /api/admin/orders/{id}

Детали заказа с позициями.

**Ответ `200`:**
```json
{
  "id": 101,
  "status": "confirmed",
  "payment_status": "success",
  "total_amount": 90000,
  "full_name": "Иван Иванов",
  "phone": "+77001234567",
  "address": "г. Алматы, ул. Абая 1",
  "comment": "",
  "items": [
    { "product_id": 42, "name": "Люстра Maytoni", "quantity": 2, "price": 45000 }
  ]
}
```

---

### GET /api/admin/knowledge

Список записей базы знаний AI-ассистента.

**Ответ `200`:**
```json
{
  "data": [
    {
      "id": 1,
      "topic": "Доставка",
      "content": "Доставка по Алматы 1-2 дня...",
      "created_at": "2024-05-01T12:00:00Z"
    }
  ]
}
```

---

### POST /api/admin/knowledge

Создать или обновить запись базы знаний. Если передан `id` — обновляет, иначе создаёт.

**Тело запроса:**
```json
{
  "id":      1,
  "topic":   "Доставка",
  "content": "Доставка по Алматы 1-2 дня, бесплатно от 50 000 тенге."
}
```

**Ответ `200`:**
```json
{ "success": true }
```

---

### DELETE /api/admin/knowledge/{id}

Удалить запись базы знаний.

**Ответ `204`** (без тела)

---

### GET /api/admin/contacts

Список заявок с формы обратной связи.

**Ответ `200`:**
```json
{
  "data": [
    {
      "id": 1,
      "name": "Иван Иванов",
      "email": "ivan@example.com",
      "message": "Хочу уточнить...",
      "status": "new",
      "created_at": "2024-05-01T12:00:00Z"
    }
  ]
}
```

---

### GET /api/admin/metadata

Общая статистика по системе.

**Ответ `200`:**
```json
{
  "products_total":  500,
  "users_total":     1200,
  "orders_total":    340,
  "orders_revenue":  15000000,
  "knowledge_count": 24
}
```

---

## Internal API (требует X-Internal-Token)

> Используется сервисами (Telegram бот, n8n, скрипты). Фронтенд эти эндпоинты не вызывает напрямую.

---

### POST /v1/chat

То же что `/api/chat`, но без опциональной авторизации.

---

### POST /v1/chat/media

Отправить медиафайл в чат. Тип: `multipart/form-data`.

| Поле | Тип | Описание |
|---|---|---|
| `session_id` | string | ID сессии |
| `user_id` | string | ID пользователя (опционально) |
| `message_type` | string | `"voice"` / `"photo"` / `"document"` |
| `platform` | string | Платформа (опционально) |
| `file` | file | Файл |

**Ответ `200`:** такой же как `/api/chat`.

---

### POST /v1/quotes

Сгенерировать PDF коммерческого предложения.

**Тело запроса:**
```json
{
  "company_name": "ТОО «Пример»",
  "contact_name": "Иван Иванов",
  "phone":        "+77001234567",
  "email":        "ivan@example.com",
  "note":         "Доставка включена",
  "items": [
    {
      "name":     "Люстра Maytoni Cascade",
      "article":  "MOD123",
      "quantity": 2,
      "price":    45000
    }
  ]
}
```

**Ответ `200`:**
```json
{ "url": "https://supabase.iq-home.kz/storage/v1/object/public/quotes/2024/05/01-143200/quote.pdf" }
```

---

### POST /v1/products/import

Массовый импорт товаров из CSV или Excel (.xlsx). Тип: `multipart/form-data`.

| Поле | Тип | Описание |
|---|---|---|
| `file` | file | CSV или XLSX файл |

**Формат файла** — первая строка заголовок, далее данные. Поддерживаемые названия колонок (на русском или английском):

| Колонка | Алиасы | Обязательно |
|---|---|---|
| article | артикул | ✅ |
| name | название, наименование | ✅ |
| price | цена | ❌ |
| type | product_type, тип | ❌ |
| brand | бренд | ❌ |
| series | серия | ❌ |
| color | цвет | ❌ |
| description | описание | ❌ |

Разделитель в CSV определяется автоматически (`,` или `;`).  
Бренд, серия и цвет подтягиваются по имени — если не найдены, поле остаётся пустым.  
Если товар с таким `article` уже существует — обновляется.

**Ответ `200`:**
```json
{
  "total":    150,
  "upserted": 148,
  "failed":   2,
  "errors": [
    "row 12 (ART-005): ..."
  ]
}
```

---

### POST /v1/products/vectorize

Запустить (пере)векторизацию всех товаров через OpenAI. Долгая операция.

**Ответ `200`:**
```json
{
  "total":   500,
  "success": 498,
  "failed":  2
}
```

---

### POST /v1/products/images

Массовая загрузка изображений. Тип: `multipart/form-data`.  
Имя файла должно начинаться с ID товара: `42_front.jpg`, `42_back.jpg`.

| Поле | Тип | Описание |
|---|---|---|
| `files` | file[] | Несколько файлов |

**Ответ `200`:**
```json
{ "uploaded": 12 }
```

---

### POST /v1/products/images/item

Добавить одно изображение к товару. Тип: `multipart/form-data`.

| Поле | Тип | Описание |
|---|---|---|
| `product_id` | string | ID товара |
| `display_order` | string | Порядок отображения (по умолч. 0) |
| `file` | file | Файл изображения |

**Ответ `201`:**
```json
{
  "id": 99,
  "product_id": 42,
  "image_url": "https://...",
  "bucket": "product-images",
  "object_path": "42/image.jpg",
  "display_order": 0
}
```

---

### PUT /v1/products/images/item

Обновить порядок отображения изображения.

**Тело запроса:**
```json
{
  "id":            99,
  "display_order": 2
}
```

**Ответ `200`:**
```json
{ "success": true }
```

---

### DELETE /v1/products/images/item?id={id}

Удалить изображение из БД и Supabase Storage.

**Ответ `204`** (без тела)

---

### POST /v1/telegram/webhook

Вебхук для Telegram бота. Вызывается Telegram-серверами.

Заголовок: `X-Telegram-Bot-Api-Secret-Token: <secret>`
