package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// MERC-TYPE-001

// TestComposeShopFacesLeavesAHiredMemberAlone covers the appearance half.
//
// composeShopFaces derives the world-sprite half of a member's appearance from
// his equipment through data.HeroAppearance, which is the PLAYER CHARACTER's
// appearance law: it composes a sheet out of a body name under heroes/ or
// heroes_l/, which is the cloaked hero art. Run over every party member it
// dressed every hired man as a hero on the next town frame, whatever the tavern
// had built him as.
//
// THE EXPECTED VALUES ARE THE MEMBERS' OWN. Each hired member is asserted to
// leave the call carrying the class it went in with, so the assertion cannot be
// satisfied by a second call to the law under test.
//
// THE UNHIRED MEMBERS ARE ASSERTED TOO. Sarindar is the case that separates
// the two rules that could be written here: he is a temporary member and not
// a hired one, and he still composes.
func TestComposeShopFacesLeavesAHiredMemberAlone(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: shopTable(), Bodies: data.BodyList{"unarmed"}}, CampaignSession: CampaignSession{Town: NewTown(c), Carried: []mapload.PartyMember{{Name: "Danath", Class: 3, Carry: &mapload.Carry{}}, {Name: "Sarindar", Class: 3, Temporary: true, Carry: &mapload.Carry{}}, {Name: "Catapult", Class: 26, Temporary: true, MercenaryType: 1, Carry: &mapload.Carry{}}, {Name: "Ballista", Class: 27, Temporary: true, MercenaryType: 2, Carry: &mapload.Carry{}}, {Name: "NPC03_1", Class: 24, Temporary: true, MercenaryType: 3, Carry: &mapload.Carry{}}, {Name: "NPC13_1", Class: 5, Temporary: true, MercenaryType: 13, Carry: &mapload.Carry{}}}}}
	s := f.TownScreen().(*townScreen)
	s.composeShopFaces()

	for i, want := range []int32{26, 27, 24, 5} {
		p := f.Carried[i+2]
		if p.Class != want {
			t.Errorf("%q left the shop compositor as class %d, want its own row's %d",
				p.Name, p.Class, want)
		}
		if p.Body != "" || p.BodyDir != "" {
			t.Errorf("%q was given the hero body %q/%q; a hired man is drawn from his own class record",
				p.Name, p.BodyDir, p.Body)
		}
	}
	for _, i := range []int{0, 1} {
		if p := f.Carried[i]; p.Body == "" || p.BodyDir == "" {
			t.Errorf("%q left the shop compositor with no composed body (%q/%q); the player character's arm stopped running",
				p.Name, p.BodyDir, p.Body)
		}
	}
}

