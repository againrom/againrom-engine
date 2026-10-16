package game

import (
	"bytes"
	"os"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
)

func TestProtectIsTheFirstScriptEditAndSurvivesHistoryAndReopen(t *testing.T) {
	b := built(t, synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12), Units: []synth.ALMUnit{{X: 12*256 + 128, Y: 14*256 + 128}}}, nil, nil).Bytes()
	e, path := openGoalMap(t, b)
	s := scriptOf(t, e)
	next, err := s.SetGoalTarget(len(s.Goals())-1, "Unit", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Key("goals"); err != nil {
		t.Fatal(err)
	}
	if len(s.Goals()) != 6 || s.Goals()[0].Arm != armProtect {
		t.Fatalf("goals %+v", s.Goals())
	}
	after := append([]byte(nil), next.SourceBytes...)
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, e.Document().SourceBytes) {
		t.Fatal("Undo lost input")
	}
	if err := e.Key("redo"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, e.Document().SourceBytes) {
		t.Fatal("Redo lost Protect")
	}
	if _, err := e.Document().Edits.SaveAs(path + ".out.alm"); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path + ".out.alm")
	if err != nil || !bytes.Equal(after, saved) {
		t.Fatalf("saved Protect differs: %v", err)
	}
	reopened, _ := openGoalMap(t, saved)
	if err := reopened.Key("goals"); err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(reopened.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil || len(sc.Triggers) != 0 || len(sc.Conditions) != 1 {
		t.Fatalf("script %+v %v", sc, err)
	}
	if sc.Conditions[0].Value[0] != uint32(m.Units[0].UnitID) || !scriptOf(t, reopened).Goals()[0].HasCell {
		t.Fatal("Protect target lost")
	}
}
