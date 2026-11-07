import json
from pathlib import Path

base = Path("Source/launcher/locales")
manual = {
    "app.title": "Republic Mod manager V 1.0 RUS — Unofficial",
    "nav.dashboard": "Dashboard",
    "nav.dashboard.hint": "Launch overview",
    "nav.mods": "Mod Manager",
    "nav.mods.hint": "Mods and playlists",
    "nav.diagnostics": "Diagnostics",
    "nav.diagnostics.hint": "Logs",
}
def load(name):
    d = json.loads((base / name).read_text(encoding="utf-8-sig"))
    return d
en = load("en.json"); ru = load("ru.json")
en["strings"].update(manual)
for k, v in en["strings"].items():
    ru["strings"].setdefault(k, v)
missing = set(ru["strings"]) - set(en["strings"])
for k in missing:
    en["strings"].setdefault(k, ru["strings"][k])
for name, file_ in (("en.json", en), ("ru.json", ru)):
    file_["strings"] = dict(sorted(file_["strings"].items()))
    (base / name).write_text(json.dumps(file_, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("en", len(en["strings"]), "ru", len(ru["strings"]), "backfilled", len(missing))
