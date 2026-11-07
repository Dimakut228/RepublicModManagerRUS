//go:build windows

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Steam preview thumbnail cache for the catalog. Files land under
// catalogCacheRoot/<knowledge root>/thumbs/<id>.<ext>; the directory is kept
// under 200 MB with an LRU prune so long sessions do not grow without bound.

const (
	thumbMaxBytes        = 1024 * 1024
	thumbDirLimit        = 200 * 1024 * 1024
	thumbTTL             = 30 * 24 * time.Hour
	thumbDownloadTimeout = 10 * time.Second
)

func thumbsRoot(knowledgeRoot string) string {
	return filepath.Join(catalogCacheRoot(knowledgeRoot), "thumbs")
}

func thumbTTLExpired(modTime time.Time, now time.Time) bool {
	return now.Sub(modTime) > thumbTTL
}

// thumbnailPathFor picks an existing cached thumbnail for the item, ignoring
// expired files.
func thumbnailPathFor(knowledgeRoot, itemID string) string {
	dir := thumbsRoot(knowledgeRoot)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, itemID+".") {
			continue
		}
		full := filepath.Join(dir, name)
		info, err := os.Stat(full)
		if err != nil {
			return ""
		}
		if thumbTTLExpired(info.ModTime(), time.Now()) {
			_ = os.Remove(full)
			continue
		}
		return full
	}
	return ""
}

var thumbExtLookup = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func thumbnailDiskBytes(dir string) int64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	var total int64
	for _, entry := range entries {
		info, err := entry.Info()
		if err == nil && !entry.IsDir() {
			total += info.Size()
		}
	}
	return total
}

// pruneThumbLRU deletes the oldest files until the directory fits the limit.
func pruneThumbLRU(dir string, limit int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type fileRow struct {
		full  string
		size  int64
		stamp int64
	}
	rows := make([]fileRow, 0, len(entries))
	var total int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		rows = append(rows, fileRow{
			full:  filepath.Join(dir, entry.Name()),
			size:  info.Size(),
			stamp: info.ModTime().Unix(),
		})
		total += info.Size()
	}
	if total <= limit {
		return
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].stamp < rows[j].stamp })
	for _, row := range rows {
		if total <= limit {
			break
		}
		if os.Remove(row.full) == nil {
			total -= row.size
		}
	}
}

type thumbFetchResult struct {
	itemID string
	path   string
	err    error
}

var thumbnailFetcherMu sync.Mutex

// fetchThumbnail downloads the preview for one item and stores it in the
// cache. The result path is empty on failure; callers should keep the text
// placeholder in that case.
func (c *catalogClient) fetchThumbnail(itemID, previewURL string) (string, error) {
	if itemID == "" || previewURL == "" || !validWorkshopID(itemID) {
		return "", fmt.Errorf("invalid thumbnail request")
	}
	if !strings.HasPrefix(previewURL, "https://") {
		// Steam may serve the preview over http on older items; upgrade for
		// the store-cached thumbs.
		previewURL = strings.Replace(previewURL, "http://", "https://", 1)
	}
	req, err := http.NewRequest(http.MethodGet, previewURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("RepublicModManagerRUS/%s", launcherVersion))
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: thumbDownloadTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("thumbnail service returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, thumbMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > thumbMaxBytes {
		return "", fmt.Errorf("thumbnail exceeds %d KB", thumbMaxBytes/1024)
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	ext, ok := thumbExtLookup[ct]
	if !ok {
		if bytes.HasPrefix(body, []byte{0x89, 'P', 'N', 'G'}) {
			ext = ".png"
		} else if bytes.HasPrefix(body, []byte{0xFF, 0xD8}) {
			ext = ".jpg"
		} else if bytes.HasPrefix(body, []byte{'R', 'I', 'F', 'F'}) {
			ext = ".webp"
		} else {
			return "", fmt.Errorf("thumbnail response is not a recognised image")
		}
	}
	dir := thumbsRoot(c.KnowledgeRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, itemID+ext)
	if writeErr := os.WriteFile(path, body, 0o644); writeErr != nil {
		return "", writeErr
	}
	pruneThumbLRU(dir, thumbDirLimit)
	return path, nil
}

// prefetchCatalogThumbs downloads the missing previews for a batch of items in
// the background and repaints the window when the batch completes. Each item
// is attempted at most once per session.
func (a *application) prefetchCatalogDescriptions(ids []string) {
	a.workshopMu.Lock()
	items := a.catalogItems
	pending := a.catalogDescPending
	if pending == nil {
		pending = map[string]bool{}
		a.catalogDescPending = pending
	}
	a.workshopMu.Unlock()
	client := &catalogClient{
		HTTPClient:    httpDefaultClient,
		KnowledgeRoot: a.knowledgeRoot,
	}
	go func() {
		for _, id := range ids {
			item, ok := items[id]
			if !ok || item.Description != "" {
				continue
			}
			thumbnailFetcherMu.Lock()
			if pending[id] {
				thumbnailFetcherMu.Unlock()
				continue
			}
			pending[id] = true
			thumbnailFetcherMu.Unlock()
			text, err := client.fetchItemDescription(id)
			thumbnailFetcherMu.Lock()
			delete(pending, id)
			thumbnailFetcherMu.Unlock()
			if err != nil || text == "" {
				continue
			}
			a.workshopMu.Lock()
			updated := a.catalogItems[id]
			updated.Description = text
			a.catalogItems[id] = updated
			a.workshopMu.Unlock()
			saveDetailsCache(detailsCachePath(client, id), updated)
			invalidateWindow(a.hwnd)
			postAppMessage(a.hwnd, wmRMLThumbnailDone, 0, 0)
			time.Sleep(150 * time.Millisecond)
		}
	}()
}

func (a *application) prefetchCatalogThumbs(ids []string) {
	a.workshopMu.Lock()
	items := a.catalogItems
	pending := a.catalogThumbPending
	if pending == nil {
		pending = map[string]bool{}
	}
	a.catalogThumbPending = pending
	a.workshopMu.Unlock()
	client := &catalogClient{
		HTTPClient:    httpDefaultClient,
		KnowledgeRoot: a.knowledgeRoot,
	}
	go func() {
		for _, id := range ids {
			thumbnailFetcherMu.Lock()
			if _, busy := pending[id]; busy {
				thumbnailFetcherMu.Unlock()
				continue
			}
			pending[id] = true
			thumbnailFetcherMu.Unlock()
			item, ok := items[id]
			if !ok || item.PreviewURL == "" {
				continue
			}
			if _, err := os.Stat(thumbnailPathFor(a.knowledgeRoot, id)); err == nil {
				// Cached file already covers this preview.
				invalidateWindow(a.hwnd)
				postAppMessage(a.hwnd, wmRMLThumbnailDone, 0, 0)
				continue
			}
			_, err := client.fetchThumbnail(id, item.PreviewURL)
			if err != nil {
				// One polite retry: image hosts occasionally answer 429 under
				// load, and the item would otherwise stay empty for the session.
				time.Sleep(150 * time.Millisecond)
				_, err = client.fetchThumbnail(id, item.PreviewURL)
			}
			thumbnailFetcherMu.Lock()
			delete(pending, id)
			thumbnailFetcherMu.Unlock()
			if err != nil {
				continue
			}
			invalidateWindow(a.hwnd)
			postAppMessage(a.hwnd, wmRMLThumbnailDone, 0, 0)
			time.Sleep(120 * time.Millisecond)
		}
	}()
}
