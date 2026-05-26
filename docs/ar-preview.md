# AR-примерка товаров — документация для фронтенда

## Суть функции

Покупатель открывает карточку товара и нажимает кнопку "Смотреть в AR". Телефон включает камеру, определяет поверхность стены/пола и накладывает на неё 3D-модель товара в реальном масштабе. Никакого приложения скачивать не нужно — всё работает прямо в браузере.

- **Android** — через Google Scene Viewer (ARCore)
- **iOS** — через AR Quick Look (встроен в Safari)
- **Desktop** — интерактивный 3D-просмотр с вращением мышью (без AR)

---

## Как API сообщает о наличии модели

В ответе `GET /api/products/:id` появилось новое поле `model_url`.

**Если у товара есть 3D-модель:**
```json
{
  "id": 92,
  "name": "JASMART FD-серия Выкл. 1-кл 10A 250V~, Белый мат.",
  "price": 4297,
  "model_url": "https://supabase.iq-home.kz/storage/v1/object/public/models/light_switch.glb",
  ...
}
```

**Если модели нет** — поле `model_url` отсутствует в ответе (не `null`, а вообще нет ключа).

Товары с 3D-моделями на данный момент:
| ID | Товар | Тип |
|----|-------|-----|
| 92 | JASMART FD Выключатель 1-кл, Белый мат. | Выключатель |
| 140 | JASMART G-серия Диммер LED 30-400W | Диммер |
| 280 | JASMART FD Розетка SCHUKO со шторками | Розетка |

---

## Подключение библиотеки

Добавить один раз в `<head>` или в конце `<body>`. Библиотека от Google, весит ~1MB, грузится с CDN:

```html
<script
  type="module"
  src="https://ajax.googleapis.com/ajax/libs/model-viewer/3.4.0/model-viewer.min.js">
</script>
```

Или через npm если используете сборщик:
```bash
npm install @google/model-viewer
```
```js
import '@google/model-viewer';
```

---

## Минимальная реализация

### Логика в компоненте карточки товара

```js
// Проверяем есть ли модель у товара
const hasAR = !!product.model_url;
```

### Шаблон (React пример)

```jsx
function ProductCard({ product }) {
  const hasAR = !!product.model_url;

  return (
    <div className="product-card">
      {/* Обычные фото товара */}
      <img src={product.images[0]?.url} alt={product.name} />

      {/* AR-блок — показываем только если есть модель */}
      {hasAR && (
        <div className="ar-section">
          <model-viewer
            src={product.model_url}
            alt={product.name}
            ar
            ar-modes="scene-viewer quick-look"
            camera-controls
            auto-rotate
            shadow-intensity="1"
            style={{ width: '100%', height: '400px' }}
          >
            {/* Кнопка AR — появляется автоматически на мобильных */}
            <button slot="ar-button" className="ar-button">
              📱 Смотреть в AR
            </button>
          </model-viewer>
        </div>
      )}
    </div>
  );
}
```

### Шаблон (Vue пример)

```vue
<template>
  <div class="product-card">
    <img :src="product.images[0]?.url" :alt="product.name" />

    <div v-if="product.model_url" class="ar-section">
      <model-viewer
        :src="product.model_url"
        :alt="product.name"
        ar
        ar-modes="scene-viewer quick-look"
        camera-controls
        auto-rotate
        shadow-intensity="1"
        style="width: 100%; height: 400px"
      >
        <button slot="ar-button" class="ar-button">
          📱 Смотреть в AR
        </button>
      </model-viewer>
    </div>
  </div>
</template>
```

### Шаблон (чистый HTML)

```html
<!-- Показываем блок только если model_url пришёл из API -->
<div id="ar-container" style="display: none;">
  <model-viewer
    id="ar-viewer"
    ar
    ar-modes="scene-viewer quick-look"
    camera-controls
    auto-rotate
    shadow-intensity="1"
    style="width: 100%; height: 400px;"
  >
    <button slot="ar-button" class="ar-button">📱 Смотреть в AR</button>
  </model-viewer>
</div>

<script>
  fetch('/api/products/92')
    .then(r => r.json())
    .then(product => {
      if (product.model_url) {
        const viewer = document.getElementById('ar-viewer');
        const container = document.getElementById('ar-container');
        viewer.setAttribute('src', product.model_url);
        viewer.setAttribute('alt', product.name);
        container.style.display = 'block';
      }
    });
</script>
```

---

## Атрибуты model-viewer — что означает каждый

| Атрибут | Обязателен | Описание |
|---------|-----------|----------|
| `src` | ✓ | URL `.glb` файла из поля `model_url` |
| `alt` | ✓ | Название товара — для доступности |
| `ar` | ✓ | Включает режим AR на мобильных |
| `ar-modes` | ✓ | `scene-viewer` — Android, `quick-look` — iOS |
| `camera-controls` | рекомендуется | Позволяет вращать модель мышью/пальцем |
| `auto-rotate` | опционально | Автоматическое вращение пока не трогают |
| `shadow-intensity` | опционально | Тень под моделью, `"1"` — реалистично |
| `poster` | опционально | Изображение-заглушка пока модель грузится |
| `loading="lazy"` | опционально | Ленивая загрузка модели |

