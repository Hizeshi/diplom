#!/usr/bin/env python3
"""
Загружает переводы справочников (бренды, цвета, серии) в Go backend.

Входной файл формат:
{
  "brands": {"kk": [{"id":37,"name":"JASMART"}], "en": [...]},
  "colors": {"kk": [{"id":1,"name":"Ақ матовый"}], "en": [...]},
  "series": {"kk": [{"id":197,"name":"FD-сериясы"}], "en": [...]}
}

Использование:
  python3 upload_refs_translations.py refs_translated.json
"""

import json
import sys
import http.client
import base64

API_HOST = "chat.iq-home.kz"
AUTH = base64.b64encode(b"Aurora:Gaming").decode()

ENDPOINTS = {
    "brands": "/api/admin/brands/{id}/i18n",
    "colors": "/api/admin/colors/{id}/i18n",
    "series": "/api/admin/series/{id}/i18n",
}


def upload_ref(resource: str, locale: str, items: list):
    endpoint_tpl = ENDPOINTS[resource]
    ok = failed = 0
    conn = http.client.HTTPSConnection(API_HOST, timeout=15)

    for item in items:
        rid = item["id"]
        payload = json.dumps({
            "locale": locale,
            "name": item.get("name", ""),
        }).encode("utf-8")

        try:
            conn.request(
                "PUT",
                endpoint_tpl.format(id=rid),
                body=payload,
                headers={
                    "Authorization": f"Basic {AUTH}",
                    "Content-Type": "application/json",
                    "Content-Length": str(len(payload)),
                },
            )
            resp = conn.getresponse()
            resp.read()
            if resp.status == 200:
                ok += 1
            else:
                print(f"  [WARN] {resource} id={rid} status={resp.status}")
                failed += 1
                conn = http.client.HTTPSConnection(API_HOST, timeout=15)
        except Exception as e:
            print(f"  [ERROR] {resource} id={rid}: {e}")
            failed += 1
            conn = http.client.HTTPSConnection(API_HOST, timeout=15)

    conn.close()
    return ok, failed


def main():
    if len(sys.argv) < 2:
        print("Использование: python3 upload_refs_translations.py refs_translated.json")
        sys.exit(1)

    with open(sys.argv[1], encoding="utf-8") as f:
        data = json.load(f)

    for resource in ("brands", "colors", "series"):
        block = data.get(resource, {})
        for locale in ("kk", "en"):
            items = block.get(locale, [])
            if not items:
                print(f"[{resource}/{locale}] нет данных, пропускаем")
                continue
            print(f"[{resource}/{locale}] Загружаю {len(items)}...")
            ok, failed = upload_ref(resource, locale, items)
            print(f"  Готово: {ok} успешно, {failed} ошибок")

    print("\nЗагрузка завершена.")


if __name__ == "__main__":
    main()
