package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func loadBrowseFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(data)
}

func TestCatalogParseFixture(t *testing.T) {
	html := loadBrowseFixture(t, "browse_p1.html")
	ids, total, err := parseBrowseResults(html)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if total <= 0 {
		t.Errorf("total = %d, want positive", total)
	}
	if len(ids) == 0 {
		t.Fatal("no items parsed from the browse page")
	}
	if len(ids) != 30 {
		t.Errorf("parsed %d items, want 30", len(ids))
	}
	if ids[0] != "3831575284" {
		t.Errorf("first id = %q", ids[0])
	}
}

func TestSortParameter(t *testing.T) {
	if got := sortParameter(""); got != "trend" {
		t.Errorf("sortParameter(\"\") = %q, want trend", got)
	}
	if got := sortParameter("toprated"); got != "toprated" {
		t.Errorf("sortParameter(toprated) = %q", got)
	}
	if got := sortParameter("bogus"); got != "trend" {
		t.Errorf("bogus order must fall back to trend, got %q", got)
	}
}

func TestCatalogDetailsEnsure100(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		count, _ := json.Number(r.FormValue("itemcount")).Int64()
		if count > 100 {
			t.Errorf("itemcount %d exceeds the 100-item API limit", count)
		}
		var ids []string
		for i := 0; r.Form.Has(fmt.Sprintf("publishedfileids[%d]", i)); i++ {
			ids = append(ids, r.FormValue(fmt.Sprintf("publishedfileids[%d]", i)))
		}
		details := []map[string]any{}
		for _, id := range ids {
			details = append(details, map[string]any{
				"publishedfileid": id,
				"title":           "Mod " + id,
				"file_size":       12,
				"subscriptions":   3,
				"favorited":       5,
				"views":           7,
				"time_created":    10,
				"time_updated":    20,
				"preview_url":     "http://example.invalid/preview.jpg",
				"tags":            []map[string]any{{"display_name": "World Map"}},
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"response": map[string]any{"publishedfiledetails": details}})
	}))
	defer server.Close()

	ids := make([]string, 120)
	for i := range ids {
		ids[i] = strconv.Itoa(1000 + i)
	}
	client := &catalogClient{
		HTTPClient:      server.Client(),
		DetailsEndpoint: server.URL,
		KnowledgeRoot:   t.TempDir(),
	}
	items, err := client.fetchDetails(ids)
	if err != nil {
		t.Fatalf("fetchDetails: %v", err)
	}
	if len(items) != 120 {
		t.Errorf("items = %d, want 120", len(items))
	}
	if calls != 2 {
		t.Errorf("HTTP calls = %d, want 2", calls)
	}
}

func TestCatalogParseDetailsBody(t *testing.T) {
	body := `{"response":{"result":1,"publishedfiledetails":[{"publishedfileid":"123","title":"Some mod","short_description":"desc","time_created":1,"time_updated":2,"subscriptions":3,"favorited":4,"views":5,"file_size":6,"preview_url":"http://x/1.jpg","tags":[{"display_name":"World Maps"}],"visibility":0,"banned":false}]}}`
	items, err := parseDetailsBody([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	item, ok := items["123"]
	if !ok {
		t.Fatal("item 123 missing")
	}
	if item.Title != "Some mod" || item.Subscriptions != 3 || item.FileSize != 6 || len(item.Tags) != 1 {
		t.Errorf("item fields wrong: %+v", item)
	}
}

func TestCatalogRetryOn5xx(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"response":{"result":1,"publishedfiledetails":[{"publishedfileid":"9","title":"x"}]}}`))
	}))
	defer server.Close()
	client := &catalogClient{HTTPClient: server.Client(), DetailsEndpoint: server.URL, KnowledgeRoot: t.TempDir()}
	items, err := client.fetchDetails([]string{"9"})
	if err != nil {
		t.Fatalf("fetchDetails after retry: %v", err)
	}
	if _, ok := items["9"]; !ok {
		t.Error("retried request did not parse item 9")
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}
