package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// motionWindow opens the views fixture's unit at 1920x1080 in lang, in the
// plain or the technical view, floating or docked in the right panel, every
// plain block open.
func motionWindow(t *testing.T, s *mapInspectionEdits, lang string, plain, docked bool) (*ui.MapEditor, *editorClicker) {
	t.Helper()
	c := &editorClicker{t: t}
	doc, err := s.project(s.model)
	if err != nil {
		t.Fatal(err)
	}
	e := ui.NewMapEditor(nil, nil, func(string) (*ui.InspectionDocument, error) { return doc, nil })
	e.SetWords(words.For(words.Language(lang)))
	e.SetHostTextEncoder(s.inspector.inspectionUTF8)
	e.SetCardFont(viewsCardFont())
	e.Layout(1920, 1080)
	view := map[bool]string{true: "plain", false: "technical"}[plain]
	right := `"minimized":true,"width":520`
	if docked {
		right = `"minimized":false,"width":520`
	}
	e.SetPanelProfile([]byte(fmt.Sprintf(`{"left":{"minimized":false,"width":260},"right":{%s,"view":%q}}`, right, view)), nil, nil)
	if err := e.Open("views.alm"); err != nil {
		t.Fatal(err)
	}
	at := slices.IndexFunc(doc.Records, func(r ui.InspectionRecord) bool { return r.Kind == ui.KindUnit })
	e.Select(at, false)
	if e.PlainView() != plain {
		t.Fatalf("plain %v, want %v", e.PlainView(), plain)
	}
	if r, ok := e.PopupRect(); !ok || docked != (r.Min.X >= 1920-520) {
		t.Fatalf("the window stands at %v, docked %v", r, docked)
	}
	if plain {
		openBlocks(t, e, c)
	}
	return e, c
}

// The Add window asks a new unit's facing among its placement fields,
// offering the original's draw and the eight directions; the chosen
// direction is the new record's facing byte, and a unit added without a
// choice keeps the draw.
func TestEditorAddWindowAsksTheUnitsFacing(t *testing.T) {
	for _, c := range []struct {
		values map[string]string
		want   byte
	}{
		{map[string]string{"identity": "1", "facing": "6", "cell": "9,10"}, 6},
		{map[string]string{"identity": "1", "cell": "9,10"}, 0},
	} {
		_, e := openAuthoring(t)
		s := e.Document().Edits.(*mapInspectionEdits)
		step := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "facing")
		if len(step.Choices) != 1+alm.FacingDirections || len(step.ChoicePad) != 9 || step.Value != 0 {
			t.Fatalf("the facing step offers %v at %d", step.Choices, step.Value)
		}
		before := s.model.Count(mapedit.KindUnit)
		addWindow(t, e, "unit", c.values)
		s = e.Document().Edits.(*mapInspectionEdits)
		if s.model.Count(mapedit.KindUnit) != before+1 {
			t.Fatal("the Add window placed no unit")
		}
		rec, _ := s.model.UnitRecord(before)
		if rec[alm.UnitFacing] != c.want {
			t.Fatalf("%v: the new unit's facing byte is %d, want %d", c.values, rec[alm.UnitFacing], c.want)
		}
	}
}

// The person window shows no facing in either view; the host still gives
// the facing field its compass pad, and each of its nine values written
// through the host's field writer gives the old setter's bytes and save.
func TestEditorHiddenFacingWritesTheOldSetterBytes(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, docked := range []bool{false, true} {
			for _, plain := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/docked=%v/plain=%v", lang, docked, plain), func(t *testing.T) {
					s, _ := viewsMap(t)
					base := s.model.Clone()
					s.inspector.archives.Base.Profile.Language = lang
					e, c := motionWindow(t, s, lang, plain, docked)
					f := e.ObjectFields()[fieldIndex(t, e.ObjectFields(), "facing")]
					wantPad := []image.Point{{0, 0}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}
					if !slices.Equal(f.ChoicePad, wantPad) {
						t.Fatal("production facing choices lost compass positions", f.ChoicePad)
					}
					if showField(t, e, c, "facing") {
						t.Fatal("the person window shows the facing")
					}
					for n := range 9 {
						if _, err := s.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: "facing", Value: int64(n), Step: 1}}); err != nil {
							t.Fatal(err)
						}
						want := base.Clone()
						if err := want.SetField(mapedit.KindUnit, 0, "facing", int64(n)); err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(s.model.Bytes(), want.Bytes()) {
							t.Fatalf("pad value %d changed bytes beyond the old setter", n)
						}
						got, err := s.model.SaveBytes()
						if err != nil {
							t.Fatal(err)
						}
						old, err := want.SaveBytes()
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(got, old) {
							t.Fatalf("saved pad value %d differs from the old setter", n)
						}
					}
				})
			}
		}
	}
}

// The editor's map draws a unit with an authored facing in that direction's
// idle frame, read from the octant-legible sheet (idle frame 8 + octant, S 0
// to SE 7 clockwise from south), and a unit without one in the frame it drew
// before.
func TestEditorMapDrawsTheAuthoredFacing(t *testing.T) {
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12), Units: []synth.ALMUnit{
		{X: 3*256 + 128, Y: 4*256 + 77, ClassID: 40, ClassSubID: 2},
	}})
	path := filepath.Join(t.TempDir(), "source.alm")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	class := facingClass()
	x := &MapInspector{tiles: &terrain.Tileset{}, table: authoringTable(), units: &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{40: class}}}
	e := x.Editor()
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	preview := func() *terrain.StaticFrame {
		for _, r := range e.Document().Records {
			if r.Kind == ui.KindUnit && r.Index == 0 {
				return r.Preview
			}
		}
		t.Fatal("no unit record")
		return nil
	}
	if preview() != class.Frames[8] {
		t.Fatal("the unit without a facing left its frame")
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	for direction, octant := range map[int64]int{1: 4, 2: 5, 3: 6, 4: 7, 5: 0, 6: 1, 7: 2, 8: 3} {
		if _, err := s.SetField(ui.KindUnit, 0, "facing", direction); err != nil {
			t.Fatal(err)
		}
		d, err := s.project(s.model)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range d.Records {
			if r.Kind == ui.KindUnit && r.Index == 0 && r.Preview != class.Frames[8+octant] {
				t.Errorf("facing %d draws a frame other than octant %d's", direction, octant)
			}
		}
	}
}

