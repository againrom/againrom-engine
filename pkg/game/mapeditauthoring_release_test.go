package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func recordFieldInput(t *testing.T, e *ui.MapEditor, key string, value int64) {
	t.Helper()
	r := e.Document().Records[e.Selected()]
	for i, f := range e.Document().Edits.Fields(r.Kind, r.Index) {
		if f.Key == key {
			if err := enterField(e, i, strconv.FormatInt(value, 10)); err != nil {
				t.Fatalf("%s %d %s: %v", r.Kind, r.Index, key, err)
			}
			return
		}
	}
	t.Fatalf("%s %d has no field %s", r.Kind, r.Index, key)
}

func recordAddInput(t *testing.T, e *ui.MapEditor, key string, cell image.Point, value int64) {
	t.Helper()
	if err := e.Key(key); err != nil {
		t.Fatal(err)
	}
	clickCell(e, cell)
	e.TextInput(strconv.FormatInt(value, 10))
	if err := e.Key("enter"); err != nil {
		t.Fatalf("%s: %v", key, err)
	}
}

func recordRecipeInput(t *testing.T, e *ui.MapEditor, magnitude int64) {
	t.Helper()
	if err := e.Key("recipe"); err != nil {
		t.Fatal(err)
	}
	e.TextInput("1")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if r := e.Document().Records[e.Selected()]; r.Kind != "Enchantment" {
		t.Fatal("the recipe was not selected", r)
	}
	recordFieldInput(t, e, "addeffect", 12)
	recordFieldInput(t, e, "low1", magnitude)
}

func authoredBlockingKind(t *testing.T, x *MapInspector) uint32 {
	t.Helper()
	for kind := uint32(1); kind < uint32(len(x.structures.Classes)); kind++ {
		if x.structures.Classes[kind] == nil || kind == 34 || kind == 35 {
			continue
		}
		m := &alm.Map{Objects: []alm.Object{{Kind: kind, Field0C: 1}}}
		s := mapload.Structures(m, x.table)[0]
		if s.Blocking != 0 && s.Field42 > 0 && s.Width > 0 && s.Height > 0 && s.Width <= 3 && s.Height <= 3 {
			return kind
		}
	}
	t.Fatal("no bounded installed structure has standing health and a blocking footprint")
	return 0
}

