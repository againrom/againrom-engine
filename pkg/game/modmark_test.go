package game

import (
	"bytes"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

func modTestSet(cap int64) mod.Set {
	return mod.Set{Base: "rom1-en", Mods: []mod.SetEntry{{ID: "skill-cap", Version: "1.0.0", Digest: "abc",
		Settings: []mod.SettingValue{{Key: "skill_cap", Value: mod.Value{Kind: mod.KindInt, Int: cap}}}}}}
}

// modTestSAV is a SAV document of the town fixture, which has actor records
// with the skill blocks the projection reads.
func modTestSAV(t *testing.T) []byte {
	t.Helper()
	city, err := sav.CityFromData(cityfixture.City(false))
	if err != nil {
		t.Fatal(err)
	}
	update := sav.CityUpdate{Money: 1}
	for _, c := range city.Roster() {
		update.Characters = append(update.Characters, originalCityBaselineUpdate(c))
	}
	raw, err := city.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// modTestActors returns the indices of the document objects that carry skill
// blocks.
func modTestActors(t *testing.T, doc sav.DocumentData) []int {
	t.Helper()
	var out []int
	for i := range doc.Objects {
		if _, _, ok := readModObject(&doc.Objects[i]); ok {
			out = append(out, i)
		}
	}
	if len(out) == 0 {
		t.Fatal("the fixture has no actor with skill blocks")
	}
	return out
}

// raiseActor writes a true skill state beyond the original domain into object i.
func raiseActor(t *testing.T, doc *sav.DocumentData, i int, level int32) modMarkObject {
	t.Helper()
	r, err := rules.New(rules.Params{SkillCap: 150})
	if err != nil {
		t.Fatal(err)
	}
	m, hasXP, _ := readModObject(&doc.Objects[i])
	m.Levels[1], m.BaseLevels[1] = level, level
	if hasXP {
		m.XP[1] = uint32(r.SkillXP(level))
		m.Experience += m.XP[1]
	}
	if err := writeModObject(&doc.Objects[i], m, hasXP); err != nil {
		t.Fatal(err)
	}
	back, _, _ := readModObject(&doc.Objects[i])
	if back != m {
		t.Fatalf("write then read changed the state: %+v != %+v", back, m)
	}
	return m
}

func TestTheModMarkProjectsAndRestoresTrueSkillState(t *testing.T) {
	raw := modTestSAV(t)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	k := -1
	for _, i := range modTestActors(t, doc) {
		if _, withXP, _ := readModObject(&doc.Objects[i]); withXP {
			k = i
		}
	}
	if k < 0 {
		t.Fatal("the fixture has no actor with a per-slot experience block")
	}
	_, hasXP, _ := readModObject(&doc.Objects[k])
	true130 := raiseActor(t, &doc, k, 130)
	set := modTestSet(150)
	if err := markDocumentForMods(&doc, set, nil, nil); err != nil {
		t.Fatal(err)
	}
	projected, _, _ := readModObject(&doc.Objects[k])
	maxLevel, maxXP := originalSkillDomain()
	if projected.Levels[1] != maxLevel || projected.BaseLevels[1] != maxLevel {
		t.Fatalf("levels not projected: %+v", projected)
	}
	if hasXP && (projected.XP[1] != maxXP || projected.Experience != true130.Experience-(true130.XP[1]-maxXP)) {
		t.Fatalf("experience not projected: %+v", projected)
	}
	encoded, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, modMarkName) {
		t.Fatal("the leaf name is not visible in the bytes the fast path searches")
	}
	loaded, _, err := applyModMark(encoded, mapload.ModContext{Set: set})
	if err != nil {
		t.Fatal(err)
	}
	cold, err := sav.DecodeDocumentData(loaded)
	if err != nil {
		t.Fatal(err)
	}
	got, _, _ := readModObject(&cold.Objects[k])
	want := true130
	want.Object = 0
	if got != want {
		t.Fatalf("restored %+v, want %+v", got, want)
	}
	// Saving the restored document again yields the same projection and mark.
	again, err := sav.DecodeDocumentData(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if err := markDocumentForMods(&again, set, nil, nil); err != nil {
		t.Fatal(err)
	}
	reencoded, err := sav.EncodeDocumentData(again)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(reencoded, encoded) {
		t.Fatal("a load then a save changed the bytes of a marked save")
	}
}

func TestAMarkedSaveThatNeedsNoRestoreLoadsAsWritten(t *testing.T) {
	doc, err := sav.DecodeDocumentData(modTestSAV(t))
	if err != nil {
		t.Fatal(err)
	}
	set := modTestSet(150)
	if err := markDocumentForMods(&doc, set, nil, nil); err != nil {
		t.Fatal(err)
	}
	mark, ok, err := readModMark(doc)
	if err != nil || !ok || len(mark.Domain) != 0 || mark.Format != modMarkFormat || mark.SetDigest != set.Digest() {
		t.Fatalf("mark %+v %v %v", mark, ok, err)
	}
	encoded, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	loaded, _, err := applyModMark(encoded, mapload.ModContext{Set: set})
	if err != nil || !bytes.Equal(loaded, encoded) {
		t.Fatal(err)
	}
}

func TestWithoutModsTheDocumentIsLeftExactlyAsItIs(t *testing.T) {
	doc, err := sav.DecodeDocumentData(modTestSAV(t))
	if err != nil {
		t.Fatal(err)
	}
	raiseActor(t, &doc, modTestActors(t, doc)[0], 130)
	before, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := markDocumentForMods(&doc, mod.Set{}, nil, nil); err != nil {
		t.Fatal(err)
	}
	after, err := sav.EncodeDocumentData(doc)
	if err != nil || !bytes.Equal(before, after) || bytes.Contains(after, modMarkName) {
		t.Fatal("an unmodded save changed", err)
	}
	loaded, _, err := applyModMark(after, mapload.ModContext{})
	if err != nil || &loaded[0] != &after[0] {
		t.Fatal("an unmodded load touched the bytes", err)
	}
}

func TestLoadRefusalsNameTheMods(t *testing.T) {
	plain := modTestSAV(t)
	mark := func(set mod.Set) []byte {
		d, err := sav.DecodeDocumentData(plain)
		if err != nil {
			t.Fatal(err)
		}
		if err := markDocumentForMods(&d, set, nil, nil); err != nil {
			t.Fatal(err)
		}
		b, err := sav.EncodeDocumentData(d)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	other := modTestSet(150)
	other.Mods = append(other.Mods, mod.SetEntry{ID: "extra", Version: "2", Digest: "e"})
	for _, c := range []struct {
		name  string
		saved []byte
		ctx   mapload.ModContext
		want  string
	}{
		{"marked save, no mods", mark(modTestSet(150)), mapload.ModContext{}, "skill-cap 1.0.0"},
		{"marked save, other setting", mark(modTestSet(150)), mapload.ModContext{Set: modTestSet(140)}, `mod "skill-cap" setting skill_cap is 140, the saved game used 150`},
		{"marked save, extra mod", mark(modTestSet(150)), mapload.ModContext{Set: other}, `mod "extra" is active but the saved game did not use it`},
		{"marked save, one mod fewer", mark(other), mapload.ModContext{Set: modTestSet(150)}, `mod "extra" 2 is missing`},
		{"unmarked save, mods", plain, mapload.ModContext{Set: modTestSet(150)}, "-mods-accept-unmarked"},
	} {
		_, _, err := applyModMark(c.saved, c.ctx)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want a refusal naming %q", c.name, err, c.want)
		}
		if err != nil && !strings.Contains(err.Error(), ErrModMark.Error()) {
			t.Errorf("%s: the refusal is not an ErrModMark: %v", c.name, err)
		}
	}
	got, _, err := applyModMark(plain, mapload.ModContext{Set: modTestSet(150), AcceptUnmarked: true})
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatal("-mods-accept-unmarked did not load the unmarked save as it is", err)
	}
}

func TestAMarkThatDoesNotBelongToTheSaveIsRefused(t *testing.T) {
	doc, err := sav.DecodeDocumentData(modTestSAV(t))
	if err != nil {
		t.Fatal(err)
	}
	k := modTestActors(t, doc)[0]
	raiseActor(t, &doc, k, 130)
	set := modTestSet(150)
	if err := markDocumentForMods(&doc, set, nil, nil); err != nil {
		t.Fatal(err)
	}
	// Change the projected field the mark describes.
	m, hasXP, _ := readModObject(&doc.Objects[k])
	m.Levels[1] = 90
	if err := writeModObject(&doc.Objects[k], m, hasXP); err != nil {
		t.Fatal(err)
	}
	encoded, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := applyModMark(encoded, mapload.ModContext{Set: set}); err == nil || !strings.Contains(err.Error(), "does not match the save's actor") {
		t.Fatalf("%v", err)
	}
}

func TestAMarkWithAnUnreadableOrUnknownPayloadIsRefused(t *testing.T) {
	for _, payload := range []string{`{"format":2}`, `{"format":1,"surprise":3}`, `not json`} {
		doc, err := sav.DecodeDocumentData(modTestSAV(t))
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeMods(&doc.State, []byte(payload)); err != nil {
			t.Fatal(err)
		}
		encoded, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		for _, ctx := range []mapload.ModContext{{}, {Set: modTestSet(150)}, {Set: modTestSet(150), AcceptUnmarked: true}} {
			if _, _, err := applyModMark(encoded, ctx); err == nil {
				t.Errorf("%q accepted with mods active %v", payload, !ctx.Set.Empty())
			}
		}
	}
}

func TestProjectionKeepsEveryValueThatIsAlreadyInTheOriginalDomain(t *testing.T) {
	for _, m := range []modMarkObject{{}, {Levels: [6]int32{0, 100, 99, 5, 0, -3}, XP: [6]uint32{0, 13779612}, Experience: 99}} {
		if m.project() != m {
			t.Errorf("%+v changed", m)
		}
	}
	over := modMarkObject{Levels: [6]int32{0, 101}, BaseLevels: [6]int32{0, 120}, XP: [6]uint32{0, 13779612 + 50}, Experience: 20}
	p := over.project()
	if p.Levels[1] != 100 || p.BaseLevels[1] != 100 || p.XP[1] != 13779612 || p.Experience != 0 {
		t.Fatalf("%+v", p)
	}
}

// modTestAddArmor appends an Armor record of the given code and definition row
// to the document and returns its object index.
func modTestAddArmor(t *testing.T, doc *sav.DocumentData, code uint16, row uint8) int {
	t.Helper()
	r, err := savedCurrentItemRecord(sim.SavedItemObject{
		Token: sim.SavedObjectToken{T0C: row},
		Value: sim.ItemStack{Code: code, Count: 1, Kind: 1, Price: 120, Weight: 6, WeightPresent: true,
			SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: row}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc.Objects = append(doc.Objects, r)
	return len(doc.Objects) - 1
}

func TestTheModMarkHoldsAModItemsStandInAndRestoresIt(t *testing.T) {
	doc, err := sav.DecodeDocumentData(modTestSAV(t))
	if err != nil {
		t.Fatal(err)
	}
	k := modTestAddArmor(t, &doc, 0x071f, 31)
	bystander := modTestAddArmor(t, &doc, 0x0a09, 16)
	trueCode, _ := savedStructureValue(&doc.Objects[k], "F40")
	trueRow, _ := savedStructureValue(&doc.Objects[k], "T0C")
	item := mapload.ModItem{Mod: "skill-cap", Key: "x", Code: uint16(trueCode), Row: uint8(trueRow), StandInCode: 0x0a07, StandInRow: 15}
	set := modTestSet(150)
	before := doc.Objects[k]
	if err := markDocumentForMods(&doc, set, []mapload.ModItem{item}, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := savedStructureValue(&doc.Objects[k], "F40"); got != 0x0a07 {
		t.Fatalf("the object holds code %#x, want the stand-in", got)
	}
	if row, _ := savedStructureValue(&doc.Objects[k], "T0C"); row != 15 {
		t.Fatalf("the object holds row %d, want the stand-in's", row)
	}
	for _, v := range before.Values {
		if v.Name == "F40" || v.Name == "T0C" {
			continue
		}
		if got, _ := savedStructureValue(&doc.Objects[k], v.Name); got != v.Value {
			t.Fatalf("%s changed: %d -> %d", v.Name, v.Value, got)
		}
	}
	if v, _ := savedStructureValue(&before, "F40"); v != trueCode {
		t.Fatal("marking changed the record it was given")
	}
	mark, present, err := readModMark(doc)
	if err != nil || !present || len(mark.Items) != 1 || mark.Items[0] != (modMarkItem{Object: k, Code: item.Code, Row: item.Row}) {
		t.Fatalf("the mark holds %+v %v %v", mark.Items, present, err)
	}
	items := []mapload.ModItem{item}
	cold := doc
	cold.Objects = append([]sav.DocumentRecordData(nil), doc.Objects...)
	if err := restoreModItems(&cold, mark.Items, items); err != nil {
		t.Fatal(err)
	}
	if got, _ := savedStructureValue(&cold.Objects[k], "F40"); got != trueCode {
		t.Fatalf("the LOAD restored code %#x, want %#x", got, trueCode)
	}
	if got, _ := savedStructureValue(&cold.Objects[k], "T0C"); got != trueRow {
		t.Fatalf("the LOAD restored row %d, want %d", got, trueRow)
	}
	if got, _ := savedStructureValue(&doc.Objects[k], "F40"); got != 0x0a07 {
		t.Fatal("restoring changed the document it was copied from")
	}
	// A second SAVE of the restored document marks the same object the same way.
	if err := markDocumentForMods(&cold, set, items, nil); err != nil {
		t.Fatal(err)
	}
	if again, _, _ := readModMark(cold); len(again.Items) != 1 || again.Items[0] != mark.Items[0] {
		t.Fatalf("a second SAVE marks %+v", again.Items)
	}
	// The mark is refused where the item is not the active mods' or the
	// object is not the stand-in.
	if err := restoreModItems(&doc, mark.Items, nil); err == nil || !strings.Contains(err.Error(), "no active mod adds") {
		t.Fatalf("a mark of an item no mod adds: %v", err)
	}
	other := item
	other.StandInCode = 0x0a08
	if err := restoreModItems(&doc, mark.Items, []mapload.ModItem{other}); err == nil || !strings.Contains(err.Error(), "does not match the save's item object") {
		t.Fatalf("a mark whose stand-in differs: %v", err)
	}
	if err := restoreModItems(&doc, []modMarkItem{{Object: 9999, Code: item.Code}}, items); err == nil || !strings.Contains(err.Error(), "item object 9999") {
		t.Fatalf("a mark of an object the save lacks: %v", err)
	}
	if err := restoreModItems(&doc, []modMarkItem{mark.Items[0], mark.Items[0]}, items); err == nil {
		t.Fatal("an object named twice was accepted")
	}
	if err := restoreModItems(&doc, []modMarkItem{{Object: bystander, Code: item.Code}}, items); err == nil {
		t.Fatal("a mark of an object that holds another item was accepted")
	}
	// Without items the mark records none, and the document keeps its codes.
	plain, err := sav.DecodeDocumentData(modTestSAV(t))
	if err != nil {
		t.Fatal(err)
	}
	pk := modTestAddArmor(t, &plain, 0x071f, 31)
	if err := markDocumentForMods(&plain, set, nil, nil); err != nil {
		t.Fatal(err)
	}
	plainMark, _, _ := readModMark(plain)
	if len(plainMark.Items) != 0 {
		t.Fatalf("items recorded without a mod item: %+v", plainMark.Items)
	}
	if got, _ := savedStructureValue(&plain.Objects[pk], "F40"); got != trueCode {
		t.Fatal("an item the mods do not add was changed")
	}
	if got, _ := savedStructureValue(&doc.Objects[bystander], "F40"); got != 0x0a09 {
		t.Fatalf("an item the mods do not add holds %#x", got)
	}
}
