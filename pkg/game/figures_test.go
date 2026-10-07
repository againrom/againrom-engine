package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// figureRow is a synthetic Humans row carrying only the three columns a figure
// is chosen by — type id, face, gender — every other cell empty.
//
// It is built from data.MinHumanRow rather than a literal width, for the reason
// that constant is exported: the streamed width has moved twice.
func figureRow(typeID, face, gender int32) []int32 {
	p := make([]int32, data.MinHumanRow)
	for i := range p {
		p[i] = -1
	}
	p[16], p[17], p[18] = typeID, face, gender
	return p
}

// Zero-mode figures split the constructed face byte, including its defaults.
func TestPlacedPersonFigureUsesConstructorFaceByte(t *testing.T) {
	const (
		fighter = 3
		mage    = 0x17
	)
	humans := dbCollection{
		{},
		{name: "ManFighter", params: figureRow(fighter, 8, 0)},
		{name: "WomanFighter", params: figureRow(fighter+1, 8, 1)},
		{name: "ManMage", params: figureRow(mage, 6, 0)},
		{name: "WomanMage", params: figureRow(mage+1, 5, 1)},
		// The streamed face high bit participates in the constructed byte.
		{name: "TopBitFace", params: figureRow(fighter+2, 0x88, 0)},
		// An absent gender keeps the constructor default of 1.
		{name: "NoGenderCell", params: figureRow(fighter+3, 4, -1)},
	}
	m := &alm.Map{Units: []alm.Unit{
		{ClassID: fighter}, {ClassID: fighter + 1},
		{ClassID: mage}, {ClassID: mage + 1},
		{ClassID: fighter + 2}, {ClassID: fighter + 3},
	}}

	got := entityFigures(m, &mapload.Table{Humans: humans})
	for _, tc := range []struct {
		name string
		id   sim.EntityID
		want figureID
	}{
		{"a man fighter", 0, figureID{Dir: data.FigureDirManFighter, Face: 8}},
		{"a woman fighter — the same face, the other directory", 1, figureID{Dir: data.FigureDirWomanFighter, Face: 8}},
		{"a man mage", 2, figureID{Dir: data.FigureDirManMage, Face: 6}},
		{"a woman mage", 3, figureID{Dir: data.FigureDirWomanMage, Face: 5}},
		{"a face byte with bit 7 set draws a woman", 4, figureID{Dir: data.FigureDirWomanFighter, Face: 8}},
		{"an empty gender cell draws a woman", 5, figureID{Dir: data.FigureDirWomanFighter, Face: 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got[tc.id] != tc.want {
				t.Errorf("entity %d resolved to %+v, want %+v", tc.id, got[tc.id], tc.want)
			}
		})
	}

	// THE WHOLE POINT, stated once as a property rather than as six rows: the
	// two people who differ ONLY in the gender cell must get different
	// directories. Revert the reading and this is the line that fails.
	if got[0].Dir == got[1].Dir {
		t.Errorf("a man and a woman of one face both drew %q", got[0].Dir)
	}
}

