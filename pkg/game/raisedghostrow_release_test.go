package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	ghostRowFireArrow     = 1
	ghostRowInvisibility  = 15
	ghostRowControlSpirit = 25
	ghostRowMission       = 20
)

// raisedGhostEntity is the one live raised Ghost of TypeID typ.
func raisedGhostEntity(t *testing.T, f *FrontEnd, typ int32) sim.Entity {
	t.Helper()
	requireLiveRaisedGhost(t, f, typ)
	for _, e := range f.live.world.Entities() {
		if e.TypeID == typ && e.MapUnitID == 0 && e.Owner == sim.SelfSlot && e.Alive() {
			return e
		}
	}
	return sim.Entity{}
}

func requireGhostRowColumns(t *testing.T, label string, e sim.Entity, def data.UnitDef, general int32) {
	t.Helper()
	cols := []struct {
		name      string
		got, want int32
	}{
		{"health regeneration period", e.HealthRegenPeriod, def.HealthRegenPeriod},
		{"mana regeneration period", e.ManaRegenPeriod, def.ManaRegenPeriod},
		{"mana maximum", e.MaxMana, def.ManaMax},
		{"see-invisible", int32(e.SeeInvisible), def.SeeInvisible},
		{"capacity", e.Capacity, data.UnitCapacity()},
		{"general skill", e.Skill[data.SkillGeneral], general},
		{"experience value", e.XPValue, def.XPValue},
	}
	for _, c := range cols {
		if c.got != c.want {
			t.Errorf("%s: raised Ghost %s = %d, want the installed row's %d", label, c.name, c.got, c.want)
		}
	}
}

