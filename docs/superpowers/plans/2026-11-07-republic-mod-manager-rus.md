# Republic Mod manager V 1.0 RUS — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Форк Republic Mod Loader: русский и английский интерфейс, каталог Steam Workshop с фильтрами и подпиской, автоперевод описаний через бесплатный Google-endpoint, собственный GPL-3.0 релиз на GitHub (portable ZIP).

**Architecture:** Всё добавляется в Go-лаунчер (`Source/launcher`, `package main`, только stdlib) и в сценарии сборки. Рантайм `Source/runtime`, инъекция и Tesmio-совместимость не трогаются. Локали — встроенные JSON (`go:embed`), каталог и перевод — новые файлы того же пакета.

**Tech Stack:** Go 1.23 windows/amd64, Win32/GDI canvas upstream, Steam Community HTML + `ISteamRemoteStorage/GetPublishedFileDetails`, `translate.googleapis.com`, CMake + VS2022 для рантайма, Python 3 для PE-ресурсов.

**Spec:** `docs/superpowers/specs/2026-11-07-rml-ru-design.md`

## Global Constraints

- Проект в `rml-ru/`; база — upstream коммит `57e567e5cccaa3c38aee7e9d599d2443d2460b10`, лицензия GPL-3.0.
- Зависимости: только stdlib Go. `go.mod` не расширяется.
- Менять можно только `Source/launcher`, `Source/docs`, `Source/assets`, `Source/build-*.ps1`, `Source/VERSION`, корневые README/NOTICE/CHANGELOG. `Source/runtime`, `Source/include` не трогать.
- Внутренние имена не менять: `rml_runtime.dll`, папка `rml/`, путь `persistentKnowledgeRoot`, константа `playlistExchangeFormat` (`"Republic Mod Loader Playlist"`), схема `config.json` (только расширение новыми полями).
- Название: `Republic Mod manager V 1.0 RUS`; EXE `RepublicModManagerRUS.exe`; версия `1.0.0` в `Source/VERSION`; пометка «Unofficial / Неофициальный» в заголовке окна и README.
- Анти-цели из spec действуют: без скрытых сетевых запросов, без обхода Steam, без присвоения upstream-бренда.
- Сеть: таймаут 10 с; ретраи только 5xx/таймаут, максимум 2; 429/403 — пауза 10 мин без повторов.
- Кэш каталога: TTL списка 6 ч, деталей 24 ч, превью 30 дней; лимит 200 МБ, LRU-чистка; писать в knowledge root (`catalog/v1` со страницами, деталями и `thumbs`, `translations/v1`).

## Review Focus

1. **Офлайн или 429 при открытии каталога.** Ожидается: последний кэш + баннер, без зависания. → Task 11: `TestCatalogRetryOn5xx` (httptest: ответ 500, затем 200; клиент повторяет).
2. **Описание мода с URL/BBCode/смайликами.** Ожидается: токены восстанавливаются после перевода, текст не портится. → Task 10: `TestMaskRoundTripURLsAndBBCode`.
3. **Ошибка перевода (таймаут/429).** Ожидается: оригинал, понятное сообщение, пауза 10 минут. → Task 10: `TestTranslatePauseAfter429`.
4. **Старый `config.json` без новых полей.** Ожидается: дефолты, старые поля сохраняются. → Task 4: `TestLoadConfigMigrationOldUpstream`.
5. **Длинные русские подписи при 150% DPI.** Ожидается: без наездов на тумблеры/иконки. → Task 8: ручной чеклист.

Каждая задача заканчивается: тесты `go test ./...` (запуск из `Source/launcher`) — PASS, коммит.

---

### Task 1: Bootstrap репозитория

**Files:** Create `Source/.gitignore` (`build/`), проектное содержимое — весь снапшот upstream.

- [ ] Шаг 1: сохранить `rml-ru/docs`, клонировать upstream в `rml-ru-tmp`, заменить им `rml-ru`, вернуть `docs`.

