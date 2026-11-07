//go:build windows

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Steam Workshop catalog UI state, fetch orchestration and actions live here.
// The drawing side is `drawCatalogPage` further down the file; the network
// layer lives in catalog_windows.go.

var httpDefaultClient = &http.Client{Timeout: 10 * time.Second}

var catalogPrimaryTags = []string{
	"Text modification", "Vehicle skin", "Landscape", "Vehicle",
	"Building", "Building skin", "Building editor element", "Script",
	"World Maps", "Biomes", "Early Start", "Soundtrack", "Help for Ukraine",
}

var catalogPeriodChoices = []string{"", "7", "30", "90", "180", "365"}

func catalogPeriodLabel(period string) string {
	switch period {
	case "7":
		return tr("catalog.period.week")
	case "30":
		return tr("catalog.period.month")
	case "90":
		return tr("catalog.period.three_months")
	case "180":
		return tr("catalog.period.six_months")
	case "365":
		return tr("catalog.period.year")
	default:
		return tr("catalog.period.all")
	}
}

func catalogSortLabelFor(code string) string {
	switch code {
	case catalogSortRecent:
		return tr("catalog.sort.recent")
	case catalogSortTopRated:
		return tr("catalog.sort.top_rated")
	default:
		return tr("catalog.sort.trend")
	}
}

/* Fetch orchestration -------------------------------------------------------------- */

func (a *application) startCatalogFetch() {
	a.workshopMu.Lock()
	a.catalogBusy = true
	a.catalogFetchGen++
	gen := a.catalogFetchGen
	a.workshopMu.Unlock()
	a.status = tr("catalog.state.loading")
	invalidateWindow(a.hwnd)
	client := &catalogClient{
		HTTPClient:    httpDefaultClient,
		KnowledgeRoot: a.knowledgeRoot,
	}
	query := catalogQuery{
		Page:             maxInt(1, a.catalogPage),
		Search:           a.catalogSearch,
		Sort:             a.cfg.CatalogSort,
		Period:           a.cfg.CatalogPeriod,
		Tags:             append([]string{}, a.catalogTags...),
		ShowIncompatible: a.cfg.CatalogShowIncompatible,
	}
	go func() {
		ids, total, stale, err := client.fetchPageCached(query)
		a.workshopMu.Lock()
		if gen != a.catalogFetchGen {
			a.workshopMu.Unlock()
			return
		}
		a.catalogIDs = ids
		a.catalogTotal = total
		a.catalogOffline = stale
		a.workshopMu.Unlock()
		if err != nil {
			a.status = tr("catalog.load_failed") + err.Error()
			a.workshopMu.Lock()
			a.catalogBusy = false
			a.catalogOffline = false
			a.workshopMu.Unlock()
			invalidateWindow(a.hwnd)
			postAppMessage(a.hwnd, wmRMLCatalogDone, 1, 0)
			return
		}
		a.catalogPage = maxInt(1, query.Page)
		a.catalogSel = 0
		details, detailsErr := client.fetchDetails(ids)
		a.workshopMu.Lock()
		if gen != a.catalogFetchGen {
			a.workshopMu.Unlock()
			return
		}
		a.catalogItems = details
		a.catalogBusy = false
		a.catalogOffline = stale || detailsErr != nil && len(details) == 0
		a.workshopMu.Unlock()
		switch {
		case detailsErr != nil:
			a.status = tr("catalog.details_failed") + detailsErr.Error()
		case stale:
			a.status = tr("catalog.state.offline")
		default:
			a.status = fmt.Sprintf(tr("catalog.rows.count"), len(details), total)
		}
		invalidateWindow(a.hwnd)
		postAppMessage(a.hwnd, wmRMLCatalogDone, 0, 0)
	}()
}

/* Selection and actions ------------------------------------------------------------ */

func (a *application) selectedCatalogItem() (catalogItem, bool) {
	a.workshopMu.Lock()
	items := a.catalogItems
	ids := a.catalogIDs
	a.workshopMu.Unlock()
	if a.catalogSel < 0 || a.catalogSel >= len(ids) {
		return catalogItem{}, false
	}
	item, ok := items[ids[a.catalogSel]]
	return item, ok
}

