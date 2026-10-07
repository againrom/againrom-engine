package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The starting weapon's materialization latch, one test per failure the
// round-2 adversarial review returned this story on. Every fixture is synthetic
// (AGENTS.md rule 2); the same four behaviours are witnessed against SHIPPED
// data by scenarios/1005-doll-and-shop.json, which is the story's integration
// witness and is not run by go test.

// fallbackAlly is TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsThe
// Figure's own fixture (world_test.go), lifted here because four cases need it:
// one party member, alive on the map, slot 1 of his equipment array EMPTY, and
// a starting weapon that therefore exists only as PartyMember.Weapon — the
// DIV-070 population.
func fallbackAlly(t *testing.T) (*mapWorld, *Mission, *sim.World, sim.EntityID, data.ItemCode) {
	t.Helper()
	const allyID sim.EntityID = 9
	weapon := data.ItemCode(0x1551)
	var worn [sim.EquipSlots]uint16
	w, err := sim.NewStockedWorld(1,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: allyID, X: 3, Y: 3, Owner: sim.SelfSlot, HP: 40, MaxHP: 40}}, nil,
		sim.Relations{}, nil,
		[]sim.Stock{{ID: allyID, Equipped: worn}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{
			{Name: "Ally", PlayerCharacter: true, CompanionNPC: 30,
				FigureDir: string(data.FigureDirManFighter), FigureFace: 1,
				Weapon: &data.Weapon{Code: weapon}, Worn: worn},
		},
		Start: mapload.Start{IDs: []sim.EntityID{allyID}}}
	mw := openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	if !mw.invSubjectSet {
		t.Fatal("openMission set no inventory subject over a one-member party")
	}
	if !mw.invSubject.WeaponFallback {
		t.Fatal("setup: the fixture's slot 1 is not drawn through the fallback")
	}
	return mw, ms, w, allyID, weapon
}

// TestASecondUnitOfTheStartingWeaponsCodeDoesNotRetireTheFallback is R3
// (round-2 adversarial review, seventh pass). The seed used to scan the
// container for the starting weapon's own CODE and read a hit as proof the
// weapon had already materialized. A code is not an identity: this member
// has never taken anything off, and the unit in his pack is a second one of
// the same kind. Since the fifth pass that misreading was written back onto
// the member permanently, so one looted duplicate retired his own doll's
// slot 1 for the rest of the campaign.
func TestASecondUnitOfTheStartingWeaponsCodeDoesNotRetireTheFallback(t *testing.T) {
	mw, ms, w, ally, weapon := fallbackAlly(t)

	if !w.ReplaceStock(sim.Stock{ID: ally, Items: []uint16{uint16(weapon)}}) {
		t.Fatal("ReplaceStock refused the live ally")
	}
	mw.switchInventorySubject(uint32(ally))

	if !mw.invSubject.WeaponFallback {
		t.Error("WeaponFallback went false with the starting weapon's code merely carried, not worn")
	}
	if ms.Party[0].WeaponMaterialized {
		t.Error("WeaponMaterialized was latched by a second unit of the same code sitting in the pack")
	}
	if occupied, _ := mw.invFigureEquipment.Occupied(1); !occupied {
		t.Error("the doll stopped drawing the starting weapon because a duplicate was in the pack")
	}
}

// TestTakingTheDrawnStartingWeaponOffPutsARealOneInThePack is the owner's
// own directive crossing the seam: pkg/ui now arms a press on a
// fallback-only slot 1, and this is the far side answering it. The weapon
// becomes a real unit in the member's own container, the persisted latch
// goes up, and NO KindUnequip is queued — there is nothing in the array
// for sim to move, which is why the gesture was refused for four passes
// instead.
func TestTakingTheDrawnStartingWeaponOffPutsARealOneInThePack(t *testing.T) {
	mw, ms, w, ally, weapon := fallbackAlly(t)

	mw.enqueueUnequip(0)

	if len(mw.pending) != 0 {
		t.Errorf("pending = %+v, want nothing queued: the array's slot 1 is empty", mw.pending)
	}
	items, _ := w.Carried(ally)
	if len(items) != 1 || items[0] != uint16(weapon) {
		t.Errorf("container = %v, want exactly the taken-off %d", items, weapon)
	}
	if !ms.Party[0].WeaponMaterialized {
		t.Error("WeaponMaterialized still false after the drawn weapon was taken off")
	}
	mw.refreshEquipment()
	if mw.invSubject.WeaponFallback {
		t.Error("the doll went on drawing the starting weapon after it was taken off")
	}

	// A SECOND TAP TAKES NOTHING OFF. The latch is up, so the fallback is no
	// longer drawn and the array is still empty: the gesture has nothing to
	// name and must not mint a second unit.
	mw.enqueueUnequip(0)
	if items, _ := w.Carried(ally); len(items) != 1 {
		t.Errorf("container = %v after a second take-off, want the same single unit", items)
	}
}