---

## Кнопка AR — стилизация

`slot="ar-button"` — это специальный слот компонента. Кнопка показывается **только на устройствах с поддержкой AR** (Android с ARCore или iOS Safari). На десктопе кнопки не будет.

```css
.ar-button {
  position: absolute;
  bottom: 16px;
  right: 16px;
  background: #ffffff;
  border: 2px solid #2563eb;
  color: #2563eb;
  border-radius: 8px;
  padding: 10px 20px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
}

.ar-button:hover {
  background: #2563eb;
  color: #ffffff;
}
```

---

## Бейдж "AR-примерка" на карточке в каталоге

В поиске (`POST /api/products/search`) поле `model_url` **не возвращается** — только в карточке товара. Чтобы показать бейдж в каталоге, нужно ориентироваться на список ID с моделями или добавить флаг `has_ar: true` в поиск (можно запросить у бэкенда).

Простой вариант — хардкод ID на фронте пока товаров мало:

```js
const AR_PRODUCT_IDS = [92, 140, 280];

// В карточке каталога
{AR_PRODUCT_IDS.includes(product.id) && (
  <span className="ar-badge">AR</span>
)}
```

```css
.ar-badge {
  background: #7c3aed;
  color: white;
  font-size: 11px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 4px;
  letter-spacing: 0.5px;
}
```

---

## Индикатор загрузки модели

Модели `.glb` могут весить 5–20MB. Пока грузится — показываем прогресс:

```jsx
<model-viewer src={product.model_url} ...>
  <div slot="progress-bar" className="progress-bar">
    <div className="progress-bar-inner" />
  </div>
  <button slot="ar-button" className="ar-button">📱 Смотреть в AR</button>
</model-viewer>
```

```css
.progress-bar {
  width: 100%;
  height: 4px;
  background: #e5e7eb;
  position: absolute;
  bottom: 0;
}
.progress-bar-inner {
  height: 100%;
  background: #2563eb;
  /* model-viewer автоматически управляет шириной через --progress-bar-width */
  width: var(--progress-bar-width, 0%);
  transition: width 0.3s;
}
```

---

## Пример полной интеграции в карточку товара

```jsx
import '@google/model-viewer';

function ProductPage({ productId }) {
  const [product, setProduct] = useState(null);

  useEffect(() => {
    fetch(`/api/products/${productId}`)
      .then(r => r.json())
      .then(setProduct);
  }, [productId]);

  if (!product) return <div>Загрузка...</div>;

  return (
    <div className="product-page">
      <h1>{product.name}</h1>
      <p className="price">{product.price.toLocaleString()} тг</p>

      {/* Галерея — если есть AR-модель, показываем 3D-просмотр вместо первого фото */}
      {product.model_url ? (
        <div className="ar-viewer-container">
          <model-viewer
            src={product.model_url}
            alt={product.name}
            ar
            ar-modes="scene-viewer quick-look"
            camera-controls
            auto-rotate
            auto-rotate-delay="3000"
            shadow-intensity="1"
            exposure="0.8"
            style={{ width: '100%', height: '450px', borderRadius: '12px' }}
          >
            <button slot="ar-button" className="ar-button">
              📱 Смотреть в своей комнате
            </button>
          </model-viewer>
          <p className="ar-hint">
            Зажмите и перетащите для вращения • На телефоне — нажмите кнопку AR
          </p>
        </div>
      ) : (
        <div className="image-gallery">
          {product.images.map(img => (
            <img key={img.id} src={img.url} alt={product.name} />
          ))}
        </div>
      )}

      <p className="description">{product.description}</p>
      <button className="buy-button">Добавить в корзину</button>
    </div>
  );
}
```

---

## Поддержка браузеров

| Платформа | Браузер | AR работает? |
|-----------|---------|--------------|
| Android 8+ | Chrome, любой браузер | ✓ (Google Scene Viewer) |
| iOS 12+ | Safari | ✓ (AR Quick Look) |
| iOS | Chrome, Firefox | ✗ (нет доступа к AR Quick Look) |
| Desktop | Chrome, Firefox, Safari | 3D без AR ✓ |

**Важно для iOS:** AR работает только в Safari. Если пользователь открыл сайт в Chrome на iPhone — кнопка AR не появится. Можно добавить подсказку:

```jsx
const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent);
const isIOSSafari = isIOS && /Safari/.test(navigator.userAgent) && !/Chrome/.test(navigator.userAgent);

{isIOS && !isIOSSafari && (
  <p className="safari-hint">
    Для AR-просмотра откройте страницу в Safari
  </p>
)}
```

---

## Краткий чеклист для реализации

- [ ] Подключить `@google/model-viewer` (CDN или npm)
- [ ] В компоненте карточки товара проверять `product.model_url`
- [ ] Если есть — рендерить `<model-viewer>` с атрибутами `ar`, `ar-modes`, `camera-controls`
- [ ] Добавить `<button slot="ar-button">` внутрь — для кнопки на мобильных
- [ ] Добавить CSS для кнопки и контейнера
- [ ] Добавить бейдж "AR" на карточки в каталоге (по списку ID или флагу)
- [ ] Опционально: индикатор загрузки, подсказка для iOS Chrome