func (a *application) selectCatalogRow(index int) {
	a.workshopMu.Lock()
	ids := a.catalogIDs
	a.workshopMu.Unlock()
	if index < 0 || index >= len(ids) {
		return
	}
	a.catalogSel = index
	a.catalogDetailScroll = 0
	a.status = tr("catalog.state.selected")
	invalidateWindow(a.hwnd)
}

func (a *application) subscribeCatalogSelection() {
	item, ok := a.selectedCatalogItem()
	if !ok {
		return
	}
	if a.game == nil || a.game.Path == "" {
		a.status = tr("catalog.subscribe.failed")
		invalidateWindow(a.hwnd)
		return
	}
	a.status = tr("catalog.subscribing") + item.Title + "…"
	invalidateWindow(a.hwnd)
	gamePath := a.game.Path
	go func() {
		outcome := subscribeWorkshopItems(gamePath, []string{item.ID})
		var message string
		switch {
		case outcome.Confirmed == 1 && outcome.Failed == 0 && outcome.Requested == 0:
			message = tr("catalog.subscribed_already")
		case outcome.Confirmed == 1 && outcome.Failed == 0:
			message = tr("catalog.subscribed_done")
		case outcome.Failed == 1:
			message = tr("catalog.subscribed_failed")
		default:
			message = tr("catalog.subscribed_requested")
		}
		a.status = message + " — " + item.Title
		invalidateWindow(a.hwnd)
	}()
}

func (a *application) unsubscribeCatalogSelection() {
	item, ok := a.selectedCatalogItem()
	if !ok {
		return
	}
	if a.game == nil || a.game.Path == "" {
		a.status = tr("catalog.subscribe.failed")
		invalidateWindow(a.hwnd)
		return
	}
	a.status = tr("catalog.unsubscribing") + item.Title + "…"
	invalidateWindow(a.hwnd)
	gamePath := a.game.Path
	go func() {
		outcome := unsubscribeWorkshopItem(gamePath, item.ID)
		message := outcome.Message
		a.status = message + " — " + item.Title
		invalidateWindow(a.hwnd)
	}()
}

func (a *application) openCatalogWorkshopPage() {
	item, ok := a.selectedCatalogItem()
	if !ok {
		return
	}
	openPath(workshopURL(item.ID))
}

func catalogTagsLabel(tags []string) string {
	if len(tags) == 0 {
		return tr("catalog.filters.none")
	}
	return strings.Join(tags, ", ")
}

func (a *application) catalogItemInstalled(itemID string) bool {
	if a.game == nil || a.game.Path == "" {
		return false
	}
	root := workshopRootFromGamePath(a.game.Path)
	if root == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(root, itemID))
	return err == nil && info.IsDir()
}

func catalogCatalogTitle(item catalogItem) string {
	title := item.Title
	if title == "" {
		title = tr("t.workshop_item") + item.ID
	}
	return title
}

func (a *application) ensureTranslated(desc string) (string, string, bool) {
	// returns (result, state, available) ; available=false if feature disabled
	if !a.cfg.AutoTranslate || currentLocaleCode() != "ru" {
		return "", "", false
	}
	if desc == "" {
		return "", "idle", true
	}
	if sourceLooksRussian(desc) {
		return "", "idle", true
	}
	a.translateMu.Lock()
	defer a.translateMu.Unlock()
	a.translateGen++
	gen := a.translateGen
	if a.translateSource == desc && (a.translateState == "done" || a.translateState == "busy" || a.translateState == "error") {
		return a.translateResult, a.translateState, true
	}
	a.translateSource = desc
	translator := defaultTranslator(a.knowledgeRoot)
	if cached, ok := translationGet(translator, desc); ok {
		a.translateResult = cached
		a.translateState = "done"
		a.translateError = ""
		return cached, "done", true
	}
	a.translateResult = ""
	a.translateState = "busy"
	a.translateError = ""
	go func() {
		text, err := translator.translate(desc)
		a.translateMu.Lock()
		defer a.translateMu.Unlock()
		if gen != a.translateGen || a.translateSource != desc {
			// Another card is current now; the finished translation belongs to
			// the requested source and stays only in the file cache.
			return
		}
		a.translateResult = text
		if err != nil {
			a.translateState = "error"
			a.translateError = err.Error()
		} else {
			a.translateState = "done"
			a.translateError = ""
		}
		invalidateWindow(a.hwnd)
		postAppMessage(a.hwnd, wmRMLTranslateDone, 0, 0)
	}()
	return "", "busy", true
}

