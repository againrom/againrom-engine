package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

// A second-game mission whose text the editor cannot read takes no new
// dialog: the add is refused with the note's words, the text stays empty and
// a save into a mod writes no text file.
func TestSecondMapEditorNewDialogNeedsAReadableMissionText(t *testing.T) {
	host, _ := secondEventTextHost(t, "#briefing\r\nGo.\r\n#event5\r\nX\x98\r\n")
	_ = host.WorkspaceFields("pages")
	if host.eventNote == "" {
		t.Fatal("the mission text reads; the fixture needs the note")
	}
	owner := ui.InspectionMessageOwner{Kind: ui.KindAction, Index: 0, Key: "value0"}
	if _, err := host.NewMessage(owner); err == nil || err.Error() != host.eventNote {
		t.Fatalf("New dialog on an unread mission text: %v", err)
	}
	if len(host.workspace.Pages[host.workspace.Sections]) != 0 {
		t.Fatalf("the refused add left text %q", host.workspace.Pages[host.workspace.Sections])
	}
	modDir := filepath.Join(t.TempDir(), "mine")
	writeTestModManifest(t, modDir)
	if _, err := host.SaveAs(filepath.Join(modDir, "maps", "m10.alm")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(modDir, "files", "en", "main", "text", "mission10.txt")); err == nil {
		t.Fatal("a save wrote a mission text that was never read")
	}
}

// Copying a first-game trigger whose page the workspace holds in the code
// page keeps the page's bytes, also bytes that are not UTF-8.
func TestTriggerCopyKeepsTheBytesOfACodePagePage(t *testing.T) {
	e, _, _ := copyPagesEditor(t, scriptAddLangs[1], false)
	s := copyEdits(t, e)
	raw := []byte("<part=1>\n\xcf\xf0\xe8\xe2\xe5\xf2\n")
	s.workspace.Pages["event01.txt"] = bytes.Clone(raw)
	copyKey(t, e, 0)
	if got := s.workspace.Pages["event03.txt"]; !bytes.Equal(got, raw) {
		t.Fatalf("the copy's page is %q, want %q", got, raw)
	}
}
