package game

import (
	"bytes"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

const townsKeysHeld = `[[town]]
id = 1
opens-after = 0
cite = ["held by hand"]
`

// townsKeysEdit is one edit of town 1 by the page's one town step.
func townsKeysEdit(s *mapInspectionEdits, field, value string) error {
	_, err := s.changeWorkspace(func(w *editorWorkspaceState) error { return s.editTown(w, s.townsBook(), 1, field, value) })
	return err
}

// Switch on gives a town back the table the towns file holds, with every key
// the modder wrote by hand; off and on again leaves the file as it was.
func TestModTownsPageSwitchOnKeepsTheKeysWrittenByHand(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			_, s, _ := handoutEditFixture(t, entry, townsKeysHeld)
			if len(s.workspace.Project.TownRecords) != 1 {
				t.Fatalf("records %+v", s.workspace.Project.TownRecords)
			}
			if err := townsKeysEdit(s, "off", ""); err != nil {
				t.Fatal(err)
			}
			if r := s.workspace.Project.TownRecords; len(r) != 1 || !r[0].Off {
				t.Fatalf("switched off: %+v", r)
			}
			if err := townsKeysEdit(s, "on", ""); err != nil {
				t.Fatal(err)
			}
			d := s.workspace.Project
			if len(d.TownRecords) != 1 || d.TownRecords[0].Off || len(d.TownRecords[0].Fields) != 2 {
				t.Fatalf("switched on: %+v", d.TownRecords)
			}
			src, changed, err := d.townsDraft()
			if err != nil || changed || !bytes.Equal(src, d.TownsSource) {
				t.Fatalf("the file after off and on (changed %v, %v):\n%s", changed, err, src)
			}
			records, _, err := mod.ParseTowns("m", mod.TownsFile, src)
			if err != nil || len(records) != 1 || len(records[0].Entry.Cite) != 1 {
				t.Fatalf("reads back %+v: %v", records, err)
			}
		})
	}
}

// A town the file holds no table of goes back to the game's with nothing
// written: the shipped case writes the bytes it wrote before.
func TestModTownsPageSwitchOnOfATownTheFileDoesNotHold(t *testing.T) {
	_, s, _ := handoutEditFixture(t, "english", "")
	for _, field := range []string{"off", "on"} {
		if err := townsKeysEdit(s, field, ""); err != nil {
			t.Fatalf("%s: %v", field, err)
		}
	}
	d := s.workspace.Project
	if len(d.TownRecords) != 0 {
		t.Fatalf("records %+v", d.TownRecords)
	}
	if _, changed, err := d.townsDraft(); err != nil || changed {
		t.Fatalf("towns draft changed %v: %v", changed, err)
	}
}

// Emptying one number of a map point leaves the point as written and says
// why, in each language; a number typed over it moves the point.
func TestModTownsPageRefusesAHalfEmptiedMapPoint(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			_, s, _ := handoutEditFixture(t, entry, townsKeysHeld)
			for _, c := range [][2]string{{"x", "10"}, {"y", "20"}} {
				if err := townsKeysEdit(s, c[0], c[1]); err != nil {
					t.Fatal(err)
				}
			}
			before := workspaceBytes(s.workspace)
			want := map[string]string{"english": "both numbers", "russian": "оба числа"}[entry]
			for _, field := range []string{"x", "y"} {
				if err := townsKeysEdit(s, field, ""); err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("%s emptied: %v", field, err)
				}
				if !bytes.Equal(before, workspaceBytes(s.workspace)) {
					t.Fatalf("%s emptied changed the draft", field)
				}
			}
			if err := townsKeysEdit(s, "x", "12"); err != nil {
				t.Fatal(err)
			}
			src, _, err := s.workspace.Project.townsDraft()
			if err != nil || !strings.Contains(string(src), `point = [12, 20]`) || !strings.Contains(string(src), `cite = ["held by hand"]`) {
				t.Fatalf("the file:\n%s\n%v", src, err)
			}
		})
	}
}
