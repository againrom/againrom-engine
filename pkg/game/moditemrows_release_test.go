package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The fixture mods of the material and weapon-row tests.
const (
	bonePikeMod   = "bone-pike"
	noIronPikeMod = "no-iron-pike"
	warGlaiveMod  = "war-glaive"
)

// modRowsFront is a front end on the lawful install under the mods ids of the
// folder dir, started as the launcher starts it: the mod set, then every data
// kind. accept is the launcher's -mods-accept-unmarked. With no id it runs
// under no mod.
func modRowsFront(t *testing.T, dir string, accept bool, ids ...string) *FrontEnd {
	t.Helper()
	f, err := modRowsFrontErr(t, dir, accept, ids...)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// modRowsFrontErr is modRowsFront answering the first refusal.
func modRowsFrontErr(t *testing.T, dir string, accept bool, ids ...string) (*FrontEnd, error) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if len(ids) == 0 {
		return f, nil
	}
	entries, err := mod.Resolve(dir, ids)
	if err != nil {
		return nil, err
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), nil, modrt.Options{})
	if err != nil {
		return nil, err
	}
	if err := f.SetMods(res.Rules, res.Set, accept); err != nil {
		return nil, err
	}
	if _, err := f.SetModData(res.Data); err != nil {
		return nil, err
	}
	return f, nil
}

// modRowsReload is a cold LOAD of a town SAV in a fresh front end under ids.
func modRowsReload(t *testing.T, raw []byte, accept bool, ids ...string) (*FrontEnd, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := modRowsFront(t, modItemsDir, accept, ids...)
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	_, town, err := load("town.sav")
	if err == nil && !town {
		t.Fatal("the town SAV did not open a town")
	}
	return g, err
}

// weaponRowNamed is the row of the install's weapon table named name.
func weaponRowNamed(t *testing.T, tab *mapload.Table, name string) int {
	t.Helper()
	for row := 1; row < tab.Weapons.Len(); row++ {
		if tab.Weapons.EntryName(row) == name {
			return row
		}
	}
	t.Fatalf("the weapon table names no %q", name)
	return 0
}

// weaponPairCode is the code of the weapon of row in the named tier and
// material.
func weaponPairCode(t *testing.T, tab *mapload.Table, row int, tier, material string) uint16 {
	t.Helper()
	shape := scaleIndex(tab.Shapes, tier, data.ShopShapes)
	m := scaleIndex(tab.Materials, material, data.ShopMaterials)
	if shape < 0 || m < 0 {
		t.Fatalf("no tier %q or material %q", tier, material)
	}
	return uint16(data.ComposeItemCode(m, data.WeaponShopClass(nil), shape, row))
}

// shelfHolds reports whether the shelf of s lists code.
func shelfHolds(s *Shop, shelf ShopShelf, code uint16) bool {
	return slices.ContainsFunc(s.Shelf(shelf), func(it ShopItem) bool { return uint16(it.Code) == code })
}

// rollUntilStocked draws the town shop's stock seed after seed, through the
// one stock generator, until the weapons shelf lists code.
func rollUntilStocked(t *testing.T, f *FrontEnd, code uint16) {
	t.Helper()
	for seed := int64(1); seed <= 4096; seed++ {
		f.Shop.Generate(f.Table, seed)
		if shelfHolds(f.Shop, ShelfWeapons, code) {
			return
		}
	}
	t.Fatalf("no seed of 4096 stocks %#x at the town's ceiling %d", code, f.Shop.ceiling)
}

// wantShownName is the install's byte spelling of the EN or RU name.
func wantShownName(t *testing.T, f *FrontEnd, en, ru string) string {
	t.Helper()
	want := map[string]string{"rom1-en": en, "rom1-ru": ru}[f.ModSet().Base]
	if want == "" {
		t.Fatalf("base %q", f.ModSet().Base)
	}
	s, err := encodeSaveLabel(want, f.textSelector())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// buyAndWield buys code in the town, wears it and checks the weapon slot.
func buyAndWield(t *testing.T, f *FrontEnd, code uint16) {
	t.Helper()
	f.Town.gold = 1_000_000
	s := currentTownShop(f, "hero")
	currentTownBuy(t, s, roomWeapons, code)
	if !modItemPackHolds(f, code) {
		t.Fatalf("the pack holds no %#x after the purchase", code)
	}
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, code)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if worn, _ := currentTownMember(t, f, "hero"); worn[0] != code {
		t.Fatalf("the hero holds %#x after wearing %#x", worn[0], code)
	}
}

