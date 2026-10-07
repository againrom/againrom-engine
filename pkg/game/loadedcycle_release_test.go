package game

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// loadedCycleArena is the ground a warrior's loaded attack cycle is interrupted
// on: mission 20's own terrain and two of its hostile creatures, with one or two
// generated warriors of the new-game screen. The first warrior stands between
// the creatures, one cell from each, so either is in reach of his blow. A second
// warrior stands beyond the creatures' guard radius, so he holds no victim until
// he is ordered.
//
// Placement, the creatures' health, their relation to the participant and their
// defence are fixtures: the creatures hold the participant as a locked ally, so
// they never strike, never turn hostile and never walk, and their defence and
// absorption are zeroed so a blow that reaches one takes health. Terrain, the
// warriors' own statistics and every order the fight runs on are the install's
// and the production input path's.
type loadedCycleArena struct {
	front       *FrontEnd
	live        *mapWorld
	app         *ui.App
	heroes      []sim.EntityID
	east, south sim.EntityID
	cx, cy      int32
}

// loadedCycleTerrainHook, when set, edits the arena's terrain copy before the
// arena world is built.
var loadedCycleTerrainHook func(block []byte, bw, bh int, cx, cy int32)

func openLoadedCycleArena(t *testing.T, heroCount int) *loadedCycleArena {
	t.Helper()
	return openLoadedCycleArenaBeside(t, heroCount, -1)
}

// openLoadedCycleArenaBeside is openLoadedCycleArena on the open ground nearest
// to the mission's structure with the given id, with the mission's structures
// standing in the arena. A negative id is the plain arena: the first open
// ground and no structure.
func openLoadedCycleArenaBeside(t *testing.T, heroCount, structure int) *loadedCycleArena {
	return openLoadedCycleArenaChoices(t, heroCount, structure, []int{0, 0, 0})
}

