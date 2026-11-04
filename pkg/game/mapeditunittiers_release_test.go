package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// tieredPlacement is the first unit, in map then record order, of an
// installed map whose class has more than one tier.
func tieredPlacement(t *testing.T, x *MapInspector) (key string, raw []byte, index int) {
	t.Helper()
	keys, maps := installedMapBytes(t, x)
	for n, b := range maps {
		m, err := alm.Open(b)
		if err != nil {
			continue
		}
		for i, u := range m.Units {
			if len(x.tiersOf(u)) > 1 && mapload.Resolve(u, x.table).Found() {
				return keys[n], b, i
			}
		}
	}
	t.Fatal("no installed map places a creature whose class has several tiers")
	return "", nil, 0
}

// fieldIndex is the position of the field key among a record's popup fields.
func fieldIndex(t *testing.T, fields []ui.InspectionField, key string) int {
	t.Helper()
	for i, f := range fields {
		if f.Key == key {
			return i
		}
	}
	t.Fatalf("no %s field", key)
	return -1
}

// editorClicker presses the centre of a rectangle as one click.
type editorClicker struct {
	t   *testing.T
	now time.Time
}

func (c *editorClicker) click(e *ui.MapEditor, r image.Rectangle, ok bool, what string) {
	c.t.Helper()
	if !ok {
		c.t.Fatalf("%s is not shown", what)
	}
	p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, c.now)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, c.now)
	c.now = c.now.Add(time.Second)
}

// chooseIdentity opens the identity drop-down of the selected unit's popup,
// chooses the choice of value and presses Apply. A row the list does not
// show is chosen by typing its label.
func chooseIdentity(t *testing.T, c *editorClicker, e *ui.MapEditor, fields []ui.InspectionField, value int64) {
	t.Helper()
	n := fieldIndex(t, fields, "identity")
	r, ok := e.FieldRect(n)
	c.click(e, r, ok, "the identity field")
	if got := len(e.DropdownChoices()); got != len(fields[n].Choices) {
		t.Fatalf("the open identity list has %d rows, the field %d choices", got, len(fields[n].Choices))
	}
	k := slices.IndexFunc(fields[n].Choices, func(ch ui.InspectionChoice) bool { return ch.Value == value })
	if k < 0 {
		t.Fatalf("the identity field offers no choice %d", value)
	}
	if r, ok = e.DropdownRowRect(k); ok {
		c.click(e, r, ok, "the identity row")
	} else {
		e.TextInput(fields[n].Choices[k].Label)
		_ = e.Key("enter")
	}
	r, ok = e.PopupButtonRect(0)
	c.click(e, r, ok, "Apply")
}

// creatureIdentity is the identity choice value of tier tr's Units row.
func creatureIdentity(tr unitTier) (mapload.Identity, int64) {
	id := mapload.Identity{Kind: mapload.UnitCreature, Row: tr.Index}
	return id, identityChoiceValue(id)
}