func TestSurvivingCreatureKeepsItsPortraitIdentityAfterPlacementPrune(t *testing.T) {
	human := alm.Unit{UnitID: 13, ClassID: 3}
	original := worldFixtureMap()
	original.Units = []alm.Unit{{UnitID: 11, ClassID: 64}, {UnitID: 12, ClassID: 64}, human}
	table := &mapload.Table{Humans: dbCollection{{}, {name: "ManFighter", params: figureRow(3, 8, 0)}}}
	figure := figureID{Dir: data.FigureDirManFighter, Face: 8}
	entity := func(id sim.EntityID, mapID uint16, class int32, humanoid bool) sim.Entity {
		return sim.Entity{ID: id, X: int32(id + 2), Y: 3, HP: 10, MaxHP: 10,
			MapUnitID: mapID, Class: class, Humanoid: humanoid}
	}
	world := func(entities ...sim.Entity) *sim.World {
		t.Helper()
		w, err := sim.NewWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
			sim.ModeCanonical, nil, entities)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}

	live := openMission(&Mission{Number: 10, Map: original, World: world(
		entity(0, 11, 64, false), entity(1, 12, 64, false), entity(2, 13, 3, true)),
	}, table, nil, worldFixtureViewer(t, original), missionSource{}, nil, nil)
	if got, ok := live.figures[2]; !ok || got != figure {
		t.Fatalf("before SAVE, surviving person figure = %+v, present %t; want %+v", got, ok, figure)
	}
	if got, ok := live.figures[1]; ok {
		t.Fatalf("before SAVE, surviving creature has human figure %+v", got)
	}

	state := &SnapshotSAVDocument{Document: &sav.DocumentData{World: &sav.DocumentWorldData{}}}
	graph := sav.SavedActorGraph{CurrentPopulation: true, Actors: []sav.ActorRecord{
		{CurrentEntity: true, Actor: sav.Actor{MapUnitID: 12}},
		{CurrentEntity: true, Actor: sav.Actor{MapUnitID: 13}},
	}}
	retained := *original
	retained.Units = slices.Clone(original.Units)
	retainSavedActorPlacements(&retained, graph, nil, state)
	if !slices.Equal(retained.Units, original.Units[1:]) {
		t.Fatalf("saved population kept placements %+v, want only the two survivors", retained.Units)
	}
	loadedWorld := world(entity(1, 12, 64, false), entity(2, 13, 3, true))
	open := func(m *alm.Map) *mapWorld {
		return openMission(&Mission{Number: 10, Map: m, World: loadedWorld, savedDocument: state},
			table, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	}
	loaded := open(&retained)
	if got, ok := loaded.figures[1]; ok {
		t.Fatalf("after LOAD, surviving creature gained human figure %+v", got)
	}
	if got, ok := loaded.figures[2]; !ok || got != figure {
		t.Fatalf("after LOAD, surviving person figure = %+v, present %t; want %+v", got, ok, figure)
	}

	ambiguous := retained
	ambiguous.Units = append(slices.Clone(retained.Units), human)
	loaded = open(&ambiguous)
	if got, ok := loaded.figures[1]; ok {
		t.Fatalf("duplicate authored map ID gave creature human figure %+v", got)
	}
	if got, ok := loaded.figures[2]; ok {
		t.Fatalf("duplicate authored map ID guessed person figure %+v", got)
	}
}

// TestComposeUnitFigurePaintsInFigureDrawOrder — 0151-layers-and-names T1:
// the weapon slot (1) paints over armour slots painted earlier in
// data.FigureDrawOrder. Slot 2 (the shield) is never occupied here, so
// data.FigureHeldLast's own T7 swap has nothing to move slot 1 behind — see
// TestComposeUnitFigurePaintsWhicheverHeldSlotIsLast, below, for the case
// where it does. Mirrors composeInventorySubject's own order test
// (inventory_test.go) at this package's other composition site, so the two
// cannot silently disagree on the order (0151 T1's own brief).
func TestComposeUnitFigurePaintsInFigureDrawOrder(t *testing.T) {
	fig := figureID{Dir: data.FigureDirManFighter, Face: 1}
	codeA, codeB, codeC := invWornCode(1), invWornCode(2), invWornCode(3)
	var eq data.Equipment
	eq.SetCode(1, codeA) // the weapon
	eq.SetCode(7, codeB)
	eq.SetCode(12, codeC)

	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(fig.Dir, fig.Face): invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, codeA):   invLayerSheetColored(0, 0, 0xff),    // blue, slot 1, painted last
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, codeB):   invLayerSheetColored(0, 0xff, 0xff), // cyan, slot 7
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, codeC):   invLayerSheetColored(0xff, 0xff, 0), // yellow, slot 12
	}

	got, mask := composeUnitFigure(src, eq, fig)
	if got == nil {
		t.Fatal("composeUnitFigure returned nil with a readable base")
	}
	if px := got.RGBAAt(0, 0); px.R != 0 || px.G != 0 || px.B != 0xff {
		t.Errorf("(0,0) = %+v, want slot 1's own blue — the weapon, painted last", px)
	}

	// THE MASK NAMES THE SAME SLOT THE PIXEL'S OWN COLOUR ALREADY PROVES
	// PAINTED LAST (1005, "the interactive doll", item 1): three layers touch
	// (0,0) in this fixture — slots 12, 7 and 1, in that order — and the mask
	// is not a record of every layer that ever painted a pixel, only of the
	// topmost one still standing. (1,1) is base-only: no layer's sparse sheet
	// touches it, so the mask reports no slot at all, on the same "the base
	// claims nothing" rule composeInventorySubject's own mask test states.
	if n, ok := mask.At(0, 0); !ok || n != 1 {
		t.Errorf("mask.At(0,0) = (%d,%v), want (1,true) — slot 1 painted here last", n, ok)
	}
	if _, ok := mask.At(1, 1); ok {
		t.Error("mask.At(1,1) named a slot; no layer's sparse sheet ever touches this pixel")
	}
}

