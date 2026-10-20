package game

import (
	"bytes"
	"fmt"
	"image"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

func TestEditorUnitDefinitionMeaningFollowsResolutionArm(t *testing.T) {
	u := unitTemplateView{s: &mapInspectionEdits{inspector: &MapInspector{}}, m: &alm.Map{Units: make([]alm.Unit, 1)}}
	for _, c := range []struct {
		unit alm.Unit
		want string
	}{
		{alm.Unit{ClassID: 9}, "Resolve from the unit class"},
		{alm.Unit{ClassID: 9, DefID: 0xcdcdcdcd}, "Resolve from the unit class"},
		{alm.Unit{ClassID: 9, DefID: 7}, "Human definition ID: 7"},
		{alm.Unit{ClassID: 64, DefID: 7}, "Ignored; resolved from the unit class"},
		{alm.Unit{ClassID: 9, ClassSubID: 4, Flags: 1, DefID: 7}, "NPC slot: 4; Human definition ID is ignored"},
	} {
		u.m.Units[0] = c.unit
		if got := u.meaning("definition", int64(c.unit.DefID)); got != c.want {
			t.Fatalf("%+v: %q want %q", c.unit, got, c.want)
		}
	}
}

func TestEditorObjectMoveAndFieldsAreOneAtomicEdit(t *testing.T) {
	raw, e := openAuthoring(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	cell := image.Pt(5, 7)
	if _, err := s.SetObject("Unit", 0, []ui.InspectionFieldValue{{Key: "hp", Value: 17}}, &cell); err != nil {
		t.Fatal(err)
	}
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	if m.Units[0].X>>8 != 5 || m.Units[0].Y>>8 != 7 || m.Units[0].CurrentHP != 17 || s.model.Mark() != 1 {
		t.Fatal("Apply did not group cell and fields")
	}
	if !s.model.Undo() || !bytes.Equal(raw, s.model.Bytes()) {
		t.Fatal("one Undo did not restore exact source")
	}
	cell = image.Pt(100, 100)
	if _, err := s.SetObject("Unit", 0, []ui.InspectionFieldValue{{Key: "hp", Value: 99}}, &cell); err == nil {
		t.Fatal("invalid cell accepted")
	}
	if !bytes.Equal(raw, s.model.Bytes()) || s.model.Mark() != 0 {
		t.Fatal("refusal wrote a partial edit")
	}
}

func TestEditorUnitMeaningsAndCardUseGameProjectionWithoutWriting(t *testing.T) {
	raw, e := openAuthoring(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	table := authoringTable()
	rows := table.Units.(dbCollection)
	for i := range rows {
		p := make([]int32, 41)
		for j := range p {
			p[j] = -1
		}
		copy(p, rows[i].params)
		rows[i].params = p
	}
	s.inspector.table = table
	for _, changes := range [][]ui.InspectionFieldValue{nil, {{Key: "hp", Value: 17}, {Key: "mana", Value: 5}, {Key: "body", Value: 31}, {Key: "defence", Value: 14}, {Key: "protection.fire", Value: 77}}} {
		entity, fields, err := s.UnitSettings(0, changes)
		if err != nil {
			t.Fatal(err)
		}
		trial := *s
		trial.model = s.model.Clone()
		trial.batch = trial.model
		if _, err := trial.SetFields("Unit", 0, changes); err != nil {
			t.Fatal(err)
		}
		m, _ := trial.model.Map()
		v := e.Document().Viewer
		mw, err := openMapWorld(m, s.inspector.table, nil, v)
		if err != nil {
			t.Fatal(err)
		}
		want := mw.entityDraws()[0]
		if entity.Char != want.Char || entity.Combat != want.Combat || entity.HP != want.HP || entity.Mana != want.Mana {
			t.Fatalf("editor card diverges from game: %+v / %+v", entity, want)
		}
		for _, f := range fields {
			_, override := overrideTemplate(unitValues{}, f.Key)
			switch {
			case f.Key == "gear":
				if len(f.Choices) != 2 || f.Group == "" {
					t.Errorf("gear choices %+v", f)
				}
			case !override:
				if f.Meaning == "" {
					t.Errorf("%s=%d has no meaning", f.Key, f.Value)
				}
			case !f.Inherit || f.Absent != overrideAbsent(f.Key) || f.Group == "":
				t.Errorf("%s is no inheriting field: %+v", f.Key, f)
			case f.Value == f.Absent && (f.Inherited == "" || f.Meaning != ""):
				t.Errorf("%s inherits without its in-game value: %+v", f.Key, f)
			}
		}
		if !bytes.Equal(s.model.Bytes(), raw) || s.model.Mark() != 0 {
			t.Fatal("preview wrote map or Undo")
		}
	}
	for _, key := range overrideKeys {
		for _, value := range []int64{overrideAbsent(key), 7} {
			_, fields, err := s.UnitSettings(0, []ui.InspectionFieldValue{{Key: key, Value: value}})
			if err != nil {
				t.Fatal(key, value, err)
			}
			f := fields[fieldIndex(t, fields, key)]
			if value == overrideAbsent(key) && (f.Inherited == "" || f.Meaning != "") || value != overrideAbsent(key) && f.Inherited != "" {
				t.Fatalf("%s=%d hint %q meaning %q", key, value, f.Inherited, f.Meaning)
			}
		}
	}
	for _, value := range []int64{0, 3, 4294967295} {
		_, fields, err := s.UnitSettings(0, []ui.InspectionFieldValue{{Key: "group", Value: value}})
		if err != nil {
			t.Fatal(err)
		}
		f := fields[fieldIndex(t, fields, "group")]
		if f.Meaning != fmt.Sprintf("Script group ID: %d", value) {
			t.Fatal(f)
		}
	}
}
