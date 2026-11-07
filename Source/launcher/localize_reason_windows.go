//go:build windows

package main

// reasonKeyByText maps upstream English warning strings (stored as data in
// WorkshopItem / Plugin structures) to locale keys. Unknown strings pass
// through untranslated: diagnostics and third-party reasons remain as-is.

var reasonKeyByText = map[string]string{
	"No local Workshop metadata":                        "warn.no_metadata",
	"No mod name":                                       "warn.no_mod_name",
	"No preview image":                                  "warn.no_preview_image",
	"No local thumbnail":                                "warn.no_local_thumbnail",
	"Workshop folder is empty":                          "warn.folder_empty",
	"Workshop page is unavailable":                      "warn.page_unavailable",
	"Workshop item appears delisted or inaccessible":    "warn.delisted",
	"Workshop item has been removed from the community": "warn.removed",
	"Workshop: Steam library could not be derived from the selected WRSR path":                       "warn.no_steam_library",
	"Could not verify Workshop revision":                                                             "update.reason.unverified",
	"Workshop item is no longer retrievable from Steam":                                              "update.reason.missing",
	"Steam is currently downloading this Workshop item":                                              "update.reason.downloading",
	"Steam reports that this Workshop item needs an update":                                          "update.reason.needs_update",
	"Workshop item is subscribed/tracked but no installed revision was found":                        "update.reason.no_installed",
	"Steam's latest manifest differs from the installed manifest":                                    "update.reason.manifest",
	"Workshop server revision is newer than the installed copy, but Steam is not flagging an update": "update.reason.server_newer_no_flag",
	"Workshop server revision is newer than the installed copy":                                      "update.reason.server_newer",
	"Installed Workshop revision matches the latest published update timestamp":                      "update.reason.current",
	"Installed locally, but the latest published timestamp could not be verified":                    "update.reason.current_unverified",
}

func localizeReason(text string) string {
	if key, ok := reasonKeyByText[text]; ok {
		return tr(key)
	}
	return text
}

func localizeWarnings(list []string) []string {
	if len(list) == 0 {
		return list
	}
	out := make([]string, len(list))
	for i, text := range list {
		out[i] = localizeReason(text)
	}
	return out
}
