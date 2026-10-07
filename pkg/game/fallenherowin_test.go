package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// fallenHeroWinMission is mission 20 of the continuity fixture campaign, whose
// win opens the town, built in the shape mission 140 ships: the script wins
// once the fallen ally 3 stands at health 1 or more. The hero (1) is a sword
// fighter wearing his sword and carrying a mace, beside a practice target (4)
// and a sack holding a bow. The companion mage (2) knows Heal and holds mana
// for one cast, below the idle-heal reserve, so only an ordered Heal lands.
func fallenHeroWinMission(t *testing.T) (*Mission, *mapWorld, *mapload.Table) {
	t.Helper()
	table := eqDefsTable(t)
	win := missionTrigger(3, 1, true, 0)
	win.Pairs[0].Cmp = sim.ScriptCmpGE
	s, err := sim.NewScript([]sim.ScriptCheck{
		{Op: sim.ScriptCheckHealth, Register: 0, Args: [10]int32{6}, Unit: 3, HasUnit: true},
		{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
	}, []sim.ScriptInstant{{Op: sim.ScriptInstantWin}}, []sim.ScriptTrigger{win})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	h := eqBladeHero()
	reward := h.Reward()
	hero := fallenHeroEntity(1, 20, 20, 8)
	hero.Reach, hero.GainsXP, hero.Mind, hero.SkillXP = 1, true, reward.Mind, reward.SkillXP
	hero.XPSlot, hero.DamageBase, hero.AlwaysHits, hero.AttackCharge, hero.AttackRelax = uint8(data.SkillBlade), 5, true, 1, 1
	hero.Skill[data.SkillBlade] = h.Skill[data.SkillBlade]
	mage := fallenHeroEntity(2, 30, 30, 8)
	mage.HP, mage.MaxHP, mage.TypeID = 30, 30, sim.HeroTypeID(true, false)
	mage.Mind, mage.MaxMana, mage.Mana, mage.KnownSpells, mage.ScanRange = 30, 100, 10, 1<<6, 19
	ally := sim.Entity{ID: 3, X: 32, Y: 30, HP: 0, MaxHP: 50, DyingTime: 8, Owner: 5, TypeID: sim.HumanTypeID, Humanoid: true}
	sim.PrepareAuthoredBody(&ally)
	target := sim.Entity{ID: 4, X: 21, Y: 20, HP: 1_000_000, MaxHP: 1_000_000, DyingTime: 200, Owner: 3, XPValue: 4}
	heal := sim.SpellRule{ID: 6, ManaCost: 10, School: 5, MaxRange: 6,
		DamageMin: 10, DamageMax: 20, TargetsUnit: true, Restorative: true}
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical,
		sim.Terrain{Block: make([]byte, worldFixtureW*worldFixtureH)}, []sim.Entity{hero, mage, ally, target}, s,
		sim.Relations{}, []sim.Sack{{X: 20, Y: 17, Items: []uint16{eqBowCode}}},
		[]sim.Stock{{ID: 1, Items: []uint16{eqMaceCode}, Equipped: [sim.EquipSlots]uint16{eqSwordCode}}},
		[]sim.SpellRule{heal})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 20, Map: m, World: w,
		Party: []mapload.PartyMember{
			{ID: "hero", Class: 100, Hero: h, Weapon: eqSword(t, table), PlayerCharacter: true, StartingHero: true},
			{ID: "mage", Class: 100, PlayerCharacter: true},
		},
		Start: mapload.Start{IDs: []sim.EntityID{1, 2}}}
	return ms, openMission(ms, table, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil), table
}

// fallenHeroWinEarn has the hero strike the practice target until he earns
// experience and then pick up the bow. It returns him as he entered and as he
// ends the phase, with his pack and worn set.
func fallenHeroWinEarn(t *testing.T, ms *Mission, mw *mapWorld) (entry, earned sim.Entity, pack []uint16, worn [sim.EquipSlots]uint16) {
	t.Helper()
	entry, _ = mw.entity(1)
	mw.tick()
	for tick := 0; tick < 4000; tick++ {
		if e, _ := mw.entity(1); e.SkillXP != entry.SkillXP {
			break
		}
		mw.strike(1, 4)
		mw.tick()
	}
	mw.orderPickup(1, 20, 17)
	for tick := 0; tick < 400 && mw.pickup.set; tick++ {
		mw.tick()
	}
	earned, _ = mw.entity(1)
	pack, _ = ms.World.Carried(1)
	worn, _ = ms.World.Equipped(1)
	if earned.SkillXP == entry.SkillXP || !slices.Contains(pack, eqBowCode) || !slices.Contains(pack, eqMaceCode) || worn[0] != eqSwordCode {
		t.Fatalf("setup: hero experience %v (entry %v), pack %v, worn %v: want earned experience, the bow and the mace carried, the sword worn",
			earned.SkillXP, entry.SkillXP, pack, worn)
	}
	return entry, earned, pack, worn
}

