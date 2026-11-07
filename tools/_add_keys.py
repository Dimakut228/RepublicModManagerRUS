import json
from pathlib import Path
base = Path("Source/launcher/locales")
addings = {
    "en": {
        "t.delete_prefix": "Delete \"",
        "t.imported_q_steam_subscription_failed": "Imported %q; Steam subscription requested for %d item(s), %d failed",
    },
    "ru": {},
}
for lang, entries in addings.items():
    p = base / f"{lang}.json"
    data = json.loads(p.read_text(encoding="utf-8-sig"))
    for k, v in entries.items():
        if v is None:
            continue
        data["strings"].setdefault(k, v)
    # enforce parity: mirror en keys missing in ru with en value, and vice versa
    other = json.loads((base / ("ru.json" if lang == "en" else "en.json")).read_text(encoding="utf-8-sig"))
    for k, v in data["strings"].items():
        other["strings"].setdefault(k, v)
    for k, v in other["strings"].items():
        data["strings"].setdefault(k, v)
    other["strings"] = dict(sorted(other["strings"].items()))
    data["strings"] = dict(sorted(data["strings"].items()))
    p.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    (base / ("ru.json" if lang == "en" else "en.json")).write_text(json.dumps(other, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("ok")