func TestTakingTheDrawnStartingWeaponOffPreservesExistingItemInstances(t *testing.T) {
	mw, _, w, ally, weapon := fallbackAlly(t)
	worn := sim.ItemInstance{Code: 0x0302, Kind: 1, Price: 17,
		Effects: []sim.ItemEffect{{Kind: 15, Mode: 1, Operand: 4}}}
	carried := sim.ItemInstance{Code: 0x0e03, Kind: 3, Price: 29,
		Effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 12}}}
	var equipped [sim.EquipSlots]sim.ItemInstance
	equipped[2] = worn
	if !w.ReplaceStock(sim.Stock{ID: ally, ItemInstances: []sim.ItemInstance{carried}, EquippedItems: equipped}) {
		t.Fatal("ReplaceStock refused the complete fixture")
	}

	mw.enqueueUnequip(0)

	items, _ := w.CarriedItems(ally)
	if len(items) != 2 || !sim.StackStateEqual(sim.StackItem(items[0], 1), sim.StackItem(carried, 1)) || items[1].Code != uint16(weapon) {
		t.Fatalf("carried instances after fallback materialization = %+v", items)
	}
	gotWorn, _ := w.EquippedItems(ally)
	if !sim.StackStateEqual(sim.StackItem(gotWorn[2], 1), sim.StackItem(worn, 1)) {
		t.Fatalf("equipped instance after fallback materialization = %+v", gotWorn[2])
	}
}

// TestDroppingTheDrawnStartingWeaponReachesTheGround is the same crossing for
// a release that lands on neither inventory box: the weapon materializes into
// the container and the drop then leaves FROM the container, so one gesture
// reaches the ground.
func TestDroppingTheDrawnStartingWeaponReachesTheGround(t *testing.T) {
	mw, ms, w, ally, weapon := fallbackAlly(t)

	mw.enqueueDrop(true, 0, 4, 4)

	if len(mw.pending) != 1 {
		t.Fatalf("pending = %+v, want exactly one command", mw.pending)
	}
	if got := mw.pending[0].Kind; got != sim.KindDropCarried {
		t.Errorf("command kind = %v, want KindDropCarried: the weapon is in the container by now", got)
	}
	items, _ := w.Carried(ally)
	if len(items) != 1 || items[0] != uint16(weapon) {
		t.Errorf("container = %v, want the materialized %d for the drop to take", items, weapon)
	}
	if !ms.Party[0].WeaponMaterialized {
		t.Error("WeaponMaterialized still false after the drawn weapon was dropped")
	}
	sim.Step(w, mw.pending)
	if items, _ := w.Carried(ally); len(items) != 0 {
		t.Errorf("container = %v after the drop was applied, want empty", items)
	}
	if len(w.Sacks()) == 0 {
		t.Error("no sack on the ground after the drop was applied")
	}
}

