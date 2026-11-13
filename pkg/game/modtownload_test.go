package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// modTownFront is a front end over f's install whose table runs set under r
// with items, as a game started under changed mods does.
func modTownFront(t *testing.T, f *FrontEnd, items []mapload.ModItem, set mod.Set, r sim.Rules) *FrontEnd {
	t.Helper()
	table := *f.Table
	table.Mods = mapload.ModContext{Items: items}
	next := &FrontEnd{InstallResources: f.InstallResources, RuntimeServices: f.RuntimeServices}
	next.Table = &table
	if err := next.SetMods(r, set, false); err != nil {
		t.Fatal(err)
	}
	return next
}

// modTownSave is the bytes of an ordinary SAVE of f.
func modTownSave(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(store.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// modFilesEdited is set with its first mod's files changed: the same id,
// version and settings.
func modFilesEdited(set mod.Set) mod.Set {
	set.Mods = slices.Clone(set.Mods)
	set.Mods[0].Digest += "-edited"
	return set
}

// modLoadRows is the list a LOAD's notice carries.
func modLoadRows(t *testing.T, err error) []ui.ModMismatchRow {
	t.Helper()
	var n *ui.ModLoadNotice
	if !errors.As(err, &n) || n.Changes.Reason != ui.ModMismatchChanged {
		t.Fatalf("the LOAD answered %v, want a notice of changed mods", err)
	}
	return n.Changes.Rows
}

func modTownCodes(f *FrontEnd, id string) []uint16 {
	var codes []uint16
	for _, m := range f.Carried {
		if m.ID != id {
			continue
		}
		for _, it := range mapload.MemberCarriedItems(m, f.Table) {
			codes = append(codes, it.Code)
		}
		for _, it := range mapload.MemberItemEquipment(m, f.Table) {
			if !it.Empty() {
				codes = append(codes, it.Code)
			}
		}
	}
	return codes
}

// modTownBuy buys one unit of code from the armour shelf for party member i.
func modTownBuy(t *testing.T, f *FrontEnd, i int, code uint16) {
	t.Helper()
	f.Shop = NewShop(1000)
	f.Shop.shelves[ShelfArmour] = []ShopItem{{Code: data.ItemCode(code), Kind: 1, Count: 1, Price: 10, Weight: 1, WeightPresent: true}}
	s := f.TownScreen().(*townScreen)
	s.room, s.shopMember = roomShop, i
	takeFromShelf(s, roomArmour)
	if action := click(s, ui.ShopControlButton, 1); !strings.HasPrefix(action.Msg, "bought") {
		t.Fatal(action.Msg)
	}
}

// modTownKeyless is a SAVE as the engine wrote it before marks named their
// items: the mark's items and layers without mod and key.
func modTownKeyless(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	mark := modTownMark(t, raw)
	for i := range mark.Items {
		mark.Items[i].Mod, mark.Items[i].Key = "", ""
	}
	for i := range mark.Layers {
		mark.Layers[i].Mod, mark.Layers[i].Key = "", ""
	}
	payload, err := json.Marshal(mark)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeMods(&doc.State, payload); err != nil {
		t.Fatal(err)
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// modTownMark is the mark of a SAVE.
func modTownMark(t *testing.T, raw []byte) modMark {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	mark, present, err := readModMark(doc)
	if err != nil || !present {
		t.Fatal("no mod mark", err)
	}
	return mark
}

// A town save whose mod no longer defines an item in the hero's pack loads
// with the item gone and the list naming it; the loaded file is untouched,
// and the next SAVE carries the new mark and loads with no list.
func TestATownSaveLoadsWithoutThePackItemItsModNoLongerDefines(t *testing.T) {
	f := currentTrainingCity(t)
	item := mapload.ModItem{Mod: "skill-cap", Key: "x", Code: 0x0e01, StandInCode: 0x0e02}
	f.Table.Mods.Items = []mapload.ModItem{item}
	set := modTestSet(150)
	if err := f.SetMods(rules.Default(), set, false); err != nil {
		t.Fatal(err)
	}
	buyTrainingItem(t, f)
	raw := modTownSave(t, f)
	kept := bytes.Clone(raw)
	hero := trainingPartyMember(t, f, "hero")

	same := modTownFront(t, f, []mapload.ModItem{item}, set, rules.Default())
	if _, town, err := same.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("the unchanged mods' LOAD", town, err)
	}
	if !slices.Contains(modTownCodes(same, "hero"), item.Code) {
		t.Fatal("the unchanged mods' LOAD lost the item")
	}

	edited := modFilesEdited(set)
	next := modTownFront(t, f, nil, edited, rules.Default())
	_, town, err := next.RestoreOriginal(raw)
	rows := modLoadRows(t, err)
	if want := []ui.ModMismatchRow{{Kind: ui.ModMismatchItemPack, Who: hero.Name, Mod: "skill-cap", What: "x"}}; !town || !reflect.DeepEqual(rows, want) {
		t.Fatalf("town %v rows %+v, want %+v", town, rows, want)
	}
	if codes := modTownCodes(next, "hero"); slices.Contains(codes, item.Code) || slices.Contains(codes, item.StandInCode) {
		t.Fatalf("the hero still holds %#x", codes)
	}
	root := cityShopRoots(t, next, slices.IndexFunc(next.Carried, func(p mapload.PartyMember) bool { return p.ID == "hero" }))
	if len(root.Pack) != 0 {
		t.Fatalf("the pack root still names %v", root.Pack)
	}
	if !bytes.Equal(raw, kept) {
		t.Fatal("the LOAD changed the file it read")
	}

	again := modTownSave(t, next)
	if mark := modTownMark(t, again); mark.SetDigest != edited.Digest() || len(mark.Items) != 0 {
		t.Fatalf("the next SAVE marks %s with %d items, want %s", mark.SetDigest, len(mark.Items), edited.Digest())
	}
	third := modTownFront(t, f, nil, edited, rules.Default())
	if _, town, err := third.RestoreOriginal(again); err != nil || !town {
		t.Fatal("the next SAVE's LOAD", town, err)
	}
}

// A town save whose mod changed only its files, with nothing the active mods
// no longer hold, loads with no list.
func TestATownSaveWhoseModChangedOnlyItsFilesLoadsWithNoList(t *testing.T) {
	f := currentTrainingCity(t)
	set := modTestSet(150)
	if err := f.SetMods(rules.Default(), set, false); err != nil {
		t.Fatal(err)
	}
	raw := modTownSave(t, f)
	for _, now := range []mod.Set{modFilesEdited(set), func() mod.Set {
		s := modFilesEdited(set)
		s.Mods[0].Version = "1.1.0"
		return s
	}()} {
		next := modTownFront(t, f, nil, now, rules.Default())
		if _, town, err := next.RestoreOriginal(raw); err != nil || !town {
			t.Fatalf("%+v: %v %v", now, town, err)
		}
	}
}

// A town save whose mod no longer defines the item the hero wears loads with
// the slot empty and the item in no pack.
func TestATownSaveLoadsWithoutTheWornItemItsModNoLongerDefines(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Town.open = true
	f.Carried[0].ID, f.Carried[0].Name = "hero", "Hero"
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	f.Carried[0].Profile.Fighter = true
	f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	item := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
	slot, _ := EquipTarget(data.ItemCode(item.Code), f.Table)
	cityShopSetTestPack(t, f, 0, sim.StackItem(item, 1))
	cityShopGraph(t, f)
	if a := screen.shopEquipFromPack(screen.packCell(0)); a.Msg != "worn" {
		t.Fatal(a)
	}
	doc, err := sav.DecodeDocumentData(cityProjectionSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	row := uint32(0)
	for i := range doc.Objects {
		if code, err := savedStructureValue(&doc.Objects[i], "F40"); err == nil && uint16(code) == item.Code && equipmentRecordClass(doc.Objects[i].Class) {
			row, _ = savedStructureValue(&doc.Objects[i], "T0C")
		}
	}
	worn := mapload.ModItem{Mod: "skill-cap", Key: "w", Code: item.Code, Row: uint8(row), StandInCode: item.Code, StandInRow: uint8(row)}
	f.Table.Mods.Items = []mapload.ModItem{worn}
	set := modTestSet(150)
	if err := f.SetMods(rules.Default(), set, false); err != nil {
		t.Fatal(err)
	}
	raw := cityProjectionSave(t, f)
	if mark := modTownMark(t, raw); len(mark.Items) != 1 {
		t.Fatalf("the mark holds %+v", mark.Items)
	}
	next := modTownFront(t, f, nil, modFilesEdited(set), rules.Default())
	_, town, err := next.RestoreOriginal(raw)
	rows := modLoadRows(t, err)
	if want := []ui.ModMismatchRow{{Kind: ui.ModMismatchItemWorn, Who: "Hero", Mod: "skill-cap", What: "w"}}; !town || !reflect.DeepEqual(rows, want) {
		t.Fatalf("town %v rows %+v, want %+v", town, rows, want)
	}
	if codes := modTownCodes(next, "hero"); slices.Contains(codes, item.Code) {
		t.Fatalf("the hero still holds %#x", codes)
	}
	if root := cityShopRoots(t, next, 0); root.Worn[slot-1] != 0 || len(root.Pack) != 0 {
		t.Fatalf("roots %+v", root)
	}
	if again := cityShopColdLoad(t, next); len(modTownCodes(again, "hero")) != 0 {
		t.Fatal("the next SAVE brought the item back")
	}
}

// Under changed mods a stamped item stays the item its mod and key name,
// though its code moved: an item inserted or removed before it shifts the
// codes, and two items sharing a stand-in do not trade places. Nothing went,
// so the LOAD answers no list.
func TestATownSaveKeepsItsModItemsWhenTheirCodesShift(t *testing.T) {
	x := mapload.ModItem{Mod: "skill-cap", Key: "x", StandInCode: 0x0e02}
	y := mapload.ModItem{Mod: "skill-cap", Key: "y", StandInCode: 0x0e02}
	at := func(m mapload.ModItem, code uint16) mapload.ModItem {
		m.Code = code
		return m
	}
	for _, c := range []struct {
		name       string
		saved, now []mapload.ModItem
		// bought and want are the hero's and the companion's code, 0 for none.
		bought, want [2]uint16
	}{
		{"an item inserted before", []mapload.ModItem{at(x, 0x0e01)}, []mapload.ModItem{at(y, 0x0e01), at(x, 0x0e03)},
			[2]uint16{0x0e01}, [2]uint16{0x0e03}},
		{"an item removed before", []mapload.ModItem{at(y, 0x0e01), at(x, 0x0e03)}, []mapload.ModItem{at(x, 0x0e01)},
			[2]uint16{0x0e03}, [2]uint16{0x0e01}},
		{"two items sharing a stand-in in the other order", []mapload.ModItem{at(x, 0x0e01), at(y, 0x0e03)},
			[]mapload.ModItem{at(y, 0x0e01), at(x, 0x0e03)}, [2]uint16{0x0e01, 0x0e03}, [2]uint16{0x0e03, 0x0e01}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := currentTrainingCity(t)
			f.Table.Mods.Items = c.saved
			set := modTestSet(150)
			if err := f.SetMods(rules.Default(), set, false); err != nil {
				t.Fatal(err)
			}
			for i, code := range c.bought {
				if code != 0 {
					modTownBuy(t, f, i, code)
				}
			}
			raw := modTownSave(t, f)
			next := modTownFront(t, f, c.now, modFilesEdited(set), rules.Default())
			if _, town, err := next.RestoreOriginal(raw); err != nil || !town {
				t.Fatal("LOAD", town, err)
			}
			for i, want := range c.want {
				if want == 0 {
					continue
				}
				got := modTownCodes(next, next.Carried[i].ID)
				if !slices.Contains(got, want) || c.bought[i] != want && slices.Contains(got, c.bought[i]) {
					t.Errorf("member %d holds %#x, want %#x", i, got, want)
				}
			}
		})
	}
}

// A save written before marks named their items: under changed mods it loads
// when its mark holds no item, and one that holds an item is refused with its
// own line; under its own mods it loads with the item.
func TestATownSaveWrittenBeforeItemKeys(t *testing.T) {
	item := mapload.ModItem{Mod: "skill-cap", Key: "x", Code: 0x0e01, StandInCode: 0x0e02}
	f := currentTrainingCity(t)
	f.Table.Mods.Items = []mapload.ModItem{item}
	set := modTestSet(150)
	if err := f.SetMods(rules.Default(), set, false); err != nil {
		t.Fatal(err)
	}
	empty := modTownKeyless(t, modTownSave(t, f))
	modTownBuy(t, f, 0, item.Code)
	held := modTownKeyless(t, modTownSave(t, f))
	if mark := modTownMark(t, held); len(mark.Items) != 1 || mark.Items[0].Key != "" || mark.keyed() {
		t.Fatalf("the keyless mark holds %+v", mark.Items)
	}
	edited := modFilesEdited(set)
	items := []mapload.ModItem{item}
	if _, town, err := modTownFront(t, f, items, edited, rules.Default()).RestoreOriginal(empty); err != nil || !town {
		t.Fatal("a keyless save without items", town, err)
	}
	_, _, err := modTownFront(t, f, items, edited, rules.Default()).RestoreOriginal(held)
	var refusal interface{ ModMismatch() ui.ModMismatch }
	if !errors.As(err, &refusal) || refusal.ModMismatch().Reason != ui.ModMismatchKeyless ||
		len(refusal.ModMismatch().Rows) != 1 || refusal.ModMismatch().Rows[0].Kind != ui.ModMismatchFiles ||
		!strings.Contains(err.Error(), "written before saves named their mod items") {
		t.Fatalf("a keyless save with an item: %v", err)
	}
	same := modTownFront(t, f, items, set, rules.Default())
	if _, town, err := same.RestoreOriginal(held); err != nil || !town || !slices.Contains(modTownCodes(same, "hero"), item.Code) {
		t.Fatal("a keyless save under its own mods", town, err)
	}
}

// A tavern hire, whom no town picker shows, gives up a removed item through
// the same take as any member. The whole LOAD with a hire needs install hire
// data, so the take runs on the restored session directly.
func TestARemovedItemComesOffATavernHireToo(t *testing.T) {
	f := currentTrainingCity(t)
	if len(f.Carried) < 2 {
		t.Fatal("the fixture party has no companion")
	}
	modTownBuy(t, f, 1, 0x0e01)
	root := cityShopRoots(t, f, 1)
	if len(root.Pack) != 1 {
		t.Fatalf("the companion's pack root is %v", root.Pack)
	}
	// A human hire, tavern type 3, keeps his own pack.
	f.Carried[1].MercenaryType = 3
	if !f.Carried[1].Hired() || f.Carried[1].UnitMember() || townPickerIndex(f.Carried, 1) == 1 {
		t.Fatal("the companion is not a hire the pickers skip")
	}
	hire, hero := f.Carried[1].ID, slices.Clone(modTownCodes(f, f.Carried[0].ID))
	if err := f.takeModLoss(&f.CampaignSession, modTownLoss{items: []modLostItem{{member: hire, id: root.Pack[0]}}}); err != nil {
		t.Fatal(err)
	}
	if codes := modTownCodes(f, hire); slices.Contains(codes, 0x0e01) {
		t.Fatalf("the hire still holds %#x", codes)
	}
	if root := cityShopRoots(t, f, 1); len(root.Pack) != 0 {
		t.Fatalf("the hire's pack root still names %v", root.Pack)
	}
	if !reflect.DeepEqual(modTownCodes(f, f.Carried[0].ID), hero) {
		t.Fatal("the take touched another member")
	}
}

// A town save whose skills stand above the cap its mod now gives loads with
// each lowered skill held to the cap and listed.
func TestATownSaveLoadsWithItsSkillsHeldToTheLoweredCap(t *testing.T) {
	f := currentTrainingCity(t)
	high, err := rules.New(rules.Params{SkillCap: 150})
	if err != nil {
		t.Fatal(err)
	}
	set := modTestSet(150)
	if err := f.SetMods(high, set, false); err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(modTownSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range modTestActors(t, doc) {
		raiseActor(t, &doc, i, 120)
	}
	if err := markDocumentForMods(&doc, set, nil, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	same := modTownFront(t, f, nil, set, high)
	if _, _, err := same.RestoreOriginal(raw); err != nil || trainingPartyMember(t, same, "hero").Hero.Skill[1] != 120 {
		t.Fatal("the unchanged mods' LOAD", err)
	}
	low, err := rules.New(rules.Params{SkillCap: 110})
	if err != nil {
		t.Fatal(err)
	}
	// Edited files and another setting value take the same one path.
	for _, now := range []mod.Set{modFilesEdited(set), modTestSet(110)} {
		next := modTownFront(t, f, nil, now, low)
		_, town, err := next.RestoreOriginal(raw)
		rows := modLoadRows(t, err)
		hero := trainingPartyMember(t, next, "hero")
		what := "blade"
		if hero.Mage {
			what = "fire"
		}
		want := ui.ModMismatchRow{Kind: ui.ModMismatchSkill, Who: hero.Name, What: what, Saved: "120", Current: "110"}
		if !town || !slices.Contains(rows, want) || hero.Hero.Skill[1] != 110 {
			t.Fatalf("%v: town %v rows %+v skill %d, want %+v", now, town, rows, hero.Hero.Skill[1], want)
		}
		for _, r := range rows {
			if r.Kind != ui.ModMismatchSkill || !slices.ContainsFunc(next.Carried, func(p mapload.PartyMember) bool { return p.Name == r.Who }) {
				t.Fatalf("row %+v names no party member's skill", r)
			}
		}
	}
}

// A town save's campaign under changed mods drops each mission the active
// campaign no longer declares, as a child, a selection and an inn candidate;
// the same file under unchanged mods keeps today's refusal.
func TestATownSaveUnderChangedModsDropsTheMissionsItsCampaignLost(t *testing.T) {
	c, projection := restoredCampaignFixture()
	c.Side, c.Offered = []int{41}, []int{30, 40, 50, 60}
	for _, selected := range []uint32{41, 51} {
		p := projection
		p.SelectedMission = selected
		saved := originalSaveWithCampaign(t, 0, p)
		sf, err := sav.Open(saved)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeOriginalCampaign(sf, saved, c, originalFileState{}, false); err == nil || !strings.Contains(err.Error(), "mission 51 is not a declared side mission") {
			t.Fatalf("unchanged mods: %v", err)
		}
		d, err := decodeOriginalCampaign(sf, saved, c, originalFileState{}, true)
		if err != nil || !slices.Equal(d.dropped, []int{51}) {
			t.Fatalf("changed mods: %v dropped %v", err, d)
		}
		want := int(selected)
		if selected == 51 {
			want = 50
		}
		if len(d.progress.children) != 1 || d.progress.children[0].mission != 41 || d.progress.selected != want ||
			!slices.Equal(d.progress.innNPC, []int{1, 2}) || !slices.Equal(d.progress.innMission, []int{0, 50}) {
			t.Fatalf("progress %+v", d.progress)
		}
	}
	if rows := droppedMissionRows([]int{51}); !reflect.DeepEqual(rows, []ui.ModMismatchRow{{Kind: ui.ModMismatchMission, What: "51"}}) {
		t.Fatalf("rows %+v", rows)
	}
}

// A town save's guest records under changed mods take the number their
// mission has now, or are dropped and listed when it is gone.
func TestATownSaveUnderChangedModsHoldsItsGuestsToTheDeclaredMissions(t *testing.T) {
	mods := mapload.ModContext{Missions: mod.MissionData{Missions: []mod.Mission{{Mod: "m", Key: "a", Number: 27}}}}
	s := &currentSessionData{GuestWaiting: true, Guest: &currentGuest{Mod: "m", Key: "a", Number: 25,
		Previous: &currentGuest{Mod: "m", Key: "gone", Number: 26}}}
	rows := adoptChangedGuests(s, mods)
	if !reflect.DeepEqual(rows, []ui.ModMismatchRow{{Kind: ui.ModMismatchGuest, Mod: "m", What: "26"}}) ||
		s.Guest == nil || s.Guest.Number != 27 || s.Guest.Previous != nil || !s.GuestWaiting {
		t.Fatalf("rows %+v session %+v", rows, s)
	}
	s = &currentSessionData{GuestWaiting: true, Guest: &currentGuest{Mod: "m", Key: "gone", Number: 26}}
	if rows := adoptChangedGuests(s, mods); len(rows) != 1 || s.Guest != nil || s.GuestWaiting {
		t.Fatalf("rows %+v session %+v", rows, s)
	}
	if rows := adoptChangedGuests(nil, mods); rows != nil {
		t.Fatal(rows)
	}
}

// A mission save under the same change, and a town save under another set of
// mods, keep today's refusal and its line.
func TestChangedModsStillRefuseAMissionSaveAndAnotherSet(t *testing.T) {
	doc, err := sav.DecodeDocumentData(modTestSAV(t))
	if err != nil {
		t.Fatal(err)
	}
	set := modTestSet(150)
	if err := markDocumentForMods(&doc, set, nil, nil); err != nil {
		t.Fatal(err)
	}
	town, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	doc.Head.Mission = 3
	mission, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	edited := modFilesEdited(set)
	if load, err := applyModMark(town, mapload.ModContext{Set: edited}, rules.Rules{}); err != nil || !load.changed {
		t.Fatalf("a town save under edited files: %v", err)
	}
	if load, err := applyModMark(town, mapload.ModContext{Set: modTestSet(140)}, rules.Rules{}); err != nil || !load.changed {
		t.Fatalf("a town save under another setting: %v", err)
	}
	extra := modFilesEdited(set)
	extra.Mods = append(extra.Mods, mod.SetEntry{ID: "extra", Version: "2", Digest: "e"})
	renamed := modFilesEdited(set)
	renamed.Mods[0].ID = "skill-cap-2"
	for _, c := range []struct {
		name  string
		saved []byte
		now   mod.Set
	}{
		{"a mission save under edited files", mission, edited},
		{"a town save with one mod more", town, extra},
		{"a town save under a renamed mod", town, renamed},
		{"a mission save under another setting", mission, modTestSet(140)},
	} {
		_, err := applyModMark(c.saved, mapload.ModContext{Set: c.now}, rules.Rules{})
		line := "this save was written under a different mod set: " + strings.Join(mod.Differences(set, c.now), "; ")
		var refusal interface{ ModMismatch() ui.ModMismatch }
		if !errors.As(err, &refusal) || refusal.ModMismatch().Reason != ui.ModMismatchSet || err.Error() != line && !strings.HasSuffix(err.Error(), line) {
			t.Errorf("%s: %v, want %q", c.name, err, line)
		}
	}
}
