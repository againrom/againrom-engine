package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// fallenBodyWinMission is mission 20 of the continuity fixture campaign, whose
// win opens the town. Its script hands the villager (4) to the player on the
// first pass and wins once the hero (1) stands within two cells of (40,40).
// Companions 2 and 3 walked in with the hero; a sack beside the hero holds a
// mace and one beside 3 a bow. Every actor has an original Humans row's
// eight-tick dying time. The hero fights unarmed, so his recovery holds no
// weapon penalty and each blow lands for exactly 30: 2 and the villager at
// health 20 fall to -10 and 3 at health 30 falls to 0.
func fallenBodyWinMission(t *testing.T) (*Mission, *mapWorld, *mapload.Table) {
	t.Helper()
	table := eqDefsTable(t)
	var give, win sim.ScriptTrigger
	give.Pairs[0] = sim.ScriptPair{Left: 1, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}
	give.Instants = [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}
	give.Once, give.Latch = true, 0
	win.Pairs[0] = sim.ScriptPair{Left: 0, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}
	win.Instants = [4]int32{1, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}
	win.Once, win.Latch = true, 1
	s, err := sim.NewScript([]sim.ScriptCheck{
		{Op: sim.ScriptCheckWithin, Register: 0, Args: [10]int32{40, 40, 2}, Unit: 1, HasUnit: true},
		{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
	}, []sim.ScriptInstant{
		{Op: sim.ScriptInstantGiveUnit, Unit: 4, HasUnit: true, Player: sim.SelfSlot, HasPlayer: true},
		{Op: sim.ScriptInstantWin},
	}, []sim.ScriptTrigger{give, win})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	hero := fallenHeroEntity(1, 30, 30, 8)
	hero.Reach, hero.DamageBase, hero.AlwaysHits, hero.AttackCharge, hero.AttackRelax = 1, 30, true, 1, 1
	first := fallenHeroEntity(2, 40, 42, 8)
	first.HP, first.MaxHP = 20, 20
	second := fallenHeroEntity(3, 41, 42, 8)
	second.HP, second.MaxHP = 30, 30
	villager := fallenHeroEntity(4, 39, 42, 8)
	villager.HP, villager.MaxHP, villager.Owner = 20, 20, 2
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical,
		sim.Terrain{Block: make([]byte, worldFixtureW*worldFixtureH)}, []sim.Entity{hero, first, second, villager}, s,
		sim.Relations{}, []sim.Sack{{X: 30, Y: 32, Items: []uint16{eqMaceCode}}, {X: 41, Y: 45, Items: []uint16{eqBowCode}}},
		nil, nil)
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 20, Map: m, World: w,
		Party: []mapload.PartyMember{
			{ID: "hero", Class: 100, PlayerCharacter: true, StartingHero: true},
			{ID: "first", Class: 100, PlayerCharacter: true},
			{ID: "second", Class: 100, PlayerCharacter: true},
		},
		Start: mapload.Start{IDs: []sim.EntityID{1, 2, 3}, Roster: map[sim.EntityID]mapload.PartyMember{
			4: {ID: "join:4", Name: "Villager", Class: 100, PlayerCharacter: true},
		}}}
	return ms, openMission(ms, table, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil), table
}

// fallenBodyWinStrike has the hero strike victim until it falls and returns
// its body.
func fallenBodyWinStrike(t *testing.T, mw *mapWorld, victim sim.EntityID) sim.Entity {
	t.Helper()
	mw.strike(1, uint32(victim))
	for tick := 0; tick < 400; tick++ {
		if e, _ := mw.entity(victim); !e.Alive() {
			return e
		}
		mw.tick()
	}
	e, _ := mw.entity(victim)
	t.Fatalf("actor %d still at health %d after 400 ticks of the hero's blows", victim, e.HP)
	return e
}

// fallenBodyWinIDs lists a party's member IDs in order.
func fallenBodyWinIDs(party []mapload.PartyMember) []string {
	var ids []string
	for _, p := range party {
		ids = append(ids, p.ID)
	}
	return ids
}

