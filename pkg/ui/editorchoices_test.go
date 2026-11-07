package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"
)

func TestChoiceListBrowsesGroupsAndLoadsOnlyTheirPictures(t *testing.T) {
	icon := image.NewRGBA(image.Rect(0, 0, 3, 4))
	var loaded []int64
	l := choiceList{back: "Back", field: InspectionField{
		Choices: []InspectionChoice{{0, "Remove"}, {-1, "Duplicate"}, {1, "Sword"}, {2, "Steel sword"}, {3, "Potion"}},
		Groups:  []InspectionChoiceGroup{{"Swords", "Iron", []int64{1}}, {"Swords", "Steel", []int64{2}}, {"Potions", "None", []int64{3}}},
		Pictures: func(choices []InspectionChoice) map[int64]*image.RGBA {
			out := map[int64]*image.RGBA{}
			for _, c := range choices {
				loaded = append(loaded, c.Value)
				out[c.Value] = icon
			}
			return out
		},
	}}
	rows := l.rows()
	if len(rows) != 4 || rows[0].Text != "Remove" || rows[1].Text != "Duplicate" || rows[2].Text != "Swords >" || len(loaded) != 0 {
		t.Fatalf("root rows=%+v loaded=%v", rows, loaded)
	}
	if _, navigate := l.take(2); !navigate {
		t.Fatal("kind was taken as an item")
	}
	rows = l.rows()
	if len(rows) != 5 || rows[2].Text != "Back" || rows[3].Text != "Iron >" || rows[4].Text != "Steel >" || len(loaded) != 0 {
		t.Fatalf("material rows=%+v loaded=%v", rows, loaded)
	}
	l.take(4)
	rows = l.rows()
	if len(rows) != 4 || rows[3].Text != "Steel sword" || rows[3].Icon != icon || !reflect.DeepEqual(loaded, []int64{2}) {
		t.Fatalf("leaf rows=%+v loaded=%v", rows, loaded)
	}
	if index, navigate := l.take(3); navigate || l.field.Choices[index].Value != 2 {
		t.Fatal("leaf changed the stored item value")
	}
	l.take(2)
	if rows = l.rows(); len(rows) != 5 || rows[3].Text != "Iron >" {
		t.Fatal("Back did not return to materials")
	}
}

func TestChoiceListBuildsFlatListsUnchanged(t *testing.T) {
	f := InspectionField{Choices: []InspectionChoice{{0, "One"}, {1, "Two"}}, Shut: []int64{1}, Icons: map[int64]*image.RGBA{0: image.NewRGBA(image.Rect(0, 0, 1, 1))}}
	l := choiceList{field: f}
	if got, want := l.rows(), choiceRows(f); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows=%+v want=%+v", got, want)
	}
}

type groupedChoiceHost struct {
	InspectionEdits
	InspectionNewRecords
	offer    InspectionNewStep
	identity InspectionNewStep
}

func (h groupedChoiceHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	if key == "item" {
		return h.offer
	}
	if key == "identity" && len(h.identity.Choices) > 0 {
		return h.identity
	}
	return h.InspectionNewRecords.NewRecordStep(r, key)
}

func newGroupedAddEditor(t *testing.T) (*MapEditor, *modesCalls) {
	t.Helper()
	e, calls := newModesEditor(t, "full")
	e.Layout(1920, 1080)
	host := e.doc.Edits
	e.doc.Edits = groupedChoiceHost{InspectionEdits: host, InspectionNewRecords: host.(InspectionNewRecords),
		offer: InspectionNewStep{
			Choices: []InspectionChoice{{1, "Sword"}, {2, "Another sword"}, {3, "Potion"}},
			Groups:  []InspectionChoiceGroup{{"Swords", "Iron", []int64{1, 2}}, {"Potions", "None", []int64{3}}},
		},
		identity: InspectionNewStep{Choices: []InspectionChoice{{100, "Heading"}, {1, "Man"}, {2, "Woman"}}, Shut: []int64{100}, Value: 1},
	}
	return e, calls
}

func TestEditorAddIdentityRows(t *testing.T) {
	e, _ := newGroupedAddEditor(t)
	_ = e.Key("unit")
	r, ok := e.addRunning()
	if !ok || r.steps[r.at].Key != "identity" || r.list == nil {
		t.Fatal("the unit Add window did not open its identity list")
	}
	want := []PickerRow{{Text: "Heading"}, {Text: "Man", Choosable: true}, {Text: "Woman", Choosable: true}}
	if got := r.list.Rows(); !reflect.DeepEqual(got, want) {
		t.Fatalf("identity rows=%+v want=%+v", got, want)
	}
}