// rewield takes the weapon off and puts it back on: the next action after a
// LOAD.
func rewield(t *testing.T, f *FrontEnd, code uint16) {
	t.Helper()
	s := currentTownShop(f, "hero")
	if action := s.ShopClick(ui.ShopControl{Kind: ui.ShopControlDoll, Index: 0}); action.Msg != "off, into the pack" {
		t.Fatalf("taking the weapon off: %q", action.Msg)
	}
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, code)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if worn, _ := currentTownMember(t, f, "hero"); worn[0] != code {
		t.Fatalf("the hero holds %#x after wearing %#x again", worn[0], code)
	}
}

// TestReleaseModMaterialAllowedPairRollsShowsAndIsWorn: the bone-pike fixture
// allows the common bone pike, a pair the shipped mask lacks and the install
// has no picture of. The pair joins the candidate pool, rolls in the town
// shop, shows the mod's picture and name, is bought and worn, and survives a
// SAVE and cold LOAD under the mod; a LOAD without the mod refuses.
func TestReleaseModMaterialAllowedPairRollsShowsAndIsWorn(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	plain := releaseFront(t)
	row := weaponRowNamed(t, plain.Table, "Pike")
	code := weaponPairCode(t, plain.Table, row, "Common", "Bone")
	if slices.Contains(weaponPoolCodes(plain.Table), data.ItemCode(code)) {
		t.Fatal("the install already sells a common bone pike")
	}
	if _, err := loadItemIcon(plain.Archives.Containers, graphicsPrefix+data.ItemIconPath(data.ItemCode(code))); err == nil {
		t.Fatal("the install already has a picture of the common bone pike")
	}

	f := modItemTownOf(t, func() *FrontEnd { return modRowsFront(t, modItemsDir, false, bonePikeMod) })
	if !slices.Contains(weaponPoolCodes(f.Table), data.ItemCode(code)) {
		t.Fatal("the mod's pair is not a shop candidate")
	}
	if got, want := len(weaponPoolCodes(f.Table)), len(weaponPoolCodes(plain.Table))+1; got != want {
		t.Fatalf("the modded pool holds %d candidates, want %d", got, want)
	}
	icon, err := loadItemIcon(f.Archives.Containers, graphicsPrefix+data.ItemIconPath(data.ItemCode(code)))
	if err != nil || icon.Bounds().Dx() != 80 || icon.Bounds().Dy() != 80 {
		t.Fatalf("icon %v", err)
	}
	if px := icon.RGBAAt(40, 40); px.A != 255 || px.B < 150 || px.R > 100 {
		t.Fatalf("the icon's centre %+v is not the mod's picture", px)
	}
	if got, want := itemName(data.ItemCode(code), f.Table), wantShownName(t, f, "Bone Pike", "Костяная пика"); got != want {
		t.Fatalf("the pair is named %q, want %q", got, want)
	}
	if w, err := data.WeaponFromCode(data.ItemCode(code), f.Table.Shapes, f.Table.Materials, f.Table.Weapons); err != nil || int(w.Row) != row {
		t.Fatalf("the pair resolves to %+v: %v", w, err)
	}

	rollUntilStocked(t, f, code)
	buyAndWield(t, f, code)
	raw := currentTownSave(t, f)
	g, err := modRowsReload(t, raw, false, bonePikeMod)
	if err != nil {
		t.Fatal(err)
	}
	if worn, _ := currentTownMember(t, g, "hero"); worn[0] != code {
		t.Fatalf("after the LOAD the hero holds %#x", worn[0])
	}
	rewield(t, g, code)
	if _, err := modRowsReload(t, raw, false); err == nil || !errors.Is(err, ErrModMark) || !strings.Contains(err.Error(), bonePikeMod) {
		t.Fatalf("a LOAD without the mod: %v", err)
	}
}

