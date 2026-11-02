package game

import (
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/vfs"
)

func TestInstallPagesKeepReadsOncePerSetAndMission(t *testing.T) {
	var k installPagesKeep
	a, b := &vfs.FS{}, &vfs.FS{}
	for range 50 {
		got, err := k.pages(a, 3)
		if err != nil {
			t.Fatal(err)
		}
		got["own.txt"] = []byte("x") // the caller's map is its own
	}
	if k.reads != 1 {
		t.Fatalf("reads=%d, want 1 over many calls", k.reads)
	}
	if got, _ := k.pages(a, 3); len(got) != 0 {
		t.Fatalf("a caller's write reached the keep: %v", got)
	}
	_, _ = k.pages(a, 4)
	_, _ = k.pages(b, 4)
	if k.reads != 3 {
		t.Fatalf("reads=%d after another mission and another archive set, want 3", k.reads)
	}
}

func TestMapReportStampKeptUntilEditOrSecond(t *testing.T) {
	f, e := workspaceFixture(t)
	workspacePage(t, e, "event01.txt", "Before")
	dir := t.TempDir()
	source := filepath.Join(dir, "map.alm")
	if _, err := e.Document().Edits.(*mapInspectionEdits).SaveAs(source); err != nil {
		t.Fatal(err)
	}
	d, err := f.x.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	s := d.Edits.(*mapInspectionEdits)
	now := time.Unix(1000, 0)
	s.clock = func() time.Time { return now }
	f.x.rules = []mapRule{{ID: "page", run: func(c *mapCheck) []mapHit { return []mapHit{{Text: string(c.pages.Files()["event01.txt"])}} }}}
	if got := s.Report(); len(got) != 1 || got[0].Text != "Before" {
		t.Fatalf("initial report=%v", got)
	}
	writeTestPlayInput(t, filepath.Join(dir, "map.text", "en", "event01.txt"), []byte("After!"))
	now = now.Add(900 * time.Millisecond)
	if got := s.Report(); got[0].Text != "Before" {
		t.Fatalf("a disk change inside the second changed the report: %v", got)
	}
	now = now.Add(200 * time.Millisecond)
	if got := s.Report(); got[0].Text != "After!" {
		t.Fatalf("a disk change was not seen after a second: %v", got)
	}
	// An edit made in the editor shows on the next call, inside the second.
	if _, err := s.SetWorkspaceField("pages", "page", "event01.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("pages", "body", "Workspace"); err != nil {
		t.Fatal(err)
	}
	if got := s.Report(); got[0].Text != "Workspace" {
		t.Fatalf("workspace edit not seen at once: %v", got)
	}
}
