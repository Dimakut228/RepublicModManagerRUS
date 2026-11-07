//go:build windows

package main

import (
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

// Steam Workshop catalog calls. Browse results arrive inside the HTML page as
// an escaped JSON block ("results" under total_count). Details come from the
// public ISteamRemoteStorage/GetPublishedFileDetails endpoint in <=100-item
// batches, without an API key.

const (
	// (appid constant wrsrSteamAppID lives in workshop_windows.go)
	catalogBrowseEndpoint  = "https://steamcommunity.com/workshop/browse/"
	defaultDetailsEndpoint = "https://api.steampowered.com/ISteamRemoteStorage/GetPublishedFileDetails/v1/"
	catalogListTTL         = 6 * time.Hour
	catalogDetailsTTL      = 24 * time.Hour
	catalogDetailsBatch    = 100
)

type catalogQuery struct {
	Page             int // 1-based
	Search           string
	Sort             string // ""|trend|mostrecent|toprated
	Period           string // ""|7|30|90|180|365
	Tags             []string
	ShowIncompatible bool
}

type catalogItem struct {
	ID            string
	Title         string
	Description   string
	TimeCreated   int64
	TimeUpdated   int64
	Subscriptions int
	Favorited     int
	Views         int
	FileSize      int64
	PreviewURL    string
	Creator       string
	Tags          []string
	Visibility    int
	Banned        bool
}

type catalogClient struct {
	HTTPClient      *http.Client
	BrowseEndpoint  string
	DetailsEndpoint string
	KnowledgeRoot   string
}

func sortParameter(sortCode string) string {
	switch sortCode {
	case catalogSortTrend, catalogSortRecent, catalogSortTopRated, "":
		if sortCode == "" {
			return catalogSortTrend
		}
		return sortCode
	default:
		return catalogSortTrend
	}
}

func catalogCachePath(knowledgeRoot string) string {
	return catalogCacheRoot(knowledgeRoot)
}

/* Browse parsing ----------------------------------------------------------------- */

var resultsAnchor = regexp.MustCompile(`"results":\[`)
var totalCountRegex = regexp.MustCompile(`"total_count":(\d+)`)
var resultsFallback = regexp.MustCompile(`"publishedfileid":"(\d+)"`)

// JSON-extract the escaped "results":[...] array from the embedded state block.
func extractResultsArray(src string, start int) (string, bool) {
	depth := 0
	i := start
	inString := false
	for i < len(src) {
		ch := src[i]
		if inString {
			if ch == '\\' && i+1 < len(src) {
				i += 2
				continue
			}
			if ch == '"' {
				inString = false
			}
		} else {
			switch ch {
			case '"':
				inString = true
			case '[':
				depth++
			case ']':
				depth--
				if depth == 0 {
					return src[start : i+1], true
				}
			}
		}
		i++
	}
	return "", false
}

type browseResultRow struct {
	PublishedFileID string `json:"publishedfileid"`
	Title           string `json:"title"`
}

func parseBrowseResults(html string) ([]string, int, error) {
	// The page embeds state JSON with escaped quotes. Work on a lightly
	// unescaped copy (\" -> ") so the bracket scanner sees the real JSON.
	unescaped := strings.ReplaceAll(html, `\"`, `"`)

	total := 0
	extracted := ""
	if totalCountMatch := totalCountRegex.FindStringSubmatch(unescaped); totalCountMatch != nil {
		total = atoiDefault(totalCountMatch[1])
	}
	if anchorIdx := strings.Index(unescaped, `"results":[`); anchorIdx >= 0 {
		start := anchorIdx + len(`"results":[`) - 1
		if array, ok := extractResultsArray(unescaped[start:], 0); ok {
			extracted = array
		}
	}
	if extracted != "" {
		var rows []browseResultRow
		if err := json.Unmarshal([]byte(extracted), &rows); err == nil && len(rows) > 0 {
			ids := make([]string, 0, len(rows))
			for _, row := range rows {
				ids = append(ids, row.PublishedFileID)
			}
			return ids, total, nil
		}
	}
	// Fallback: id regex sweep (order preserved).
	ids := resultsFallback.FindAllStringSubmatch(unescaped, -1)
	out := make([]string, 0, len(ids))
	for _, m := range ids {
		out = append(out, m[1])
	}
	if len(out) == 0 {
		return nil, total, fmt.Errorf("catalog page did not yield any items")
	}
	return out, total, nil
}

func atoiDefault(value string) int {
	n := 0
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0
		}
		n = n*10 + int(value[i]-'0')
	}
	return n
}