func openLoadedCycleArenaChoices(t *testing.T, heroCount, structure int, choices []int) *loadedCycleArena {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Loaded cycle", Choices: choices, Stats: []int{40, 30, 30, 30}})
	if choices[chargenChoiceClass] == 1 {
		party[0].KnownSpells |= 1 << 26
		party[0].Book = sim.Spellbook{State: sim.BookPresent}
		for _, rule := range mapload.SpellRules(f.Table) {
			if party[0].KnownSpells&(1<<rule.ID) != 0 && rule.ID >= 1 && rule.ID <= 28 {
				defensive := uint8(0)
				if rule.Defensive {
					defensive = 1
				}
				party[0].Book.Slots[rule.ID-1] = sim.BookSpell{Range: rule.MaxRange, Defensive: defensive, ManaCost: uint16(rule.ManaCost)}
			}
		}
	}
	for len(party) < heroCount {
		next := mapload.CloneParty(party[:1])[0]
		next.ID = fmt.Sprintf("companion-%d", len(party))
		party = append(party, next)
	}
	a := f.App("loaded cycle")
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
	if len(live.mission.ids) < heroCount {
		t.Fatalf("mission 20 opened %d party actors, want %d", len(live.mission.ids), heroCount)
	}
	heroes := make([]sim.Entity, heroCount)
	for i := range heroes {
		e, ok := live.entity(live.mission.ids[i])
		if !ok {
			t.Fatalf("no party actor %d", i)
		}
		heroes[i] = e
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
	var structures []sim.Structure
	px, py, found := openGroundSquare(terrain.Block, int(bounds.Width), int(bounds.Height), 15)
	if structure >= 0 {
		structures = live.world.Structures()
		at := slices.IndexFunc(structures, func(s sim.Structure) bool { return int(s.ID) == structure })
		if at < 0 {
			t.Fatalf("mission 20 has no structure %d", structure)
		}
		px, py, found = openGroundSquareBeside(terrain.Block, int(bounds.Width), int(bounds.Height), 15, structures[at])
	}
	if !found {
		t.Fatal("mission 20 has no open ground square of 15 cells")
	}
	cx, cy := int32(px+7), int32(py+7)
	if loadedCycleTerrainHook != nil {
		terrain.Block = slices.Clone(terrain.Block)
		loadedCycleTerrainHook(terrain.Block, int(bounds.Width), int(bounds.Height), cx, cy)
	}
	place := func(e sim.Entity, x, y int32) sim.Entity {
		e.X, e.Y, e.PostX, e.PostY = x, y, x, y
		e.TargetX, e.TargetY, e.HasTarget = 0, 0, false
		e.Transit, e.TransitTotal = 0, 0
		return e
	}
	east := place(trio[0], cx+1, cy)
	south := place(trio[2], cx, cy+1)
	for _, c := range []*sim.Entity{&east, &south} {
		c.HP, c.MaxHP = arenaHealth, arenaHealth
		c.Defence, c.Absorption, c.Resistance = -1000, 0, [5]uint8{}
	}
	heroes[0] = place(heroes[0], cx, cy)
	if heroCount > 1 {
		heroes[1] = place(heroes[1], cx, cy-6)
	}

	entities := []sim.Entity{east, south}
	stocks := make([]sim.Stock, 0, heroCount)
	var codes []uint16
	for i := range heroes {
		worn, _ := live.world.EquippedItems(heroes[i].ID)
		for slot := range worn {
			worn[slot].ObjectID = 0
		}
		entities = append(entities, heroes[i])
		stocks = append(stocks, sim.Stock{ID: heroes[i].ID, EquippedItems: worn, ItemInstances: party[i].CarriedItems})
		for _, item := range worn {
			if item.Code != 0 {
				codes = append(codes, item.Code)
			}
		}
		for _, item := range party[i].CarriedItems {
			codes = append(codes, item.Code)
		}
	}
	arena, err := sim.NewStructuredWorld(2020, bounds, sim.ModeCanonical, terrain, entities, nil, rel, nil, stocks,
		mapload.SpellRules(f.Table), sim.GhostTemplate{}, structures)
	if err != nil {
		t.Fatal(err)
	}
	mapload.DeclareCodeWeights(arena, f.Table, codes)
	mapload.BindSourceDerive(arena)
	live.world = arena
	candidate := *live.mission.state
	candidate.World, candidate.savedDocument = arena, nil
	candidate.ActorManifest = &SnapshotActorManifest{Version: actorManifestVersion}
	if live.mission.state.ActorManifest != nil {
		for _, row := range live.mission.state.ActorManifest.Actors {
			keep := row.ID <= 2
			for _, h := range heroes {
				keep = keep || row.ID == h.ID
			}
			if keep {
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
	ids := make([]sim.EntityID, heroCount)
	for i := range heroes {
		ids[i] = heroes[i].ID
	}
	return &loadedCycleArena{front: f, live: live, app: a, heroes: ids, east: east.ID, south: south.ID, cx: cx, cy: cy}
}

func (arena *loadedCycleArena) now() int64 { return int64(arena.live.world.Tick()) }

func (arena *loadedCycleArena) get(id sim.EntityID) sim.Entity {
	e, _ := arena.live.entity(id)
	return e
}

func (arena *loadedCycleArena) tap(t *testing.T, x, y int) {
	t.Helper()
	for _, edge := range []string{"press", "release"} {
		if err := arena.app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

// selectFirstHero selects the first hero with a click on him.
func (arena *loadedCycleArena) selectFirstHero(t *testing.T) {
	t.Helper()
	if err := arena.app.HeadlessSelectEntity(uint32(arena.heroes[0])); err != nil {
		t.Fatal(err)
	}
	if got := arena.app.HeadlessSelection(); !slices.Equal(got, []uint32{uint32(arena.heroes[0])}) {
		t.Fatalf("the selection is %v, want the first warrior alone", got)
	}
}

// orderAttack arms the attack cursor and clicks the creature with the current
// selection standing, and requires the one command the click queued. It costs
// three frames, and every frame advances the world by one tick.
func (arena *loadedCycleArena) orderAttack(t *testing.T, victim sim.EntityID) {
	t.Helper()
	if err := arena.app.HeadlessKey("attack"); err != nil {
		t.Fatal(err)
	}
	x, y, err := arena.app.HeadlessEntityPoint(uint32(victim))
	if err != nil {
		t.Fatal(err)
	}
	arena.tap(t, x, y)
	if got := arena.live.pending; len(got) != 1 || got[0].Kind != sim.KindAttack || got[0].Entity != arena.heroes[0] ||
		got[0].X != int32(victim) {
		t.Fatalf("the click queued %+v, want the warrior's attack on creature %d", got, victim)
	}
}

// clickGround clicks the ground cell with the current selection standing and
// the current cursor armed, and returns the commands the click queued. The pixel
// is the first in the middle of the view the production hit test resolves to the
// cell. It costs two frames.
func (arena *loadedCycleArena) clickGround(t *testing.T, col, row int32) []sim.Command {
	t.Helper()
	before := len(arena.live.pending)
	for py := 100; py < 560; py += 4 {
		for px := 160; px < 750; px += 4 {
			c, r, err := arena.app.HeadlessDropCell(px, py)
			if err != nil || int32(c) != col || int32(r) != row {
				continue
			}
			arena.tap(t, px, py)
			return slices.Clone(arena.live.pending[before:])
		}
	}
	t.Fatalf("cell (%d,%d) has no visible ground pointer witness", col, row)
	return nil
}

// orderMove clicks the ground cell with the current selection standing and
// returns the commands the click queued, which must be moves to the cell.
func (arena *loadedCycleArena) orderMove(t *testing.T, col, row int32) []sim.Command {
	t.Helper()
	got := arena.clickGround(t, col, row)
	for _, cmd := range got {
		if cmd.Kind != sim.KindGroupMoveTo || cmd.X != col || cmd.Y != row {
			t.Fatalf("the ground click queued %+v, want moves to (%d,%d)", got, col, row)
		}
	}
	return got
}

// untilLoaded advances until the hero is in the middle of a charge on the
// creature.
func (arena *loadedCycleArena) untilLoaded(t *testing.T, hero, victim sim.EntityID) {
	t.Helper()
	for range 120 {
		arena.live.tick()
		if h := arena.get(hero); h.HasAttackTarget && h.AttackTarget == victim && h.AttackPhase == sim.AttackCharging {
			if h.AttackCountdown < 5 {
				t.Fatalf("the charge has %d ticks left when first seen, too few to order into", h.AttackCountdown)
			}
			return
		}
	}
	t.Fatalf("the warrior never began a charge on creature %d in 120 ticks", victim)
}

// loadedOnLanding requires the hero to be in the middle of a charge on the
// creature now, which is when a command queued by the last click lands: on the
// next tick.
func (arena *loadedCycleArena) loadedOnLanding(t *testing.T, hero, victim sim.EntityID) {
	t.Helper()
	h := arena.get(hero)
	t.Logf("the second order lands with the warrior in phase %d, %d ticks left, victim %v/%d", h.AttackPhase, h.AttackCountdown,
		h.HasAttackTarget, h.AttackTarget)
	if h.AttackPhase != sim.AttackCharging || h.AttackCountdown < 1 || !h.HasAttackTarget || h.AttackTarget != victim {
		t.Fatalf("the warrior is not loaded on creature %d when the second order lands: phase %d, %d ticks left, victim %v/%d",
			victim, h.AttackPhase, h.AttackCountdown, h.HasAttackTarget, h.AttackTarget)
	}
}

// reload saves the mission as it stands, restores the saved game on a second
// front end through the ordinary load path, and returns the restored mission
// beside a control world decoded from the saved world bytes.
func (arena *loadedCycleArena) reload(t *testing.T) (*mapWorld, *sim.World) {
	t.Helper()
	snapshot, label, err := arena.front.Snapshot(true)
	if err != nil {
		t.Fatalf("the save was refused: %v", err)
	}
	raw, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatalf("the save was refused: %v", err)
	}
	decoded, _, err := DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	back := releaseFront(t)
	opener, town, err := back.Restore(decoded)
	if err != nil || town || opener == nil {
		t.Fatal("restore", err, town)
	}
	app := back.App("loaded cycle restore")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	var control sim.World
	if err := control.UnmarshalBinary(snapshot.World); err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(&control)
	return back.live, &control
}

func cellDistance(e sim.Entity, x, y int32) int32 {
	return max(absCell(e.X-x), absCell(e.Y-y))
}

func absCell(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// openGroundSquareBeside returns the top-left cell of the size-by-size square of
// the block plane with no closed cell whose centre is nearest the structure,
// among those whose centre stands at least nine cells from it, so that a unit
// in the middle of the square has ground to cross before it can use the
// structure.
func openGroundSquareBeside(block []byte, width, height, size int, s sim.Structure) (int, int, bool) {
	bestX, bestY, best := 0, 0, -1
	for y := 0; y+size <= height; y++ {
		for x := 0; x+size <= width; x++ {
			open := true
			for dy := 0; dy < size && open; dy++ {
				for dx := 0; dx < size; dx++ {
					if block[(y+dy)*width+x+dx] != 0 {
						open = false
						break
					}
				}
			}
			if !open {
				continue
			}
			d := int(max(absCell(int32(x+size/2)-s.Col), absCell(int32(y+size/2)-s.Row)))
			if d >= 9 && (best < 0 || d < best) {
				bestX, bestY, best = x, y, d
			}
		}
	}
	return bestX, bestY, best >= 0
}

// blowOutcome is what the ticks after an order showed of the creature's health
// and of the warrior's place.
type blowOutcome struct {
	strikeTick int64 // the tick the charge ended, or -1
	lost       int32 // health the creature lost in the strike tick
	leftTick   int64 // the first tick the warrior stood on another cell, or -1
}

// watch advances until the warrior has left his cell or n ticks have passed, and
// records the tick his charge ended and what the creature lost to it.
func (arena *loadedCycleArena) watch(hero, victim sim.EntityID, n int) blowOutcome {
	out := blowOutcome{strikeTick: -1, leftTick: -1}
	start := arena.get(hero)
	for range n {
		before, hp := arena.get(hero), arena.get(victim).HP
		arena.live.tick()
		now := arena.get(hero)
		if out.strikeTick < 0 && before.AttackPhase == sim.AttackCharging && now.AttackPhase != sim.AttackCharging {
			out.strikeTick, out.lost = arena.now(), hp-arena.get(victim).HP
		}
		if now.X != start.X || now.Y != start.Y {
			out.leftTick = arena.now()
			return out
		}
	}
	return out
}

// A unit told to walk while its blow was on the way gave the blow up and walked
// at once. The original finishes the blow it began and then walks
// (AI-ORDER-039, HERO-CADENCE-112, DIV-1563). A different victim waits beside
// the active victim until the loaded physical cycle finishes.
//
// The warrior is selected, ordered onto a creature with the attack key and a
// click, and ordered again in the middle of his charge, all through App input.
// The charge lasts seven ticks and every App frame advances the world by one, so
// each second order is given with the selection already standing and lands with
// at least one tick of the charge left. The creature never strikes back and its
// health is what the warrior's blow takes, so the blow either landed before he
// left or it did not. One subtest saves the mission with the move waiting and
// loads it on a second front end: the save is written, the loaded warrior holds
// the same pair and the loaded world stays equal to the saved world bytes for
// 120 ticks.
func TestReleaseALoadedAttackCycleFinishesBeforeAMoveOrder(t *testing.T) {
	t.Run("move: the blow lands and then the warrior walks", func(t *testing.T) {
		arena := openLoadedCycleArena(t, 1)
		hero, east := arena.heroes[0], arena.east
		arena.selectFirstHero(t)
		arena.orderAttack(t, east)
		arena.untilLoaded(t, hero, east)
		start := arena.get(hero)
		full := arena.get(east).HP
		goalX, goalY := arena.cx-4, arena.cy-3
		if cmds := arena.orderMove(t, goalX, goalY); len(cmds) != 1 || cmds[0].Entity != hero {
			t.Fatalf("the ground click queued %+v, want one move for the warrior", cmds)
		}
		arena.loadedOnLanding(t, hero, east)
		arena.live.tick()
		h := arena.get(hero)
		t.Logf("after the move order: victim %v/%d phase %d countdown %d destination %v (%d,%d) at (%d,%d)",
			h.HasAttackTarget, h.AttackTarget, h.AttackPhase, h.AttackCountdown, h.HasTarget, h.TargetX, h.TargetY, h.X, h.Y)
		if !h.HasAttackTarget || h.AttackTarget != east || h.AttackPhase != sim.AttackCharging {
			t.Fatalf("the move order dropped the loaded cycle: victim %v/%d phase %d", h.HasAttackTarget, h.AttackTarget, h.AttackPhase)
		}
		if !h.HasTarget || h.TargetX != goalX || h.TargetY != goalY {
			t.Fatalf("the destination is %v (%d,%d), want (%d,%d)", h.HasTarget, h.TargetX, h.TargetY, goalX, goalY)
		}
		out := arena.watch(hero, east, 400)
		t.Logf("strike at tick %d took %d health; the warrior left his cell at tick %d", out.strikeTick, out.lost, out.leftTick)
		if out.strikeTick < 0 || out.leftTick < 0 || out.strikeTick >= out.leftTick {
			t.Fatalf("the warrior left at tick %d and the blow resolved at tick %d, want the blow first", out.leftTick, out.strikeTick)
		}
		if out.lost <= 0 || arena.get(east).HP >= full {
			t.Fatalf("the blow took %d health from the creature it was loaded on", out.lost)
		}
		hp := arena.get(east).HP
		for range 400 {
			arena.live.tick()
			if cellDistance(arena.get(hero), goalX, goalY) <= 1 {
				break
			}
		}
		end := arena.get(hero)
		if cellDistance(end, goalX, goalY) > 1 || end.X == start.X && end.Y == start.Y {
			t.Fatalf("the warrior stands at (%d,%d), want him on the clicked cell (%d,%d)", end.X, end.Y, goalX, goalY)
		}
		if got := arena.get(east).HP; got < hp {
			t.Fatalf("the creature lost health after the warrior left: %d to %d", hp, got)
		}
		t.Logf("the warrior walked from (%d,%d) to (%d,%d) at tick %d", start.X, start.Y, end.X, end.Y, arena.now())
	})

	t.Run("formation move: the loaded warrior finishes his blow and the other walks at once", func(t *testing.T) {
		arena := openLoadedCycleArena(t, 2)
		loaded, idle, east := arena.heroes[0], arena.heroes[1], arena.east
		arena.selectFirstHero(t)
		arena.orderAttack(t, east)
		arena.untilLoaded(t, loaded, east)
		loadedStart, idleStart := arena.get(loaded), arena.get(idle)
		full := arena.get(east).HP
		if h := arena.get(idle); h.HasAttackTarget || h.HasTarget || h.AttackPhase != sim.AttackReady {
			t.Fatalf("the second warrior was not idle when the order was given: victim %v destination %v phase %d",
				h.HasAttackTarget, h.HasTarget, h.AttackPhase)
		}
		if err := arena.app.HeadlessKey("select-all"); err != nil {
			t.Fatal(err)
		}
		if got := arena.app.HeadlessSelection(); len(got) != 2 {
			t.Fatalf("select-all selected %v, want both warriors", got)
		}
		goalX, goalY := arena.cx-4, arena.cy-3
		cmds := arena.orderMove(t, goalX, goalY)
		if len(cmds) != 2 || cmds[0].Entity == cmds[1].Entity {
			t.Fatalf("the ground click queued %+v, want one move for each warrior", cmds)
		}
		arena.loadedOnLanding(t, loaded, east)
		arena.live.tick()
		h, i := arena.get(loaded), arena.get(idle)
		t.Logf("after the formation order: loaded warrior victim %v/%d phase %d destination %v (%d,%d); idle warrior destination %v (%d,%d)",
			h.HasAttackTarget, h.AttackTarget, h.AttackPhase, h.HasTarget, h.TargetX, h.TargetY, i.HasTarget, i.TargetX, i.TargetY)
		if !h.HasAttackTarget || h.AttackTarget != east || h.AttackPhase != sim.AttackCharging || !h.HasTarget {
			t.Fatalf("the formation move dropped the loaded cycle: victim %v/%d phase %d destination %v",
				h.HasAttackTarget, h.AttackTarget, h.AttackPhase, h.HasTarget)
		}
		if !i.HasTarget {
			t.Fatal("the idle warrior was given no destination")
		}
		idleLeft := int64(-1)
		out := blowOutcome{strikeTick: -1, leftTick: -1}
		for n := 0; n < 400 && out.leftTick < 0; n++ {
			before, hp := arena.get(loaded), arena.get(east).HP
			arena.live.tick()
			now := arena.get(loaded)
			if idleLeft < 0 && (arena.get(idle).X != idleStart.X || arena.get(idle).Y != idleStart.Y) {
				idleLeft = arena.now()
			}
			if out.strikeTick < 0 && before.AttackPhase == sim.AttackCharging && now.AttackPhase != sim.AttackCharging {
				out.strikeTick, out.lost = arena.now(), hp-arena.get(east).HP
			}
			if now.X != loadedStart.X || now.Y != loadedStart.Y {
				out.leftTick = arena.now()
			}
		}
		t.Logf("the idle warrior left at tick %d; the loaded warrior's blow resolved at tick %d, took %d health, and he left at tick %d",
			idleLeft, out.strikeTick, out.lost, out.leftTick)
		if idleLeft < 0 || idleLeft > out.leftTick {
			t.Fatalf("the idle warrior left at tick %d, want him gone by the time the loaded warrior left at tick %d", idleLeft, out.leftTick)
		}
		if out.strikeTick < 0 || out.leftTick < 0 || out.strikeTick >= out.leftTick {
			t.Fatalf("the loaded warrior left at tick %d and his blow resolved at tick %d, want the blow first", out.leftTick, out.strikeTick)
		}
		if out.lost <= 0 || arena.get(east).HP >= full {
			t.Fatalf("the loaded warrior's blow took %d health", out.lost)
		}
	})

	t.Run("save and load while the move waits behind the blow", func(t *testing.T) {
		arena := openLoadedCycleArena(t, 1)
		hero, east := arena.heroes[0], arena.east
		arena.selectFirstHero(t)
		arena.orderAttack(t, east)
		arena.untilLoaded(t, hero, east)
		goalX, goalY := arena.cx-4, arena.cy-3
		arena.orderMove(t, goalX, goalY)
		arena.loadedOnLanding(t, hero, east)
		arena.live.tick()
		held := arena.get(hero)
		if !held.HasAttackTarget || !held.HasTarget || held.AttackPhase != sim.AttackCharging {
			t.Fatalf("the move did not wait behind the blow: victim %v destination %v phase %d", held.HasAttackTarget, held.HasTarget, held.AttackPhase)
		}
		restored, control := arena.reload(t)
		back, _ := restored.entity(hero)
		t.Logf("after load: victim %v/%d phase %d countdown %d destination %v (%d,%d)", back.HasAttackTarget, back.AttackTarget,
			back.AttackPhase, back.AttackCountdown, back.HasTarget, back.TargetX, back.TargetY)
		if !back.HasAttackTarget || back.AttackTarget != east || !back.HasTarget || back.TargetX != goalX || back.TargetY != goalY ||
			back.AttackPhase != sim.AttackCharging || back.AttackCountdown != held.AttackCountdown {
			t.Fatalf("the loaded game lost the waiting move: victim %v/%d destination %v (%d,%d) phase %d countdown %d, saved %d",
				back.HasAttackTarget, back.AttackTarget, back.HasTarget, back.TargetX, back.TargetY, back.AttackPhase,
				back.AttackCountdown, held.AttackCountdown)
		}
		for n := range 120 {
			sim.Step(control, nil)
			sim.Step(restored.world, nil)
			if control.Hash() != restored.world.Hash() {
				t.Fatalf("the loaded game differs from the saved world bytes %d ticks after the load", n+1)
			}
		}
		end, _ := restored.entity(hero)
		if end.X == held.X && end.Y == held.Y {
			t.Fatalf("the loaded warrior never left his cell in 120 ticks")
		}
		t.Logf("120 ticks after the load the world hash equals the control's; the warrior walked from (%d,%d) to (%d,%d)", held.X, held.Y, end.X, end.Y)
	})

	t.Run("attack on another creature: the old blow finishes before the requested victim", func(t *testing.T) {
		arena := openLoadedCycleArena(t, 1)
		bindings, err := arena.live.mission.state.Map.CellBindings()
		if err != nil {
			t.Fatal(err)
		}
		tails := make([]sim.CellTail, len(bindings))
		for i, binding := range bindings {
			tails[i] = sim.CellTail{X: int32(binding.X), Y: int32(binding.Y), Bytes: [6]byte{binding.Spell, binding.Power, binding.SourceX, binding.SourceY, binding.LastX, binding.LastY}}
		}
		if err := arena.live.world.DeclareCellTails(tails); err != nil {
			t.Fatal(err)
		}
		load := func(bytes []byte) *FrontEnd {
			back := releaseFront(t)
			back.Options = OptionsStore{}
			back.SetDeterministicFrames(true)
			opener, town, err := back.RestoreOriginal(bytes)
			if err != nil || town || opener == nil {
				t.Fatal("ordinary pending-victim LOAD", err, town)
			}
			app := back.App("pending victim LOAD")
			t.Cleanup(app.StopAudio)
			app.Layout(1024, 768)
			if err := app.OpenMission(opener); err != nil {
				t.Fatal(err)
			}
			return back
		}
		hero, east, south := arena.heroes[0], arena.east, arena.south
		arena.selectFirstHero(t)
		arena.orderAttack(t, east)
		arena.untilLoaded(t, hero, east)
		baseline, baselineLabel, err := arena.front.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		baselineHash := arena.live.world.Hash()
		baselineRaw, err := arena.front.ExportCurrentSave(baseline, baselineLabel)
		if err != nil || arena.live.world.Hash() != baselineHash {
			t.Fatal("uninterrupted installed-cycle SAVE failed or changed World", err)
		}
		baselineCold := load(baselineRaw)
		assertCurrentWorldEqual(t, arena.live.world, baselineCold.live.world, "installed loaded-cycle control before requested victim")
		if pendingVictimEntity(t, baselineCold.live.world, hero).HasPendingAttackTarget {
			t.Fatal("uninterrupted installed cycle acquired a requested victim")
		}
		eastHP, southHP := arena.get(east).HP, arena.get(south).HP
		arena.orderAttack(t, south)
		arena.loadedOnLanding(t, hero, east)
		controlBytes, err := arena.live.world.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var control sim.World
		if err := control.UnmarshalBinary(controlBytes); err != nil {
			t.Fatal(err)
		}
		mapload.BindSourceDerive(&control)
		sim.Step(&control, nil)
		arena.live.tick()
		h := arena.get(hero)
		without := pendingVictimEntity(t, &control, hero)
		if !h.HasAttackTarget || h.AttackTarget != east || h.AttackPhase != without.AttackPhase || h.AttackCountdown != without.AttackCountdown ||
			h.Facing != without.Facing || h.DesiredFacing != without.DesiredFacing || h.TurnRemaining != without.TurnRemaining || h.TurnTotal != without.TurnTotal ||
			!h.HasPendingAttackTarget || h.PendingAttackTarget != south || h.PendingAttackTargetKind != sim.AttackTargetUnit {
			t.Fatalf("player Attack changed the loaded cycle instead of queuing its victim: applied %+v; no-command control %+v", h, without)
		}
		snapshot, label, err := arena.front.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		savedHash := arena.live.world.Hash()
		raw, err := arena.front.ExportCurrentSave(snapshot, label)
		if err != nil || arena.live.world.Hash() != savedHash {
			t.Fatal("ordinary SAV export failed or changed the loaded cycle", err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		record := generatedActorRecord(t, &doc, hero)
		eastKey := savedRecordValueForTest(t, *generatedActorRecord(t, &doc, east), "Identity")
		southKey := savedRecordValueForTest(t, *generatedActorRecord(t, &doc, south), "Identity")
		if eastKey == 0 || southKey == 0 || eastKey == southKey || savedRecordValueForTest(t, *record, "U5C") != eastKey ||
			binary.LittleEndian.Uint32(savedRecordRawForTest(t, *record, "U158")[0xc:]) != southKey {
			t.Fatal("ordinary SAV conflated active U5C and requested U158 victims")
		}
		cold := load(raw)
		if arena.live.world.Hash() != cold.live.world.Hash() {
			before, _ := arena.live.world.MarshalBinary()
			after, _ := cold.live.world.MarshalBinary()
			currentMenuWorldDiagnostics(t, arena.live.world, cold.live.world)
			for offset := 0; offset < min(len(before), len(after)); offset++ {
				if before[offset] != after[offset] {
					t.Logf("pending-victim byte difference at %d, lengths %d/%d: live %x cold %x", offset, len(before), len(after), before[max(0, offset-8):min(len(before), offset+24)], after[max(0, offset-8):min(len(after), offset+24)])
					break
				}
			}
		}
		assertCurrentWorldEqual(t, arena.live.world, cold.live.world, "installed pending-victim LOAD")
		loss, err := sav.CloneDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		actions, err := readCurrentActions(&loss)
		if err != nil || actions == nil {
			t.Fatal("installed pending-victim supplement", err)
		}
		for i := range actions.Actions.Actors {
			row := &actions.Actions.Actors[i]
			if row.Entity == hero {
				if !row.HasPendingAttackTarget || row.PendingAttackTarget != south {
					t.Fatal("installed current actions lack the requested victim")
				}
				row.HasPendingAttackTarget, row.PendingAttackTarget, row.PendingAttackTargetKind = false, 0, sim.AttackTargetUnit
			}
		}
		payload, err := json.Marshal(actions)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&loss.State, payload); err != nil {
			t.Fatal(err)
		}
		if binary.LittleEndian.Uint32(savedRecordRawForTest(t, *generatedActorRecord(t, &loss, hero), "U158")[0xc:]) != southKey {
			t.Fatal("installed loss control changed the raw requested victim")
		}
		lossRaw, err := sav.EncodeDocumentData(loss)
		if err != nil {
			t.Fatal(err)
		}
		lossy := load(lossRaw)
		lost := pendingVictimEntity(t, lossy.live.world, hero)
		if lost.HasPendingAttackTarget || !lost.HasAttackTarget || lost.AttackTarget != east || lost.AttackPhase != h.AttackPhase || lost.AttackCountdown != h.AttackCountdown {
			t.Fatalf("typed queue loss changed the old cycle or restored the discarded queue: %+v", lost)
		}
		firstEast, firstSouth := int64(-1), int64(-1)
		seenRelax, seenBoundaryOne, seenBoundaryTwo := false, false, false
		var afterEast int32
		for tick := range 400 {
			arena.live.tick()
			cold.live.tick()
			lossy.live.tick()
			assertCurrentWorldEqual(t, arena.live.world, cold.live.world, fmt.Sprintf("installed pending-victim successor %d", tick))
			h = arena.get(hero)
			if h.AttackTarget == east {
				seenRelax = seenRelax || h.AttackPhase == sim.AttackRelaxing
				seenBoundaryOne = seenBoundaryOne || h.AttackPhase == sim.AttackBoundaryOne
				seenBoundaryTwo = seenBoundaryTwo || h.AttackPhase == sim.AttackBoundaryTwo
			}
			if firstEast < 0 && arena.get(east).HP < eastHP {
				firstEast, afterEast = arena.now(), arena.get(east).HP
				if arena.get(south).HP != southHP {
					t.Fatal("requested victim took damage before the old blow")
				}
			}
			if firstSouth < 0 && arena.get(south).HP < southHP {
				firstSouth = arena.now()
				if firstEast < 0 || firstSouth <= firstEast || !seenRelax || !seenBoundaryOne || !seenBoundaryTwo || arena.get(east).HP != afterEast || h.AttackTarget != south || h.HasPendingAttackTarget {
					t.Fatalf("requested blow skipped the old recovery or boundaries: first %d/%d phases %t/%t/%t actor %+v", firstEast, firstSouth, seenRelax, seenBoundaryOne, seenBoundaryTwo, h)
				}
			}
			if pendingVictimEntity(t, lossy.live.world, south).HP != southHP {
				t.Fatal("requested victim was struck after its typed queue was removed")
			}
		}
		if firstEast < 0 || firstSouth < 0 || pendingVictimEntity(t, lossy.live.world, east).HP >= eastHP {
			t.Fatal("installed ticks did not execute both victim blows and the surviving loss-control blow")
		}
		t.Logf("old victim %d struck at tick %d; requested victim %d struck at tick %d after recovery and both boundaries; loss control held requested HP %d", east, firstEast, south, firstSouth, southHP)
	})
}
