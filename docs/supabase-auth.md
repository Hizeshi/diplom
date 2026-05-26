# Supabase Auth — Документация для фронтенда

## Конфигурация

```ts
// lib/supabase.ts
import { createClient } from '@supabase/supabase-js'

export const supabase = createClient(
  'https://supabase.iq-home.kz',
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoiYW5vbiIsImlzcyI6InN1cGFiYXNlIiwiZXhwIjoxNzk5NTM1NjAwfQ.RATSRoI-eBiMucsCuV_t1oObkhRuLd1UpaEFQ0mtoeM'
)
```

> **Важно:** anon key — публичный, его можно хранить в коде. Никогда не используй `service_role` key на фронтенде.

---

## Установка

```bash
npm install @supabase/supabase-js
```

---

## Как получить токен для Go бэкенда

Все защищённые эндпоинты (`/api/user/*`) требуют JWT токен. Он хранится в сессии Supabase:

```ts
const { data: { session } } = await supabase.auth.getSession()
const token = session?.access_token  // передавать в Authorization: Bearer <token>
```

Пример запроса к бэкенду:

```ts
async function apiRequest(path: string, options: RequestInit = {}) {
  const { data: { session } } = await supabase.auth.getSession()
  
  return fetch(`https://chat.iq-home.kz${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(session ? { Authorization: `Bearer ${session.access_token}` } : {}),
      ...options.headers,
    },
  })
}
```

---

## Регистрация

### По email + пароль

```ts
const { data, error } = await supabase.auth.signUp({
  email: 'user@example.com',
  password: 'password123',
  options: {
    data: {
      full_name: 'Иван Иванов',  // сохраняется в user_metadata
    }
  }
})

if (error) {
  console.error(error.message)
  // 'User already registered' — пользователь уже существует
  // 'Password should be at least 6 characters' — слабый пароль
}

// После signUp Supabase отправит письмо с подтверждением на email.
// До подтверждения session будет null.
if (data.session) {
  // email confirmation отключён — сразу залогинен
}
```

### Подтверждение email

После клика на ссылку в письме Supabase перенаправит на `SITE_URL` с токеном в URL. Supabase JS клиент обработает его автоматически если слушать `onAuthStateChange`.

---

## Вход

### По email + пароль

```ts
const { data, error } = await supabase.auth.signInWithPassword({
  email: 'user@example.com',
  password: 'password123',
})

if (error) {
  console.error(error.message)
  // 'Invalid login credentials' — неверный email или пароль
  // 'Email not confirmed' — email не подтверждён
}

const session = data.session   // содержит access_token и refresh_token
const user    = data.user      // данные пользователя
```

### Через Google (OAuth)

```ts
const { error } = await supabase.auth.signInWithOAuth({
  provider: 'google',
  options: {
    redirectTo: 'https://iq-home.kz',  // куда вернуть после авторизации
  }
})
```

После успешной авторизации Google перенаправит обратно с токеном в URL — Supabase JS автоматически подхватит сессию.

---

## Текущий пользователь и сессия

### Получить сессию (с кэшем, быстро)

```ts
const { data: { session } } = await supabase.auth.getSession()

if (!session) {
  // пользователь не авторизован
}

const token  = session.access_token   // JWT для Go бэкенда
const userId = session.user.id        // UUID пользователя
const email  = session.user.email
```

### Получить пользователя (запрос к серверу, актуально)

```ts
const { data: { user }, error } = await supabase.auth.getUser()
```

### Слушать изменения состояния авторизации

```ts
// Вызвать один раз при инициализации приложения
const { data: { subscription } } = supabase.auth.onAuthStateChange((event, session) => {
  switch (event) {
    case 'SIGNED_IN':
      // пользователь вошёл
      console.log('user:', session?.user)
      break
    case 'SIGNED_OUT':
      // пользователь вышел
      break
    case 'TOKEN_REFRESHED':
      // токен обновлён автоматически
      break
    case 'PASSWORD_RECOVERY':
      // пользователь перешёл по ссылке сброса пароля
      break
    case 'USER_UPDATED':
      // профиль обновлён
      break
  }
})