// jsonBalancedSpan returns the region [start, end] starting at the first open
// bracket in src that closes with the matching close bracket.
func jsonBalancedSpan(src string) (string, bool) {
	depth := 0
	i := 0
	inString := false
	for i < len(src) {
		ch := src[i]
		if inString {
			if ch == '\\' && i+1 < len(src) {
				i += 2
				continue
			}
			if ch == '"' {
				inString = false
			}
		} else {
			switch ch {
			case '"':
				inString = true
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return src[:i+1], true
				}
			}
		}
		i++
	}
	return "", false
}

// parseCatalogCategories extracts the public tag names Steam advertises in the
// "declaredTags" block of the browse page. Unknown shapes yield nil so the
// client falls back to the built-in list.
func parseCatalogCategories(html string) []string {
	unescaped := strings.ReplaceAll(html, `\"`, `"`)
	idx := strings.Index(unescaped, `"declaredTags":{`)
	if idx < 0 {
		return nil
	}
	block, ok := jsonBalancedSpan(unescaped[idx+len(`"declaredTags":`):])
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	var declared struct {
		MtxTags []struct {
			Tags []struct {
				Name        string `json:"name"`
				DisplayName string `json:"display_name"`
				AdminOnly   bool   `json:"admin_only"`
			} `json:"tags"`
		} `json:"mtx_tags"`
	}
	if json.Unmarshal([]byte(block), &declared) == nil {
		for _, group := range declared.MtxTags {
			for _, tag := range group.Tags {
				name := tag.DisplayName
				if name == "" {
					name = tag.Name
				}
				if tag.AdminOnly || name == "" || seen[name] {
					continue
				}
				seen[name] = true
				out = append(out, name)
			}
		}
		return out
	}
	re := regexp.MustCompile(`"name":"([^"]{1,64})"`)
	for _, m := range re.FindAllStringSubmatch(block, -1) {
		name := m[1]
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

/* Fetching ------------------------------------------------------------------------ */

type cachedPage struct {
	CreatedAt int64    `json:"createdAt"`
	Total     int      `json:"total"`
	IDs       []string `json:"ids"`
}

func pageCachePath(client *catalogClient, q catalogQuery) string {
	seed := fmt.Sprintf("p%q_s%q_t%q_p%q_tags%v_inc%v", q.Page, q.Search, q.Sort, q.Period, q.Tags, q.ShowIncompatible)
	sum := sha256.Sum256([]byte(seed))
	return filepath.Join(catalogCachePath(client.KnowledgeRoot), "page-"+hex.EncodeToString(sum[:])[:20]+".json")
}

func detailsCachePath(client *catalogClient, id string) string {
	return filepath.Join(catalogCachePath(client.KnowledgeRoot), "details-"+id+".json")
}

func loadPageCache(path string, ttl time.Duration) (cachedPage, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cachedPage{}, false
	}
	var cached cachedPage
	if json.Unmarshal(data, &cached) != nil {
		return cachedPage{}, false
	}
	if time.Since(time.Unix(cached.CreatedAt, 0)) > ttl {
		return cachedPage{}, false
	}
	return cached, true
}

func savePageCache(path string, page cachedPage) {
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o755); mkdirErr != nil {
		return
	}
	page.CreatedAt = time.Now().Unix()
	data, marshalErr := json.MarshalIndent(page, "", " ")
	if marshalErr != nil {
		return
	}
	_ = os.WriteFile(path, append(data, '\n'), 0o644)
}

func loadDetailsCache(path string, ttl time.Duration) (catalogItem, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return catalogItem{}, false
	}
	var item catalogItem
	if json.Unmarshal(data, &item) != nil || item.ID == "" {
		return catalogItem{}, false
	}
	info, statErr := os.Stat(path)
	if statErr == nil && time.Since(info.ModTime()) > ttl {
		return catalogItem{}, false
	}
	return item, true
}

func saveDetailsCache(path string, item catalogItem) {
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o755); mkdirErr != nil {
		return
	}
	data, marshalErr := json.MarshalIndent(item, "", " ")
	if marshalErr != nil {
		return
	}
	_ = os.WriteFile(path, append(data, '\n'), 0o644)
}

