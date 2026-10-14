package game

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// On shipped mission 10 a Fire Ball armed from the book shows the move cursor
// over explored ground out of current sight and a click there queues a move;
// over ground in sight it shows the cast cursor and casts. The second game is
// the control: it casts on both.
func TestReleaseCastNeedsCurrentSight(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	firstGame := f.Archives.Game() == base.GameROM1
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[5] = 100
	party := []mapload.PartyMember{{ID: "mage", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 1 << cursorFireBall,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000}}}
	a := f.App("cast needs current sight")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	if len(live.mission.ids) < 1 || live.fog == nil {
		t.Fatalf("mission opened with %d party members, fog %v", len(live.mission.ids), live.fog != nil)
	}
	id := live.mission.ids[0]
	mage, ok := live.entity(id)
	if !ok {
		t.Fatal("missing the mage")
	}
	mage.X, mage.Y = 29, 50
	m := live.mission.state.Map
	w, err := sim.NewStructuredWorld(1, live.world.Bounds(), sim.ModeCanonical,
		sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)},
		[]sim.Entity{mage}, nil, live.world.Relations(), nil, nil, mapload.SpellRules(f.Table), sim.GhostTemplate{}, live.world.Structures())
	if err != nil {
		t.Fatal(err)
	}
	live.world = w
	live.push()
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	live.push()
	if _, _, err := a.HeadlessSpellPoint(cursorFireBall); err != nil {
		if err := a.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
	sx, sy, err := a.HeadlessSpellPoint(cursorFireBall)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, sx, sy); err != nil {
			t.Fatal(err)
		}
	}
	if _, current, armed := live.view.QuickSpellState(); current != cursorFireBall || !armed {
		t.Fatalf("the book click selected %d armed %v", current, armed)
	}

	for _, c := range []struct {
		name     string
		lit      bool
		wantCast bool
	}{
		{"explored, dark", false, !firstGame},
		{"in sight", true, true},
	} {
		inspectionCentre(live, 31, 50)
		for i := range live.fog.visible {
			live.fog.explored[i] = 1
			live.fog.visible[i] = 0
			if c.lit {
				live.fog.visible[i] = 1
			}
		}
		live.push()
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		x, y, err := a.HeadlessGroundPoint()
		if err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		got, ok := a.HeadlessMapCursor()
		want := "move"
		if c.wantCast {
			want = "cast"
		}
		t.Logf("root=%s ground=%s pointer=(%d,%d) cursor=%s", f.Archives.Root, c.name, x, y, got)
		if !ok || got != want {
			t.Fatalf("%s: cursor %q, want %q", c.name, got, want)
		}
		live.pending = nil
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		if len(live.pending) != 1 {
			t.Fatalf("%s: the click queued %+v, want one command", c.name, live.pending)
		}
		cmd := live.pending[0]
		t.Logf("root=%s ground=%s command kind=%d spell=%d", f.Archives.Root, c.name, cmd.Kind, castCommandSpell(cmd))
		if c.wantCast && castCommandSpell(cmd) != cursorFireBall || !c.wantCast && cmd.Kind != sim.KindGroupMoveTo {
			t.Fatalf("%s: the click queued %+v, want cast=%v", c.name, cmd, c.wantCast)
		}
		if _, current, armed := live.view.QuickSpellState(); current != cursorFireBall || !armed {
			t.Fatalf("%s: after the click the spell is %d armed %v, want Fire Ball kept", c.name, current, armed)
		}
		live.pending = nil
	}
}
