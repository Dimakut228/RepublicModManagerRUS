//go:build windows

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Translation of mod descriptions via the free Google Translate endpoint.
// Only the description text leaves the machine; names, paths and settings never
// travel. Errors never block the UI: the caller always keeps the original text.

const (
	translateMaxChunk = 3800
	translatePauseTTL = 10 * time.Minute
	translateTimeout  = 10 * time.Second
)

const translateEndpoint = "https://clients5.google.com/translate_a/t?client=dict-chrome-ex&sl=auto&tl=ru&q="

type translator struct {
	client        *http.Client // injectable for tests
	knowledgeRoot string
	endpoint      string // overrides translateEndpoint when set (tests)
	pausedUntil   time.Time
}

type translateCacheEntry struct {
	SourceHash string `json:"sourceHash"`
	Translated string `json:"translated"`
	Created    int64  `json:"created"`
}

func defaultTranslator(knowledgeRoot string) *translator {
	return &translator{
		client:        &http.Client{Timeout: translateTimeout},
		knowledgeRoot: knowledgeRoot,
	}
}

func normalizeSource(raw string) string {
	text := strings.ReplaceAll(raw, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.TrimSpace(text)
}

func sourceHash(raw string) string {
	sum := sha256.Sum256([]byte(normalizeSource(raw)))
	return hex.EncodeToString(sum[:8])
}

// Cyrillic presence check: already-Russian text is passed through untouched.
func sourceLooksRussian(raw string) bool {
	runes := []rune(normalizeSource(raw))
	if len(runes) == 0 {
		return false
	}
	cyrillic := 0
	letters := 0
	for _, r := range runes {
		if r >= 0x0400 && r <= 0x04FF {
			cyrillic++
		}
		if r == ' ' || r == '\n' {
			continue
		}
		letters++
	}
	if letters == 0 {
		return false
	}
	return float64(cyrillic)/float64(letters) > 0.4
}

var protectURL = regexp.MustCompile(`https?://\S+`)
var protectBBCode = regexp.MustCompile(`\[[^\]}]{1,64}?\]|\[\]\[[^\]]*\]`)
var protectPlaceholder = regexp.MustCompile(`\{[A-Za-z0-9_%$$\.]+\}|%[svdqxfedXbltoOUP]`)
var maskToken = regexp.MustCompile(`⟦(\d+)⟧`)

func maskMarkup(raw string) (string, []string) {
	tokens := []string{}
	mask := func(pattern *regexp.Regexp, text string) string {
		return pattern.ReplaceAllStringFunc(text, func(match string) string {
			idx := len(tokens)
			tokens = append(tokens, match)
			return fmt.Sprintf("⟦%d⟧", idx)
		})
	}
	out := mask(protectURL, raw)
	out = mask(protectPlaceholder, out)
	out = mask(protectBBCode, out)
	return out, tokens
}

func unmaskMarkup(text string, tokens []string) (string, error) {
	missing := false
	out := maskToken.ReplaceAllStringFunc(text, func(match string) string {
		idxGroup := maskToken.FindStringSubmatch(match)
		if idxGroup == nil || len(idxGroup) < 2 {
			return match
		}
		idx := 0
		if _, err := fmt.Sscanf(idxGroup[1], "%d", &idx); err != nil || idx >= len(tokens) {
			missing = true
			return match
		}
		return tokens[idx]
	})
	if missing {
		return "", fmt.Errorf("translated text lost protected tokens")
	}
	return out, nil
}

// segmentChunks splits text into chunks no longer than translateMaxChunk,
// preferring paragraph and sentence boundaries.
func segmentChunks(text string) []string {
	if text == "" {
		return []string{}
	}
	chunks := []string{}
	for _, paragraph := range strings.Split(text, "\n\n") {
		if len(paragraph) == 0 {
			continue
		}
		if len(paragraph) <= translateMaxChunk {
			chunks = append(chunks, paragraph)
			continue
		}
		rest := paragraph
		for len(rest) > 0 {
			cut := min(translateMaxChunk, len(rest))
			if cut < len(rest) {
				// Prefer a sentence boundary or a space close to the limit.
				window := rest[:cut]
				best := strings.LastIndexAny(window, ".!?")
				if best > translateMaxChunk/2 {
					cut = best + 1
				} else {
					if spaced := strings.LastIndex(window[:cut], " "); spaced > translateMaxChunk/2 {
						cut = spaced + 1
					}
				}
			}
			chunks = append(chunks, strings.TrimSpace(rest[:cut]))
			rest = strings.TrimSpace(rest[cut:])
		}
	}
	return chunks
}

func pausedTranslatorError(t *translator) error {
	until := t.pausedUntil.Format("15:04:05")
	return fmt.Errorf("translation is paused until %s", until)
}

// translate performs the machine translation of the raw description string.
// It respects the pause window, transfers the text in <=3800-char chunks,
// protects markup tokens and caches the result per source hash.
func (t *translator) translate(raw string) (string, error) {
	source := normalizeSource(raw)
	if source == "" {
		return raw, nil
	}
	if sourceLooksRussian(source) {
		return raw, nil
	}
	if cached, ok := translationGet(t, source); ok {
		return cached, nil
	}
	if !t.pausedUntil.IsZero() && time.Now().Before(t.pausedUntil) {
		return "", pausedTranslatorError(t)
	}
	masked, tokens := maskMarkup(source)
	chunks := segmentChunks(masked)
	translatedChunks := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		out, err := t.translateChunk(chunk)
		if err != nil {
			return "", err
		}
		restored, err := unmaskMarkup(out, tokens)
		if err != nil {
			return "", err
		}
		translatedChunks = append(translatedChunks, restored)
	}
	translated := strings.Join(translatedChunks, "\n\n")
	if strings.TrimSpace(translated) == "" {
		return raw, fmt.Errorf("translation is empty")
	}
	translationPut(t, source, translated)
	return translated, nil
}

