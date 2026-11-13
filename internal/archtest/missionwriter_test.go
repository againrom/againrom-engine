package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestEditorMissionEmitterUsesTheSharedWriter(t *testing.T) {
	b, err := os.ReadFile("../../pkg/game/mapedittestplay.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "mod.WriteMissions(") || strings.Contains(s, `"[[mission]]`) {
		t.Fatal("editor missions must use the shared declaration writer")
	}
}

// The editor's towns draft is written by the one row writer: the project
// document asks mod.WriteOfferRows, and no game source spells an offer table.
func TestEditorTownsDraftUsesTheSharedRowWriter(t *testing.T) {
	b, err := os.ReadFile("../../pkg/game/mapeditormod.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "mod.WriteOfferRows(") {
		t.Fatal("the towns draft must use the shared row writer")
	}
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
