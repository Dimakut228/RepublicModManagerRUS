//go:build windows

package main

import (
	"os"
	"path/filepath"
	"unsafe"
)

// Settings support: language switching (instant, persisted), cache clearing.

func translationCacheRoot(knowledgeRoot string) string {
	return filepath.Join(knowledgeRoot, "translations", "v1")
}

func catalogCacheRoot(knowledgeRoot string) string {
	return filepath.Join(knowledgeRoot, "catalog", "v1")
}

func translationCachePath(knowledgeRoot string) string {
	return filepath.Join(translationCacheRoot(knowledgeRoot), string(currentLocaleCode()))
}

func clearTranslationCache(knowledgeRoot string) {
	_ = os.RemoveAll(translationCacheRoot(knowledgeRoot))
}

func clearCatalogCache(knowledgeRoot string) {
	_ = os.RemoveAll(catalogCacheRoot(knowledgeRoot))
}

// applyLanguageSetting persists the language choice, re-resolves the active
// locale and refreshes the window title. Invalid codes are ignored.
func applyLanguageSetting(app *application, code string) {
	switch code {
	case localeAuto, localeEnglish, localeRussian:
	default:
		return
	}
	app.cfg.Language = code
	_ = initLocale(code)
	_ = saveConfig(app.configPath, app.cfg)
	app.refreshWindowTitle()
}

func (a *application) refreshWindowTitle() {
	if a.hwnd == 0 {
		return
	}
	title := utf16Ptr(tr("app.title"))
	procSetWindowTextW.Call(a.hwnd, uintptr(unsafe.Pointer(title)))
}
