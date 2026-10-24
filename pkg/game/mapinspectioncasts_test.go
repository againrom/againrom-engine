package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func cellSpellFixture(t *testing.T) *ui.MapEditor {
	t.Helper()
	_, e := openRecords(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	for _, k := range []uint16{0, 8, 4, 12} {
		rec := make([]byte, 38)
		binary.LittleEndian.PutUint32(rec, 19)
		binary.LittleEndian.PutUint32(rec[4:], 3)
		binary.LittleEndian.PutUint32(rec[8:], 4)
		binary.LittleEndian.PutUint16(rec[12:], k)
		binary.LittleEndian.PutUint32(rec[14:], 7)
		if k == 8 {
			binary.LittleEndian.PutUint32(rec[14:], 0)
		}
		binary.LittleEndian.PutUint32(rec[18:], 13|1<<16)
		binary.LittleEndian.PutUint32(rec[22:], 2)
		for i, v := range []uint16{5, 6, 0xdead, 3, 4, 0xbeef} {
			binary.LittleEndian.PutUint16(rec[26+2*i:], v)
		}
		if _, err := s.model.Place(mapedit.KindEnchantment, rec); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "spells.alm")
	if err := os.WriteFile(path, s.model.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	e = s.inspector.Editor()
	e.Layout(1280, 800)
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	return e
}

func addCellSpell(t *testing.T, e *ui.MapEditor, template int, cell image.Point, link string) {
	t.Helper()
	if err := e.Key("casts"); err != nil {
		t.Fatal(err)
	}
	addSteps(t, e, map[string]string{"template": strconv.Itoa(template), "cell": strconv.Itoa(cell.X) + "," + strconv.Itoa(cell.Y), "spell": "13", "power": "20", "link": link})
}

func TestEditorCellSpellsKeyPointerBytesAndClear(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	original := s.model.Bytes()
	badX := int64(1) << 32
	if strconv.IntSize == 64 {
		_, err := s.AddRecord(ui.InspectionNewRecord{Kind: "Enchantment", HasCell: true, Cell: image.Pt(int(badX)+8, 8), Values: []ui.InspectionFieldValue{{Key: "template", Value: 0}}})
		if err == nil || !bytes.Equal(original, s.model.Bytes()) {
			t.Fatal("cell narrowed before validation")
		}
	}
	unknown, _ := s.model.Record(mapedit.KindEnchantment, 3)
	if !strings.Contains(e.Document().Records[recordIndex(t, e, "Enchantment", 3)].Label, "12") {
		t.Fatal("unknown kind absent from list")
	}
	for _, template := range []int{0, 1, 2} {
		before := s.model.Bytes()
		count := s.model.Count(mapedit.KindEnchantment)
		seed, _ := s.model.Record(mapedit.KindEnchantment, template)
		cell := image.Pt(8, 8)
		if template != 0 {
			cell = image.Pt(3, 4)
		}
		link := "7"
		if template == 1 {
			link = "0"
		}
		addCellSpell(t, e, template, cell, link)
		index, added := count, 1
		if template == 2 {
			index, added = template, 0
		}
		if s.model.Count(mapedit.KindEnchantment) != count+added || e.Selected() != recordIndex(t, e, "Enchantment", index) {
			t.Fatal("Add count or popup target", template, s.model.Count(mapedit.KindEnchantment), e.Selected())
		}
		rec, _ := s.model.Record(mapedit.KindEnchantment, index)
		want := bytes.Clone(seed)
		binary.LittleEndian.PutUint32(want[4:], uint32(cell.X))
		binary.LittleEndian.PutUint32(want[8:], uint32(cell.Y))
		binary.LittleEndian.PutUint32(want[18:], 13|20<<16)
		if !bytes.Equal(rec, want) {
			t.Fatalf("added kind %d: %x want %x", template, rec, want)
		}
		if err := enterField(e, 0, "7"); err != nil {
			t.Fatal(err)
		}
		if err := enterField(e, 1, "100"); err != nil {
			t.Fatal(err)
		}
		rec, _ = s.model.Record(mapedit.KindEnchantment, index)
		binary.LittleEndian.PutUint32(want[18:], 7|100<<16)
		if !bytes.Equal(rec, want) {
			t.Fatalf("edited kind %d: %x want %x", template, rec, want)
		}
		if template == 2 {
			if err := enterField(e, 0, "0"); err != nil {
				t.Fatal(err)
			}
		} else {
			r, ok := e.PopupButtonRect(1)
			if ok {
				messageClick(t, e, r, ok)
			}
			if err := e.Key("delete"); err != nil {
				t.Fatal(err)
			}
		}
		if s.model.Count(mapedit.KindEnchantment) != count+added-1 {
			t.Fatal("delete did not remove entry")
		}
		for !bytes.Equal(s.model.Bytes(), before) && s.model.CanUndo() {
			if err := e.Key("undo"); err != nil {
				t.Fatal(err)
			}
		}
		if !bytes.Equal(s.model.Bytes(), before) {
			t.Fatal("route undo")
		}
	}
	e.Select(recordIndex(t, e, "Enchantment", 3), false)
	if err := e.Key("delete"); err == nil {
		t.Fatal("unknown kind deleted")
	}
	got, _ := s.model.Record(mapedit.KindEnchantment, 3)
	if !bytes.Equal(got, unknown) || !bytes.Equal(s.model.Bytes(), original) {
		t.Fatal("unknown bytes changed")
	}
}

func TestEditorCellSpellsAddReusesCastingStructure(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	template, _ := s.model.Record(mapedit.KindEnchantment, 2)
	binary.LittleEndian.PutUint32(template, 99)
	binary.LittleEndian.PutUint32(template[4:], 6)
	binary.LittleEndian.PutUint16(template[26:], 88)
	index, err := s.model.Place(mapedit.KindEnchantment, template)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.model.SetField(mapedit.KindEnchantment, 2, "spell", 7); err != nil {
		t.Fatal(err)
	}
	before := s.model.Bytes()
	target, _ := s.model.Record(mapedit.KindEnchantment, 2)
	at := bytes.Index(before, target)
	if at < 0 || bytes.Count(before, target) != 1 {
		t.Fatal("ambiguous target bytes")
	}
	want := bytes.Clone(before)
	binary.LittleEndian.PutUint32(want[at+18:], 13|20<<16)
	addCellSpell(t, e, index, image.Pt(3, 4), "7")
	if s.model.Count(mapedit.KindEnchantment) != 5 || !bytes.Equal(s.model.Bytes(), want) {
		t.Fatal("existing casting structure was not updated in place")
	}
	if e.Selected() != recordIndex(t, e, "Enchantment", 2) {
		t.Fatal("popup did not open on the existing casting structure")
	}
	if _, ok := e.FieldRect(0); !ok {
		t.Fatal("casting structure popup is closed")
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("one Add undo did not restore all bytes", err)
	}
}

func TestEditorCellSpellsNoneDeletesAndRefusesAdd(t *testing.T) {
	for _, tc := range []struct {
		name     string
		template int
		zeroX    bool
	}{{"trap", 0, false}, {"unit", 1, false}, {"structure", 2, false}, {"zero-x-trap", 0, true}} {
		t.Run(tc.name, func(t *testing.T) {
			template := tc.template
			e := cellSpellFixture(t)
			s := e.Document().Edits.(*mapInspectionEdits)
			if tc.zeroX {
				if err := s.model.SetField(mapedit.KindEnchantment, template, "triggerx", 0); err != nil {
					t.Fatal(err)
				}
			}
			loot := s.model.GroundSack(8*256, 8*256, 257)
			binary.LittleEndian.PutUint32(loot[26:], uint32(template+1))
			if _, err := s.model.Place(mapedit.KindLoot, loot); err != nil {
				t.Fatal(err)
			}
			before := s.model.Bytes()
			e.Select(recordIndex(t, e, "Enchantment", template), false)
			r, ok := e.FieldRect(0)
			messageClick(t, e, r, ok)
			_ = e.Key("selectall")
			e.TextInput("0")
			r, ok = e.PopupButtonRect(0)
			messageClick(t, e, r, ok)
			if s.model.Count(mapedit.KindEnchantment) != 3 {
				t.Fatal("None kept a cell spell record")
			}
			if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), before) {
				t.Fatal("None undo did not restore all bytes", err)
			}
			add := ui.InspectionNewRecord{Kind: "Enchantment", HasCell: true, Cell: image.Pt(3, 4), Values: []ui.InspectionFieldValue{
				{Key: "template", Value: int64(template)}, {Key: "link", Value: 7}, {Key: "spell", Value: 0}, {Key: "power", Value: 20}}}
			if template == 1 {
				add.Values[1].Value = 0
			}
			if _, err := s.AddRecord(add); err == nil || !strings.Contains(err.Error(), "spell") || !bytes.Equal(s.model.Bytes(), before) {
				t.Fatal("Add with None was not refused with a reason", err)
			}
			if err := e.Key("casts"); err != nil {
				t.Fatal(err)
			}
			addStepsN(t, e, map[string]string{"template": strconv.Itoa(template), "link": strconv.FormatInt(add.Values[1].Value, 10), "cell": "3,4"}, 3)
			_ = e.Key("clear")
			e.TextInput("0")
			_ = e.Key("enter")
			if key, note, ok := e.AddStep(); !ok || key != "spell" || note == "" || !bytes.Equal(s.model.Bytes(), before) {
				t.Fatal("Add popup did not explain the None refusal", key, note)
			}
		})
	}
}

func TestEditorCellSpellsPointerApplyKeepsOtherBytes(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	e.Select(recordIndex(t, e, "Enchantment", 0), false)
	before, _ := s.model.Record(mapedit.KindEnchantment, 0)
	r, ok := e.FieldRect(1)
	messageClick(t, e, r, ok)
	_ = e.Key("selectall")
	e.TextInput("99")
	r, ok = e.PopupButtonRect(0)
	messageClick(t, e, r, ok)
	want := bytes.Clone(before)
	binary.LittleEndian.PutUint16(want[20:], 99)
	got, _ := s.model.Record(mapedit.KindEnchantment, 0)
	if !bytes.Equal(got, want) {
		t.Fatalf("pointer Apply: %x want %x", got, want)
	}
}
