import json
from pathlib import Path
base = Path("Source/launcher/locales")
en_add = {
    "t.workshop_tab_label": "Workshop mods",
    "dashboard.metric.discovered_value": "%d plugins / %d workshop mods",
    "dashboard.metric.last_launch_state_none": "Never launched",
    "dashboard.metric.last_launch_state_ok": "Launcher OK",
    "dashboard.metric.last_launch_state_err": "Launch error",
}
ru_add = {
    "t.workshop_tab_label": "Моды Workshop",
    "dashboard.metric.discovered_value": "Плагины: %d • Workshop: %d",
    "dashboard.metric.last_launch_state_none": "Не запускался",
    "dashboard.metric.last_launch_state_ok": "Запуск прошёл",
    "dashboard.metric.last_launch_state_err": "Ошибка запуска",
}
for fname, extra in (("en.json", en_add), ("ru.json", ru_add)):
    p = base / fname
    d = json.loads(p.read_text(encoding="utf-8-sig"))
    for k, v in extra.items():
        d["strings"].setdefault(k, v)
    d["strings"] = dict(sorted(d["strings"].items()))
    p.write_text(json.dumps(d, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("keys added")
