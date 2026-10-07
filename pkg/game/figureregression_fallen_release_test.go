package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// fell lowers one actor to health hp through the mission script's own health
// write and lets the ordinary ticks run.
func (r *figureRig) fell(id sim.EntityID, hp int32, ticks int) {
	t := r.t
	t.Helper()
	r.damageTo(id, hp)
	for range ticks {
		r.f.live.tick()
	}
}

func (r *figureRig) damageTo(id sim.EntityID, hp int32) {
	t := r.t
	t.Helper()
	e, ok := r.f.live.entity(id)
	if !ok {
		t.Fatalf("entity %d is absent", id)
	}
	if amount := e.HP - hp; amount > 0 {
		if err := r.f.live.world.HeadlessDamage(id, amount); err != nil {
			t.Fatal(err)
		}
	}
}

// auditFallen checks what a fallen or dead unit is drawn as: a player's
// character and a placed Hero stay on their PC body's corpse record, a hired
// mercenary and a placed person on their own class's, never the NPC class a
// lost PC binding falls back to (HERO-APPEAR-041, UNIT-APPEAR-030).
func auditFallen(r *figureRig, id sim.EntityID, wantArt *terrain.UnitClass, stage string) []string {
	mw := r.f.live
	want := wantArt
	if wantArt != nil && wantArt.Corpse != nil {
		want = wantArt.Corpse
	}
	e, _ := mw.entity(id)
	d, drawn := r.drawsByID()[id]
	if !drawn {
		return []string{fmt.Sprintf("%s: entity %d is not drawn (health %d, decay %d)", stage, id, e.HP, e.Decay)}
	}
	if isPCBody(mw.units, wantArt) {
		// A hero's worn set drops with him, and the body his equipment names
		// follows it: the identity is a PC body, whichever one. The stock
		// class records' own corpses are the NPC sprites a lost binding shows.
		for _, class := range mw.units.Classes {
			if class != nil && class.Corpse != nil && d.Art == class.Corpse {
				return []string{fmt.Sprintf("%s: entity %d (health %d, decay %d) is drawn as the NPC corpse %q",
					stage, id, e.HP, e.Decay, artName(d.Art))}
			}
		}
		for _, body := range mw.units.Bodies {
			if body != nil && (d.Art == body.Corpse || body.Corpse == nil && d.Art == body) {
				return nil
			}
		}
		return []string{fmt.Sprintf("%s: entity %d (health %d, decay %d) is drawn as %q, which is no PC body's record",
			stage, id, e.HP, e.Decay, artName(d.Art))}
	}
	if d.Art != want {
		return []string{fmt.Sprintf("%s: entity %d (health %d, decay %d) is drawn as %q, want %q",
			stage, id, e.HP, e.Decay, artName(d.Art), artName(want))}
	}
	return nil
}

