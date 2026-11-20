//go:build release

package game

import (
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// secondFacts is what a saved second-game map holds of its placements, read
// from the record payloads and not through the editor.
type secondFacts struct {
	structs                 []uint16
	structX, structY        []uint32
	units                   [][]byte
	sackOwner, sackX, sackY []uint32
	sackItems               []int
	sackBind                [][]uint32
	meta                    [3]uint32
	markers                 uint32
	width, height           int
	hasStructs, hasUnits    bool
	hasSacks                bool
}

func (f secondFacts) tag(i int) uint32 {
	return binary.LittleEndian.Uint32(f.units[i][alm.SecondUnitTag:]) & 0xffff
}
func (f secondFacts) link(i int) uint32 {
	return binary.LittleEndian.Uint32(f.units[i][alm.SecondUnitSack:])
}
func (f secondFacts) ux(i int) uint32 { return binary.LittleEndian.Uint32(f.units[i][alm.UnitX:]) }
func (f secondFacts) uy(i int) uint32 { return binary.LittleEndian.Uint32(f.units[i][alm.UnitY:]) }

func readSecondFacts(t *testing.T, raw []byte) secondFacts {
	t.Helper()
	doc, err := alm.OpenDocument(raw, alm.SecondFormat)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.OpenROM2(raw)
	if err != nil {
		t.Fatal(err)
	}
	f := secondFacts{width: int(m.Width), height: int(m.Height)}
	le := binary.LittleEndian
	for _, o := range m.Objects {
		f.structs = append(f.structs, o.Field12)
		f.structX, f.structY = append(f.structX, o.X), append(f.structY, o.Y)
	}
	for i, id := range doc.RecordTypeIDs() {
		p := doc.RecordPayload(i)
		switch id {
		case alm.TypeMeta:
			f.meta = [3]uint32{le.Uint32(p[alm.MetaObjectCount:]), le.Uint32(p[alm.MetaUnitCount:]), le.Uint32(p[alm.MetaLootCount:])}
		case alm.TypeObjects:
			f.hasStructs = true
		case alm.TypeUnits:
			f.hasUnits = true
			for q := 0; q+alm.SecondUnitSize <= len(p); q += alm.SecondUnitSize {
				f.units = append(f.units, p[q:q+alm.SecondUnitSize])
			}
		case alm.TypeTileMarkers:
			f.markers = le.Uint32(p)
		case alm.TypeLoot:
			f.hasSacks = true
			for q := 0; q+alm.LootHeadWide <= len(p); {
				n := int(le.Uint32(p[q+alm.LootCount:]))
				f.sackOwner = append(f.sackOwner, le.Uint32(p[q+alm.LootOwner:]))
				f.sackX, f.sackY = append(f.sackX, le.Uint32(p[q+alm.LootX:])), append(f.sackY, le.Uint32(p[q+alm.LootX+4:]))
				f.sackItems = append(f.sackItems, n)
				var bind []uint32
				for k := 0; k < n; k++ {
					bind = append(bind, le.Uint32(p[q+alm.LootHeadWide+k*alm.LootElementSize+alm.LootElementMarker:]))
				}
				f.sackBind = append(f.sackBind, bind)
				q += alm.LootHeadWide + n*alm.LootElementSize
			}
		}
	}
	return f
}

func lowestFree(used map[uint32]bool) uint32 {
	for id := uint32(1); ; id++ {
		if !used[id] {
			return id
		}
	}
}

// checkSecondFacts judges the claims a saved map must keep: unique ids,
// counters equal to the record counts, sack links and bindings in range.
func checkSecondFacts(t *testing.T, name string, f secondFacts) {
	t.Helper()
	seen := map[uint32]bool{}
	for _, id := range f.structs {
		if id != 0 && seen[uint32(id)] {
			t.Errorf("%s: id %d shared", name, id)
		}
		seen[uint32(id)] = true
	}
	for i := range f.units {
		if id := f.tag(i); id != 0 && seen[id] {
			t.Errorf("%s: id %d shared", name, id)
		}
		seen[f.tag(i)] = true
		if hi := binary.LittleEndian.Uint32(f.units[i][alm.SecondUnitTag:]) >> 16; hi != f.ux(i)&0xffff {
			t.Errorf("%s: unit %d tag high word %#x, x %#x", name, i, hi, f.ux(i))
		}
	}
	if f.meta != [3]uint32{uint32(len(f.structs)), uint32(len(f.units)), uint32(len(f.sackOwner))} {
		t.Errorf("%s: counters %v, records %d %d %d", name, f.meta, len(f.structs), len(f.units), len(f.sackOwner))
	}
	linked := map[int]uint32{}
	for i := range f.units {
		if v := int(f.link(i)); v != 0 {
			if v > len(f.sackOwner) || f.sackOwner[v-1] != f.tag(i) {
				t.Errorf("%s: unit %d links sack %d", name, i, v)
			} else {
				linked[v-1] = f.tag(i)
			}
		}
	}
	for j, o := range f.sackOwner {
		if o != 0 && linked[j] != o {
			t.Errorf("%s: sack %d of unit %d has no link", name, j, o)
		}
		for _, b := range f.sackBind[j] {
			if b > f.markers {
				t.Errorf("%s: sack %d binds element %d of %d", name, j, b, f.markers)
			}
		}
	}
}

func TestReleaseSecondMapPlacesMovesAndDeletes(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	var lackStructs, lackUnits, lackSacks, unknownItems, items int
	pick := -1
	for i, raw := range raws {
		f := readSecondFacts(t, raw)
		checkSecondFacts(t, sources[i], f)
		lackStructs, lackUnits, lackSacks = lackStructs+b2i(!f.hasStructs), lackUnits+b2i(!f.hasUnits), lackSacks+b2i(!f.hasSacks)
		if pick >= 0 || f.width < 24 || f.height < 24 || len(f.structs) == 0 {
			continue
		}
		for u := range f.units {
			if f.link(u) != 0 {
				pick = i
				break
			}
		}
	}
	if pick < 0 {
		t.Fatal("no installed map holds a structure and a unit with a sack")
	}
	source, raw := sources[pick], raws[pick]
	was := readSecondFacts(t, raw)
	m, err := alm.OpenROM2(raw)
	if err != nil {
		t.Fatal(err)
	}
	d, err := x.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(secondEdits)

	// The identity choice of a unit the map holds, the structure kind of one it
	// holds and the first catalog item.
	choice := map[mapload.Identity]int64{}
	for _, c := range realChoices(host.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity").Choices) {
		if id, ok := x.identityOf(c.Value); ok {
			choice[id] = c.Value
		}
	}
	var identity int64 = -1
	for _, u := range m.Units {
		if id, _, ok := mapload.IdentifyAt(u, x.table, x.identityMission); ok {
			if v, ok := choice[id]; ok {
				identity = v
				break
			}
		}
	}
	if identity < 0 {
		t.Fatal("no unit of the map is a catalog identity")
	}
	itemChoices := host.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindSack}, "item").Choices
	items = len(itemChoices)
	for _, c := range itemChoices {
		if err := x.checkItem(c.Value); err != nil {
			unknownItems++
		}
	}
	item := int64(-1)
	for _, c := range itemChoices {
		if x.checkItem(c.Value) == nil {
			item = c.Value
			break
		}
	}
	if item < 0 {
		t.Fatal("no catalog item is placeable")
	}
	w, h := was.width, was.height
	unitCell, structCell, sackCell := image.Pt(w/2, h/2), image.Pt(w/2-2, h/2-2), image.Pt(w/2+2, h/2)
	moved := image.Pt(w/2+1, h/2+1)

	used := map[uint32]bool{}
	for _, id := range was.structs {
		used[uint32(id)] = true
	}
	for u := range was.units {
		used[was.tag(u)] = true
	}
	wantUnit := lowestFree(used)
	used[wantUnit] = true
	wantStruct := lowestFree(used)

	add := func(r ui.InspectionNewRecord) int {
		t.Helper()
		if err := host.CheckNewRecord(r, "cell"); err != nil {
			t.Fatalf("%v: %v", r.Kind, err)
		}
		next, err := host.AddRecord(r)
		if err != nil || next == nil {
			t.Fatalf("%v: %v", r.Kind, err)
		}
		return next.AddedRecord.Index
	}
	kind := int64(uint8(m.Objects[0].Kind))
	newUnit := add(ui.InspectionNewRecord{Kind: ui.KindUnit, Cell: unitCell, HasCell: true, Values: []ui.InspectionFieldValue{{Key: "identity", Value: identity}}})
	add(ui.InspectionNewRecord{Kind: ui.KindStructure, Cell: structCell, HasCell: true, Values: []ui.InspectionFieldValue{{Key: "kind", Value: kind}}})
	add(ui.InspectionNewRecord{Kind: ui.KindSack, Cell: sackCell, HasCell: true, Values: []ui.InspectionFieldValue{{Key: "item", Value: item}}})
	if _, err := host.Move(ui.KindUnit, newUnit, moved); err != nil {
		t.Fatal(err)
	}
	// An edge cell is refused for a structure.
	if _, err := host.AddRecord(ui.InspectionNewRecord{Kind: ui.KindStructure, Cell: image.Pt(3, 3), HasCell: true, Values: []ui.InspectionFieldValue{{Key: "kind", Value: kind}}}); err == nil {
		t.Fatal("a structure within 8 cells of the edge was placed")
	}
	// Delete the unit with a sack and the first structure.
	gone := -1
	for u := range was.units {
		if was.link(u) != 0 {
			gone = u
			break
		}
	}
	goneID, goneStruct := was.tag(gone), uint32(was.structs[0])
	if _, err := host.Delete(ui.KindUnit, gone); err != nil {
		t.Fatal(err)
	}
	if _, err := host.Delete(ui.KindStructure, 0); err != nil {
		t.Fatal(err)
	}
	target, err := host.SaveAs(filepath.Join(t.TempDir(), "placed.alm"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	got := readSecondFacts(t, saved)
	checkSecondFacts(t, "saved", got)

	// Ids: the new unit and structure take the lowest free ids; the deleted
	// ones are gone; no other id changed.
	wantIDs := map[uint32]bool{wantUnit: true, wantStruct: true}
	for u := range was.units {
		if u != gone {
			wantIDs[was.tag(u)] = true
		}
	}
	for _, id := range was.structs[1:] {
		wantIDs[uint32(id)] = true
	}
	gotIDs := map[uint32]bool{}
	for _, id := range got.structs {
		gotIDs[uint32(id)] = true
	}
	for u := range got.units {
		gotIDs[got.tag(u)] = true
	}
	if len(gotIDs) != len(wantIDs) || gotIDs[goneID] || gotIDs[goneStruct] && goneStruct != 0 {
		t.Errorf("ids %v, want %v", gotIDs, wantIDs)
	}
	for id := range wantIDs {
		if !gotIDs[id] {
			t.Errorf("id %d missing from the saved map", id)
		}
	}
	// Positions: the moved unit, the new structure and the new sack stand on
	// their cells' centres.
	nu := -1
	for u := range got.units {
		if got.tag(u) == wantUnit {
			nu = u
		}
	}
	if nu < 0 || got.ux(nu) != uint32(moved.X)<<8|0x80 || got.uy(nu) != uint32(moved.Y)<<8|0x80 {
		t.Errorf("moved unit at %d,%d", got.ux(nu), got.uy(nu))
	}
	last := len(got.structs) - 1
	if got.structs[last] != uint16(wantStruct) || got.structX[last] != uint32(structCell.X)<<8|0x80 || got.structY[last] != uint32(structCell.Y)<<8|0x80 {
		t.Errorf("new structure id %d at %d,%d", got.structs[last], got.structX[last], got.structY[last])
	}
	sack := len(got.sackOwner) - 1
	if got.sackOwner[sack] != 0 || got.sackItems[sack] != 1 || got.sackX[sack] != uint32(sackCell.X)<<8|0x80 || got.sackY[sack] != uint32(sackCell.Y)<<8|0x80 {
		t.Errorf("new sack owner %d items %d at %d,%d", got.sackOwner[sack], got.sackItems[sack], got.sackX[sack], got.sackY[sack])
	}
	// The deleted unit took its own sacks; the new sack is the only one added.
	ownSacks := 0
	for _, o := range was.sackOwner {
		if o == goneID {
			ownSacks++
		}
	}
	// A new unit copies its template's own sack.
	if want := len(was.sackOwner) - ownSacks + 1 + b2i(nu >= 0 && got.link(nu) != 0); len(got.sackOwner) != want || ownSacks == 0 {
		t.Errorf("sacks %d, want %d (the unit owned %d)", len(got.sackOwner), want, ownSacks)
	}
	// Untouched units keep their bytes but for the sack link.
	k := 0
	for u := range was.units {
		if u == gone {
			continue
		}
		a, b := slices.Clone(was.units[u]), slices.Clone(got.units[k])
		clear(a[alm.SecondUnitSack : alm.SecondUnitSack+4])
		clear(b[alm.SecondUnitSack : alm.SecondUnitSack+4])
		if !slices.Equal(a, b) {
			t.Errorf("unit %d changed beyond its sack link", u)
		}
		k++
	}
	// The saved map loads through the second-game loader and shows the change.
	reloaded, err := x.Open(target)
	if err != nil {
		t.Fatalf("the saved map does not reload: %v", err)
	}
	count := map[ui.RecordKind]int{}
	for _, r := range reloaded.Records {
		count[r.Kind]++
	}
	if count[ui.KindUnit] != len(got.units) || count[ui.KindStructure] != len(got.structs) {
		t.Errorf("reloaded units %d structures %d, saved %d %d", count[ui.KindUnit], count[ui.KindStructure], len(got.units), len(got.structs))
	}
	t.Logf("PASS: %s placed unit %d, structure %d and a sack, moved the unit, deleted unit %d (%d sacks) and structure %d; saved and reloaded (%s); installed maps lacking structures %d, units %d, sacks %d of %d; catalog items %d, not placeable %d",
		source, wantUnit, wantStruct, goneID, ownSacks, goneStruct, installLanguage(x), lackStructs, lackUnits, lackSacks, len(raws), items, unknownItems)
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// The first game's placement keeps its words and its save.
func TestReleaseFirstMapPlacementSavesAsBefore(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	d, err := x.Open("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(ui.InspectionNewRecords)
	was, err := alm.Open(d.SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	var pick int64 = -1
	for _, c := range realChoices(host.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity").Choices) {
		pick = c.Value
		break
	}
	cell := mapload.Playable(was).Min
	if _, err := host.AddRecord(ui.InspectionNewRecord{Kind: ui.KindUnit, Cell: cell, HasCell: true,
		Values: []ui.InspectionFieldValue{{Key: "identity", Value: pick}}}); err != nil {
		t.Fatal(err)
	}
	target, err := d.Edits.(interface {
		SaveAs(string) (string, error)
	}).SaveAs(filepath.Join(t.TempDir(), "placed.alm"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	got, err := alm.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	n := len(got.Units) - 1
	if len(got.Units) != len(was.Units)+1 || got.Units[n].X != uint32(cell.X)<<8|newUnitFraction || got.Units[n].Y != uint32(cell.Y)<<8|newUnitFraction {
		t.Fatalf("units %d -> %d; new unit at %d,%d", len(was.Units), len(got.Units), got.Units[n].X, got.Units[n].Y)
	}
	t.Logf("PASS: first-game placement saved and reloaded: %d -> %d units (%s)", len(was.Units), len(got.Units), installLanguage(x))
}
