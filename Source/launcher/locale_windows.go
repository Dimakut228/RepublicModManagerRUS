//go:build windows

package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"syscall"
)

// Embedded locale files. Locale selection is resolved once at startup from
// the persisted language setting ("auto", "en", "ru") and can be switched at
// runtime from the settings modal; tr() always reads the active bundle.

//go:embed locales/en.json locales/ru.json
var localeFS embed.FS

const (
	localeAuto    = "auto"
	localeEnglish = "en"
	localeRussian = "ru"
)

type localeMeta struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type localeFile struct {
	Meta    localeMeta        `json:"meta"`
	Strings map[string]string `json:"strings"`
}

var (
	localeBundles map[string]localeFile
	activeLocale  = localeEnglish
)

func parseLocale(data []byte) (localeFile, error) {
	var file localeFile
	if err := json.Unmarshal(data, &file); err != nil {
		return localeFile{}, err
	}
	if file.Strings == nil {
		file.Strings = map[string]string{}
	}
	return file, nil
}

func loadLocaleBundles() error {
	if localeBundles != nil {
		return nil
	}
	localeBundles = map[string]localeFile{}
	for _, code := range []string{localeEnglish, localeRussian} {
		data, err := localeFS.ReadFile("locales/" + code + ".json")
		if err != nil {
			return err
		}
		file, err := parseLocale(data)
		if err != nil {
			return err
		}
		localeBundles[code] = file
	}
	return nil
}

func nativeUILanguageCode() string {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultUILanguage")
	value, _, _ := proc.Call()
	langID := uint16(value)
	const russian = 0x0019
	if langID&0x3ff == russian {
		return localeRussian
	}
	return localeEnglish
}

func resolveLocaleCode(setting string) string {
	switch setting {
	case localeEnglish, localeRussian:
		return setting
	default:
		return nativeUILanguageCode()
	}
}

func initLocale(cfgLanguage string) error {
	if err := loadLocaleBundles(); err != nil {
		return err
	}
	switch cfgLanguage {
	case "", localeAuto, localeEnglish, localeRussian:
		activeLocale = resolveLocaleCode(cfgLanguage)
		return nil
	default:
		return fmt.Errorf("unknown language setting %q", cfgLanguage)
	}
}

func applyLanguage(code string) {
	switch code {
	case localeEnglish, localeRussian:
		activeLocale = code
	case localeAuto:
		activeLocale = resolveLocaleCode(localeAuto)
	}
}

func currentLocaleCode() string {
	return activeLocale
}

func availableLocaleCodes() []string {
	return []string{localeAuto, localeRussian, localeEnglish}
}

func localeString(code, key string) (string, bool) {
	file, ok := localeBundles[code]
	if !ok {
		return "", false
	}
	value, ok := file.Strings[key]
	return value, ok
}

func tr(key string, args ...any) string {
	_ = loadLocaleBundles()
	value, ok := localeString(activeLocale, key)
	if !ok {
		value, ok = localeString(localeEnglish, key)
	}
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(value, args...)
	}
	return value
}