func authoredRecordsMod(t *testing.T, arm string) (string, uint16) {
	t.Helper()
	root := newMapMod(t)
	x := installedInspector(t)
	if err := x.SetMod(filepath.Join(root, newMapModID)); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	e.Document().Viewer.SetFlat(true)
	_ = e.Key("drop")
	clickCell(e, image.Pt(16, 12))
	class := creatureClass(t, x)
	for _, cell := range []image.Point{image.Pt(22, 22), image.Pt(20, 22)} {
		recordAddInput(t, e, "unit", cell, int64(class))
		recordFieldInput(t, e, "group", 71)
	}
	recordAddInput(t, e, "unit", image.Pt(18, 22), int64(class))
	recordFieldInput(t, e, "group", 72)
	recordAddInput(t, e, "structure", image.Pt(10, 18), int64(authoredBlockingKind(t, x)))
	_ = e.Key("clone")
	clickCell(e, image.Pt(15, 18))
	m, _ := alm.Open(e.Document().SourceBytes)
	if len(m.Objects) != 2 || m.Objects[0].Field12 == m.Objects[1].Field12 {
		t.Fatal("fresh structure and clone have ambiguous identities", m.Objects)
	}
	e.Select(recordIndex(t, e, "Structure", 0), true)
	recordFieldInput(t, e, "owner", 2)
	if arm == "structure" {
		s := e.Document().Edits.(ui.InspectionScript)
		if _, err := s.SetGoalTarget(goalRowOf(t, s, armWin, "structure destroyed"), "Structure", 0); err != nil {
			t.Fatal(err)
		}
		recordFieldInput(t, e, "identity", 55)
	}
	if arm == "group" {
		s := e.Document().Edits.(ui.InspectionScript)
		if _, err := s.SetGoalTarget(goalRowOf(t, s, armWin, "group wiped out"), "Unit", 0); err != nil {
			t.Fatal(err)
		}
	}
	e.Select(recordIndex(t, e, "Unit", 0), true)
	if err := e.Key("stock"); err != nil {
		t.Fatal(err)
	}
	const code = 0x1106
	recordFieldInput(t, e, "additem", code)
	recordRecipeInput(t, e, 5)
	recordAddInput(t, e, "sack", image.Pt(17, 12), code)
	recordRecipeInput(t, e, 9)
	before := bytes.Clone(e.Document().SourceBytes)
	recordAddInput(t, e, "unit", image.Pt(25, 20), int64(class))
	ringFinding := false
	for _, finding := range e.Document().Edits.(ui.InspectionChecker).Report() {
		if finding.Rule == "unit-terrain" && finding.Kind == "Unit" && finding.Index == 3 && strings.Contains(finding.Text, "the game removes a unit placed there") {
			ringFinding = true
		}
	}
	if !ringFinding {
		t.Fatal("the authored border placement has no explanatory load-removal finding")
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("the ring-control placement was not undone exactly", err)
	}
	wantSaved, err := alm.SaveForm(before)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Key("saveas"); err != nil || e.Dirty() {
		t.Fatalf("save authored records: %v (%s)", err, e.Message())
	}
	target := filepath.Join(root, newMapModID, "mission", "map.alm")
	saved, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(wantSaved, saved) {
		t.Fatalf("save differs from the accepted ALM save form: %v", err)
	}
	e2 := x.Editor()
	if err := e2.Open(target); err != nil || !bytes.Equal(e2.Document().SourceBytes, saved) {
		t.Fatalf("saved ALM did not reopen: %v", err)
	}
	e2.Select(recordIndex(t, e2, "Stock", 0), false)
	recordFieldInput(t, e2, "recipe1", 0)
	plain, _ := alm.Open(e2.Document().SourceBytes)
	plainLoot, _ := plain.Loot()
	if plainLoot.Records[0].Elements[0].TileMarkerIndex != 0 || plainLoot.Records[1].Elements[0].TileMarkerIndex != 2 || len(plain.Enchantments) != 2 {
		t.Fatal("clearing a recipe after reopen changed other links or recipe indices", plainLoot, plain.Enchantments)
	}
	cleared := bytes.Clone(e2.Document().SourceBytes)
	e2.Select(recordIndex(t, e2, "Enchantment", 1), false)
	if err := e2.Key("delete"); err == nil || !bytes.Equal(cleared, e2.Document().SourceBytes) {
		t.Fatal("referenced recipe removal was accepted or changed bytes", err)
	}
	e2.Select(recordIndex(t, e2, "Enchantment", 0), false)
	if err := e2.Key("delete"); err != nil {
		t.Fatal("unused recipe removal failed", err)
	}
	removed, _ := alm.Open(e2.Document().SourceBytes)
	removedLoot, _ := removed.Loot()
	if len(removed.Enchantments) != 1 || removedLoot.Records[1].Elements[0].TileMarkerIndex != 1 || removed.Enchantments[0].Elements[0].Low != 9 {
		t.Fatal("unused recipe removal changed a surviving binding", removedLoot, removed.Enchantments)
	}
	if err := e2.Key("undo"); err != nil || !bytes.Equal(cleared, e2.Document().SourceBytes) {
		t.Fatal("recipe cleanup did not restore its links atomically", err)
	}
	e2.Select(recordIndex(t, e2, "Stock", 0), false)
	recordFieldInput(t, e2, "recipe1", 2)
	shared, _ := alm.Open(e2.Document().SourceBytes)
	sharedLoot, _ := shared.Loot()
	if sharedLoot.Records[0].Elements[0].TileMarkerIndex != 2 || sharedLoot.Records[1].Elements[0].TileMarkerIndex != 2 || len(shared.Enchantments) != 2 {
		t.Fatal("changing a recipe after reopen changed other links or recipe indices", sharedLoot, shared.Enchantments)
	}
	_ = e2.Key("undo")
	_ = e2.Key("undo")
	if !bytes.Equal(e2.Document().SourceBytes, saved) {
		t.Fatal("reopened recipe edits did not undo exactly")
	}
	m, _ = alm.Open(saved)
	for _, u := range m.Units {
		t.Logf("authored unit id%d group%d owner%d class%d at%d,%d", u.UnitID, u.GroupID, u.Owner, u.ClassID, u.X>>8, u.Y>>8)
	}
	l, _ := m.Loot()
	if len(l.Records) != 2 || l.Records[0].Owner != uint32(m.Units[0].UnitID) ||
		l.Records[0].Elements[0].TileMarkerIndex != 1 || l.Records[1].Elements[0].TileMarkerIndex != 2 ||
		m.Units[0].GroupID != 71 || m.Units[1].GroupID != 71 || m.Objects[0].Field0E != 2 {
		t.Fatal("reopened authoring state differs", m.Units, m.Objects, l)
	}
	return root, code
}

