package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

func TestArmsPointToTheirScriptRecord(t *testing.T) {
	e, _ := openGoalMap(t, goalMap(t, true))
	s := scriptOf(t, e)
	if _, err := s.SetGoalTarget(len(s.Goals())-1, "Unit", 0); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		row       int
		arm, kind string
		index     int
	}{
		{0, "Win", "Trigger", 0}, {1, "Win", "Trigger", 1}, {2, "Lose", "Trigger", 2}, {3, "Protect", "Condition", 4},
	} {
		t.Run(tc.arm+string(rune('0'+tc.row)), func(t *testing.T) {
			g := s.Goals()[tc.row]
			if g.Arm != tc.arm || g.OpenKind != tc.kind || g.OpenIndex != tc.index {
				t.Fatalf("arm link=%+v, want %s %d", g, tc.kind, tc.index)
			}
		})
	}
}

func TestArmRoutesByteHashes(t *testing.T) {
	for _, route := range []string{"arm", "add", "drop"} {
		t.Run(route, func(t *testing.T) {
			e, _ := openGoalMap(t, goalMap(t, true))
			s := scriptOf(t, e)
			var d *ui.InspectionDocument
			var err error
			switch route {
			case "arm":
				d, err = s.SetGoalTarget(0, "Unit", 3)
			case "add":
				d, err = s.SetGoalTarget(len(s.Goals())-1, "Unit", 0)
			case "drop":
				d, err = s.SetDropCell(image.Pt(8, 9))
			}
			if err != nil || d == nil {
				t.Fatalf("edit: %v", err)
			}
			want := map[string]string{
				"arm":  "4cd407eb0f2d0dca0f91998e8e45ac5ba8606a54e359c0b14754a60ef689af4d",
				"add":  "ae7c136581a8ad208f5a91edb8c434fbe006a975ab051938975cfb1ec9f8c895",
				"drop": "15a378fbd7aa98b907f9626355dd33c80ca346a41a1be55304faef4bfb944756",
			}[route]
			if fmt.Sprintf("%x", sha256.Sum256(d.SourceBytes)) != want {
				t.Fatal("route differs from base bytes")
			}
			driven, _ := openGoalMap(t, goalMap(t, true))
			driven.Layout(1600, 900)
			if route == "drop" {
				// The drop tool moves the map's one zone by a drag; a click
				// on an empty cell would add a second zone.
				_ = driven.Key("drop")
				dragCell(driven, driven.Document().Drops[0], image.Pt(8, 9))
			} else {
				_ = driven.Key("goals")
				cell := image.Pt(21, 14)
				if route == "add" {
					for i := 0; i < 3; i++ {
						_ = driven.Key("down")
					}
					cell = image.Pt(12, 14)
				}
				_ = driven.Key("enter")
				driven.ShowCell(cell)
				clickCell(driven, cell)
			}
			if !bytes.Equal(driven.Document().SourceBytes, d.SourceBytes) {
				t.Fatalf("key route differs from base bytes: row=%d message=%q", driven.ReportSelected(), driven.Message())
			}
			t.Logf("%s SHA256=%x", route, sha256.Sum256(d.SourceBytes))
		})
	}
}

func TestArmPopupAndScriptsEditHaveSameBytesAndUndo(t *testing.T) {
	b := goalMap(t, true)
	e, _ := openGoalMap(t, b)
	e.Layout(1280, 800)
	_ = e.Key("goals")
	r, ok := e.GoalOpenRect(0)
	if !ok {
		t.Fatal("arm Open missing")
	}
	trigClick(t, e, r, true, "arm Open")
	if e.Selected() != recordOf(e.Document(), "Trigger", 0) {
		t.Fatal("wrong trigger")
	}
	r, ok = e.FieldRect(1)
	trigClick(t, e, r, ok, "Fire once")
	r, ok = e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply")
	after := append([]byte(nil), e.Document().SourceBytes...)
	other, _ := openGoalMap(t, b)
	other.Layout(1280, 800)
	_ = other.Key("scripts")
	r, ok = treeMemberRect(other, "scripts", 0)
	trigClick(t, other, r, ok, "Scripts trigger")
	r, ok = other.FieldRect(1)
	trigClick(t, other, r, ok, "Fire once")
	r, ok = other.PopupButtonRect(0)
	trigClick(t, other, r, ok, "Apply")
	if bytes.Equal(after, b) || !bytes.Equal(after, other.Document().SourceBytes) {
		t.Fatal("different popup edit bytes")
	}
	backFromArm(t, e)
	if e.ReportSelected() != 0 || len(e.PopupLines()) != 6 || e.Selected() != recordOf(e.Document(), "Condition", 0) {
		t.Fatal("Back after Apply lost G panel")
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, b) {
		t.Fatalf("Undo: %v", err)
	}
}

