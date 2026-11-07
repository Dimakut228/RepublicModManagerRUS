package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Scope lists per localisation task. A later task extends the list; make sure
// every entry here is covered by tr() wrappers before the suite may go green.
var taskScopeFunctions = []string{
	"drawSidebar",
	"drawStatusBar",
	"drawDashboardPage",
	"drawDashboardPlaylistCard",
	"drawDashboardSettingsCard",
	"drawBuildSHATooltip",
	"drawGenericTooltip",
	"drawToast",
}

var displayCallRe = regexp.MustCompile(`c\.(label|text|metricCard|sectionTitle)\(`)

func functionBody(t *testing.T, file, name string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	re := regexp.MustCompile(`(?s)func ` + name + `\([^\r\n]*\{.*?\r?\n\}\r?\n`)
	match := re.Find(data)
	if match == nil {
		t.Fatalf("function %s not found in %s", name, file)
	}
	return string(match)
}

func literalIsLocalised(callArgs string) bool {
	trimmed := callArgs
	trimmed = strings.TrimPrefix(trimmed, "fmt.Sprintf(")
	if strings.HasPrefix(trimmed, `tr("`) || strings.HasPrefix(trimmed, "tr(\"") {
		return true
	}
	// allow concatenations that start with tr(...)
	if strings.HasPrefix(trimmed, "tr(") {
		return true
	}
	return false
}

func TestNoHardcodedDashboardStrings(t *testing.T) {
	const file = "ui_pages_windows.go"
	for _, name := range taskScopeFunctions {
		body := functionBody(t, file, name)
		for _, m := range displayCallRe.FindAllStringSubmatchIndex(body, -1) {
			argsStart := m[1]
			lit := firstLiteralInArgs(body[argsStart:])
			if lit == "" {
				continue
			}
			if !literalContainsText(lit) {
				continue
			}
			args := body[argsStart:]
			if literalIsLocalised(strings.TrimLeft(args, " \t")) {
				continue
			}
			t.Errorf("%s: unlocalised UI text %q in c.%s", name, lit, body[m[0]:m[1]])
		}
	}
}

func firstLiteralInArgs(args string) string {
	idx := strings.IndexAny(args, "\"`")
	if idx < 0 {
		return ""
	}
	if idx > 0 && strings.ContainsAny(args[:idx], "abcdefghijklmnopqrstuvwxyz") {
		// The first argument is a variable/expression, not a literal.
		return ""
	}
	if args[idx] != '"' {
		return ""
	}
	end := idx + 1
	for end < len(args) {
		if args[end] == '\\' {
			end += 2
			continue
		}
		if args[end] == '"' {
			return args[idx+1 : end]
		}
		end++
	}
	return ""
}

func literalContainsText(lit string) bool {
	runes := []rune(lit)
	letters := 0
	for _, r := range runes {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			letters++
		}
	}
	return letters >= 2
}

func TestWindowTitleLocalized(t *testing.T) {
	data, err := os.ReadFile("main_windows.go")
	if err != nil {
		t.Fatalf("read main_windows.go: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, `utf16Ptr(tr("app.title"))`) {
		t.Error("window title must be localised via tr(\"app.title\")")
	}
	if strings.Contains(src, `utf16Ptr("Republic Mod Loader")`) {
		t.Error("window title still hard-coded to the upstream name")
	}
}