func TestEditorAddSackKeyboardSecondItem(t *testing.T) {
	e, calls := newGroupedAddEditor(t)
	_ = e.Key("sack")
	e.prompt.path = "25,20"
	for _, key := range []string{"enter", "enter", "down", "enter", "down", "enter", "up", "enter"} {
		_ = e.Key(key)
	}
	r, ok := e.addRunning()
	if !ok || r.steps[r.at].Key != "item" || len(calls.lines) != 0 {
		t.Fatalf("Back after the first item finished the sack: open=%v writes=%v", ok, calls.lines)
	}
	if !reflect.DeepEqual(r.choices.path, []string{"Swords"}) || !reflect.DeepEqual(r.many[r.at], []int64{1}) {
		t.Fatalf("Back changed the item values or failed to navigate: path=%v items=%v", r.choices.path, r.many[r.at])
	}
	for _, key := range []string{"enter", "down", "enter", "down", "enter", "down", "enter"} {
		_ = e.Key(key)
		if _, ok := e.addRunning(); !ok || len(calls.lines) != 0 {
			t.Fatalf("navigation or item selection finished the sack at %q: writes=%v", key, calls.lines)
		}
	}
	if !reflect.DeepEqual(r.choices.path, []string{"Potions", "None"}) || !reflect.DeepEqual(r.many[r.at], []int64{1, 3}) || e.prompt.path != "" {
		t.Fatalf("second group pick: path=%v items=%v pending=%q", r.choices.path, r.many[r.at], e.prompt.path)
	}
	_ = e.Key("enter")
	if _, ok := e.addRunning(); ok || len(calls.lines) != 1 || !strings.Contains(calls.lines[0], "AddRecord Sack (25,20) true [{item 1 false  false} {item 3 false  false}]") {
		t.Fatalf("empty Enter on the item row did not finish with both items: open=%v writes=%v", ok, calls.lines)
	}
}

func TestEditorAddSackKeyboardBackAfterHighlight(t *testing.T) {
	e, calls := newGroupedAddEditor(t)
	_ = e.Key("sack")
	e.prompt.path = "25,20"
	for _, key := range []string{"enter", "enter", "down", "enter", "down"} {
		_ = e.Key(key)
	}
	if e.prompt.path != "1" {
		t.Fatalf("the item highlight did not set its value: %q", e.prompt.path)
	}
	_ = e.Key("up")
	_ = e.Key("enter")
	r, ok := e.addRunning()
	if !ok || len(calls.lines) != 0 {
		t.Fatalf("Back finished the sack: open=%v writes=%v", ok, calls.lines)
	}
	if e.prompt.path != "" || len(r.many[r.at]) != 0 || !reflect.DeepEqual(r.choices.path, []string{"Swords"}) {
		t.Fatalf("Back added the highlighted item or failed to navigate: pending=%q items=%v path=%v", e.prompt.path, r.many[r.at], r.choices.path)
	}
}

func TestEditorAddSackBrowsesGroupsWithoutWritingNavigationValues(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	e.Layout(1920, 1080)
	reads := 0
	host := e.doc.Edits
	e.doc.Edits = groupedChoiceHost{InspectionEdits: host, InspectionNewRecords: host.(InspectionNewRecords), offer: InspectionNewStep{
		Choices: []InspectionChoice{{1, "Sword"}, {2, "Another sword"}},
		Groups:  []InspectionChoiceGroup{{"Swords", "Iron", []int64{1, 2}}},
		Pictures: func(choices []InspectionChoice) map[int64]*image.RGBA {
			reads++
			if len(choices) != 2 {
				t.Fatal("the group did not ask for its two pictures")
			}
			return map[int64]*image.RGBA{1: image.NewRGBA(image.Rect(0, 0, 2, 3))}
		},
	}}
	_ = e.Key("sack")
	e.prompt.path = "25,20"
	_ = e.Key("enter")
	r, ok := e.addRunning()
	if !ok || r.list.Len() != 1 || r.list.Rows()[0].Text != "Swords >" || reads != 0 {
		t.Fatal("the new sack opened the full item list")
	}
	_ = e.Key("enter")
	_ = e.Key("down")
	_ = e.Key("enter")
	if reads != 1 || r.list.Len() != 3 || len(calls.lines) != 0 {
		t.Fatal("navigation wrote an item or read other groups")
	}
	for range 3 {
		_ = e.ChromeImage()
	}
	if reads != 1 {
		t.Fatal("painting loaded group pictures")
	}
	e.addTake(2)
	_ = e.Key("enter")
	_ = e.Key("enter")
	if len(calls.lines) != 1 || !strings.Contains(calls.lines[0], "AddRecord Sack (25,20) true [{item 2 false  false}]") {
		t.Fatal("the grouped pick changed the sack writer input", calls.lines)
	}
}
