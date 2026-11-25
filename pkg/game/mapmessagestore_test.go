package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// A second-game map makes a new message as the first game's does: the
// send-event action names the first free event number, a new section at the
// end of the mission text holds it, its target opens that section, and one
// undo takes back both. A number whose header the game's substring lookup
// finds first is skipped (R2-ENGINE-050). The edited section is saved into
// the mod as the rest of the text is.
func TestSecondMapEditorCreatesAMessage(t *testing.T) {
	const file = "main/text/mission10.txt"
	host, _ := secondEventTextHost(t, "#event10\r\nTen.\r\n"+eventTextFixture)
	before, text := bytes.Clone(host.model.Bytes()), string(host.workspace.Pages[file])
	d, err := host.SetField(ui.KindAction, 0, "value0", ui.NewMessageChoice)
	if err != nil || d == nil {
		t.Fatalf("new message: %v", err)
	}
	if got := string(host.workspace.Pages[file]); got != text+"#event2\r\n\r\n" {
		t.Fatalf("new section %q", got)
	}
	if v := d.Script.Actions[0].Values[0]; v != 2 {
		t.Fatalf("the action sends event %d, want 2", v)
	}
	if target := d.Script.OpenTarget(ui.ScriptMessageParameter, 2); target == nil || target.Page != "event2" {
		t.Fatalf("event 2's target %+v", target)
	}
	if _, err := host.Undo(); err != nil || !bytes.Equal(host.model.Bytes(), before) || string(host.workspace.Pages[file]) != text {
		t.Fatalf("one undo leaves the map or the text changed: %v", err)
	}
	if _, err := host.Redo(); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetWorkspaceField("pages", "page", "event2"); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetWorkspaceField("pages", "body", "Run."); err != nil {
		t.Fatal(err)
	}
	modDir := filepath.Join(t.TempDir(), "mine")
	writeTestModManifest(t, modDir)
	if _, err := host.SaveAs(filepath.Join(modDir, "maps", "m10.alm")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(modDir, "files", "en", filepath.FromSlash(file)))
	if err != nil || string(b) != text+"#event2\r\nRun.\r\n" {
		t.Fatalf("the mod's text %q: %v", b, err)
	}
}

// A letter the font cannot show, and a NUL, are refused in both games'
// pages, by one check, in the same words.
func TestBothMapEditorsRefuseALetterTheFontCannotShowAlike(t *testing.T) {
	second, _ := secondEventTextHost(t, eventTextFixture)
	_, e := workspaceFixture(t)
	first := e.Document().Edits.(*mapInspectionEdits)
	for _, text := range []string{"日", "a\x00b"} {
		var refusals []string
		for _, row := range []struct {
			host *mapInspectionEdits
			page string
		}{{first, "event01.txt"}, {second, "event5"}} {
			if _, err := row.host.SetWorkspaceField("pages", "page", row.page); err != nil {
				t.Fatal(err)
			}
			image := bytes.Clone(row.host.workspaceImage)
			_, err := row.host.SetWorkspaceField("pages", "body", text)
			if err == nil || !bytes.Equal(image, row.host.workspaceImage) {
				t.Fatalf("%s: %q is kept: %v", row.page, text, err)
			}
			refusals = append(refusals, err.Error())
		}
		if refusals[0] != refusals[1] {
			t.Fatalf("the games refuse %q in other words: %q", text, refusals)
		}
	}
}

// The first game's pages window lists the pages the map holds, each with the
// actions that show it, and a page's number in the list opens it, as the
// second game's lists its sections.
func TestFirstMapPagesWindowListsItsPagesAndTheirReaders(t *testing.T) {
	_, e := workspaceFixture(t)
	workspacePage(t, e, "event02.txt", "Second page")
	workspacePage(t, e, "briefing.txt", "Go north")
	s := e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.AddScript("Action"); err != nil {
		t.Fatal(err)
	}
	action := s.model.Count(mapedit.KindAction) - 1
	if _, err := s.SetField("Action", action, "value0", 2); err != nil {
		t.Fatal(err)
	}
	x := s.inspector
	var labels []string
	for _, c := range s.WorkspaceFields("pages")[0].Choices {
		labels = append(labels, c.Label)
	}
	want := []string{x.inspectionUTF8("briefing.txt: " + x.word("editor.text.unnamed")), x.inspectionUTF8("event02.txt: " + fmt.Sprintf(x.word("editor.node.action"), action+1))}
	if !slices.Equal(labels, want) {
		t.Fatalf("page choices %q, want %q", labels, want)
	}
	if _, err := s.SetWorkspaceField("pages", "page", "2"); err != nil {
		t.Fatal(err)
	}
	if body := s.WorkspaceFields("pages")[1].TextValue; body != "Second page" {
		t.Fatalf("page 2 shows %q", body)
	}
}
