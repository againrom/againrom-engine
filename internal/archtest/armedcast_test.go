package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestArmedCastFieldsWrittenOnlyByItsHome: the armed Cast fields are written
// in pkg/ui only by the methods of armedCast in armedcast.go.
func TestArmedCastFieldsWrittenOnlyByItsHome(t *testing.T) {
	dir := filepath.Join(moduleRootForTest(t), "pkg", "ui")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	write := regexp.MustCompile(`\.(spellArmed|itemCast)\s*(=[^=]|,)`)
	var hits []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "armedcast.go" {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if write.MatchString(line) {
				hits = append(hits, name+":"+strconv.Itoa(i+1))
			}
		}
	}
	if len(hits) != 0 {
		t.Fatalf("armed cast fields written outside armedcast.go: %v", hits)
	}
}