func TestArmRowPopupMatchesNodeSelection(t *testing.T) {
	for _, arm := range []string{"Win", "Protect"} {
		t.Run(arm, func(t *testing.T) {
			b := goalMap(t, true)
			row, node := 0, 0
			if arm == "Protect" {
				e, _ := openGoalMap(t, b)
				s := scriptOf(t, e)
				d, err := s.SetGoalTarget(len(s.Goals())-1, "Unit", 0)
				if err != nil {
					t.Fatal(err)
				}
				b, row, node = d.SourceBytes, 3, 4
			}
			e, _ := openGoalMap(t, b)
			want := recordOf(e.Document(), "Condition", node)
			e.Select(want, false)
			lines := e.PopupLines()
			if len(lines) != 6 {
				t.Fatalf("node popup lines=%v", lines)
			}
			_ = e.Key("goals")
			for e.ReportSelected() < row {
				_ = e.Key("down")
			}
			if e.Selected() != want || !reflect.DeepEqual(e.PopupLines(), lines) {
				t.Fatalf("G row hides or changes node: selected=%d want=%d lines=%v want=%v", e.Selected(), want, e.PopupLines(), lines)
			}
		})
	}
}

func TestArmOpenBackDeleteMatchesRow(t *testing.T) {
	for _, arm := range []string{"Win", "Protect"} {
		t.Run(arm, func(t *testing.T) {
			b := goalMap(t, true)
			row := 0
			if arm == "Protect" {
				e, _ := openGoalMap(t, b)
				s := scriptOf(t, e)
				d, err := s.SetGoalTarget(len(s.Goals())-1, "Unit", 0)
				if err != nil {
					t.Fatal(err)
				}
				b, row = d.SourceBytes, 3
			}
			var wantBytes []byte
			var wantMessage string
			for _, opened := range []bool{false, true} {
				e, _ := openGoalMap(t, b)
				_ = e.Key("goals")
				for e.ReportSelected() < row {
					_ = e.Key("down")
				}
				selected, lines := e.Selected(), e.PopupLines()
				if opened {
					r, ok := e.GoalOpenRect(row)
					trigClick(t, e, r, ok, "arm Open")
					backFromArm(t, e)
					if e.Selected() != selected || !reflect.DeepEqual(e.PopupLines(), lines) {
						t.Errorf("Back changed visible record: selected=%d want=%d lines=%v want=%v", e.Selected(), selected, e.PopupLines(), lines)
					}
				}
				_ = e.Key("clone")
				if e.Message() != "Select a unit, structure or sack first." || e.Selected() != selected || !bytes.Equal(b, e.Document().SourceBytes) {
					t.Fatal("C must refuse the displayed condition before Open and after Back")
				}
				_ = e.Key("delete")
				if !opened {
					wantBytes, wantMessage = append([]byte(nil), e.Document().SourceBytes...), e.Message()
					if arm == "Win" && (!bytes.Equal(b, wantBytes) || wantMessage != "node ID 1 is referenced by a trigger; clear its references first") {
						t.Fatal("Win Del must refuse the referenced condition")
					}
					if arm == "Protect" && (bytes.Equal(b, wantBytes) || wantMessage != "") {
						t.Fatal("Protect Del must delete its condition")
					}
				} else if !bytes.Equal(wantBytes, e.Document().SourceBytes) || e.Message() != wantMessage {
					t.Errorf("Del after Back differs: message=%q want=%q bytesEqual=%v", e.Message(), wantMessage, bytes.Equal(wantBytes, e.Document().SourceBytes))
				}
			}
		})
	}
}

func TestArmAddRowFieldKeepsSelectionRefusal(t *testing.T) {
	e, _ := openGoalMap(t, goalMap(t, true))
	_ = e.Key("goals")
	for i := 0; i < 3; i++ {
		_ = e.Key("down")
	}
	_ = e.Key("field")
	if e.Selected() != -1 || e.Message() != "Select a unit, structure or sack first." {
		t.Fatalf("E on add row: selected=%d message=%q", e.Selected(), e.Message())
	}
}