```powershell
Move-Item rml-ru\docs ..\rml-ru-keep-docs
git clone https://github.com/Ultimate-Universe/WRSR-RepublicModLoader.git rml-ru-tmp
Remove-Item rml-ru -Recurse -Force
Move-Item rml-ru-tmp rml-ru
Move-Item ..\rml-ru-keep-docs rml-ru\docs
```

- [ ] Шаг 2: `git -C rml-ru remote add upstream https://github.com/Ultimate-Universe/WRSR-RepublicModLoader.git`; добавить файл `Source/.gitignore` со строкой `build/`.
- [ ] Шаг 3: commit `docs: add RML-RU design and plan` (папка docs) и `chore: add build gitignore`.
- [ ] Шаг 4: проверка — `git -C rml-ru log --oneline -3` показывает 2 новых коммита поверх `57e567e`.

### Task 2: Toolchain + baseline build

**ASK USER:** установка Go и CMake через winget — только с явного согласия пользователя (уже предупреждали в процессе дизайна).

- [ ] Шаг 1: `go version` — если нет: `winget install --id GoLang.Go --silent` (≥ 1.23). `cmake --version` — если нет: `winget install --id Kitware.CMake --silent` (≥ 3.24).
- [ ] Шаг 2: VS2022 C++ уже установлен (проверено ранее, директория `2022` есть). Python 3.12 есть в PATH.
- [ ] Шаг 3: baseline: `cd Source; .\build-all.ps1`. Ожидание: созданы `build\RepublicModLoader.exe` (до переименования) и `build\rml_runtime.dll`, скрипт завершён без ошибок.
- [ ] Шаг 4: commit, если появились новые файлы игнора/правок; иначе отметить в плане, что baseline зелёный.

### Task 3: Локализация — ядро

**Files:**
- Create: `Source/launcher/locales/en.json`, `Source/launcher/locales/ru.json` — тела вида `{"meta":{"code":"en","name":"English","version":1},"strings":{}}`
- Create: `Source/launcher/locale_windows.go`
- Create: `Source/launcher/locale_windows_test.go`

**Interfaces (Produces):**

```go
//go:embed locales/en.json locales/ru.json
var localeFS embed.FS

type localeMeta struct { Code string; Name string; Version int }
type localeFile struct {
    Meta    localeMeta        `json:"meta"`
    Strings map[string]string `json:"strings"`
}

func initLocale(cfgLanguage string) error // "auto"|"en"|"ru"; auto: GetUserDefaultUILanguage 0x0419 → "ru", иначе "en"
func applyLanguage(code string)           // переключение текущей локали в памяти; конфиг не пишет
func currentLocaleCode() string           // "en" | "ru"
func tr(key string, args ...any) string   // fmt.Sprintf(trVal(key)); нет ключа → en; нет в en → сам key
func availableLocaleCodes() []string      // ["auto","en","ru"]
```

- [ ] Шаг 1: тесты (имя → проверка):
  - `TestLocaleParityKeys` — набор ключей en.json == ru.json;
  - `TestLocaleFormatVerbs` — число и тип глаголов (`%d`, `%s`, `%v`) в en/ru совпадают;
  - `TestLocaleFallbackUnknownKey` — `tr("missing.key")` возвращает `"missing.key"`;
  - `TestTrFormatsArgs` — `tr("key.with.verb", 3)` == форматированное значение.
  Запуск: `go test ./...` из `Source/launcher` → FAIL (нет `tr`).
- [ ] Шаг 2: реализовать `locale_windows.go`.
- [ ] Шаг 3: `go test ./...` → PASS.
- [ ] Шаг 4: commit `feat: locale engine core`.

### Task 4: Конфигурация — новые поля

**Files:** Modify `Source/launcher/model.go` (структура `AppConfig` ~строка 76, `defaultConfig()` ~строка 213), Create `Source/launcher/config_rmlrus_test.go`.

Новые поля `AppConfig` (точно с именами и omitempty):

