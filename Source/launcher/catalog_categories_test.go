//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCatalogCategoriesFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "browse_p1.html"))
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	cats := parseCatalogCategories(string(data))
	want := []string{"Text modification", "Vehicle skin", "Landscape", "Vehicle", "Building skin", "Building editor element", "Building"}
	if len(cats) < len(want) {
		t.Fatalf("categories = %v, want at least %d entries", len(cats), len(want))
	}
	for i, w := range want {
		if cats[i] != w {
			t.Errorf("cats[%d] = %q, want %q", i, cats[i], w)
		}
	}
}

func TestParseCatalogCategoriesBrokenHTML(t *testing.T) {
	if cats := parseCatalogCategories("<html>nothing useful"); len(cats) != 0 {
		t.Errorf("expected nil for broken html, got %v", cats)
	}
}

func TestBuiltInCategoriesFallback(t *testing.T) {
	if len(catalogPrimaryTags) < 5 {
		t.Errorf("builtin tags list %v looks too short", catalogPrimaryTags)
	}
	if !strings.Contains(strings.Join(catalogPrimaryTags, ","), "Text modification") {
		t.Errorf("missing core category")
	}
}
