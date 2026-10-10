package game

// The picture prefers the body (AC-10, plan SC-8).
//
// An INTERNAL test, because what it drives is the push itself: the seam this
// story adds is one statement inside entityDraws, and asking about it from
// outside would mean asking about a whole front end instead.
//
// Every fixture is hand-assembled render-tier data. No archive, no decode, no
// window.

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	pictureW, pictureH = 12, 12
	pictureSeed        = 0x0085
	pictureClass       = 3
	pictureBodyClass   = 4
	pictureCaption     = "Localized actor"
	pictureBodyCaption = "Localized equipment body"
	pictureMember      = sim.EntityID(0)
	pictureOther       = sim.EntityID(1)
)

// pictureBundle is one class, drawn with two frames of its OWN size, and the
// body is drawn with three of a size the class never has — so which sheet a
// draw came from is readable off the frame alone.
func pictureBundle() (*terrain.UnitSet, *terrain.UnitClass) {
	rec := worldFixtureArt(48, 56, 24, 50, 3, 3, 2)
	rec.Name = "Human Swordsman"
	body := *rec
	body.Name = "Equipment body"
	body.Frames = worldFixtureArt(48, 56, 24, 50, 7, 2, 3).Frames
	return &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{pictureClass: rec, pictureBodyClass: &body}}, &body
}

