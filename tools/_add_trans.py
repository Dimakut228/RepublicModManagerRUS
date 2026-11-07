import json
from pathlib import Path
base = Path("Source/launcher/locales")
en_more = {
    "catalog.detail.original": "Original",
    "catalog.detail.translated": "Translated",
    "catalog.detail.refresh": "Re-translate",
    "catalog.detail.translate_busy": "Translating…",
    "catalog.detail.translate_failed": "Translation failed: ",
    "catalog.translate.notice": "Mod descriptions are translated via Google; you can turn this off in the settings.",
    "catalog.translate.notice_ok": "Got it",
    "catalog.translate.notice_ack": "Translation settings reminder acknowledged",
}
ru_more = {
    "catalog.detail.original": "Оригинал",
    "catalog.detail.translated": "Перевод",
    "catalog.detail.refresh": "Перевести заново",
    "catalog.detail.translate_busy": "Перевод…",
    "catalog.detail.translate_failed": "Не удалось перевести: ",
    "catalog.translate.notice": "Описание модов переводится через Google; отключить можно в настройках.",
    "catalog.translate.notice_ok": "Понятно",
    "catalog.translate.notice_ack": "Напоминание о настройках перевода закрыто",
}
for name, extra in (("en.json", en_more), ("ru.json", ru_more)):
    p = base / name
    d = json.loads(p.read_text(encoding="utf-8-sig"))
    for k, v in extra.items():
        d["strings"].setdefault(k, v)
    data = {"meta": d["meta"], "strings": dict(sorted(d["strings"].items()))}
    p.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("translation keys")
