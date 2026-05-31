# AI-проверка совместимости товаров в корзине

## Суть

`GET /api/user/cart/compatibility` — берёт корзину авторизованного пользователя,
отправляет её в LLM и возвращает список проблем совместимости.

Работает с товарами JASMART: проверяет серии, количество постов, тип диммера,
проходные выключатели, розетки SCHUKO.

---

## Авторизация

Требует Supabase JWT в заголовке:

```
Authorization: Bearer <access_token>
```

Без токена → `401 Unauthorized`.

---

## Запрос

```
GET /api/user/cart/compatibility
```

Параметров нет. Корзина берётся автоматически по токену пользователя.

---

## Ответ

### Поля

| Поле | Тип | Описание |
|------|-----|----------|
| `compatible` | `boolean` | `true` — проблем нет или только предупреждения без блокирующих ошибок |
| `item_count` | `number` | Количество товаров в корзине на момент проверки |
| `issues` | `Issue[]` | Список найденных проблем (пустой если всё ок) |

### Issue

| Поле | Тип | Описание |
|------|-----|----------|
| `type` | `"incompatible"` \| `"warning"` | Тип проблемы |
| `product_ids` | `number[]` | ID товаров, которых касается проблема |
| `message` | `string` | Описание проблемы на русском языке (1–2 предложения) |

**`type: "incompatible"`** — физически несовместимые товары (например, рамка FD + механизм G-серии).
Рекомендуется блокировать оформление заказа или показывать ошибку.

**`type: "warning"`** — требует внимания, но не блокирует (например, диммер без уточнения типа ламп).
Показывать как предупреждение.

---

## Примеры ответов

### Корзина совместима

```json
{
  "compatible": true,
  "item_count": 3,
  "issues": []
}
```

### Есть проблема (разные серии)

```json
{
  "compatible": false,
  "item_count": 2,
  "issues": [
    {
      "type": "incompatible",
      "product_ids": [10, 20],
      "message": "Рамка G-серия не подходит к механизму FD — серии физически несовместимы. Выберите рамку и механизм одной серии."
    }
  ]
}
```

### Предупреждение (диммер)

```json
{
  "compatible": true,
  "item_count": 1,
  "issues": [
    {
      "type": "warning",
      "product_ids": [30],
      "message": "Диммер рассчитан на резистивную нагрузку. Уточните тип ламп и наличие нейтрального провода перед покупкой."
    }
  ]
}
```

### Несколько проблем одновременно

```json
{
  "compatible": false,
  "item_count": 4,
  "issues": [
    {
      "type": "incompatible",
      "product_ids": [10, 20],
      "message": "Рамка FD не подходит к механизму G-серии."
    },
    {
      "type": "warning",
      "product_ids": [31],
      "message": "Один проходной выключатель не имеет смысла — для управления из двух точек нужно минимум два."
    }
  ]
}
```

### Корзина пуста

```json
{
  "compatible": true,
  "item_count": 0,
  "issues": []
}
```

---

## Когда вызывать

Рекомендуемые места:

1. **Страница корзины** — при открытии и после каждого изменения состава.
2. **Кнопка «Оформить заказ»** — проверить перед переходом к checkout.
3. **Карточка товара → «Добавить в корзину»** — можно вызвать после добавления чтобы сразу сообщить о конфликте.

Не нужно вызывать при каждом рендере компонента — только при изменении корзины.

---

## Реализация на React

```tsx
// hooks/useCompatibility.ts
import { useEffect, useState } from 'react'

interface Issue {
  type: 'incompatible' | 'warning'
  product_ids: number[]
  message: string
}

interface CompatibilityResult {
  compatible: boolean
  item_count: number
  issues: Issue[]
}

export function useCompatibility(cartVersion: number) {
  const [result, setResult] = useState<CompatibilityResult | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (cartVersion === 0) return

    setLoading(true)
    fetch('/api/user/cart/compatibility', {
      headers: { Authorization: `Bearer ${supabase.auth.session()?.access_token}` },
    })
      .then(r => r.json())
      .then(setResult)
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [cartVersion])

  return { result, loading }
}
```

