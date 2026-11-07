import json
from pathlib import Path
base = Path("Source/launcher/locales")
en_more = {"catalog.detail.thumbnails_missing": "Preview loading…",
           "catalog.err.network": "Network error",
           "catalog.err.status": "Steam returned status ",
           "catalog.err.page": "Steam returned an error page",
           "catalog.err.parse": "Could not parse the Steam page",
           "catalog.err.details": "Details service error"}
ru_more = {"catalog.detail.thumbnails_missing": "Превью загружается…",
           "catalog.err.network": "Нет сети",
           "catalog.err.status": "Steam вернул статус ",
           "catalog.err.page": "Steam вернул страницу ошибки",
           "catalog.err.parse": "Не удалось разобрать страницу Steam",
           "catalog.err.details": "Сбой сервиса деталей"}
for fname, extra in (("en.json", en_more), ("ru.json", ru_more)):
    p = base / fname
    d = json.loads(p.read_text(encoding="utf-8-sig"))
    for k, v in extra.items():
        d["strings"].setdefault(k, v)
    d["strings"] = dict(sorted(d["strings"].items()))
    p.write_text(json.dumps(d, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("thumbs keys")