func authoredRecordItems(t *testing.T, f *FrontEnd, code uint16) string {
	t.Helper()
	unit := byMapUnitID(f.live.world.Entities(), 1)
	if len(unit) != 1 || unit[0].Group != 71 || unit[0].HP <= 0 {
		t.Fatal("authored actor or group differs", unit)
	}
	second := byMapUnitID(f.live.world.Entities(), 2)
	if len(second) != 1 || second[0].Group != 71 || second[0].HP <= 0 {
		for _, ent := range f.live.world.Entities() {
			t.Logf("live id%d map%d group%d owner%d hp%d decay%d", ent.ID, ent.MapUnitID, ent.Group, ent.Owner, ent.HP, ent.Decay)
		}
		t.Fatal("second authored group member differs", second)
	}
	other := byMapUnitID(f.live.world.Entities(), 3)
	if len(other) != 1 || other[0].Group != 72 || other[0].HP <= 0 {
		t.Fatal("unrelated authored group member differs", other)
	}
	items, _ := f.live.world.CarriedItems(unit[0].ID)
	var stock []sim.ItemInstance
	for _, item := range items {
		if item.Code == code {
			stock = append(stock, item)
		}
	}
	ground := groundAt(f.live.world.Sacks(), 17, 12)
	if len(stock) != 1 || !reflect.DeepEqual(stock[0].Effects, []sim.ItemEffect{{Kind: 12, Operand: 5}}) ||
		ground == nil || len(ground.ItemInstances) != 1 || ground.ItemInstances[0].Code != code ||
		!reflect.DeepEqual(ground.ItemInstances[0].Effects, []sim.ItemEffect{{Kind: 12, Operand: 9}}) {
		t.Fatalf("stock/ground recipes differ: stock %+v ground %+v", stock, ground)
	}
	return fmt.Sprintf("%04x stock%v ground%v", code, stock[0].Effects, ground.ItemInstances[0].Effects)
}