func TestComposeUnitFigurePaintsWhicheverHeldSlotIsLast(t *testing.T) {
	fig := figureID{Dir: data.FigureDirManFighter, Face: 1}
	weaponCode := data.ItemCode(2) // field D = 2, row 2 of the fixture list below
	shieldCode := invWornCode(5)
	var eq data.Equipment
	eq.SetCode(1, weaponCode)
	eq.SetCode(2, shieldCode)

	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(fig.Dir, fig.Face):    invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, weaponCode): invLayerSheetColored(0, 0, 0xff), // blue
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, shieldCode): invLayerSheetColored(0, 0xff, 0), // green
		HeroPictureAddress: []byte("unarmed\nswordsman\n"), // row 2 -> "swordsman", one-handed
	}

	got, _ := composeUnitFigure(src, eq, fig)
	if got == nil {
		t.Fatal("composeUnitFigure returned nil with a readable base")
	}
	if px := got.RGBAAt(0, 0); px.R != 0 || px.G != 0xff || px.B != 0 {
		t.Errorf("(0,0) = %+v, want the shield's own green — a one-handed body paints the shield last", px)
	}
}

func TestDollEvidenceRunsTheActualCompositorForEachHeldLayer(t *testing.T) {
	fig := figureID{Dir: data.FigureDirManFighter, Face: 1}
	weaponCode, shieldCode := data.ItemCode(2), invWornCode(5)
	slots := [sim.EquipSlots]uint16{uint16(weaponCode), uint16(shieldCode)}
	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(fig.Dir, fig.Face):    invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, weaponCode): invLayerSheetColored(0, 0, 0xff),
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, shieldCode): invLayerSheetColored(0, 0xff, 0),
		HeroPictureAddress: []byte("unarmed\nswordsman\n"),
	}
	got := composeDollEvidence(src, slots, fig)
	if !got.FullDrawn || !got.BareDrawn || !got.WeaponDrawn || !got.ShieldDrawn {
		t.Fatalf("one compositor arm did not draw: %+v", got)
	}
	if got.Weapon == got.Bare || got.Shield == got.Bare {
		t.Fatalf("a held layer left the base digest unchanged: bare=%x weapon=%x shield=%x", got.Bare, got.Weapon, got.Shield)
	}
	if got.Full == got.Bare {
		t.Fatalf("the full weapon+shield composition remained bare: %+v", got)
	}
}

