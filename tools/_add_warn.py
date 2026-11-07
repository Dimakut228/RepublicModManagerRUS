import json
from pathlib import Path
base = Path("Source/launcher/locales")
en_more = {
    "warn.no_metadata": "No local Workshop metadata",
    "warn.no_mod_name": "No mod name",
    "warn.no_preview_image": "No preview image",
    "warn.no_local_thumbnail": "No local thumbnail",
    "warn.folder_empty": "Workshop folder is empty",
    "warn.page_unavailable": "Workshop page is unavailable",
    "warn.delisted": "Workshop item appears delisted or inaccessible",
    "warn.removed": "Workshop item has been removed from the community",
    "warn.no_steam_library": "Workshop: Steam library could not be derived from the selected WRSR path",
}
ru_more = {
    "warn.no_metadata": "Нет локальных метаданных Workshop",
    "warn.no_mod_name": "Нет имени мода",
    "warn.no_preview_image": "Нет превью",
    "warn.no_local_thumbnail": "Нет локального превью",
    "warn.folder_empty": "Папка Workshop пуста",
    "warn.page_unavailable": "Страница Workshop недоступна",
    "warn.delisted": "Модуль Workshop скрыт или недоступен",
    "warn.removed": "Модуль удалён из Workshop",
    "warn.no_steam_library": "Workshop: библиотека Steam не определена по пути WRSR",
}
for name, extra in (("en.json", en_more), ("ru.json", ru_more)):
    p = base / name
    d = json.loads(p.read_text(encoding="utf-8-sig"))
    for k, v in extra.items():
        d["strings"].setdefault(k, v)
    # enforce parity
    en = json.loads((base / "en.json").read_text(encoding="utf-8-sig"))["strings"]
    ru = json.loads((base / "ru.json").read_text(encoding="utf-8-sig"))["strings"]
    for k, v in en.items():
        ru.setdefault(k, v)
    for k, v in ru.items():
        en.setdefault(k, v)
    (base / "en.json").write_text(json.dumps({"meta": json.loads((base / "en.json").read_text(encoding="utf-8-sig"))["meta"], "strings": dict(sorted(en.items()))}, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    (base / "ru.json").write_text(json.dumps({"meta": json.loads((base / "ru.json").read_text(encoding="utf-8-sig"))["meta"], "strings": dict(sorted(ru.items()))}, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("keys", len(en), len(ru))