func (c *catalogClient) fetchPageCached(q catalogQuery) (ids []string, total int, stale bool, err error) {
	if c.KnowledgeRoot == "" {
		html, total, err := c.fetchBrowseHTML(q)
		if err != nil {
			return nil, 0, false, err
		}
		ids, _, parseErr := parseBrowseResults(html)
		return ids, total, false, parseErr
	}
	path := pageCachePath(c, q)
	if page, ok := loadPageCache(path, catalogListTTL); ok {
		return page.IDs, page.Total, false, nil
	}
	html, liveTotal, liveErr := c.fetchBrowseHTML(q)
	if liveErr == nil {
		ids := []string{}
		if recovered, _, parseErr := parseCatalogPageIDs(html); parseErr == nil {
			ids = recovered
		}
		if len(ids) > 0 {
			savePageCache(path, cachedPage{Total: liveTotal, IDs: ids})
			return ids, liveTotal, false, nil
		}
	}
	if stalePage, ok := loadPageCacheStale(path); ok {
		return stalePage.IDs, stalePage.Total, true, nil
	}
	return nil, 0, false, liveErr
}

func parseCatalogPageIDs(html string) ([]string, int, error) {
	return parseBrowseResults(html)
}

// fetchItemDescription reads the full description from the item's community
// page. The details API sometimes returns an empty short_description for
// Workshop items whose author left it blank in the metadata feed.
func (c *catalogClient) fetchItemDescription(id string) (string, error) {
	if !validWorkshopID(id) {
		return "", fmt.Errorf("invalid Workshop item id")
	}
	req, err := http.NewRequest(http.MethodGet, workshopURL(id)+"&l=english", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("RepublicModManagerRUS/%s", launcherVersion))
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := c.doRequestWithRetries(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("item page returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	text := string(body)
	if idx := strings.Index(text, `class="workshopItemDescription"`); idx >= 0 {
		if open := strings.Index(text[idx:], ">"); open >= 0 {
			rest := text[idx+open+1:]
			if close := strings.Index(rest, "</div>"); close >= 0 {
				raw := rest[:close]
				raw = stripHTMLTags(raw)
				raw = strings.TrimSpace(plainWorkshopDescription(raw))
				if len(raw) > 6000 {
					raw = raw[:6000]
				}
				return raw, nil
			}
		}
	}
	return "", fmt.Errorf("item page did not contain a description block")
}

var htmlTagRegex = regexp.MustCompile(`<[^>]+>`)

func stripHTMLTags(text string) string {
	text = htmlTagRegex.ReplaceAllString(text, " ")
	replacer := strings.NewReplacer(
		"&amp;", "&", "&quot;", `"`, "&#39;", "'", "&lt;", "<", "&gt;", ">",
		"&nbsp;", " ", "\r\n", "\n", "\r", "\n",
	)
	text = replacer.Replace(text)
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	return text
}

// stripBareURLLines drops lines that consist of a bare URL only (usually a
// workshop preview image link); inline links inside sentences survive.
func stripBareURLLines(text string) string {
	// Preview image URLs are noise in a description preview. Remove Steam's
	// media hosts outright; ordinary links inside sentences survive.
	text = regexp.MustCompile(`https?://(?:images\.steamusercontent\.com|steamuserimages-[a-z0-9]+\.akamaihd\.net)/\S+`).ReplaceAllString(text, "")
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	bareURL := regexp.MustCompile(`^\s*https?://\S+\s*$`)
	for _, line := range lines {
		if bareURL.MatchString(line) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// back to the built-in catalogPrimaryTags list on error.
func (c *catalogClient) fetchCategories() []string {
	root := catalogCachePath(c.KnowledgeRoot)
	if root == "" {
		return nil
	}
	path := filepath.Join(root, "categories.json")
	data, _ := os.ReadFile(path)
	var cached struct {
		CreatedAt  int64    `json:"createdAt"`
		Categories []string `json:"categories"`
	}
	if json.Unmarshal(data, &cached) == nil && time.Since(time.Unix(cached.CreatedAt, 0)) < 24*time.Hour && len(cached.Categories) > 0 {
		return cached.Categories
	}
	// Refresh regardless of TTL — the list is small and Steam changes it.
	html, _, err := c.fetchBrowseHTML(catalogQuery{Page: 1, Sort: catalogSortTrend})
	if err != nil {
		return nil
	}
	cats := parseCatalogCategories(html)
	if len(cats) == 0 {
		return nil
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o755); mkdirErr == nil {
		payload, _ := json.MarshalIndent(map[string]any{
			"createdAt":  time.Now().Unix(),
			"categories": cats,
		}, "", " ")
		_ = os.WriteFile(path, append(payload, '\n'), 0o644)
	}
	return cats
}

func loadPageCacheStale(path string) (cachedPage, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cachedPage{}, false
	}
	var cached cachedPage
	if json.Unmarshal(data, &cached) != nil || len(cached.IDs) == 0 {
		return cachedPage{}, false
	}
	return cached, true
}

func (c *catalogClient) cachePageDetails(ids []string, fetch func([]string) (map[string]catalogItem, error)) (map[string]catalogItem, error) {
	MissingIDs := []string{}
	for _, id := range ids {
		if _, ok := loadDetailsCache(detailsCachePath(c, id), catalogDetailsTTL); !ok {
			MissingIDs = append(MissingIDs, id)
		}
	}
	if len(MissingIDs) == 0 {
		return c.loadAllDetails(ids)
	}
	items, err := fetch(MissingIDs)
	if err != nil {
		return nil, err
	}
	for id, item := range items {
		if item.ID == id {
			saveDetailsCache(detailsCachePath(c, id), item)
		}
	}
	return c.loadAllDetails(ids)
}

func (c *catalogClient) loadAllDetails(ids []string) (map[string]catalogItem, error) {
	out := map[string]catalogItem{}
	for _, id := range ids {
		if item, ok := loadDetailsCache(detailsCachePath(c, id), catalogDetailsTTL); ok {
			out[id] = item
		}
	}
	return out, nil
}

func (c *catalogClient) doRequestWithRetries(req *http.Request) (*http.Response, error) {
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := client.Do(req.Clone(req.Context()))
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(time.Duration(300*attempt) * time.Millisecond)
	}
	return nil, lastErr
}

func (c *catalogClient) fetchBrowseHTML(q catalogQuery) (string, int, error) {
	browse := c.BrowseEndpoint
	if browse == "" {
		browse = catalogBrowseEndpoint
	}
	values := url.Values{}
	values.Set("appid", wrsrSteamAppID)
	values.Set("p", fmt.Sprintf("%d", maxInt(1, q.Page)))
	values.Set("numperpage", "30")
	values.Set("section", "readytouseitems")
	values.Set("browsesort", sortParameter(q.Sort))
	if q.Period != "" && sortParameter(q.Sort) == catalogSortTrend {
		values.Set("days", q.Period)
	}
	if q.Search != "" {
		values.Set("searchtext", q.Search)
	}
	for _, tag := range q.Tags {
		values.Add("requiredtags[]", tag)
	}
	if q.ShowIncompatible {
		// Steam's browse page exposes "Show incompatible items" through the
		// showincompatible query flag.
		values.Set("showincompatible", "1")
	}
	req, err := http.NewRequest(http.MethodGet, browse+"?"+values.Encode(), nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("RepublicModManagerRUS/%s", launcherVersion))
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := c.doRequestWithRetries(req)
	if err != nil {
		return "", 0, fmt.Errorf("catalog network error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("catalog returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
	if err != nil {
		return "", 0, err
	}
	html := string(body)
	if strings.Contains(html, "<title>Steam Community :: Error</title>") {
		return "", 0, fmt.Errorf("catalog returned an error page")
	}
	ids, total, sweepErr := parseBrowseResults(html)
	if sweepErr != nil {
		return "", 0, sweepErr
	}
	_ = ids
	return html, total, nil
}

func (c *catalogClient) fetchDetails(ids []string) (map[string]catalogItem, error) {
	if c.KnowledgeRoot != "" {
		return c.cachePageDetails(ids, c.fetchDetailsNetwork)
	}
	return c.fetchDetailsNetwork(ids)
}

func (c *catalogClient) fetchDetailsNetwork(ids []string) (map[string]catalogItem, error) {
	endpoint := c.DetailsEndpoint
	if endpoint == "" {
		endpoint = defaultDetailsEndpoint
	}
	out := map[string]catalogItem{}
	for start := 0; start < len(ids); start += catalogDetailsBatch {
		end := min(start+catalogDetailsBatch, len(ids))
		batch := ids[start:end]
		values := url.Values{}
		values.Set("itemcount", fmt.Sprintf("%d", len(batch)))
		for i, id := range batch {
			values.Set(fmt.Sprintf("publishedfileids[%d]", i), id)
		}
		req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(values.Encode()))
		if err != nil {
			return out, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "RepublicModManagerRUS/1.0.0")
		resp, err := c.doRequestWithRetries(req)
		if err != nil {
			return out, fmt.Errorf("catalog details network error: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if resp.StatusCode >= 500 && readErr == nil {
			// retry once more for transient 5xx
			retryResp, retryErr := c.doRequestWithRetries(req)
			if retryErr == nil {
				body, readErr = io.ReadAll(io.LimitReader(retryResp.Body, 8<<20))
				retryResp.Body.Close()
				resp = retryResp
			}
		}
		if readErr != nil {
			return out, readErr
		}
		if resp.StatusCode != http.StatusOK {
			return out, fmt.Errorf("catalog details returned status %d", resp.StatusCode)
		}
		items, err := parseDetailsBody(body)
		if err != nil {
			return out, err
		}
		for id, item := range items {
			out[id] = item
		}
	}
	return out, nil
}

/* Details parsing ------------------------------------------------------------------ */

type detailsResponse struct {
	Response struct {
		Result               int          `json:"result"`
		PublishedFileDetails []detailsRow `json:"publishedfiledetails"`
	} `json:"response"`
}

type detailsRow struct {
	PublishedFileID  string    `json:"publishedfileid"`
	Title            string    `json:"title"`
	ShortDescription string    `json:"short_description"`
	Description      string    `json:"description"`
	TimeCreated      flexInt64 `json:"time_created"`
	TimeUpdated      flexInt64 `json:"time_updated"`
	Subscriptions    flexInt64 `json:"subscriptions"`
	Favorited        flexInt64 `json:"favorited"`
	Views            flexInt64 `json:"views"`
	FileSize         flexInt64 `json:"file_size"`
	PreviewURL       string    `json:"preview_url"`
	Creator          string    `json:"creator"`
	Visibility       flexInt64 `json:"visibility"`
	Banned           flexBool  `json:"banned"`
	Tags             []struct {
		DisplayName string `json:"display_name"`
	} `json:"tags"`
}

// flexInt64 accepts either a JSON number or a string that holds digits. Steam
// sends some of these fields differently across item shapes.
type flexInt64 int64

// flexBool accepts Steam's mixed boolean encoding (true/false or 0/1).
type flexBool bool

func (f *flexBool) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(strings.Trim(string(data), `"`))
	switch text {
	case "1", "true", "True", "TRUE":
		*f = true
	case "0", "false", "False", "FALSE", "", "null":
		*f = false
	default:
		*f = false
	}
	return nil
}

func (f *flexInt64) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(strings.Trim(string(data), `"`))
	if text == "" || text == "null" {
		*f = 0
		return nil
	}
	value := 0
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			*f = 0
			return nil
		}
		value = value*10 + int(text[i]-'0')
	}
	*f = flexInt64(value)
	return nil
}

func parseDetailsBody(data []byte) (map[string]catalogItem, error) {
	var decoded detailsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("catalog details parse failed: %w", err)
	}
	out := map[string]catalogItem{}
	for _, row := range decoded.Response.PublishedFileDetails {
		description := strings.TrimSpace(row.Description)
		if description == "" {
			description = strings.TrimSpace(row.ShortDescription)
		}
		description = strings.TrimSpace(plainWorkshopDescription(stripBareURLLines(stripHTMLTags(description))))
		item := catalogItem{
			ID:            row.PublishedFileID,
			Title:         row.Title,
			Description:   description,
			TimeCreated:   int64(row.TimeCreated),
			TimeUpdated:   int64(row.TimeUpdated),
			Subscriptions: int(row.Subscriptions),
			Favorited:     int(row.Favorited),
			Views:         int(row.Views),
			FileSize:      int64(row.FileSize),
			PreviewURL:    row.PreviewURL,
			Creator:       row.Creator,
			Visibility:    int(row.Visibility),
			Banned:        bool(row.Banned),
		}
		for _, tag := range row.Tags {
			item.Tags = append(item.Tags, tag.DisplayName)
		}
		if validWorkshopID(item.ID) {
			out[item.ID] = item
		}
	}
	return out, nil
}