func (a *application) restartTranslate(desc string) {
	if desc == "" {
		return
	}
	translator := defaultTranslator(a.knowledgeRoot)
	translationDrop(translator.knowledgeRoot, desc)
	a.translateMu.Lock()
	a.translateGen++
	a.translateSource = ""
	a.translateState = ""
	a.translateResult = ""
	a.translateError = ""
	a.translateMu.Unlock()
	a.ensureTranslated(desc)
}

func (a *application) editCatalogSearch(r rune) {
	const backspace = 8
	const enter = 13
	const escape = 27
	switch r {
	case enter:
		a.catalogSearchFocus = false
		a.catalogPage = 1
		a.startCatalogFetch()
		invalidateWindow(a.hwnd)
		return
	case escape:
		a.catalogSearchFocus = false
		invalidateWindow(a.hwnd)
		return
	case backspace:
		if len(a.catalogSearch) > 0 {
			a.catalogSearch = a.catalogSearch[:len(a.catalogSearch)-1]
		}
		invalidateWindow(a.hwnd)
		return
	}
	if r < 0x20 {
		return
	}
	if len(a.catalogSearch) >= 96 {
		return
	}
	a.catalogSearch += string(r)
	invalidateWindow(a.hwnd)
}

func (a *application) scrollActiveCatalogDetail(delta int) {
	// Positive delta = scroll up a line (content moves down); clamp inside
	// the description viewport via the detail scroll helper below.
	step := delta * 60
	a.catalogDetailScroll = maxInt(0, a.catalogDetailScroll+step)
}

func scrollActiveCatalog(a *application, delta int) {
	a.scrollActiveCatalogDetail(delta)
}

/* Drawing --------------------------------------------------------------------------- */

