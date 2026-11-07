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
	"Workshop: Steam library could not be derived from the selected WRSR path": "warn.no_steam_library",
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
