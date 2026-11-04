package game

import (
	"bytes"
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// The identity choice's value: the kind in the high word, and in the low
// word the Units row of a creature, the Humans row of a person or the NPC
// record of a named character.
const (
	identityCreature = 0
	identityPerson   = 1
	identityNamed    = 2
)

func identityValue(kind, n int) int64 { return int64(kind)<<32 | int64(uint32(n)) }

// unitIdentityEditor is a planted map with one placement on each rung:
// unit 0 a Humans row by type, 1 a creature (Units row 1), 2 a Humans row
// by definition id (row 2), 3 a named character (NPC record 8, row 2), 4 a
// second creature of unit 1's row on another cell.
func unitIdentityEditor(t *testing.T) (*mapInspectionEdits, []byte) {
	t.Helper()
	f, s, _ := nativeIdentityEditor(t)
	f.Table.Units = dbCollection{{}, unitsRow("Wolf", map[int]int32{0x1d: 40, 0x1e: 1}),
		unitsRow("Wolf pack leader", map[int]int32{0x1d: 40, 0x1e: 2}), unitsRow("Bear", map[int]int32{0x1d: 41, 0x1e: 1})}
	f.Units.Classes[40] = worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
	f.Units.Classes[41] = worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
	more, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: []synth.ALMUnit{
		{X: 10 << 8, Y: 8 << 8, ClassID: 40, ClassSubID: 1, UnitID: 8},
		{X: 12 << 8, Y: 8 << 8, ClassID: 3, DefID: 4321, UnitID: 9},
		{X: 12 << 8, Y: 10 << 8, ClassID: 3, ClassSubID: 8, Flags: 1, UnitID: 10},
		{X: 14 << 8, Y: 8 << 8, ClassID: 40, ClassSubID: 1, UnitID: 11},
	}}))
	if err != nil {
		t.Fatal(err)
	}
	for i := range 4 {
		rec, _ := more.UnitRecord(i)
		n, err := s.model.PlaceUnit(rec)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.model.SetUnitField(n, "owner", 1); err != nil {
			t.Fatal(err)
		}
	}
	raw := s.model.Bytes()
	doc, err := s.inspector.inspect(raw, "identity.alm")
	if err != nil {
		t.Fatal(err)
	}
	return doc.Edits.(*mapInspectionEdits), raw
}

func identityField(t *testing.T, s *mapInspectionEdits, unit int) ui.InspectionField {
	t.Helper()
	fields := s.Fields(ui.KindUnit, unit)
	return fields[fieldIndex(t, fields, "identity")]
}

// The popup shows the kind mapload.Resolve answers and offers only the
// fields that kind's rung reads; the raw keys stand read-only beside it.
func TestEditorUnitIdentityShowsTheResolvedKindAndOnlyItsFields(t *testing.T) {
	s, _ := unitIdentityEditor(t)
	m, _ := s.model.Map()
	want := map[int]int64{0: identityValue(identityPerson, 1), 1: identityValue(identityCreature, 1),
		2: identityValue(identityPerson, 2), 3: identityValue(identityNamed, 8)}
	look := map[int]bool{0: true, 2: true}
	for unit, value := range want {
		r := mapload.Resolve(m.Units[unit], s.inspector.table)
		if !r.Found() {
			t.Fatalf("unit %d resolves to nothing", unit)
		}
		f := identityField(t, s, unit)
		if f.Value != value {
			t.Errorf("unit %d (%s arm, row %d) shows identity %#x, want %#x", unit, r.Arm, r.Index, f.Value, value)
		}
		found := false
		for _, c := range f.Choices {
			found = found || c.Value == value
		}
		if !found {
			t.Errorf("unit %d: the identity choice lacks its own entry %#x", unit, value)
		}
		fields := s.Fields(ui.KindUnit, unit)
		keys := false
		for _, g := range fields {
			switch g.Key {
			case "class", "tier", "definition", "npc", "npcflag":
				t.Errorf("unit %d offers the raw key %q", unit, g.Key)
			case "face", "sex":
				if !look[unit] {
					t.Errorf("unit %d (%s arm) offers %q, which its rung does not read", unit, r.Arm, g.Key)
				}
			case "keys":
				keys = g.ReadOnly && len(g.Lines) >= 4
			}
		}
		if !keys {
			t.Errorf("unit %d has no read-only line of its four raw keys", unit)
		}
	}
}

