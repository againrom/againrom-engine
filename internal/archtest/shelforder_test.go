package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShelfOrderHasOneKeyBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pkg/game/shopsort.go", "pkg/secondshop/shownorder.go"} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "shopstock.ShelfOrder[") || strings.Contains(string(b), "shopstock.SortBy(") {
			t.Errorf("%s must supply client keys to shopstock.ShelfOrder", name)
		}
	}
}