// typeText types text into field key's input over what it holds and
// presses Apply.
func typeText(t *testing.T, e *ui.MapEditor, c *editorClicker, key, text string) {
	t.Helper()
	if !showField(t, e, c, key) {
		t.Fatalf("%s does not show", key)
	}
	r, ok := e.FieldRect(fieldIndex(t, e.ObjectFields(), key))
	c.click(e, r, ok, key)
	for range 16 {
		_ = e.Key("end")
		_ = e.Key("backspace")
	}
	if text != "" {
		e.TextInput(text)
	}
	r, ok = e.PopupButtonRect(0)
	c.click(e, r, ok, "Apply")
}

// A placed unit's speed, sight and facing, in both views of the unit
// window, floating and in the right panel, in English and Russian: an empty
// speed or sight shows the kind's value; each value set writes its byte; a
// value outside the range is refused with its reason and writes nothing; an
// emptied field clears its byte; Undo restores the record. The basic view
// stands speed and sight on the card; neither view shows the facing,
// which the host's field writer still writes.
func TestEditorUnitSpeedSightFacingInBothViews(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, docked := range []bool{false, true} {
			for _, plain := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/docked=%v/plain=%v", lang, docked, plain), func(t *testing.T) {
					s, raw := viewsMap(t)
					s.inspector.archives.Base.Profile.Language = lang
					s.inspector.recordOptions = nil
					sheet := s.unitViewSheets()[0].Values
					e, c := motionWindow(t, s, lang, plain, docked)
					fields := e.ObjectFields()
					for key, kind := range map[string]int32{"speed": sheet.Speed, "sight": sheet.Sight} {
						f := fields[fieldIndex(t, fields, key)]
						if !f.Inherit || f.Value != f.Absent || f.Inherited != strconv.Itoa(int(kind)) {
							t.Fatalf("empty %s: inherit %v value %d, shows %q, want the kind's %d", key, f.Inherit, f.Value, f.Inherited, kind)
						}
					}
					facing := fields[fieldIndex(t, fields, "facing")]
					random := s.inspector.inspectionUTF8(s.inspector.word("editor.unit.facing.random"))
					if len(facing.Choices) != 1+alm.FacingDirections || facing.Value != 0 || facing.Choices[0].Label != random {
						t.Fatalf("facing offers %v at %d", facing.Choices, facing.Value)
					}
					if plain {
						blocks := e.PlainFieldBlocks()
						if blocks["speed"] != "card" || blocks["sight"] != "card" || blocks["facing"] != "" {
							t.Fatalf("plain blocks speed %q sight %q facing %q", blocks["speed"], blocks["sight"], blocks["facing"])
						}
					}
					typeField(t, e, c, "speed", 37)
					typeField(t, e, c, "sight", 7)
					if showField(t, e, c, "facing") {
						t.Fatal("the person window shows the facing")
					}
					if _, err := s.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: "facing", Value: 3, Step: 1}}); err != nil {
						t.Fatal(err)
					}
					rec, _ := s.model.UnitRecord(0)
					if rec[alm.UnitSpeed] != 37 || rec[alm.UnitSight] != 7 || rec[alm.UnitFacing] != 3 {
						t.Fatalf("bytes % x after the three edits", []byte{rec[alm.UnitSpeed], rec[alm.UnitSight], rec[alm.UnitFacing]})
					}
					set := s.model.Bytes()
					for _, bad := range []struct {
						key, text, reason string
					}{{"speed", "256", "0..255"}, {"sight", strconv.Itoa(alm.SightCells + 1), fmt.Sprintf("0..%d", alm.SightCells)}} {
						refused, rc := motionWindow(t, s, lang, plain, docked)
						typeText(t, refused, rc, bad.key, bad.text)
						if !bytes.Equal(s.model.Bytes(), set) {
							t.Fatalf("%s %s was written", bad.key, bad.text)
						}
						if !strings.Contains(strings.Join(refused.PopupLines(), "\n"), bad.reason) {
							t.Fatalf("%s %s refused without its range: %q", bad.key, bad.text, refused.PopupLines())
						}
					}
					typeText(t, e, c, "speed", "")
					if rec, _ := s.model.UnitRecord(0); rec[alm.UnitSpeed] != 0 || rec[alm.UnitSight] != 7 {
						t.Fatalf("an emptied speed left byte %d", rec[alm.UnitSpeed])
					}
					for range 4 {
						if _, err := s.Undo(); err != nil {
							t.Fatal(err)
						}
					}
					if !bytes.Equal(s.model.Bytes(), raw) {
						t.Fatal("Undo did not restore the unit's record")
					}
					if v, _ := s.model.FieldValue(mapedit.KindUnit, 0, "facing"); v != 0 {
						t.Fatalf("Undo left facing %d", v)
					}
				})
			}
		}
	}
}