// TestComposeUnitFigurePaintsTheWeaponLastForATwoHandedBody — the swap's
// other arm: a two-handed body ("swordsman2h") paints the weapon last, over
// the shield, exactly HERO-FIGURE-060's own predicate.
func TestComposeUnitFigurePaintsTheWeaponLastForATwoHandedBody(t *testing.T) {
	fig := figureID{Dir: data.FigureDirManFighter, Face: 1}
	weaponCode := data.ItemCode(2) // field D = 2, row 2 of the fixture list below
	shieldCode := invWornCode(5)
	var eq data.Equipment
	eq.SetCode(1, weaponCode)
	eq.SetCode(2, shieldCode)

	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(fig.Dir, fig.Face):    invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, weaponCode): invLayerSheetColored(0, 0, 0xff), // blue
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, shieldCode): invLayerSheetColored(0, 0xff, 0), // green
		HeroPictureAddress: []byte("unarmed\nswordsman2h\n"), // row 2 -> "swordsman2h"
	}

	got, _ := composeUnitFigure(src, eq, fig)
	if got == nil {
		t.Fatal("composeUnitFigure returned nil with a readable base")
	}
	if px := got.RGBAAt(0, 0); px.R != 0 || px.G != 0 || px.B != 0xff {
		t.Errorf("(0,0) = %+v, want the weapon's own blue — a two-handed body paints the weapon last", px)
	}
}

// TestComposeUnitFigurePaintsTheSecondSideOfAPairedSlot — 0151-layers-and-
// names T2: an occupied slot data.HasItemFigureSecondaryLayer names paints
// its "secondary" sheet too, over the "primary" one already painted for the
// same slot — the fix for gloves and shoulder pieces drawn on one side only.
func TestComposeUnitFigurePaintsTheSecondSideOfAPairedSlot(t *testing.T) {
	fig := figureID{Dir: data.FigureDirManFighter, Face: 1}
	const pairedSlot = 4 // data.HasItemFigureSecondaryLayer(4) is true.
	code := invWornCode(9)
	var eq data.Equipment
	eq.SetCode(pairedSlot, code)

	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(fig.Dir, fig.Face):       invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, code):          invLayerSheetColored(0, 0, 0xff), // blue, the primary side
		graphicsPrefix + data.ItemFigureSecondaryLayerPath(fig.Dir, code): invLayerSheetColored(0, 0xff, 0), // green, the second side
	}

	got, _ := composeUnitFigure(src, eq, fig)
	if got == nil {
		t.Fatal("composeUnitFigure returned nil with a readable base")
	}
	if px := got.RGBAAt(0, 0); px.R != 0 || px.G != 0xff || px.B != 0 {
		t.Errorf("(0,0) = %+v, want the second side's own green, painted after the primary", px)
	}
}

// TestComposeUnitFigureLeavesAnUnpairedSlotAtItsPrimaryAlone — the negative
// half of the test above: a slot data.HasItemFigureSecondaryLayer does NOT
// name attempts no second address, so its primary colour survives even
// though a "secondary" sheet exists at that address in the archive (it
// would belong to a different slot's code in a real install; this fixture
// plants it at slot 7's own address to prove the miss is a real refusal to
// look, not a coincidental absence).
func TestComposeUnitFigureLeavesAnUnpairedSlotAtItsPrimaryAlone(t *testing.T) {
	fig := figureID{Dir: data.FigureDirManFighter, Face: 1}
	const unpairedSlot = 7 // data.HasItemFigureSecondaryLayer(7) is false.
	code := invWornCode(9)
	var eq data.Equipment
	eq.SetCode(unpairedSlot, code)

	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(fig.Dir, fig.Face):       invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, code):          invLayerSheetColored(0, 0, 0xff), // blue
		graphicsPrefix + data.ItemFigureSecondaryLayerPath(fig.Dir, code): invLayerSheetColored(0, 0xff, 0), // must not be read
	}

	got, _ := composeUnitFigure(src, eq, fig)
	if got == nil {
		t.Fatal("composeUnitFigure returned nil with a readable base")
	}
	if px := got.RGBAAt(0, 0); px.R != 0 || px.G != 0 || px.B != 0xff {
		t.Errorf("(0,0) = %+v, want the primary's own blue — slot 7 carries no second side", px)
	}
}

