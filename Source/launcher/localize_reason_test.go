package main

import "testing"

func TestLocalizeReasonKnownStrings(t *testing.T) {
	cases := map[string]string{
		"No local Workshop metadata":   "",
		"No mod name":                  "",
		"No preview image":             "",
		"No local thumbnail":           "",
		"Workshop folder is empty":     "",
		"Workshop page is unavailable": "",
	}
	for known := range cases {
		got := localizeReason(known)
		if got == known {
			t.Errorf("localizeReason(%q) must translate, got source text", known)
		}
	}
	if got := localizeReason("running as expected"); got != "running as expected" {
		t.Errorf("unknown reason must pass through, got %q", got)
	}
}

func TestLocalizeWarningsPassesThroughUnknown(t *testing.T) {
	in := []string{"No local Workshop metadata", "custom text"}
	out := localizeWarnings(in)
	if len(out) != 2 || out[0] == in[0] || out[1] != in[1] {
		t.Errorf("localizeWarnings = %v", out)
	}
}
