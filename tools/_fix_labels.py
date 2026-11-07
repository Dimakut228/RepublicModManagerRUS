import json
from pathlib import Path
base = Path("Source/launcher/locales")
en = {
    "t.mod_manager_title": "Mod Manager",
    "t.diagnostics_title": "Diagnostics",
    "t.unknown": "Unknown",
    "t.refresh": "Refresh",
    "t.scan_for_updates": "Check updates",
    "t.all_workshop": "All Workshop",
    "t.hide_disabled": "Hide disabled",
    "t.needs_attention": "Needs attention",
    "t.open_file_location": "Open folder",
    "t.force_update": "Reinstall",
    "update.state.current": "up to date",
    "update.state.update": "update available",
    "update.state.stale": "possibly stale",
    "update.state.downloading": "downloading",
    "update.state.missing": "missing files",
    "update.state.unavailable": "unavailable",
    "update.state.unknown": "unknown",
    "update.reason.unverified": "Could not check the Workshop revision",
    "update.reason.missing": "Item is no longer retrievable from Steam",
    "update.reason.downloading": "Steam is downloading this item",
    "update.reason.needs_update": "Steam reports an update is needed",
    "update.reason.no_installed": "No installed revision found",
    "update.reason.manifest": "Steam manifest differs from the installed one",
    "update.reason.server_newer_no_flag": "Server revision is newer, Steam is not flagging an update",
    "update.reason.server_newer": "Server revision is newer than the installed copy",
    "update.reason.current": "Installed revision matches the latest published update",
    "update.reason.current_unverified": "Installed locally; latest published revision could not be checked",
}
ru = {
    "t.mod_manager_title": "Менеджер модов",
    "t.diagnostics_title": "Диагностика",
    "t.unknown": "Неизвестно",
    "t.refresh": "Обновить",
    "t.scan_for_updates": "Обновления",
    "t.all_workshop": "Все моды",
    "t.hide_disabled": "Скрыть выкл.",
    "t.needs_attention": "Проблемные",
    "t.open_file_location": "Открыть папку",
    "t.force_update": "Переустановить",
    "update.state.current": "актуален",
    "update.state.update": "есть обновление",
    "update.state.stale": "возможно устарел",
    "update.state.downloading": "загружается",
    "update.state.missing": "нет файлов",
    "update.state.unavailable": "недоступен",
    "update.state.unknown": "неизвестно",
    "update.reason.unverified": "Не удалось проверить ревизию Workshop",
    "update.reason.missing": "Элемент больше не доступен в Steam",
    "update.reason.downloading": "Steam загружает этот элемент",
    "update.reason.needs_update": "Steam сообщает, что нужно обновление",
    "update.reason.no_installed": "Установленная ревизия не найдена",
    "update.reason.manifest": "Манифест Steam отличается от установленного",
    "update.reason.server_newer_no_flag": "Ревизия на сервере новее, но Steam не помечает обновление",
    "update.reason.server_newer": "Ревизия на сервере новее установленной копии",
    "update.reason.current": "Установленная ревизия совпадает с последней опубликованной",
    "update.reason.current_unverified": "Установлено локально; последнюю ревизию проверить не удалось",
}
for fname, extra in (("en.json", en), ("ru.json", ru)):
    p = base / fname
    d = json.loads(p.read_text(encoding="utf-8-sig"))
    for k, v in extra.items():
        d["strings"][k] = v
    d["strings"] = dict(sorted(d["strings"].items()))
    p.write_text(json.dumps(d, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("ok")