// fallenHeroWinTown takes the won mission 20 through continuity and requires
// the town.
func fallenHeroWinTown(t *testing.T, ms *Mission, table *mapload.Table) *FrontEnd {
	t.Helper()
	f := continuityFront(t)
	f.Table = table
	f.Archives, f.Tiles = &Archives{Containers: missionArchive(t, 30)}, &terrain.Tileset{}
	dest, msg, _ := f.continuity(20, ms, listAdvance)()
	if dest != ui.NoticeToTown || !f.Town.Open() {
		t.Fatalf("Victory went to %v (%q), town open=%v: want the town", dest, msg, f.Town.Open())
	}
	return f
}

// A hero felled by the K key still lies at health -9 through 0 when his
// companion's ordered Heal raises the fallen ally and the script wins. The
// win's town carries the experience he earned, the bow he picked up, the mace
// he carried and the sword he wore; the finished world holds him raised at
// full health, and the next mission opens with him at full health holding
// all of it.
func TestAHeroLyingFallenAtTheWinCarriesHisMissionIntoTown(t *testing.T) {
	ms, mw, table := fallenHeroWinMission(t)
	_, earned, pack, worn := fallenHeroWinEarn(t, ms, mw)

	mw.affect(1, true)
	mw.tick()
	mw.attackOrCast(2, 3, 6, 0, 0, false)
	for tick := 0; tick < 800 && !mw.mission.announced; tick++ {
		mw.tick()
	}
	fallen, _ := mw.entity(1)
	if !mw.mission.announced || mw.mission.outcome != sim.OutcomeWon || ms.World.Outcome() != sim.OutcomeWon {
		t.Fatalf("announced=%v outcome=%v world=%v with the hero at health %d: want the script's win",
			mw.mission.announced, mw.mission.outcome, ms.World.Outcome(), fallen.HP)
	}
	if fallen.Alive() || !fallen.Restorable() || fallen.SkillXP != earned.SkillXP || fallen.Defence != earned.Defence>>1 {
		t.Fatalf("hero at the win: health %d restorable=%v experience %v defence %d, want a fallen body Heal can raise holding his experience",
			fallen.HP, fallen.Restorable(), fallen.SkillXP, fallen.Defence)
	}

	f := fallenHeroWinTown(t, ms, table)
	if len(f.Carried) != 2 || f.Carried[0].ID != "hero" || f.Carried[0].Carry == nil {
		carried := len(f.Carried) > 0 && f.Carried[0].Carry != nil
		t.Fatalf("town party of %d, first member carries his mission=%v: want the hero first, holding what he ended the mission with",
			len(f.Carried), carried)
	}
	carry := f.Carried[0].Carry
	if carry.SkillXP != earned.SkillXP || !reflect.DeepEqual(carry.Items, pack) || carry.Equipped != worn {
		t.Fatalf("town hero carries experience %v, pack %v, worn %v; want %v, %v, %v",
			carry.SkillXP, carry.Items, carry.Equipped, earned.SkillXP, pack, worn)
	}
	// Heal's transition doubles the halved defence back, so an odd defence
	// comes back one lower (HERO-REVIVE-068, DIV-1448); the next mission
	// derives this fixture's defence anew.
	raised, _ := mw.entity(1)
	if !raised.Alive() || raised.HP != raised.MaxHP || raised.Decay != sim.DecayNone || raised.Defence != earned.Defence>>1<<1 {
		t.Fatalf("finished world hero health %d/%d stage %d defence %d, want him raised at full health with defence %d",
			raised.HP, raised.MaxHP, raised.Decay, raised.Defence, earned.Defence>>1<<1)
	}

	next, err := StartMission(f.Archives.Containers, f.Town.Chapter(), f.Table, openDifficulty, f.Carried)
	if err != nil {
		t.Fatalf("StartMission(%d): %v", f.Town.Chapter(), err)
	}
	opened, ok := heroEntity(next)
	if !ok || !opened.Alive() || opened.HP != opened.MaxHP || opened.SkillXP != earned.SkillXP || opened.Defence != earned.Defence {
		t.Fatalf("mission %d hero present=%v health %d/%d experience %v defence %d, want full health, %v and defence %d",
			f.Town.Chapter(), ok, opened.HP, opened.MaxHP, opened.SkillXP, opened.Defence, earned.SkillXP, earned.Defence)
	}
	carried, _ := next.World.Carried(opened.ID)
	wearing, _ := next.World.Equipped(opened.ID)
	if !reflect.DeepEqual(carried, pack) || wearing != worn {
		t.Fatalf("mission %d hero carries %v and wears %v, want %v and %v", f.Town.Chapter(), carried, wearing, pack, worn)
	}
	t.Logf("fallen at health %d when the script won; town hero experience %v; mission %d opened him at %d/%d",
		fallen.HP, carry.SkillXP, f.Town.Chapter(), opened.HP, opened.MaxHP)
}

