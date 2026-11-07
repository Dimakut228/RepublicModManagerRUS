"""One-shot English literal → tr(key) migration helper for the RML-RU fork.

Usage:
  python tools/apply_strings.py --audit <task>
  python tools/apply_strings.py --apply <task>

Tasks are defined in TASKS below. The script replaces exact double-quoted
literals with tr("<key>") calls, optionally limited to the bodies of the
listed functions, and appends the accepted mappings to Source/launcher/
locales/en.json (values are the original upstream strings).

Run from the repository root.
"""

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
LAUNCHER = ROOT / "Source" / "launcher"
EN_JSON = LAUNCHER / "locales" / "en.json"

# Chosen semantic keys; everything else gets an automatic slug key.
FIXED = {
    "REPUBLIC": "brand.line1",
    "MOD LOADER": "brand.line2",
    "Workers & Resources: Soviet Republic": "game.name",
    "Click to Copy": "tooltip.copy_build_sha",
    "Launch Workers & Resources": "dashboard.launch",
    "Launching and validating runtime…": "dashboard.launching",
    "Ready to launch with RML": "dashboard.ready",
    "Build Version: —": "dashboard.build_version.none",
    "Build Version: ": "dashboard.build_version",
    "Build Version: untested": "dashboard.build_version.untested",
    "Build SHA: —": "dashboard.build_sha.none",
    "Build SHA: ": "dashboard.build_sha",
    "NEEDS PATH": "dashboard.state.needs_path",
    "RECOGNISED": "dashboard.state.recognised",
    "UNTESTED": "dashboard.state.untested",
    "Choose executable": "dashboard.choose_exe",
    "No previous launch": "dashboard.no_launch",
    "Launch error": "dashboard.launch_error",
    "Loader OK": "dashboard.loader_ok",
    "Last bootstrap completed successfully": "dashboard.loader_ok_detail",
    "Last bootstrap reported a problem": "dashboard.loader_problem",
    "Discovered": "dashboard.metric.discovered",
    "%d Plugins / %d Workshop": "dashboard.metric.discovered_value",
    "Enabled": "dashboard.metric.enabled",
    "%d enabled Workshop item(s) may need updating": "dashboard.metric.enabled_detail_update",
    "%d Development project(s) active": "dashboard.metric.enabled_detail_dev",
    "Last launch": "dashboard.metric.last_launch",
    "Playlist": "dashboard.playlist.label",
    "Active mod configuration": "dashboard.playlist.title",
    "%d enabled • %s": "dashboard.playlist.detail",
    "Automatic dependency ordering": "dashboard.playlist.auto",
    "Saved / manual playlist order": "dashboard.playlist.manual",
    "Settings": "dashboard.settings.label",
    "Launch & support": "dashboard.settings.title",
    "Detailed Diagnostic Logging": "settings.verbose_diag",
    "Automatic crash archives": "settings.auto_crash",
    "Automatic ordering": "settings.auto_order",
    "Close after launch": "settings.close_after",
    "Hide while WRSR runs": "settings.auto_hide",
    "Ready": "status.ready",
    "Search filters this Mod list as you type": "status.search_hint",
}

# Literals left untouched everywhere (identifiers handled manually or
# intentionally kept untranslated).
EXCLUDED = {
    "Dashboard", "Mod Manager", "Diagnostics",
    "Launch overview", "Mods and playlists", "Logs",
    "RML %s  •  API %d",
    "Default",
    "Republic Mod Loader",
    "RepublicModLoaderWindowClass",
    "\\r\\n",
    "Locale: %v",
    # launcher-log / debug-only lines stay in English by design
    "DEEP TRACE pre-launch compatibility: detectedDLLs=%d enabledDLLs=%d sharedTargets=%d potentialConflicts=%d bridgeCandidates=%d overall=%s",
    "DEEP TRACE plugin[%02d] force=%t status=%q statusClass=%d required=%v optional=%v dependencies=%v conflicts=%v loadBefore=%v loadAfter=%v",
    "DEEP TRACE plugin[%02d] validation=%s",
    "DEEP TRACE target=%s plugin=%s name=%q loader=%s action=%s phase=%s evidence=%s managed=%t",
    "DEEP TRACE target=%s result=%s temperature=%s reason=%s interactions=%d",
    "DIAGNOSTIC Development enabled=%d local=%d",
    "DIAGNOSTIC plugin[%02d] id=%s name=%q version=%s sha256=%s loader=%s legacyAPI=%d priority=%d workshop=%s development=%s path=%s",
    "DIAGNOSTIC Symbol Cache schema=%d resolver=%d symbols=%d",
    "DIAGNOSTIC symbol id=%s rva=0x%X confidence=%s source=%s size=%d",
    "DIAGNOSTIC TRACE: enabled plugin/load-order snapshot BEGIN",
    "DIAGNOSTIC TRACE: enabled plugin/load-order snapshot END",
    "DIAGNOSTIC Workshop enabled=%d installed=%d",
    "COMPATIBILITY REPORT JSON",
    "LAUNCHER CONFIG JSON",
    "RUNTIME REPORT JSON",
    "EXCEPTION",
    "FAILED",
    "[PLUGIN DISABLED] id=%s path=%s",
    "[PLUGIN INVALID] id=%s path=%s reason=%s",
    "[PLUGIN QUEUED] id=%s loader=%s sha256=%s path=%s",
    "CreateWindowExW: %v",
    "GetMessageW: %v",
    "GetModuleHandleW: %v",
    "RegisterClassExW: %v",
    "launch profile: %s (%s); session=%s",
    "Workshop force update BEGIN: %s (%s)",
    "Workshop force update END: %s (%s) unsubscribed=%t filesDeleted=%t resubscribed=%t downloadRequested=%t downloading=%t message=%s",
    "Workshop inventory: installed=%d enabled=%d; Normal profile uses an explicit enabled-ID allow-list",
    "Development inventory: local=%d enabled=%d root=%s",
    "Development DLL payload: %s <- WIP %s (%s)",
    "Workshop hybrid plugin: %s <- Workshop %s (%s)",
    "Workshop hybrid plugin links discovered: %d",
    "launcher executable=%s runtime root=%s resolution=%s",
    "Exports",
    "manual support archive",
}

