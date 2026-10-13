package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// armlessMap is a 32x32 map with four units (ids 10..13, groups 20..23; unit
// 13 belongs to roster slot 2), one structure with id 7 and a script that holds
// only a drop cell: no win arm and no lose arm.
func armlessMap(t *testing.T) []byte {
	t.Helper()
	actions := []scriptNode{{label: "drop", opcode: 0x10002, id: 1, values: []uint32{5, 6}, tags: []uint32{5, 6}}}
	o := synth.ALMOptions{Width: 32, Height: 32, Type7Payload: goalScript(actions, nil, nil)}
	for i := 0; i < 4; i++ {
		o.Units = append(o.Units, synth.ALMUnit{X: uint32(12+3*i)*256 + 128, Y: 14*256 + 128, ClassID: 40, ClassSubID: 2})
	}
	o.Objects = []synth.ALMObject{{X: 20*256 + 128, Y: 20*256 + 128, Kind: 1, Field12: 7}}
	return built(t, o, nil, func(b []byte, payload func(uint32) int) {
		binary.LittleEndian.PutUint32(b[payload(6)+3*70+0x14:], 2)
	}).Bytes()
}

func goalIndex(t *testing.T, goals []ui.InspectionGoal, arm, text string) int {
	t.Helper()
	for i, g := range goals {
		if g.Arm == arm && g.Index == -1 && strings.Contains(g.Text, text) {
			return i
		}
	}
	t.Fatalf("no %s row that adds %q in %+v", arm, text, goals)
	return -1
}

func TestAMapWithoutArmsOffersToAddAWinAndALoseArm(t *testing.T) {
	e, _ := openGoalMap(t, armlessMap(t))
	goals := scriptOf(t, e).Goals()
	var rows []string
	for _, g := range goals {
		rows = append(rows, g.Arm+"/"+g.Pick+"/"+g.Text)
	}
	want := []string{
		"Win/Unit/add: the death of a unit wins",
		"Win/Unit/add: a group wiped out wins",
		"Win/Structure/add: a structure destroyed wins",
		"Lose/Unit/add: the death of a party unit loses",
		"Protect/Unit/add a check that loses the mission when a unit dies",
	}
	if strings.Join(rows, "\n") != strings.Join(want, "\n") {
		t.Fatalf("rows:\n%s", strings.Join(rows, "\n"))
	}
	// A map that has both arms offers no add row for either.
	e2, _ := openGoalMap(t, goalMap(t, true))
	for _, g := range scriptOf(t, e2).Goals() {
		if g.Index == -1 && g.Arm != armProtect {
			t.Fatalf("a map with both arms offers %+v", g)
		}
	}
}

func TestAddingAnArmWritesItsRecordsAndUndoRemovesThem(t *testing.T) {
	for _, c := range []struct {
		name, arm, text, pick string
		index                 int
		opcode                uint32
		tag, value            uint32
	}{
		{"unit win", armWin, "death of a unit", "Unit", 1, 5, 4, 11},
		{"group win", armWin, "group wiped out", "Unit", 2, 1, 2, 22},
		{"structure win", armWin, "structure destroyed", "Structure", 0, 21, 9, 7},
		{"party unit loss", armLose, "party unit", "Unit", 0, 5, 4, 10},
	} {
		t.Run(c.name, func(t *testing.T) {
			b := armlessMap(t)
			e, _ := openGoalMap(t, b)
			s := scriptOf(t, e)
			row := goalIndex(t, s.Goals(), c.arm, c.text)
			next, err := s.SetGoalTarget(row, c.pick, c.index)
			if err != nil || next == nil {
				t.Fatalf("add: %v", err)
			}
			m, err := alm.Open(next.SourceBytes)
			if err != nil {
				t.Fatal(err)
			}
			sc, _ := m.Script()
			if len(sc.Actions) != 2 || len(sc.Conditions) != 2 || len(sc.Triggers) != 1 {
				t.Fatalf("script %d/%d/%d", len(sc.Actions), len(sc.Conditions), len(sc.Triggers))
			}
			check := sc.Conditions[0]
			if check.Opcode != c.opcode || check.Type[0] != c.tag || check.Value[0] != c.value {
				t.Fatalf("check %+v, want opcode %d type %d value %d", check, c.opcode, c.tag, c.value)
			}
			// The panel now lists the new arm as a row with its target, and
			// no longer offers the arm kind.
			var listed bool
			for _, g := range s.Goals() {
				listed = listed || (g.Arm == c.arm && g.Index == 0 && g.Kind == "Condition")
				if g.Arm == c.arm && g.Index == -1 {
					t.Errorf("the add row stayed: %+v", g)
				}
			}
			if !listed {
				t.Fatalf("the new arm is not listed: %+v", s.Goals())
			}
			_ = e.Key("undo")
			if !bytes.Equal(e.Document().SourceBytes, b) || e.Dirty() || e.Document().Edits.CanUndo() {
				t.Fatal("one undo did not restore the exact bytes")
			}
			_ = e.Key("redo")
			if !bytes.Equal(e.Document().SourceBytes, next.SourceBytes) {
				t.Fatal("redo did not restore the arm")
			}
		})
	}
}

