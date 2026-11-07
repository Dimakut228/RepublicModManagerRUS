import json
from pathlib import Path
base = Path("Source/launcher/locales")
en = {
    "dashboard.metric.discovered_value": "%d plugins / %d workshop mods",
    "dashboard.metric.enabled_detail_update": "%d workshop mods need updating",
    "dashboard.metric.enabled_detail_dev": "%d Development projects active",
}
ru = {
    "dashboard.metric.discovered_value": "Плагины: %d • Workshop: %d",
    "dashboard.metric.enabled_detail_update": "%d модов Workshop нужно обновить",
    "dashboard.metric.enabled_detail_dev": "%d активных проектов Development",
}
for fname, extra in (("en.json", en), ("ru.json", ru)):
    p = base / fname
    d = json.loads(p.read_text(encoding="utf-8-sig"))
    for k, v in extra.items():
        d["strings"][k] = v
    d["strings"] = dict(sorted(d["strings"].items()))
    p.write_text(json.dumps(d, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("ok")
