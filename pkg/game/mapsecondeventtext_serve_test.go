package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/locale"
)

// A section body is accepted only when the game shows it as written: the
// characters whose font bytes do not survive the mod text file's serve and
// read-back are refused by name, and the text that survives is accepted.
func TestSecondMapEventTextCodecNamesWhatTheGameShowsAsOtherLetters(t *testing.T) {
	ru := TextCode{From: locale.WindowsCyrillic, To: locale.DOSCyrillic}
	got := string(ru.ModTextUnserved([]byte("Знак № 5, 10°, a·b, є ї"), 1))
	if got != "№°·єї" {
		t.Fatalf("unserved runes %q, want the five", got)
	}
	if lost := ru.ModTextUnserved([]byte("Сундук открыт. Ё ё"), 1); len(lost) != 0 {
		t.Fatalf("text that survives lost %q", string(lost))
	}
	if lost := (TextCode{}).ModTextUnserved([]byte("Plain text"), 0); len(lost) != 0 {
		t.Fatalf("English text lost %q", string(lost))
	}
}

// An edited text saved into one mod and then, with Save As, into another is
// written into the second as well: the text written follows what the target
// mod plus the install serve, not the last save.
func TestSecondMapEditedMissionTextFollowsTheTargetMod(t *testing.T) {
	host, _ := secondEventTextHost(t, eventTextFixture)
	if _, err := host.SetWorkspaceField("pages", "page", "event7"); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetWorkspaceField("pages", "body", "Edited."); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	want := "#briefing\r\nGo north.\r\n#event5\r\nThe gate opens.\r\nRun!\r\n#event7\r\nEdited.\r\n"
	for _, name := range []string{"first", "second"} {
		folder := filepath.Join(dir, name)
		writeTestModManifest(t, folder)
		if _, err := host.SaveAs(filepath.Join(folder, "maps", "m10.alm")); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(folder, "files", "en", "main", "text", "mission10.txt"))
		if err != nil || string(b) != want {
			t.Fatalf("mod %s holds %q, %v", name, b, err)
		}
	}
	// A text equal to the install's writes no file.
	plain, _ := secondEventTextHost(t, eventTextFixture)
	folder := filepath.Join(dir, "plain")
	writeTestModManifest(t, folder)
	if _, err := plain.SaveAs(filepath.Join(folder, "maps", "m10.alm")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(folder, "files")); err == nil {
		t.Fatal("an untouched text wrote a file")
	}
}
