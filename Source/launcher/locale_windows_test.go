package main

import (
	"path"
	"reflect"
	"strings"
	"testing"
)

func localeVerbCounts(value string) map[string]int {
	counts := map[string]int{}
	for i := 0; i < len(value)-1; i++ {
		if value[i] != '%' {
			continue
		}
		c := value[i+1]
		switch c {
		case 'd', 's', 'v', 'f':
			counts[string(c)]++
			i++
		case '%':
			i++
		}
	}
	return counts
}

func TestLocaleParityKeys(t *testing.T) {
	en := mustLocale(t, "locales/en.json")
	ru := mustLocale(t, "locales/ru.json")
	for key := range en.Strings {
		if _, ok := ru.Strings[key]; !ok {
			t.Errorf("ru locale is missing key %q", key)
		}
	}
	for key := range ru.Strings {
		if _, ok := en.Strings[key]; !ok {
			t.Errorf("en locale is missing key %q", key)
		}
	}
}

func TestLocaleFormatVerbs(t *testing.T) {
	en := mustLocale(t, "locales/en.json")
	ru := mustLocale(t, "locales/ru.json")
	for key, enValue := range en.Strings {
		want := localeVerbCounts(enValue)
		got := localeVerbCounts(ru.Strings[key])
		for verb, n := range want {
			if got[verb] != n {
				t.Errorf("key %q: ru has %d %%%s verb(s), en has %d", key, got[verb], verb, n)
			}
		}
		delete(got, "")
		if len(got) > len(want) {
			for verb := range got {
				if verb != "" && want[verb] == 0 {
					t.Errorf("key %q: ru has extra %%%s verb", key, verb)
				}
			}
		}
	}
}

func TestLocaleFallbackUnknownKey(t *testing.T) {
	if got := tr("totally.missing.key"); got != "totally.missing.key" {
		t.Errorf("tr(missing) = %q, want the key itself", got)
	}
}

func TestTrFormatsArgs(t *testing.T) {
	if got := tr("test.verb.count", 3); got != "3 items" {
		t.Errorf("tr(test.verb.count, 3) = %q", got)
	}
	if got := tr("test.verb.name", "abc"); got != "name: abc" {
		t.Errorf("tr(test.verb.name, abc) = %q", got)
	}
}

func TestFallbackToEnglishValue(t *testing.T) {
	// A key that exists in en but is absent in the active locale falls back to en.
	en := mustLocale(t, "locales/en.json")
	key := "test.verb.count"
	val, ok := en.Strings[key]
	if !ok {
		t.Fatalf("en locale lacks %q", key)
	}
	applyLanguage("ru")
	defer applyLanguage("en")
	removed, had := localeBundles["ru"].Strings[key]
	delete(localeBundles["ru"].Strings, key)
	defer func() {
		if had {
			localeBundles["ru"].Strings[key] = removed
		}
	}()
	if got := tr(key); got != val {
		t.Errorf("fallback tr(%q) = %q, want %q", key, got, val)
	}
}

func TestApplyLanguageSwitchesLocale(t *testing.T) {
	defer applyLanguage("en")
	applyLanguage("ru")
	if currentLocaleCode() != "ru" {
		t.Errorf("currentLocaleCode() = %q, want ru", currentLocaleCode())
	}
	applyLanguage("en")
	if currentLocaleCode() != "en" {
		t.Errorf("currentLocaleCode() = %q, want en", currentLocaleCode())
	}
}

func TestAvailableLocaleCodes(t *testing.T) {
	want := []string{"auto", "en", "ru"}
	if got := availableLocaleCodes(); !reflect.DeepEqual(got, want) {
		t.Errorf("availableLocaleCodes() = %v, want %v", got, want)
	}
}

func TestInitLocaleResolvesExplicitLanguage(t *testing.T) {
	defer applyLanguage("en")
	if err := initLocale("ru"); err != nil {
		t.Fatalf("initLocale(ru): %v", err)
	}
	if got := currentLocaleCode(); got != "ru" {
		t.Errorf("currentLocaleCode() = %q, want ru", got)
	}
	if err := initLocale("bogus"); err == nil {
		t.Error("initLocale(bogus) should fail")
	}
}

func mustLocale(t *testing.T, pathToCheck string) localeFile {
	t.Helper()
	data, err := localeFS.ReadFile(pathToCheck)
	if err != nil {
		t.Fatalf("read %s: %v", pathToCheck, err)
	}
	file, err := parseLocale(data)
	if err != nil {
		t.Fatalf("parse %s: %v", pathToCheck, err)
	}
	if file.Meta.Code != strings.TrimSuffix(path.Base(pathToCheck), ".json") {
		t.Errorf("%s: meta.code = %q", pathToCheck, file.Meta.Code)
	}
	return file
}