// The script wins with the whole party standing. Continue closes the panel,
// and the hero's blows take companion 3 to 0, then companion 2 and the villager
// who joined the party to -10 while their dying time runs, so Victory ends the
// mission with both bodies lying at -10. The mission-end cull raises 3 and
// leaves the two bodies (PARTY-CULL-004): the town party and the next mission
// hold the hero and 3 only, each with what he or she ended the mission with.
func TestAMemberAtMinusTenAtTheWinDoesNotCrossToTheNextMission(t *testing.T) {
	ms, mw, table := fallenBodyWinMission(t)
	for tick := 0; tick < 64 && len(mw.mission.party) < 4; tick++ {
		mw.tick()
	}
	if got := fallenBodyWinIDs(mw.mission.party); !slices.Equal(got, []string{"hero", "first", "second", "join:4"}) || mw.mission.ids[3] != 4 {
		t.Fatalf("setup: live party %v ids %v, want the villager joined as the fourth member", got, mw.mission.ids)
	}
	mw.orderPickup(1, 30, 32)
	for tick := 0; tick < 400 && mw.pickup.set; tick++ {
		mw.tick()
	}
	mw.orderPickup(3, 41, 45)
	for tick := 0; tick < 400 && mw.pickup.set; tick++ {
		mw.tick()
	}
	mw.enqueue(3, 41, 42)
	for tick := 0; tick < 400; tick++ {
		if e, _ := mw.entity(3); e.X == 41 && e.Y == 42 {
			break
		}
		mw.tick()
	}
	heroPack, _ := ms.World.Carried(1)
	secondPack, _ := ms.World.Carried(3)
	if !slices.Contains(heroPack, eqMaceCode) || !slices.Contains(secondPack, eqBowCode) {
		t.Fatalf("setup: hero carries %v, companion 3 carries %v: want the mace and the bow picked up", heroPack, secondPack)
	}

	mw.enqueue(1, 40, 41)
	for tick := 0; tick < 800 && !mw.mission.announced; tick++ {
		mw.tick()
	}
	if mw.mission.outcome != sim.OutcomeWon || !mw.mission.open {
		t.Fatalf("outcome %v, panel open=%v: want the script's win on the success panel", mw.mission.outcome, mw.mission.open)
	}
	if dest, _, _ := mw.advanceNotice(ui.NoticeContinue); dest != ui.NoticeStay || mw.mission.open {
		t.Fatalf("Continue went to %v with the panel open=%v", dest, mw.mission.open)
	}
	mw.enqueue(1, 40, 41)
	for tick := 0; tick < 400; tick++ {
		if e, _ := mw.entity(1); e.X == 40 && e.Y == 41 {
			break
		}
		mw.tick()
	}
	second := fallenBodyWinStrike(t, mw, 3)
	first := fallenBodyWinStrike(t, mw, 2)
	joined := fallenBodyWinStrike(t, mw, 4)
	first, _ = mw.entity(2)
	if second.HP != 0 || !second.Restorable() || first.HP != -10 || !first.Dying() || joined.HP != -10 || !joined.Dying() {
		t.Fatalf("bodies at 3 health %d restorable=%v, 2 health %d dying=%v, villager health %d dying=%v: "+
			"want 3 at 0 and both others at -10 inside their dying time",
			second.HP, second.Restorable(), first.HP, first.Dying(), joined.HP, joined.Dying())
	}
	if mw.mission.outcome != sim.OutcomeWon {
		t.Fatalf("outcome %v after the blows, want the win to stand while the bodies lie inside their dying time", mw.mission.outcome)
	}
	hero, _ := mw.entity(1)
	heroPack, _ = ms.World.Carried(1)
	secondPack, _ = ms.World.Carried(3)

	f := continuityFront(t)
	f.Table = table
	f.Archives, f.Tiles = &Archives{Containers: missionArchive(t, 30)}, &terrain.Tileset{}
	if dest, msg, _ := f.continuity(20, ms, mw.advanceNotice)(ui.NoticeVictory); dest != ui.NoticeToTown || !f.Town.Open() {
		t.Fatalf("Victory went to %v (%q), town open=%v: want the town", dest, msg, f.Town.Open())
	}
	if got := fallenBodyWinIDs(f.Carried); !slices.Equal(got, []string{"hero", "second"}) {
		t.Errorf("town party %v, want the hero and companion 3 without the two bodies at -10", got)
	}
	next, err := StartMission(f.Archives.Containers, f.Town.Chapter(), f.Table, openDifficulty, f.Carried)
	if err != nil {
		t.Fatalf("StartMission(%d): %v", f.Town.Chapter(), err)
	}
	if got := fallenBodyWinIDs(next.Party); !slices.Equal(got, []string{"hero", "second"}) || len(next.Start.IDs) != 2 {
		t.Fatalf("mission %d party %v ids %v, want the hero and companion 3 only", f.Town.Chapter(), got, next.Start.IDs)
	}
	if c := f.Carried[0].Carry; c == nil || c.SkillXP != hero.SkillXP || !slices.Equal(c.Items, heroPack) {
		t.Fatalf("town hero carries %+v, want his experience %v and pack %v", c, hero.SkillXP, heroPack)
	}
	if c := f.Carried[1].Carry; c == nil || !slices.Equal(c.Items, secondPack) {
		t.Fatalf("town companion 3 carries %+v, want the pack %v she ended the mission with", c, secondPack)
	}
	for _, id := range []sim.EntityID{2, 4} {
		if e, _ := mw.entity(id); e.Alive() || e.HP != -10 {
			t.Fatalf("the cull moved body %d to health %d, want it left lying at -10", id, e.HP)
		}
	}
	if raised, _ := mw.entity(3); !raised.Alive() || raised.HP != raised.MaxHP {
		t.Fatalf("the cull left companion 3 at health %d/%d, want her raised at full health", raised.HP, raised.MaxHP)
	}

	owned := 0
	for _, e := range next.World.Entities() {
		if e.Owner == sim.SelfSlot {
			owned++
		}
	}
	companion, _ := next.World.Entity(next.Start.IDs[1])
	companionPack, _ := next.World.Carried(companion.ID)
	if owned != 2 || !companion.Alive() || companion.HP != companion.MaxHP || !slices.Contains(companionPack, eqBowCode) {
		t.Fatalf("mission %d opens %d player actors, companion 3 at health %d/%d carrying %v: want two, her at full health with the bow",
			f.Town.Chapter(), owned, companion.HP, companion.MaxHP, companionPack)
	}
}
