package game

import (
	"bytes"
	"image"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestEditorPersonWindowNativeDraftSaveAndColdLoad(t *testing.T) {
	for _, size := range []image.Point{{1280, 800}, {800, 600}, {640, 480}} {
		t.Run(size.String(), func(t *testing.T) {
			f, s, raw := nativeInventoryEditor(t)
			doc, err := s.project(s.model)
			if err != nil {
				t.Fatal(err)
			}
			e := ui.NewMapEditor(nil, nil, func(string) (*ui.InspectionDocument, error) { return doc, nil })
			e.Layout(size.X, size.Y)
			if err := e.Open("person.alm"); err != nil {
				t.Fatal(err)
			}
			e.Select(recordOf(doc, "Unit", 0), false)
			c := &editorClicker{t: t}
			var field func(string) ui.InspectionField
			field = func(key string) ui.InspectionField {
				t.Helper()
				fields := e.ObjectFields()
				i := fieldIndex(t, fields, key)
				f := fields[i]
				if f.Parent != "" {
					field(f.Parent)
					_ = e.Key("escape")
				}
				if tab, ok := e.ObjectSectionRect(f.Section); ok {
					c.click(e, tab, true, "section")
				}
				e.ScrollDetails(-1000)
				r, ok := e.FieldRect(i)
				for n := 0; !ok && n < 60; n++ {
					e.ScrollDetails(1)
					r, ok = e.FieldRect(i)
				}
				c.click(e, r, ok, key)
				return f
			}
			choose := func(key string, value int64) {
				t.Helper()
				f := field(key)
				if len(f.Choices) == 0 {
					for range 16 {
						_ = e.Key("backspace")
					}
					e.TextInput(strconv.FormatInt(value, 10))
					e.ObjectFields()
					return
				}
				i := slices.IndexFunc(f.Choices, func(c ui.InspectionChoice) bool { return c.Value == value })
				r, ok := e.DropdownRowRect(i)
				if !ok && i >= 0 {
					e.TextInput(f.Choices[i].Label)
					_ = e.Key("enter")
					e.ObjectFields()
					return
				}
				c.click(e, r, ok, key+" choice")
				e.ObjectFields()
			}
			typeValue := func(key string, value int64) {
				t.Helper()
				field(key)
				for range 16 {
					_ = e.Key("backspace")
				}
				e.TextInput(strconv.FormatInt(value, 10))
				e.ObjectFields()
			}
			choose("face", 7)
			choose("wear1", 0x102)
			r, ok := e.PopupButtonRect(1)
			c.click(e, r, ok, "Cancel")
			if !bytes.Equal(raw, s.model.Bytes()) || s.CanUndo() {
				t.Fatal("Cancel wrote the native draft")
			}
			e.Select(recordOf(doc, "Unit", 0), false)
			choose("face", 7)
			choose("sex", 0)
			choose("definition", 4321)
			choose("wear1", 0x102)
			choose("wearmaterial1", 1)
			choose("weartier1", 1)
			choose("wearrecipe1", -1)
			choose("wearrecipe1.addeffect", 17)
			typeValue("wearrecipe1.low1", 5)
			choose("carryadd", 0x102)
			choose("carryrecipe2", -1)
			choose("carryrecipe2.addeffect", 17)
			typeValue("carryrecipe2.low1", 9)
			for _, v := range []struct {
				key   string
				value int64
			}{{"hp", 50}, {"mana", 7}, {"body", 33}, {"defence", 44}, {"skill.pike", 7}, {"protection.astral", 55}, {"group", 4}} {
				typeValue(v.key, v.value)
			}
			if !bytes.Equal(raw, s.model.Bytes()) || s.CanUndo() {
				t.Fatal("draft wrote the map")
			}
			r, ok = e.PopupButtonRect(0)
			c.click(e, r, ok, "Apply")
			after := s.model.Bytes()
			if bytes.Equal(raw, after) {
				t.Fatal("Apply made no edit", e.PopupLines())
			}
			m, _ := s.model.Map()
			sheets, err := unitSheets(m, s.inspector.table)
			if err != nil || m.Units[0].ClassSubID&127 != 7 || m.Units[0].Flags&5 != 0 || m.Units[0].DefID != 4321 || m.Units[0].GroupID != 4 ||
				sheets[0].Worn[0].Code != 0x1122 || len(sheets[0].Worn[0].Effects) != 1 || sheets[0].Worn[0].Effects[0].Operand != 5 ||
				!slices.ContainsFunc(sheets[0].Carried, func(it sim.ItemInstance) bool {
					return it.Code == 0x102 && len(it.Effects) == 1 && it.Effects[0].Operand == 9
				}) {
				t.Fatal("whole draft lost an item or identity", err)
			}
			v := sheets[0].Values
			if v.HP != 50 || v.Body != 33 {
				t.Fatal("whole draft lost a stat override", v)
			}
			for _, value := range []ui.InspectionFieldValue{{Key: "hp", Value: 50}, {Key: "mana", Value: 7}, {Key: "body", Value: 33}, {Key: "defence", Value: 44}, {Key: "skill.pike", Value: 7}, {Key: "protection.astral", Value: 55}} {
				stored, err := s.model.FieldValue(mapedit.KindUnit, 0, value.Key)
				if err != nil || stored != value.Value {
					t.Fatal("whole draft lost a stored override", value, stored, err)
				}
			}
			if _, err := s.Undo(); err != nil || !bytes.Equal(raw, s.model.Bytes()) || s.CanUndo() {
				t.Fatal("whole draft Undo was not exact", err)
			}
			if _, err := s.Redo(); err != nil || !bytes.Equal(after, s.model.Bytes()) {
				t.Fatal("whole draft Redo", err)
			}
			path, err := s.SaveAs(filepath.Join(t.TempDir(), "person.alm"))
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := s.inspector.Open(path)
			expectedSave, saveErr := s.model.SaveBytes()
			if err != nil || saveErr != nil || !bytes.Equal(expectedSave, reopened.SourceBytes) {
				t.Fatal("whole draft reopen", err)
			}
			nativeEditorMissionSAV(t, f, s, nativeInventoryEditor)
		})
	}
}