// TestReleaseFigureRegressionFallen fells a primary hero, a carried companion,
// a hired mercenary and a placed Hero by the script's health write and the
// ordinary ticks. At health 0 through -9 (Heal can still raise the body), past
// -9 and once the body has decayed and been looted, each keeps the art its
// identity selects, fresh and after SAVE and a cold LOAD.
func TestReleaseFigureRegressionFallen(t *testing.T) {
	t.Run("party hero and companion", func(t *testing.T) {
		r := openFigureMission(t, 70, figureHeroes[1])
		mw := r.f.live
		var heroes []sim.EntityID
		bodies := map[sim.EntityID]*terrain.UnitClass{}
		for i, p := range mw.mission.party {
			if !p.Hired() && data.ComposesFigure(p.Class) {
				id := mw.mission.ids[i]
				heroes = append(heroes, id)
				bodies[id] = mw.art[id]
			}
		}
		if len(heroes) < 2 {
			t.Fatalf("party holds %d player characters, want a hero and a companion", len(heroes))
		}
		for _, hp := range []int32{0, -5, -9} {
			for _, id := range heroes {
				r.damageTo(id, hp)
			}
			for range 2 {
				mw.tick()
			}
			for _, id := range heroes {
				e, _ := mw.entity(id)
				if e.Alive() || !e.Restorable() || e.HP != hp {
					t.Fatalf("hero %d at health %d: alive %v restorable %v", id, e.HP, e.Alive(), e.Restorable())
				}
				requireClean(t, fmt.Sprintf("fresh hero at %d", hp), auditFallen(r, id, bodies[id], fmt.Sprintf("health %d", hp)))
			}
			cold := r.coldLoad()
			for _, id := range heroes {
				requireClean(t, fmt.Sprintf("loaded hero at %d", hp), auditFallen(cold, id, bodies[id], fmt.Sprintf("loaded, health %d", hp)))
			}
			for _, id := range heroes {
				if err := mw.world.HeadlessHeal(id); err != nil {
					t.Fatal(err)
				}
			}
			mw.tick()
		}
		// Past -9 the loss rule ends the mission; the art stays on the PC body
		// through the dying frames and the decay stages.
		for _, id := range heroes {
			r.damageTo(id, -100)
		}
		for range 2 {
			mw.tick()
		}
		for _, id := range heroes {
			requireClean(t, "dead hero, dying", auditFallen(r, id, bodies[id], "past -9"))
		}
		for range 80 {
			mw.tick()
		}
		for _, id := range heroes {
			requireClean(t, "dead hero, bones", auditFallen(r, id, bodies[id], "decayed"))
		}
	})
	t.Run("hired mercenary", func(t *testing.T) {
		r := openFigureMission(t, 70, figureHeroes[0])
		mw := r.f.live
		var id sim.EntityID
		var class *terrain.UnitClass
		for i, p := range mw.mission.party {
			if p.Hired() && data.ComposesFigure(p.Class) && !data.FigureHasHorse(p.Class) {
				id, class = mw.mission.ids[i], mw.units.Classes[p.Class]
				break
			}
		}
		if class == nil {
			t.Fatal("no hired person in the party")
		}
		r.fell(id, -5, 6)
		requireClean(t, "fresh fallen mercenary", auditFallen(r, id, class, "health -5"))
		if d := r.drawsByID()[id]; !d.Restorable {
			t.Error("a mercenary at health -5 is not restorable")
		}
		cold := r.coldLoad()
		requireClean(t, "loaded fallen mercenary", auditFallen(cold, id, class, "loaded, health -5"))
		r.fell(id, -100, 80)
		requireClean(t, "decayed mercenary", auditFallen(r, id, class, "decayed"))
		if d := r.drawsByID()[id]; d.Restorable || isPCBody(mw.units, d.Art) {
			t.Errorf("a decayed mercenary is restorable %v or on a PC body", d.Restorable)
		}
		cold = r.coldLoad()
		requireClean(t, "loaded decayed mercenary", auditFallen(cold, id, class, "loaded, decayed"))
	})
	t.Run("placed Hero", func(t *testing.T) {
		r := openFigureMission(t, 40, figureHeroes[0])
		mw := r.f.live
		var id sim.EntityID
		found := false
		for _, e := range mw.world.Entities() {
			if _, placed := mw.mission.state.Start.Roster[e.ID]; placed && data.FigureIsHero(e.TypeID) && e.Alive() && e.MapUnitID != 0 {
				id, found = e.ID, true
				break
			}
		}
		if !found {
			t.Fatal("mission 40 holds no placed Hero")
		}
		body := mw.art[id]
		if !isPCBody(mw.units, body) {
			t.Fatalf("the placed Hero starts on %q, not a PC body", artName(body))
		}
		for _, step := range []struct {
			hp    int32
			ticks int
			name  string
		}{{-5, 6, "dying"}, {-100, 3, "dead"}, {-100, 80, "looted"}} {
			r.fell(id, step.hp, step.ticks)
			requireClean(t, "fresh placed Hero "+step.name, auditFallen(r, id, body, step.name))
			cold := r.coldLoad()
			requireClean(t, "loaded placed Hero "+step.name, auditFallen(cold, id, body, "loaded, "+step.name))
		}
	})
}

// TestReleaseFigureRegressionTierPalettes reads every unit row of the shipped
// Units collection. Rows that share a class and differ in their Face column are
// the tiers of one family; each tier resolves to its own palette over the
// family's one sheet, so four tiers are four pictures.
func TestReleaseFigureRegressionTierPalettes(t *testing.T) {
	f := releaseFront(t)
	rows := map[int32]map[int32]bool{}
	for i := 0; i < f.Table.Units.Len(); i++ {
		def, err := data.NewUnitDef(f.Table.Units.EntryName(i), f.Table.Units.EntryParams(i))
		if err != nil || f.Units.Classes[def.TypeID] == nil {
			continue
		}
		if rows[def.TypeID] == nil {
			rows[def.TypeID] = map[int32]bool{}
		}
		rows[def.TypeID][def.Face] = true
	}
	families, tiers := 0, 0
	for class, faces := range rows {
		if len(faces) < 2 {
			continue
		}
		families++
		c := f.Units.Classes[class]
		var order []int
		for face := range faces {
			order = append(order, int(face))
		}
		slices.Sort(order)
		for a := 0; a < len(order); a++ {
			fa := c.TierFrames(order[a])
			if len(fa) != len(c.Frames) || len(fa) == 0 {
				t.Errorf("class %d %q tier %d holds %d frames, the sheet %d", class, c.Name, order[a], len(fa), len(c.Frames))
				continue
			}
			tiers++
			for b := a + 1; b < len(order); b++ {
				if fb := c.TierFrames(order[b]); len(fb) > 0 && fa[0].Palette == fb[0].Palette {
					t.Errorf("class %d %q: tiers %d and %d resolve to one palette", class, c.Name, order[a], order[b])
				}
			}
		}
	}
	if families < 10 || tiers < 4*families {
		t.Errorf("the shipped Units collection holds %d tiered families and %d tiers, want at least 10 families of four", families, tiers)
	}
	t.Logf("%d tiered families, %d tier palettes, all distinct within a family", families, tiers)
}
