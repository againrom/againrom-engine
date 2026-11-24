package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestManifestListSpanHasOneScanner: the manifest and data readers and the
// manifest writer cross a list's lines through continueList and scanLine
// only. A second bracket or comment scanner in pkg/mod is a second route.
func TestManifestListSpanHasOneScanner(t *testing.T) {
	dir := filepath.Join(moduleRootForTest(t), "pkg", "mod")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	decl := regexp.MustCompile(`(?m)^func (manifestValueEnd|bracketDepth|stripComment)\(`)
	calls := regexp.MustCompile(`\b(manifestValueEnd|bracketDepth|stripComment)\(`)
	var hits []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range decl.FindAllString(string(src), -1) {
			hits = append(hits, name+": "+m)
		}
		for _, m := range calls.FindAllString(string(src), -1) {
			hits = append(hits, name+": "+m)
		}
	}
	if len(hits) != 0 {
		t.Fatalf("second list scanner route in pkg/mod:\n%s", strings.Join(hits, "\n"))
	}
}
