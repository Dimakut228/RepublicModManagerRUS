package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMaskRoundTripURLsAndBBCode(t *testing.T) {
	raw := "[b]Legacy systems[/b] visit https://example.com/path?q=1 and {$SHIPMENT} tokens."
	masked, tokens := maskMarkup(raw)
	if strings.Contains(masked, "https://") || strings.Contains(masked, "[b]") || strings.Contains(masked, "{$SHIPMENT}") {
		t.Errorf("masked text still contains protectable markup: %q", masked)
	}
	restored, err := unmaskMarkup(masked, tokens)
	if err != nil {
		t.Fatalf("unmask: %v", err)
	}
	if restored != raw {
		t.Errorf("round trip = %q, want %q", restored, raw)
	}
}

func TestUnmaskMissingTokenFails(t *testing.T) {
	if _, err := unmaskMarkup("⟦0⟧ and text", nil); err == nil {
		t.Error("unmask must fail when a token reference has no source")
	}
}

func TestTranslatePauseAfter429(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	translator := &translator{client: server.Client(), knowledgeRoot: t.TempDir(), endpoint: server.URL}
	if _, err := translator.translate("hello world"); err == nil {
		t.Error("expected an error from a 429 response")
	}
	if !time.Now().Before(translator.pausedUntil) {
		t.Error("pause window must be set after a 429")
	}
	if _, err := translator.translate("hello world"); err == nil || !strings.Contains(err.Error(), "paused") && !strings.Contains(err.Error(), "Пауза") {
		t.Errorf("second call must be paused, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 HTTP call, got %d", calls)
	}
}

func TestTranslateChunkLargeText(t *testing.T) {
	chunks := segmentChunks(strings.Repeat("Ab ", 3000))
	if len(chunks) < 2 {
		t.Errorf("expected large text to split into multiple chunks, got %d", len(chunks))
	}
	for _, chunk := range chunks {
		if len(chunk) > translateMaxChunk+64 {
			t.Errorf("chunk too large: %d", len(chunk))
		}
	}
}

func TestTranslationCacheRoundTrip(t *testing.T) {
	root := t.TempDir()
	translator := &translator{client: http.DefaultClient, knowledgeRoot: root, endpoint: "http://example.invalid"}
	source := "Line one\nLine two"
	translationPut(translator, source, "Строка один\nСтрока два")
	got, ok := translationGet(translator, source)
	if !ok || got != "Строка один\nСтрока два" {
		t.Errorf("cache get = (%q, %v)", got, ok)
	}
}

func TestGoogleTranslateResponseParse(t *testing.T) {
	body := `[[["привет","hello",null,null,10]]]`
	got, err := parseGoogleTranslateBody([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got != "привет" {
		t.Errorf("parse = %q", got)
	}
}