// TestAMidMissionLatchReachesTheSave is R1. Both mid-mission writers reach
// Mission.Party; Snapshot writes f.liveParty. While liveDriver took
// mapload.OwnParty's deep copy the two were different arrays, so the latch was
// correct in memory and false on disk — reload restored a member whose
// starting weapon had never been taken off, and the retired fallback came back
// holding it.
//
// The write below goes through materializeStartingWeapon, the one function
// that raises the latch, applied to the mission's own record exactly as rearm
// and switchInventorySubject apply it.
func TestAMidMissionLatchReachesTheSave(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: saveTown(t), Carried: saveParty()}}
	w, err := sim.NewWorld(3, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), nil)
	if err != nil {
		t.Fatal(err)
	}
	party := saveParty()
	f.liveDriver(&mapWorld{world: w, commanded: map[sim.EntityID]bool{},
		swing: map[sim.EntityID]int{}, phase: map[sim.EntityID]sim.AttackPhase{}}, 10, party)

	materializeStartingWeapon(&party[0])

	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatalf("Snapshot(true): %v", err)
	}
	if len(s.Party) != 1 {
		t.Fatalf("saved party = %d members, want 1", len(s.Party))
	}
	if !s.Party[0].WeaponMaterialized {
		t.Error("a mid-mission WeaponMaterialized write is absent from the save")
	}
	if s.Party[0].ID == "" {
		t.Error("the saved member lost the stable identity liveDriver assigns")
	}
}

// TestTakingARealWeaponOffInTheShopRetiresTheFallback is R2. The shop set the
// latch on its viaFallback arm alone, so taking a REAL slot-1 item off left it
// down: shopWeaponFallbackCode then offered the member's starting weapon into
// the slot he had just emptied, and the next take-off appended a SECOND unit
// of it to the pack, minted from nothing. Reachable in shipped content by any
// member whose slot 1 the loader filled for real and whose latch no mission
// seed has raised — every companion added by addChapterCompanions, since
// openMission seeds ms.Party[0] alone.
func TestTakingARealWeaponOffInTheShopRetiresTheFallback(t *testing.T) {
	real := invWornCode(1)
	starting := invWornCode(2)
	f, s := shopRoom(t, nil)
	f.Carried[0].Weapon = &data.Weapon{Code: starting}
	s.shopWornSlots(0)[0] = uint16(real)

	if act := s.shopUnequipDoll(1); act.Msg != "off, into the pack" {
		t.Fatalf("shopUnequipDoll(1) over a real weapon = %+v, want the ordinary success message", act)
	}
	if !f.Carried[0].WeaponMaterialized {
		t.Error("WeaponMaterialized still false after a real slot-1 weapon was taken off")
	}
	if code, ok := s.shopWeaponFallbackCode(0); ok {
		t.Errorf("the shop offers fallback %d into the slot just emptied of a real weapon", code)
	}

	if act := s.shopUnequipDoll(1); act.Msg != "" {
		t.Errorf("a second take-off on the now-empty slot said %q, want silence", act.Msg)
	}
	items := f.Carried[0].Carry.Items
	if len(items) != 1 || items[0] != uint16(real) {
		t.Errorf("pack = %v, want exactly the one real weapon %d", items, real)
	}
}

// TestStagingARealWeaponOnTheTableRetiresTheFallback is R2 through the shop's
// other take-off door, which had the same one-armed write.
func TestStagingARealWeaponOnTheTableRetiresTheFallback(t *testing.T) {
	real := invWornCode(1)
	starting := invWornCode(2)
	f, s := shopRoom(t, nil)
	f.Carried[0].Weapon = &data.Weapon{Code: starting}
	s.shopWornSlots(0)[0] = uint16(real)

	if act := s.shopUnequipToTable(1); act.Msg != "on the table" {
		t.Fatalf("shopUnequipToTable(1) over a real weapon = %+v, want \"on the table\"", act)
	}
	if !f.Carried[0].WeaponMaterialized {
		t.Error("WeaponMaterialized still false after a real slot-1 weapon reached the table")
	}
	if code, ok := s.shopWeaponFallbackCode(0); ok {
		t.Errorf("the shop offers fallback %d into the slot just emptied onto the table", code)
	}
	if table := f.Shop.Table(); len(table) != 1 || table[0].Code != real {
		t.Errorf("table = %+v, want the one real weapon %d", table, real)
	}
}

