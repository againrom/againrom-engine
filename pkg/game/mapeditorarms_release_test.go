package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

type expectedArmLink struct {
	arm, kind string
	index     int
}

func expectedArmLinks(sc alm.Script) []expectedArmLink {
	var out []expectedArmLink
	seen := map[string]bool{}
	add := func(arm string, node, trigger int) {
		for slot, tag := range sc.Conditions[node].Type {
			if tag != 2 && tag != 4 && tag != 9 {
				continue
			}
			key := fmt.Sprintf("%s/%d/%d", arm, node, slot)
			if seen[key] {
				continue
			}
			seen[key] = true
			link := expectedArmLink{arm, "Condition", node}
			if trigger >= 0 {
				link.kind, link.index = "Trigger", trigger
			}
			out = append(out, link)
		}
	}
	for ti, tr := range sc.Triggers {
		for _, aid := range tr.Acts {
			var op uint32
			for _, a := range sc.Actions {
				if a.ID == aid {
					op = a.Opcode
					break
				}
			}
			arm := map[uint32]string{4: "Win", 5: "Lose"}[op]
			if arm == "" {
				continue
			}
			for pair := range tr.Left {
				for _, id := range []uint32{tr.Left[pair], tr.Right[pair]} {
					if id == 0 {
						continue
					}
					for ci, c := range sc.Conditions {
						if c.ID == id {
							add(arm, ci, ti)
							break
						}
					}
				}
			}
		}
	}
	for ci, c := range sc.Conditions {
		if c.Opcode == 18 {
			add("Protect", ci, -1)
		}
	}
	return out
}

func armOpenButton(t *testing.T, e *ui.MapEditor, row int) image.Rectangle {
	t.Helper()
	for i := 0; i < 200; i++ {
		if r, ok := e.GoalOpenRect(row); ok {
			return r
		}
		_ = e.Key("pagedown")
	}
	t.Fatalf("arm row %d has no reachable Open", row)
	return image.Rectangle{}
}

func backFromArm(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	for n := 0; n < 8; n++ {
		r, ok := e.PopupButtonRect(n)
		if !ok {
			if n == 0 {
				t.Fatal("popup missing")
			}
			r, ok = e.PopupButtonRect(n - 1)
			trigClick(t, e, r, ok, "Back")
			return
		}
	}
	t.Fatal("Back button not found")
}

func armSaveUnchanged(t *testing.T, e *ui.MapEditor, source []byte) {
	t.Helper()
	if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, source) {
		t.Fatal("opening changed bytes")
	}
	path := filepath.Join(t.TempDir(), "opened-arm.alm")
	if _, err := e.Document().Edits.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(saved, source) {
		t.Fatalf("Save As after Open: %v", err)
	}
}

func TestReleaseMapEditorArmsOpenAndKeepBytes(t *testing.T) {
	x := installedInspector(t)
	keys, sources := installedMapBytes(t, x)
	counts := map[string]int{}
	maps, drops := 0, 0
	for mi, key := range keys {
		t.Run(key, func(t *testing.T) {
			e := x.Editor()
			e.Layout(1280, 800)
			if err := e.Open(key); err != nil {
				t.Fatal(err)
			}
			closeMapPanel(t, e)
			m, err := alm.Open(sources[mi])
			if err != nil {
				t.Fatal(err)
			}
			sc, err := m.Script()
			if err != nil {
				t.Fatal(err)
			}
			want := expectedArmLinks(sc)
			if len(want) > 0 {
				maps++
			}
			goals := e.Document().Edits.(ui.InspectionScript).Goals()
			var actual []int
			for row, g := range goals {
				if g.Index >= 0 {
					actual = append(actual, row)
				}
			}
			if len(actual) != len(want) {
				t.Fatalf("arm rows=%d want=%d", len(actual), len(want))
			}
			_ = e.Key("goals")
			for i, w := range want {
				row := actual[i]
				g := goals[row]
				if g.Arm != w.arm || g.OpenKind != w.kind || g.OpenIndex != w.index {
					t.Fatalf("row %d link=%+v want=%+v", row, g, w)
				}
				r := armOpenButton(t, e, row)
				trigClick(t, e, r, true, "arm Open")
				if e.Selected() != recordOf(e.Document(), w.kind, w.index) || len(e.PopupLines()) == 0 {
					t.Fatalf("%s row %d opened wrong popup", w.arm, row)
				}
				armSaveUnchanged(t, e, sources[mi])
				backFromArm(t, e)
				if e.ReportSelected() != row || e.Selected() != recordOf(e.Document(), g.Kind, g.Index) || len(e.PopupLines()) == 0 {
					t.Fatal("Back lost G selection")
				}
				counts[w.arm]++
			}
			dropRow := len(goals)
			for ai, a := range sc.Actions {
				if a.Opcode == 0x10002 {
					rows, _ := e.ReportOpen()
					if dropRow >= len(rows) || rows[dropRow].Cell != image.Pt(int(a.Value[0]), int(a.Value[1])) {
						t.Fatal("wrong displayed Drop cell")
					}
					r := armOpenButton(t, e, dropRow)
					trigClick(t, e, r, true, "drop Open")
					if e.Selected() != recordOf(e.Document(), "Action", ai) {
						t.Fatal("wrong Drop holder")
					}
					armSaveUnchanged(t, e, sources[mi])
					backFromArm(t, e)
					dropRow++
					drops++
				}
			}
			if e.Dirty() {
				t.Fatal("opening dirtied map")
			}
			_, _, saved := areaSaveAs(t, e, "arms-untouched.alm")
			if !bytes.Equal(saved, sources[mi]) {
				t.Fatal("Save As changed bytes")
			}
		})
	}
	t.Logf("root=%s maps=%d mapsWithArms=%d Win=%d Lose=%d Protect=%d Drop=%d", os.Getenv("AGAINROM_ASSETS"), len(keys), maps, counts["Win"], counts["Lose"], counts["Protect"], drops)
}

func TestReleaseMapEditorArmsFrames(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/131.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	_ = e.Key("goals")
	dir, lang := os.Getenv("AGAINROM_EDITOR_FRAMES"), installLanguage(x)
	writeEditorFrame(t, e, filepath.Join(dir, "arms-panel-"+lang+".png"), dir != "", image.Rectangle{})
	r := armOpenButton(t, e, 0)
	trigClick(t, e, r, true, "arm Open")
	writeEditorFrame(t, e, filepath.Join(dir, "arms-trigger-"+lang+".png"), dir != "", image.Rectangle{})
}