// Choosing an identity writes the keys its rung reads, so mapload.Resolve
// then answers that identity; choosing the current one, or nothing, writes
// the map byte-equal.
func TestEditorUnitIdentityChoiceWritesTheChosenRow(t *testing.T) {
	cases := []struct {
		unit, kind, n int
		arm           mapload.Arm
		index         int
	}{
		{1, identityPerson, 2, mapload.ArmServerID, 2},
		{1, identityNamed, 8, mapload.ArmNPC, 2},
		{0, identityCreature, 3, mapload.ArmUnits, 3},
		{3, identityPerson, 1, mapload.ArmServerID, 1},
		{2, identityNamed, 7, mapload.ArmNPC, 1},
		{3, identityCreature, 2, mapload.ArmUnits, 2},
	}
	for _, c := range cases {
		s, raw := unitIdentityEditor(t)
		if _, err := s.SetField(ui.KindUnit, c.unit, "identity", identityValue(c.kind, c.n)); err != nil {
			t.Fatalf("unit %d to kind %d #%d: %v", c.unit, c.kind, c.n, err)
		}
		m, _ := s.model.Map()
		r := mapload.Resolve(m.AuthoredUnits[c.unit], s.inspector.table)
		if r.Arm != c.arm || r.Index != c.index {
			t.Errorf("unit %d to kind %d #%d resolves %s row %d, want %s row %d", c.unit, c.kind, c.n, r.Arm, r.Index, c.arm, c.index)
		}
		if got := identityField(t, s, c.unit).Value; got != identityValue(c.kind, c.n) {
			t.Errorf("unit %d shows %#x after the choice", c.unit, got)
		}
		if _, err := s.Undo(); err != nil || !bytes.Equal(raw, s.model.Bytes()) {
			t.Errorf("one Undo does not restore unit %d: %v", c.unit, err)
		}
	}
	s, raw := unitIdentityEditor(t)
	for unit := range 5 {
		if _, err := s.SetField(ui.KindUnit, unit, "identity", identityField(t, s, unit).Value); err != nil || s.CanUndo() {
			t.Fatalf("re-choosing unit %d's own identity edited the map: %v", unit, err)
		}
	}
	path, err := s.SaveAs(filepath.Join(t.TempDir(), "same.alm"))
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := s.inspector.Open(path)
	if err != nil || !bytes.Equal(reopened.SourceBytes, raw) {
		t.Fatal("an untouched map did not write byte-equal", err)
	}
}

// The Add window asks one identity step, from the popup's choice, and
// creates a unit of each kind.
func TestEditorAddWindowCreatesEachIdentity(t *testing.T) {
	for _, c := range []struct {
		kind, n int
		arm     mapload.Arm
		index   int
	}{{identityCreature, 2, mapload.ArmUnits, 2}, {identityPerson, 2, mapload.ArmServerID, 2}, {identityNamed, 8, mapload.ArmNPC, 2}} {
		s, _ := unitIdentityEditor(t)
		step := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity")
		popup := identityField(t, s, 1).Choices
		if len(step.Choices) == 0 || len(step.Choices) != len(popup) {
			t.Fatalf("the Add window offers %d identities, the popup %d", len(step.Choices), len(popup))
		}
		r := ui.InspectionNewRecord{Kind: ui.KindUnit, Cell: image.Pt(20, 20), HasCell: true, Values: []ui.InspectionFieldValue{
			{Key: "identity", Value: identityValue(c.kind, c.n)}, {Key: "owner", Value: 1}, {Key: "group", Value: 30}}}
		if _, err := s.AddRecord(r); err != nil {
			t.Fatalf("add kind %d: %v", c.kind, err)
		}
		m, _ := s.model.Map()
		got := mapload.Resolve(m.AuthoredUnits[len(m.AuthoredUnits)-1], s.inspector.table)
		if got.Arm != c.arm || got.Index != c.index {
			t.Errorf("added kind %d #%d resolves %s row %d", c.kind, c.n, got.Arm, got.Index)
		}
	}
}