func TestReleaseMapEditorFreshRecordAuthoringSAVAndGoals(t *testing.T) {
	t.Run("group", func(t *testing.T) {
		root, code := authoredRecordsMod(t, "group")
		f, app := startArmsMission(t, root)
		want := authoredRecordItems(t, f, code)
		if out := runScript(t, f, app, 40); out != sim.OutcomeUndecided {
			t.Fatal("fresh map is already complete", out)
		}
		store, name, _ := menuSAVE(t, f, app, OriginalStore{})
		cold := guestFront(t, root, newMapModID)
		a := cold.App("authored records cold SAV")
		a.Layout(1024, 768)
		save, list, load := cold.SaveSeams(store, OriginalStore{}, nil)
		a.SetSaveSeams(save, list, load)
		groundAppLoad(t, a, list, localOriginalSaveToken(name))
		if got := authoredRecordItems(t, cold, code); got != want {
			t.Fatal("cold LOAD changed authored items", got, want)
		}
		killScriptUnit(t, cold, 3)
		if out := runScript(t, cold, a, 80); out != sim.OutcomeUndecided {
			t.Fatal("an unrelated group death won the authored group target", out)
		}
		killScriptUnit(t, cold, 1)
		if out := runScript(t, cold, a, 80); out != sim.OutcomeUndecided {
			for _, ent := range cold.live.world.Entities() {
				t.Logf("id%d map%d group%d owner%d hp%d decay%d", ent.ID, ent.MapUnitID, ent.Group, ent.Owner, ent.HP, ent.Decay)
			}
			t.Fatal("one member death won before the last member died", out)
		}
		killScriptUnit(t, cold, 2)
		if out := runScript(t, cold, a, 120); out != sim.OutcomeWon {
			t.Fatal("the authored group's last member did not win", out)
		}
		t.Log(want, "survived ALM reopen, gameplay materialization and cold SAV LOAD")
	})
	t.Run("structure", func(t *testing.T) {
		root, _ := authoredRecordsMod(t, "structure")
		f, app := startArmsMission(t, root)
		ss := f.live.world.Structures()
		if len(ss) != 2 || ss[0].Field42 == 0 || ss[1].Field42 == 0 || ss[0].Blocking == 0 {
			t.Fatal("authored structures have no standing obstacle", ss)
		}
		for _, cell := range blockingCells(ss[0], image.Pt(int(ss[0].Col), int(ss[0].Row))) {
			if walksTo(f, f.live.mission.ids[0], cell) {
				t.Fatal("the hero entered an authored blocking cell", cell)
			}
		}
		felled := func(id sim.StructureID) {
			if err := f.live.world.ImportOriginalStructureHealth([]sim.OriginalStructureHealth{{ID: id, Health: 0, MaxHealth: 1}}); err != nil {
				t.Fatal(err)
			}
		}
		felled(ss[1].ID)
		if out := runScript(t, f, app, 80); out != sim.OutcomeUndecided {
			t.Fatal("destroying the clone won the first structure's target", out)
		}
		felled(ss[0].ID)
		if out := runScript(t, f, app, 120); out != sim.OutcomeWon {
			t.Fatal("destroying the renamed original target did not win", out)
		}
	})
}

func TestReleaseMapEditorUnitStockCloneHasIndependentInventory(t *testing.T) {
	root, code := authoredRecordsMod(t, "group")
	x := installedInspector(t)
	if err := x.SetMod(filepath.Join(root, newMapModID)); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(filepath.Join(root, newMapModID, "mission", "map.alm")); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	e.Document().Viewer.SetFlat(true)
	e.Select(recordIndex(t, e, "Unit", 0), true)
	before := bytes.Clone(e.Document().SourceBytes)
	_ = e.Key("clone")
	clickCell(e, image.Pt(22, 18))
	after := bytes.Clone(e.Document().SourceBytes)
	if bytes.Equal(before, after) {
		t.Fatal("the unit/stock clone made no edit")
	}
	_ = e.Key("undo")
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("unit and stock clone did not undo together")
	}
	_ = e.Key("redo")
	if !bytes.Equal(after, e.Document().SourceBytes) {
		t.Fatal("unit and stock clone did not redo together")
	}
	e.Select(recordIndex(t, e, "Unit", 3), false)
	recordFieldInput(t, e, "group", 73)
	if err := e.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	f, app := startArmsMission(t, root)
	source := byMapUnitID(f.live.world.Entities(), 1)
	clone := byMapUnitID(f.live.world.Entities(), 4)
	if len(source) != 1 || len(clone) != 1 || source[0].ID == clone[0].ID || clone[0].Group != 73 {
		t.Fatal("clone inventory has no independent actor", source, clone)
	}
	want, _ := f.live.world.CarriedItems(source[0].ID)
	got, _ := f.live.world.CarriedItems(clone[0].ID)
	if len(got) != 1 || got[0].Code != code || !reflect.DeepEqual(want, got) {
		t.Fatal("clone did not materialize the copied stock", want, got)
	}
	f.live.pending = append(f.live.pending, sim.DropCarried(clone[0].ID, 0, sim.CellPoint{X: 21, Y: 18}))
	f.live.tick()
	check := func(f *FrontEnd) {
		source := byMapUnitID(f.live.world.Entities(), 1)
		clone := byMapUnitID(f.live.world.Entities(), 4)
		if len(source) != 1 || len(clone) != 1 {
			t.Fatal("a stock owner disappeared")
		}
		original, _ := f.live.world.CarriedItems(source[0].ID)
		copied, _ := f.live.world.CarriedItems(clone[0].ID)
		if !reflect.DeepEqual(original, want) || len(copied) != 0 {
			t.Fatal("dropping cloned stock altered the original or retained the clone", original, copied)
		}
		found := 0
		for _, sack := range f.live.world.Sacks() {
			for _, item := range sack.ItemInstances {
				if item.Code == code && reflect.DeepEqual(item.Effects, []sim.ItemEffect{{Kind: 12, Operand: 5}}) {
					found++
				}
			}
		}
		if found != 1 {
			t.Fatal("cloned stock did not reach its independent ground destination", found)
		}
	}
	check(f)
	store, name, _ := menuSAVE(t, f, app, OriginalStore{})
	cold := guestFront(t, root, newMapModID)
	a := cold.App("cloned inventory cold SAV")
	a.Layout(1024, 768)
	save, list, load := cold.SaveSeams(store, OriginalStore{}, nil)
	a.SetSaveSeams(save, list, load)
	groundAppLoad(t, a, list, localOriginalSaveToken(name))
	check(cold)
}