// TestASeenRealSlotOneKeepsTheFallbackRetiredAfterAnUnequip witnesses
// resolveWeaponMaterialized's own WRITE-BACK, the line that raises the latch
// the first time the seed sees slot 1 genuinely occupied.
//
// It exists because the fifth pass claimed that line was witnessed by
// TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsTheFigure and it was
// not: removing the write-back left the whole suite green, since rearm's own
// write raised the same bit a frame later. The difference is observable only
// where the seed runs and rearm does not, which is what this drives: select the
// member with the weapon really worn, take it off through sim alone, and select
// him again. Without the write-back the second seed reads an empty slot 1, finds
// no history, and the retired starting weapon is drawn back onto the doll.
func TestASeenRealSlotOneKeepsTheFallbackRetiredAfterAnUnequip(t *testing.T) {
	mw, ms, w, ally, weapon := fallbackAlly(t)

	if !w.ReplaceStock(sim.Stock{ID: ally, Items: []uint16{uint16(weapon)}}) {
		t.Fatal("ReplaceStock refused the live ally")
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: ally, X: 0, Y: 1}})
	mw.switchInventorySubject(uint32(ally))
	if mw.invSubject.WeaponFallback {
		t.Fatal("setup: the fallback is still drawn over a genuinely worn slot 1")
	}
	if !ms.Party[0].WeaponMaterialized {
		t.Fatal("the seed did not write the latch back when it saw slot 1 really occupied")
	}

	sim.Step(w, []sim.Command{{Kind: sim.KindUnequip, Entity: ally, X: 1}})
	mw.switchInventorySubject(uint32(ally))
	if mw.invSubject.WeaponFallback {
		t.Error("the starting weapon was drawn back onto the doll after a real slot-1 item was taken off")
	}
}