// При размонтировании компонента:
subscription.unsubscribe()
```

---

## Выход

```ts
const { error } = await supabase.auth.signOut()
// После этого session = null, onAuthStateChange вызовется с SIGNED_OUT
```

---

## Сброс пароля

### Шаг 1 — отправить письмо

```ts
const { error } = await supabase.auth.resetPasswordForEmail(
  'user@example.com',
  {
    redirectTo: 'https://iq-home.kz/update-password',
  }
)
// Supabase отправит письмо со ссылкой
```

### Шаг 2 — установить новый пароль

На странице `/update-password` пользователь попадает уже авторизованным (Supabase обработал токен из URL). Обнови пароль:

```ts
// Слушать PASSWORD_RECOVERY событие:
supabase.auth.onAuthStateChange(async (event, session) => {
  if (event === 'PASSWORD_RECOVERY') {
    const newPassword = prompt('Введите новый пароль')
    
    const { error } = await supabase.auth.updateUser({
      password: newPassword
    })
    
    if (!error) {
      alert('Пароль обновлён!')
    }
  }
})
```

---

## Обновление профиля

### Изменить email

```ts
const { error } = await supabase.auth.updateUser({
  email: 'new@example.com'
})
// Supabase отправит письмо с подтверждением на новый email
```

### Изменить пароль

```ts
const { error } = await supabase.auth.updateUser({
  password: 'newpassword123'
})
```

### Изменить user_metadata (имя, доп. данные)

```ts
const { error } = await supabase.auth.updateUser({
  data: {
    full_name: 'Новое Имя',
    phone: '+77001234567',
  }
})
```

---

## Автообновление токена

Supabase JS обновляет `access_token` автоматически через `refresh_token`. Токен живёт **1 час**, рефреш — до истечения сессии. Ничего делать не нужно.

Если нужен свежий токен перед каждым запросом:

```ts
// Supabase сам обновит если нужно при вызове getSession()
const { data: { session } } = await supabase.auth.getSession()
const token = session?.access_token
```

---

## Защищённые роуты (React пример)

```tsx
// hooks/useAuth.ts
import { useEffect, useState } from 'react'
import { Session } from '@supabase/supabase-js'
import { supabase } from '@/lib/supabase'

export function useAuth() {
  const [session, setSession] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Получить начальную сессию
    supabase.auth.getSession().then(({ data: { session } }) => {
      setSession(session)
      setLoading(false)
    })

    // Слушать изменения
    const { data: { subscription } } = supabase.auth.onAuthStateChange((_event, session) => {
      setSession(session)
    })

    return () => subscription.unsubscribe()
  }, [])

  return { session, user: session?.user, loading }
}
```

```tsx
// components/ProtectedRoute.tsx
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { session, loading } = useAuth()

  if (loading) return <div>Загрузка...</div>
  if (!session) return <Navigate to="/login" />

  return <>{children}</>
}
```

---

## Полный пример: форма входа

```tsx
import { useState } from 'react'
import { supabase } from '@/lib/supabase'

export function LoginForm() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError('')

    const { error } = await supabase.auth.signInWithPassword({ email, password })

    if (error) {
      setError(
        error.message === 'Invalid login credentials'
          ? 'Неверный email или пароль'
          : error.message
      )
    }
    // При успехе onAuthStateChange сработает и обновит состояние приложения

    setLoading(false)
  }

  return (
    <form onSubmit={handleSubmit}>
      <input type="email" value={email} onChange={e => setEmail(e.target.value)} />
      <input type="password" value={password} onChange={e => setPassword(e.target.value)} />
      {error && <p>{error}</p>}
      <button type="submit" disabled={loading}>
        {loading ? 'Входим...' : 'Войти'}
      </button>
      <button type="button" onClick={() => supabase.auth.signInWithOAuth({ provider: 'google' })}>
        Войти через Google
      </button>
    </form>
  )
}
```

---

## Частые ошибки

| Сообщение | Причина | Решение |
|---|---|---|
| `Invalid login credentials` | Неверный email/пароль или пользователь не существует | Показать общее сообщение без уточнения |
| `Email not confirmed` | Не подтверждён email | Попросить проверить почту |
| `User already registered` | Email уже используется | Предложить войти или сбросить пароль |
| `Password should be at least 6 characters` | Слишком короткий пароль | Валидация на клиенте |
| `Token has expired or is invalid` | Ссылка из письма устарела | Запросить новое письмо |
| `Email rate limit exceeded` | Слишком много писем | Подождать и попробовать позже |

---

## Таблица profiles

После регистрации Supabase создаёт запись в `auth.users`. Наш бэк хранит доп. данные в таблице `profiles` (связана с `auth.users` по UUID).

Получить данные профиля можно через Go бэк:
```
GET /api/user/session  →  { id, email, full_name, phone, avatar_url, role }
```

Обновить профиль:
```
PUT /api/admin/users/{id}  (admin)
POST /api/user/avatar       (загрузка аватара)
```