// A fighter's weapon is painted before the slot 8 armour, so the armour covers
// it where the two overlap; a tail slot 1 paints it again over everything for
// a two-handed body.
func TestComposeUnitFigurePaintsASlotOneWeaponBeforeSlotEightArmour(t *testing.T) {
	fig := figureID{Dir: data.FigureDirManFighter, Face: 1}
	weaponCode, armourCode := data.ItemCode(2), invWornCode(5)
	var eq data.Equipment
	eq.SetCode(1, weaponCode)
	eq.SetCode(8, armourCode)
	for _, tc := range []struct {
		name  string
		list  string
		wantB uint8 // blue is the weapon, green the armour
	}{
		{"one-handed body", "unarmed\nswordsman\n", 0},
		{"two-handed body", "unarmed\nswordsman2h\n", 0xff},
	} {
		src := missionSource{
			graphicsPrefix + data.ItemFigureBasePath(fig.Dir, fig.Face):    invBaseSheet(),
			graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, weaponCode): invLayerSheetColored(0, 0, 0xff),
			graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, armourCode): invLayerSheetColored(0, 0xff, 0),
			HeroPictureAddress: []byte(tc.list),
		}
		got, mask := composeUnitFigure(src, eq, fig)
		if got == nil {
			t.Fatalf("%s: composeUnitFigure returned nil", tc.name)
		}
		px := got.RGBAAt(0, 0)
		slot, _ := mask.At(0, 0)
		wantSlot := 8
		if tc.wantB != 0 {
			wantSlot = 1
		}
		if px.B != tc.wantB || slot != wantSlot {
			t.Errorf("%s: (0,0) = %+v and slot %d, want blue %#x and slot %d", tc.name, px, slot, tc.wantB, wantSlot)
		}
	}
}

// A mage's slot 9 is tagged in the hit map and never painted, and its slot 2
// has no step at all.
func TestComposeFiguresOfAMagePaintNeitherSlotNineNorSlotTwo(t *testing.T) {
	fig := figureID{Dir: data.FigureDirManMage, Face: 1}
	slotNine, slotTwo := invWornCode(5), invWornCode(6)
	var eq data.Equipment
	eq.SetCode(9, slotNine)
	eq.SetCode(2, slotTwo)
	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(fig.Dir, fig.Face):  invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, slotNine): invLayerSheetColored(0, 0xff, 0),
		graphicsPrefix + data.ItemFigureLayerPath(fig.Dir, slotTwo):  invLayerSheetColored(0, 0, 0xff),
	}
	bare, _ := composeUnitFigure(src, data.Equipment{}, fig)
	got, mask := composeUnitFigure(src, eq, fig)
	if got == nil || bare == nil {
		t.Fatal("composeUnitFigure returned nil with a readable base")
	}
	if got.RGBAAt(0, 0) != bare.RGBAAt(0, 0) {
		t.Errorf("(0,0) = %+v, want the bare figure's %+v: neither slot is blitted for a mage", got.RGBAAt(0, 0), bare.RGBAAt(0, 0))
	}
	if n, ok := mask.At(0, 0); !ok || n != 9 {
		t.Errorf("hit map at (0,0) = (%d,%v), want slot 9 tagged and slot 2 absent", n, ok)
	}

	subject, _ := composeInventorySubject(src, 1, eq, fig.Dir, fig.Face)
	if subject.Figure == nil {
		t.Fatal("composeInventorySubject drew no figure")
	}
	if subject.Figure.RGBAAt(0, 0) != bare.RGBAAt(0, 0) {
		t.Errorf("inventory (0,0) = %+v, want the bare figure's %+v", subject.Figure.RGBAAt(0, 0), bare.RGBAAt(0, 0))
	}
	if n, ok := subject.SlotMask.At(0, 0); !ok || n != 9 {
		t.Errorf("inventory hit map at (0,0) = (%d,%v), want slot 9", n, ok)
	}
}
