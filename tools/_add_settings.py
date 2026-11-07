import json
from pathlib import Path
base = Path("Source/launcher/locales")
en_more = {
    "settings.modal.kicker": "SETTINGS",
    "settings.modal.title": "Launcher settings",
    "settings.language.label": "Interface language",
    "settings.language.auto": "Auto (Windows)",
    "settings.language.ru": "Русский",
    "settings.language.en": "English",
    "settings.language.cycle": "Auto (Windows)",
    "settings.autotranslate.label": "Translate mod descriptions automatically",
    "settings.clear.translations": "Clear translation cache",
    "settings.clear.catalog": "Clear catalog cache",
    "settings.close.btn": "Done",
    "settings.open.hint": "Settings",
    "t.settings_closed": "Closed the settings dialog",
    "t.language_selected_auto": "Interface language: Auto (Windows)",
    "t.language_selected_ru": "Interface language: Русский",
    "t.language_selected_en": "Interface language: English",
    "t.autotranslate_enabled": "Automatic translation of mod descriptions enabled",
    "t.autotranslate_disabled": "Automatic translation of mod descriptions disabled",
    "t.cleared_translations_cache": "Translation cache cleared",
    "t.cleared_catalog_cache": "Catalog cache cleared",
}
ru_more = {
    "settings.modal.kicker": "НАСТРОЙКИ",
    "settings.modal.title": "Настройки программы",
    "settings.language.label": "Язык интерфейса",
    "settings.language.auto": "Авто (Windows)",
    "settings.language.ru": "Русский",
    "settings.language.en": "English",
    "settings.language.cycle": "Авто (Windows)",
    "settings.autotranslate.label": "Переводить описания модов автоматически",
    "settings.clear.translations": "Очистить кэш переводов",
    "settings.clear.catalog": "Очистить кэш каталога",
    "settings.close.btn": "Готово",
    "settings.open.hint": "Настройки",
    "t.settings_closed": "Диалог настроек закрыт",
    "t.language_selected_auto": "Язык интерфейса: Авто (Windows)",
    "t.language_selected_ru": "Язык интерфейса: Русский",
    "t.language_selected_en": "Язык интерфейса: English",
    "t.autotranslate_enabled": "Автоперевод описаний модов включён",
    "t.autotranslate_disabled": "Автоперевод описаний модов отключён",
    "t.cleared_translations_cache": "Кэш переводов очищен",
    "t.cleared_catalog_cache": "Кэш каталога очищен",
}
for name, extra in (("en.json", en_more), ("ru.json", ru_more)):
    p = base / name
    d = json.loads(p.read_text(encoding="utf-8-sig"))
    for k, v in extra.items():
        d["strings"].setdefault(k, v)
    data = {"meta": d["meta"]}
    data["strings"] = dict(sorted(d["strings"].items()))
    p.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("settings keys added")
