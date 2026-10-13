package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPlayerCharacterPredicateIsStatedOnce fails on a copy of the hero
// predicate (a player character who is not a hired unit) outside its one
// definition in pkg/game/speakeractors.go.
func TestPlayerCharacterPredicateIsStatedOnce(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	copyForm := regexp.MustCompile(`PlayerCharacter\s*&&[^\n]*MercenaryType\s*==\s*0|MercenaryType\s*==\s*0\s*&&[^\n]*PlayerCharacter`)
	definition := filepath.Join("pkg", "game", "speakeractors.go")
	found := 0
	for _, dir := range []string{"pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			for _, loc := range copyForm.FindAllIndex(b, -1) {
				if rel == definition {
					found++
					continue
				}
				t.Errorf("%s: a copy of the hero predicate at byte %d; call playerCharacter", rel, loc[0])
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if found != 1 {
		t.Errorf("the predicate is defined %d times in %s, want 1", found, definition)
	}
}
