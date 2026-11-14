package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func draftViewSources(t *testing.T) map[string]string {
	t.Helper()
	root := moduleRootForTest(t)
	files := map[string]string{}
	for path := range DraftViewFiles {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		files[path] = string(b)
	}
	return files
}

func TestDraftReadersReachTheDraftsThroughProjectView(t *testing.T) {
	got, err := DraftViewFindings(draftViewSources(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got {
		t.Error(f)
	}
}

func TestDraftViewSecondRoutesAreFound(t *testing.T) {
	files := draftViewSources(t)
	files["pkg/game/modmapsave.go"] = strings.Replace(files["pkg/game/modmapsave.go"],
		"files = projectView(files, drafts).names", "files = slices.DeleteFunc(files, func(string) bool { return drafts == nil })", 1)
	files["pkg/game/mapeditormodcontext.go"] = strings.Replace(files["pkg/game/mapeditormodcontext.go"],
		"root string, view draftView)", "root string, view draftView, extra map[string][]byte)", 1)
	files["pkg/game/mapeditormod.go"] = strings.Replace(files["pkg/game/mapeditormod.go"],
		"rows []mod.Mission, view draftView)", "rows []mod.Mission, pending map[string][]byte)", 1)
	files["pkg/game/mapeditorcampaign.go"] = strings.Replace(files["pkg/game/mapeditorcampaign.go"],
		"projectView(x.mod.files, d.Files)", "d.Files", 1)
	got, err := DraftViewFindings(files)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"readInventory deletes a name itself", "readInventory does not call projectView", "openModContext takes a draft map", "missionFiles takes a draft map", "missionDraft does not call projectView"} {
		if !strings.Contains(joined, want) {
			t.Errorf("findings %q lack %q", joined, want)
		}
	}
}
