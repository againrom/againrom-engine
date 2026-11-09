package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// partyCrossingRecordFile holds, as numbers, what every crossing of the
// party's items produces from the fixtures below: each list's units with
// every form field, the town graph's roots and records, the SAV's hash.
const partyCrossingRecordFile = "testdata/partyitemcrossings.golden"

// partyCrossingTrace numbers each recorded line by crossing and step.
type partyCrossingTrace struct{ b strings.Builder }

// line records v: a unit list as each unit's form fields followed by the
// hash of every field of every unit, and anything long as its size and hash.
func (w *partyCrossingTrace) line(crossing, step int, v any) {
	var nums []string
	partyCrossingNumbers(&nums, reflect.ValueOf(v))
	if items, ok := v.([]sim.ItemInstance); ok && len(items) <= 32 {
		fmt.Fprintf(&w.b, "%d %d %d", crossing, step, len(items))
		for _, it := range items {
			fmt.Fprintf(&w.b, " %d %d %d %d %v %d %d", it.Code, it.Kind, it.Price, it.Weight, partyCrossingFlag(it.WeightPresent), it.ObjectID, len(it.Effects))
			for _, e := range it.Effects {
				fmt.Fprintf(&w.b, " %d %d %d", e.Kind, e.Mode, e.Operand)
			}
		}
		fmt.Fprintf(&w.b, " %d\n", partyCrossingHash(nums))
		return
	}
	if len(nums) > 64 {
		nums = []string{fmt.Sprint(len(nums)), fmt.Sprint(partyCrossingHash(nums))}
	}
	fmt.Fprintf(&w.b, "%d %d %s\n", crossing, step, strings.Join(nums, " "))
}

func partyCrossingFlag(b bool) int {
	if b {
		return 1
	}
	return 0
}

func partyCrossingHash(nums []string) uint64 {
	h := fnv.New64a()
	for _, n := range nums {
		h.Write([]byte(n))
		h.Write([]byte{' '})
	}
	return h.Sum64()
}

// partyCrossingNumbers writes v as numbers: a walk of every field, a
// length before each list, a presence before each pointer, keys in order,
// and a string as its FNV hash.
func partyCrossingNumbers(out *[]string, v reflect.Value) {
	if !v.IsValid() {
		*out = append(*out, "0")
		return
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		*out = append(*out, fmt.Sprint(v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		*out = append(*out, fmt.Sprint(v.Uint()))
	case reflect.Bool:
		n := 0
		if v.Bool() {
			n = 1
		}
		*out = append(*out, fmt.Sprint(n))
	case reflect.String:
		h := fnv.New64a()
		h.Write([]byte(v.String()))
		*out = append(*out, fmt.Sprint(h.Sum64()))
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice {
			*out = append(*out, fmt.Sprint(v.Len()))
		}
		for i := 0; i < v.Len(); i++ {
			partyCrossingNumbers(out, v.Index(i))
		}
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			*out = append(*out, "0")
			return
		}
		*out = append(*out, "1")
		partyCrossingNumbers(out, v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			partyCrossingNumbers(out, v.Field(i))
		}
	case reflect.Map:
		keys := v.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			var x, y []string
			partyCrossingNumbers(&x, a)
			partyCrossingNumbers(&y, b)
			return strings.Compare(strings.Join(x, " "), strings.Join(y, " "))
		})
		*out = append(*out, fmt.Sprint(len(keys)))
		for _, k := range keys {
			partyCrossingNumbers(out, k)
			partyCrossingNumbers(out, v.MapIndex(k))
		}
	default:
		panic("partyCrossingNumbers: " + v.Kind().String())
	}
}

// partyCrossingFront is the bag fixture's campaign with bag in the bag, the
// potion table loaded and the town's item graph built from the party, with
// a record for every Effect, so a retired record shows.
func partyCrossingFront(t *testing.T, bag []sim.ItemInstance) (*FrontEnd, *townScreen) {
	t.Helper()
	f, _, screen, _ := secondBagFront(t, bag)
	secondBagPotionTable(f)
	return f, screen
}

