package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
)

// TestReleaseTriggerWizardReadsEveryShippedRowAsAnItem opens every trigger
// of every installed map in the wizard at 1920x1080: every row reads as a
// wizard item except the pairs of two IDs no node has and the distance check
// over two units, which stay Other; the map saved without an edit carries
// its script record byte for byte.
func TestReleaseTriggerWizardReadsEveryShippedRowAsAnItem(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	triggers, rows, unbuilt, twoUnits := 0, 0, 0, 0
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		e.Layout(1920, 1080)
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		before := append([]byte(nil), e.Document().SourceBytes...)
		sc := wizScript(t, before)
		for i, tr := range sc.Triggers {
			want := 0
			for p := range tr.Left {
				switch l, r := tr.Left[p], tr.Right[p]; {
				case l == 0 && r == 0:
				case l == 0xFFFFFFFF && r == 0xFFFFFFFF:
					want++
					unbuilt++
				case releaseDistanceOverTwoUnits(sc.Conditions, l) || releaseDistanceOverTwoUnits(sc.Conditions, r):
					want++
					twoUnits++
				}
			}
			e.Select(trigRecord(t, e, "Trigger", i), true)
			n, kept := e.WizardRowCounts()
			triggers++
			rows += n
			if kept != want {
				t.Errorf("%s trigger %d %q: %d of %d rows read as Other, want %d:\n%s", key, i, tr.Name, kept, n, want, trigLines(e))
			}
		}
		s := e.Document().Edits.(*mapInspectionEdits)
		path := filepath.Join(t.TempDir(), "saved.alm")
		if _, err := s.SaveAs(path); err != nil {
			t.Fatalf("%s: Save As refused: %v", key, err)
		}
		saved, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if a, b := releaseScriptRecord(t, before), releaseScriptRecord(t, saved); !bytes.Equal(a, b) {
			t.Errorf("%s: the script record saved without an edit differs (%d bytes, was %d)", key, len(b), len(a))
		}
	}
	t.Logf("%d triggers, %d rows; Other: %d pairs of unbuilt IDs, %d distance checks over two units", triggers, rows, unbuilt, twoUnits)
	if triggers == 0 {
		t.Fatal("no trigger read")
	}
}

// releaseDistanceOverTwoUnits reports whether the condition id is the
// cell-to-unit distance check written over two units.
func releaseDistanceOverTwoUnits(nodes []alm.ScriptNode, id uint32) bool {
	for _, n := range nodes {
		if n.ID == id {
			return n.Opcode == 7 && n.Type[0] == 4 && n.Type[1] == 4
		}
	}
	return false
}

// releaseScriptRecord is the payload of the map's script record, nil when
// it has none.
func releaseScriptRecord(t *testing.T, b []byte) []byte {
	t.Helper()
	d, err := alm.OpenDocument(b)
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range d.RecordTypeIDs() {
		if id == alm.TypeScript {
			return d.RecordPayload(i)
		}
	}
	return nil
}
