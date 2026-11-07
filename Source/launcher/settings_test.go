package main

import (
	"path/filepath"
	"testing"
)

func TestLanguageSwitchPersistsConfig(t *testing.T) {
	base := t.TempDir()
	app := &application{
		baseDir:      base,
		configPath:   filepath.Join(base, "rml", "config.json"),
		knowledgeRoot: filepath.Join(base, "kb"),
	}
	app.cfg = defaultConfig()
	defer applyLanguage("en")
	applyLanguageSetting(app, "ru")
	if app.cfg.Language != "ru" {
		t.Errorf("cfg.Language = %q, want ru", app.cfg.Language)
	}
	if currentLocaleCode() != "ru" {
		t.Errorf("currentLocaleCode() = %q, want ru", currentLocaleCode())
	}
	reloaded := loadConfig(app.configPath)
	if reloaded.Language != "ru" {
		t.Errorf("persisted Language = %q, want ru", reloaded.Language)
	}
}