FUNCTION_SCOPES = {
    "dashboard_ui": [
        "drawSidebar", "drawStatusBar", "drawDashboardPage",
        "drawDashboardPlaylistCard", "drawDashboardSettingsCard",
        "drawBuildSHATooltip",
    ],
}


def slug_key(literal: str) -> str:
    words = re.findall(r"[A-Za-z0-9]+", literal)
    base = "_".join(words[:4]).lower()
    base = re.sub(r"[^a-z0-9_]", "", base)[:36].strip("_") or "text"
    return "t." + base


def skip_literal(literal: str) -> bool:
    if literal in EXCLUDED:
        return True
    if len(literal) < 2:
        return True
    # lowercase identifiers, paths, ids, enum states: no spaces, no uppercase
    if re.fullmatch(r"[a-z0-9_\-./:%]+", literal):
        return True
    if "://" in literal:
        return True
    if re.fullmatch(r"[\w\-.]+\.(json|exe|dll|log|png|ini|bin)", literal):
        return True
    if letter_count(literal) < 2:
        return True
    return False


def letter_count(text: str) -> int:
    return sum(1 for ch in text if ch.isalpha() and ch.isascii())


def function_span(src: str, name: str):
    start = src.find(f"func {name}(")
    if start < 0:
        raise SystemExit(f"function {name} not found")
    open_brace = src.index("{", start)
    depth = 0
    i = open_brace
    in_string = False
    while i < len(src):
        ch = src[i]
        if in_string:
            if ch == "\\":
                i += 2
                continue
            if ch == '"':
                in_string = False
        elif ch == '"':
            in_string = True
        elif ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                return start, i + 1
        i += 1
    raise SystemExit(f"function {name}: closing brace not found")


def replace_literals(src: str, mapping: dict):
    replacements = []
    for match in re.finditer(r'"((?:[^"\\\n]|\\.)*)"', src):
        literal = match.group(1)
        if skip_literal(literal):
            continue
        replacements.append((match.start(), match.end(), literal))
    out = src
    for start, end, literal in reversed(replacements):
        key = mapping.setdefault(literal, FIXED.get(literal) or slug_key(literal))
        out = out[:start] + f'tr("{key}")' + out[end:]
    return out, mapping


def scope_replace(src: str, scope: list, mapping: dict):
    for name in scope:
        a, b = function_span(src, name)
        body, mapping = replace_literals(src[a:b], mapping)
        src = src[:a] + body + src[b:]
    return src, mapping


def load_en():
    return json.loads(EN_JSON.read_text(encoding="utf-8-sig"))


def save_en(file):
    file["strings"] = dict(sorted(file["strings"].items()))
    EN_JSON.write_text(json.dumps(file, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def save_en_at(path, file):
    file["strings"] = dict(sorted(file["strings"].items()))
    path.write_text(json.dumps(file, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def main():
    args = sys.argv[1:]
    if len(args) < 2 or args[0] not in ("--audit", "--apply"):
        raise SystemExit(__doc__)
    mode, task = args[0], args[1]
    task_def = TASKS.get(task)
    if task_def is None:
        raise SystemExit(f"unknown task {task}")

    en = load_en()
    mapping = {}
    report = []
    for target in task_def["files"]:
        path = ROOT / target
        src = path.read_bytes().decode("utf-8")
        scope = task_def.get("scopes", {}).get(target)
        if scope:
            new_src, mapping = scope_replace(src, scope, mapping)
        else:
            new_src, mapping = replace_literals(src, mapping)
        if new_src != src:
            report.append(target)
        if mode == "--apply":
            path.write_bytes(new_src.encode("utf-8"))
    for literal, key in sorted(mapping.items()):
        existing = en["strings"].get(key)
        if existing is not None and existing != literal:
            print(f"CONFLICT {key}: {existing!r} != {literal!r}")
            continue
        en["strings"][key] = literal
    if mode == "--apply":
        save_en(en)
        ru_path = LAUNCHER / "locales" / "ru.json"
        ru = json.loads(ru_path.read_text(encoding="utf-8-sig"))
        for key, value in en["strings"].items():
            ru["strings"].setdefault(key, value)
        save_en_at(ru_path, ru)
    if mode == "--audit" and "--list" in args:
        for literal, key in sorted(mapping.items(), key=lambda kv: kv[1]):
            print(f"{key} = {literal!r}")
    print(f"{mode} task={task} files_changed={report} keys={len(mapping)}")


TASKS = {
    "dashboard_ui": {
        "files": ["Source/launcher/ui_pages_windows.go"],
        "scopes": {"Source/launcher/ui_pages_windows.go": FUNCTION_SCOPES["dashboard_ui"]},
    },
    "status_texts": {
        "files": [
            "Source/launcher/app_windows.go",
            "Source/launcher/main_windows.go",
        ],
    },
}

if __name__ == "__main__":
    main()
