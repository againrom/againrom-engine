package game

import (
	"bytes"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func currentScriptRoleSource() alm.Script {
	return alm.Script{
		Actions: []alm.ScriptNode{{ID: 10, Opcode: uint32(sim.ScriptInstantGiveUnit),
			Type: [10]uint32{4, 3}, Value: [10]uint32{10002, 8}}},
		Conditions: []alm.ScriptNode{
			{ID: 1, Opcode: uint32(sim.ScriptCheckUnitDistance), Type: [10]uint32{4, 4}, Value: [10]uint32{10001, 10002}},
			{ID: 2, Opcode: uint32(sim.ScriptCheckConstant), Type: [10]uint32{1}, Value: [10]uint32{3}},
			{ID: 3, Opcode: uint32(sim.ScriptCheckAlive), Type: [10]uint32{4}, Value: [10]uint32{77}},
			{ID: 4, Opcode: uint32(sim.ScriptCheckAlive), Type: [10]uint32{4}, Value: [10]uint32{10003}},
		},
		Triggers: []alm.ScriptTrigger{{Left: [3]uint32{1}, Right: [3]uint32{2}, Cmp: [3]uint32{uint32(sim.ScriptCmpLT)}, Acts: [4]uint32{10}, Once: 1}},
	}
}

func currentScriptRoleProgram(t *testing.T, refs mapload.ScriptRefs) *sim.Script {
	t.Helper()
	s, _, err := mapload.CompileScriptFrom(currentScriptRoleSource(), refs)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func currentScriptRoleWorld(t *testing.T, program *sim.Script) *sim.World {
	t.Helper()
	w, err := sim.NewScriptedWorld(11, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, make([]byte, 16*16), []sim.Entity{
		{ID: 0, Owner: sim.SelfSlot, X: 5, Y: 3, HP: 100, MaxHP: 100},
		{ID: 1, Owner: sim.SelfSlot, X: 3, Y: 3, HP: 100, MaxHP: 100},
		{ID: 7, Owner: 2, X: 4, Y: 4, HP: 100, MaxHP: 100},
	}, program)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestCurrentScriptRolesKeepExactIdentityAndSuccessor(t *testing.T) {
	for _, companion := range []sim.EntityID{0, 99} {
		refs := mapload.ScriptRefs{Units: map[uint16]sim.EntityID{77: 7}, Hero: 1, HasHero: true}
		world := currentScriptRoleWorld(t, currentScriptRoleProgram(t, refs))
		refs.Companion, refs.HasCompanion = companion, true
		want := currentScriptRoleWorld(t, currentScriptRoleProgram(t, refs))
		roles := currentScriptRoleProgram(t, mapload.ScriptRefs{Hero: 1, HasHero: true, Companion: companion, HasCompanion: true})
		if err := restoreCurrentScriptRoles(world, roles); err != nil {
			t.Fatal(err)
		}
		if world.Hash() != want.Hash() {
			t.Fatal("role repair changed more than the exact compiled endpoint", companion)
		}
		for range 32 {
			sim.Step(world, nil)
			sim.Step(want, nil)
			if world.Hash() != want.Hash() {
				t.Fatal("role successor differs", companion, world.Tick())
			}
		}
		checks := world.Script().Checks()
		if !checks[0].HasUnit2 || checks[0].Unit2 != companion || checks[2].Unit != 7 || !checks[2].HasUnit || checks[3].HasUnit {
			t.Fatal("role repair replaced an ordinary map reference or invented an unknown role", checks)
		}
		if companion == 0 && world.Entities()[0].Owner != 8 {
			t.Fatal("rebound entity-zero companion did not execute Give Unit")
		}
		next, available := world.NextEntityID()
		if companion == 99 && (world.ScriptRegister(0) != 255 || !available || next != 100) {
			t.Fatal("detached endpoint lost missing-target behavior or ID reservation")
		}
	}
}

func TestCurrentScriptRolesPreserveBoundReferencesAndChangedProgram(t *testing.T) {
	refs := mapload.ScriptRefs{Units: map[uint16]sim.EntityID{77: 7}, Hero: 1, HasHero: true, Companion: 7, HasCompanion: true}
	bound := currentScriptRoleProgram(t, refs)
	roles := currentScriptRoleProgram(t, mapload.ScriptRefs{Hero: 1, HasHero: true, Companion: 0, HasCompanion: true})
	world := currentScriptRoleWorld(t, bound)
	before := world.Hash()
	if err := restoreCurrentScriptRoles(world, roles); err != nil || world.Hash() != before {
		t.Fatal("current bound reference was replaced by party policy", err)
	}
	for range 32 {
		sim.Step(world, nil)
	}
	if world.Entities()[2].Owner != 8 || world.Entities()[0].Owner != sim.SelfSlot {
		t.Fatal("successor ignored the already-bound ordinary endpoint")
	}
	refs.Companion, refs.HasCompanion = 0, false
	base := currentScriptRoleProgram(t, refs)
	checks, instants := base.Checks(), base.Instants()
	instants[0].Player = 3
	changed, err := sim.NewScript(checks, instants, base.Triggers())
	if err != nil {
		t.Fatal(err)
	}
	world = currentScriptRoleWorld(t, changed)
	old, _ := world.MarshalBinary()
	if err := restoreCurrentScriptRoles(world, roles); err != nil {
		t.Fatal(err)
	}
	next, _ := world.MarshalBinary()
	if !bytes.Equal(old, next) {
		t.Fatal("late mismatched instruction partially restored earlier role references")
	}
}

func TestCurrentScriptPartyRefsUseSuppliedIdentity(t *testing.T) {
	party := []mapload.PartyMember{heroMember("hero", false, true, 3, starting), heroMember("npc:22", true, true, 1)}
	ids := []sim.EntityID{91, 0}
	refs := campaignScriptPartyRefs(&alm.Map{}, shippedHeroTable(t), party, func(i int) sim.EntityID { return ids[i] }, nil)
	if !refs.HasHero || refs.Hero != 91 || !refs.HasCompanion || refs.Companion != 0 || refs.Units != nil {
		t.Fatal("current party role used construction ordinals instead of final IDs", refs)
	}
}

func TestStatueQuestRoleResolvesByTheMaleMageFaceFourPredicate(t *testing.T) {
	src := alm.Script{Conditions: []alm.ScriptNode{
		{ID: 1, Opcode: uint32(sim.ScriptCheckAlive), Type: [10]uint32{4}, Value: [10]uint32{10006}},
		{ID: 2, Opcode: uint32(sim.ScriptCheckAlive), Type: [10]uint32{4}, Value: [10]uint32{10005}},
	}}
	table := shippedHeroTable(t)
	for _, position := range []int{1, 4} {
		party := []mapload.PartyMember{heroMember("hero", true, true, 1, starting), {ID: "other"}, {ID: "hire", MercenaryType: 1},
			{ID: "another"}, {ID: "last"}, {ID: "sixth"}}
		party[position] = heroMember("rood", false, true, 4)
		ids := []sim.EntityID{91, 42, 13, 17, 0, 78}
		refs := campaignScriptPartyRefs(&alm.Map{}, table, party, func(i int) sim.EntityID { return ids[i] }, nil)
		program, _, err := mapload.CompileScriptFrom(src, refs)
		if err != nil {
			t.Fatal(err)
		}
		checks := program.Checks()
		if !checks[0].HasUnit || checks[0].Unit != ids[position] {
			t.Fatalf("position=%d checks=%+v", position, checks)
		}
		party[position] = heroMember("rood", false, true, 3)
		refs = campaignScriptPartyRefs(&alm.Map{}, table, party, func(i int) sim.EntityID { return ids[i] }, nil)
		_, rep, err := mapload.CompileScriptFrom(src, refs)
		if err != nil || !slices.Contains(rep.OmittedChecks, 1) {
			t.Fatal("a male mage with another face satisfied ordinal 5", err, rep.OmittedChecks)
		}
	}
}