// TestReleaseModMaterialForbiddenPairNeverRolls: the no-iron-pike fixture
// forbids the common iron pike. Over 256 seeds at two ceilings the shipped
// game stocks it and the modded game never does; every other candidate stays.
func TestReleaseModMaterialForbiddenPairNeverRolls(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	plain := releaseFront(t)
	f := modRowsFront(t, modItemsDir, false, noIronPikeMod)
	row := weaponRowNamed(t, plain.Table, "Pike")
	code := weaponPairCode(t, plain.Table, row, "Common", "Iron")
	shipped, modded := weaponPoolCodes(plain.Table), weaponPoolCodes(f.Table)
	if !slices.Contains(shipped, data.ItemCode(code)) {
		t.Fatal("the install does not sell a common iron pike")
	}
	if want := slices.DeleteFunc(slices.Clone(shipped), func(c data.ItemCode) bool { return c == data.ItemCode(code) }); !slices.Equal(modded, want) {
		t.Fatalf("the modded pool %v, want the shipped one less %#x", modded, code)
	}
	for _, ceiling := range []int32{2000, 1 << 30} {
		stocked, rolled := 0, 0
		for seed := int64(1); seed <= 256; seed++ {
			a, b := NewShop(ceiling), NewShop(ceiling)
			a.Generate(plain.Table, seed)
			b.Generate(f.Table, seed)
			for _, shelf := range []ShopShelf{ShelfWeapons, ShelfArmour, ShelfMagic, ShelfBooks} {
				if shelfHolds(a, shelf, code) {
					stocked++
				}
				if shelfHolds(b, shelf, code) {
					rolled++
				}
			}
		}
		if stocked == 0 || rolled != 0 {
			t.Fatalf("ceiling %d: the shipped shops stock the pike %d times, the modded %d", ceiling, stocked, rolled)
		}
	}
}

// glaiveFixture is the war-glaive row and its base, the Pike, on f and on the
// unmodded plain.
type glaiveFixture struct {
	row, pikeRow int
	code, pike   uint16
}

func glaiveOf(t *testing.T, f, plain *FrontEnd) glaiveFixture {
	t.Helper()
	g := glaiveFixture{row: plain.Table.Weapons.Len(), pikeRow: weaponRowNamed(t, plain.Table, "Pike")}
	if f.Table.Weapons.Len() != g.row+1 || f.Table.Weapons.EntryName(g.row) != "war_glaive" {
		t.Fatalf("the modded weapon table holds %d rows, row %d named %q", f.Table.Weapons.Len(), g.row, f.Table.Weapons.EntryName(g.row))
	}
	g.code = weaponPairCode(t, f.Table, g.row, "Common", "Iron")
	g.pike = weaponPairCode(t, f.Table, g.pikeRow, "Common", "Iron")
	return g
}

