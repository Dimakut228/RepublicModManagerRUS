package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigMigrationOldUpstream(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	oldConfig := []byte("{\n  \"autoOrder\": true,\n  \"gamePath\": \"E:\\\\games\\\\SovietRepublic\\\\SOVIET64.exe\",\n  \"selectedPlaylist\": \"Default\",\n  \"autoCrashArchives\": true\n}\n")
	if err := os.WriteFile(path, oldConfig, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg := loadConfig(path)
	if cfg.Language != "auto" {
		t.Errorf("Language = %q, want auto", cfg.Language)
	}
	if !cfg.AutoTranslate {
		t.Error("AutoTranslate should default to true for old configs")
	}
	if !cfg.AutoOrder {
		t.Error("old autoOrder = true was lost")
	}
	if cfg.GamePath != "E:\\games\\SovietRepublic\\SOVIET64.exe" {
		t.Errorf("old gamePath = %q", cfg.GamePath)
	}
	if cfg.CatalogSort != "trend" {
		t.Errorf("CatalogSort = %q, want trend", cfg.CatalogSort)
	}
}

func TestSaveConfigRoundTripNewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := defaultConfig()
	cfg.Language = "ru"
	cfg.AutoTranslate = false
	cfg.TranslationNoticeShown = true
	cfg.CatalogSort = "mostrecent"
	cfg.CatalogPeriod = "30"
	cfg.CatalogShowIncompatible = true
	if err := saveConfig(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := loadConfig(path)
	if got.Language != "ru" || got.AutoTranslate != false || !got.TranslationNoticeShown {
		t.Errorf("round-trip lost fields: %+v", got)
	}
	if got.CatalogSort != "mostrecent" || got.CatalogPeriod != "30" || !got.CatalogShowIncompatible {
		t.Errorf("round-trip lost catalog fields: %+v", got)
	}
}

func TestCatalogSortNormalization(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := []byte("{\"catalogSort\":\"bogus\",\"language\":\"auto\"}")
	_ = os.WriteFile(path, raw, 0o644)
	cfg := loadConfig(path)
	if cfg.CatalogSort != "trend" {
		t.Errorf("CatalogSort = %q, want trend after normalization", cfg.CatalogSort)
	}
}

func TestLanguageAutoDetectionLoadsBundle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := []byte("{\"language\":\"ru\"}")
	_ = os.WriteFile(path, raw, 0o644)
	cfg := loadConfig(path)
	defer applyLanguage("en")
	if err := initLocale(cfg.Language); err != nil {
		t.Fatalf("initLocale: %v", err)
	}
	if currentLocaleCode() != "ru" {
		t.Errorf("currentLocaleCode() = %q, want ru", currentLocaleCode())
	}
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatalf("probe: %v", err)
	}
}
