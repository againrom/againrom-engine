package game

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

// draftKindSamples is a file of the kind k under each of its names: its
// first fixed name and one file under each folder its paths name.
func draftKindSamples(k mod.DataKind) []string {
	var out []string
	if names := k.DraftNames(); len(names) > 0 {
		out = append(out, names[0])
	}
	for _, p := range k.Paths {
		if strings.Contains(p, "*") {
			out = append(out, strings.NewReplacer("{layer}", mod.CommonFiles, "**/", "", "*", "sample").Replace(p))
		}
	}
	return out
}

// Each kind the editor drafts is one row of mod.DraftKinds, and each reader
// of the drafts reads it from the row: the document's load holds a held
// kind's file, a draft of a kind the run reads builds the context again (a
// cited kind's only while the project holds the files citing it), the save
// drops a held kind's file the project no longer holds, and the page
// transaction writes only a held kind's files.
func TestEveryDraftedKindRowIsReadByEachReader(t *testing.T) {
	kinds := mod.DraftKinds()
	if len(kinds) == 0 {
		t.Fatal("no kind is drafted")
	}
	var all []string
	for _, k := range kinds {
		all = append(all, draftKindSamples(k)...)
	}
	for _, k := range kinds {
		for _, rel := range draftKindSamples(k) {
			t.Run(rel, func(t *testing.T) { readsDraftedKindRow(t, k, rel, all) })
		}
	}
}

// readsDraftedKindRow holds each reader of the drafts to the row of k for
// its file rel; all is a file of every drafted kind.
func readsDraftedKindRow(t *testing.T, k mod.DataKind, rel string, all []string) {
	if !k.DraftFile(rel) {
		t.Fatalf("the sample %s is not a file of the kind", rel)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("folder"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &editorMod{id: "m", dir: dir, files: []string{rel}}
	if got := m.document().Files[rel]; k.Draft.Held && string(got) != "folder" {
		t.Errorf("the load holds %q of %s", got, rel)
	}

	rebuilds := func(held []string) bool {
		w := &editorWorkspaceState{Project: &editorModDocument{Files: map[string][]byte{}}}
		for _, name := range held {
			w.Project.Files[name] = []byte("held")
		}
		before := w.contextDraft()
		w.Project.Files[rel] = []byte("draft")
		return !bytes.Equal(before, w.contextDraft())
	}
	switch k.Draft.Context {
	case mod.DraftNone:
		if rebuilds(all) {
			t.Errorf("a draft of %s builds the context again", rel)
		}
	case mod.DraftAlways:
		if !rebuilds(nil) {
			t.Errorf("a draft of %s leaves the context", rel)
		}
	case mod.DraftCited:
		if rebuilds(nil) || !rebuilds(all) {
			t.Errorf("a draft of %s builds the context again alone %t, with the citing files %t", rel, rebuilds(nil), rebuilds(all))
		}
	}

	if importedModFile(rel) != (k.Draft.Held || strings.HasPrefix(rel, mod.AssetsFolder+"/")) {
		t.Errorf("the save's drop of %s is %t", rel, importedModFile(rel))
	}

	w := &editorWorkspaceState{Project: &editorModDocument{Files: map[string][]byte{}}}
	files := kindFiles{s: &mapInspectionEdits{inspector: &MapInspector{mod: m}}, w: w}
	if err := files.put(rel, []byte("draft")); (err == nil) != k.Draft.Held {
		t.Errorf("the page transaction puts %s: %v", rel, err)
	}
	if err := files.drop(rel); (err == nil) != k.Draft.Held {
		t.Errorf("the page transaction drops %s: %v", rel, err)
	}
	if b, ok := w.Project.Files[rel]; k.Draft.Held && (!ok || b != nil) {
		t.Errorf("the dropped %s is %q, %t; want a deletion", rel, b, ok)
	}
}