// TestReleaseModWeaponRowCopiesItsBaseWithTheStatedNumbers: the war-glaive
// fixture adds a weapon row from the Pike. The glaive has the damage, range
// and price the mod states and every other number of the Pike, the Pike's
// pairs in the shop, its pictures and its body, and the mod's name.
func TestReleaseModWeaponRowCopiesItsBaseWithTheStatedNumbers(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	plain := releaseFront(t)
	f := modRowsFront(t, modItemsDir, false, warGlaiveMod)
	g := glaiveOf(t, f, plain)
	tab := f.Table
	glaive, err := data.WeaponFromCode(data.ItemCode(g.code), tab.Shapes, tab.Materials, tab.Weapons)
	if err != nil {
		t.Fatal(err)
	}
	pike, err := data.WeaponFromCode(data.ItemCode(g.pike), plain.Table.Shapes, plain.Table.Materials, plain.Table.Weapons)
	if err != nil {
		t.Fatal(err)
	}
	// The stated 40-80 scales by tier and material as the Pike's own 28-56
	// does: the same pair of a Pike row whose parameters 6 and 7 read 40 and 80.
	stated := slices.Clone(plain.Table.Weapons.EntryParams(g.pikeRow))
	stated[6], stated[7] = 40, 80
	restated := data.NewRowOverlay(plain.Table.Weapons, map[int]data.OverlayRow{g.pikeRow: {Name: "Pike", Params: stated}})
	want, err := data.WeaponFromCode(data.ItemCode(g.pike), plain.Table.Shapes, plain.Table.Materials, restated)
	if err != nil {
		t.Fatal(err)
	}
	if pike.Range != 1 || want.DamageBase <= pike.DamageBase || want.DamageSpread <= pike.DamageSpread {
		t.Fatalf("the shipped pike is %+v, the restated one %+v", pike, want)
	}
	if glaive.DamageBase != want.DamageBase || glaive.DamageSpread != want.DamageSpread || glaive.Range != 3 {
		t.Fatalf("the glaive is %+v, want damage %d+%d and range 3", glaive, want.DamageBase, want.DamageSpread)
	}
	if glaive.ToHit != pike.ToHit || glaive.Defence != pike.Defence || glaive.AttackType != pike.AttackType ||
		glaive.ChargeTime != pike.ChargeTime || glaive.RelaxTime != pike.RelaxTime || glaive.Weight != pike.Weight {
		t.Fatalf("the glaive %+v differs from the pike %+v past the stated numbers", glaive, pike)
	}
	gp, pp := tab.Weapons.EntryParams(g.row), plain.Table.Weapons.EntryParams(g.pikeRow)
	for col := range pp {
		switch col {
		case 2, 6, 7, 11:
			continue
		}
		if gp[col] != pp[col] {
			t.Fatalf("parameter %d is %d, the pike's %d", col, gp[col], pp[col])
		}
	}
	if gp[2] != 900 || gp[6] != 40 || gp[7] != 80 || gp[11] != 3 {
		t.Fatalf("the glaive's row parameters %v", gp)
	}

	// The shop: the glaive's pairs are the Pike's, at the glaive's price.
	var glaivePairs, pikePairs []int
	price := map[int]int32{}
	for _, c := range shopWeaponPool(tab, 1<<30) {
		switch c.Row {
		case g.row:
			glaivePairs = append(glaivePairs, c.Shape<<8|c.Material)
			price[c.Shape<<8|c.Material] = c.Price
		case g.pikeRow:
			pikePairs = append(pikePairs, c.Shape<<8|c.Material)
		}
	}
	if len(pikePairs) == 0 || !slices.Equal(glaivePairs, pikePairs) {
		t.Fatalf("the glaive's pairs %v, the pike's %v", glaivePairs, pikePairs)
	}
	for pair, p := range price {
		if want := data.ItemPrice(900, tab.Shapes, tab.Materials, pair>>8, pair&0xff); p != want {
			t.Fatalf("pair %#x costs %d, want %d", pair, p, want)
		}
	}
	if got := len(weaponPoolCodes(tab)); got != len(weaponPoolCodes(plain.Table))+len(pikePairs) {
		t.Fatalf("the modded pool holds %d candidates", got)
	}

	// Name, picture and body.
	if got, want := itemName(data.ItemCode(g.code), tab), wantShownName(t, f, "War Glaive", "Боевая глефа"); got != want {
		t.Fatalf("the glaive is named %q, want %q", got, want)
	}
	read := func(fe *FrontEnd, addr string) []byte {
		b, err := fe.Archives.Containers.ReadFile(addr)
		if err != nil {
			t.Fatalf("%s: %v", addr, err)
		}
		return b
	}
	if !bytes.Equal(read(f, graphicsPrefix+data.ItemIconPath(data.ItemCode(g.code))), read(plain, graphicsPrefix+data.ItemIconPath(data.ItemCode(g.pike)))) {
		t.Fatal("the glaive's icon is not the pike's")
	}
	for _, dir := range []data.FigureDir{data.FigureDirManFighter, data.FigureDirWomanFighter} {
		if !bytes.Equal(read(f, graphicsPrefix+data.ItemFigureLayerPath(dir, data.ItemCode(g.code))), read(plain, graphicsPrefix+data.ItemFigureLayerPath(dir, data.ItemCode(g.pike)))) {
			t.Fatalf("the glaive's %s figure layer is not the pike's", dir)
		}
	}
	var withGlaive, withPike [sim.EquipSlots]uint16
	withGlaive[0], withPike[0] = g.code, g.pike
	gb, _, gc, gok := data.HeroAppearance(f.Bodies, equipmentFromSlots(withGlaive), false, false)
	pb, _, pc, pok := data.HeroAppearance(plain.Bodies, equipmentFromSlots(withPike), false, false)
	if !gok || !pok || gb != pb || gc != pc {
		t.Fatalf("the glaive hero is drawn with %q class %d, the pike hero with %q class %d", gb, gc, pb, pc)
	}
}

