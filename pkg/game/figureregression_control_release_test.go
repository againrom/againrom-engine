package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// placedIDs finds the first living placed entity of each kind the loss controls
// damage: a Hero, an ordinary person and a creature of a tiered family.
func (r *figureRig) placedIDs() (hero, person, creature sim.EntityID) {
	t := r.t
	t.Helper()
	mw := r.f.live
	pair := placementOf(mw)
	for _, e := range mw.world.Entities() {
		if !e.Alive() {
			continue
		}
		i, ok := pair(e)
		if !ok {
			continue
		}
		res := mapload.Resolve(mw.mission.state.Map.Units[i], r.f.Table)
		switch {
		case res.Arm != mapload.ArmUnits && data.FigureIsHero(e.TypeID) && hero == 0:
			hero = e.ID
		case res.Arm != mapload.ArmUnits && !data.FigureIsHero(e.TypeID) && person == 0:
			if fig, ok := mw.figures[e.ID]; ok && !fig.Dir.Mage() {
				person = e.ID
			}
		case res.Arm == mapload.ArmUnits && creature == 0 && mw.tiers[e.ID] != 0:
			creature = e.ID
		}
	}
	return
}

// TestReleaseFigureRegressionLossControls damages the resolved state the way
// each defect did and requires the matching audit to name it, so a clean audit
// in the other tests is not an audit that cannot fail.
func TestReleaseFigureRegressionLossControls(t *testing.T) {
	npcClass := func(r *figureRig, id sim.EntityID) *terrain.UnitClass {
		e, _ := r.f.live.entity(id)
		for typ, c := range r.f.live.units.Classes {
			if c != nil && int32(typ) != e.TypeID && !isPCBody(r.f.live.units, c) && c != r.f.live.art[id] {
				return c
			}
		}
		t.Fatal("no other class")
		return nil
	}
	party := func(r *figureRig, hired bool) sim.EntityID {
		for i, p := range r.f.live.mission.party {
			if p.Hired() == hired && data.ComposesFigure(p.Class) && !data.FigureHasHorse(p.Class) && !(hired && figureDirMage(p)) {
				return r.f.live.mission.ids[i]
			}
		}
		t.Fatal("no party member")
		return 0
	}
	controls := []struct {
		name    string
		mission int
		hero    figureHero
		audit   func(*figureRig) []string
		damage  func(r *figureRig)
	}{
		{"hero drawn as an NPC class", 70, figureHeroes[0], auditPartyFigures, func(r *figureRig) {
			id := party(r, false)
			r.f.live.art[id] = npcClass(r, id)
		}},
		{"hero without the background cape", 70, figureHeroes[0], auditPartyFigures, func(r *figureRig) {
			id := party(r, false)
			fig := r.f.live.figures[id]
			fig.Hero = false
			r.f.live.figures[id] = fig
		}},
		{"mercenary drawn as a hero", 70, figureHeroes[0], auditPartyFigures, func(r *figureRig) {
			hero, merc := party(r, false), party(r, true)
			r.f.live.art[merc] = r.f.live.art[hero]
		}},
		{"mercenary with the hero cape", 70, figureHeroes[0], auditPartyFigures, func(r *figureRig) {
			id := party(r, true)
			fig := r.f.live.figures[id]
			fig.Hero = true
			r.f.live.figures[id] = fig
		}},
		{"mercenary with an inventory", 70, figureHeroes[0], auditPartyFigures, func(r *figureRig) {
			hero, merc := party(r, false), party(r, true)
			pack, _ := r.f.live.world.Carried(hero)
			if len(pack) == 0 {
				t.Fatal("the hero carries nothing to hand over")
			}
			if err := r.f.live.world.MoveCarried(hero, merc, pack[0], 1); err != nil {
				t.Fatal(err)
			}
		}},
		{"placed Hero drawn as an NPC", 40, figureHeroes[0], auditPlacedFigures, func(r *figureRig) {
			id, _, _ := r.placedIDs()
			r.f.live.art[id] = npcClass(r, id)
		}},
		{"placed person with a hero cape", 40, figureHeroes[0], auditPlacedFigures, func(r *figureRig) {
			_, id, _ := r.placedIDs()
			fig := r.f.live.figures[id]
			fig.Hero = true
			r.f.live.figures[id] = fig
		}},
		{"enemy tiers drawn identically", 90, figureHeroes[0], auditPlacedFigures, func(r *figureRig) {
			_, _, id := r.placedIDs()
			r.f.live.tiers[id] = 0
		}},
	}
	for _, c := range controls {
		t.Run(c.name, func(t *testing.T) {
			r := openFigureMission(t, c.mission, c.hero)
			if bad := c.audit(r); len(bad) != 0 {
				t.Fatalf("the audit finds the undamaged state wanting: %v", bad)
			}
			c.damage(r)
			bad := c.audit(r)
			if len(bad) == 0 {
				t.Fatalf("the audit passes the damaged state")
			}
			t.Logf("named: %s", bad[0])
		})
	}
}

