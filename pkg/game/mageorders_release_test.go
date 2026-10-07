package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// mageOrdersArena is the ground a mage's orders are measured on: mission 20's
// own terrain, spell rules and two of its hostile creatures, with the generated
// mage of the new-game screen. The mage stands three cells short of the nearer
// creature and five short of the farther one, on one row. Placement, the
// creatures' health and their relation to the participant are fixtures: the
// creatures hold the participant as a locked ally, so they never strike, never
// turn hostile and never walk, and every change in their health is the mage's
// doing. Terrain, the mage's own statistics, spells and every order the fight
// runs on are the install's and the production input path's.
type mageOrdersArena struct {
	live            *mapWorld
	app             *ui.App
	mage, near, far sim.EntityID
}

const (
	mageFireArrow = 1
	mageShield    = 18
	arenaHealth   = 3000
)

func openMageOrdersArena(t *testing.T) *mageOrdersArena {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Mage orders", Choices: []int{0, 1, 0}, Stats: []int{30, 30, 40, 40}})
	a := f.App("mage orders")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	mage, ok := live.entity(live.mission.ids[0])
	if !ok {
		t.Fatal("no party actor")
	}
	if mage.WeaponSpell != mageFireArrow || mage.KnownSpells&(1<<mageFireArrow) == 0 || mage.KnownSpells&(1<<mageShield) == 0 {
		t.Fatalf("the generated mage no longer fires and knows Fire Arrow and Shield: weapon spell %d, book %#x", mage.WeaponSpell, mage.KnownSpells)
	}
	var trio []sim.Entity
	for _, e := range live.world.Entities() {
		if e.Owner == 2 && e.Group == 1 {
			trio = append(trio, e)
		}
	}
	if len(trio) != 3 || trio[0].ID != 0 || trio[2].ID != 2 {
		t.Fatalf("mission 20's first hostile group changed: %d entities", len(trio))
	}
	rel := live.world.Relations()
	rel.Set(sim.SelfSlot, 2, 1)
	rel.Set(2, sim.SelfSlot, 2)
	if !rel.Hostile(sim.SelfSlot, 2) || rel.Hostile(2, sim.SelfSlot) {
		t.Fatal("the arena's relations did not take")
	}

	m := live.mission.state.Map
	terrain := sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)}
	bounds := live.world.Bounds()
	px, py, found := openGroundSquare(terrain.Block, int(bounds.Width), int(bounds.Height), 15)
	if !found {
		t.Fatal("mission 20 has no open ground square of 15 cells")
	}
	cx, cy := int32(px+7), int32(py+7)
	place := func(e sim.Entity, x, y int32) sim.Entity {
		e.X, e.Y, e.PostX, e.PostY = x, y, x, y
		e.TargetX, e.TargetY, e.HasTarget = 0, 0, false
		e.Transit, e.TransitTotal = 0, 0
		return e
	}
	mage = place(mage, cx-2, cy)
	near := place(trio[0], cx+1, cy)
	far := place(trio[2], cx+3, cy)
	far.HP, far.MaxHP = arenaHealth, arenaHealth
	near.HP, near.MaxHP = arenaHealth, arenaHealth
	if int32(mage.ScanRange) < 5 {
		t.Fatalf("the mage sees %d cells, less than the farther creature's five", mage.ScanRange)
	}
	worn, _ := live.world.EquippedItems(mage.ID)
	arena, err := sim.NewStructuredWorld(2020, bounds, sim.ModeCanonical, terrain,
		[]sim.Entity{far, near, mage}, nil, rel, nil,
		[]sim.Stock{{ID: mage.ID, EquippedItems: worn, ItemInstances: party[0].CarriedItems}},
		mapload.SpellRules(f.Table), sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	codes := make([]uint16, 0)
	for _, item := range worn {
		if item.Code != 0 {
			codes = append(codes, item.Code)
		}
	}
	for _, item := range party[0].CarriedItems {
		codes = append(codes, item.Code)
	}
	mapload.DeclareCodeWeights(arena, f.Table, codes)
	mapload.BindSourceDerive(arena)
	live.world = arena
	candidate := *live.mission.state
	candidate.World, candidate.savedDocument = arena, nil
	candidate.ActorManifest = &SnapshotActorManifest{Version: actorManifestVersion}
	if live.mission.state.ActorManifest != nil {
		for _, row := range live.mission.state.ActorManifest.Actors {
			if row.ID == mage.ID || row.ID <= 2 {
				candidate.ActorManifest.Actors = append(candidate.ActorManifest.Actors, row)
			}
		}
	}
	live.mission.state = &candidate
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	inspectionCentre(live, int(cx), int(cy))
	live.push()
	if err := a.HeadlessSelectEntity(uint32(mage.ID)); err != nil {
		t.Fatal(err)
	}
	live.push()
	return &mageOrdersArena{live: live, app: a, mage: mage.ID, near: near.ID, far: far.ID}
}

func (arena *mageOrdersArena) health(id sim.EntityID) int32 {
	e, _ := arena.live.entity(id)
	return e.HP
}

