package game

import (
	"testing"

	"againrom/pkg/sim"
)

// scriptStandMission is an installed mission whose script holds Stand Ground
// group commands.
const scriptStandMission = 40

type scriptStandArena struct {
	front   *FrontEnd
	live    *mapWorld
	node    sim.ScriptInstant
	members []sim.EntityID
}

func openScriptStandArena(t *testing.T) []*scriptStandArena {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	app := f.App("script stand ground")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(scriptStandMission, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	var out []*scriptStandArena
	for _, in := range live.world.Script().Instants() {
		if in.Op != sim.ScriptInstantGroupOrder || !in.HasGroup || in.Args[0] != 3 {
			continue
		}
		a := &scriptStandArena{front: f, live: live, node: in}
		for _, e := range live.world.Entities() {
			if e.Alive() && !e.OffMap && e.Group == in.Group && e.Owner != 0 && e.Owner != sim.SelfSlot {
				a.members = append(a.members, e.ID)
			}
		}
		out = append(out, a)
	}
	return out
}

// walkers sends the group on a walk through the script's Move command.
func (a *scriptStandArena) walkers(t *testing.T) *sim.World {
	t.Helper()
	e, _ := a.live.entity(a.members[0])
	move := a.node
	move.Args[0], move.Args[1], move.Args[2] = 4, e.X+6, e.Y
	started, _ := scriptStandPrograms(t, a.live.world, move)
	for range 9 {
		sim.Step(started, nil)
	}
	return started
}

func scriptStandPrograms(t *testing.T, base *sim.World, in sim.ScriptInstant) (treated, control *sim.World) {
	t.Helper()
	program, err := sim.NewScript(nil, []sim.ScriptInstant{in}, []sim.ScriptTrigger{{
		Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	empty, err := sim.NewScript(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if treated, err = sim.NewControlledScriptWorld(base, program); err != nil {
		t.Fatal(err)
	}
	if control, err = sim.NewControlledScriptWorld(base, empty); err != nil {
		t.Fatal(err)
	}
	return treated, control
}

// The installed mission's Stand Ground command stands a group on a walk; the
// control world without the node keeps walking. SAVE and cold LOAD follow.
func TestReleaseScriptStandGroundStopsAGroupOnTheMove(t *testing.T) {
	arenas := openScriptStandArena(t)
	if len(arenas) == 0 {
		t.Fatalf("mission %d holds no Stand Ground group command", scriptStandMission)
	}
	witnessed := 0
	for _, a := range arenas {
		if len(a.members) < 2 {
			continue
		}
		base := a.walkers(t)
		treated, control := scriptStandPrograms(t, base, a.node)
		owner := func() uint32 { e, _ := treated.Entity(a.members[0]); return e.Owner }()
		fired := false
		for k := 0; k < 40 && !fired; k++ {
			sim.Step(treated, nil)
			sim.Step(control, nil)
			order, _, _ := treated.FrozenGroupAI(owner, a.node.Group)
			fired = order == 3
		}
		if !fired {
			t.Fatalf("group %d: the node never ran", a.node.Group)
		}
		stood := map[sim.EntityID]sim.Entity{}
		walking := 0
		for _, id := range a.members {
			e, _ := treated.Entity(id)
			c, _ := control.Entity(id)
			if e.HasTarget || e.HasAttackTarget {
				t.Fatalf("group %d member %d still holds destination %v victim %v after Stand Ground", a.node.Group, id, e.HasTarget, e.HasAttackTarget)
			}
			if c.HasTarget {
				walking++
			}
			stood[id] = e
		}
		if walking == 0 {
			t.Fatalf("group %d: no member of the control world was on a walk", a.node.Group)
		}
		for range 60 {
			sim.Step(treated, nil)
		}
		for id, e := range stood {
			if got, _ := treated.Entity(id); got.X != e.X || got.Y != e.Y || got.HasTarget {
				t.Fatalf("group %d member %d moved from (%d,%d) to (%d,%d) after Stand Ground", a.node.Group, id, e.X, e.Y, got.X, got.Y)
			}
		}
		form, err := treated.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold sim.World
		if err := cold.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		if cold.Hash() != treated.Hash() {
			t.Fatalf("group %d: the byte form resumes to a different hash", a.node.Group)
		}
		a.live.world = treated
		snapshot, label, err := a.front.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		hash := treated.Hash()
		raw, err := a.front.ExportCurrentSave(snapshot, label)
		if err != nil || treated.Hash() != hash {
			t.Fatal("SAVE failed or changed the World", err)
		}
		back := releaseFront(t)
		back.Options = OptionsStore{}
		back.SetDeterministicFrames(true)
		opener, town, err := back.RestoreOriginal(raw)
		if err != nil || town || opener == nil {
			t.Fatal("LOAD", err, town)
		}
		app := back.App("script stand ground LOAD")
		t.Cleanup(app.StopAudio)
		app.Layout(1024, 768)
		if err := app.OpenMission(opener); err != nil {
			t.Fatal(err)
		}
		assertEntitiesEqual(t, treated, back.live.world, a.members, "cold LOAD of the stood group")
		for id := range stood {
			want, _ := treated.Entity(id)
			if got, _ := back.live.entity(id); got.HasTarget || got.ActorState != want.ActorState || got.HasAttackTarget != want.HasAttackTarget {
				t.Fatalf("loaded member %d holds destination %v victim %v state %#x, want state %#x victim %v and no destination", id, got.HasTarget, got.HasAttackTarget, got.ActorState, want.ActorState, want.HasAttackTarget)
			}
		}
		for range 8 {
			back.live.tick()
		}
		for id, e := range stood {
			if got, _ := back.live.entity(id); got.X != e.X && got.Y != e.Y {
				t.Fatalf("loaded member %d left (%d,%d) for (%d,%d)", id, e.X, e.Y, got.X, got.Y)
			}
		}
		witnessed++
		t.Logf("group %d: %d members stood, %d of them on a walk in the control world", a.node.Group, len(a.members), walking)
	}
	if witnessed == 0 {
		t.Fatal("no Stand Ground node names a group of two or more members")
	}
}