// TestReleaseMapEditorUnitTierPopupWritesTheSetFieldBytes chooses another
// Units row of an installed creature's class through its popup's identity
// choice: the map's bytes are those of SetField of the row's keys on the
// installed bytes, one Undo restores the installed bytes and Redo the edit;
// the unit then draws with the chosen tier's palette; the raw secondary key
// is refused.
func TestReleaseMapEditorUnitTierPopupWritesTheSetFieldBytes(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	key, raw, i := tieredPlacement(t, x)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(key); err != nil {
		t.Fatal(err)
	}
	rec := recordOf(e.Document(), "Unit", i)
	if rec < 0 {
		t.Fatalf("no unit record %d", i)
	}
	e.Select(rec, true)
	fields := objectEdits(t, e).Fields("Unit", i)
	installed, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	u := installed.Units[i]
	tiers := x.tiersOf(u)
	now := mapload.Resolve(u, x.table).Index
	k := slices.IndexFunc(tiers, func(tr unitTier) bool { return tr.Index != now })
	if k < 0 {
		t.Fatalf("the class offers no other tier: %+v", tiers)
	}
	id, value := creatureIdentity(tiers[k])
	placed, ok := mapload.Place(u, id, x.table, 0)
	if !ok {
		t.Fatalf("Units row %d keys no unit", id.Row)
	}
	want := int64(placed.ClassSubID)
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	chooseIdentity(t, c, e, fields, value)
	edited := e.Document().SourceBytes

	witness, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	if placed.ClassID != u.ClassID {
		if err := witness.SetField(mapedit.KindUnit, i, "class", int64(placed.ClassID)); err != nil {
			t.Fatal(err)
		}
	}
	if err := witness.SetField(mapedit.KindUnit, i, "tier", want); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(edited, witness.Bytes()) {
		t.Fatalf("%s unit %d: the popup's bytes differ from SetField tier %d", key, i, want)
	}
	changed := 0
	for n := range raw {
		if raw[n] != edited[n] {
			changed++
		}
	}
	if changed == 0 || changed > 2 {
		t.Fatalf("the tier change moved %d bytes, want 1 or 2", changed)
	}
	m, err := alm.Open(edited)
	if err != nil || int64(m.Units[i].ClassSubID) != want {
		t.Fatalf("unit %d reads secondary key %d after the change, want %d (%v)", i, m.Units[i].ClassSubID, want, err)
	}
	tier := entityTiers(m, x.table)[sim.EntityID(i)]
	frames := x.units.Classes[int32(m.Units[i].ClassID)].TierFrames(tier)
	preview := e.Document().Records[rec].Preview
	if tier == 0 || preview == nil || len(frames) == 0 || preview.Palette != frames[0].Palette {
		t.Fatalf("the unit does not draw with tier %d's palette", tier)
	}
	t.Logf("%s unit %d: tier %d -> %d, %d bytes", key, i, u.ClassSubID, want, changed)

	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatalf("Undo does not restore the installed bytes: %v", err)
	}
	if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, edited) {
		t.Fatalf("Redo does not restore the edit: %v", err)
	}

	before := e.Document().SourceBytes
	if _, err := objectEdits(t, e).SetField("Unit", i, "tier", int64(u.ClassSubID)); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("the raw secondary key was not refused with its reason: %v", err)
	}
	if !bytes.Equal(e.Document().SourceBytes, before) {
		t.Fatal("a refused secondary key wrote")
	}
}

// fourTierClass is the lowest class key whose class offers exactly four tiers.
func fourTierClass(t *testing.T, x *MapInspector) (int16, []unitTier) {
	t.Helper()
	var keys []int
	for id := range x.units.Classes {
		keys = append(keys, int(id))
	}
	sort.Ints(keys)
	for _, id := range keys {
		if id < -32768 || id > 32767 {
			continue
		}
		if tiers := x.tiersOf(alm.Unit{ClassID: int16(id)}); len(tiers) == 4 {
			return int16(id), tiers
		}
	}
	t.Fatal("no installed class has four tiers")
	return 0, nil
}

// TestReleaseMapEditorNewUnitTakesTheChosenTier adds a unit through the Add
// window's identity and cell steps, choosing the third Units row of a
// four-tier class: the record the map gains is the one NewUnitRecord builds
// with that row's class and secondary keys.
func TestReleaseMapEditorNewUnitTakesTheChosenTier(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	class, tiers := fourTierClass(t, x)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(NewMapSource("32x32")); err != nil {
		t.Fatal(err)
	}
	before := e.Document().SourceBytes
	cell := image.Pt(12, 14)
	id, value := creatureIdentity(tiers[2])
	keys, ok := mapload.Place(alm.Unit{}, id, x.table, 0)
	if !ok || keys.ClassID != class {
		t.Fatalf("Units row %d keys class %d (%v), want class %d", id.Row, keys.ClassID, ok, class)
	}
	_ = e.Key("unit")
	if key, _, _ := e.AddStep(); key != "identity" || !bytes.Equal(e.Document().SourceBytes, before) {
		t.Fatalf("the Add window first asks %q; the map changed: %v", key, !bytes.Equal(e.Document().SourceBytes, before))
	}
	want := keys.ClassSubID
	addSteps(t, e, map[string]string{"identity": fmt.Sprint(value), "cell": fmt.Sprintf("%d,%d", cell.X, cell.Y)})
	witness, err := mapedit.New(before)
	if err != nil {
		t.Fatal(err)
	}
	view, err := witness.Map()
	if err != nil {
		t.Fatal(err)
	}
	owner := uint32(min(2, max(1, len(view.Groups))))
	xw, yw := cellWords(cell, newUnitFraction, newUnitFraction)
	rec, err := witness.NewUnitRecord(xw, yw, class, want, owner)
	if err != nil {
		t.Fatal(err)
	}
	index, err := witness.PlaceUnit(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := witness.SetField(mapedit.KindUnit, index, "group", 1); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e.Document().SourceBytes, witness.Bytes()) {
		t.Fatalf("the added unit is not NewUnitRecord(class %d, tier %d)", class, want)
	}
	m, _ := alm.Open(e.Document().SourceBytes)
	if len(m.Units) != 1 || m.Units[0].ClassSubID != want {
		t.Fatalf("the new map holds %+v", m.Units)
	}
}