func (arena *mageOrdersArena) click(t *testing.T, x, y int) {
	t.Helper()
	for _, edge := range []string{"press", "release"} {
		if err := arena.app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

// openBook shows the spell strip if it does not list the spell.
func (arena *mageOrdersArena) openBook(t *testing.T, spell uint32) {
	t.Helper()
	if _, _, err := arena.app.HeadlessSpellPoint(spell); err != nil {
		if err := arena.app.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
}

// orderAttack arms the attack cursor and clicks the creature, and returns the
// one command the click queued.
func (arena *mageOrdersArena) orderAttack(t *testing.T, victim sim.EntityID) {
	t.Helper()
	if err := arena.app.HeadlessKey("attack"); err != nil {
		t.Fatal(err)
	}
	x, y, err := arena.app.HeadlessEntityPoint(uint32(victim))
	if err != nil {
		t.Fatal(err)
	}
	arena.click(t, x, y)
	if got := arena.live.pending; len(got) != 1 || got[0].Kind != sim.KindAttack || got[0].Entity != arena.mage ||
		got[0].X != int32(victim) {
		t.Fatalf("the click queued %+v, want the mage's attack on creature %d", got, victim)
	}
}

// selfCast picks Shield in the spell strip and clicks the mage.
func (arena *mageOrdersArena) selfCast(t *testing.T) {
	t.Helper()
	arena.openBook(t, mageShield)
	sx, sy, err := arena.app.HeadlessSpellPoint(mageShield)
	if err != nil {
		t.Fatal(err)
	}
	arena.click(t, sx, sy)
	if _, current, armed := arena.live.view.QuickSpellState(); current != mageShield || !armed {
		t.Fatalf("the strip click left spell %d armed %v, want Shield armed", current, armed)
	}
	mx, my, err := arena.app.HeadlessEntityPoint(uint32(arena.mage))
	if err != nil {
		t.Fatal(err)
	}
	arena.click(t, mx, my)
	if got := arena.live.pending; len(got) != 1 || got[0].Kind != sim.KindCast || got[0].Entity != arena.mage ||
		got[0].X != int32(arena.mage) || got[0].Y != mageShield {
		t.Fatalf("the click queued %+v, want the mage's Shield on itself", got)
	}
}

// holds fails the test unless the mage holds the creature as its victim.
func (arena *mageOrdersArena) holds(t *testing.T, victim sim.EntityID, what string) {
	t.Helper()
	m, _ := arena.live.entity(arena.mage)
	if !m.HasAttackTarget || m.AttackTargetKind != sim.AttackTargetUnit || m.AttackTarget != victim {
		t.Fatalf("tick %d, %s: the mage holds victim %v/%d, want creature %d", arena.live.world.Tick(), what,
			m.HasAttackTarget, m.AttackTarget, victim)
	}
}

// step advances one tick and reports which creatures lost health in it.
func (arena *mageOrdersArena) step() (nearHit, farHit bool) {
	near, far := arena.health(arena.near), arena.health(arena.far)
	arena.live.tick()
	return arena.health(arena.near) < near, arena.health(arena.far) < far
}

// run counts hits while requiring the farther victim after every tick.
func (arena *mageOrdersArena) run(t *testing.T, n int, what string) (nearHits, farHits int) {
	t.Helper()
	for range n {
		nearHit, farHit := arena.step()
		if nearHit {
			nearHits++
		}
		if farHit {
			farHits++
		}
		arena.holds(t, arena.far, what)
	}
	return nearHits, farHits
}

// Tester item: a mage ordered onto an enemy a little farther than another
// attacked the nearer one and ignored the order, and a self-cast order given
// during an attack made it stand where it was, where the original's mage
// interrupted the attack, cast and went on attacking.
//
// The mage is selected, ordered with the attack key and a click on the farther
// creature, and later given Shield through the spell strip and a click on
// itself, all through App input. The creatures stand still, so a creature
// losing health is what the mage struck. The armed run has Fire Arrow set to
// autocast by a right-click on its strip cell, the way a player arms it: an
// armed row fired at the nearest creature and kept the mage in a cast nearly
// every tick, which is when an attack order was dropped.
func TestReleaseAMageKeepsItsOrderedEnemyAndResumesAfterASelfCast(t *testing.T) {
	for _, tc := range []struct {
		name  string
		armed bool
	}{{"weapon spell alone", false}, {"Fire Arrow armed for autocast", true}} {
		t.Run(tc.name, func(t *testing.T) {
			arena := openMageOrdersArena(t)
			live, a := arena.live, arena.app
			if tc.armed {
				arena.openBook(t, mageFireArrow)
				x, y, err := a.HeadlessSpellPoint(mageFireArrow)
				if err != nil {
					t.Fatal(err)
				}
				if err := a.HeadlessPointer("right-press", x, y); err != nil {
					t.Fatal(err)
				}
				if got := live.pending; len(got) != 1 || got[0].Kind != sim.KindAutocast || got[0].Entity != arena.mage ||
					got[0].X != mageFireArrow {
					t.Fatalf("the right-click queued %+v, want Fire Arrow autocast on the mage", got)
				}
				if err := a.HeadlessPointer("right-release", x, y); err != nil {
					t.Fatal(err)
				}
				live.tick()
				if e, _ := live.entity(arena.mage); e.AutoSpell != mageFireArrow {
					t.Fatalf("autocast is %d after the toggle, want Fire Arrow", e.AutoSpell)
				}
			}
			nearFirst, farFirst := 0, 0
			for range 120 {
				nearHit, farHit := arena.step()
				if nearHit {
					nearFirst++
				}
				if farHit {
					farFirst++
				}
			}
			t.Logf("with no order the mage struck the nearer creature on %d ticks and the farther on %d", nearFirst, farFirst)

			arena.orderAttack(t, arena.far)
			old, _ := live.entity(arena.mage)
			if !tc.armed && (!old.HasAttackTarget || old.AttackTarget != arena.near || old.AttackPhase != sim.AttackCasting) {
				t.Fatalf("the weapon-only control has no loaded cast on the old victim: %+v", old)
			}
			oldHealth := arena.health(arena.near)
			oldHits := 0
			transferred, queued := false, false
			relaxed, boundaryOne, boundaryTwo := false, false, false
			for waited := 0; waited < 120; waited++ {
				nearHit, farHit := arena.step()
				if nearHit {
					oldHits++
				}
				mage, _ := live.entity(arena.mage)
				if id, kind, ok := mage.RequestedAttackTarget(); !ok || id != arena.far || kind != sim.AttackTargetUnit {
					t.Fatalf("tick %d: the admitted attack request changed to %v/%d/%d", live.world.Tick(), ok, id, kind)
				}
				if mage.HasPendingAttackTarget {
					queued = true
					relaxed = relaxed || mage.AttackPhase == sim.AttackRelaxing
					boundaryOne = boundaryOne || mage.AttackPhase == sim.AttackBoundaryOne
					boundaryTwo = boundaryTwo || mage.AttackPhase == sim.AttackBoundaryTwo
					if !mage.HasAttackTarget || mage.AttackTarget != old.AttackTarget || mage.AttackTargetKind != old.AttackTargetKind ||
						mage.PendingAttackTarget != arena.far || mage.PendingAttackTargetKind != sim.AttackTargetUnit {
						t.Fatalf("tick %d: pending attack lost its active or requested victim: %+v", live.world.Tick(), mage)
					}
					if farHit {
						t.Fatal("the requested victim was struck before the loaded cycle completed")
					}
					continue
				}
				arena.holds(t, arena.far, "after the loaded cycle")
				transferred = true
				t.Logf("requested victim became active after %d ordinary ticks; queued %v, old phase %d, old victim health %d -> %d",
					waited+1, queued, old.AttackPhase, oldHealth, arena.health(arena.near))
				break
			}
			if !transferred {
				t.Fatal("the admitted requested victim did not become active within 120 ordinary ticks")
			}
			if (old.AttackPhase == sim.AttackCharging || old.AttackPhase == sim.AttackCasting) &&
				(!queued || oldHits != 1 || !relaxed || !boundaryOne || !boundaryTwo) {
				t.Fatalf("old cycle: queued %v, hits %d, recovery %v, boundaries %v/%v; want queued, one strike and all boundaries",
					queued, oldHits, relaxed, boundaryOne, boundaryTwo)
			}
			nearHits, farHits := arena.run(t, 300, "after the order")
			mage, _ := live.entity(arena.mage)
			t.Logf("ordered onto the farther creature: struck the nearer on %d ticks, the farther on %d; mana %d/%d",
				nearHits, farHits, mage.Mana, mage.MaxMana)
			if nearHits != 0 {
				t.Fatalf("the mage ordered onto the farther creature struck the nearer one on %d ticks", nearHits)
			}
			if farHits < 3 {
				t.Fatalf("the creature the mage was ordered onto was struck on %d ticks in 300", farHits)
			}

			before, _ := live.entity(arena.mage)
			arena.selfCast(t)
			paid, applied := false, false
			nearAfter, farBefore, farAfter := 0, 0, 0
			for range 300 {
				nearHit, farHit := arena.step()
				arena.holds(t, arena.far, "during and after the self-cast")
				if e, _ := live.entity(arena.mage); e.Mana < before.Mana {
					paid = true
				}
				if nearHit {
					nearAfter++
				}
				switch {
				case applied && farHit:
					farAfter++
				case !applied && farHit:
					farBefore++
				}
				applied = applied || live.world.HasEffectSpell(arena.mage, mageShield)
			}
			mage, _ = live.entity(arena.mage)
			t.Logf("self-cast: paid %v, applied %v; struck the nearer on %d ticks, the farther on %d before the effect and %d after; mana %d/%d",
				paid, applied, nearAfter, farBefore, farAfter, mage.Mana, mage.MaxMana)
			if !paid || !applied {
				t.Fatalf("the self-cast paid %v and applied %v", paid, applied)
			}
			if nearAfter != 0 {
				t.Fatalf("around the self-cast the mage struck the nearer creature on %d ticks", nearAfter)
			}
			if farAfter < 3 {
				t.Fatalf("after the self-cast the mage struck the creature it was ordered onto on %d ticks in the remainder", farAfter)
			}
		})
	}
}
