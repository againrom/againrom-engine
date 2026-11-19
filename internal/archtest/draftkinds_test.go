package archtest

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func draftKindSources(t *testing.T) map[string]string {
	t.Helper()
	root := moduleRootForTest(t)
	files := map[string]string{}
	for _, path := range append([]string{DraftKindTable}, slices.Collect(maps.Keys(DraftKindReaders))...) {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		files[path] = string(b)
	}
	return files
}

func TestDraftedKindIsOneDataKindsRow(t *testing.T) {
	got, err := DraftKindFindings(draftKindSources(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got {
		t.Error(f)
	}
}

func TestDraftedKindBranchesAreFound(t *testing.T) {
	files := draftKindSources(t)
	replace := func(path, old, new string) {
		t.Helper()
		if !strings.Contains(files[path], old) {
			t.Fatalf("%s lacks %q", path, old)
		}
		files[path] = strings.Replace(files[path], old, new, 1)
	}
	replace(DraftKindTable, "Draft: Drafting{Context: DraftAlways, Held: true}, draft: draftSpeakers,", "draft: draftSpeakers,")
	replace("pkg/game/mapinspectionworkspace.go", "if !mod.RebuildsContext(name, holds) {", "if !mod.RebuildsContext(name, holds) && name != mod.TerrainFile {")
	replace("pkg/game/modassetimport.go", "|| mod.HeldDraft(name)", "|| mod.HeldDraft(name) || name == mod.SpeakersFile")
	replace("pkg/game/mapeditormod.go", "for _, rel := range k.DraftNames() {", "for _, rel := range append(k.DraftNames(), mod.StringsFile(\"en\")) {")
	got, err := DraftKindFindings(files)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"row speakers has a draft reader and no Draft", "contextDraft names mod.TerrainFile", "importedModFile names mod.SpeakersFile", "document names mod.StringsFile"} {
		if !strings.Contains(joined, want) {
			t.Errorf("findings %q lack %q", joined, want)
		}
	}
}