// gameStats is a spawned actor's popup numbers read off the entity by this
// test, independently of the popup's own reading.
func gameStats(e sim.Entity) []int64 {
	out := []int64{int64(e.HP), int64(e.MaxHP), int64(e.Mana), int64(e.MaxMana),
		int64(e.Mind), int64(e.Spirit), int64(e.Reaction), int64(e.ToHit), int64(e.DamageBase), int64(e.DamageBase + e.DamageSpread),
		int64(e.Defence), int64(e.Absorption), int64(e.Speed), int64(e.ScanRange), int64(e.XPValue),
		int64(e.TypeID), int64(e.GoldChance), int64(e.TreasureMin), int64(e.TreasureMin + e.TreasureMax), int64(e.KnownSpells)}
	body := int64(0)
	if e.NativeBasis.BodyPresent {
		body = int64(e.NativeBasis.Body)
	}
	out = append(out, body)
	for _, v := range e.Protection {
		out = append(out, int64(v))
	}
	for _, v := range e.Resistance {
		out = append(out, int64(v))
	}
	for _, v := range e.Skill {
		out = append(out, int64(v))
	}
	return out
}

// sheetStats is the popup sheet's numbers in gameStats's order.
func sheetStats(v unitValues) []int64 {
	known := int64(0)
	for _, id := range v.Spells {
		known |= 1 << id
	}
	out := []int64{int64(v.HP), int64(v.MaxHP), int64(v.Mana), int64(v.MaxMana),
		int64(v.Mind), int64(v.Spirit), int64(v.Reaction), int64(v.ToHit), int64(v.DamageMin), int64(v.DamageMax),
		int64(v.Defence), int64(v.Absorption), int64(v.Speed), int64(v.Sight), int64(v.XP),
		int64(v.TypeID), int64(v.GoldChance), int64(v.GoldMin), int64(v.GoldMax), known, int64(v.Body)}
	for _, p := range v.Protection {
		out = append(out, int64(p))
	}
	for _, r := range v.Resistance {
		out = append(out, int64(r))
	}
	for _, k := range v.Skill {
		out = append(out, int64(k))
	}
	return out
}

func sheetItemCodes(items []sim.ItemInstance) []uint16 {
	var out []uint16
	for _, it := range items {
		if !it.Empty() {
			out = append(out, it.Code)
		}
	}
	return out
}

// sheetMismatch names what differs between the popup's sheet and the actor
// the game spawned with its body's items, or "" when nothing does.
func sheetMismatch(s unitSheet, e sim.Entity, worn [sim.EquipSlots]sim.ItemInstance, carried []sim.ItemInstance) string {
	if a, b := sheetStats(s.Values), gameStats(e); !slices.Equal(a, b) {
		return fmt.Sprintf("values %v, game %v", a, b)
	}
	if a, b := sheetItemCodes(s.Worn[:]), sheetItemCodes(worn[:]); !slices.Equal(a, b) {
		return fmt.Sprintf("worn %v, game %v", a, b)
	}
	if a, b := sheetItemCodes(s.Carried), sheetItemCodes(carried); !slices.Equal(a, b) {
		return fmt.Sprintf("carried %v, game %v", a, b)
	}
	if s.Suppressed != e.SuppressCorpseLoot {
		return fmt.Sprintf("suppressed %v, game %v", s.Suppressed, e.SuppressCorpseLoot)
	}
	return ""
}

