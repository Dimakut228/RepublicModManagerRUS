import json
from pathlib import Path
base = Path("Source/launcher/locales")
en = {
    "dashboard.header_title": "Republic Mod manager",
    "dashboard.no_launch": "Never launched",
}
ru = {
    "dashboard.header_title": "Republic Mod manager",
    "dashboard.no_launch": "Не запускался",
}
for fname, extra in (("en.json", en), ("ru.json", ru)):
    p = base / fname
    d = json.loads(p.read_text(encoding="utf-8-sig"))
    for k, v in extra.items():
        d["strings"][k] = v
    d["strings"] = dict(sorted(d["strings"].items()))
    p.write_text(json.dumps(d, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("ok")