// TestRecomputeRaisedSkillsReadsTheLiveLatchNotTheMissionOpenSnapshot is the
// round-2 adversarial review's eighth-pass counterexample: recomputeRaisedSkills
// (rearm.go) read everEquipped as SUBJECT-SCOPED — mw.invSubjectSet and this
// entity being mw.invSubject — on the premise that only the inventory
// screen's own door produces sim.KindUnequip, over invSubject alone. This
// story's own new gesture (world.go's materializeFallbackWeapon, armed by the
// owner's press-to-take-off directive) raises PartyMember.WeaponMaterialized
// for WHICHEVER member is subject at the time, and the flag survives a later
// subject switch; recomputeRaisedSkills also ran over EVERY party member with
// a raised skill, not only the current subject. A companion disarmed while he
// was briefly the subject, then left non-subject when his skill later rose,
// read everEquipped false regardless — re-armed with the starting weapon's
// damage the instant a raise reached rearm.go's own pass.
//
// It drives every step through the production entry points the counterexample
// names: switchInventorySubject to make the ally the subject, enqueueUnequip
// to tap the fallback off (world.go's own materializeFallbackWeapon, not a
// direct field write), switchInventorySubject again to move off him, and one
// real KindAttack tick to raise his own skill through sim's ordinary award
// path — recomputeRaisedSkills is never called directly.
//
// TO CONFIRM IT WITNESSES THE FIX, revert rearm.go's everEquipped line to
// `mw.invSubjectSet && sim.EntityID(mw.invSubject.ID) == c.id &&
// mw.invWeaponEverEquipped`. The final assertion reddens: DamageBase reads
// the sworded value because the subject-scoped read never saw the ally's own
// latch.
func TestRecomputeRaisedSkillsReadsTheLiveLatchNotTheMissionOpenSnapshot(t *testing.T) {
	const hero, ally, victim = sim.EntityID(7), sim.EntityID(9), sim.EntityID(11)
	table := eqDefsTable(t)
	sword := eqSword(t, table)

	// Body 1 keeps the bare-handed swing at data's own floor (rearm.go's own
	// doc: "ftol(1.1^Body/20) ... zero below Body 32"), and both numbers below
	// are read off data.Hero.Recompute — never hand-computed — so a change to
	// that arithmetic reddens the setup check instead of silently comparing
	// the wrong pair of numbers.
	allyHero := data.Hero{Body: 1, Reaction: 20, Mind: 30, Spirit: 15}
	bare := allyHero.Recompute(data.Profile{}, data.Loadout{}).Combat.DamageBase
	sworded := allyHero.Recompute(data.Profile{}, data.Loadout{Weapon: sword}).Combat.DamageBase
	if bare == sworded {
		t.Fatalf("setup: bare and sworded DamageBase both %d; choose a fixture that discriminates", bare)
	}
	reward := allyHero.Reward()

	var allyWorn [sim.EquipSlots]uint16
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{
			{ID: hero, X: 3, Y: 3, HP: 10, MaxHP: 10},
			// DamageBase is set DIRECTLY on the raw fixture, on
			// TestRearmRecomputesToHitWhenTheCreditedSkillRisesWithNoEquipThatTick's
			// own precedent: the attack below resolves and pays experience
			// (payExperience refuses removed <= 0) BEFORE recomputeRaisedSkills
			// or rearm ever run for this tick, so a non-zero blow needs a
			// number here independent of the derived combat block this test
			// is itself about to witness change.
			{ID: ally, X: 10, Y: 10, HP: 100, MaxHP: 100, Reach: 1, Owner: 2,
				GainsXP: true, TypeID: sim.HumanTypeID, Mind: reward.Mind, SkillXP: reward.SkillXP,
				XPSlot: uint8(data.SkillBlade), DamageBase: 5, AlwaysHits: true,
				AttackCharge: 1, AttackRelax: 1},
			{ID: victim, X: 11, Y: 10, HP: 1_000_000, MaxHP: 1_000_000,
				DyingTime: 200, Owner: 3, XPValue: 100},
		}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: ally, Equipped: allyWorn}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{
			{Hero: data.Hero{Body: 30, Reaction: 20, Mind: 15, Spirit: 15}},
			{Weapon: sword, Worn: allyWorn},
		},
		Start: mapload.Start{IDs: []sim.EntityID{hero, ally}}}
	mw := openMission(ms, table, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)

	if !mw.invSubjectSet || mw.invSubject.ID != uint32(hero) {
		t.Fatalf("setup: subject after open = %+v, want the hero (Start.IDs[0])", mw.invSubject)
	}

	// THE FIRST PROBE STEP: the ally becomes the subject, his drawn fallback
	// weapon is tapped off — materializeFallbackWeapon, not a direct write —
	// and the persisted latch on ms.Party[1] goes up.
	mw.switchInventorySubject(uint32(ally))
	if !mw.invSubject.WeaponFallback {
		t.Fatal("setup: the ally's slot 1 is not drawn through the fallback")
	}
	mw.enqueueUnequip(0)
	if !ms.Party[1].WeaponMaterialized {
		t.Fatal("setup: tapping the fallback off did not raise the ally's own latch")
	}
	if items, _ := w.Carried(ally); len(items) != 1 {
		t.Fatalf("setup: ally's container = %v after the tap-off, want the one materialized unit", items)
	}

	// THE SECOND PROBE STEP: the subject moves back to the hero, off the ally
	// entirely.
	mw.switchInventorySubject(uint32(hero))
	if mw.invSubject.ID != uint32(hero) {
		t.Fatal("setup: the subject did not move back to the hero")
	}

	// THE ALLY RAISES A SKILL WHILE NOT THE SUBJECT. He always hits, so his
	// first blow crosses S(0)=0 in Blade and awardSkill raises the level to 1
	// inside this one tick — recomputeRaisedSkills' own trigger.
	mw.pending = append(mw.pending, sim.Command{Kind: sim.KindAttack, Entity: ally, X: int32(victim)})
	mw.tick()

	e, ok := mw.entity(ally)
	if !ok {
		t.Fatal("the ally is gone after one tick")
	}
	if e.Skill[data.SkillBlade] == 0 {
		t.Fatal("setup: the ally's Blade level did not rise — recomputeRaisedSkills had nothing to recompute")
	}
	if e.DamageBase != bare {
		t.Errorf("DamageBase = %d, want %d (bare-handed, latch raised, slot 1 empty) — "+
			"got %d, the sworded value: the ally's own latch was ignored and he was re-armed "+
			"with the sold starting weapon's damage", e.DamageBase, bare, e.DamageBase)
	}
}
