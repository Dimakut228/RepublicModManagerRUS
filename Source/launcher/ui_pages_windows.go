//go:build windows

package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

var navItems = []struct{ LabelKey, HintKey string }{
	{LabelKey: "nav.dashboard", HintKey: "nav.dashboard.hint"},
	{LabelKey: "nav.mods", HintKey: "nav.mods.hint"},
	{LabelKey: "nav.diagnostics", HintKey: "nav.diagnostics.hint"},
}

func renderApplication(app *application, dc uintptr, width, height int) {
	ensureFonts(app.dpi)
	app.hits = app.hits[:0]
	c := &canvas{dc: dc, app: app, width: width, height: height, scale: app.scale}
	c.fillRect(0, 0, width, height, palette.Background)

	sidebar := c.s(222)
	statusH := c.s(42)
	drawSidebar(c, sidebar, height-statusH)

	contentX := sidebar + c.s(30)
	contentY := c.s(22)
	contentW := width - contentX - c.s(30)
	contentH := height - contentY - statusH - c.s(10)
	if contentW < c.s(600) {
		contentW = c.s(600)
	}

	app.tooltipText = ""
	switch app.page {
	case 1:
		drawPluginsPage(c, contentX, contentY, contentW, contentH)
	case 2:
		drawDiagnosticsPage(c, contentX, contentY, contentW, contentH)
	default:
		drawDashboardPage(c, contentX, contentY, contentW, contentH)
	}
	// Page-owned overlays render after page content so dropdowns/tooltips are not
	// painted over by the table/detail panels beneath them.
	if (app.page == 0 || app.page == 1) && app.playlistDropdown && !app.playlistModal && !app.playlistDeleteModal && !app.playlistImportModal && !app.workshopUnsubscribeModal && !app.workshopForceUpdateModal {
		drawPlaylistDropdownOverlay(c)
	}
	drawStatusBar(c, sidebar, height-statusH, width-sidebar, statusH)
	if app.hover == "copy-build-sha" && app.page == 0 && !app.playlistModal && !app.playlistDeleteModal && !app.playlistImportModal && !app.workshopUnsubscribeModal && !app.workshopForceUpdateModal {
		drawBuildSHATooltip(c)
	}
	if app.playlistModal {
		drawPlaylistModal(c)
	}
	if app.playlistDeleteModal {
		drawPlaylistDeleteModal(c)
	}
	if app.playlistImportModal {
		drawPlaylistImportModal(c)
	}
	if app.workshopUnsubscribeModal {
		drawWorkshopUnsubscribeModal(c)
	}
	if app.workshopForceUpdateModal {
		drawWorkshopForceUpdateModal(c)
	}
	if app.tooltipText != "" {
		drawGenericTooltip(c, app.tooltipText)
	}
	if app.toastText != "" && time.Now().Before(app.toastUntil) {
		drawToast(c, app.toastText)
	}
}

