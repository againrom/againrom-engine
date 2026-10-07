package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	cursorFireArrow = 1
	cursorFireBall  = 2
	cursorSacrifice = 4
	cursorShield    = 18
	cursorTeleport  = 26
)

func TestReleaseCastCursorFollowsTheSpellKind(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[5] = 100
	known := uint32(1<<cursorFireArrow | 1<<cursorFireBall | 1<<cursorShield | 1<<cursorTeleport | 1<<cursorSacrifice)
	mage := func(id string, x, y int, start bool) mapload.PartyMember {
		return mapload.PartyMember{ID: id, PlayerCharacter: start, StartingHero: start, Mage: true, Class: 0x18,
			Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: known,
			Saved: &mapload.Saved{Cell: mapload.Cell{X: int32(x), Y: int32(y)}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000}}
	}
	party := []mapload.PartyMember{mage("mage", 29, 50, true), mage("friend", 33, 50, false)}
	a := f.App("cast cursor by kind")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	if len(live.mission.ids) < 2 {
		t.Fatalf("mission opened with %d party members, want 2", len(live.mission.ids))
	}
	id, friend := live.mission.ids[0], live.mission.ids[1]
	actors := make([]sim.Entity, 0, 2)
	for n, who := range []sim.EntityID{id, friend} {
		e, ok := live.entity(who)
		if !ok {
			t.Fatalf("missing party member %d", n)
		}
		e.X, e.Y = int32(29+4*n), 50
		actors = append(actors, e)
	}
	m := live.mission.state.Map
	w, err := sim.NewStructuredWorld(1, live.world.Bounds(), sim.ModeCanonical,
		sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)},
		actors, nil, live.world.Relations(), nil, nil, mapload.SpellRules(f.Table), sim.GhostTemplate{}, live.world.Structures())
	if err != nil {
		t.Fatal(err)
	}
	live.world = w
	live.push()
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	live.push()

	book := func(open bool) {
		t.Helper()
		_, _, err := a.HeadlessSpellPoint(cursorFireArrow)
		if (err == nil) != open {
			if err := a.HeadlessKey("book"); err != nil {
				t.Fatal(err)
			}
		}
		if _, _, err := a.HeadlessSpellPoint(cursorFireArrow); (err == nil) != open {
			t.Fatalf("book open=%v after the toggle, want %v", err == nil, open)
		}
	}
	click := func(x, y int) {
		t.Helper()
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
	}
	where := func(name string) (int, int) {
		t.Helper()
		inspectionCentre(live, 31, 50)
		live.push()
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		var x, y int
		var err error
		switch name {
		case "caster":
			x, y, err = a.HeadlessEntityPoint(uint32(id))
		case "friend":
			x, y, err = a.HeadlessEntityPoint(uint32(friend))
		default:
			x, y, err = a.HeadlessGroundPoint()
		}
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return x, y
	}
	cursor := func(name string) string {
		t.Helper()
		x, y := where(name)
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		got, ok := a.HeadlessMapCursor()
		if !ok {
			t.Fatal("no map cursor")
		}
		return got
	}
	type kindWant struct{ caster, friend, ground bool }
	kinds := []struct {
		spell uint32
		want  kindWant
		cast  string // where a cast click is made
	}{
		{cursorFireArrow, kindWant{true, true, false}, "friend"},
		{cursorFireBall, kindWant{true, true, true}, "ground"},
		{cursorTeleport, kindWant{true, true, true}, "ground"},
		{cursorShield, kindWant{true, false, false}, "caster"},
		// Last: it spends the caster's health and mana.
		{cursorSacrifice, kindWant{true, false, false}, "caster"},
	}
	check := func(stage string, spell uint32, want kindWant) {
		t.Helper()
		for place, wantCast := range map[string]bool{"caster": want.caster, "friend": want.friend, "ground": want.ground} {
			if got := cursor(place); (got == "cast") != wantCast {
				t.Fatalf("spell %d, %s, over the %s: cursor %q, want cast=%v", spell, stage, place, got, wantCast)
			}
		}
	}
	none := kindWant{}
	for _, k := range kinds {
		book(true)
		sx, sy, err := a.HeadlessSpellPoint(k.spell)
		if err != nil {
			t.Fatal(err)
		}
		click(sx, sy)
		if _, current, armed := live.view.QuickSpellState(); current != k.spell || !armed {
			t.Fatalf("spell %d: the book click selected %d armed %v", k.spell, current, armed)
		}
		check("book open", k.spell, k.want)

		book(false)
		check("book closed", k.spell, none)

		if err := a.HeadlessKey("c"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := a.HeadlessSpellPoint(cursorFireArrow); err != nil {
			t.Fatalf("spell %d: C did not open the book", k.spell)
		}
		check("C hook, book open", k.spell, k.want)
		book(false)
		check("C hook, book closed", k.spell, k.want)

		live.pending = nil
		x, y := where(k.cast)
		click(x, y)
		if len(live.pending) != 1 || castCommandSpell(live.pending[0]) != k.spell {
			t.Fatalf("spell %d: the cast click at the %s queued %+v", k.spell, k.cast, live.pending)
		}
		if _, current, armed := live.view.QuickSpellState(); current != k.spell || armed {
			t.Fatalf("spell %d after the cast: current %d live %v, want selected and spent", k.spell, current, armed)
		}
		if _, _, err := a.HeadlessSpellPoint(cursorFireArrow); err == nil {
			t.Fatalf("spell %d: the book stayed open after the hook's cast", k.spell)
		}
		check("after the hook's cast", k.spell, none)
		live.pending = nil
		for tick := 0; tick < 400; tick++ {
			live.tick()
		}
	}
	if selected, ok := live.view.SelectedUnit(); !ok || selected != uint32(id) {
		t.Fatal("casting changed the selection")
	}
}

func castCommandSpell(c sim.Command) uint32 {
	switch c.Kind {
	case sim.KindCast:
		return uint32(c.Y)
	case sim.KindCastAt:
		return uint32(c.Spell)
	}
	return 0
}

// The installed table names two self spells, Shield by its row and Fire
// Sacrifice by its id; every other book entry keeps its kind.
func TestReleaseSelfSpellKindsAreShieldAndFireSacrifice(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	mage := sim.Entity{KnownSpells: ^uint32(0)}
	var self, point []uint32
	for _, e := range spellbookOf(sim.Rules{}, mage, mapload.SpellRules(f.Table), nil, ui.Words{}, nil) {
		if e.SelfOnly {
			self = append(self, e.ID)
		}
		if e.PointTarget {
			point = append(point, e.ID)
		}
	}
	if !slices.Equal(self, []uint32{cursorSacrifice, cursorShield}) {
		t.Fatalf("self spells %v, want Fire Sacrifice and Shield", self)
	}
	if !slices.Contains(point, uint32(cursorSacrifice)) || slices.Contains(point, uint32(cursorShield)) {
		t.Fatalf("cell-ordered spells %v: Fire Sacrifice must keep its cell order and Shield none", point)
	}
	// Loss control: the row rule alone misses Fire Sacrifice.
	for _, rule := range mapload.SpellRules(f.Table) {
		if rule.ID == cursorSacrifice && spellSelfOnly(rule) {
			t.Fatal("Fire Sacrifice row reads as self by the row rule; the id arm is not needed")
		}
	}
}