func (t *translator) translateChunk(chunk string) (string, error) {
	endpoint := t.endpoint
	if endpoint == "" {
		endpoint = translateEndpoint
	}
	sep := "?"
	if strings.Contains(endpoint, "?") {
		sep = "&"
	}
	reqURL := endpoint + sep + "q=" + url.QueryEscape(chunk)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "RepublicModManagerRUS/1.0.0")
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("translation network error: %w", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden:
		t.pausedUntil = time.Now().Add(translatePauseTTL)
		return "", fmt.Errorf("translation service refused the request (status %d); paused for 10 minutes", resp.StatusCode)
	case resp.StatusCode == http.StatusOK:
	default:
		return "", fmt.Errorf("translation service returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	value, err := parseGoogleTranslateBody(body)
	if err != nil {
		return "", err
	}
	return value, nil
}

func parseGoogleTranslateBody(body []byte) (string, error) {
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", fmt.Errorf("translation response parse failed: %w", err)
	}
	var out strings.Builder
	segments, _ := decoded.([]any)
	if segments == nil {
		return "", fmt.Errorf("translation response shape unexpected")
	}
	first, _ := segments[0].([]any)
	for _, seg := range first {
		row, _ := seg.([]any)
		if len(row) == 0 {
			continue
		}
		if piece, ok := row[0].(string); ok {
			out.WriteString(piece)
		}
	}
	return out.String(), nil
}

// translationGet/Put store translations keyed by the normalised source hash.
func translationGet(t *translator, source string) (string, bool) {
	dir := translationCachePath(t.knowledgeRoot)
	data, err := os.ReadFile(filepath.Join(dir, sourceHash(source)+".json"))
	if err != nil {
		return "", false
	}
	var entry translateCacheEntry
	if json.Unmarshal(data, &entry) != nil {
		return "", false
	}
	if entry.Translated == "" {
		return "", false
	}
	return entry.Translated, true
}

func translationPut(t *translator, source, translated string) {
	dir := translationCachePath(t.knowledgeRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	entry := translateCacheEntry{
		SourceHash: sourceHash(source),
		Translated: translated,
		Created:    time.Now().Unix(),
	}
	data, err := json.MarshalIndent(entry, "", " ")
	if err != nil {
		return
	}
	path := filepath.Join(dir, entry.SourceHash+".json")
	if _, statErr := os.Stat(path); statErr == nil {
		return
	}
	_ = os.WriteFile(path, append(data, '\n'), 0o644)
}

var _ = bytes.TrimSpace // keep imports honest if refactors touch helpers
