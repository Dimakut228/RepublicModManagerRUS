import json
from pathlib import Path
from decimal import Decimal

base = Path("Source/launcher/locales")
for name in ("en.json", "ru.json"):
    p = base / name
    data = json.loads(p.read_text(encoding="utf-8-sig"))
    fixed = 0
    for key, value in list(data["strings"].items()):
        if "\\n" in value:
            data["strings"][key] = value.replace("\\n", "\n")
            fixed += 1
    # enforce parity: any keys missing on the other side mirror their value
    otherPath = base / ("ru.json" if name == "en.json" else "en.json")
    other = json.loads(otherPath.read_text(encoding="utf-8-sig"))
    for k, v in data["strings"].items():
        other["strings"].setdefault(k, v)
    for k, v in other["strings"].items():
        data["strings"].setdefault(k, v)
    for k, v in list(other["strings"].items()):
        if "\\n" in v:
            other["strings"][k] = v.replace("\\n", "\n")
    data["strings"] = dict(sorted(data["strings"].items()))
    other["strings"] = dict(sorted(other["strings"].items()))
    p.write_text(json.dumps(data, ensure_ascii=False, indent=2, sort_keys=False) + "\n", encoding="utf-8")
    otherPath.write_text(json.dumps(other, ensure_ascii=False, indent=2, sort_keys=False) + "\n", encoding="utf-8")
    print(name, "fixed", fixed)