// TestReleaseMapEditorUnitValuesEqualTheGame opens every installed map in the
// editor and compares each unit's popup values with the actor the game spawns
// from the same map at Normal difficulty, its worn and carried items and its
// corpse rule included; the popup's lines carry the compared numbers. A
// perturbed popup value is reported, which is this comparison's loss control.
func TestReleaseMapEditorUnitValuesEqualTheGame(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	keys, maps := installedMapBytes(t, x)
	var compared, mapsCompared, lossChecked int
	skipped := map[string]int{}
	for n, raw := range maps {
		key := keys[n]
		d, err := x.Open(key)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		host := d.Edits.(*mapInspectionEdits)
		view, err := host.model.Map()
		if err != nil {
			t.Fatal(key, err)
		}
		sheets, sheetErr := unitSheets(view, x.table)
		game, gameErr := alm.Open(raw)
		if gameErr != nil {
			t.Fatal(key, gameErr)
		}
		mission, missionErr := StartMissionFrom(game, key, sourceMission(key), x.table, mapload.DifficultyNormal, nil)
		switch {
		case sheetErr != nil && missionErr != nil:
			skipped["neither the popup nor the game builds the map"] += len(game.Units)
			continue
		case sheetErr != nil || missionErr != nil:
			t.Fatalf("%s: popup build %v, game build %v", key, sheetErr, missionErr)
		}
		mapsCompared++
		authored, _ := alm.Open(raw)
		for i := range authored.Units {
			e, ok := mission.World.Entity(sim.EntityID(i))
			if !ok || e.MapUnitID != authored.Units[i].UnitID {
				skipped["the game's world holds no actor with this placement's id"]++
				continue
			}
			worn, _ := mission.World.EquippedItems(e.ID)
			carried, _ := mission.World.CarriedItems(e.ID)
			if why := sheetMismatch(sheets[i], e, worn, carried); why != "" {
				t.Errorf("%s unit %d: %s", key, i, why)
				continue
			}
			compared++
			v := sheets[i].Values
			line := x.inspectionUTF8(fmt.Sprintf(x.word("editor.unit.combat"), v.ToHit, v.DamageMin, v.DamageMax, v.Defence, v.Absorption))
			rec := d.Records[recordOf(d, "Unit", i)]
			if !slices.ContainsFunc(rec.Lines, func(l ui.InspectionLine) bool { return l.Text == line }) {
				t.Errorf("%s unit %d: the popup does not show %q", key, i, line)
			}
			if lossChecked == 0 {
				perturbed := sheets[i]
				perturbed.Values.ToHit++
				if sheetMismatch(perturbed, e, worn, carried) == "" {
					t.Fatal("loss control: a perturbed to-hit is not reported")
				}
				perturbed = sheets[i]
				perturbed.Worn[0] = sim.ItemInstance{Code: worn[0].Code + 1}
				if sheetMismatch(perturbed, e, worn, carried) == "" {
					t.Fatal("loss control: a perturbed weapon is not reported")
				}
				lossChecked++
			}
		}
	}
	t.Logf("%d maps compared; %d placements compared; not compared: %v", mapsCompared, compared, skipped)
	if compared == 0 || lossChecked == 0 {
		t.Fatal("nothing was compared")
	}
}