// TestAHeroAtMinusTenAtTheWinReturnsAsHeEnteredTheMission is the other side of
// the boundary (DIV-1488). The K key fells the hero and three presses of the L
// key's damage take him to -10 inside his dying window as the script wins.
// Heal can no longer raise him, so the cull leaves him as he lies: the town
// carries him forward as he entered mission 20, and mission 30 opens him at
// full health with his entry experience and without the bow.
func TestAHeroAtMinusTenAtTheWinReturnsAsHeEnteredTheMission(t *testing.T) {
	// The win's tick, measured on the same fixture without a fall.
	control, controlWorld, _ := fallenHeroWinMission(t)
	fallenHeroWinEarn(t, control, controlWorld)
	controlWorld.attackOrCast(2, 3, 6, 0, 0, false)
	win := 0
	for ; win < 800 && !controlWorld.mission.announced; win++ {
		controlWorld.tick()
	}

	ms, mw, table := fallenHeroWinMission(t)
	entry, _, _, _ := fallenHeroWinEarn(t, ms, mw)
	mw.attackOrCast(2, 3, 6, 0, 0, false)
	for tick := 0; tick < 800 && !mw.mission.announced; tick++ {
		if tick == win-6 {
			mw.affect(1, true)
		} else if tick > win-6 && tick <= win-3 {
			mw.affect(1, false)
		}
		mw.tick()
	}
	lying, _ := mw.entity(1)
	if mw.mission.outcome != sim.OutcomeWon || lying.HP > -10 || !lying.Dying() {
		t.Fatalf("outcome %v with the hero at health %d dying=%v (control won on tick %d): want the win with him at -10 or below inside his dying window",
			mw.mission.outcome, lying.HP, lying.Dying(), win)
	}

	f := fallenHeroWinTown(t, ms, table)
	if len(f.Carried) == 0 || !reflect.DeepEqual(f.Carried[0], ms.Party[0]) {
		t.Fatalf("town party of %d does not open with the hero as he entered the mission", len(f.Carried))
	}
	next, err := StartMission(f.Archives.Containers, f.Town.Chapter(), f.Table, openDifficulty, f.Carried)
	if err != nil {
		t.Fatalf("StartMission(%d): %v", f.Town.Chapter(), err)
	}
	opened, ok := heroEntity(next)
	carried, _ := next.World.Carried(opened.ID)
	if !ok || !opened.Alive() || opened.HP != opened.MaxHP || opened.SkillXP != entry.SkillXP || slices.Contains(carried, eqBowCode) {
		t.Fatalf("mission %d hero present=%v health %d/%d experience %v carrying %v, want full health, the entry experience %v and no bow",
			f.Town.Chapter(), ok, opened.HP, opened.MaxHP, opened.SkillXP, carried, entry.SkillXP)
	}
}

// TestAFallenHiredManTheCullRaisesStillLeavesThePool: MERC-DEATH-006 tallies
// the living mercenaries before the cull. Two hired men of one type walk in
// with band type ids and the K key fells one before the win. The end cull
// raises him like any restorable band actor (PARTY-ENDCULL-026), and the tally
// taken before it still counts one living man, so the pool becomes 1.
func TestAFallenHiredManTheCullRaisesStillLeavesThePool(t *testing.T) {
	s, err := sim.NewScript(missionChecks(),
		[]sim.ScriptInstant{{Op: sim.ScriptInstantWin}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	ents := []sim.Entity{
		{ID: 0, X: 10, Y: 10, HP: 20, MaxHP: 20, DyingTime: 8, Owner: sim.SelfSlot, Domain: sim.DomainGround, TypeID: sim.HumanTypeID, GainsXP: true},
		{ID: 1, X: 11, Y: 10, HP: 15, MaxHP: 15, DyingTime: 8, Owner: sim.SelfSlot, Domain: sim.DomainGround, TypeID: sim.HumanTypeID},
		{ID: 2, X: 12, Y: 10, HP: 15, MaxHP: 15, DyingTime: 8, Owner: sim.SelfSlot, Domain: sim.DomainGround, TypeID: sim.HumanTypeID},
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), ents, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	sim.Step(w, []sim.Command{sim.Kill(2)})
	for i := 0; i < 64 && w.Outcome() == sim.OutcomeUndecided; i++ {
		sim.Step(w, nil)
	}
	ids := []sim.EntityID{0, 1, 2}
	fallen := gaEntity(t, w, 2)
	if w.Outcome() != sim.OutcomeWon || !fallen.Restorable() {
		t.Fatalf("fixture outcome %v with the hired man at health %d: want the win with him restorable", w.Outcome(), fallen.HP)
	}
	f := mercenaryBoundaryFront(t)
	party := []mapload.PartyMember{
		{ID: "hero", StartingHero: true, PlayerCharacter: true, Class: 100},
		{ID: "merc:3:1", MercenaryType: boundaryHiredType, Class: sim.HumanTypeID},
		{ID: "merc:3:2", MercenaryType: boundaryHiredType, Class: sim.HumanTypeID},
	}

	f.FinishMission(10, party, w, ids)

	if raised := gaEntity(t, w, 2); !raised.Alive() {
		t.Fatalf("the cull left the hired man at health %d, want him raised like any restorable band actor", raised.HP)
	}
	if got := f.Town.MercenaryPool(boundaryHiredType); got != 1 {
		t.Errorf("hired type %d pool = %d, want 1: one of two walked out alive, and the tally runs before the cull raises the other",
			boundaryHiredType, got)
	}
}