func drawCatalogSearchBox(c *canvas, x, y, w, h int) {
	text := c.app.catalogSearch
	active := c.app.catalogSearchFocus
	fill, border := palette.Background, palette.Border
	if c.app.hover == "catalog-search" || active {
		fill, border = palette.SurfaceRaised, palette.BorderBright
	}
	c.roundedRect(x, y, w, h, c.s(8), fill, border)
	display := text
	color := palette.Text
	if display == "" {
		display, color = tr("catalog.search.placeholder"), palette.TextFaint
	}
	if active && text != "" {
		display += "  │"
	}
	clearW := c.s(28)
	c.text(display, x+c.s(11), y, w-clearW-c.s(8), h, fonts.Small, color, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.addHit("catalog-search", x, y, w, h)
	if text != "" {
		cx := x + w - clearW
		c.text("×", cx, y, clearW, h, fonts.BodyStrong, palette.TextMuted, dtCenter|dtVCenter|dtSingleLine)
		c.addHit("catalog-search-clear", cx, y, clearW, h)
	}
}

func catalogChipEncoded(tag string) string {
	return strings.ReplaceAll(tag, ":", ";")
}

func catalogChipDecoded(tag string) string {
	return strings.ReplaceAll(tag, ";", ":")
}

func drawCatalogPage(c *canvas, x, y, w, h int) {
	app := c.app
	cursor := c.sectionTitle(tr("catalog.title"), tr("catalog.subtitle"), tr("catalog.title.detail"), x, y, w)
	if !app.cfg.TranslationNoticeShown {
		noticeH := c.s(38)
		c.roundedRect(x, cursor, w, noticeH, c.s(9), palette.WarnSoft, palette.Warn)
		c.text(tr("catalog.translate.notice"), x+c.s(16), cursor, w-c.s(160)-c.s(32), noticeH, fonts.Small, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.compactButton("catalog-translate-notice-ok", tr("catalog.translate.notice_ok"), x+w-c.s(150), cursor, c.s(134), noticeH-c.s(6)-c.s(4))
		c.addHit("catalog-translate-notice-ok", x+w-c.s(150), cursor, c.s(134), noticeH-c.s(6)-c.s(4))
		cursor += noticeH + c.s(10)
	}
	_ = cursor

	filtersTop := cursor + c.s(6)
	gap := c.s(10)
	searchW := minInt(c.s(320), w*2/5)
	drawCatalogSearchBox(c, x, filtersTop, searchW, c.s(38))

	// sort + period
	sortX := x + searchW + gap
	sortW := maxInt(c.s(170), w-c.s(440)-gap*4)
	c.compactButton("catalog-sort", fmt.Sprintf("%s: %s", tr("catalog.filters.sort"), catalogSortLabelFor(app.cfg.CatalogSort)), sortX, filtersTop, sortW, c.s(38))
	c.addHit("catalog-sort", sortX, filtersTop, sortW, c.s(38))
	periodX := sortX + sortW + gap
	if app.cfg.CatalogSort == "" || sortParameter(app.cfg.CatalogSort) == catalogSortTrend {
		c.compactButton("catalog-period", fmt.Sprintf("%s: %s", tr("catalog.filters.period"), catalogPeriodLabel(app.cfg.CatalogPeriod)), periodX, filtersTop, c.s(190), c.s(38))
		c.addHit("catalog-period", periodX, filtersTop, c.s(190), c.s(38))
	} else {
		// The period parameter only affects "trend" on Steam's side; hide it
		// for the other real sorts rather than pretending it is in effect.
		c.text(tr("catalog.filters.period_hidden"), periodX, filtersTop, c.s(190), c.s(38), fonts.Tiny, palette.TextFaint, dtLeft|dtVCenter|dtSingleLine)
	}

	cursor = filtersTop + c.s(52)
	// Tag chips wrap into up to 3 rows.
	chipY := cursor
	chipX := x
	chipH := c.s(30)
	for _, tag := range catalogPrimaryTags {
		label := tag
		for _, sel := range app.catalogTags {
			if sel == tag {
				label = "• " + tag
				break
			}
		}
		width := c.textWidth(label, fonts.Small) + c.s(28)
		if chipX+width > x+w {
			chipX = x
			chipY += chipH + c.s(6)
		}
		c.compactButton("catalog-tag:"+catalogChipEncoded(tag), label, chipX, chipY, width, chipH)
		c.addHit("catalog-tag:"+catalogChipEncoded(tag), chipX, chipY, width, chipH)
		chipX += width + c.s(8)
	}
	if chipX != x {
		chipY += chipH + c.s(10)
	} else {
		chipY += c.s(4)
	}
	// Third row of filters.
	c.compactButton("catalog-show-incompatible", tr("catalog.filters.incompatible"), x, chipY, c.s(250), c.s(32))
	c.addHit("catalog-show-incompatible", x, chipY, c.s(250), c.s(32))
	compatLabel := tr("t.off")
	if app.cfg.CatalogShowIncompatible {
		compatLabel = tr("t.on")
	}
	c.text(compatLabel, x+c.s(258), chipY, c.s(40), c.s(32), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)
	c.compactButton("catalog-reset", tr("catalog.filters.reset"), x+w-c.s(150), chipY, c.s(150), c.s(32))
	c.addHit("catalog-reset", x+w-c.s(150), chipY, c.s(150), c.s(32))

	cursor = chipY + c.s(44)
	if app.catalogBusy {
		c.text(tr("catalog.state.loading"), x, cursor, w, c.s(26), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)
	} else if app.catalogOffline {
		c.text(tr("catalog.state.offline"), x, cursor, w, c.s(26), fonts.Tiny, palette.Warn, dtLeft|dtVCenter|dtSingleLine)
	} else if len(app.catalogIDs) == 0 {
		c.text(tr("catalog.state.empty"), x, cursor, w, c.s(26), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)
	}

	// Split the remainder between the list (left) and the details (right).
	listW := int(float64(w-gap) * 0.42)
	detailX := x + listW + gap
	detailW := w - listW - gap
	listTop := cursor + c.s(14)
	listBottom := y + h - c.s(76)
	app.catalogListX, app.catalogListY, app.catalogListW, app.catalogListH = x, listTop, listW, maxInt(c.s(120), listBottom-listTop)
	c.card(x, listTop, listW, app.catalogListH)
	drawCatalogListColumn(c, x, listTop, listW, app.catalogListH)

	c.card(detailX, listTop, detailW, app.catalogListH)
	drawCatalogDetailPane(c, detailX, listTop, detailW, maxInt(c.s(120), listBottom-listTop))

	// Pagination at the bottom edge.
	pageY := y + h - c.s(64)
	c.compactButton("catalog-prev", tr("catalog.page.prev"), x, pageY, c.s(96), c.s(38))
	c.addHit("catalog-prev", x, pageY, c.s(96), c.s(38))
	pages := catalogPagesTotal(app.catalogTotal)
	text := fmt.Sprintf("%s %d / %d (%s)", tr("catalog.page.label"), app.catalogPage, pages, fmt.Sprintf(tr("t.d_shown_d_total"), len(app.catalogIDs), app.catalogTotal))
	c.text(text, x+c.s(110), pageY, w-c.s(260), c.s(38), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)
	c.compactButton("catalog-next", tr("catalog.page.next"), x+w-c.s(96), pageY, c.s(96), c.s(38))
	c.addHit("catalog-next", x+w-c.s(96), pageY, c.s(96), c.s(38))
}

func catalogPagesTotal(total int) int {
	if total <= 0 {
		return 1
	}
	pages := total / 30
	if total%30 != 0 {
		pages++
	}
	return maxInt(1, pages)
}

func drawCatalogListColumn(c *canvas, x, y, w, h int) {
	app := c.app
	rowH := c.s(64)
	visible := maxInt(1, (h-c.s(12))/rowH)
	offset := clampListScroll(app.catalogHelpScroll, len(app.catalogIDs), visible)
	app.catalogHelpScroll = offset
	// Column header
	c.text(tr("catalog.list.title"), x+c.s(16), y+c.s(6), w-c.s(32), c.s(24), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)
	for visual := 0; visual < visible; visual++ {
		i := offset + visual
		if i >= len(app.catalogIDs) {
			break
		}
		item, ok := app.catalogItems[app.catalogIDs[i]]
		ry := y + c.s(34) + visual*rowH
		id := fmt.Sprintf("catalog-row:%d", i)
		selected := i == app.catalogSel
		fill := palette.SurfaceRaised
		if selected {
			fill = palette.AccentSoft
		} else if app.hover == id {
			fill = palette.SurfaceHover
		}
		c.roundedRect(x+c.s(6), ry, w-c.s(12), rowH-c.s(4), c.s(8), fill, fill)
		title := tr("t.workshop_item") + app.catalogIDs[i]
		if ok {
			title = catalogCatalogTitle(item)
		}
		c.text(title, x+c.s(18), ry+c.s(4), w-c.s(36), c.s(26), fonts.BodyStrong, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		second := tr("catalog.detail.id") + app.catalogIDs[i]
		if ok {
			second = fmt.Sprintf(tr("catalog.list.meta"), item.Subscriptions, item.Views, formatCatalogSize(item.FileSize))
		}
		c.text(second, x+c.s(18), ry+c.s(30), w-c.s(36), c.s(24), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.addHit(id, x+c.s(6), ry, w-c.s(12), rowH-c.s(4))
	}
}

func catalogDetailScrollFor(offset int) int { return offset }

func drawCatalogDetailPane(c *canvas, x, y, w, h int) {
	item, ok := c.app.selectedCatalogItem()
	if !ok {
		c.text(tr("catalog.detail.empty"), x+c.s(20), y+c.s(20), w-c.s(40), c.s(40), fonts.Small, palette.TextMuted, dtLeft|dtWordBreak)
		return
	}
	title := catalogCatalogTitle(item)
	c.text(title, x+c.s(20), y+c.s(16), w-c.s(40), c.s(34), fonts.Heading, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	metaLine := fmt.Sprintf(tr("catalog.list.meta"), item.Subscriptions, item.Views, formatCatalogSize(item.FileSize))
	c.text(metaLine, x+c.s(20), y+c.s(52), w-c.s(40), c.s(24), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)

	// Buttons
	bottomY := y + h - c.s(46)
	installed := c.app.catalogItemInstalled(item.ID)
	if installed {
		c.pill(tr("catalog.state.installed"), 0, x+c.s(20), bottomY, c.s(180), c.s(38))
		c.addHit("catalog-subscribe-disabled", x+c.s(20), bottomY, c.s(180), c.s(38))
		c.compactButton("catalog-unsubscribe", tr("catalog.unsubscribe.btn"), x+c.s(212), bottomY, c.s(160), c.s(38))
		c.addHit("catalog-unsubscribe", x+c.s(212), bottomY, c.s(160), c.s(38))
	} else {
		c.button("catalog-subscribe", tr("catalog.subscribe.btn"), x+c.s(20), bottomY, c.s(180), c.s(38), true)
		c.addHit("catalog-subscribe", x+c.s(20), bottomY, c.s(180), c.s(38))
	}
	c.compactButton("catalog-open-page", tr("t.open_workshop"), x+c.s(384), bottomY, c.s(180), c.s(38))
	c.addHit("catalog-open-page", x+c.s(384), bottomY, c.s(180), c.s(38))

	descriptionY := y + c.s(88)
	descriptionH := bottomY - descriptionY - c.s(12)
	if descriptionH < c.s(60) {
		descriptionH = c.s(60)
	}
	// translation header row
	chipH := c.s(30)
	_, _, translateAvailable := c.app.ensureTranslated(item.Description)
	if translateAvailable {
		c.compactButton("catalog-show-original", tr("catalog.detail.original"), x+c.s(20), descriptionY, c.s(120), chipH)
		c.addHit("catalog-show-original", x+c.s(20), descriptionY, c.s(120), chipH)
		c.compactButton("catalog-show-translated", tr("catalog.detail.translated"), x+c.s(148), descriptionY, c.s(120), chipH)
		c.addHit("catalog-show-translated", x+c.s(148), descriptionY, c.s(120), chipH)
		state, _, _ := c.app.describeTranslate(item.Description)
		if state == "busy" {
			c.text(tr("catalog.detail.translate_busy"), x+c.s(284), descriptionY, w-c.s(304), chipH, fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)
		} else if state == "error" {
			c.text(tr("catalog.detail.translate_failed")+c.app.translateError, x+c.s(284), descriptionY, w-c.s(304), chipH, fonts.Tiny, palette.Bad, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		}
	c.compactButton("catalog-refresh-translate", tr("catalog.detail.refresh"), x+c.s(284), descriptionY, c.s(130), chipH)
		c.addHit("catalog-refresh-translate", x+c.s(284), descriptionY, c.s(130), chipH)
		text := item.Description
		if !c.app.catalogShowOriginal && c.app.translateState == "done" && c.app.translateResult != "" {
			text = c.app.translateResult
		}
		drawScrollableDescription(c, 3, text, x+c.s(20), descriptionY+chipH+c.s(6), w-c.s(40), maxInt(c.s(60), descriptionH-chipH-c.s(6)), fonts.Small, palette.Text)
		return
	}
	drawScrollableDescription(c, 3, item.Description, x+c.s(20), descriptionY, w-c.s(40), descriptionH, fonts.Small, palette.Text)
}

func (a *application) describeTranslate(desc string) (string, string, bool) {
	_, state, available := a.ensureTranslated(desc)
	return a.translateResult, state, available
}

func formatCatalogSize(size int64) string {
	if size <= 0 {
		return "—"
	}
	const mib = 1024 * 1024
	if size < mib {
		return fmt.Sprintf(tr("catalog.size.kb"), size/1024)
	}
	value := float64(size) / float64(mib)
	return fmt.Sprintf(tr("catalog.size.mb"), value)
}

