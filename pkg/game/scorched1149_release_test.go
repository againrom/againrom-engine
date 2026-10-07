package game

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Installed sheets/map/spells and real App clicks; actor population and start
// position are controlled. This does not claim an original-runtime trace.
func TestReleaseFireScenery1149(t *testing.T) {
	f := releaseFront(t)
	// Literal section pairs from the complete installed 82-class registry.
	pairs := [][2]int{{0, 2}, {1, 2}, {3, 5}, {4, 5}, {6, 8}, {7, 8}, {9, 11}, {10, 11}, {12, 14}, {13, 14}, {15, 17}, {16, 17}, {18, 20}, {19, 20}, {40, 41}, {42, 43}, {44, 45}, {46, 47}, {48, 49}, {50, 51}, {52, 53}, {54, 55}, {56, 57}, {58, 59}, {60, 61}, {62, 63}, {64, 65}, {66, 67}}
	for _, pair := range pairs {
		c, d := f.Statics.Classes[pair[0]+1], f.Statics.Classes[pair[1]+1]
		if c == nil || d == nil || c.Dead == nil || len(d.Frames) == 0 || c.Dead.Frame != d.Frames[0] || c.Dead.Width != d.Width || c.Dead.Height != d.Height || c.Dead.CenterX != d.CenterX || c.Dead.CenterY != d.CenterY || c.Dead.Index != 0 || len(c.Dead.Timeline) != 0 {
			t.Fatalf("dead class mapping %v", pair)
		}
	}
	count := 0
	for _, c := range f.Statics.Classes {
		if c != nil && c.Dead != nil {
			count++
		}
	}
	if count != 28 {
		t.Fatalf("dead forms=%d want28", count)
	}
	for _, spell := range []uint32{2, 3} {
		t.Run(fmt.Sprintf("spell%d", spell), func(t *testing.T) { releaseFireScenery1149(t, spell) })
	}
}

func releaseFireScenery1149(t *testing.T, spell uint32) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[1] = 100
	party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true, Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 1<<2 | 1<<3,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000, HealthRegenPeriod: 100, ManaRegenPeriod: 50}}}
	a := f.App("fire scenery")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	m := live.mission.state.Map
	x, y := -1, -1
	for cy := 20; cy < m.Height-20 && x < 0; cy++ {
		for cx := 20; cx < m.Width-20; cx++ {
			c := f.Statics.Classes[m.Overlay[cy*m.Width+cx]]
			if c != nil && c.Dead != nil && m.Tiles[cy*m.Width+cx]&0x2000 == 0 && !terrain.Resolve(m.Tiles[cy*m.Width+cx]).Water {
				x, y = cx, cy
				break
			}
		}
	}
	if x < 0 {
		t.Fatal("no intact installed tree")
	}
	actor, ok := live.entity(live.mission.ids[0])
	if !ok {
		t.Fatal("mage missing")
	}
	actor.X, actor.Y = int32(x-4), int32(y)
	actor.HasTarget, actor.Transit, actor.TransitTotal = false, 0, 0
	w, err := sim.NewStructuredWorld(1149, live.world.Bounds(), sim.ModeCanonical, sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)}, []sim.Entity{actor}, nil, live.world.Relations(), nil, nil, mapload.SpellRules(f.Table), sim.GhostTemplate{}, live.world.Structures())
	if err != nil {
		t.Fatal(err)
	}
	live.world = w
	live.commanded[actor.ID] = true
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	inspectionCentre(live, x, y)
	if err := a.HeadlessSelectEntity(uint32(actor.ID)); err != nil {
		t.Fatal(err)
	}
	live.push()
	_, before := live.view.ScorchedScenery()
	{
		if _, _, err := a.HeadlessSpellPoint(spell); err != nil {
			if err = a.HeadlessKey("book"); err != nil {
				t.Fatal(err)
			}
		}
		px, py, err := a.HeadlessSpellPoint(spell)
		if err != nil {
			t.Fatal(err)
		}
		for _, edge := range []string{"press", "release"} {
			if err = a.HeadlessPointer(edge, px, py); err != nil {
				t.Fatal(err)
			}
		}
		px, py = -1, -1
		for sy := 100; sy < 650 && px < 0; sy += 3 {
			for sx := 100; sx < 800; sx += 3 {
				cx, cy, e := a.HeadlessDropCell(sx, sy)
				if e == nil && cx == x && cy == y {
					px, py = sx, sy
					break
				}
			}
		}
		if px < 0 {
			t.Fatal("target cell has no input point")
		}
		for _, edge := range []string{"press", "release"} {
			if err = a.HeadlessPointer(edge, px, py); err != nil {
				t.Fatal(err)
			}
		}
		if len(live.pending) != 1 || live.pending[0].Kind != sim.KindCastAt || live.pending[0].Spell != uint16(spell) {
			t.Fatalf("click queued %+v", live.pending)
		}
		burnFrames := make(map[int]bool)
		for range 140 {
			live.tick()
			for _, flame := range live.burningSceneryDraws() {
				burnFrames[flame.Frame] = true
			}
		}
		if len(burnFrames) < 2 {
			t.Fatal("installed tree fire did not animate before replacement", burnFrames)
		}
		key := uint16(y*256 + x)
		if !slices.Contains(w.ScorchedCells(), key) {
			t.Fatalf("spell%d did not mark tree", spell)
		}
		ground, dead := live.view.ScorchedScenery()
		if ground == 0 || dead <= before {
			t.Fatalf("spell%d viewer ground/dead=%d/%d before=%d target=%d,%d tile=%x cells=%v", spell, ground, dead, before, x, y, m.Tiles[y*m.Width+x], w.ScorchedCells())
		}
		t.Logf("spell%d target=%d,%d ground=%d dead=%d", spell, x, y, ground, dead)
	}
	// Ordinary mission SaveStore, then a fresh front end and new viewer.
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return time.Unix(1149, 0) })
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	cold := releaseFront(t)
	_, _, load := cold.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatal("cold load", town, err)
	}
	ca := cold.App("cold fire scenery")
	ca.Layout(1024, 768)
	if err = ca.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	g, d := live.view.ScorchedScenery()
	cg, cd := cold.live.view.ScorchedScenery()
	if cells := cold.live.world.ScorchedCells(); !slices.Equal(w.ScorchedCells(), cells) || g != cg || d != cd {
		t.Fatalf("cold scene changed %d/%d %v -> %d/%d %v", g, d, w.ScorchedCells(), cg, cd, cells)
	}
	if s := cold.HeadlessSnapshot(ui.ScreenMap); s.ScorchedGround != g || s.BurnedObjects != d {
		t.Fatal("ordinary headless evidence differs")
	}
}