// The popup says up front whether a dialogue at mission start shows the
// unit, as which speaker, or why not, from UnitSpeaker.
func TestEditorUnitPopupStatesItsSpeaker(t *testing.T) {
	s, _ := unitIdentityEditor(t)
	x := s.inspector
	for unit := range 5 {
		n, miss := s.UnitSpeaker(unit)
		fields := s.Fields(ui.KindUnit, unit)
		f := fields[fieldIndex(t, fields, "speaker")]
		text := f.Meaning
		for _, l := range f.Lines {
			text += "\n" + l.Text
		}
		if miss.Cause == ui.SpeakerShown {
			label, _ := x.speakerLabel(n)
			if !strings.Contains(text, x.inspectionUTF8(label)) {
				t.Errorf("unit %d is shown as speaker %d, the popup says %q", unit, n, text)
			}
			continue
		}
		why := x.inspectionUTF8(x.word(speakerMissWord(miss.Cause)))
		if cut := strings.Index(why, "%"); cut >= 0 {
			why = why[:cut]
		}
		if !strings.Contains(text, why) {
			t.Errorf("unit %d: miss %v, the popup says %q, want %q", unit, miss.Cause, text, why)
		}
	}
}

// speakerMissWord is the word the speaker pick's refusal gives for cause.
func speakerMissWord(cause ui.SpeakerCause) string {
	return map[ui.SpeakerCause]string{ui.SpeakerBorder: "editor.speaker.why.border", ui.SpeakerNoPlacement: "editor.speaker.why.noplacement",
		ui.SpeakerDown: "editor.speaker.why.down", ui.SpeakerNoFit: "editor.speaker.why.nofit",
		ui.SpeakerNotFirst: "editor.speaker.why.notfirst", ui.SpeakerNoPicture: "editor.speaker.why.nopicture", ui.SpeakerNoMission: "editor.speaker.why.nomission"}[cause]
}

// Two units of one kind: the unit parameter's choices tell them apart by
// more than the number, each stores its own unit's number, and the hero
// band is named by role.
func TestEditorUnitChoicesTellUnitsOfOneKindApart(t *testing.T) {
	s, _ := unitIdentityEditor(t)
	m, _ := s.model.Map()
	d, err := s.project(s.model)
	if err != nil {
		t.Fatal(err)
	}
	labels := map[int]string{}
	for _, tg := range d.Script.Targets[mapload.ScriptUnit] {
		if tg.Record < 0 || d.Records[tg.Record].Kind != ui.KindUnit {
			continue
		}
		u := m.Units[d.Records[tg.Record].Index]
		if tg.Value != uint32(u.UnitID) {
			t.Errorf("unit #%d's choice stores %d, want its number %d", d.Records[tg.Record].Index, tg.Value, u.UnitID)
		}
		labels[d.Records[tg.Record].Index] = tg.ChoiceText()
		cell := fmt.Sprintf("%d,%d", u.X>>8, u.Y>>8)
		if !strings.Contains(tg.ChoiceText(), cell) {
			t.Errorf("unit #%d's choice %q does not name its cell %s", d.Records[tg.Record].Index, tg.ChoiceText(), cell)
		}
	}
	strip := func(i int) string {
		return strings.ReplaceAll(labels[i], fmt.Sprint(m.Units[i].UnitID), "")
	}
	if labels[1] == "" || strip(1) == strip(4) {
		t.Errorf("two wolves read alike but for the number: %q, %q", labels[1], labels[4])
	}
	heroes := s.inspector.editorHeroLabels()
	for _, tg := range d.Script.Targets[mapload.ScriptUnit] {
		if tg.Value == 10001 && tg.ChoiceText() != heroes.Label(1, 1) {
			t.Errorf("hero 1 reads %q, want its role label %q", tg.ChoiceText(), heroes.Label(1, 1))
		}
	}
}
