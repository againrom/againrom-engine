package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

func TestRefusedModSaveKeepsEditsAndSaveAsWritesThem(t *testing.T) {
	_, e := workspaceFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.AddRecord(ui.InspectionNewRecord{Kind: "Message"}); err != nil {
		t.Fatal(err)
	}
	s.inspector.mod = &editorMod{dir: t.TempDir(), id: "x"}
	model, pages := bytes.Clone(s.model.Bytes()), len(s.workspace.Pages)
	if !s.Dirty() || pages != 1 {
		t.Fatal("setup", s.Dirty(), pages)
	}
	if _, err := s.SaveIntoMod(); err == nil {
		t.Fatal("the mod save was not refused")
	}
	if !s.Dirty() || !bytes.Equal(model, s.model.Bytes()) || len(s.workspace.Pages) != pages {
		t.Fatal("the refusal changed the edits")
	}
	s.inspector.mod = nil
	out := filepath.Join(t.TempDir(), "out.alm")
	if _, err := s.SaveAs(out); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(out); err != nil || len(b) == 0 || s.Dirty() {
		t.Fatalf("Save As: %v, %d bytes, dirty %v", err, len(b), s.Dirty())
	}
}
