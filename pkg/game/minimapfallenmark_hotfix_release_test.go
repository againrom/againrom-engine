package game

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A hired mace mercenary who falls past the point Heal can raise him has no
// minimap dot: in sight while the party walks away, under fog once it has,
// and after SAVE and a cold LOAD. The living party keeps its green dots, a
// hostile in sight keeps its red dot and a hostile under fog stays unmarked
// (DIV-1455). The fight and the walk are player attack and move orders over
// ordinary ticks.
//
// The mission's 144-cell map is sampled onto a 128-pixel minimap, so two
// neighbouring cells can share a pixel. Every absence below is read at a cell
// whose neighbours cannot supply the colour being tested.
func TestReleaseMinimapRetiresAFallenMercenaryDot(t *testing.T) {
	began := time.Now()
	const maceSquad, mission = 14, 60
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	town := NewTown(f.Campaign.Value())
	for m := range f.Campaign.Value().Chapters {
		if m < mission {
			town.Won(m)
		}
	}
	town.gold = 1_000_000
	f.Town = town
	f.Carried = f.NextParty()
	f.arriveInTown()
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	s.composeShopFaces()
	if msg, ok := s.toggleMercenary(maceSquad); !ok {
		t.Fatalf("tavern refused mercenary type %d: %s", maceSquad, msg)
	}
	closeNotices := func(app *ui.App) {
		t.Helper()
		for n := 0; app.HeadlessNoticeOpen() && n < 16; n++ {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
		if app.HeadlessNoticeOpen() {
			t.Fatal("mission notice did not close")
		}
	}
	a := f.App("fallen mercenary minimap")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(mission, f.Carried)); err != nil {
		t.Fatal(err)
	}
	closeNotices(a)

	type member struct {
		id    sim.EntityID
		start image.Point
		merc  bool
	}
	var party []member
	for i, p := range f.live.mission.party {
		e, ok := f.live.entity(f.live.mission.ids[i])
		if !ok {
			t.Fatalf("party member %d has no entity", i)
		}
		merc := p.MercenaryType == maceSquad
		if merc && (p.Weapon == nil || p.Weapon.AttackType != data.SkillBludgen) {
			t.Fatalf("hired type %d member %q does not wield a mace: %+v", maceSquad, p.Name, p.Weapon)
		}
		party = append(party, member{id: e.ID, start: image.Pt(int(e.X), int(e.Y)), merc: merc})
	}
	cellOf := func(e sim.Entity) image.Point { return image.Pt(int(e.X), int(e.Y)) }
	fogAt := func(fe *FrontEnd, c image.Point) uint8 {
		if c.X < 0 || c.Y < 0 || c.X >= fe.live.fog.cols || c.Y >= fe.live.fog.rows {
			return ui.FogUnseen
		}
		return fe.live.fog.project()[c.Y*fe.live.fog.cols+c.X]
	}
	markAt := func(app *ui.App, c image.Point) (local, other bool) {
		t.Helper()
		local, other, err := app.HeadlessMinimapMarkAt(c.X, c.Y)
		if err != nil {
			t.Fatal(err)
		}
		return local, other
	}
	// darkAround reports whether c and its eight neighbours are all out of
	// sight, so no drawn mark can share c's pixel.
	darkAround := func(fe *FrontEnd, c image.Point) bool {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if fogAt(fe, c.Add(image.Pt(dx, dy))) == ui.FogVisible {
					return false
				}
			}
		}
		return true
	}
	// partyBeside is the living party member on or beside c, whose green mark
	// could share c's pixel.
	partyBeside := func(fe *FrontEnd, c image.Point) (sim.EntityID, bool) {
		for _, e := range fe.live.world.Entities() {
			if d := cellOf(e).Sub(c); e.Alive() && e.Owner == sim.SelfSlot && chebyshevDist(d.X, d.Y) <= 1 {
				return e.ID, true
			}
		}
		return 0, false
	}
	restorable := func(e sim.Entity) bool { return e.MaxHP > 0 && e.OrdinaryTargetable() }
	rel := f.live.world.Relations()

	// Every living mercenary attacks the nearest living hostile until a mace
	// mercenary falls past Heal. The hero holds his place: a crowd now fills
	// the free cells nearest its victim (DIV-2225), and with the hero fighting
	// as well he fell first on this map.
	var body sim.Entity
	victims := map[sim.EntityID]sim.EntityID{}
	for n := 0; n < 1200 && body.ID == 0; n++ {
		for _, m := range party {
			e, ok := f.live.entity(m.id)
			if !ok || !e.Alive() || !m.merc {
				continue
			}
			if v, ok := f.live.entity(victims[m.id]); victims[m.id] != 0 && ok && v.Alive() {
				continue
			}
			best, bestD := sim.EntityID(0), 1<<30
			for _, h := range f.live.world.Entities() {
				if !h.Alive() || !rel.Hostile(sim.SelfSlot, h.Owner) || h.MaxHP < 30 {
					continue
				}
				if d := chebyshevDist(int(h.X-e.X), int(h.Y-e.Y)); d < bestD {
					best, bestD = h.ID, d
				}
			}
			if best != 0 {
				victims[m.id] = best
				f.live.attackOrCast(uint32(m.id), uint32(best), 0, 0, 0, false)
			}
		}
		f.live.tick()
		for _, m := range party {
			e, _ := f.live.entity(m.id)
			if !m.merc && !e.Alive() {
				t.Fatalf("tick %d: the hero fell before a mercenary", n)
			}
			if m.merc && !e.Alive() && !restorable(e) {
				body = e
				t.Logf("tick %d: mace mercenary %d fell at %v, HP %d of %d, stage %d", n, e.ID, cellOf(e), e.HP, e.MaxHP, e.Decay)
			}
		}
	}
	if body.ID == 0 {
		t.Fatal("no mace mercenary fell past Heal in 1200 ticks")
	}
	at := cellOf(body)

	// In sight: checked on the first tick of the walk away on which the body
	// is still visible and no living party member stands beside it.
	checkInSight := func() {
		t.Helper()
		green, _ := markAt(a, at)
		if green {
			t.Errorf("in sight: the fallen mercenary at %v still has a green minimap dot", at)
		}
		seen := 0
		for _, e := range f.live.world.Entities() {
			if !e.Alive() || !rel.Hostile(sim.SelfSlot, e.Owner) || fogAt(f, cellOf(e)) != ui.FogVisible {
				continue
			}
			if _, other := markAt(a, cellOf(e)); other {
				seen++
			}
		}
		if seen == 0 {
			t.Fatal("in sight: no living hostile has a red minimap dot")
		}
		t.Logf("in sight: body at %v, fog %d, green dot %t; %d living hostiles in sight marked", at, fogAt(f, at), green, seen)
	}

	// The survivors walk back to where they started until the body is under fog.
	for _, m := range party {
		if e, _ := f.live.entity(m.id); e.Alive() {
			f.live.enqueue(uint32(m.id), m.start.X, m.start.Y)
		}
	}
	walked, inSight := 0, false
	for ; walked < 600 && fogAt(f, at) == ui.FogVisible; walked++ {
		if _, beside := partyBeside(f, at); !inSight && !beside {
			checkInSight()
			inSight = true
		}
		f.live.tick()
	}
	if !inSight {
		t.Fatalf("the body's cell %v left sight before the party left its side", at)
	}
	if fogAt(f, at) == ui.FogVisible {
		t.Fatalf("the body's cell %v stayed in sight after the walk", at)
	}
	t.Logf("the body's cell left sight %d ticks into the walk", walked)
	checkUnderFog := func(fe *FrontEnd, app *ui.App, stage string) {
		t.Helper()
		var drawn []ui.MapEntity
		for _, d := range fe.live.entityDraws() {
			if d.Cell == at && d.Life == ui.LifeDead && d.Owner == sim.SelfSlot {
				drawn = append(drawn, d)
			}
		}
		if len(drawn) == 0 {
			t.Fatalf("%s: no local body is drawn at %v", stage, at)
		}
		if id, ok := partyBeside(fe, at); ok {
			t.Fatalf("%s: living party member %d stands beside the body at %v", stage, id, at)
		}
		green, _ := markAt(app, at)
		if green {
			t.Errorf("%s: the fallen mercenary at %v still has a green minimap dot under fog %d", stage, at, fogAt(fe, at))
		}
		living, hidden := 0, 0
		rel := fe.live.world.Relations()
		for _, e := range fe.live.world.Entities() {
			if !e.Alive() {
				continue
			}
			c := cellOf(e)
			switch {
			case e.Owner == sim.SelfSlot:
				living++
				if local, _ := markAt(app, c); !local {
					t.Errorf("%s: living party member %d at %v, %d cells from the body, has no green dot", stage, e.ID, c, chebyshevDist(c.X-at.X, c.Y-at.Y))
				}
			case rel.Hostile(sim.SelfSlot, e.Owner) && darkAround(fe, c):
				hidden++
				if _, other := markAt(app, c); other {
					t.Errorf("%s: hostile %d under fog at %v has a minimap dot", stage, e.ID, c)
				}
			}
		}
		if living == 0 || hidden == 0 {
			t.Fatalf("%s: %d living party members and %d hostiles under fog; both controls need one", stage, living, hidden)
		}
		t.Logf("%s: body %d at %v, HP %d, fog %d, green dot %t; %d living party members checked; %d hostiles under fog checked",
			stage, drawn[0].ID, at, drawn[0].HP, fogAt(fe, at), green, living, hidden)
	}
	checkUnderFog(f, a, "under fog")

	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE = %q: %v", name, err)
	}
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	_, _, load := cold.SaveSeams(store, OriginalStore{}, nil)
	reopen, inTown, err := load(localOriginalSaveToken(name))
	if err != nil || inTown {
		t.Fatalf("cold LOAD: town=%t %v", inTown, err)
	}
	b := cold.App("fallen mercenary minimap reload")
	t.Cleanup(b.StopAudio)
	b.Layout(1024, 768)
	if err := b.OpenMission(reopen); err != nil {
		t.Fatal(err)
	}
	closeNotices(b)
	checkUnderFog(cold, b, "after SAVE and cold LOAD")
	t.Logf("elapsed %v", time.Since(began))
}
