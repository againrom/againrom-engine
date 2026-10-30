package ui

import (
	"encoding/json"
	"image"
	"reflect"
	"strconv"
	"testing"
)

// A unit added by Enter, by add_record and by a click on the same cell is
// one record. Class 1 leaves the optional tier step out; class 2 asks it,
// and every other step is left at the window's default. The click comes at
// the class step and at the step after it.
func TestEditorAddPlacementBuildsTheRecordEnterAndAddRecordBuild(t *testing.T) {
	cell := image.Pt(25, 20)
	for _, c := range []struct {
		class  int
		before int
	}{{1, 0}, {1, 1}, {2, 0}, {2, 1}} {
		name := "class " + strconv.Itoa(c.class) + " click after " + strconv.Itoa(c.before) + " steps"
		start := func() (*MapEditor, *walkHost) {
			e, host := newWalkEditor(t)
			e.openAdd("Unit")
			e.prompt.path = strconv.Itoa(c.class)
			return e, host
		}

		enter, enterHost := start()
		for n := 0; n < 8; n++ {
			r, ok := enter.addRunning()
			if !ok {
				break
			}
			if r.steps[r.at].Control == addCell {
				enter.prompt.path = "25,20"
			}
			_ = enter.Key("enter")
		}
		if len(enterHost.added) != 1 {
			t.Fatalf("%s: Enter added %v", name, enterHost.added)
		}

		click, clickHost := start()
		if c.before > 0 {
			_ = click.Key("enter")
		}
		click.Fit()
		modesClick(click, click.CellPoint(cell))
		if len(clickHost.added) != 1 {
			t.Fatalf("%s: the click added %v", name, clickHost.added)
		}

		x, y := new(int), new(int)
		*x, *y = cell.X, cell.Y
		class, _ := json.Marshal(c.class)
		command, err := func() (InspectionNewRecord, error) {
			e, _ := newWalkEditor(t)
			return e.newRecordOf("Unit", x, y, map[string]json.RawMessage{"class": class}, nil)
		}()
		if err != nil {
			t.Fatalf("%s: add_record refused: %v", name, err)
		}
		if !reflect.DeepEqual(clickHost.added[0], enterHost.added[0]) || !reflect.DeepEqual(command, enterHost.added[0]) {
			t.Errorf("%s: Enter %+v, add_record %+v, click %+v", name, enterHost.added[0], command, clickHost.added[0])
		}
	}
}

// The placement preview draws the record the click then adds.
func TestEditorAddPlacementPreviewIsTheClickedRecord(t *testing.T) {
	e, host := newWalkEditor(t)
	e.openAdd("Unit")
	e.prompt.path = "1"
	cell := image.Pt(25, 20)
	want := e.addPlacement(cell)
	e.Fit()
	modesClick(e, e.CellPoint(cell))
	if len(host.added) != 1 || !reflect.DeepEqual(host.added[0], want) {
		t.Fatalf("the click added %v, the preview held %+v", host.added, want)
	}
	for _, v := range want.Values {
		if v.Key == "tier" {
			t.Fatalf("the preview holds the tier of a skipped step: %+v", want)
		}
	}
}