// TestSwitchInventorySubjectRefusesAHiredMember covers the inventory half.
//
// switchInventorySubject's own caller is documented as switching the window
// "when exactly one persistent party character is selected", and the function
// tested nothing of the kind: every entry in mission.party became the
// interactive subject. pkg/ui's inventoryEligible then requires only that the
// selection BE the subject, so selecting any hired man armed the worn box and
// the pack bar over him.
//
// IT IS NOT ONLY PRESENTATION. `PARTY-MERC-007` is High that a mercenary does
// not cross a mission boundary and that what crosses instead is the count per
// type, so nothing put into a hired man's pack survives him: an interactive
// pack is a place to lose an item in.
//
// THE REFUSAL IS OBSERVED AS THE SUBJECT NOT MOVING, which is the production
// result rather than a second reading of the gate: the previously switched-to
// member keeps the window, the selection is no longer the subject, and the two
// interactive boxes draw nothing.
func TestSwitchInventorySubjectRefusesAHiredMember(t *testing.T) {
	const humanID, siegeID, mercID sim.EntityID = 1, 2, 3
	world, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{
			{ID: humanID, X: 1, Y: 1, HP: 40, MaxHP: 40, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID},
			{ID: siegeID, X: 3, Y: 3, HP: 250, MaxHP: 250, Owner: sim.SelfSlot, TypeID: 26},
			{ID: mercID, X: 5, Y: 5, HP: 14, MaxHP: 14, Owner: sim.SelfSlot, TypeID: 24},
		}, nil)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	v, err := ui.NewViewer("hired", terrain.Grid{Width: 8, Height: 8, Tiles: make([]uint16, 64)}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	mission := &Mission{Number: 10, World: world,
		Party: []mapload.PartyMember{
			{Name: "Danath", Class: 3, Carry: &mapload.Carry{}},
			{Name: "Catapult", Class: 26, Temporary: true, MercenaryType: 1, Carry: &mapload.Carry{}},
			{Name: "NPC03_1", Class: 24, Temporary: true, MercenaryType: 3, Carry: &mapload.Carry{}},
		},
		Start: mapload.Start{IDs: []sim.EntityID{humanID, siegeID, mercID}}}
	mw := openMission(mission, nil, nil, v, nil, nil, nil)
	if _, ok := mw.figures[mercID]; !ok {
		t.Fatal("hired human lost its read-only portrait identity")
	}
	if got, ok := mw.figures[siegeID]; ok {
		t.Fatalf("siege hire gained a human portrait identity %+v", got)
	}

	mw.switchInventorySubject(uint32(humanID))
	if !mw.invSubjectSet || mw.invSubject.ID != uint32(humanID) {
		t.Fatalf("the player character did not take the window: set %v id %d", mw.invSubjectSet, mw.invSubject.ID)
	}
	for _, id := range []sim.EntityID{siegeID, mercID} {
		mw.switchInventorySubject(uint32(id))
		if mw.invSubject.ID != uint32(humanID) {
			t.Errorf("the inventory subject moved to hired entity %d; a hired man owns no inventory window",
				mw.invSubject.ID)
			mw.switchInventorySubject(uint32(humanID))
		}
	}
}

// TestCanonicalizePartyAppearanceLeavesAHiredMemberAlone covers the SECOND
// appearance site, and it is a separate witness because the shop one cannot
// see it.
//
// composeShopFaces answers for the town. canonicalizePartyAppearance answers
// for every mission open: it re-derives every party member's drawn body from
// the world's own equipment slots, which is what makes an appearance a
// derivation rather than something a producer has to remember to update. Run
// over every member it restored the composed hero sheet at the start of every
// mission, so gating the tavern builder alone would have left the defect
// exactly where the owner saw it -- in the mission.
//
// THE BODY LIST MUST BE NON-EMPTY. This function's own doc records that an
// empty list matches nothing and makes it a no-op, which is what keeps it out
// of every other synthetic fixture; a witness passing one would pass whatever
// the gate did.
func TestCanonicalizePartyAppearanceLeavesAHiredMemberAlone(t *testing.T) {
	const heroID, mercID, siegeID sim.EntityID = 1, 2, 3
	world, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{
			{ID: heroID, X: 1, Y: 1, HP: 40, MaxHP: 40, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID},
			{ID: mercID, X: 3, Y: 3, HP: 14, MaxHP: 14, Owner: sim.SelfSlot, TypeID: 24},
			{ID: siegeID, X: 5, Y: 5, HP: 250, MaxHP: 250, Owner: sim.SelfSlot, TypeID: 26},
		}, nil)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	ms := &Mission{Number: 10, World: world,
		Party: []mapload.PartyMember{
			{Name: "Danath", Class: 3},
			{Name: "NPC03_1", Class: 24, Temporary: true, MercenaryType: 3},
			{Name: "Catapult", Class: 26, Temporary: true, MercenaryType: 1},
		},
		Start: mapload.Start{IDs: []sim.EntityID{heroID, mercID, siegeID}}}

	canonicalizePartyAppearance(ms, nil, nil, data.BodyList{"unarmed"})

	for i, want := range []int32{24, 26} {
		p := ms.Party[i+1]
		if p.Class != want {
			t.Errorf("%q left the mission-open refresh as class %d, want its own row's %d",
				p.Name, p.Class, want)
		}
		if p.Body != "" || p.BodyDir != "" {
			t.Errorf("%q was given the hero body %q/%q at the mission open; a hired man is drawn from his own class record",
				p.Name, p.BodyDir, p.Body)
		}
	}
	if p := ms.Party[0]; p.Body == "" || p.BodyDir == "" {
		t.Errorf("%q left the mission-open refresh with no composed body (%q/%q); the player character's arm stopped running",
			p.Name, p.BodyDir, p.Body)
	}
}