```go
Language                 string `json:"language,omitempty"`                 // auto|ru|en
AutoTranslate            bool   `json:"autoTranslateDescriptions,omitempty"`
TranslationNoticeShown   bool   `json:"translationNoticeShown,omitempty"`
CatalogSort              string `json:"catalogSort,omitempty"`
CatalogPeriod            string `json:"catalogPeriod,omitempty"`
CatalogShowIncompatible  bool   `json:"catalogShowIncompatible,omitempty"`
```

`defaultConfig()` добавляет: `Language:"auto"`, `AutoTranslate:true`, `CatalogSort:"trend"`; остальные — нулевые.

Хук: `loadConfig(path)` после разбора — если файл старый (без `language`), применить `initLocale("auto")`; иначе `initLocale(cfg.Language)`. `saveConfig` не меняет старые поля.

- [ ] Шаг 1: тесты:
  - `TestLoadConfigMigrationOldUpstream` — JSON без новых полей → Language "auto", AutoTranslate true, старые поля читаются;
  - `TestSaveConfigRoundTripNewFields` — новые поля выживают при сохранении/повторном чтении;
  - `TestLanguageAutoDetection` — подмена конфига с `language:"ru"` инициализирует локаль `"ru"` (проверка через `currentLocaleCode()`).
- [ ] Шаг 2: `go test ./...` FAIL → реализовать → PASS.
- [ ] Шаг 3: commit `feat: config language and catalog fields`.

### Task 5: Локализация дашборда и навигации

**Files:** Modify `Source/launcher/ui_pages_windows.go` (верхняя навигация ~строка 280, `drawDashboard*`: плейлист-карта ~493, настройки-карта ~526–551, ряд запуска ~466–489), `app_windows.go` (статусные строки, привязанные к этим экранам, например ~1275), `main_windows.go` (title ~405, `showFatalMessage` ~463).

Название окна: значение локали `app.title` = `Republic Mod manager V 1.0 RUS — Unofficial`. После `CreateWindowExW` вызвать `procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(tr("app.title")))))`; если прок `SetWindowTextW` отсутствует в реестре `user32.NewProc` — добавить. При смене языка в настройках заголовок обновить тем же способом.

Ключи этого экрана (формат конечных значений берём из текущих английских строк upstream, чтобы не менять смысл):
`nav.dashboard`, `nav.mods`, `nav.playlists`, `nav.diagnostics`, `app.title`, `app.boot.ready`, `dashboard.metric.plugins`, `dashboard.metric.enabled`, `dashboard.metric.last_launch`, `dashboard.launch`, `dashboard.launching`, `dashboard.ready_state`, `playlist.card.label`, `playlist.active_config`, `dashboard.settings.label`, `settings.verbose_diag`, `settings.auto_crash`, `settings.auto_order`, `settings.close_after`, `settings.auto_hide`.

Плейсхолдеры динамических строк: `%d`/`%s` в значении локали; вызов как `tr("dashboard.metric.enabled", n, m)`.

- [ ] Шаг 1: lint-тест в `locale_windows_test.go`: `TestNoHardcodedDashboardStrings` — regexp по `ui_pages_windows.go`: `c.label\("` , `c.button\(",` , `c.metricCard\("` с литералом `[A-Za-z]` внутри первой строки-аргумента → есть совпадение, значит FAIL. Allowlist литералов: `""` (пустые строки).
- [ ] Шаг 2: конверсия вызовов на `tr(...)`; значения en.json заполняются текущими upstream-строками.
- [ ] Шаг 3: `go test ./...` PASS; `.\Source\build-launcher.ps1`; ручной запуск и осмотр дашборда.
- [ ] Шаг 4: commit `feat: localize dashboard and nav`.

### Task 6: Локализация списков модов и карточек

**Files:** Modify `Source/launcher/mod_list_windows.go`, `ui_pages_windows.go` (плейлисты/мастерская/разработка: списки и карточки ~1067–1585, `drawModSearchBox` ~564), `app_windows.go` — статусы страницы модов.

