package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The manifest, the row files, the speakers and the items share one TOML pair
// reader and one pair writer: no second value scanner stays in pkg/mod.
func TestModTOMLHasOnePairReaderAndWriter(t *testing.T) {
	files, err := filepath.Glob("../../pkg/mod/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, gone := range []string{"func rewriteSpan(", "func parseStrings(", "func parseString(", "func parseInt("} {
			if strings.Contains(string(src), gone) {
				t.Errorf("%s: %s is a second route; use parseTOMLWith and rewritePair", file, gone)
			}
		}
		if !strings.HasSuffix(file, "toml.go") && strings.Contains(string(src), "func rewritePair(") {
			t.Errorf("%s holds a second rewritePair", file)
		}
		if strings.HasSuffix(file, "manifestwrite.go") {
			text := string(src)
			for _, need := range []string{"parseTOMLWith(", "rewritePair("} {
				if !strings.Contains(text, need) {
					t.Errorf("%s must use %s", file, need)
				}
			}
			for _, scan := range []string{"strings.IndexByte(", "strings.HasPrefix(", "strings.TrimSpace("} {
				if strings.Contains(text, scan) {
					t.Errorf("%s: %s is a second line scanner; use parseTOMLWith", file, scan)
				}
			}
		}
	}
}