// reachStrike places the first other living person of the mission cells
// cells east (or west) of the hero and orders one attack. It answers the
// tick of the first blow that lands, the hit points it took, and whether the
// hero had left his cell by then.
func reachStrike(t *testing.T, mw *mapWorld, hero sim.EntityID, cells int32) (tick int, lost int32, moved bool) {
	t.Helper()
	var victim sim.EntityID
	found := false
	for _, e := range mw.world.Entities() {
		if e.ID != hero && e.Owner != sim.SelfSlot && e.Alive() && !e.OffMap {
			victim, found = e.ID, true
			break
		}
	}
	if !found {
		t.Fatal("the mission holds no other person to strike")
	}
	hv, _ := mw.entity(hero)
	placed := false
	for _, dx := range []int32{cells, -cells} {
		if mw.world.HeadlessPlace(victim, hv.X+dx, hv.Y) == nil {
			placed = true
			break
		}
	}
	if !placed {
		t.Fatalf("no free cell %d cells from the hero", cells)
	}
	before, _ := mw.entity(victim)
	mw.strike(uint32(hero), uint32(victim))
	for tick = 1; tick <= 400; tick++ {
		mw.tick()
		v, _ := mw.entity(victim)
		h, _ := mw.entity(hero)
		if h.X != hv.X || h.Y != hv.Y {
			moved = true
		}
		if v.HP < before.HP {
			return tick, before.HP - v.HP, moved
		}
	}
	t.Fatalf("no blow landed in 400 ticks")
	return 0, 0, moved
}