Ключи: `list.plugins`, `list.workshop`, `list.development`, `list.search_placeholder`, `list.empty`, `plugin.details`, `plugin.description`, `plugin.settings`, `workshop.details`, `workshop.update_status`, `workshop.embedded_dlls`, `dev.details`, `dev.embedded_dlls`, `list.col.name`, `list.col.id`, `list.col.on`, `sort.arrow`, суффиксы сортировки.

Названия модов (динамические данные) не трогаем.

- [ ] Шаг 1: расширить `TestNoHardcodedDashboardStrings` на эту область (regexp-список файлов в одном месте).
- [ ] Шаг 2: конверсия; en.json дополняется.
- [ ] Шаг 3: `go test ./...` PASS; `build-launcher.ps1`; вручную страницы.
- [ ] Шаг 4: commit `feat: localize mod lists`.

### Task 7: Локализация плейлистов, диагностики, предупреждений

**Files:** Modify `Source/launcher/playlists_windows.go`, `ui_pages_windows.go` (плейлисты + диагностика), `workshop_windows.go` (строки `LocalUnsupportedReasons` — собрала по формуле в spec §9: "No local Workshop metadata", "No mod name", "No preview image", "Workshop folder is empty", "Workshop page is unavailable", "Workshop item appears delisted or inaccessible", "Workshop item has been removed from the community"), `compatibility.go`, `workshop_update_windows.go`.

Вводим:

```go
var reasonKeyByText = map[string]string{ /* полный список строк → локаль-ключ */ }
func localizeReason(s string) string             // hit → tr(key); промах → исходная строка
func localizeWarnings(ss []string) []string      // map по каждому элементу
```

При правке все найденные предупреждения (через `grep 'reasons = append'` и `'Warnings = append'`) добавлять в таблицу с точной upstream-строкой, не меняя upstream-логику.

- [ ] Шаг 1: тест `TestLocalizeReasonKnownStrings` — известные 8 reason-строк переводятся, строка `"whatever"` проходит нетронута.
- [ ] Шаг 2: конверсия + примерные ключи.
- [ ] Шаг 3: `go test ./...` PASS.
- [ ] Шаг 4: commit `feat: localize playlists and warnings`.

### Task 8: Полный ru.json + визуальный проход

- [ ] Шаг 1: ru.json — заполняются все ключи до паритета (конфликтов не должно быть). Патч (если нужно) минимальных ширин кнопок на основе `DrawTextW` с `DT_CALCRECT` для измерения текста: мин ширина кнопки = max(машинная, textWidth + padding).
- [ ] Шаг 2: ручной DPI чеклист: масштабы 100% / 125% / 150% (Свойства отображения). На драмборде и списке мастерской: не наезжает текст, кнопка запуска читается, модалки откры? все строки на русском.
- [ ] Шаг 3: commit `feat: complete ru locale`.

### Task 9: Окно настроек

**Files:** Modify `Source/launcher/app_windows.go` (state: `settingsModal bool`, хендлеры `settings-*`, открываемтые из драмборда), `ui_pages_windows.go` (`drawSettingsModal`).

Содержимое:
- язык: выпадашка Авто/Русский/English → `applyLanguage`, `saveConfig` с `Language`, затем обновить заголовок окна;
- «Переводить описания автоматически» toggle → `AutoTranslate` + `saveConfig`;
- «Очистить кэш переводов» → удалить `translations/v1` (реализация удаления — здесь, кэш создаёт Task 10);
- «Очистить кэш каталога» → удалить `catalog/v1` (журнал превью тоже; каталог создаёт Task 11);
- «Закрыть».

- [ ] Шаг 1: unit `TestLanguageSwitchPersistsConfig` — переключение меняет `cfg.Language` и локаль немедленно.
- [ ] Шаг 2: реализация.
- [ ] Шаг 3: ручной: язык применяется к заголовку окна и навигации без перезапуска.
- [ ] Шаг 4: commit `feat: settings modal`.

### Task 10: Перевод описаний — движок

**Files:** Create `Source/launcher/translate_windows.go`, `Source/launcher/translate_windows_test.go`.

