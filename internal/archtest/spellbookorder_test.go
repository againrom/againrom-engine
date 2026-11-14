package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The rule-by-ID map that orders the book is built only in pkg/game's bookOrder.
func TestBookCellOrderHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	byID := regexp.MustCompile(`map\[\w+\]\s*\*?sim\.SpellRule`)
	var homes []string
	for file, src := range files {
		file = filepath.ToSlash(file)
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if byID.MatchString(src) {
			homes = append(homes, file)
		}
	}
	if len(homes) != 1 || !strings.HasSuffix(homes[0], "pkg/game/spell.go") {
		t.Fatalf("the cell order map is built in %v, want only pkg/game/spell.go", homes)
	}
}
