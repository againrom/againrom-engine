package archtest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	for _, v := range checkPlayerCharacterPredicates(loadPartyGuardSources(t, root)) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}

func loadPartyGuardSources(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
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
			files[filepath.ToSlash(rel)] = string(b)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return files
}

func checkPlayerCharacterPredicates(files map[string]string) []Violation {
	copyForm := regexp.MustCompile(`PlayerCharacter\s*&&\s*![\w.]+\.Membership\(\)\.Is\((?:\w+\.)?TavernMember\)|![\w.]+\.Membership\(\)\.Is\((?:\w+\.)?TavernMember\)\s*&&\s*[\w.]+\.PlayerCharacter|PlayerCharacter\s*&&[^\n]*MercenaryType\s*==\s*0|MercenaryType\s*==\s*0\s*&&[^\n]*PlayerCharacter`)
	const definition = "pkg/game/speakeractors.go"
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []Violation
	found := 0
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		for _, loc := range copyForm.FindAllStringIndex(files[name], -1) {
			if name == definition {
				found++
				continue
			}
			out = append(out, Violation{From: name, Reason: fmt.Sprintf("a copy of the hero predicate at byte %d; call playerCharacter", loc[0])})
		}
	}
	if found != 1 {
		out = append(out, Violation{From: definition, Reason: fmt.Sprintf("the predicate is defined %d times, want 1", found)})
	}
	return out
}

func TestPlayerCharacterPredicateNamesPlantedCopy(t *testing.T) {
	const definition = "return p.PlayerCharacter && !p.Membership().Is(mapload.TavernMember)"
	for _, copy := range []struct{ name, file, src string }{
		{"membership", "pkg/game/plantedhero.go", definition},
		{"reversed membership", "cmd/plantedhero/main.go", "return !p.Membership().Is(mapload.TavernMember) && p.PlayerCharacter"},
		{"legacy", "pkg/game/plantedhero.go", "return p.PlayerCharacter && p.MercenaryType == 0"},
	} {
		t.Run(copy.name, func(t *testing.T) {
			files := map[string]string{"pkg/game/speakeractors.go": definition}
			if vs := checkPlayerCharacterPredicates(files); len(vs) != 0 {
				t.Fatalf("one definition: %v", vs)
			}
			files[copy.file] = copy.src
			vs := checkPlayerCharacterPredicates(files)
			if len(vs) != 1 || vs[0].From != copy.file || !strings.Contains(vs[0].Reason, "copy of the hero predicate") {
				t.Fatalf("planted copy %s: %v", copy.file, vs)
			}
			t.Logf("rejected %s: %s", vs[0].From, vs[0].Reason)
		})
	}
}
