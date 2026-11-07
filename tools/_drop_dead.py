import json
from pathlib import Path
base = Path("Source/launcher/locales")
dead = {"t.plugins_omitempty", "t.schemaversion", "t.workshopitems"}
for name in ("en.json", "ru.json"):
    p = base / name
    d = json.loads(p.read_text(encoding="utf-8-sig"))
    for k in list(d["strings"]):
        if k in dead:
            del d["strings"][k]
    d["strings"] = dict(sorted(d["strings"].items()))
    p.write_text(json.dumps(d, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("dead keys removed")