// mercenaryHumans is a Humans collection holding one NPC03_1 row, built from
// the streamer's own documented slot map rather than from an install: slot 16 is
// the type id, 17 the face, 18 the gender (`DAT-HUMANS-008`, and pkg/data's own
// TestEachHumanSlotHasExactlyOneOutcome states the whole map). Everything else
// is left at the sentinel so the row states only what this witness is about.
type mercenaryHumans struct {
	name   string
	params []int32
}

func (c mercenaryHumans) Len() int                  { return 2 }
func (c mercenaryHumans) EntryName(i int) string    { return []string{"", c.name}[i] }
func (c mercenaryHumans) EntryParams(i int) []int32 { return [][]int32{nil, c.params}[i] }
func (c mercenaryHumans) EntryStrings(int) []string { return nil }

func mercenaryHumansRow(name string, typeID, face, gender int32) mercenaryHumans {
	params := make([]int32, 26)
	for i := range params {
		params[i] = -1
	}
	params[16], params[17], params[18] = typeID, face, gender
	return mercenaryHumans{name: name, params: params}
}

// TestBuildMercenarySquadGivesAHiredHumanHisOwnRowIdentity is the report at its
// source: the tavern builder itself.
//
// It used to finish by running data.HeroAppearance over the row's equipment and
// overwriting Body, BodyDir and Class with the answer -- the PLAYER CHARACTER's
// appearance law, which composes a sheet out of a body name under heroes/ or
// heroes_l/ and is the cloaked hero art the owner reported. Measured against the
// shipped tables before this fix, NPC03_1 left the tavern as body "mage" under
// "heroes" at class 23.
//
// THE THREE ASSERTED FIELDS ARE WHAT THE RENDERER READS. terrain.UnitSet's own
// doc states the rule: a member with no Body misses the Bodies map and is drawn
// from the class record its Class names, which is how a placed person is drawn
// and how buildSiegeSquad's Catapult and Ballista already were.
//
// HIS FIGURE IS ASSERTED TOO, and it is the field that did NOT move. A placed
// person's wire class byte is gender+0x21 or gender+0x23, so he takes the hero
// arm for the FIGURE (`HERO-APPEAR-041` through data.FigureFor) even though the
// world draws him as his class. A change that stripped the row's identity
// wholesale would satisfy the first three assertions and fail this one.
func TestBuildMercenarySquadGivesAHiredHumanHisOwnRowIdentity(t *testing.T) {
	const typeID, face, gender int32 = 24, 2, 0
	c := townCampaign(t)
	table := shopTable()
	table.Humans = mercenaryHumansRow("NPC03_1", typeID, face, gender)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: table, Bodies: data.BodyList{"unarmed"}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	s := f.TownScreen().(*townScreen)

	members, ok := s.buildMercenarySquad(3, 2)
	if !ok || len(members) != 2 {
		t.Fatalf("buildMercenarySquad(3, 2) = %d members, ok %v", len(members), ok)
	}
	wantDir, wantFace := data.FigureFor(typeID, face, gender)
	for i, m := range members {
		if m.Class != typeID {
			t.Errorf("member %d left the tavern as class %d, want its own row's type id %d",
				i, m.Class, typeID)
		}
		if m.Body != "" || m.BodyDir != "" {
			t.Errorf("member %d left the tavern wearing the hero body %q/%q; a hired man is drawn from his own class record",
				i, m.BodyDir, m.Body)
		}
		if m.MercenaryType != 3 || !m.Hired() {
			t.Errorf("member %d left the tavern with MercenaryType %d; Hired reports %v",
				i, m.MercenaryType, m.Hired())
		}
		if m.FigureDir != string(wantDir) || m.FigureFace != wantFace {
			t.Errorf("member %d figure = %q/%d, want the row's own %q/%d",
				i, m.FigureDir, m.FigureFace, wantDir, wantFace)
		}
	}
}