```tsx
// components/CompatibilityBanner.tsx
import { useCompatibility } from '../hooks/useCompatibility'

export function CompatibilityBanner({ cartVersion }: { cartVersion: number }) {
  const { result, loading } = useCompatibility(cartVersion)

  if (loading || !result || result.issues.length === 0) return null

  const errors   = result.issues.filter(i => i.type === 'incompatible')
  const warnings = result.issues.filter(i => i.type === 'warning')

  return (
    <div className="compatibility-banner">
      {errors.map((issue, idx) => (
        <div key={idx} className="compatibility-issue compatibility-issue--error">
          <span className="compatibility-issue__icon">⚠️</span>
          <p>{issue.message}</p>
        </div>
      ))}
      {warnings.map((issue, idx) => (
        <div key={idx} className="compatibility-issue compatibility-issue--warning">
          <span className="compatibility-issue__icon">ℹ️</span>
          <p>{issue.message}</p>
        </div>
      ))}
    </div>
  )
}
```

```tsx
// pages/CartPage.tsx
export function CartPage() {
  const [cartVersion, setCartVersion] = useState(1)

  const handleCartChange = () => setCartVersion(v => v + 1)

  return (
    <div>
      <CartItems onAdd={handleCartChange} onRemove={handleCartChange} />

      {/* Блок совместимости — появляется автоматически при проблемах */}
      <CompatibilityBanner cartVersion={cartVersion} />

      <CheckoutButton />
    </div>
  )
}
```

---

## Реализация на Vue

```vue
<!-- components/CompatibilityBanner.vue -->
<template>
  <div v-if="issues.length > 0" class="compatibility-banner">
    <div
      v-for="(issue, idx) in issues"
      :key="idx"
      :class="['compatibility-issue', `compatibility-issue--${issue.type}`]"
    >
      <span>{{ issue.type === 'incompatible' ? '⚠️' : 'ℹ️' }}</span>
      <p>{{ issue.message }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useSupabaseUser } from '@/composables/supabase'

const props = defineProps<{ cartVersion: number }>()

const issues = ref([])
const { token } = useSupabaseUser()

watch(() => props.cartVersion, async () => {
  const res = await fetch('/api/user/cart/compatibility', {
    headers: { Authorization: `Bearer ${token.value}` },
  })
  const data = await res.json()
  issues.value = data.issues ?? []
}, { immediate: true })
</script>
```

---

## CSS (базовый)

```css
.compatibility-banner {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 16px 0;
}

.compatibility-issue {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 16px;
  border-radius: 8px;
  font-size: 14px;
  line-height: 1.5;
}

.compatibility-issue--error {
  background: #fff5f5;
  border: 1px solid #fc8181;
  color: #c53030;
}

.compatibility-issue--warning {
  background: #fffbeb;
  border: 1px solid #f6ad55;
  color: #92400e;
}

.compatibility-issue__icon {
  font-size: 16px;
  flex-shrink: 0;
  margin-top: 1px;
}
```

---

## Блокировка checkout при несовместимости

```tsx
function CheckoutButton({ cartVersion }: { cartVersion: number }) {
  const { result } = useCompatibility(cartVersion)

  const hasBlockingIssue = result?.issues.some(i => i.type === 'incompatible') ?? false

  return (
    <button
      disabled={hasBlockingIssue}
      title={hasBlockingIssue ? 'Устраните несовместимость перед оформлением' : ''}
      onClick={handleCheckout}
    >
      Оформить заказ
    </button>
  )
}
```

---

## Коды ошибок

| HTTP | Причина |
|------|---------|
| `200` | Успешная проверка (даже если есть проблемы) |
| `401` | Нет или невалидный JWT |
| `500` | Ошибка на сервере (LLM недоступен и т.п.) |

При `500` — показывать корзину без блока совместимости, не блокировать checkout.

---

## Что проверяет AI

| Правило | Тип |
|---------|-----|
| Рамка и механизмы разных серий (FD / G-серия / G-Flex) | `incompatible` |
| Количество механизмов не совпадает с суммой постов рамок | `incompatible` |
| Диммер в корзине — нужно уточнить тип ламп и проводку | `warning` |
| Один проходной выключатель (нужно минимум 2) | `warning` |
| Розетка SCHUKO — нужна трёхпроводная разводка | `warning` |