**Interfaces (Produces):**
```go
const translateMaxChunk = 3800

type translator struct {
    client        *http.Client // укоплекс testable
    knowledgeRoot string
    pausedUntil   time.Time
}
func (t *translator) translate(raw string) (string, error) // URL endpoint фиксирован в src
func normalizeSource(raw string) string                   // trims, CRLF→LF; ключ хеша sha256
func translationCachePath(knowledgeRoot string) string    // .../translations/v1/ru/
func translationGet(t, source string) (string, bool)
func translationPut(t, source, translated string)
func maskMarkup(raw string) (masked string, tokens []string)
func unmaskMarkup(translated string, tokens []string) (string, error)
```

Разметка: URL `https?://\S+`, BBCode `[br] [*] [b] …`, заголовки `[h2]`, числа формат — токены `⟦0⟧`.. . Разбить по абзацам ≤ 3800 символов на запрос. При 429/403/таймауте — `pausedUntil = now+10m`, вернуть ошибку; UI в это время показывает оригинал и статус «Перевод недоступен, повтор позже». Ошибка перевода не может обрушить UI.

- [ ] Тесты:
  - `TestMaskRoundTripURLsAndBBCode` — `[b]Hi[/b] go to https://example.com/a?b=1` → после round-trip URL и теги целы;
  - `TestTranslatePauseAfter429` — httptest сервер отвечает 429; второй вызвать отказывается до 10 мин;
  - `TestTranslateChunkLargeText` — 9000 символов → 3 чанка;
  - `TestTranslationCacheRoundTrip` — get/put по tooltip memo.
- [ ] Ручной: `POST` на реальный Google endpoint с коротким текстом — перевод возвращается (в Blocked-sостояние не попадаем).
- [ ] commit `feat: translation engine`.

### Task 11: Каталог — сеть, parse, кэш

**Files:** Create `Source/launcher/catalog_windows.go`, `Source/launcher/catalog_windows_test.go`, `Source/launcher/testdata/browse_p1.html` (сохраненная страница Steam: 30 items), фиксures `browse_p2.html`, `details_body.json`.

**Interfaces (Produces):**
```go
type catalogQuery struct {
    Page int
    Search string
    Sort string         // ""|"trend"|"mostrecent"|"toprated"
    Period string       // ""|"7"|"30"|"90"|"180"|"365"
    Tags []string
    ShowIncompatible bool
}
type catalogItem struct {
    ID string; Title string; URL string
    TimeCreated int64; TimeUpdated int64
    Subscriptions int; Favorited int; Views int
    FileSize int64
    PreviewURL string
    Tags []string
    Description string
    Visibility int; Banned bool
}
type catalogClient struct {
    HTTPClient *http.Client
    KnowledgeRoot string
}
func (c *catalogClient) fetchPage(q catalogQuery) (ids []string, total int, err error)
func (c *catalogClient) fetchDetails(ids []string) (map[string]catalogItem, error)
func catalogCachePath(knowledgeRoot string) string
func sortParameter(sortCode string) string
```

Browse URL параметры:
`https://steamcommunity.com/workshop/browse/?appid=784150&p={page}&numperpage=30&section=readytouseitems&browsesort={sort}&days={days}&searchtext={searchtext}&requiredtags[]={tag}` — повторение параметра `requiredtags[]` для каждой метки. `ShowIncompatible=true` → Если параметр фактически работает, применить; иначе фильтр по полю item → примечание (определится по живой странице, mark fixed в коде).

Parse: собственных regexp по блокам `workshopItem` (основан на сохранённом HTML в testdata). Сторонние HTML-парсеры запрещены. Если распознано 0 → err с ключом локали `catalog.parse.failed` (не пустой список).

Details API: `POST https://api.steampowered.com/ISteamRemoteStorage/GetPublishedFileDetails/v1/` — тело `itemcount=N&publishedfileids[0]=<id>...`, пакет ≤ 100; без ключа. Нормализация item: description Strip HTMLucs, description PLUS `plain` формат. `FileSize` — `file_size`.