// TestReleaseModWeaponRowStrikesAtItsRangeAndSavesInAMission: a hero who
// equips the glaive in mission 10 has the Pike hero's combat numbers but for
// the stated damage and reach; he strikes a person three cells away without
// stepping, which the Pike hero cannot. A mission SAVE with the glaive held,
// a cold LOAD under the mod and the next strike keep the row; the SAV holds
// the Pike as the stand-in and the mark the glaive; a LOAD without the mod
// refuses.
func TestReleaseModWeaponRowStrikesAtItsRangeAndSavesInAMission(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	plain := releaseFront(t)
	f := modRowsFront(t, modItemsDir, false, warGlaiveMod)
	g := glaiveOf(t, f, plain)

	mw, hero := heroSwingMission(t, f, g.code)
	pw, phero := heroSwingMission(t, plain, g.pike)
	// The derived combat numbers follow the equipment on the next frames.
	for range 3 {
		mw.tick()
		pw.tick()
	}
	ge, _ := mw.entity(hero)
	pe, _ := pw.entity(phero)
	glaive, _ := data.WeaponFromCode(data.ItemCode(g.code), f.Table.Shapes, f.Table.Materials, f.Table.Weapons)
	pike, _ := data.WeaponFromCode(data.ItemCode(g.pike), plain.Table.Shapes, plain.Table.Materials, plain.Table.Weapons)
	if ge.Reach != 3 || pe.Reach != 1 || ge.DamageBase-pe.DamageBase != glaive.DamageBase-pike.DamageBase ||
		ge.DamageSpread-pe.DamageSpread != glaive.DamageSpread-pike.DamageSpread || glaive.DamageBase == pike.DamageBase ||
		ge.ToHit != pe.ToHit || ge.Defence != pe.Defence {
		t.Fatalf("the glaive hero: reach %d base %d spread %d to-hit %d defence %d; the pike hero: reach %d base %d spread %d to-hit %d defence %d",
			ge.Reach, ge.DamageBase, ge.DamageSpread, ge.ToHit, ge.Defence, pe.Reach, pe.DamageBase, pe.DamageSpread, pe.ToHit, pe.Defence)
	}
	// Drawn with the Pike's body.
	var withPike [sim.EquipSlots]uint16
	withPike[0] = g.pike
	body, _, _, ok := data.HeroAppearance(plain.Bodies, equipmentFromSlots(withPike), false, false)
	if !ok {
		t.Fatal("the pike hero resolves no body")
	}
	mw.refreshAppearance()
	checkModBodyArt(t, mw, hero, body)
	if _, lost, moved := reachStrike(t, mw, hero, 3); moved || lost < 1 || lost > ge.DamageBase+ge.DamageSpread {
		t.Fatalf("the glaive blow at three cells: moved %v, %d hit points", moved, lost)
	}
	if _, _, moved := reachStrike(t, pw, phero, 3); !moved {
		t.Fatal("the pike hero struck three cells away without stepping")
	}

	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE wrote %q: %v", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	mark, present, err := readModMark(doc)
	if err != nil || !present {
		t.Fatalf("the save's mark: %v %v", present, err)
	}
	held := 0
	for _, it := range mark.Items {
		if it.Code != g.code {
			continue
		}
		held++
		obj := &doc.Objects[it.Object]
		code, _ := savedStructureValue(obj, "F40")
		row, _ := savedStructureValue(obj, "T0C")
		if uint16(code) != g.pike || int(row) != g.pikeRow || int(it.Row) != g.row {
			t.Fatalf("the glaive object holds code %#x row %d, the mark row %d", code, row, it.Row)
		}
	}
	if held == 0 {
		t.Fatalf("the mark names no glaive: %+v", mark.Items)
	}

	h := modRowsFront(t, modItemsDir, false, warGlaiveMod)
	_, _, load := h.SaveSeams(store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("LOAD of the mission SAVE: town=%v err=%v", town, err)
	}
	app := h.App("glaive loaded")
	t.Cleanup(app.StopAudio)
	app.SetCutscenes(nil)
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	lhero := equipmentReturnHero(t, h)
	if worn, _ := h.live.world.EquippedItems(lhero); worn[0].Code != g.code {
		t.Fatalf("after the LOAD the hero holds %#x", worn[0].Code)
	}
	le, _ := h.live.entity(lhero)
	if le.Reach != 3 || le.DamageBase != ge.DamageBase || le.DamageSpread != ge.DamageSpread {
		t.Fatalf("after the LOAD: reach %d base %d spread %d", le.Reach, le.DamageBase, le.DamageSpread)
	}
	if _, lost, moved := reachStrike(t, h.live, lhero, 3); moved || lost < 1 || lost > le.DamageBase+le.DamageSpread {
		t.Fatalf("the next blow after the LOAD: moved %v, %d hit points", moved, lost)
	}

	p := releaseFront(t)
	_, _, pload := p.SaveSeams(store, OriginalStore{}, nil)
	if _, _, err := pload(localOriginalSaveToken(name)); err == nil || !errors.Is(err, ErrModMark) || !strings.Contains(err.Error(), warGlaiveMod) {
		t.Fatalf("a LOAD without the mod: %v", err)
	}
}

// TestReleaseModWeaponRowIsSoldWornAndSavedInTheTown: the glaive rolls in the
// town shop, is bought and worn, and a town SAVE, a cold LOAD under the mod
// and the next action keep it worn.
func TestReleaseModWeaponRowIsSoldWornAndSavedInTheTown(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	plain := releaseFront(t)
	f := modItemTownOf(t, func() *FrontEnd { return modRowsFront(t, modItemsDir, false, warGlaiveMod) })
	g := glaiveOf(t, f, plain)
	rollUntilStocked(t, f, g.code)
	buyAndWield(t, f, g.code)
	raw := currentTownSave(t, f)
	h, err := modRowsReload(t, raw, false, warGlaiveMod)
	if err != nil {
		t.Fatal(err)
	}
	if worn, _ := currentTownMember(t, h, "hero"); worn[0] != g.code {
		t.Fatalf("after the LOAD the hero holds %#x", worn[0])
	}
	rewield(t, h, g.code)
	if _, err := modRowsReload(t, raw, false); err == nil || !errors.Is(err, ErrModMark) || !strings.Contains(err.Error(), warGlaiveMod) {
		t.Fatalf("a LOAD without the mod: %v", err)
	}
}