func TestAddingAnArmRefusesWhatItCannotName(t *testing.T) {
	b := armlessMap(t)
	e, _ := openGoalMap(t, b)
	s := scriptOf(t, e)
	goals := s.Goals()
	for name, call := range map[string]func() error{
		"a structure for a unit win": func() error {
			_, err := s.SetGoalTarget(goalIndex(t, goals, armWin, "death of a unit"), "Structure", 0)
			return err
		},
		"a unit for a structure win": func() error {
			_, err := s.SetGoalTarget(goalIndex(t, goals, armWin, "structure destroyed"), "Unit", 0)
			return err
		},
		"a unit of the enemy side a loss": func() error {
			_, err := s.SetGoalTarget(goalIndex(t, goals, armLose, "party unit"), "Unit", 3)
			return err
		},
	} {
		if err := call(); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if !bytes.Equal(e.Document().SourceBytes, b) || e.Dirty() {
		t.Fatal("a refused arm changed the bytes")
	}
}

func TestAnArmIsRefusedWhereAVariableTakesItsRegister(t *testing.T) {
	actions := []scriptNode{{label: "set", opcode: 3, id: 1, values: []uint32{1, 5}, tags: []uint32{1, 1}}}
	o := synth.ALMOptions{Width: 32, Height: 32, Type7Payload: goalScript(actions, nil, nil),
		Units: []synth.ALMUnit{{X: 12*256 + 128, Y: 14*256 + 128, ClassID: 40}, {X: 15*256 + 128, Y: 14*256 + 128, ClassID: 40}}}
	e, _ := openGoalMap(t, built(t, o, nil, nil).Bytes())
	s := scriptOf(t, e)
	if _, err := s.SetGoalTarget(goalIndex(t, s.Goals(), armWin, "death of a unit"), "Unit", 1); err == nil || !strings.Contains(err.Error(), "register") {
		t.Fatalf("variable 1 against the arm's registers 0 and 1: %v", err)
	}
}

func TestWindowKeysAddAWinArmAndAnInstalledArmStaysUnchanged(t *testing.T) {
	b := armlessMap(t)
	e, _ := openGoalMap(t, b)
	_ = e.Key("goals")
	rows, open := e.ReportOpen()
	if !open || len(rows) != 5 {
		t.Fatalf("the script panel: open %v, %d rows", open, len(rows))
	}
	_ = e.Key("enter")
	clickCell(e, image.Pt(15, 14)) // unit 2, id 11
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := m.Script()
	if len(sc.Triggers) != 1 || sc.Conditions[0].Opcode != 5 || sc.Conditions[0].Value[0] != 11 {
		t.Fatalf("after the click: %d triggers, %+v", len(sc.Triggers), sc.Conditions)
	}
	if _, open := e.ReportOpen(); !open {
		t.Fatal("the script panel did not stay open")
	}
	_ = e.Key("escape")
	_ = e.Key("undo")
	if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, b) {
		t.Fatal("undo did not restore the bytes")
	}
}