// TestReleaseRaisedGhostTakesItsWholeUnitsRow: a Ghost raised in mission 20
// regenerates after a Fire Arrow, alone reveals an invisible creature and has
// capacity 300; SAVE, cold LOAD and 256 ticks keep the live World hash.
func TestReleaseRaisedGhostTakesItsWholeUnitsRow(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	row := data.NotFound
	for i := 1; i < f.Table.Units.Len(); i++ {
		if f.Table.Units.EntryName(i) == "Ghost" {
			row = i
			break
		}
	}
	if row == data.NotFound {
		t.Fatal("installed Units table has no Ghost row")
	}
	params := f.Table.Units.EntryParams(row)
	def, err := data.NewUnitDef("Ghost", params)
	if err != nil {
		t.Fatal(err)
	}
	general := params[14]
	if def.HealthRegenPeriod == 0 || def.SeeInvisible == 0 || general <= 0 {
		t.Fatalf("installed Ghost row cannot discriminate: period %d see-invisible %d general %d", def.HealthRegenPeriod, def.SeeInvisible, general)
	}

	party := f.ChargenParty(ui.ChargenResult{Name: "Ghost row witness", Choices: []int{0, 1, 0}, Stats: []int{20, 25, 40, 40}})
	// No heal of the mage's own may reach the Ghost.
	party[0].KnownSpells = 1<<ghostRowControlSpirit | 1<<ghostRowFireArrow | 1<<ghostRowInvisibility
	f.Carried = party
	if err := f.App("raised Ghost row").OpenMission(f.MissionOpenerWith(ghostRowMission, f.Carried)); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	typ := w.Ghost().TypeID
	var mage, victim, hidden sim.Entity
	// The largest corpse gives a Ghost that survives one Fire Arrow.
	for _, e := range w.Entities() {
		switch {
		case mage.ID == 0 && e.Owner == sim.SelfSlot && e.Humanoid:
			mage = e
		case e.Owner != sim.SelfSlot && e.Alive() && !e.Humanoid && e.TypeID != typ && e.MaxHP > victim.MaxHP:
			victim = e
		}
	}
	for _, e := range w.Entities() {
		if e.ID != victim.ID && e.Owner != sim.SelfSlot && e.Alive() && !e.Humanoid && e.TypeID != typ && e.MaxHP > hidden.MaxHP {
			hidden = e
		}
	}
	if mage.ID == 0 || victim.ID == 0 || hidden.ID == 0 {
		t.Fatalf("mission has no mage %d or two creatures %d %d", mage.ID, victim.ID, hidden.ID)
	}
	placed := false
	for _, d := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if w.HeadlessPlace(victim.ID, mage.X+d[0], mage.Y+d[1]) == nil {
			placed = true
			break
		}
	}
	if !placed {
		t.Fatal("no free cell beside the mage")
	}
	if err := w.HeadlessKill(victim.ID); err != nil {
		t.Fatal(err)
	}
	for range 256 {
		f.live.tick()
		if e, ok := liveEntity(f, victim.ID); ok && e.Decay == sim.DecayBones {
			break
		}
	}
	f.live.pending = append(f.live.pending, sim.Cast(mage.ID, victim.ID, ghostRowControlSpirit))
	for range 256 {
		f.live.tick()
		if _, ok := liveEntity(f, victim.ID); !ok {
			break
		}
	}
	ghost := raisedGhostEntity(t, f, typ)
	requireGhostRowColumns(t, "raise", ghost, def, general)

	f.live.pending = append(f.live.pending, sim.Cast(mage.ID, ghost.ID, ghostRowFireArrow))
	damaged := false
	for range 256 {
		f.live.tick()
		if e := raisedGhostEntity(t, f, typ); e.HP < e.MaxHP {
			ghost, damaged = e, true
			break
		}
	}
	if !damaged {
		t.Fatal("ordinary Fire Arrow did not damage the raised Ghost")
	}
	low, healed := ghost.HP, sim.Entity{}
	for n := 0; n < 1024 && healed.ID == 0; n++ {
		f.live.tick()
		e := raisedGhostEntity(t, f, typ)
		if e.HP > low {
			healed = e
		}
		low = min(low, e.HP)
	}
	if healed.ID == 0 {
		t.Fatalf("raised Ghost did not regenerate from %d over 1024 ordinary ticks", low)
	}
	t.Logf("Fire Arrow left the Ghost at %d/%d; regeneration raised it to %d by tick %d", low, healed.MaxHP, healed.HP, f.live.world.Tick())

	// Only the Ghost's see-invisible range may reach the invisible creature.
	for n := 0; n < 8192; n++ {
		if m, _ := liveEntity(f, mage.ID); m.Mana >= m.MaxMana {
			break
		}
		f.live.tick()
	}
	near := false
	for _, d := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}} {
		m, _ := liveEntity(f, mage.ID)
		if w.HeadlessPlace(hidden.ID, m.X+d[0], m.Y+d[1]) == nil {
			near = true
			break
		}
	}
	if !near {
		t.Fatal("no free cell beside the mage for the second creature")
	}
	if why := f.live.world.BookSpellRefusal(mage.ID, hidden.ID, ghostRowInvisibility); why != "" {
		t.Fatal("mage cannot cast Invisibility:", why)
	}
	f.live.pending = append(f.live.pending, sim.Cast(mage.ID, hidden.ID, ghostRowInvisibility))
	attached := false
	for range 256 {
		f.live.tick()
		if f.live.world.HasEffectSpell(hidden.ID, ghostRowInvisibility) {
			attached = true
			break
		}
	}
	if !attached {
		t.Fatal("ordinary Invisibility cast did not attach")
	}
	cheb := func(a, b sim.Entity) int32 { return max(ghostRowAbs(a.X-b.X), ghostRowAbs(a.Y-b.Y)) }
	revealedByOther := func() bool {
		target, _ := liveEntity(f, hidden.ID)
		for _, e := range f.live.world.Entities() {
			if e.Owner == sim.SelfSlot && e.TypeID != typ && cheb(e, target) <= int32(e.SeeInvisible) {
				return true
			}
		}
		return false
	}
	ghost = raisedGhostEntity(t, f, typ)
	isolated := false
	for k := int32(ghost.SeeInvisible); k >= 1 && !isolated; k-- {
		for x := -k; x <= k && !isolated; x++ {
			for y := -k; y <= k && !isolated; y++ {
				if max(ghostRowAbs(x), ghostRowAbs(y)) == k && w.HeadlessPlace(hidden.ID, ghost.X+x, ghost.Y+y) == nil {
					target, _ := liveEntity(f, hidden.ID)
					d := cheb(ghost, target)
					isolated = d >= 1 && d <= int32(ghost.SeeInvisible) && !revealedByOther()
				}
			}
		}
	}
	if !isolated {
		t.Fatal("no free cell within the Ghost's see-invisible range that no other player actor reveals")
	}
	ghost = raisedGhostEntity(t, f, typ)
	target, _ := liveEntity(f, hidden.ID)
	if d := cheb(ghost, target); d < 1 || d > int32(ghost.SeeInvisible) {
		t.Fatalf("invisible creature stands %d cells from the Ghost; want 1..%d", d, ghost.SeeInvisible)
	}
	if f.live.world.InvisibleTo(hidden.ID, sim.SelfSlot) {
		t.Fatal("raised Ghost does not reveal an invisible creature within its see-invisible range")
	}
	t.Logf("invisible creature %d cell(s) from the Ghost revealed by see-invisible %d alone; capacity %d", cheb(ghost, target), ghost.SeeInvisible, ghost.Capacity)

	path, _ := writeOrdinarySAV(t, f, "ghost-row.sav")
	cold := loadAreaContinuation(t, path)
	coldGhost := raisedGhostEntity(t, cold, typ)
	requireGhostRowColumns(t, "cold LOAD", coldGhost, def, general)
	for n := 0; n <= 256; n++ {
		if f.live.world.Hash() != cold.live.world.Hash() {
			live, cg := raisedGhostEntity(t, f, typ), raisedGhostEntity(t, cold, typ)
			t.Fatalf("cold World hash differs from live at tick %d after LOAD (Ghost live %+v cold %+v)", n, live, cg)
		}
		if n < 256 {
			f.live.tick()
			cold.live.tick()
		}
	}
	requireGhostRowColumns(t, "cold +256", raisedGhostEntity(t, cold, typ), def, general)
}

func ghostRowAbs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