func drawPlaylistModal(c *canvas) {
	// Dim the application behind a compact launcher-owned text-entry dialog.
	c.fillRect(0, 0, c.width, c.height, rgb(8, 11, 16))
	w, h := c.s(520), c.s(244)
	x, y := (c.width-w)/2, (c.height-h)/2
	c.roundedRect(x, y, w, h, c.s(16), palette.Surface, palette.BorderBright)
	c.label(tr("t.plugin_playlist"), x+c.s(24), y+c.s(20), w-c.s(48), palette.AccentBright)
	title := tr("t.save_current_configuration")
	detail := tr("t.stores_enabled_mods_and")
	button := tr("t.save_playlist")
	if c.app.playlistModalMode == "rename" {
		title = tr("t.rename_playlist")
		detail = tr("t.choose_a_new_name")
		button = tr("t.rename")
	}
	c.text(title, x+c.s(24), y+c.s(43), w-c.s(48), c.s(34), fonts.Heading, palette.Text, dtLeft|dtVCenter|dtSingleLine)
	c.text(detail, x+c.s(24), y+c.s(81), w-c.s(48), c.s(42), fonts.Small, palette.TextMuted, dtLeft|dtWordBreak)
	fieldY := y + c.s(128)
	c.roundedRect(x+c.s(24), fieldY, w-c.s(48), c.s(42), c.s(9), palette.SurfaceRaised, palette.BorderBright)
	draft := c.app.playlistDraft
	if draft == "" {
		draft = tr("t.type_a_playlist_name")
	}
	c.text(draft+map[bool]string{true: "  │", false: ""}[c.app.playlistDraft != ""], x+c.s(36), fieldY, w-c.s(72), c.s(42), fonts.Body, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.compactButton("playlist-modal-cancel", tr("t.cancel"), x+w-c.s(212), y+h-c.s(54), c.s(88), c.s(34))
	c.button("playlist-modal-save", button, x+w-c.s(116), y+h-c.s(54), c.s(92), c.s(34), true)
}

func drawPlaylistImportModal(c *canvas) {
	c.fillRect(0, 0, c.width, c.height, rgb(8, 11, 16))
	w, h := c.s(540), c.s(250)
	x, y := (c.width-w)/2, (c.height-h)/2
	c.roundedRect(x, y, w, h, c.s(16), palette.Surface, palette.BorderBright)
	c.label(tr("t.import_playlist"), x+c.s(24), y+c.s(20), w-c.s(48), palette.AccentBright)
	c.text(tr("t.import_a_republic_mod"), x+c.s(24), y+c.s(46), w-c.s(48), c.s(34), fonts.Heading, palette.Text, dtLeft|dtVCenter|dtSingleLine)
	dropY := y + c.s(92)
	c.roundedRect(x+c.s(24), dropY, w-c.s(48), c.s(82), c.s(10), palette.SurfaceRaised, palette.BorderBright)
	c.text(tr("t.drag_and_drop_a"), x+c.s(36), dropY+c.s(10), w-c.s(72), c.s(28), fonts.BodyStrong, palette.Text, dtCenter|dtVCenter|dtSingleLine)
	c.text(tr("t.or_choose_a_file"), x+c.s(36), dropY+c.s(40), w-c.s(72), c.s(24), fonts.Small, palette.TextMuted, dtCenter|dtVCenter|dtSingleLine)
	c.compactButton("playlist-import-cancel", tr("t.cancel"), x+w-c.s(206), y+h-c.s(52), c.s(82), c.s(34))
	c.button("playlist-import-browse", tr("t.browse"), x+w-c.s(116), y+h-c.s(52), c.s(92), c.s(34), true)
}

func drawPlaylistDeleteModal(c *canvas) {
	c.fillRect(0, 0, c.width, c.height, rgb(8, 11, 16))
	w, h := c.s(470), c.s(210)
	x, y := (c.width-w)/2, (c.height-h)/2
	c.roundedRect(x, y, w, h, c.s(16), palette.Surface, palette.BorderBright)
	c.label(tr("t.delete_playlist"), x+c.s(24), y+c.s(20), w-c.s(48), palette.Bad)
	name := c.app.playlistDeleteName
	c.text(tr("t.delete_prefix")+name+"\"?", x+c.s(24), y+c.s(49), w-c.s(48), c.s(34), fonts.Heading, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(tr("t.this_removes_only_the"), x+c.s(24), y+c.s(88), w-c.s(48), c.s(43), fonts.Small, palette.TextMuted, dtLeft|dtWordBreak)
	c.compactButton("playlist-delete-cancel", tr("t.cancel"), x+w-c.s(210), y+h-c.s(52), c.s(84), c.s(34))
	// Draw a red confirmation button rather than reusing the primary accent.
	bx, by, bw, bh := x+w-c.s(116), y+h-c.s(52), c.s(92), c.s(34)
	fill := palette.BadSoft
	if c.app.hover == "playlist-delete-confirm" {
		fill = palette.Bad
	}
	c.roundedRect(bx, by, bw, bh, c.s(8), fill, palette.Bad)
	c.text(tr("t.delete"), bx+c.s(8), by, bw-c.s(16), bh, fonts.SmallStrong, palette.Bad, dtCenter|dtVCenter|dtSingleLine)
	if c.app.hover == "playlist-delete-confirm" {
		c.text(tr("t.delete"), bx+c.s(8), by, bw-c.s(16), bh, fonts.SmallStrong, palette.White, dtCenter|dtVCenter|dtSingleLine)
	}
	c.addHit("playlist-delete-confirm", bx, by, bw, bh)
}

func drawWorkshopUnsubscribeModal(c *canvas) {
	c.fillRect(0, 0, c.width, c.height, rgb(8, 11, 16))
	w, h := c.s(510), c.s(220)
	x, y := (c.width-w)/2, (c.height-h)/2
	c.roundedRect(x, y, w, h, c.s(16), palette.Surface, palette.BorderBright)
	c.label(tr("t.steam_workshop"), x+c.s(24), y+c.s(20), w-c.s(48), palette.Bad)
	c.text(tr("t.unsubscribe_from_this_mod"), x+c.s(24), y+c.s(47), w-c.s(48), c.s(34), fonts.Heading, palette.Text, dtLeft|dtVCenter|dtSingleLine)
	name := c.app.workshopUnsubscribeName
	id := c.app.workshopUnsubscribeID
	c.text(name, x+c.s(24), y+c.s(83), w-c.s(48), c.s(25), fonts.BodyStrong, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(tr("t.workshop_id")+id, x+c.s(24), y+c.s(108), w-c.s(48), c.s(22), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)
	c.text(tr("t.steam_will_unsubscribe_the"), x+c.s(24), y+c.s(133), w-c.s(48), c.s(32), fonts.Small, palette.TextMuted, dtLeft|dtWordBreak)
	c.compactButton("workshop-unsubscribe-cancel", tr("t.cancel"), x+w-c.s(214), y+h-c.s(52), c.s(86), c.s(34))
	c.dangerButton("workshop-unsubscribe-confirm", tr("t.unsubscribe"), x+w-c.s(120), y+h-c.s(52), c.s(96), c.s(34), true)
}

func drawBuildSHATooltip(c *canvas) {
	if c.app.mouseX < 0 || c.app.mouseY < 0 {
		return
	}
	// Keep the tooltip anchored just below the pointer.
	x, y := c.app.mouseX+c.s(1), c.app.mouseY+c.s(1)
	w, h := c.s(90), c.s(23)
	if x+w > c.width-c.s(4) {
		x = c.width - w - c.s(4)
	}
	if y+h > c.height-c.s(4) {
		y = c.height - h - c.s(4)
	}
	c.roundedRect(x, y, w, h, c.s(5), rgb(24, 31, 41), palette.BorderBright)
	c.text(tr("tooltip.copy_build_sha"), x+c.s(6), y, w-c.s(12), h, fonts.Tiny, palette.Text, dtLeft|dtVCenter|dtSingleLine)
}

func drawGenericTooltip(c *canvas, text string) {
	if text == "" || c.app.mouseX < 0 || c.app.mouseY < 0 {
		return
	}
	pad := c.s(8)
	w := c.textWidth(text, fonts.Small) + pad*2
	maxW := maxInt(c.s(140), c.width-c.s(10))
	if w > maxW {
		w = maxW
	}
	h := c.s(25)
	x, y := c.app.mouseX+c.s(2), c.app.mouseY+c.s(2)
	if x+w > c.width-c.s(4) {
		x = c.width - w - c.s(4)
	}
	if y+h > c.height-c.s(4) {
		y = c.height - h - c.s(4)
	}
	c.roundedRect(x, y, w, h, c.s(5), rgb(24, 31, 41), palette.BorderBright)
	c.text(text, x+pad, y, w-pad*2, h, fonts.Small, palette.Text, dtLeft|dtVCenter|dtSingleLine)
}

func drawToast(c *canvas, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	pad := c.s(12)
	w := c.textWidth(text, fonts.SmallStrong) + pad*2
	if w > c.s(430) {
		w = c.s(430)
	}
	if w < c.s(220) {
		w = c.s(220)
	}
	h := c.s(42)
	x := c.width - w - c.s(18)
	y := c.height - h - c.s(58)
	c.roundedRect(x, y, w, h, c.s(9), palette.SurfaceRaised, palette.BorderBright)
	c.text(text, x+pad, y, w-pad*2, h, fonts.SmallStrong, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
}

func drawPlaylistDropdownOverlay(c *canvas) {
	names := c.app.playlistNames()
	if len(names) == 0 || c.app.playlistDropW <= 0 {
		return
	}
	x, y, w := c.app.playlistDropX, c.app.playlistDropY, c.app.playlistDropW
	rowH := c.s(30)
	// Default is pinned to the first row. Saved playlists below it are scrollable
	// so the dropdown remains usable even with the configured 64-playlist limit.
	available := c.height - y - c.s(58)
	maxRows := maxInt(2, available/rowH)
	visibleSaved := minInt(maxRows-1, maxInt(0, len(names)-1))
	maxScroll := maxInt(0, len(names)-1-visibleSaved)
	if c.app.playlistScroll > maxScroll {
		c.app.playlistScroll = maxScroll
	}
	if c.app.playlistScroll < 0 {
		c.app.playlistScroll = 0
	}
	rows := 1 + visibleSaved
	h := rows*rowH + c.s(8)
	c.roundedRect(x, y, w, h, c.s(9), palette.SurfaceRaised, palette.BorderBright)

	drawRow := func(optionIndex, visualRow int, name string) {
		ry := y + c.s(4) + visualRow*rowH
		id := fmt.Sprintf("playlist-option:%d", optionIndex)
		selected := strings.EqualFold(name, c.app.cfg.SelectedPlaylist)
		hovered := c.app.hover == id
		fill := palette.SurfaceRaised
		if selected {
			fill = palette.AccentSoft
		} else if hovered {
			fill = palette.SurfaceHover
		}
		c.roundedRect(x+c.s(4), ry, w-c.s(8), rowH-c.s(2), c.s(6), fill, fill)
		if selected {
			c.circle(x+c.s(12), ry+c.s(10), c.s(8), palette.AccentBright, palette.AccentBright)
		}
		c.text(name, x+c.s(28), ry, w-c.s(43), rowH-c.s(2), fonts.Small, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.addHit(id, x+c.s(4), ry, w-c.s(8), rowH-c.s(2))
	}
	drawRow(0, 0, names[0])
	for i := 0; i < visibleSaved; i++ {
		global := 1 + c.app.playlistScroll + i
		if global >= len(names) {
			break
		}
		drawRow(global, i+1, names[global])
	}
	if len(names)-1 > visibleSaved {
		trackX := x + w - c.s(8)
		trackY := y + c.s(4) + rowH
		trackH := visibleSaved*rowH - c.s(4)
		c.fillRect(trackX, trackY, c.s(2), trackH, palette.Border)
		totalSaved := len(names) - 1
		thumbH := maxInt(c.s(14), trackH*visibleSaved/totalSaved)
		thumbY := trackY
		if maxScroll > 0 {
			thumbY += (trackH - thumbH) * c.app.playlistScroll / maxScroll
		}
		c.fillRect(trackX, thumbY, c.s(2), thumbH, palette.TextFaint)
	}
}

func drawSidebar(c *canvas, w, h int) {
	c.fillRect(0, 0, w, h, palette.Sidebar)
	c.line(w-c.s(1), 0, w-c.s(1), h, c.s(1), palette.Border)
	c.drawBrand(c.s(25), c.s(25))
	c.text(tr("brand.line1"), c.s(80), c.s(22), w-c.s(96), c.s(24), fonts.Title, palette.Text, dtLeft|dtVCenter|dtSingleLine)
	c.text(tr("brand.line2"), c.s(80), c.s(45), w-c.s(96), c.s(18), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)

	y := c.s(105)
	for i, item := range navItems {
		id := fmt.Sprintf("nav:%d", i)
		selected := c.app.page == i
		hovered := c.app.hover == id
		fill := palette.Sidebar
		border := palette.Sidebar
		if selected {
			fill, border = palette.AccentSoft, palette.AccentSoft
		}
		if hovered && !selected {
			fill, border = palette.Surface, palette.Border
		}
		x := c.s(14)
		rowW := w - c.s(28)
		rowH := c.s(55)
		c.roundedRect(x, y, rowW, rowH, c.s(11), fill, border)
		if selected {
			c.roundedRect(x, y+c.s(9), c.s(4), rowH-c.s(18), c.s(4), palette.AccentBright, palette.AccentBright)
		}
		iconFill := palette.SurfaceRaised
		iconText := palette.TextMuted
		if selected {
			iconFill, iconText = palette.Accent, palette.White
		}
		label := tr(item.LabelKey)
		hint := tr(item.HintKey)
		iconChar, _ := utf8.DecodeRuneInString(label)
		c.circle(x+c.s(15), y+c.s(13), c.s(29), iconFill, iconFill)
		c.text(string(iconChar), x+c.s(15), y+c.s(13), c.s(29), c.s(29), fonts.SmallStrong, iconText, dtCenter|dtVCenter|dtSingleLine)
		c.text(label, x+c.s(55), y+c.s(8), rowW-c.s(66), c.s(24), fonts.BodyStrong, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.text(hint, x+c.s(55), y+c.s(29), rowW-c.s(66), c.s(18), fonts.Tiny, palette.TextFaint, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.addHit(id, x, y, rowW, rowH)
		y += c.s(62)
	}

}

var statusFailureKeywords = []string{
	"fail", "cannot", "error", "abort", "invalid",
	"ошибк", "не удал", "невозможн", "прерва", "сбой", "провал",
}

func statusIndicatesFailure(status string) bool {
	lower := strings.ToLower(status)
	for _, keyword := range statusFailureKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func drawStatusBar(c *canvas, x, y, w, h int) {	c.fillRect(x, y, w, h, palette.Sidebar)
	c.line(x, y, x+w, y, c.s(1), palette.Border)
	tx := x + c.s(24)
	if c.app.busy {
		c.busyIndicator(tx, y+c.s(16))
		tx += c.s(62)
	} else {
		color := palette.Good
		if statusIndicatesFailure(c.app.status) {
			color = palette.Bad
		}
		c.circle(tx, y+c.s(16), c.s(9), color, color)
		tx += c.s(18)
	}
	c.text(c.app.status, tx, y, w-c.s(300), h, fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	right := fmt.Sprintf("RML %s  •  API %d", launcherVersion, loaderAPI)
	c.text(right, x+w-c.s(255), y, c.s(230), h, fonts.Tiny, palette.TextFaint, dtRight|dtVCenter|dtSingleLine)
}

func drawWorkshopForceUpdateModal(c *canvas) {
	c.fillRect(0, 0, c.width, c.height, rgb(8, 11, 16))
	w, h := c.s(560), c.s(268)
	x, y := (c.width-w)/2, (c.height-h)/2
	c.roundedRect(x, y, w, h, c.s(16), palette.Surface, palette.BorderBright)
	c.label(tr("t.workshop_maintenance"), x+c.s(24), y+c.s(20), w-c.s(48), palette.Warn)
	c.text(tr("t.force_update_this_mod"), x+c.s(24), y+c.s(47), w-c.s(48), c.s(34), fonts.Heading, palette.Text, dtLeft|dtVCenter|dtSingleLine)
	name := c.app.workshopForceUpdateName
	id := c.app.workshopForceUpdateID
	c.text(name, x+c.s(24), y+c.s(83), w-c.s(48), c.s(25), fonts.BodyStrong, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(tr("t.workshop_id")+id, x+c.s(24), y+c.s(108), w-c.s(48), c.s(22), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)
	c.text(tr("t.rml_will_unsubscribe_the"), x+c.s(24), y+c.s(136), w-c.s(48), c.s(64), fonts.Small, palette.TextMuted, dtLeft|dtWordBreak)
	c.compactButton("workshop-force-update-cancel", tr("t.cancel"), x+w-c.s(220), y+h-c.s(52), c.s(90), c.s(34))
	bx, by, bw, bh := x+w-c.s(122), y+h-c.s(52), c.s(98), c.s(34)
	fill := palette.WarnSoft
	if c.app.hover == "workshop-force-update-confirm" {
		fill = palette.Warn
	}
	c.roundedRect(bx, by, bw, bh, c.s(8), fill, palette.Warn)
	fg := palette.Warn
	if c.app.hover == "workshop-force-update-confirm" {
		fg = palette.White
	}
	c.text(tr("t.force_update"), bx+c.s(6), by, bw-c.s(12), bh, fonts.SmallStrong, fg, dtCenter|dtVCenter|dtSingleLine)
	c.addHit("workshop-force-update-confirm", bx, by, bw, bh)
}

func drawDashboardPage(c *canvas, x, y, w, h int) {
	cursor := c.sectionTitle(tr("t.central_committee"), "Republic Mod Loader", "", x, y, w)

	gameH := c.s(168)
	c.card(x, cursor, w, gameH)
	gameName := tr("game.name")
	buildVersion := tr("dashboard.build_version.none")
	buildSHA := tr("dashboard.build_sha.none")
	gameClass := 2
	if c.app.game != nil {
		if c.app.game.Version != "" {
			buildVersion = tr("dashboard.build_version") + c.app.game.Version
		} else {
			buildVersion = tr("dashboard.build_version.untested")
		}
		if c.app.game.SHA256 != "" {
			buildSHA = tr("dashboard.build_sha") + c.app.game.SHA256
		}
		if c.app.game.Known {
			gameClass = 0
		} else {
			gameClass = 1
		}
	} else if c.app.gameError != "" {
		buildSHA = tr("dashboard.build_sha") + safeLine(c.app.gameError)
	}
	c.text(gameName, x+c.s(22), cursor+c.s(18), w-c.s(220), c.s(36), fonts.Heading, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(buildVersion, x+c.s(34), cursor+c.s(60), w-c.s(230), c.s(24), fonts.Body, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	shaColor := palette.TextMuted
	if c.app.hover == "copy-build-sha" && c.app.game != nil && c.app.game.SHA256 != "" {
		shaColor = palette.AccentBright
	}
	shaX := x + c.s(34)
	shaAvailable := w - c.s(68)
	c.text(buildSHA, shaX, cursor+c.s(91), shaAvailable, c.s(24), fonts.Body, shaColor, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	if c.app.game != nil && c.app.game.SHA256 != "" {
		shaTextW := minInt(shaAvailable, c.textWidth(buildSHA, fonts.Body))
		if shaTextW > 0 {
			c.addHit("copy-build-sha", shaX, cursor+c.s(91), shaTextW, c.s(24))
		}
	}
	fill, fg := statusColors(gameClass, "")
	c.roundedRect(x+w-c.s(176), cursor+c.s(22), c.s(145), c.s(32), c.s(16), fill, fill)
	gameState := tr("dashboard.state.needs_path")
	if c.app.game != nil && c.app.game.Known {
		gameState = tr("dashboard.state.recognised")
	}
	if c.app.game != nil && !c.app.game.Known {
		gameState = tr("dashboard.state.untested")
	}
	c.text(gameState, x+w-c.s(176), cursor+c.s(22), c.s(145), c.s(32), fonts.Tiny, fg, dtCenter|dtVCenter|dtSingleLine)
	c.compactButton("browse", tr("dashboard.choose_exe"), x+w-c.s(176), cursor+c.s(70), c.s(145), c.s(36))
	cursor += gameH + c.s(16)

	gap := c.s(14)
	metricW := (w - gap*2) / 3
	metricH := c.s(116)
	lastState := tr("dashboard.no_launch")
	lastDetail := ""
	lastClass := 1
	if c.app.lastLaunchError != "" {
		lastState, lastDetail, lastClass = tr("dashboard.launch_error"), safeLine(c.app.lastLaunchError), 2
	} else if c.app.report != nil {
		if c.app.report.BootstrapStatus == 0 {
			lastState, lastDetail, lastClass = tr("dashboard.loader_ok"), tr("dashboard.loader_ok_detail"), 0
		} else {
			lastState, lastDetail, lastClass = fmt.Sprintf(tr("t.0x_x"), c.app.report.BootstrapStatus), tr("dashboard.loader_problem"), 2
		}
	}
	c.metricCard(tr("dashboard.metric.discovered"), fmt.Sprintf(tr("dashboard.metric.discovered_value"), len(c.app.plugins), len(c.app.workshopItems)), "", x, cursor, metricW, metricH, 0)
	enabledWorkshop := 0
	enabledDevelopment := 0
	enabledWorkshopUpdates := 0
	for i := range c.app.workshopItems {
		item := &c.app.workshopItems[i]
		if !item.Enabled {
			continue
		}
		enabledWorkshop++
		if item.UpdateState == "update" || item.UpdateState == "stale" {
			enabledWorkshopUpdates++
		}
	}
	for i := range c.app.developmentItems {
		if c.app.developmentItems[i].Enabled {
			enabledDevelopment++
		}
	}
	enabledDetail := ""
	enabledClass := 0
	if enabledWorkshopUpdates > 0 {
		enabledDetail = fmt.Sprintf(tr("dashboard.metric.enabled_detail_update"), enabledWorkshopUpdates)
		enabledClass = 1
	} else if enabledDevelopment > 0 {
		enabledDetail = fmt.Sprintf(tr("dashboard.metric.enabled_detail_dev"), enabledDevelopment)
	}
	c.metricCard(tr("dashboard.metric.enabled"), fmt.Sprintf(tr("dashboard.metric.discovered_value"), c.app.enabledCount(), enabledWorkshop), enabledDetail, x+metricW+gap, cursor, metricW, metricH, enabledClass)
	c.metricCard(tr("dashboard.metric.last_launch"), lastState, lastDetail, x+(metricW+gap)*2, cursor, metricW, metricH, lastClass)
	cursor += metricH + c.s(16)

	lowerH := maxInt(c.s(168), h-(cursor-y)-c.s(94))
	leftW := int(float64(w-gap) * 0.58)
	rightW := w - gap - leftW
	drawDashboardPlaylistCard(c, x, cursor, leftW, lowerH)
	drawDashboardSettingsCard(c, x+leftW+gap, cursor, rightW, lowerH)

	launchY := y + h - c.s(72)
	c.roundedRect(x, launchY, w, c.s(58), c.s(14), palette.SurfaceRaised, palette.BorderBright)
	launchText := tr("dashboard.launch")
	if c.app.busy {
		launchText = tr("dashboard.launching")
	}
	buttonW := c.s(278)
	c.button("launch", launchText, x+w-buttonW-c.s(10), launchY+c.s(8), buttonW, c.s(42), true)
	state := tr("dashboard.ready")
	if ok, reason := c.app.canLaunch(); !ok {
		state = reason
	}
	c.text(state, x+c.s(20), launchY, c.s(maxInt(260, w-buttonW-c.s(50))), c.s(58), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
}

func drawDashboardPlaylistCard(c *canvas, x, y, w, h int) {
	c.card(x, y, w, h)
	c.label(tr("dashboard.playlist.label"), x+c.s(18), y+c.s(15), w-c.s(36), palette.TextMuted)
	name := c.app.cfg.SelectedPlaylist
	if name == "" {
		name = defaultPlaylistName
	}
	c.text(tr("dashboard.playlist.title"), x+c.s(18), y+c.s(39), w-c.s(36), c.s(25), fonts.Title, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

	dropX := x + c.s(18)
	dropY := y + c.s(76)
	dropW := w - c.s(36)
	dropH := c.s(38)
	fill, border := palette.SurfaceRaised, palette.Border
	if c.app.hover == "playlist-dropdown" || c.app.playlistDropdown {
		fill, border = palette.SurfaceHover, palette.BorderBright
	}
	c.roundedRect(dropX, dropY, dropW, dropH, c.s(8), fill, border)
	c.text(name, dropX+c.s(12), dropY, dropW-c.s(42), dropH, fonts.BodyStrong, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	cx, cy := dropX+dropW-c.s(19), dropY+dropH/2
	c.line(cx-c.s(4), cy-c.s(2), cx, cy+c.s(2), c.s(1), palette.TextMuted)
	c.line(cx, cy+c.s(2), cx+c.s(4), cy-c.s(2), c.s(1), palette.TextMuted)
	c.addHit("playlist-dropdown", dropX, dropY, dropW, dropH)
	c.app.playlistDropX, c.app.playlistDropY, c.app.playlistDropW = dropX, dropY+dropH+c.s(3), dropW

	mode := tr("dashboard.playlist.auto")
	if !c.app.cfg.AutoOrder {
		mode = tr("dashboard.playlist.manual")
	}
	detail := fmt.Sprintf(tr("dashboard.playlist.detail"), c.app.enabledCount(), mode)
	c.text(detail, x+c.s(18), dropY+c.s(50), w-c.s(36), c.s(30), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
}

func drawDashboardSettingsCard(c *canvas, x, y, w, h int) {
	c.card(x, y, w, h)
	c.label(tr("dashboard.settings.label"), x+c.s(18), y+c.s(15), w-c.s(36), palette.TextMuted)
	c.text(tr("dashboard.settings.title"), x+c.s(18), y+c.s(36), w-c.s(36), c.s(25), fonts.Title, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

	rows := []struct {
		Title string
		ID    string
		Value bool
	}{
		{tr("settings.verbose_diag"), "verbose-diagnostics", c.app.cfg.VerboseDiagnostics},
		{tr("settings.auto_crash"), "auto-crash-archives", c.app.cfg.AutoCrashArchives},
		{tr("settings.auto_order"), "auto-order", c.app.cfg.AutoOrder},
		{tr("settings.close_after"), "close-after", c.app.cfg.CloseAfterLaunch},
		{tr("settings.auto_hide"), "auto-hide", c.app.cfg.AutoHideOnLaunch},
	}
	contentY := y + c.s(65)
	available := maxInt(c.s(90), h-c.s(76))
	rowH := maxInt(c.s(23), available/len(rows))
	for i, row := range rows {
		ry := contentY + i*rowH
		if i > 0 {
			c.line(x+c.s(18), ry, x+w-c.s(18), ry, 1, palette.Border)
		}
		c.text(row.Title, x+c.s(18), ry, w-c.s(86), rowH, fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.toggle(row.ID, row.Value, x+w-c.s(60), ry+(rowH-c.s(23))/2)
	}
}

func drawModSearchBox(c *canvas, x, y, w, h int) {
	text := c.app.activeSearchText()
	active := c.app.modSearchFocus
	fill, border := palette.Background, palette.Border
	if c.app.hover == "mod-search" || active {
		fill, border = palette.SurfaceRaised, palette.BorderBright
	}
	c.roundedRect(x, y, w, h, c.s(8), fill, border)
	display := text
	color := palette.Text
	if display == "" {
		display, color = tr("t.search_name_id"), palette.TextFaint
	}
	if active && text != "" {
		display += "  │"
	}
	clearW := c.s(28)
	c.text(display, x+c.s(11), y, w-clearW-c.s(8), h, fonts.Small, color, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.addHit("mod-search", x, y, w, h)
	if text != "" {
		cx := x + w - clearW
		c.text("×", cx, y, clearW, h, fonts.BodyStrong, palette.TextMuted, dtCenter|dtVCenter|dtSingleLine)
		c.addHit("mod-search-clear", cx, y, clearW, h)
	}
}

func drawListScrollbar(c *canvas, prefix string, x, y, w, h, total, visible, offset int) {
	view := 0
	if prefix == "workshop" {
		view = 1
	} else if prefix == "development" {
		view = 2
	}
	c.app.modScrollView = view
	c.app.modScrollTrackH = h
	c.app.modScrollTotal = total
	c.app.modScrollVisible = visible
	c.app.modScrollThumbY = y
	c.app.modScrollThumbH = h
	if total <= visible || total <= 0 || h <= c.s(20) {
		return
	}
	c.roundedRect(x, y, w, h, w, palette.Background, palette.Background)
	thumbH := maxInt(c.s(28), int(float64(h)*float64(visible)/float64(total)))
	if thumbH > h {
		thumbH = h
	}
	maxOffset := total - visible
	travel := h - thumbH
	thumbY := y
	if maxOffset > 0 {
		thumbY += int(float64(travel) * float64(offset) / float64(maxOffset))
	}
	c.app.modScrollThumbY = thumbY
	c.app.modScrollThumbH = thumbH
	c.roundedRect(x, thumbY, w, thumbH, w, palette.BorderBright, palette.BorderBright)
	c.addHit(prefix+"-scroll-thumb", x-c.s(7), thumbY, w+c.s(14), thumbH)
	if thumbY > y {
		c.addHit(prefix+"-scroll-pageup", x-c.s(6), y, w+c.s(12), thumbY-y)
	}
	if thumbY+thumbH < y+h {
		c.addHit(prefix+"-scroll-pagedown", x-c.s(6), thumbY+thumbH, w+c.s(12), y+h-(thumbY+thumbH))
	}
}

func detailThumbnailWidth(c *canvas, panelW int) int {
	inner := maxInt(c.s(120), panelW-c.s(40))
	// Reserve enough room for the status pill and two short metadata lines. The
	// preview reaches the Workshop-style 190px width on normal panels but
	// shrinks rather than pushing metadata off-screen on a narrow window.
	return minInt(c.s(190), maxInt(c.s(88), inner-c.s(118)))
}

func drawScrollableDescription(c *canvas, view int, text string, x, y, w, h int, font uintptr, color uint32) {
	if w <= 0 || h <= 0 {
		return
	}
	barW := c.s(7)
	textW := w
	contentH := c.textHeight(text, maxInt(1, w-barW-c.s(6)), font)
	if contentH < h {
		contentH = h
	}
	offset := c.app.pluginDetailScroll
	if view == 1 {
		offset = c.app.workshopDetailScroll
	} else if view == 2 {
		offset = c.app.developmentDetailScroll
	}
	offset = clampPixelScroll(offset, contentH, h)
	if view == 1 {
		c.app.workshopDetailScroll = offset
	} else if view == 2 {
		c.app.developmentDetailScroll = offset
	} else {
		c.app.pluginDetailScroll = offset
	}
	c.app.detailScrollX, c.app.detailScrollY, c.app.detailScrollW, c.app.detailScrollH = x, y, w, h
	c.app.detailScrollTotal, c.app.detailScrollVisible, c.app.detailScrollView = contentH, h, view
	c.app.detailScrollThumbY, c.app.detailScrollThumbH = y, h
	c.clippedWrappedText(text, x, y, textW-barW-c.s(6), h, font, color, offset, contentH)
	if contentH <= h {
		return
	}
	trackX := x + w - barW
	c.roundedRect(trackX, y, barW, h, barW, palette.Background, palette.Background)
	thumbH := maxInt(c.s(24), int(float64(h)*float64(h)/float64(contentH)))
	if thumbH > h {
		thumbH = h
	}
	maxOffset := contentH - h
	travel := h - thumbH
	thumbY := y
	if maxOffset > 0 {
		thumbY += int(float64(travel) * float64(offset) / float64(maxOffset))
	}
	c.app.detailScrollThumbY, c.app.detailScrollThumbH = thumbY, thumbH
	c.roundedRect(trackX, thumbY, barW, thumbH, barW, palette.BorderBright, palette.BorderBright)
	prefix := "plugin"
	if view == 1 {
		prefix = "workshop"
	} else if view == 2 {
		prefix = "development"
	}
	c.addHit(prefix+"-detail-scroll-thumb", trackX-c.s(6), thumbY, barW+c.s(12), thumbH)
	if thumbY > y {
		c.addHit(prefix+"-detail-scroll-pageup", trackX-c.s(6), y, barW+c.s(12), thumbY-y)
	}
	if thumbY+thumbH < y+h {
		c.addHit(prefix+"-detail-scroll-pagedown", trackX-c.s(6), thumbY+thumbH, barW+c.s(12), y+h-(thumbY+thumbH))
	}
}

func drawPluginsPage(c *canvas, x, y, w, h int) {
	cursor := c.sectionTitle(tr("t.bureau_of_modification"), "Mod Manager", "", x, y, w)

	switchW, switchH := c.s(420), c.s(38)
	switchX := x
	c.roundedRect(switchX, cursor, switchW, switchH, c.s(10), palette.Surface, palette.Border)
	third := switchW / 3
	labels := []string{tr("t.plugins"), tr("t.workshop"), tr("t.development")}
	hits := []string{"mod-view:plugins", "mod-view:workshop", "mod-view:development"}
	for i := 0; i < 3; i++ {
		bx := switchX + i*third
		bw := third
		if i == 2 {
			bw = switchW - 2*third
		}
		fill := palette.Surface
		textColor := palette.TextMuted
		if c.app.modView == i {
			fill = palette.AccentSoft
			textColor = palette.AccentBright
		}
		insetX := c.s(2)
		insetW := bw - c.s(4)
		c.roundedRect(bx+insetX, cursor+c.s(2), insetW, switchH-c.s(4), c.s(8), fill, fill)
		c.text(labels[i], bx, cursor, bw, switchH, fonts.SmallStrong, textColor, dtCenter|dtVCenter|dtSingleLine)
		c.addHit(hits[i], bx, cursor, bw, switchH)
	}
	cursor += switchH + c.s(9)

	toolbarH := c.s(86)
	c.roundedRect(x, cursor, w, toolbarH, c.s(11), palette.Surface, palette.Border)
	row1Y := cursor + c.s(7)
	row2Y := cursor + c.s(46)
	c.compactButton("refresh", tr("t.refresh"), x+c.s(8), row1Y, c.s(76), c.s(32))
	if c.app.modView == 2 {
		c.compactButton("development-open-root", tr("t.open_projects_folder"), x+c.s(91), row1Y, c.s(142), c.s(32))
	} else {
		c.compactButton("mod-scan-updates", tr("t.scan_for_updates"), x+c.s(91), row1Y, c.s(126), c.s(32))
	}

	allLabel := tr("t.all_plugins")
	allValue := c.app.allPluginsEnabled()
	hideValue := c.app.cfg.HideDisabledPlugins
	needsValue := c.app.cfg.PluginNeedsAttentionOnly
	if c.app.modView == 1 {
		allLabel = tr("t.all_workshop")
		allValue = c.app.allWorkshopEnabled()
		hideValue = c.app.cfg.HideDisabledWorkshop
		needsValue = c.app.cfg.WorkshopNeedsAttentionOnly
	} else if c.app.modView == 2 {
		allLabel = tr("t.all_development")
		allValue = c.app.allDevelopmentEnabled()
		hideValue = c.app.cfg.HideDisabledDevelopment
		needsValue = false
	}
	allX := x + c.s(247)
	c.text(allLabel, allX, row1Y, c.s(96), c.s(32), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.toggle("mod-enable-all", allValue, allX+c.s(98), row1Y+c.s(6))
	hideX := allX + c.s(152)
	c.text(tr("t.hide_disabled"), hideX, row1Y, c.s(96), c.s(32), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.toggle("mod-hide-disabled", hideValue, hideX+c.s(98), row1Y+c.s(6))

	attentionX := x + c.s(10)
	searchX := attentionX
	attentionW := 0
	if c.app.modView != 2 {
		c.text(tr("t.needs_attention"), attentionX, row2Y, c.s(106), c.s(32), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.toggle("mod-needs-attention", needsValue, attentionX+c.s(108), row2Y+c.s(6))
		attentionW = c.s(162)
		searchX = attentionX + attentionW
	}
	if c.app.modView == 1 {
		pluginOnlyX := searchX
		c.text(tr("t.includes_dll"), pluginOnlyX, row2Y, c.s(86), c.s(32), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.toggle("mod-includes-plugin", c.app.cfg.WorkshopIncludesPluginOnly, pluginOnlyX+c.s(88), row2Y+c.s(6))
		searchX = pluginOnlyX + c.s(140)
	}
	gap2 := c.s(7)
	autoW, resetW := 0, 0
	if c.app.modView == 0 {
		autoLabel := tr("t.auto_order") + map[bool]string{true: tr("t.on"), false: tr("t.off")}[c.app.cfg.AutoOrder]
		autoW = maxInt(c.s(100), c.textWidth(autoLabel, fonts.SmallStrong)+c.s(18))
		resetW = maxInt(c.s(118), c.textWidth(tr("t.reset_recommended"), fonts.SmallStrong)+c.s(18))
	}
	reservedRight := c.s(10)
	if autoW > 0 {
		reservedRight += autoW + gap2 + resetW + gap2
	}
	availableSearch := x + w - reservedRight - searchX
	searchW := minInt(c.s(270), maxInt(c.s(120), availableSearch))
	drawModSearchBox(c, searchX, row2Y, searchW, c.s(32))
	nextX := searchX + searchW + c.s(9)
	if c.app.modView == 0 {
		autoLabel := tr("t.auto_order") + map[bool]string{true: tr("t.on"), false: tr("t.off")}[c.app.cfg.AutoOrder]
		c.compactButton("auto-order", autoLabel, nextX, row2Y, autoW, c.s(32))
		nextX += autoW + gap2
		c.compactButton("reset", tr("t.reset_recommended"), nextX, row2Y, resetW, c.s(32))
		nextX += resetW + gap2
	}
	if w-(nextX-x) > c.s(110) {
		summary := fmt.Sprintf(tr("t.d_shown_d_total"), len(c.app.visiblePluginIndices()), len(c.app.plugins))
		if c.app.modView == 1 {
			summary = fmt.Sprintf(tr("t.d_shown_d_installed"), len(c.app.visibleWorkshopIndices()), len(c.app.workshopItems))
		} else if c.app.modView == 2 {
			summary = fmt.Sprintf(tr("t.d_shown_d_local"), len(c.app.visibleDevelopmentIndices()), len(c.app.developmentItems))
		}
		c.text(summary, nextX, row2Y, x+w-nextX-c.s(10), c.s(32), fonts.Tiny, palette.TextMuted, dtRight|dtVCenter|dtSingleLine|dtEndEllipsis)
	}
	cursor += toolbarH + c.s(10)

	playlistH := c.s(50)
	c.roundedRect(x, cursor, w, playlistH, c.s(11), palette.Surface, palette.Border)
	c.label(tr("dashboard.playlist.label"), x+c.s(14), cursor+c.s(5), c.s(75), palette.TextFaint)
	name := c.app.cfg.SelectedPlaylist
	if name == "" {
		name = defaultPlaylistName
	}
	dropX := x + c.s(92)
	dropY := cursor + c.s(8)
	dropW := maxInt(c.s(260), minInt(c.s(440), w-c.s(250)))
	dropH := c.s(34)
	fill, border := palette.SurfaceRaised, palette.Border
	if c.app.hover == "playlist-dropdown" || c.app.playlistDropdown {
		fill, border = palette.SurfaceHover, palette.BorderBright
	}
	c.roundedRect(dropX, dropY, dropW, dropH, c.s(8), fill, border)
	c.text(name, dropX+c.s(12), dropY, dropW-c.s(42), dropH, fonts.BodyStrong, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	cx, cy := dropX+dropW-c.s(19), dropY+dropH/2
	c.line(cx-c.s(4), cy-c.s(2), cx, cy+c.s(2), c.s(1), palette.TextMuted)
	c.line(cx, cy+c.s(2), cx+c.s(4), cy-c.s(2), c.s(1), palette.TextMuted)
	c.addHit("playlist-dropdown", dropX, dropY, dropW, dropH)
	c.app.playlistDropX, c.app.playlistDropY, c.app.playlistDropW = dropX, dropY+dropH+c.s(3), dropW

	buttonSize := c.s(34)
	gapB := c.s(8)
	startX := x + w - (buttonSize*5 + gapB*4) - c.s(8)
	canEdit := !strings.EqualFold(name, defaultPlaylistName) && c.app.selectedPlaylistIndex() >= 0
	c.squareIconButton("playlist-save", "save", startX, dropY, buttonSize, false, true, tr("t.save_playlist"))
	c.squareIconButton("playlist-rename", "pencil", startX+(buttonSize+gapB), dropY, buttonSize, false, canEdit, tr("t.rename_playlist"))
	c.squareIconButton("playlist-export", "export", startX+2*(buttonSize+gapB), dropY, buttonSize, false, canEdit, tr("t.export_playlist"))
	c.squareIconButton("playlist-import", "import", startX+3*(buttonSize+gapB), dropY, buttonSize, false, true, tr("t.import_playlist"))
	c.squareIconButton("playlist-delete", "trash", startX+4*(buttonSize+gapB), dropY, buttonSize, true, canEdit, tr("t.delete_playlist"))
	cursor += playlistH + c.s(12)

	gap := c.s(14)
	leftW := int(float64(w-gap) * 0.58)
	rightW := w - gap - leftW
	panelH := h - (cursor - y) - c.s(10)
	switch c.app.modView {
	case 1:
		drawWorkshopTable(c, x, cursor, leftW, panelH)
		drawWorkshopDetail(c, x+leftW+gap, cursor, rightW, panelH)
	case 2:
		drawDevelopmentTable(c, x, cursor, leftW, panelH)
		drawDevelopmentDetail(c, x+leftW+gap, cursor, rightW, panelH)
	default:
		drawPluginTable(c, x, cursor, leftW, panelH)
		drawPluginDetail(c, x+leftW+gap, cursor, rightW, panelH)
	}
}

func drawPluginTable(c *canvas, x, y, w, h int) {
	c.card(x, y, w, h)
	headerH := c.s(42)
	c.fillRect(x+c.s(1), y+c.s(1), w-c.s(2), headerH, palette.SurfaceRaised)
	statusW := c.s(146)
	statusX := x + w - statusW - c.s(8)
	idW := c.s(92)
	idX := statusX - idW - c.s(10)
	c.text(tr("t.plugin"), x+c.s(18), y, maxInt(c.s(60), idX-x-c.s(28)), headerH, fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine)
	c.text(tr("t.item_id"), idX, y, idW, headerH, fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(tr("t.status"), statusX, y, statusW, headerH, fonts.Tiny, palette.TextMuted, dtCenter|dtVCenter|dtSingleLine)

	rowH := c.s(61)
	maxRows := maxInt(1, (h-headerH-c.s(6))/rowH)
	visible := c.app.visiblePluginIndices()
	c.app.pluginScroll = clampListScroll(c.app.pluginScroll, len(visible), maxRows)
	offset := c.app.pluginScroll
	rowY := y + headerH
	end := minInt(len(visible), offset+maxRows)
	for pos := offset; pos < end; pos++ {
		i := visible[pos]
		p := c.app.plugins[i]
		selected := c.app.selected == i
		id := fmt.Sprintf("plugin-row:%d", i)
		fill := palette.Surface
		nameColor := palette.Text
		if !p.BasicValid {
			fill = palette.Background
			nameColor = palette.TextMuted
		}
		if selected {
			fill = palette.AccentSoft
		} else if c.app.hover == id {
			fill = palette.SurfaceHover
		}
		c.fillRect(x+c.s(1), rowY, w-c.s(2), rowH, fill)
		if selected {
			c.fillRect(x+c.s(1), rowY, c.s(4), rowH, palette.AccentBright)
		}
		nameW := maxInt(c.s(60), idX-x-c.s(28))
		c.text(p.Manifest.Name, x+c.s(18), rowY+c.s(6), nameW, c.s(25), fonts.BodyStrong, nameColor, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		pluginSubline := p.Manifest.ID
		pluginSubColor := palette.TextFaint
		if !p.BasicValid {
			pluginSubColor = palette.Bad
			if len(p.PEValidation.Failures) > 0 {
				pluginSubline += "  •  " + p.PEValidation.Failures[0]
			}
		}
		if p.WorkshopID != "" {
			if item := c.app.pluginWorkshopItem(p); item != nil {
				label := workshopUpdateShortLabel(item)
				if label != "" {
					pluginSubline += "  •  " + label
				}
				switch item.UpdateState {
				case "current":
					pluginSubColor = palette.Good
				case "update", "stale":
					pluginSubColor = palette.Warn
				case "missing", "unavailable":
					pluginSubColor = palette.Bad
				case "downloading":
					pluginSubColor = palette.AccentBright
				}
			}
		}
		if p.BasicValid && p.Setting.Enabled && !canLoad(p) {
			pluginSubline += tr("t.needs_attention")
			pluginSubColor = palette.Bad
		} else if p.BasicValid && p.Setting.Enabled && p.Status == tr("t.warning") {
			pluginSubline += tr("t.warning")
			pluginSubColor = palette.Warn
		}
		c.text(pluginSubline, x+c.s(18), rowY+c.s(31), nameW, c.s(20), fonts.Tiny, pluginSubColor, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		itemID := tr("t.n_a")
		if p.WorkshopID != "" {
			itemID = p.WorkshopID
		}
		c.text(itemID, idX, rowY, idW, rowH, fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		if p.BasicValid {
			c.statusToggle(fmt.Sprintf("plugin-toggle:%d", i), p.Setting.Enabled, statusX, rowY+c.s(15), statusW, c.s(31))
		} else {
			c.invalidStatusControl(fmt.Sprintf("plugin-toggle:%d", i), statusX, rowY+c.s(15), statusW, c.s(31))
		}
		c.addHit(id, x, rowY, maxInt(c.s(20), statusX-x-c.s(4)), rowH)
		c.line(x+c.s(15), rowY+rowH-1, x+w-c.s(15), rowY+rowH-1, 1, palette.Border)
		rowY += rowH
	}
	c.app.modListX, c.app.modListY, c.app.modListW, c.app.modListH = x, y+headerH, w, h-headerH
	drawListScrollbar(c, "plugin", x+w-c.s(10), y+headerH+c.s(4), c.s(6), h-headerH-c.s(8), len(visible), maxRows, offset)
	if len(visible) == 0 {
		msg := tr("t.no_plugins_match_the")
		if len(c.app.plugins) == 0 {
			msg = tr("t.no_plugin_dlls_were")
		}
		c.text(msg, x+c.s(22), y+c.s(70), w-c.s(44), c.s(70), fonts.Body, palette.TextMuted, dtCenter|dtVCenter|dtWordBreak)
	}
}

func pluginInterfaceSummary(p *Plugin) string {
	if p == nil {
		return tr("t.interfaces_unknown")
	}
	parts := []string{}
	if p.HasRMLInterface {
		parts = append(parts, tr("t.rml"))
	}
	if p.HasTesmioInterface {
		if p.LegacyAPI > 0 {
			parts = append(parts, fmt.Sprintf(tr("t.tesmio_api_d"), p.LegacyAPI))
		} else {
			parts = append(parts, tr("t.tesmio"))
		}
	}
	if len(parts) == 0 {
		return tr("t.interface_raw_native_dll")
	}
	return tr("t.interfaces") + strings.Join(parts, " + ")
}

func pluginLoadPathSummary(p *Plugin) string {
	if p == nil {
		return tr("t.load_path_unknown")
	}
	switch p.LoaderKind {
	case pluginKindRML:
		return tr("t.load_path_native_rml")
	case pluginKindTesmio:
		return tr("t.load_path_tesmio_compatibility")
	default:
		if p.HasRMLInterface || p.HasTesmioInterface {
			return tr("t.load_path_raw_loadlibrary")
		}
		return tr("t.load_path_raw_loadlibrary")
	}
}

func pluginValidationDetails(p *Plugin) string {
	if p == nil {
		return ""
	}
	v := p.PEValidation
	peState := tr("t.invalid")
	if v.PEReadable {
		peState = tr("t.valid")
	}
	dllState := tr("t.unable_to_determine")
	if v.PEReadable {
		if v.DeclaredDLL {
			dllState = tr("t.yes")
		} else {
			dllState = tr("t.no")
		}
	}
	if p.BasicValid {
		return tr("t.binary_validation_valid_npe") + v.Architecture + tr("t.npe_format") + v.PEFormat + tr("t.ndeclared_as_dll") + dllState
	}
	var b strings.Builder
	b.WriteString(tr("t.plugin_validation_failed_n"))
	b.WriteString(tr("t.pe_image") + peState + "\n")
	b.WriteString(tr("t.architecture") + v.Architecture + "\n")
	b.WriteString(tr("t.pe_format") + v.PEFormat + "\n")
	b.WriteString(tr("t.declared_as_dll") + dllState)
	// Individual admission failures are rendered as prominent banners below the
	// validation facts. Keeping them out of this paragraph prevents the same
	// failure from being repeated as ordinary metadata and as a banner.
	if !v.PEReadable && strings.TrimSpace(v.ParserError) != "" {
		b.WriteString(tr("t.nparser_detail") + v.ParserError)
	}
	return b.String()
}

func pluginValidationBannerHeights(c *canvas, failures []string, width int) []int {
	if c == nil || len(failures) == 0 || width <= 0 {
		return nil
	}
	textW := maxInt(c.s(40), width-c.s(28))
	heights := make([]int, 0, len(failures))
	for _, failure := range failures {
		if strings.TrimSpace(failure) == "" {
			continue
		}
		textH := c.textHeight(failure, textW, fonts.SmallStrong)
		heights = append(heights, maxInt(c.s(36), textH+c.s(14)))
	}
	return heights
}

func drawPluginValidationBanners(c *canvas, failures []string, heights []int, x, y, w int) int {
	if c == nil || len(failures) == 0 || len(heights) == 0 || w <= 0 {
		return 0
	}
	used := 0
	hi := 0
	for _, failure := range failures {
		failure = strings.TrimSpace(failure)
		if failure == "" || hi >= len(heights) {
			continue
		}
		h := heights[hi]
		hi++
		by := y + used
		c.roundedRect(x, by, w, h, c.s(8), palette.BadSoft, palette.Bad)
		c.text(failure, x+c.s(14), by+c.s(5), w-c.s(28), h-c.s(10), fonts.SmallStrong, palette.Bad, dtCenter|dtVCenter|dtWordBreak)
		used += h
		if hi < len(heights) {
			used += c.s(8)
		}
	}
	return used
}

func drawPluginDetail(c *canvas, x, y, w, h int) {
	c.card(x, y, w, h)
	p := c.app.selectedPlugin()
	if p == nil {
		c.text(tr("t.select_a_plugin_to"), x+c.s(24), y+c.s(40), w-c.s(48), c.s(90), fonts.Body, palette.TextMuted, dtCenter|dtVCenter|dtWordBreak)
		return
	}
	c.label(tr("t.plugin_details"), x+c.s(20), y+c.s(16), w-c.s(40), palette.TextMuted)
	c.text(p.Manifest.Name, x+c.s(20), y+c.s(37), w-c.s(40), c.s(36), fonts.Heading, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	sourceLine := tr("t.local_plugin")
	if p.WorkshopID != "" {
		sourceLine = tr("t.workshop_id") + p.WorkshopID
	}
	c.text(sourceLine, x+c.s(20), y+c.s(72), w-c.s(40), c.s(24), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

	thumbX, thumbY := x+c.s(20), y+c.s(105)
	thumbW := detailThumbnailWidth(c, w)
	thumbH := c.s(108)
	c.roundedRect(thumbX, thumbY, thumbW, thumbH, c.s(9), palette.Background, palette.Border)
	if image := c.app.ensurePluginImage(); image != 0 {
		drawGDIPlusImageContainedNative(c.dc, image, thumbX+c.s(3), thumbY+c.s(3), thumbW-c.s(6), thumbH-c.s(6))
	} else {
		c.text(tr("t.no_package_preview"), thumbX+c.s(8), thumbY, thumbW-c.s(16), thumbH, fonts.Small, palette.TextFaint, dtCenter|dtVCenter|dtSingleLine|dtEndEllipsis)
	}

	pluginState, pluginStateClass := tr("t.disabled"), 3
	if !p.BasicValid {
		pluginState, pluginStateClass = tr("t.invalid"), 2
	} else if p.Setting.Enabled {
		pluginState, pluginStateClass = tr("t.enabled"), 0
	}
	metaX := thumbX + thumbW + c.s(14)
	metaW := maxInt(c.s(82), x+w-metaX-c.s(20))
	c.pill(pluginState, pluginStateClass, metaX, thumbY+c.s(8), minInt(c.s(104), metaW), c.s(28))
	c.text(pluginInterfaceSummary(p), metaX, thumbY+c.s(42), metaW, c.s(22), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(pluginLoadPathSummary(p), metaX, thumbY+c.s(65), metaW, c.s(22), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	thirdLine := tr("t.binary") + p.PEValidation.Architecture + " • " + p.PEValidation.PEFormat
	thirdColor := palette.TextMuted
	if item := c.app.pluginWorkshopItem(p); item != nil {
		label := workshopUpdateShortLabel(item)
		if label != "" {
			thirdLine = tr("t.update") + strings.ToUpper(label)
			switch item.UpdateState {
			case "current":
				thirdColor = palette.Good
			case "update", "stale":
				thirdColor = palette.Warn
			case "missing", "unavailable":
				thirdColor = palette.Bad
			case "downloading":
				thirdColor = palette.AccentBright
			}
		}
	}
	c.text(thirdLine, metaX, thumbY+c.s(87), metaW, c.s(18), fonts.Tiny, thirdColor, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

	buttonY3 := y + h - c.s(60)
	buttonY2 := buttonY3 - c.s(48)
	buttonY1 := buttonY2 - c.s(48)
	buttonTop := buttonY3
	if p.WorkshopID != "" {
		buttonTop = buttonY1
	}
	controlY := buttonTop - c.s(64)
	heroBottom := thumbY + thumbH
	descLabelY := heroBottom + c.s(14)
	descTextY := descLabelY + c.s(20)
	if controlY < descTextY+c.s(42) {
		controlY = descTextY + c.s(42)
	}
	descH := maxInt(c.s(36), controlY-descTextY-c.s(10))
	c.label(tr("t.description"), x+c.s(20), descLabelY, w-c.s(40), palette.TextFaint)
	body := p.Manifest.Description
	if len(p.Manifest.Dependencies) > 0 {
		body += tr("t.n_ndependencies") + strings.Join(p.Manifest.Dependencies, "  •  ")
	}
	if len(p.Manifest.RequiredSymbols) > 0 {
		body += tr("t.n_nrequired_symbols") + strings.Join(p.Manifest.RequiredSymbols, "  •  ")
	}
	// Invalid PE-admission failures have their own high-visibility banners below,
	// so do not repeat those same strings in the ordinary metadata paragraph.
	if len(p.Reasons) > 0 && p.BasicValid {
		body += tr("t.n_nruntime_metadata_notes") + strings.Join(p.Reasons, "  •  ")
	}
	body += tr("t.n_n") + pluginValidationDetails(p)
	bannerW := w - c.s(40)
	bannerHeights := []int(nil)
	bannerBlockH := 0
	if !p.BasicValid {
		bannerHeights = pluginValidationBannerHeights(c, p.PEValidation.Failures, bannerW)
		for i, bh := range bannerHeights {
			bannerBlockH += bh
			if i+1 < len(bannerHeights) {
				bannerBlockH += c.s(8)
			}
		}
	}
	if bannerBlockH > 0 {
		// Reserve banner space inside the details body. The textual description
		// remains independently scrollable if it is long, while every validation
		// failure remains immediately visible without being buried in prose.
		descH = maxInt(c.s(36), descH-bannerBlockH-c.s(10))
	}
	drawScrollableDescription(c, 0, body, x+c.s(20), descTextY, w-c.s(40), descH, fonts.Small, palette.TextMuted)
	if bannerBlockH > 0 {
		drawPluginValidationBanners(c, p.PEValidation.Failures, bannerHeights, x+c.s(20), descTextY+descH+c.s(10), bannerW)
	}

	c.divider(x+c.s(20), controlY, w-c.s(40))
	controlY += c.s(10)
	c.text(tr("t.callback_priority"), x+c.s(20), controlY, w-c.s(40), c.s(22), fonts.BodyStrong, palette.Text, dtLeft|dtVCenter|dtSingleLine)
	c.text(tr("t.lower_priorities_run_earlier"), x+c.s(20), controlY+c.s(23), w-c.s(170), c.s(19), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.compactButton("priority-minus", "− 10", x+w-c.s(170), controlY+c.s(4), c.s(48), c.s(34))
	c.roundedRect(x+w-c.s(116), controlY+c.s(4), c.s(42), c.s(34), c.s(8), palette.Background, palette.Border)
	c.text(fmt.Sprintf("%d", p.Setting.Priority), x+w-c.s(116), controlY+c.s(4), c.s(42), c.s(34), fonts.SmallStrong, palette.Text, dtCenter|dtVCenter|dtSingleLine)
	c.compactButton("priority-plus", "+ 10", x+w-c.s(68), controlY+c.s(4), c.s(48), c.s(34))

	settingsLabel := tr("t.open_plugin_ini_settings")
	settingsAvailable := p.SettingsPath != "" || (p.SettingsCount > 1 && p.SettingsDir != "")
	if p.SettingsPath == "" {
		if p.SettingsCount > 1 && p.SettingsDir != "" {
			settingsLabel = tr("t.open_settings_folder")
		} else {
			settingsLabel = tr("t.no_plugin_ini_found")
		}
	}
	if p.WorkshopID != "" {
		gap := c.s(8)
		buttonW := (w - c.s(40) - gap) / 2
		c.buttonEnabled("plugin-open-settings", settingsLabel, x+c.s(20), buttonY1, w-c.s(40), c.s(40), false, settingsAvailable)
		c.button("plugin-workshop-open-page", tr("t.open_workshop"), x+c.s(20), buttonY2, buttonW, c.s(40), false)
		c.button("plugin-workshop-open-folder", tr("t.open_file_location"), x+c.s(20)+buttonW+gap, buttonY2, buttonW, c.s(40), false)
		canAction := !c.app.workshopActionBusy && !c.app.workshopUpdateBusy
		if item := c.app.workshopItemByID(p.WorkshopID); item != nil && item.Unsubscribed {
			canAction = false
		}
		c.button("plugin-workshop-force-update", tr("t.force_update"), x+c.s(20), buttonY3, buttonW, c.s(40), true)
		c.dangerButton("plugin-workshop-unsubscribe", tr("t.unsubscribe"), x+c.s(20)+buttonW+gap, buttonY3, buttonW, c.s(40), canAction)
	} else {
		c.buttonEnabled("plugin-open-settings", settingsLabel, x+c.s(20), buttonY3, w-c.s(40), c.s(40), false, settingsAvailable)
	}
}

func drawDevelopmentTable(c *canvas, x, y, w, h int) {
	c.card(x, y, w, h)
	headerH := c.s(42)
	c.fillRect(x+c.s(1), y+c.s(1), w-c.s(2), headerH, palette.SurfaceRaised)
	statusW := c.s(146)
	statusX := x + w - statusW - c.s(8)
	idW := c.s(92)
	idX := statusX - idW - c.s(10)
	c.text(tr("t.development_mod"), x+c.s(18), y, maxInt(c.s(60), idX-x-c.s(28)), headerH, fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(tr("t.project_id"), idX, y, idW, headerH, fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(tr("t.status"), statusX, y, statusW, headerH, fonts.Tiny, palette.TextMuted, dtCenter|dtVCenter|dtSingleLine)

	rowH := c.s(61)
	maxRows := maxInt(1, (h-headerH-c.s(6))/rowH)
	visible := c.app.visibleDevelopmentIndices()
	c.app.developmentScroll = clampListScroll(c.app.developmentScroll, len(visible), maxRows)
	offset := c.app.developmentScroll
	rowY := y + headerH
	end := minInt(len(visible), offset+maxRows)
	for pos := offset; pos < end; pos++ {
		i := visible[pos]
		item := &c.app.developmentItems[i]
		selected := c.app.selectedDevelopment == i
		id := fmt.Sprintf("development-row:%d", i)
		fill := palette.Surface
		if selected {
			fill = palette.AccentSoft
		} else if c.app.hover == id {
			fill = palette.SurfaceHover
		}
		c.fillRect(x+c.s(1), rowY, w-c.s(2), rowH, fill)
		if selected {
			c.fillRect(x+c.s(1), rowY, c.s(4), rowH, palette.AccentBright)
		}
		nameW := maxInt(c.s(60), idX-x-c.s(28))
		c.text(item.Name, x+c.s(18), rowY+c.s(6), nameW, c.s(25), fonts.BodyStrong, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		sub := tr("t.local_unpublished_project")
		subColor := palette.TextFaint
		if len(item.EmbeddedPluginIDs) > 0 {
			sub = fmt.Sprintf(tr("t.local_unpublished_project_d"), len(item.EmbeddedPluginIDs))
			subColor = palette.AccentBright
		}
		if len(item.Issues) > 0 {
			sub += fmt.Sprintf(tr("t.d_issue_s"), len(item.Issues))
			subColor = palette.Warn
		}
		c.text(sub, x+c.s(18), rowY+c.s(31), nameW, c.s(20), fonts.Tiny, subColor, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.text(item.ID, idX, rowY, idW, rowH, fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.statusToggle(fmt.Sprintf("development-toggle:%d", i), item.Enabled, statusX, rowY+c.s(15), statusW, c.s(31))
		c.addHit(id, x, rowY, maxInt(c.s(20), statusX-x-c.s(4)), rowH)
		c.line(x+c.s(15), rowY+rowH-1, x+w-c.s(15), rowY+rowH-1, 1, palette.Border)
		rowY += rowH
	}
	c.app.modListX, c.app.modListY, c.app.modListW, c.app.modListH = x, y+headerH, w, h-headerH
	drawListScrollbar(c, "development", x+w-c.s(10), y+headerH+c.s(4), c.s(6), h-headerH-c.s(8), len(visible), maxRows, offset)
	if len(visible) == 0 {
		msg := tr("t.no_development_mods_match")
		if len(c.app.developmentItems) == 0 {
			msg = tr("t.no_numeric_development_project")
		}
		c.text(msg, x+c.s(22), y+c.s(70), w-c.s(44), c.s(80), fonts.Body, palette.TextMuted, dtCenter|dtVCenter|dtWordBreak)
	}
}

func drawDevelopmentDetail(c *canvas, x, y, w, h int) {
	c.card(x, y, w, h)
	item := c.app.selectedDevelopmentItem()
	if item == nil {
		c.text(tr("t.select_a_local_development"), x+c.s(24), y+c.s(40), w-c.s(48), c.s(90), fonts.Body, palette.TextMuted, dtCenter|dtVCenter|dtWordBreak)
		return
	}
	c.label(tr("t.development_mod"), x+c.s(20), y+c.s(16), w-c.s(40), palette.TextMuted)
	c.text(item.Name, x+c.s(20), y+c.s(37), w-c.s(40), c.s(36), fonts.Heading, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(tr("t.workshop_id")+item.ID, x+c.s(20), y+c.s(72), w-c.s(40), c.s(24), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

	thumbX, thumbY := x+c.s(20), y+c.s(105)
	thumbW := detailThumbnailWidth(c, w)
	thumbH := c.s(108)
	c.roundedRect(thumbX, thumbY, thumbW, thumbH, c.s(9), palette.Background, palette.Border)
	if image := c.app.ensureDevelopmentImage(); image != 0 {
		drawGDIPlusImageContainedNative(c.dc, image, thumbX+c.s(3), thumbY+c.s(3), thumbW-c.s(6), thumbH-c.s(6))
	} else {
		c.text(tr("t.no_preview_image"), thumbX+c.s(8), thumbY, thumbW-c.s(16), thumbH, fonts.Small, palette.TextFaint, dtCenter|dtVCenter|dtSingleLine|dtEndEllipsis)
	}
	state, stateClass := tr("t.disabled"), 3
	if item.Enabled {
		state, stateClass = tr("t.enabled"), 0
	}
	metaX := thumbX + thumbW + c.s(14)
	metaW := maxInt(c.s(82), x+w-metaX-c.s(20))
	c.pill(state, stateClass, metaX, thumbY+c.s(8), minInt(c.s(104), metaW), c.s(28))
	packageType := tr("t.development_workshop_project")
	if len(item.EmbeddedPluginIDs) > 0 {
		packageType = fmt.Sprintf(tr("t.hybrid_package_d_embedded"), len(item.EmbeddedPluginIDs))
	}
	c.text(packageType, metaX, thumbY+c.s(42), metaW, c.s(40), fonts.Tiny, palette.TextMuted, dtLeft|dtWordBreak)
	if len(item.Issues) > 0 {
		c.text(fmt.Sprintf(tr("t.needs_attention_d_project"), len(item.Issues)), metaX, thumbY+c.s(86), metaW, c.s(20), fonts.Tiny, palette.Warn, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	}

	rowY := thumbY + thumbH + c.s(15)
	c.label(tr("t.embedded_dll_payloads"), x+c.s(20), rowY, w-c.s(40), palette.TextFaint)
	rowY += c.s(20)
	payloads := tr("t.none")
	if len(item.EmbeddedPluginIDs) > 0 {
		payloads = strings.Join(item.EmbeddedPluginIDs, "  •  ")
	}
	payloadH := minInt(c.s(42), maxInt(c.s(20), c.textHeight(payloads, w-c.s(40), fonts.Small)))
	c.text(payloads, x+c.s(20), rowY, w-c.s(40), payloadH, fonts.Small, palette.TextMuted, dtLeft|dtWordBreak)
	rowY += payloadH + c.s(10)
	c.label(tr("t.description"), x+c.s(20), rowY, w-c.s(40), palette.TextFaint)
	rowY += c.s(20)

	buttonY := y + h - c.s(60)
	descH := maxInt(c.s(45), buttonY-rowY-c.s(12))
	description := item.Description
	if len(item.Issues) > 0 {
		description += tr("t.n_nproject_checks_n") + strings.Join(item.Issues, "\n• ")
	}
	drawScrollableDescription(c, 2, description, x+c.s(20), rowY, w-c.s(40), descH, fonts.Small, palette.TextMuted)

	gap := c.s(8)
	buttonW := (w - c.s(40) - gap) / 2
	c.button("development-open-folder", tr("t.open_project_folder"), x+c.s(20), buttonY, buttonW, c.s(40), false)
	configLabel := tr("t.open_workshopconfig_ini")
	configAvailable := item.ConfigPath != ""
	if !configAvailable {
		configLabel = tr("t.config_not_found")
	}
	c.buttonEnabled("development-open-config", configLabel, x+c.s(20)+buttonW+gap, buttonY, buttonW, c.s(40), false, configAvailable)
}

func drawWorkshopTable(c *canvas, x, y, w, h int) {
	c.card(x, y, w, h)
	headerH := c.s(42)
	c.fillRect(x+c.s(1), y+c.s(1), w-c.s(2), headerH, palette.SurfaceRaised)
	statusW := c.s(146)
	statusX := x + w - statusW - c.s(8)
	idW := c.s(92)
	idX := statusX - idW - c.s(10)
	nameLabel := workshopSortLabel(tr("t.workshop_mod"), "name", c.app)
	idLabel := workshopSortLabel(tr("t.item_id"), "id", c.app)
	statusLabel := workshopSortLabel(tr("t.status"), "on", c.app)
	nameColor, idColor, statusColor := palette.TextMuted, palette.TextMuted, palette.TextMuted
	if c.app.workshopSort == "name" {
		nameColor = palette.AccentBright
	}
	if c.app.workshopSort == "id" {
		idColor = palette.AccentBright
	}
	if c.app.workshopSort == "on" {
		statusColor = palette.AccentBright
	}
	c.text(nameLabel, x+c.s(18), y, maxInt(c.s(60), idX-x-c.s(28)), headerH, fonts.Tiny, nameColor, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(idLabel, idX, y, idW, headerH, fonts.Tiny, idColor, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(statusLabel, statusX, y, statusW, headerH, fonts.Tiny, statusColor, dtCenter|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.addHit("workshop-sort:name", x, y, maxInt(c.s(20), idX-x), headerH)
	c.addHit("workshop-sort:id", idX, y, idW, headerH)
	c.addHit("workshop-sort:on", statusX, y, statusW, headerH)

	rowH := c.s(61)
	maxRows := maxInt(1, (h-headerH-c.s(6))/rowH)
	visible := c.app.visibleWorkshopIndices()
	c.app.workshopScroll = clampListScroll(c.app.workshopScroll, len(visible), maxRows)
	offset := c.app.workshopScroll
	rowY := y + headerH
	end := minInt(len(visible), offset+maxRows)
	for pos := offset; pos < end; pos++ {
		i := visible[pos]
		item := &c.app.workshopItems[i]
		selected := c.app.selectedWorkshop == i
		id := fmt.Sprintf("workshop-row:%d", i)
		fill := palette.Surface
		if selected {
			fill = palette.AccentSoft
		} else if c.app.hover == id {
			fill = palette.SurfaceHover
		}
		c.fillRect(x+c.s(1), rowY, w-c.s(2), rowH, fill)
		if selected {
			c.fillRect(x+c.s(1), rowY, c.s(4), rowH, palette.AccentBright)
		}
		nameW := maxInt(c.s(60), idX-x-c.s(28))
		c.text(item.Name, x+c.s(18), rowY+c.s(6), nameW, c.s(25), fonts.BodyStrong, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		sub := tr("t.steam_workshop")
		subColor := palette.TextFaint
		if item.Unsubscribed {
			sub = tr("t.unsubscribed")
			subColor = palette.Bad
		} else if item.UpdateState == "stale" {
			sub = tr("t.potentially_stale") + item.UpdateReason
			subColor = palette.Warn
		} else if item.UpdateState == "update" {
			sub = tr("t.update_available") + item.UpdateReason
			subColor = palette.Warn
		} else if item.UpdateState == "downloading" {
			sub = tr("t.steam_update_downloading_queued")
			subColor = palette.AccentBright
		} else if item.UpdateState == "missing" {
			sub = tr("t.installed_copy_missing")
			subColor = palette.Bad
		} else if item.UpdateState == "current" {
			sub = tr("t.steam_workshop_up_to")
			subColor = palette.Good
		} else if item.Unsupported {
			sub = tr("t.needs_attention")
			if len(item.UnsupportedReasons) > 0 {
				sub += " — " + localizeReason(item.UnsupportedReasons[0])
			}
			subColor = palette.Bad
		} else if item.RemoteSupportState == "ok" {
			sub = tr("t.steam_workshop_page_verified")
		}
		c.text(sub, x+c.s(18), rowY+c.s(31), nameW, c.s(20), fonts.Tiny, subColor, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.text(item.ID, idX, rowY, idW, rowH, fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		c.statusToggle(fmt.Sprintf("workshop-toggle:%d", i), item.Enabled, statusX, rowY+c.s(15), statusW, c.s(31))
		c.addHit(id, x, rowY, maxInt(c.s(20), statusX-x-c.s(4)), rowH)
		c.line(x+c.s(15), rowY+rowH-1, x+w-c.s(15), rowY+rowH-1, 1, palette.Border)
		rowY += rowH
	}
	c.app.modListX, c.app.modListY, c.app.modListW, c.app.modListH = x, y+headerH, w, h-headerH
	drawListScrollbar(c, "workshop", x+w-c.s(10), y+headerH+c.s(4), c.s(6), h-headerH-c.s(8), len(visible), maxRows, offset)
	if len(visible) == 0 {
		msg := tr("t.no_workshop_mods_match")
		if len(c.app.workshopItems) == 0 {
			msg = tr("t.no_installed_wrsr_workshop")
		}
		c.text(msg, x+c.s(22), y+c.s(70), w-c.s(44), c.s(70), fonts.Body, palette.TextMuted, dtCenter|dtVCenter|dtWordBreak)
	}
}

func drawWorkshopDetail(c *canvas, x, y, w, h int) {
	c.card(x, y, w, h)
	item := c.app.selectedWorkshopItem()
	if item == nil {
		c.text(tr("t.select_an_installed_workshop"), x+c.s(24), y+c.s(40), w-c.s(48), c.s(90), fonts.Body, palette.TextMuted, dtCenter|dtVCenter|dtWordBreak)
		return
	}
	c.label(tr("t.workshop_details"), x+c.s(20), y+c.s(16), w-c.s(40), palette.TextMuted)
	c.text(item.Name, x+c.s(20), y+c.s(37), w-c.s(40), c.s(36), fonts.Heading, palette.Text, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	c.text(tr("t.workshop_id")+item.ID, x+c.s(20), y+c.s(72), w-c.s(40), c.s(24), fonts.Small, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

	thumbX, thumbY := x+c.s(20), y+c.s(105)
	if item.Unsupported {
		reason := tr("t.needs_attention")
		if len(item.UnsupportedReasons) > 0 {
			reason += ": " + strings.Join(localizeWarnings(item.UnsupportedReasons), "; ")
		}
		c.text(reason, x+c.s(20), y+c.s(96), w-c.s(40), c.s(29), fonts.Tiny, palette.Bad, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		thumbY = y + c.s(126)
	}
	thumbW := detailThumbnailWidth(c, w)
	thumbH := c.s(108)
	c.roundedRect(thumbX, thumbY, thumbW, thumbH, c.s(9), palette.Background, palette.Border)
	image := c.app.ensureWorkshopImage()
	if image != 0 {
		drawGDIPlusImageContainedNative(c.dc, image, thumbX+c.s(3), thumbY+c.s(3), thumbW-c.s(6), thumbH-c.s(6))
	} else {
		c.text(tr("t.no_local_thumbnail"), thumbX+c.s(8), thumbY, thumbW-c.s(16), thumbH, fonts.Small, palette.TextFaint, dtCenter|dtVCenter|dtSingleLine|dtEndEllipsis)
	}
	state := tr("t.disabled")
	class := 3
	if item.Enabled {
		state, class = tr("t.enabled"), 0
	}
	if item.Unsubscribed {
		state, class = tr("t.unsubscribed"), 2
	}
	c.pill(state, class, thumbX+thumbW+c.s(14), thumbY+c.s(8), c.s(104), c.s(28))
	workshopStateDetail := tr("t.playlist_inclusion")
	if len(item.EmbeddedPluginIDs) > 0 {
		workshopStateDetail = fmt.Sprintf(tr("t.hybrid_package_d_linked"), len(item.EmbeddedPluginIDs))
	}
	c.text(workshopStateDetail, thumbX+thumbW+c.s(14), thumbY+c.s(42), maxInt(c.s(80), w-thumbW-c.s(54)), c.s(48), fonts.Tiny, palette.TextMuted, dtLeft|dtWordBreak)

	descY := thumbY + thumbH + c.s(16)
	if item.UpdateState != "" {
		updateLabel := strings.ToUpper(item.UpdateState)
		updateColor := palette.TextMuted
		if item.UpdateState == "current" {
			updateColor = palette.Good
		}
		if item.UpdateState == "update" || item.UpdateState == "stale" {
			updateColor = palette.Warn
		}
		if item.UpdateState == "missing" || item.UpdateState == "unavailable" {
			updateColor = palette.Bad
		}
		c.label(tr("t.update_status"), x+c.s(20), descY, w-c.s(40), palette.TextFaint)
		descY += c.s(19)
		c.text(updateLabel+"  •  "+item.UpdateReason, x+c.s(20), descY, w-c.s(40), c.s(30), fonts.Tiny, updateColor, dtLeft|dtWordBreak)
		descY += c.s(31)
		times := tr("t.installed") + formatWorkshopUpdateTime(item.InstalledUpdatedUnix) + tr("t.workshop") + formatWorkshopUpdateTime(item.RemoteUpdatedUnix)
		c.text(times, x+c.s(20), descY, w-c.s(40), c.s(20), fonts.Tiny, palette.TextMuted, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
		descY += c.s(25)
	}
	c.label(tr("t.description"), x+c.s(20), descY, w-c.s(40), palette.TextFaint)
	descY += c.s(20)
	buttonY2 := y + h - c.s(62)
	buttonY1 := buttonY2 - c.s(48)
	descH := maxInt(c.s(55), buttonY1-descY-c.s(12))
	drawScrollableDescription(c, 1, item.Description, x+c.s(20), descY, w-c.s(40), descH, fonts.Small, palette.TextMuted)
	gap := c.s(8)
	buttonW := (w - c.s(40) - gap) / 2
	c.button("workshop-open-page", tr("t.open_workshop"), x+c.s(20), buttonY1, buttonW, c.s(40), false)
	c.button("workshop-open-folder", tr("t.open_file_location"), x+c.s(20)+buttonW+gap, buttonY1, buttonW, c.s(40), false)
	c.button("workshop-force-update", tr("t.force_update"), x+c.s(20), buttonY2, buttonW, c.s(40), true)
	canUnsubscribe := !c.app.workshopActionBusy && !c.app.workshopUpdateBusy && !item.Unsubscribed
	c.dangerButton("workshop-unsubscribe", tr("t.unsubscribe"), x+c.s(20)+buttonW+gap, buttonY2, buttonW, c.s(40), canUnsubscribe)
}

func drawDiagnosticsPage(c *canvas, x, y, w, h int) {
	cursor := c.sectionTitle(tr("t.support_diagnostics"), "Diagnostics", "", x, y, w)
	gap := c.s(14)
	topH := c.s(104)
	cardW := (w - gap*2) / 3
	state, detail, cls := tr("t.no_report"), tr("t.no_wrsr_session_recorded"), 1
	if c.app.report != nil {
		if c.app.report.BootstrapStatus == 0 {
			state, detail, cls = tr("t.ok"), tr("t.last_rml_bootstrap_completed"), 0
		} else {
			state, detail, cls = fmt.Sprintf(tr("t.0x_x"), c.app.report.BootstrapStatus), tr("t.last_bootstrap_reported_an"), 2
		}
	}
	c.metricCard(tr("t.last_session"), state, detail, x, cursor, cardW, topH, cls)
	detailed := tr("t.off")
	if c.app.cfg.VerboseDiagnostics {
		detailed = tr("t.on")
	}
	c.metricCard(tr("t.detailed_logging"), detailed, tr("t.full_troubleshooting_trace_is"), x+cardW+gap, cursor, cardW, topH, map[bool]int{true: 1, false: 0}[c.app.cfg.VerboseDiagnostics])
	archives := tr("t.manual_only")
	if c.app.cfg.AutoCrashArchives {
		archives = tr("t.automatic")
	}
	c.metricCard(tr("t.crash_archives"), archives, tr("t.timestamped_zip_on_abnormal"), x+(cardW+gap)*2, cursor, cardW, topH, 0)
	cursor += topH + c.s(14)

	buttons := []struct {
		ID    string
		Label string
	}{
		{"refresh", tr("t.refresh")},
		{"open-log", tr("t.open_session_log")},
		{"open-verbose-log", tr("t.open_detailed_log")},
		{"create-support-archive", tr("t.create_support_zip")},
		{"open-crash-folder", tr("t.crash_archives")},
	}
	buttonGap := c.s(7)
	innerW := w - c.s(14)
	lineW, rows := 0, 1
	widths := make([]int, len(buttons))
	for i, b := range buttons {
		bw := maxInt(c.s(72), c.textWidth(b.Label, fonts.SmallStrong)+c.s(20))
		widths[i] = bw
		if lineW > 0 && lineW+buttonGap+bw > innerW {
			rows++
			lineW = bw
		} else {
			if lineW > 0 {
				lineW += buttonGap
			}
			lineW += bw
		}
	}
	buttonH := c.s(34)
	toolbarH := c.s(10) + rows*buttonH + (rows-1)*c.s(5)
	c.roundedRect(x, cursor, w, toolbarH, c.s(10), palette.Surface, palette.Border)
	bx, by := x+c.s(7), cursor+c.s(5)
	for i, b := range buttons {
		bw := widths[i]
		if bx > x+c.s(7) && bx+bw > x+w-c.s(7) {
			bx = x + c.s(7)
			by += buttonH + c.s(5)
		}
		c.compactButton(b.ID, b.Label, bx, by, bw, buttonH)
		bx += bw + buttonGap
	}
	cursor += toolbarH + c.s(12)

	logH := h - (cursor - y) - c.s(10)
	c.roundedRect(x, cursor, w, logH, c.s(12), rgb(7, 10, 14), palette.Border)
	c.fillRect(x+c.s(1), cursor+c.s(1), c.s(5), logH-c.s(2), palette.Accent)
	c.label(tr("t.latest_session_high_level"), x+c.s(22), cursor+c.s(14), w-c.s(44), palette.TextFaint)
	log := c.app.logTail
	if len(log) > 16000 {
		log = log[len(log)-16000:]
	}
	c.text(log, x+c.s(22), cursor+c.s(40), w-c.s(44), logH-c.s(54), fonts.Mono, palette.TextMuted, dtLeft|dtWordBreak)
}

func hitAt(app *application, x, y int) string {
	// Later-drawn regions win, which lets toggles sit over their parent row.
	for i := len(app.hits) - 1; i >= 0; i-- {
		h := app.hits[i]
		if x >= h.X && y >= h.Y && x < h.X+h.W && y < h.Y+h.H {
			return h.ID
		}
	}
	return ""
}
