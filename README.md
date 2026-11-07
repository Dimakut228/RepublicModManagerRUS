# Republic Mod manager V 1.0 RUS — Неофициальный (Unofficial)

Лаунчер и менеджер DLL-плагинов для **Workers & Resources: Soviet Republic** (fork
Republic Mod Loader, GPL-3.0).

- Каталог Steam Workshop внутри программы: поиск, категории-фильтры, период, сортировки,
  пагинация, карточки модов.
- Подписка/отписка без захода на страницу Steam (Steam сам скачивает и обновляет файлы).
- Машинный перевод описаний модов на русский (включается/выключается в настройках; только
  текст описания отправляется на перевод, есть локальный кэш).
- Русский и английский интерфейс; язык меняется в настройках без перезапуска.
- Плейлисты, диагностика, поддержка Tesmio-плагинов — как в upstream Republic Mod Loader.

Скачивание файлов модов остаётся за Steam: лаунчер подписывается на элементы, Steam
качать контент. Никакого обхода Steam.

## Установка (portable ZIP)

1. Распаковать архив в любую папку.
2. Запустить `RepublicModManagerRUS.exe`.
3. Если игра не найдена — указать путь к `SOVIET64.exe`.
4. Включать плагины и моды, собирать плейлисты, запускать игру кнопкой.

Для(machine-перевода описаний) нужен интернет. Кэш хранится в `%LOCALAPPDATA%`.

## Сборка из исходников

Требуется: Go 1.23+, CMake 3.24+, Visual Studio 2022 C++ (x64), Python 3.

```powershell
cd Source
.\build-all.ps1
```

Результат: `Source\build\RepublicModManagerRUS.exe` и `Source\build\rml_runtime.dll`.

## Лицензия и атрибуция

- Исходный проект: Republic Mod Loader 1.0.1 (по коммиту `57e567e`) — GPL-3.0.
- Этот форк распространяется под GPL-3.0; исходники поставляются с каждым релизом.
- «Republic Mod manager V 1.0 RUS — Unofficial» — неаффелированный сторонний проект;
  не связан с 3DIVISION, Hooded Horse, Valve или автором Republic Mod Loader.
- Подробности: `NOTICE.md`.

---

# Republic Mod manager V 1.0 RUS — Unofficial (English)

Launcher and DLL plugin manager for **Workers & Resources: Soviet Republic** (a GPL-3.0
fork of Republic Mod Loader).

- In-app Steam Workshop catalog: search, category filters, period, sorting, pagination,
  mod cards.
- Subscribe/unsubscribe in one click; Steam itself downloads and updates files.
- Machine translation of mod descriptions into Russian (toggle in settings; only
  description text is sent, cached locally).
- Russian and English UI; switchable without restarting.
- Playlists, diagnostics, Tesmio compatibility — same as upstream.

Downloads stay in Steam: the launcher subscribes to items, Steam provides files.

## Install (portable ZIP)

1. Unpack anywhere.
2. Run `RepublicModManagerRUS.exe`.
3. Point it to `SOVIET64.exe` when prompted.
4. Enable plugins/mods, build playlists, launch.

Description translation needs internet access. Cache lives under `%LOCALAPPDATA%`.

## Build from source

Requirements: Go 1.23+, CMake 3.24+, Visual Studio 2022 C++ (x64), Python 3.

```powershell
cd Source
.\build-all.ps1
```

Outputs: `Source\build\RepublicModManagerRUS.exe` and `Source\build\rml_runtime.dll`.

## Licence and attribution

- Base project: Republic Mod Loader 1.0.1 (commit `57e567e`) — GPL-3.0.
- This fork is distributed under GPL-3.0; the source ships with every release.
- “Republic Mod manager V 1.0 RUS — Unofficial” is an unaffiliated third-party build,
  not endorsed by 3DIVISION, Hooded Horse, Valve, or the Republic Mod Loader author.
- Details: `NOTICE.md`.