func TestReleaseMapEditorNamedRecordChoicesUseExistingPrompts(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	e.Document().Viewer.SetFlat(true)
	s := e.Document().Edits.(*mapInspectionEdits)
	chosen := map[string]int64{}
	for _, c := range []struct {
		key, kind, field string
		cell             image.Point
	}{{"structure", "Structure", "kind", image.Pt(10, 18)}, {"unit", "Unit", "class", image.Pt(20, 20)}, {"sack", "Sack", "item", image.Pt(17, 12)}} {
		rows := s.RecordChoices(c.kind, -1, c.field)
		if len(rows) == 0 || rows[0].Label == "" || strings.Contains(rows[0].Label, "raw ") {
			t.Fatalf("%s has no visible installed name: %+v", c.kind, rows)
		}
		chosen[c.kind] = rows[0].Value
		_ = e.Key(c.key)
		clickCell(e, c.cell)
		if err := e.Key("down"); err != nil {
			t.Fatal(err)
		}
		if err := e.Key("enter"); err != nil {
			t.Fatalf("%s named choice refused: %v", c.kind, err)
		}
	}
	m, _ := alm.Open(e.Document().SourceBytes)
	loot, _ := m.Loot()
	if len(m.Objects) != 1 || int64(m.Objects[0].Kind) != chosen["Structure"] || len(m.Units) != 1 || int64(m.Units[0].ClassID) != chosen["Unit"] ||
		len(loot.Records) != 1 || int64(loot.Records[0].Elements[0].ItemCode()) != chosen["Sack"] {
		t.Fatal("the keyboard choices did not reach the existing record constructors", m.Objects, m.Units, loot)
	}
	_ = e.Key("recipe")
	_ = e.Key("down")
	if err := e.Key("enter"); err != nil {
		t.Fatal("the named item number did not create a recipe", err)
	}
	r := e.Document().Records[e.Selected()]
	fields := e.Document().Edits.Fields(r.Kind, r.Index)
	spellField := -1
	var spellChoices []ui.InspectionChoice
	for i, f := range fields {
		if f.Key == "spell" {
			spellField, spellChoices = i, f.Choices
		}
	}
	if spellField < 0 || len(spellChoices) < 2 || spellChoices[1].Label == "" {
		t.Fatal("recipe spell field has no installed names", fields)
	}
	_ = e.Key("field")
	for i := 0; i < spellField; i++ {
		_ = e.Key("tab")
	}
	_ = e.Key("down")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	m, _ = alm.Open(e.Document().SourceBytes)
	if uint16(m.Enchantments[0].SpellRaw) != uint16(spellChoices[1].Value) {
		t.Fatal("the named spell did not reach the recipe field", m.Enchantments)
	}
}
