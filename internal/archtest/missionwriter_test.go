package archtest

import (
	"os"
	"strings"
	"testing"
)

// The editor's missions and towns files are written through their kinds'
// writers: no game source calls the row writers itself.
func TestEditorRowFilesAreWrittenThroughTheirKinds(t *testing.T) {
	entries, err := os.ReadDir("../../pkg/game")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile("../../pkg/game/" + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, call := range []string{"mod.WriteMissions(", "mod.WriteOfferRows(", `"[[mission]]`} {
			if strings.Contains(string(src), call) {
				t.Fatalf("%s uses %s; write through the kind", e.Name(), call)
			}
		}
	}
}

// No game source spells an offer table.
func TestEditorTownsDraftSpellsNoOfferTable(t *testing.T) {
	entries, err := os.ReadDir("../../pkg/game")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile("../../pkg/game/" + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{`"[[tavern]]`, `"[[shop]]`, `"[[school]]`} {
			if strings.Contains(string(src), table) {
				t.Fatalf("%s writes %s itself; use mod.WriteOfferRows", e.Name(), table)
			}
		}
	}
}

func TestModWindowUsesTheSharedEditorRows(t *testing.T) {
	b, err := os.ReadFile("../../pkg/ui/mapeditormod.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "e.objectFieldRows(") || strings.Contains(s, "drawPushButton(") || strings.Contains(s, "func draw") || strings.Contains(s, "NewPicker(") {
		t.Fatal("mod window must use the shared popup rows and choices")
	}
}