func figureDirMage(p mapload.PartyMember) bool {
	dir, _ := memberFigure(p)
	return dir.Mage()
}

// TestReleaseFigureRegressionTownLossControls damages the town's dolls and the
// hired members' packs and requires the town audit to name each.
func TestReleaseFigureRegressionTownLossControls(t *testing.T) {
	controls := []struct {
		name   string
		damage func(r *figureTownRig)
	}{
		{"hero doll without the cape", func(r *figureTownRig) {
			for i, p := range r.f.Carried {
				if !p.Hired() && !figureDirMage(p) {
					dir, face := memberFigure(p)
					r.s.shopFigures[i] = oracleFigure(r.f.Archives.Containers, figureID{Dir: dir, Face: face}, mapload.EquipmentFromParty(p))
					return
				}
			}
			t.Fatal("no fighter hero")
		}},
		{"hired doll with the hero cape", func(r *figureTownRig) {
			for i, p := range r.f.Carried {
				if p.Hired() && data.ComposesFigure(p.Class) && !data.FigureHasHorse(p.Class) && !figureDirMage(p) {
					dir, face := memberFigure(p)
					bare := oracleFigure(r.f.Archives.Containers, figureID{Dir: dir, Face: face}, mapload.EquipmentFromParty(p))
					r.s.shopFigures[i] = oracleCaped(t, r.f.Archives.Containers, dir, bare)
					return
				}
			}
			t.Fatal("no hired fighter")
		}},
		{"hired member with a pack", func(r *figureTownRig) {
			for i, p := range r.f.Carried {
				if p.Hired() {
					p.Carry = &mapload.Carry{Items: []uint16{1}}
					r.f.Carried[i] = p
					return
				}
			}
			t.Fatal("no hired member")
		}},
	}
	for _, c := range controls {
		t.Run(c.name, func(t *testing.T) {
			r := openFigureTown(t, 70, figureHeroes[0])
			if bad := auditTownParty(r); len(bad) != 0 {
				t.Fatalf("the audit finds the undamaged town wanting: %v", bad)
			}
			c.damage(r)
			bad := auditTownParty(r)
			if len(bad) == 0 {
				t.Fatal("the audit passes the damaged town")
			}
			t.Logf("named: %s", bad[0])
		})
	}
}

// TestReleaseFigureRegressionFallenLossControl moves a felled hero's art to an
// NPC class, as a lost PC binding did, and requires the fallen audit to name it.
func TestReleaseFigureRegressionFallenLossControl(t *testing.T) {
	r := openFigureMission(t, 70, figureHeroes[1])
	mw := r.f.live
	var id sim.EntityID
	for i, p := range mw.mission.party {
		if !p.Hired() && data.ComposesFigure(p.Class) {
			id = mw.mission.ids[i]
			break
		}
	}
	body := mw.art[id]
	r.fell(id, -5, 2)
	if bad := auditFallen(r, id, body, "health -5"); len(bad) != 0 {
		t.Fatalf("the audit finds the felled hero wanting: %v", bad)
	}
	for _, c := range mw.units.Classes {
		if c != nil && !isPCBody(mw.units, c) {
			mw.art[id] = c
			break
		}
	}
	if bad := auditFallen(r, id, body, "health -5"); len(bad) == 0 {
		t.Fatal("the audit passes a felled hero drawn as an NPC class")
	}
}