type manyToOneHumans struct {
	names  []string
	params [][]int32
}

func (c manyToOneHumans) Len() int                  { return len(c.names) }
func (c manyToOneHumans) EntryName(i int) string    { return c.names[i] }
func (c manyToOneHumans) EntryParams(i int) []int32 { return c.params[i] }
func (c manyToOneHumans) EntryStrings(int) []string { return nil }

// mercenaryHumanRow builds one row's params on mercenaryHumansRow's own slot
// map (DAT-HUMANS-008) above, adding the RotationSpeed column at slot 7
// (pkg/data/humandef.go's parse switch), since this witness is about that
// column specifically and mercenaryHumansRow leaves it at the sentinel.
func mercenaryHumanRow(typeID, face, gender, rotationSpeed int32) []int32 {
	params := make([]int32, 26)
	for i := range params {
		params[i] = -1
	}
	params[7], params[16], params[17], params[18] = rotationSpeed, typeID, face, gender
	return params
}

// TestBuildMercenarySquadCarriesTheHiredRowsOwnRotationSpeed is the P2
// witness (adversarial review pass 2, section 4.2). RotationSpeedBase's
// hired arm used to resolve a hired member's base by TypeID through
// data.FindHumanByType, which is many-to-one over the Humans collection and
// returns the ASCENDING-FIRST match. NPC11_1 is one of the review's own
// measured rows: TypeID 7, own RotationSpeed 12, sharing that TypeID with an
// earlier row (Man_Axe, RotationSpeed 19 there) that a by-TypeID lookup
// alone lands on instead.
//
// The witness owes a hired row that is NOT the first row of its TypeID
// (review, section 2, "What the fix owes"): a fixture built on the first
// row of a shared TypeID would pass whether or not the fix is in place,
// because FindHumanByType's ascending walk reaches it correctly by
// coincidence -- which is exactly what happened to 16 of the 52 shipped
// rows, including NPC13_1 and NPC14_1 by name.
func TestBuildMercenarySquadCarriesTheHiredRowsOwnRotationSpeed(t *testing.T) {
	const npc11TypeID, npc11Rotation = int32(7), int32(12)
	const decoyRotation = int32(19)
	c := townCampaign(t)
	table := shopTable()
	table.Humans = manyToOneHumans{
		names: []string{"", "Man_Axe", "NPC11_1"},
		params: [][]int32{
			nil,
			mercenaryHumanRow(npc11TypeID, 0, 0, decoyRotation),
			mercenaryHumanRow(npc11TypeID, 2, 0, npc11Rotation),
		},
	}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: table, Bodies: data.BodyList{"unarmed"}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	s := f.TownScreen().(*townScreen)

	members, ok := s.buildMercenarySquad(11, 1)
	if !ok || len(members) != 1 {
		t.Fatalf("buildMercenarySquad(11, 1) = %d members, ok %v", len(members), ok)
	}
	m := members[0]
	if m.Class != npc11TypeID {
		t.Fatalf("member landed on class %d, want NPC11_1's own TypeID %d", m.Class, npc11TypeID)
	}
	if m.HiredRotationSpeed != npc11Rotation {
		t.Fatalf("HiredRotationSpeed = %d, want %d (NPC11_1's own row); FindHumanByType alone answers %d (Man_Axe, the ascending-first same-TypeID row)",
			m.HiredRotationSpeed, npc11Rotation, decoyRotation)
	}

	// End to end: PartyLoadout, which every mission-open mint and every live
	// recompute reads through (RotationSpeedBase's own doc: "the one place
	// this resolution is written"), must honour the carried value rather
	// than re-deriving it.
	loadout := mapload.PartyLoadout(m, table)
	if loadout.RotationSpeed != npc11Rotation {
		t.Fatalf("PartyLoadout(member, table).RotationSpeed = %d, want %d (NPC11_1's own row)",
			loadout.RotationSpeed, npc11Rotation)
	}
}
