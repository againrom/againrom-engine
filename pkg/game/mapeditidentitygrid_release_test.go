package game

import (
	"bytes"
	"image"
	"slices"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func TestReleaseMapEditorIdentityGridUsesInstalledRowsAndTiers(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open(newMapSource(48, 48)); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	s := e.Document().Edits.(*mapInspectionEdits)
	step := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity")
	var values, offered []int64
	blocks := map[string][]int64{}
	for _, group := range step.Groups {
		if group.Block == "" || group.Head == "" || len(group.Values) == 0 {
			t.Fatal("installed grid has an empty group", group)
		}
		values = append(values, group.Values...)
		blocks[group.Block] = append(blocks[group.Block], group.Values...)
	}
	for _, choice := range step.Choices {
		if slices.Contains(step.Shut, choice.Value) {
			continue
		}
		offered = append(offered, choice.Value)
		id, _ := s.identityOf(choice.Value)
		if id.Kind != mapload.UnitCreature {
			continue
		}
		u, ok := mapload.Place(alm.Unit{}, id, x.table, s.mission())
		if !ok {
			t.Fatal("installed identity cannot be placed", id)
		}
		c := x.units.Classes[int32(u.ClassID)]
		frames := c.TierFrames(entityTiers(&alm.Map{Units: []alm.Unit{u}}, x.table)[0])
		if len(frames) == 0 {
			continue
		}
		n, _ := terrain.SelectUnitFrame(c.Anim, len(frames), false, 0, 0, 0)
		want, got := frames[n].RGBA(), s.NewRecordPreview(ui.KindUnit, "identity", choice.Value)
		if want == nil || got == nil || want.Bounds() != got.Bounds() || !bytes.Equal(want.Pix, got.Pix) {
			t.Errorf("installed row %d preview differs from its placed tier", id.Row)
		}
	}
	slices.Sort(values)
	slices.Sort(offered)
	if !slices.Equal(values, offered) || len(blocks) != 2 {
		t.Fatal("installed grid loses identities or its two blocks")
	}
	for _, block := range []string{"creatures", "people"} {
		members := blocks[x.inspectionUTF8(x.word("editor.unit.groups."+block))]
		if len(members) == 0 {
			t.Fatal("installed block is empty", block)
		}
		for _, value := range []int64{members[0], members[len(members)-1]} {
			e := x.Editor()
			if err := e.Open(newMapSource(48, 48)); err != nil {
				t.Fatal(err)
			}
			e.Layout(1280, 800)
			e.Document().Viewer.SetFlat(true)
			c := &editorClicker{t: t, now: time.Unix(50, 0)}
			r, ok := e.BarRect("units")
			c.click(e, r, ok, "Units")
			index := -1
			for _, choice := range step.Choices {
				if slices.Contains(step.Shut, choice.Value) {
					continue
				}
				index++
				if choice.Value == value {
					break
				}
			}
			for tries := 0; tries < len(values)+16; tries++ {
				if r, ok = e.PlaceGridCellRect(index); ok {
					break
				}
				popup, open := e.PopupRect()
				if !open {
					t.Fatal("Units popup closed while scrolling")
				}
				p := image.Pt((popup.Min.X+popup.Max.X)/2, (popup.Min.Y+popup.Max.Y)/2)
				e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, WheelY: -1}, c.now)
				c.now = c.now.Add(time.Second)
			}
			c.click(e, r, ok, "block edge identity")
			cell := image.Pt(24, 24)
			e.ShowCell(cell)
			p := e.CellPoint(cell)
			c.click(e, image.Rect(p.X-1, p.Y-1, p.X+2, p.Y+2), true, "placement cell")
			placed := e.Document().Edits.(*mapInspectionEdits)
			m, err := placed.model.Map()
			if err != nil || len(m.AuthoredUnits) != 1 {
				t.Fatal("grid press did not place exactly one unit", err)
			}
			id, _, found := mapload.IdentifyAt(m.AuthoredUnits[0], x.table, placed.mission())
			if !found || identityChoiceValue(id) != value {
				t.Errorf("block edge %d placed identity %+v", value, id)
			}
		}
	}
}