func pictureDriver(t *testing.T, art map[sim.EntityID]*terrain.UnitClass,
	set *terrain.UnitSet, ents []sim.Entity) *mapWorld {
	t.Helper()
	w, err := sim.NewWorld(pictureSeed, sim.Bounds{Width: pictureW, Height: pictureH},
		sim.ModeCanonical, make([]byte, pictureW*pictureH), ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	v, err := ui.NewViewer("picture", terrain.Grid{
		Width: pictureW, Height: pictureH, Tiles: make([]uint16, pictureW*pictureH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	words := ui.AuthoredWords()
	words.UnitNames[pictureClass], words.UnitNames[pictureBodyClass] = pictureCaption, pictureBodyCaption
	v.SetWords(words)
	return newMapWorldWith(w, nil, set, nil, nil, art, v)
}

func pictureEntities() []sim.Entity {
	return []sim.Entity{
		{ID: pictureMember, X: 4, Y: 4, Class: pictureClass, TypeID: pictureClass, HP: 100, MaxHP: 100},
		{ID: pictureOther, X: 8, Y: 4, Class: pictureClass, TypeID: pictureClass, HP: 100, MaxHP: 100},
	}
}

func pictureDraw(t *testing.T, mw *mapWorld, id sim.EntityID) ui.MapEntity {
	t.Helper()
	for _, d := range mw.entityDraws() {
		if d.ID == uint32(id) {
			return d
		}
	}
	t.Fatalf("the push carries no entry for entity %d", id)
	return ui.MapEntity{}
}

// AC-10 — the member the body was resolved for draws the body's frames, and
// every other entity of the same class draws its own record's. Two entities of
// ONE class key is the point: a change that swapped the class rather than the
// entity's picture would move both.
func TestOnlyTheEntityAnArtEntryNamesDrawsTheBody(t *testing.T) {
	set, body := pictureBundle()
	rec := set.Classes[pictureClass]
	mw := pictureDriver(t, map[sim.EntityID]*terrain.UnitClass{pictureMember: body},
		set, pictureEntities())

	member := pictureDraw(t, mw, pictureMember)
	if member.Art != body {
		t.Errorf("the member's art is %p, want the resolved body %p", member.Art, body)
	}
	if member.Frame == nil || member.Frame.Width != 7 || member.Frame.Height != 2 {
		t.Errorf("the member draws %+v, want a frame of the body's sheet (7x2)", member.Frame)
	}
	if frameIndex(body, member.Frame) < 0 {
		t.Error("the member's frame is not one of the body's own")
	}
	// The geometry crossed with it, so the placement is still the record's.
	if member.Art.Width != rec.Width || member.Art.CenterY != rec.CenterY {
		t.Errorf("the member's canvas is %dx%d centre y %d, want the record's %dx%d / %d",
			member.Art.Width, member.Art.Height, member.Art.CenterY,
			rec.Width, rec.Height, rec.CenterY)
	}
	if member.Name != pictureCaption {
		t.Errorf("the member is named %q, want its own installed caption %q", member.Name, pictureCaption)
	}

	other := pictureDraw(t, mw, pictureOther)
	if other.Name != pictureCaption {
		t.Errorf("the other entity is named %q, want its own installed caption %q", other.Name, pictureCaption)
	}
	if other.Art != rec {
		t.Errorf("the other entity's art is %p, want its own class record %p", other.Art, rec)
	}
	if other.Frame == nil || other.Frame.Width != 3 || other.Frame.Height != 3 {
		t.Errorf("the other entity draws %+v, want a frame of the record's sheet (3x3)", other.Frame)
	}
}

func TestNoOverrideLeavesEveryEntityOnItsOwnClass(t *testing.T) {
	set, _ := pictureBundle()
	rec := set.Classes[pictureClass]
	for _, tc := range []struct {
		name string
		art  map[sim.EntityID]*terrain.UnitClass
	}{
		{"nil lookup", nil},
		{"empty lookup", map[sim.EntityID]*terrain.UnitClass{}},
		{"an entry for nobody present", map[sim.EntityID]*terrain.UnitClass{99: rec}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := pictureDriver(t, tc.art, set, pictureEntities())
			for _, id := range []sim.EntityID{pictureMember, pictureOther} {
				d := pictureDraw(t, mw, id)
				if d.Art != rec {
					t.Errorf("entity %d draws %p, want its own class record %p", id, d.Art, rec)
				}
				if d.Frame == nil || d.Frame.Width != 3 {
					t.Errorf("entity %d draws %+v, want the record's own sheet", id, d.Frame)
				}
			}
		})
	}
}

// A fall follows the OVERRIDE's corpse link, not the entity's class key — which
// is what makes the override "the whole class" rather than "the frames". The
// body here links to a corpse the record does not, so the two answers differ.
func TestAFallFollowsTheBodysOwnCorpseLink(t *testing.T) {
	set, body := pictureBundle()
	// A dying block from frame 0, four frames per direction over eight
	// directions — the shape the death path selects in.
	corpse := worldFixtureArt(48, 56, 24, 50, 9, 9, 32)
	corpse.Anim = terrain.UnitAnim{S: 16, D: 8, DyingBase: 0, DyingSlot: 4}
	body.Corpse = corpse

	ents := pictureEntities()
	ents[0].HP, ents[0].MaxHP = -1000, 100 // dead, not merely downed
	mw := pictureDriver(t, map[sim.EntityID]*terrain.UnitClass{pictureMember: body}, set, ents)

	d := pictureDraw(t, mw, pictureMember)
	if d.Art != corpse {
		t.Errorf("the fallen member draws %p, want the body's own corpse class %p", d.Art, corpse)
	}
	if d.Frame == nil || d.Frame.Width != 9 {
		t.Errorf("the fallen member draws %+v, want a frame of the corpse sheet (9x9)", d.Frame)
	}
}

// partyArt pairs the start's own two slices, and the wiring that carries it
// from the front end's bundle to the driver (plan SC-8).

func TestPartyArtPairsTheStartsIdsWithTheBundlesBodies(t *testing.T) {
	body := &terrain.UnitClass{}
	set := &terrain.UnitSet{Bodies: map[string]*terrain.UnitClass{
		data.HeroBodyKey("heroes_l", "swordsman"): body,
	}}
	ms := &Mission{
		// One member the bundle holds a (directory, body) pair for, one
		// wearing the SAME name under the OTHER directory — D-3's own
		// coupling, so a name alone could not tell these two apart — one
		// whose body the bundle does not hold at all, and one wearing none,
		// which is every party a map ever placed.
		Party: []mapload.PartyMember{
			{Body: "swordsman", BodyDir: "heroes_l"},
			{Body: "swordsman", BodyDir: "heroes"},
			{Body: "xbowman", BodyDir: "heroes_l"},
			{},
		},
		Start: mapload.Start{IDs: []sim.EntityID{7, 8, 9, 10}},
	}

	got := partyArt(ms, set)
	if len(got) != 1 {
		t.Fatalf("partyArt holds %d entries, want 1 — only the resolved (directory, body) pair", len(got))
	}
	if got[7] != body {
		t.Errorf("id 7 states %p, want the bundle's body %p", got[7], body)
	}
	for _, id := range []sim.EntityID{8, 9, 10} {
		if _, present := got[id]; present {
			t.Errorf("id %d has an entry; a (directory, body) pair the bundle does not hold states nothing", id)
		}
	}
	if partyArt(ms, nil) != nil {
		t.Error("a driver with no bundle states a picture for somebody")
	}
	// A Start from before ids were recorded pairs nothing rather than panicking.
	short := &Mission{Party: ms.Party, Start: mapload.Start{}}
	if got := partyArt(short, set); len(got) != 0 {
		t.Errorf("an idless start paired %d entries", len(got))
	}
}

// The wiring end to end: the authored body, the party the front end starts a
// mission with, the ids the start minted, and the picture the push carries.
func TestAStartedMissionDrawsItsMemberFromTheResolvedBody(t *testing.T) {
	f := missionFrontEnd(t)
	// A small invented list (SC-3); index 0 is the bare-handed name a nil
	// weapon's derivation answers.
	list := data.NewBodyList("swordsman")
	bodyName, dir, class, ok := data.HeroAppearance(list, data.Equipment{}, false, false)
	if !ok {
		t.Fatal("fixture: the bare-handed row derived no name or no directory")
	}
	rec := worldFixtureArt(48, 56, 24, 50, 3, 3, 2)
	body := *rec
	body.Frames = worldFixtureArt(48, 56, 24, 50, 7, 2, 3).Frames
	units := &terrain.UnitSet{
		Classes: map[int32]*terrain.UnitClass{class: rec},
		Bodies:  map[string]*terrain.UnitClass{data.HeroBodyKey(dir, bodyName): &body},
	}

	// This art fixture is smaller than the closed map border. Give the actor
	// an explicit resumed cell; fresh seating on it correctly refuses now.
	party := MissionParty(nil, list, nil)
	party[0].Saved = &mapload.Saved{Cell: mapload.Cell{X: 3, Y: 3}, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP}
	ms, err := StartMission(f.Archives.Containers, 10, f.Table, openDifficulty, party)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	mw := openMission(ms, f.Table, units, worldFixtureViewer(t, ms.Map),
		f.Archives.Containers, nil, nil)

	if len(ms.Start.IDs) != 1 {
		t.Fatalf("the start minted %d ids, want 1", len(ms.Start.IDs))
	}
	d := pictureDraw(t, mw, ms.Start.IDs[0])
	if d.Art != &body {
		t.Errorf("the member draws %p, want the resolved body %p", d.Art, &body)
	}
	if d.Frame == nil || d.Frame.Width != 7 || d.Frame.Height != 2 {
		t.Errorf("the member draws %+v, want a frame of the body's sheet (7x2)", d.Frame)
	}
	// Every unit the MAP placed still draws its own class record's art, which
	// for this fixture's placements is no art at all — they resolve to no class.
	for _, e := range ms.World.Entities() {
		if e.ID == ms.Start.IDs[0] {
			continue
		}
		if got := pictureDraw(t, mw, e.ID); got.Art == &body {
			t.Errorf("map-placed entity %d drew the party member's body", e.ID)
		}
	}
}

// pacedAppearanceMission builds a mapWorld whose subject is a real entity in
// a real, stocked sim.World — so mw.currentEquipment reads real slots and a
// sim.Step KindEquip command actually moves them — with invParty.list and
// .mage set BY HAND after openMission rather than read off an archive: these
// tests are about refreshAppearance's own mechanics over an equipment
// change, not about where the list comes from —
// TestAMissionsOpenAndItsFirstRefreshAgreeOnOneEquipmentSet (frontend_test.go)
// is what witnesses that wiring, off a real two-container install.
func pacedAppearanceMission(t *testing.T, equipped [sim.EquipSlots]uint16, items []uint16,
	set *terrain.UnitSet, list data.BodyList, mage bool) (mw *mapWorld, subject sim.EntityID) {
	t.Helper()
	subject = sim.EntityID(7)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: subject, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: subject, Items: items, Equipped: equipped}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{}},
		Start: mapload.Start{IDs: []sim.EntityID{subject}}}
	mw = openMission(ms, nil, set, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	mw.invParty.list, mw.invParty.mage = list, mage
	return mw, subject
}

func TestPacedMovesTheDrawnClassWhenEquipmentComposesADifferentBody(t *testing.T) {
	list := data.NewBodyList("unarmed", "swordsman")
	oldBody := &terrain.UnitClass{Name: "unarmed body"}
	newBody := &terrain.UnitClass{Name: "swordsman body"}
	set := &terrain.UnitSet{Bodies: map[string]*terrain.UnitClass{
		data.HeroBodyKey("heroes_l", "unarmed"):   oldBody,
		data.HeroBodyKey("heroes_l", "swordsman"): newBody,
	}}

	// Slot 1 holds a code whose field D is 1 — HeroBodyFor's row-1 arm,
	// list[0] "unarmed" — and the pack carries a second code, field D 2,
	// ready to be equipped into the same slot.
	var startEq [sim.EquipSlots]uint16
	startEq[0] = 1
	mw, subject := pacedAppearanceMission(t, startEq, []uint16{2}, set, list, false)

	mw.paced() // settles the tracker at the opening equipment
	if got := mw.art[subject]; got != oldBody {
		t.Fatalf("setup: after the open the drawn body is %p, want the opening body %p", got, oldBody)
	}

	// THE TICK: an equip applied directly through sim.Step, exactly the
	// move enqueueEquip's own command names — index 0 of the container into
	// slot 1.
	sim.Step(mw.world, []sim.Command{{Kind: sim.KindEquip, Entity: subject, X: 0, Y: 1}})

	mw.paced()

	if got := mw.art[subject]; got != newBody {
		t.Errorf("the drawn class is %p, want the new equipment's own body %p", got, newBody)
	}
}

func TestPacedLoadsNothingAndMovesNothingWhenEquipmentDoesNotChange(t *testing.T) {
	list := data.NewBodyList("unarmed")
	set := &terrain.UnitSet{}
	var eq [sim.EquipSlots]uint16
	eq[0] = 1

	mw, subject := pacedAppearanceMission(t, eq, nil, set, list, false)
	reads := 0
	mw.mission.src = countingSource{reads: &reads}

	// ONE SETTLING CALL, first — paced's OTHER two refreshes (refreshPack,
	// refreshEquipment) answer their OWN first difference off the SAME
	// tracker shape refreshAppearance does, and each legitimately touches
	// the archive once for a reason this task does not own. What AC-11
	// claims is the arm AFTER that: every tracker, appearance's own
	// included, has now seen this equipment once.
	mw.paced()
	before, baseline := mw.art[subject], reads

	for i := 0; i < 5; i++ {
		mw.paced()
	}

	if reads != baseline {
		t.Errorf("the archive was read %d further time(s) over unchanged equipment, want none", reads-baseline)
	}
	if got := mw.art[subject]; got != before {
		t.Errorf("the drawn class moved from %p to %p over unchanged equipment", before, got)
	}
}

func TestPacedLeavesTheExistingEntryStandingWhenTheBodyCannotResolve(t *testing.T) {
	list := data.NewBodyList("unarmed")
	existing := &terrain.UnitClass{Name: "existing"}
	set := &terrain.UnitSet{} // no Classes and no Bodies: nothing can resolve
	var eq [sim.EquipSlots]uint16
	eq[0] = 1

	mw, subject := pacedAppearanceMission(t, eq, nil, set, list, false)
	mw.art = map[sim.EntityID]*terrain.UnitClass{subject: existing}

	mw.paced()

	if got := mw.art[subject]; got != existing {
		t.Errorf("an unresolvable body wrote %p, want the previous entry %p left standing", got, existing)
	}
}
