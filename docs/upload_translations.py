#!/usr/bin/env python3
"""
Загружает переводы товаров в Go backend.

Использование:
  python3 upload_translations.py translated.json
"""

import json
import sys
import http.client
import base64

API_HOST = "chat.iq-home.kz"
AUTH = base64.b64encode(b"Aurora:Gaming").decode()


def upload(locale: str, items: list):
    ok = 0
    failed = 0
    conn = http.client.HTTPSConnection(API_HOST, timeout=15)

    for i, item in enumerate(items):
        product_id = item["id"]
        payload = json.dumps({
            "locale": locale,
            "name": item.get("name", ""),
            "description": item.get("description", ""),
            "params": item.get("params", ""),
        }).encode("utf-8")

        try:
            conn.request(
                "PUT",
                f"/api/admin/products/{product_id}/i18n",
                body=payload,
                headers={
                    "Authorization": f"Basic {AUTH}",
                    "Content-Type": "application/json",
                    "Content-Length": str(len(payload)),
                },
            )
            resp = conn.getresponse()
            resp.read()  # drain body

            if resp.status == 200:
                ok += 1
            else:
                print(f"  [WARN] id={product_id} status={resp.status}")
                failed += 1
                conn = http.client.HTTPSConnection(API_HOST, timeout=15)

        except Exception as e:
            print(f"  [ERROR] id={product_id}: {e}")
            failed += 1
            conn = http.client.HTTPSConnection(API_HOST, timeout=15)

        if (i + 1) % 50 == 0:
            print(f"  [{locale}] {i + 1}/{len(items)} загружено...")

    conn.close()
    return ok, failed


def main():
    if len(sys.argv) < 2:
        print("Использование: python3 upload_translations.py translated.json")
        sys.exit(1)

    with open(sys.argv[1], encoding="utf-8") as f:
        data = json.load(f)

    for locale in ("kk", "en"):
        items = data.get(locale, [])
        if not items:
            print(f"[{locale}] нет данных, пропускаем")
            continue

        print(f"\n[{locale}] Загружаю {len(items)} переводов...")
        ok, failed = upload(locale, items)
        print(f"[{locale}] Готово: {ok} успешно, {failed} ошибок")

    print("\nЗагрузка завершена.")


if __name__ == "__main__":
    main()
