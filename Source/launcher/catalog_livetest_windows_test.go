//go:build windows

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestLiveCatalogFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	client := &catalogClient{HTTPClient: &http.Client{Timeout: 10 * time.Second}, KnowledgeRoot: t.TempDir()}
	ids, total, err := client.fetchPage(catalogQuery{Page: 1, Sort: "trend"})
	if err != nil {
		t.Fatalf("live fetch failed: %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("no ids")
	}
	fmt.Printf("live ids=%d total=%d first=%s\n", len(ids), total, ids[0])
	details, err := client.fetchDetails(ids[:3])
	if err != nil {
		t.Fatalf("details: %v", err)
	}
	for id, item := range details {
		fmt.Printf("detail %s %s size=%d\n", id, item.Title, item.FileSize)
	}
}