Кэш: файлы `catalog/v1/page-<hash>.json` (query-hash), TTL 6 ч; `catalog/v1/details-<id>.json`, TTL 24 ч; `catalog/v1/thumbs/<id>.jpg`, TTL 30 дней; общий лимит 200 МБ LRU.

- [ ] Тестовые данные: живыми запросами сохранить `browse_p1.html`, `browse_p2.html` и ответ `GetPublishedFileDetails` для 2–3 ID как `details_body.json`.
- [ ] Тесты:
  - `TestCatalogParseFixture` — что распознаны 30 items, все ID начинаются с цифр;
  - `TestCatalogRetryOn5xx` — client inject: first `HTTP 500` response, second → 200; реакция — повтор и успех;
  - `TestCatalogDetailsEnsure100` — список 120 ID → 2 POST-пакета.
- [ ] Ручной: запустить debug вариант с сохранением в кэш, посмотреть 2 страницы — ID и headers сходятся со Steam.
- [ ] commit `feat: catalog network layer`.

### Task 12: Каталог — UI и подписка

**Files:** Create `Source/launcher/catalog_ui_windows.go`; Modify `Source/launcher/ui_pages_windows.go` (раздел вкладка `Установленные | Каталог Steam` на странице Workshop; drawCatalogList, drawCatalogDetail, фильтр-чат, пагинация), `app_windows.go` (state: `catalogMode`, `catalogItems`, `catalogBusy`, `catalogPage`, `catalogTotal`, выбор карточек), `main_windows.go` (клик и Tab/скролл переходы под каталогом).

State:
- `catalogMode bool` — false = «Установленные» (существующий список), true = «Каталог Steam»;
- список: thumbnail (использовать существующий загрузчик GDI+ `loadGDIPlusImage` на файлах из `catalog/v1/thumbs`), название, автор, подписки, размер, бейдж «Установлен»;
- фильтры: строка поиска, категории (мультивыбор), период (меню), сортировка (меню), сохраняются в конфиг (`CatalogSort`, `CatalogPeriod`, `CatalogShowIncompatible`), кнопка «Сбросить»;
- карточка: большое превью, название, автор, дата создания/обновления, размер, статус (не установлен / подписан / установлен), кнопка Подписаться/Отписаться — вызов существующей `subscribeWorkshopItems(gamePath, []string{id})`; описание показывается как Task 13 подключит перевод;
- пагинация: «Назад», «Вперёд», счётчик «страница X из Y».

Состояния UI: busy (спиннер "Загрузка каталога…"), empty result (пусто + совет), offline (баннер со старым кэшем), parse error (плохая причина).

- [ ] Тест unit: `TestCatalogPageIncrement` — навигация не выходит за последний page (по `total`) и не меньше 1.
- [ ] Тест: попытка подписки при `gamePath == ""` → понятная ошибка, состояние не меняется.
- [ ] Ручной на этой машине (игра `E:\SteamLibrary...`, 40 установленных модов): подписка нового мода через каталог → Steam скачивает → статус «Установлен»; отписка в мастерской удаляет мод из списка.
- [ ] Ручной DPI 100/125/150.
- [ ] commit `feat: catalog UI and subscribe`.

### Task 13: Перевод в карточках + одноразовое уведомление

**Files:** Modify `Source/launcher/ui_pages_windows.go` (в `drawCatalogDetail` и в существующей карточке мастерской: блоки «Перевод» / «Оригинал», кнопки «Перевести заново» / «Показать оригинал»), `Source/launcher/app_windows.go` (запуск goroutine перевода, состояние текущей карточки, флаг notice).

Behaviour:
- если `AutoTranslate` == false или `currentLocaleCode()` == "en" → показывается только оригинал, кнопки перевода прячутся;
- при открытии карточки: если кэш есть, перевод показывается сразу; иначе асинхронный запрос, без блокировки UI;
- переключатель «Показать оригинал»/«Показать перевод» и кнопка «Перевести заново»;
- при первом открытии каталога, если `TranslationNoticeShown` == false: баннер `catalog.translate_notice` = «Описания переводятся через Google; отключить можно в настройках» и флаг сохранить в конфиг.