// oldTownSave is a town SAV written with no mod. AGAINROM_OLD_TOWN_SAV names
// one written by an earlier engine; otherwise this tree writes it, which the
// base-unchanged comparison shows is byte for byte what main writes.
func oldTownSave(t *testing.T) []byte {
	t.Helper()
	if path := os.Getenv("AGAINROM_OLD_TOWN_SAV"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	return currentTownSave(t, currentTown(t, nil, nil))
}

// TestReleaseModItemRowsLoadAnOldSave: an unmarked town SAV loads with no
// mod and, as for every mod, under the fixture mods when the launcher accepts
// unmarked saves; the next action, buying the mod's weapon, works.
func TestReleaseModItemRowsLoadAnOldSave(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	raw := oldTownSave(t)
	if sav.MayCarryNativeMods(raw) {
		t.Fatal("the old save carries a mod leaf")
	}
	plain, err := modRowsReload(t, raw, false)
	if err != nil {
		t.Fatal(err)
	}
	if worn, _ := currentTownMember(t, plain, "hero"); worn[0] == 0 {
		t.Fatal("the old save's hero holds no weapon")
	}
	if _, err := modRowsReload(t, raw, false, warGlaiveMod); err == nil || !errors.Is(err, ErrModMark) {
		t.Fatalf("an unmarked save under a mod without -mods-accept-unmarked: %v", err)
	}
	for _, ids := range [][]string{{warGlaiveMod}, {bonePikeMod}, {noIronPikeMod}} {
		g, err := modRowsReload(t, raw, true, ids...)
		if err != nil {
			t.Fatalf("%v: %v", ids, err)
		}
		pw, _ := currentTownMember(t, plain, "hero")
		if worn, _ := currentTownMember(t, g, "hero"); worn != pw || g.Town.Gold() != plain.Town.Gold() {
			t.Fatalf("%v: the hero holds %v gold %d, without the mod %v gold %d", ids, worn, g.Town.Gold(), pw, plain.Town.Gold())
		}
		if ids[0] == warGlaiveMod {
			code := glaiveOf(t, g, plain).code
			rollUntilStocked(t, g, code)
			buyAndWield(t, g, code)
		}
	}
}

// itemRowsMod writes a mod folder id under dir holding the files given by
// their path inside it.
func itemRowsMod(t *testing.T, dir, id string, files map[string]string) {
	t.Helper()
	files["mod.toml"] = "id = \"" + id + "\"\ntitle = \"T\"\nversion = \"1.0.0\"\nauthor = \"A\"\ndescription = \"D\"\napi = 1\n" +
		"applies-to = [\"rom1\"]\nentry = \"main.star\"\n" + files["mod.toml"]
	if files["main.star"] == "" {
		files["main.star"] = "def init(game, settings):\n    game.data.add(\"data/items.toml\")\n"
	}
	for rel, body := range files {
		path := filepath.Join(dir, id, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestReleaseModItemRowsRefusals: on the install, each refusal names the mod,
// the file and the line, and with load-after two mods on one mask bit load,
// the later winning.
func TestReleaseModItemRowsRefusals(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	text := func(key, s string) map[string]string {
		return map[string]string{"text/en/strings.toml": key + " = \"" + s + "\"\n", "text/ru/strings.toml": key + " = \"" + s + "\"\n"}
	}
	with := func(m map[string]string, rel, body string) map[string]string {
		m[rel] = body
		return m
	}
	dir := t.TempDir()
	itemRowsMod(t, dir, "no-picture", with(text("n", "N"), "data/items.toml",
		"[[material]]\nitem = \"Pike\"\ntier = \"Common\"\nmaterial = \"Steel\"\nallow = true\n"))
	itemRowsMod(t, dir, "bad-base", with(text("n", "N"), "data/items.toml",
		"[[weapon]]\nkey = \"w\"\nname = \"n\"\nbase = \"Spear\"\n"))
	itemRowsMod(t, dir, "bad-param", with(text("n", "N"), "data/items.toml",
		"[[weapon]]\nkey = \"w\"\nname = \"n\"\nbase = \"Pike\"\nattack-type = 5\n"))
	itemRowsMod(t, dir, "glaive-a", with(text("n", "N"), "data/items.toml",
		"[[weapon]]\nkey = \"glaive\"\nname = \"n\"\nbase = \"Pike\"\n"))
	itemRowsMod(t, dir, "glaive-b", with(text("n", "N"), "data/items.toml",
		"[[weapon]]\nkey = \"glaive\"\nname = \"n\"\nbase = \"Pike\"\nrange = 2\n"))
	itemRowsMod(t, dir, "uncommon-steel-off", map[string]string{"data/items.toml": "[[material]]\nitem = \"Pike\"\ntier = \"Uncommon\"\nmaterial = \"Steel\"\nallow = false\n"})
	itemRowsMod(t, dir, "uncommon-steel-on", map[string]string{"data/items.toml": "[[material]]\nitem = \"Pike\"\ntier = \"Uncommon\"\nmaterial = \"Steel\"\nallow = true\n"})
	itemRowsMod(t, dir, "uncommon-steel-on-after", map[string]string{"mod.toml": "load-after = [\"uncommon-steel-off\"]\n", "data/items.toml": "[[material]]\nitem = \"Pike\"\ntier = \"Uncommon\"\nmaterial = \"Steel\"\nallow = true\n"})

	for _, c := range []struct {
		ids  []string
		want []string
	}{
		{[]string{"no-picture"}, []string{"no-picture", "data/items.toml:1", `weapon row "Pike" tier "Common" material "Steel" has no inventory picture`}},
		{[]string{"bad-base"}, []string{"bad-base", "data/items.toml:4", `the weapon table has no row named "Spear"`}},
		{[]string{"bad-param"}, []string{"bad-param", "data/items.toml:5", `"attack-type" is not a weapon parameter a mod may set`}},
		{[]string{"glaive-a", "glaive-b"}, []string{"glaive-a", "glaive-b", "glaive"}},
		{[]string{"uncommon-steel-off", "uncommon-steel-on"}, []string{"uncommon-steel-off", "uncommon-steel-on", "Uncommon Steel"}},
	} {
		_, err := modRowsFrontErr(t, dir, false, c.ids...)
		if err == nil {
			t.Fatalf("%v loaded", c.ids)
		}
		for _, w := range c.want {
			if !strings.Contains(err.Error(), w) {
				t.Fatalf("%v: %v; want %q in it", c.ids, err, w)
			}
		}
	}

	f := modRowsFront(t, dir, false, "uncommon-steel-off", "uncommon-steel-on-after")
	plain := releaseFront(t)
	code := weaponPairCode(t, f.Table, weaponRowNamed(t, f.Table, "Pike"), "Uncommon", "Steel")
	if !slices.Contains(weaponPoolCodes(plain.Table), data.ItemCode(code)) || !slices.Equal(weaponPoolCodes(f.Table), weaponPoolCodes(plain.Table)) {
		t.Fatal("with load-after the later mod's allow does not stand")
	}
}

// TestReleaseModWeaponRowTakesABodyChoice: a weapon-bodies record names the
// added row by its key, and the glaive hero is drawn with the chosen body
// while his class key stays the Pike hero's.
func TestReleaseModWeaponRowTakesABodyChoice(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := t.TempDir()
	itemRowsMod(t, dir, "glaive-club", map[string]string{
		"main.star":            "def init(game, settings):\n    game.data.add(\"data/items.toml\")\n    game.data.add(\"data/wpnbodies.toml\")\n",
		"text/en/strings.toml": "n = \"N\"\n", "text/ru/strings.toml": "n = \"N\"\n",
		"data/items.toml":     "[[weapon]]\nkey = \"glaive\"\nname = \"n\"\nbase = \"Pike\"\n",
		"data/wpnbodies.toml": "[[weapon]]\nweapon = \"glaive\"\nbody = \"clubman\"\n",
	})
	plain := releaseFront(t)
	f := modRowsFront(t, dir, false, "glaive-club")
	row := plain.Table.Weapons.Len()
	var withGlaive, withPike [sim.EquipSlots]uint16
	withGlaive[0] = weaponPairCode(t, f.Table, row, "Common", "Iron")
	withPike[0] = weaponPairCode(t, f.Table, weaponRowNamed(t, f.Table, "Pike"), "Common", "Iron")
	gb, _, gc, gok := data.HeroAppearance(f.Bodies, equipmentFromSlots(withGlaive), false, false)
	pb, _, pc, pok := data.HeroAppearance(f.Bodies, equipmentFromSlots(withPike), false, false)
	if !gok || !pok || gb != "clubman" || pb == "clubman" || gc != pc {
		t.Fatalf("the glaive hero is drawn with %q class %d, the pike hero with %q class %d", gb, gc, pb, pc)
	}
}
