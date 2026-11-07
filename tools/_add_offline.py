import json
from pathlib import Path
base = Path("Source/launcher/locales")
en_add = {
    "catalog.state.offline": "Offline — showing the last saved catalog data; press Refresh to retry",
    "catalog.state.installed": "Installed",
    "catalog.filters.period_hidden": "Period applies only to Trending",
}
ru_add = {
    "catalog.state.offline": "Нет сети — показаны сохранённые данные каталога; нажмите Обновить, чтобы повторить",
    "catalog.state.installed": "Установлен",
    "catalog.filters.period_hidden": "Период работает только для сортировки «Популярные»",
}
for fname, extra in (("en.json", en_add), ("ru.json", ru_add)):
    p = base / fname
    d = json.loads(p.read_text(encoding="utf-8-sig"))
    for k, v in extra.items():
        d["strings"].setdefault(k, v)
    d["strings"] = dict(sorted(d["strings"].items()))
    p.write_text(json.dumps(d, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("keys added")