// TestReleaseMapEditorUnitTierCensus reports, per root, the creature
// placements per tier, each class with several tiers and the keys that
// resolve for it, and every shipped (class, key) of the Units arm that
// resolves to no row.
func TestReleaseMapEditorUnitTierCensus(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	keys, maps := installedMapBytes(t, x)
	perTier := map[uint16]int{}
	classes := map[int16][]uint16{}
	unresolved := map[string][]string{}
	creatures := 0
	for n, raw := range maps {
		m, err := alm.Open(raw)
		if err != nil {
			continue
		}
		for i, u := range m.Units {
			r := mapload.Resolve(u, x.table)
			if r.Arm != mapload.ArmUnits {
				continue
			}
			creatures++
			if !r.Found() {
				k := fmt.Sprintf("class %d key %d", u.ClassID, u.ClassSubID)
				unresolved[k] = append(unresolved[k], fmt.Sprintf("%s#%d", keys[n], i))
				continue
			}
			tiers := x.tiersOf(u)
			if len(tiers) > 1 {
				perTier[u.ClassSubID]++
				if classes[u.ClassID] == nil {
					for _, tr := range tiers {
						classes[u.ClassID] = append(classes[u.ClassID], tr.Face)
					}
				}
			}
		}
	}
	var all []int
	for id := range x.units.Classes {
		if id >= -32768 && id <= 32767 {
			all = append(all, int(id))
		}
	}
	sort.Ints(all)
	var multi []string
	for _, id := range all {
		tiers := x.tiersOf(alm.Unit{ClassID: int16(id)})
		if len(tiers) < 2 {
			continue
		}
		var faces []string
		for _, tr := range tiers {
			faces = append(faces, fmt.Sprintf("%d %s", tr.Face, tr.Name))
		}
		multi = append(multi, fmt.Sprintf("class %d (%d palettes): %s", id, len(x.units.Classes[int32(id)].Tiers), strings.Join(faces, ", ")))
	}
	t.Logf("%d Units-arm placements on %d maps; multi-tier placements per tier key: %v", creatures, len(maps), perTier)
	t.Logf("%d classes have several tiers:\n%s", len(multi), strings.Join(multi, "\n"))
	t.Logf("shipped (class, key) resolving to no Units row: %d: %v", len(unresolved), unresolved)
	if len(multi) == 0 || len(perTier) == 0 {
		t.Fatal("the install has no multi-tier class or placement")
	}
	if dir := os.Getenv("AGAINROM_TIER_CENSUS"); dir != "" {
		text := fmt.Sprintf("placements %d maps %d per-tier %v\n%s\nunresolved %v\n", creatures, len(maps), perTier, strings.Join(multi, "\n"), unresolved)
		if err := os.WriteFile(filepath.Join(dir, "census.txt"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// unitWhere is the first unit, in map then record order, of an installed
// map that want accepts with its popup sheet, and its map's key.
func unitWhere(t *testing.T, x *MapInspector, what string, want func(u alm.Unit, r mapload.Resolution, s unitSheet) bool) (string, int) {
	t.Helper()
	keys, maps := installedMapBytes(t, x)
	for n, b := range maps {
		m, err := alm.Open(b)
		if err != nil {
			continue
		}
		sheets, err := unitSheets(m, x.table)
		if err != nil {
			continue
		}
		authored, _ := alm.Open(b)
		for i, u := range authored.Units {
			if want(u, mapload.Resolve(u, x.table), sheets[i]) {
				return keys[n], i
			}
		}
	}
	t.Fatalf("no installed map places %s", what)
	return "", 0
}

// scrollTo scrolls the open popup by whole wheel notches until the row
// holding heading is among the first three shown.
func scrollTo(t *testing.T, e *ui.MapEditor, heading string) {
	t.Helper()
	at := slices.Index(e.PopupLines(), heading)
	if at < 0 {
		t.Fatalf("the popup shows no %q heading", heading)
	}
	e.ScrollDetails(at / 3)
}

// sectionTo shows the section that holds heading, then scrolls it into view.
// Since the person setup window shows its sections one at a time, a
// person's heading may lie in a section other than the one shown.
func sectionTo(t *testing.T, c *editorClicker, e *ui.MapEditor, heading string) {
	t.Helper()
	for _, section := range []string{"identity", "equipment", "inventory", "stats"} {
		if slices.Contains(e.PopupLines(), heading) {
			break
		}
		if r, ok := e.ObjectSectionRect(section); ok {
			c.click(e, r, true, "the "+section+" section")
		}
	}
	scrollTo(t, e, heading)
}

// TestReleaseMapEditorUnitTierFrames writes the unit popup's tier and
// characteristics frames, rendered headless from the install, as PNGs into
// AGAINROM_EDITOR_FRAMES; without it the same steps run. In each frame the
// installed fonts keep every glyph, except under an open drop-down's list.
func TestReleaseMapEditorUnitTierFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	shot := func(e *ui.MapEditor, name string, cover image.Rectangle) {
		t.Helper()
		writeEditorFrame(t, e, filepath.Join(dir, name+".png"), dir != "", cover)
	}
	opened := func(key string) *ui.MapEditor {
		t.Helper()
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		closeMapPanel(t, e)
		return e
	}
	addUnit := func(e *ui.MapEditor, cell image.Point, tr unitTier) {
		t.Helper()
		_, value := creatureIdentity(tr)
		addWindow(t, e, "unit", map[string]string{"identity": fmt.Sprint(value), "cell": fmt.Sprintf("%d,%d", cell.X, cell.Y)})
	}

	_, tiers := fourTierClass(t, x)
	e := opened(NewMapSource("32x32"))
	for n, tr := range tiers {
		addUnit(e, image.Pt(12+2*n, 14), tr)
	}
	if m, _ := alm.Open(e.Document().SourceBytes); len(m.Units) != 4 {
		t.Fatalf("the new map holds %d units, want 4", len(m.Units))
	}
	_ = e.Key("escape")
	e.ShowCell(image.Pt(15, 14))
	shot(e, "tiers-on-map", image.Rectangle{})

	e = opened(NewMapSource("32x32"))
	_ = e.Key("unit")
	_ = e.Key("down")
	shot(e, "new-unit-tier", image.Rectangle{})

	sheetHeading := x.inspectionUTF8(x.word("editor.unit.sheet"))
	lootHeading := x.inspectionUTF8(x.word("editor.unit.loot"))
	key, _, i := tieredPlacement(t, x)
	e = opened(key)
	rec := recordOf(e.Document(), "Unit", i)
	e.Select(rec, true)
	fields := objectEdits(t, e).Fields("Unit", i)
	r, ok := e.FieldRect(fieldIndex(t, fields, "identity"))
	c.click(e, r, ok, "the identity field")
	shot(e, "creature-tier-dropdown", image.Rect(r.Min.X, r.Max.Y, 1280, 800))

	e = opened(key)
	e.Select(rec, true)
	sectionTo(t, c, e, sheetHeading)
	cell := e.Document().Records[rec].Cell
	p := e.CellPoint(cell)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, c.now)
	shot(e, "creature-sheet-and-hover", image.Rectangle{})

	key, i = unitWhere(t, x, "an equipped human", func(u alm.Unit, r mapload.Resolution, s unitSheet) bool {
		return r.Arm != mapload.ArmUnits && r.Found() && !s.Suppressed && slices.ContainsFunc(s.Worn[1:], func(it sim.ItemInstance) bool { return !it.Empty() })
	})
	e = opened(key)
	e.Select(recordOf(e.Document(), "Unit", i), true)
	sectionTo(t, c, e, sheetHeading)
	shot(e, "human-sheet", image.Rectangle{})
	sectionTo(t, c, e, lootHeading)
	shot(e, "human-loot", image.Rectangle{})

	key, i = unitWhere(t, x, "an NPC-named template", func(u alm.Unit, r mapload.Resolution, s unitSheet) bool {
		return r.Arm != mapload.ArmUnits && s.Suppressed
	})
	e = opened(key)
	e.Select(recordOf(e.Document(), "Unit", i), true)
	sectionTo(t, c, e, lootHeading)
	shot(e, "npc-template-loot", image.Rectangle{})

	key, i = unitWhere(t, x, "a named character", func(u alm.Unit, r mapload.Resolution, s unitSheet) bool {
		return r.Arm == mapload.ArmNPC && r.Found()
	})
	e = opened(key)
	e.Select(recordOf(e.Document(), "Unit", i), true)
	sectionTo(t, c, e, sheetHeading)
	shot(e, "npc-arm-sheet", image.Rectangle{})
}
