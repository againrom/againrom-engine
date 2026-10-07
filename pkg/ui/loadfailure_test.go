package ui

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

func TestMalformedTacticalContinuationFitsTheLoadFailureLine(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10}, {ID: 2, X: 2, Y: 1, HP: 10, MaxHP: 10}})
	if err != nil {
		t.Fatal(err)
	}
	actions := w.Actions()
	ids := []sim.EntityID{2, 1}
	actions.ActorTraversal = &ids
	actions.Actors[0].ActorState = 0x16
	actions.Actors[0].Retreat = &sim.RetreatContinuation{Known: true, Progress: 1}
	if err := w.RestoreActions(actions, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if err := sim.CheckSaveForm(raw); err != nil {
		t.Fatal("valid tactical form refused", err)
	}
	span := int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
	start := len(raw) - 9 - span
	badFooter, duplicate, progress := bytes.Clone(raw), bytes.Clone(raw), bytes.Clone(raw)
	badFooter[len(badFooter)-1] ^= 1
	copy(duplicate[start+9:start+13], duplicate[start+5:start+9])
	progress[start+17+5] = 5
	name := strings.Repeat("s", refusalNameWidth)
	for _, data := range [][]byte{{raw[0]}, badFooter, duplicate, progress} {
		err := sim.CheckSaveForm(data)
		if err == nil {
			t.Fatal("corrupt tactical form was accepted")
		}
		line := loadFailure(name, err)
		if line != name+": malformed tactical continuation" || len([]rune(line)) > pickerCols || clipRunes(line, pickerCols) != line || strings.Contains(line, "sim:") {
			t.Fatal("tactical LOAD reason leaked a prefix or was clipped", line)
		}
	}
}

// The load window's refusal line (1032 return 1).
//
// THE DEFECT THIS WITNESSES. sim.CheckSaveForm's refusal is what the player
// reads when a save will not open: pkg/game returns it unwrapped, chooseLoad
// puts it on the flow message line, and app.go draws that line through
// clipRunes(msg, pickerCols). The sentence returned before this return was 126
// runes of byte-form vocabulary and did not name the save, so the player read
// half a clause about a format number and the rest was cut with nothing on
// screen marking the cut.
//
// WHAT THIS TEST CANNOT SEE is how long a real save name is. Names are made by
// pkg/game's SaveStore, a tier above this one, so the width below is stated
// here rather than measured. The owner's own corpus of 65 files has a longest
// name of 26 runes.
const refusalNameWidth = 27

// TestEveryRefusedVersionsMessageFitsTheLineTheWindowDraws walks the WHOLE
// population: every one of the 256 version bytes that sim.CheckSaveForm does
// not accept as a bare version byte.
//
// Exactly two version bytes must come back with no error: the current form and
// its predecessor, which need no further check. That assertion is what keeps
// this test from passing on a build that refuses nothing.
func TestEveryRefusedVersionsMessageFitsTheLineTheWindowDraws(t *testing.T) {
	name := strings.Repeat("s", refusalNameWidth)

	accepted, refused := 0, 0
	for i := 0; i <= 255; i++ {
		v := byte(i)
		err := sim.CheckSaveForm([]byte{v})
		if err == nil {
			accepted++
			continue
		}
		refused++

		line := loadFailure(name, err)
		if !strings.HasPrefix(line, name+": ") {
			t.Errorf("version %d: the line does not name the save: %q", v, line)
		}
		if n := len([]rune(line)); n > pickerCols {
			t.Errorf("version %d: the line is %d runes and the window holds %d: %q", v, n, pickerCols, line)
		}
		if got := clipRunes(line, pickerCols); got != line {
			t.Errorf("version %d: the window cuts the refusal to %q", v, got)
		}
		if strings.Contains(line, "sim:") {
			t.Errorf("version %d: the line carries a package prefix: %q", v, line)
		}
	}
	if accepted != 2 {
		t.Errorf("%d version bytes are accepted as a bare version byte, want exactly 2 (the current form and its predecessor)", accepted)
	}
	if refused != 254 {
		t.Errorf("%d version bytes are refused, want 254", refused)
	}
}

// TestARefusalTooLongForTheLineIsMarkedWhereItIsCut keeps the test above from
// being the only thing standing between the player and a silent cut: if a
// future refusal does not fit, the draw still says so.
func TestARefusalTooLongForTheLineIsMarkedWhereItIsCut(t *testing.T) {
	line := loadFailure(strings.Repeat("s", refusalNameWidth), errTooLongForTheLine)
	if len([]rune(line)) <= pickerCols {
		t.Fatal("the fixture fits the line; it cannot show a cut")
	}
	drawn := clipRunes(line, pickerCols)
	if len([]rune(drawn)) != pickerCols {
		t.Fatalf("the drawn line is %d runes, want pickerCols = %d", len([]rune(drawn)), pickerCols)
	}
	if !strings.HasSuffix(drawn, clipMark) {
		t.Errorf("the cut is not marked: %q", drawn)
	}
}

type longError struct{}

func (longError) Error() string { return strings.Repeat("word ", 60) }

var errTooLongForTheLine error = longError{}
