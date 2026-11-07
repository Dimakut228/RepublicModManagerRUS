import json, re, sys
from pathlib import Path
base = Path("Source/launcher")
en = json.loads((base/"locales/en.json").read_text(encoding="utf-8-sig"))["strings"]
ru = json.loads((base/"locales/ru.json").read_text(encoding="utf-8-sig"))["strings"]
problems = 0
for path in list(base.glob("*.go")):
    text = path.read_text(encoding="utf-8")
    for m in re.finditer(r'(?<![A-Za-z_])tr\("([^"]+)"\)', text):
        key = m.group(1)
        if key not in en:
            print(f"MISSING en {key} in {path.name}")
            problems += 1
        if key not in ru:
            print(f"MISSING ru {key} in {path.name}")
            problems += 1
print("missing:", problems)
sys.exit(1 if problems else 0)