- [ ] Ручной чеклист из spec §17: описание переводится, повторное открытие — кэш (проверка задержки < 100 мс), выключение перевода.
- [ ] commit `feat: translate descriptions in detail cards`.

### Task 14: Брендинг, версия, сборка, доки

**Files:** Modify `Source/VERSION` → `1.0.0`; Modify `Source/build-launcher.ps1` (строка 6: `$Out = Join-Path $Build 'RepublicModManagerRUS.exe'`; pe_resources: `--icon Source/assets/RepublicModManagerRUS.ico`, `--file-description 'Republic Mod manager V 1.0 RUS'`, `--original-filename 'RepublicModManagerRUS.exe'`); Create `Source/assets/RepublicModManagerRUS.ico` — сгенерировать нейтральный 256×256 PNG (например, буква R на фоне шестерёнки; сделать инструментом генерации изображений), конвертировать в ICO набором размеров 16/32/48/64/128/256 (Pillow, `pip install pillow` с согласия); Replace `Source/assets/brand_logo.png` своим нейтральным логотипом (сохранить ту же пропорцию/ширину, чтобы код загрузки не менялся); Modify `README.md`, Create `NOTICE.md`, Create/Update `CHANGELOG.md`.

NOTICE.md:
```
Based on Republic Mod Loader 1.0.1 by Ultimate-Universe (GPL-3.0)
Upstream: https://github.com/Ultimate-Universe/WRSR-RepublicModLoader
Upstream commit: 57e567e5cccaa3c38aee7e9d599d2443d2460b10
Unofficial fork; not affiliated with 3DIVISION, Hooded Horse, Valve, or the original author.
```

Внутренние имена и строки остаются: `playlistExchangeFormat`, `rml_runtime.dll`, `rml/`, `persistentKnowledgeRoot`.

- [ ] Шаг 1: `build-all.ps1` → `build\RepublicModManagerRUS.exe` (≥ 1 МБ) + `build\rml_runtime.dll`.
- [ ] Шаг 2: PE-свойства EXE (правый клик → Свойства → Подробно): File description = `Republic Mod manager V 1.0 RUS`, Product = название форка, значок = своя иконка.
- [ ] Шаг 3: заголовок окна = `Republic Mod manager V 1.0 RUS — Unofficial`; панель задач и окно корректны; запуск игры с 40 модами работает (инъекция не затронута).
- [ ] Шаг 4: commit `chore: rebrand to Republic Mod manager V 1.0 RUS`.

### Task 15: Релиз — упаковка + GitHub

**ASK USER:** URL GitHub-репозитория (или разрешение создать через `gh repo create` при наличии `gh auth`).

- [ ] Шаг 1: полный `Source\build-all.ps1` — финальная сборка.
- [ ] Шаг 2: чеклист spec §17 вручную: каталог, поиск, категории, подписка, отписка, перевод, плейлисты, DPI, офлайн, 40 установленных модов, запуск игры.
- [ ] Шаг 3: пакет ZIP — `RepublicModManagerRUS-1.0.0.zip` с содержимым:
  ```
  RepublicModManagerRUS.exe
  rml_runtime.dll
  rml/config.json
  rml/assets/<новые изображения>
  ```
  (без каталогов `rml/state`, `logs`, `Exports`).
- [ ] Шаг 4: `git -C rml-ru add -A; git -C rml-ru tag v1.0.0`; `git remote add origin <URL>`; push: `git push -u origin main --tags`.
- [ ] Шаг 5: GitHub Release по тегу `v1.0.0`; тело описания: «based on Republic Mod Loader 1.0.1 (GPL-3.0), upstream commit `57e567e`» + ссылка на README + ZIP.
- [ ] Шаг 6: commit `chore: release 1.0.0` (README/CHANGELOG необходимые правки post-release, чтобы они вошли в релиз).