func partyCrossingGraph(t *testing.T, f *FrontEnd) {
	t.Helper()
	p, err := newCityObjectProjection(nil, f.Carried, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	g := p.graph
	g.EffectRecords = map[sim.SavedObjectID]sim.SavedEffectObject{}
	for k, id := range g.Effects {
		serial := uint32(k + 1)
		g.EffectRecords[id] = sim.SavedEffectObject{ID: id, Token: sim.SavedObjectToken{Identity: 0x62000000 + serial*16, RuntimeID: serial, T0E: 0x21}}
	}
	f.Town.cityObjects = g
}

// partyCrossingState records the bag, each member's pack and worn items
// and the town's graph.
func partyCrossingState(w *partyCrossingTrace, crossing, step int, f *FrontEnd) {
	w.line(crossing, step, *f.Town.second.heroBag())
	for k := range f.Carried {
		w.line(crossing, step+1+k, mapload.MemberCarriedItems(f.Carried[k], f.Table))
		w.line(crossing, step+1+k, f.Carried[k].WornItems)
		w.line(crossing, step+1+k, f.Carried[k].Layers)
	}
	if f.Town.cityObjects != nil {
		w.line(crossing, step+10, *f.Town.cityObjects)
	}
}

func partyCrossingPack(t *testing.T, f *FrontEnd, member int, items []sim.ItemInstance) {
	t.Helper()
	next, ok := memberWithPack(f.Carried[member], items, f.Table, true)
	if !ok {
		t.Fatalf("member %d's pack refused the units", member)
	}
	f.Carried[member] = next
}

// partyCrossingUnits are a pack's four units: a potion of the mission's own
// making, a weapon with its weight, each with a bag form; a unit at a negative
// price and an armour with a second effect, with none.
func partyCrossingUnits(f *FrontEnd) []sim.ItemInstance {
	carried := secondBagWeapon.Clone()
	carried.Weight, carried.WeightPresent = 9, true
	unpriced := sim.ItemInstance{Code: 0x0e4d, Price: -1}
	twice := sim.ItemInstance{Code: 0x1402, Kind: 1, Price: 30, Effects: []sim.ItemEffect{{Kind: 41, Operand: 5 | 2<<16}, {Kind: 7}}}
	return []sim.ItemInstance{mapload.ItemInstanceFromCode(secondBagPotion.Code, f.Table), unpriced, carried, twice}
}

// partyCrossings runs every crossing over its fixture and records each result.
func partyCrossings(t *testing.T) string {
	w := &partyCrossingTrace{}

	// 1: the bag leaves to the hero's pack with a location's departure; an
	// empty bag leaves nothing.
	f, _ := partyCrossingFront(t, secondBagSample())
	partyCrossingPack(t, f, 0, []sim.ItemInstance{secondBagWeapon})
	partyCrossingGraph(t, f)
	partyCrossingState(w, 1, 0, f)
	f.CampaignSession.leftLocation(f.townInstall())
	partyCrossingState(w, 1, 20, f)
	f.CampaignSession.leftLocation(f.townInstall())
	partyCrossingState(w, 1, 40, f)

	// 2: arrival with a pack holding units with and without a bag form.
	f, screen := partyCrossingFront(t, nil)
	partyCrossingPack(t, f, 0, partyCrossingUnits(f))
	partyCrossingGraph(t, f)
	partyCrossingState(w, 2, 0, f)
	if err := screen.sess.partyItems(screen.in.townInstall().table).arriveTown(); err != nil {
		t.Fatal(err)
	}
	partyCrossingState(w, 2, 20, f)

	// 3: arrival over the unit bound: the bag stops at its bound.
	f, screen = partyCrossingFront(t, slices.Repeat([]sim.ItemInstance{secondBagPotion}, maxSecondBagUnits-1))
	partyCrossingPack(t, f, 0, partyCrossingUnits(f))
	partyCrossingGraph(t, f)
	if err := screen.sess.partyItems(screen.in.townInstall().table).arriveTown(); err != nil {
		t.Fatal(err)
	}
	partyCrossingState(w, 3, 0, f)

	// 4: an arrival whose graph cannot follow moves nothing and answers
	// with its error, by hash.
	f, screen = partyCrossingFront(t, []sim.ItemInstance{secondBagWeapon})
	partyCrossingPack(t, f, 0, []sim.ItemInstance{secondBagPotion})
	f.Town.cityObjects = &cityObjectTopology{}
	err := screen.sess.partyItems(screen.in.townInstall().table).arriveTown()
	if err == nil {
		t.Fatal("the arrival accepted a broken graph")
	}
	w.line(4, 0, err.Error())
	partyCrossingState(w, 4, 1, f)

	// 5: a town LOAD; a companion keeps his pack, the hero's units without a
	// bag form stay, a hero with none has his pack emptied, Carry form too.
	f, _ = partyCrossingFront(t, nil)
	partyCrossingPack(t, f, 0, partyCrossingUnits(f))
	party := mapload.CloneParty(f.Carried)
	if err := partyCrossingTakeLoadedPack(party, f.Table); err != nil {
		t.Fatal(err)
	}
	for k := range party {
		w.line(5, k, mapload.MemberCarriedItems(party[k], f.Table))
	}
	party = mapload.CloneParty(f.Carried)
	party[0].CarriedItems, party[0].Carried = nil, nil
	party[0].Carry = &mapload.Carry{Items: []uint16{secondBagPotion.Code}, ItemInstances: []sim.ItemInstance{secondBagPotion}, OrderedStacks: []sim.ItemStack{sim.StackItem(secondBagPotion, 1)}}
	mapload.SetCarriedFromCarry(&party[0])
	if err := partyCrossingTakeLoadedPack(party, f.Table); err != nil {
		t.Fatal(err)
	}
	w.line(5, 8, party[0].Carry.ItemInstances)
	w.line(5, 9, mapload.MemberCarriedItems(party[0], f.Table))
	chars := []sav.Character{{Items: []sav.Piece{{Code: 0x0205, Stack: 2}}}, {Hero: true, Items: []sav.Piece{{Code: 0x0205, Price: 40, Stack: 3}, {Code: 0x0e4d, Price: -1}}}}
	bag, err := secondHeroBag(chars)
	if err != nil {
		t.Fatal(err)
	}
	w.line(5, 10, bag)

	// 6: the chosen member's pack as the shop reads and writes it.
	f, _ = partyCrossingFront(t, nil)
	partyCrossingPack(t, f, 1, partyCrossingUnits(f))
	partyCrossingPack(t, f, 0, []sim.ItemInstance{secondBagWeapon})
	partyCrossingGraph(t, f)
	shown := partyCrossingShown(f, 1)
	w.line(6, 0, shown)
	w.line(6, 1, partyCrossingShown(f, 0))
	w.line(6, 2, partyCrossingShown(f, 5))
	if !partyCrossingWriteShown(f, 1, []sim.ItemInstance{shown[1], secondBagArmour}, true) {
		t.Fatal("the shop's pack write was refused")
	}
	partyCrossingState(w, 6, 3, f)
	if partyCrossingWriteShown(f, 5, nil, true) {
		t.Fatal("a write for no member was accepted")
	}
	partyCrossingState(w, 6, 23, f)

	// 7: a table settle: the axe back to the hero's bag; then the undo's own
	// way when the page has no deal.
	s, c := secondDollShop(t, secondDollAxe)
	s.click(ui.ShopControlPackCell, s.packCell(0))
	w.line(7, 0, c.shop.Table())
	s.settleSecondTable()
	w.line(7, 1, c.shop.Table())
	w.line(7, 2, *c.heroBag())
	s, c = secondDollShop(t, secondDollAxe)
	s.click(ui.ShopControlPackCell, s.packCell(0))
	s.row.dealOf = nil
	s.settleSecondTable()
	w.line(7, 3, c.shop.Table())
	w.line(7, 4, *c.heroBag())

	// 8: a mission Shop commit: a purchase changes the world's pack and
	// purse, and closing the page leaves them.
	m := missionShopFront(t, 35)
	shop := missionShopOpened(t, m)
	missionShopBuy(m.townUI)
	items, _ := m.live.world.CarriedItems(shop.hero)
	e, _ := m.live.world.Entity(shop.hero)
	w.line(8, 0, items)
	w.line(8, 1, m.live.world.Purse(uint32(e.Owner)))
	m.townUI.closeMissionShop()
	items, _ = m.live.world.CarriedItems(shop.hero)
	w.line(8, 2, items)
	w.line(8, 3, m.live.world.Purse(uint32(e.Owner)))

	// 9: a SAVE of the bag: the whole file's hash and length.
	raw := secondTownSave(t)
	sum := sha256.Sum256(raw)
	w.line(9, 0, binary.BigEndian.Uint64(sum[:8]))
	w.line(9, 1, len(raw))

	// 10: the first game's own routes: a town without the campaign state
	// has no bag, and its shop writes a member's pack as the units are.
	first := &townScreen{shopController: shopController{sess: &CampaignSession{Town: NewTown(Campaign{}),
		Carried: []mapload.PartyMember{{ID: "hero"}}}, in: &InstallResources{Table: shopTable()}}}
	first.sess.leftLocation(townInstall{table: first.in.Table})
	if err := first.sess.partyItems(first.in.Table).arriveTown(); err != nil {
		t.Fatal(err)
	}
	w.line(10, 0, first.sess.Carried[0].CarriedItems)
	first.row = shopBinding{bagUnit: sim.ItemInstance.Clone}
	if !first.setShopPackItemInstancesRefresh([]sim.ItemInstance{secondBagWeapon, secondBagPotion}, true) {
		t.Fatal("the first game's pack write was refused")
	}
	w.line(10, 1, first.sess.Carried[0].CarriedItems)
	first.sess.Carried[0].Carry = &mapload.Carry{}
	if !first.setShopPackItemInstancesRefresh([]sim.ItemInstance{secondBagPotion}, true) {
		t.Fatal("the first game's carried pack write was refused")
	}
	w.line(10, 2, first.sess.Carried[0].Carry.ItemInstances)
	w.line(10, 3, first.sess.Carried[0].Carried)
	return w.b.String()
}

// partyCrossingTakeLoadedPack, partyCrossingShown and partyCrossingWriteShown
// are the crossings this record calls by name; each is one line, so a move of
// its builder changes only the line.
func partyCrossingTakeLoadedPack(party []mapload.PartyMember, table *mapload.Table) error {
	return secondPartyItems.takeLoadedPack(party, table)
}

func partyCrossingShown(f *FrontEnd, member int) []sim.ItemInstance {
	return f.CampaignSession.partyItems(f.Table).shopView(member)
}

func partyCrossingWriteShown(f *FrontEnd, member int, items []sim.ItemInstance, refresh bool) bool {
	return f.CampaignSession.partyItems(f.Table).writeShopView(member, items, refresh)
}

// TestPartyItemCrossingsRecord states what each crossing of the party's
// items produces; a change to any unit, a graph node or a SAV byte fails it.
func TestPartyItemCrossingsRecord(t *testing.T) {
	got := partyCrossings(t)
	if os.Getenv("AGAINROM_WRITE_PARTY_ITEM_RECORD") != "" {
		if err := os.WriteFile(filepath.FromSlash(partyCrossingRecordFile), []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(filepath.FromSlash(partyCrossingRecordFile))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("the crossings changed from %s:\n%s", partyCrossingRecordFile, partyCrossingFirstDiff(got, string(want)))
	}
}

func partyCrossingFirstDiff(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := range min(len(g), len(w)) {
		if g[i] != w[i] {
			return fmt.Sprintf("line %d:\n got %.300s\nwant %.300s", i+1, g[i], w[i])
		}
	}
	return fmt.Sprintf("lines: got %d, want %d", len(g), len(w))
}
