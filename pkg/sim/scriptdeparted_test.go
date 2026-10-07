package sim

import "testing"

// Every body here dies through an attack order and leaves through ticks.

const (
	dpFighter EntityID = 1
	dpVictim  EntityID = 2
)

func dpEnts(domain Domain, blow int32) []Entity {
	f := cbFighter(dpFighter, 1, 1, 1, 0)
	f.DamageBase, f.Domain = blow, domain
	v := Entity{ID: dpVictim, X: 2, Y: 1, HP: 5, MaxHP: 5, DyingTime: 20, Domain: domain}
	return []Entity{f, v}
}

func dpBoundEnts(domain Domain, blow int32) []Entity {
	ents := dpEnts(domain, blow)
	ents[1].MapUnitID = 9
	ents[1].Capacity = 100
	ents[1].SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 2, Identity: 0x10000002, RuntimeID: 31}
	ents[1].ActorLoad = ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 1}}
	return ents
}

func dpAttack(w *World) {
	scriptTicks(w, scriptCycle, nil)
	Step(w, []Command{Attack(dpFighter, dpVictim)})
}

func dpUntilGone(t *testing.T, w *World, limit int) {
	t.Helper()
	for n := 0; indexOfEntity(w.entities, dpVictim) >= 0; n++ {
		if n > limit {
			t.Fatalf("victim still held after %d ticks: %+v", limit, w.entities[indexOfEntity(w.entities, dpVictim)])
		}
		Step(w, nil)
	}
}

func dpReload(t *testing.T, w *World) *World {
	t.Helper()
	var loaded World
	if err := loaded.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	return &loaded
}

func dpBoth(t *testing.T, live, loaded *World, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		Step(live, nil)
		Step(loaded, nil)
	}
	if live.Hash() != loaded.Hash() {
		t.Fatal("the reloaded world diverged from the live one")
	}
}

func dpAliveDistanceScript(t *testing.T) *Script {
	return mustScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckAlive, Register: 0, Unit: dpVictim, HasUnit: true},
			{Op: ScriptCheckUnitDistance, Register: 1, Unit: dpVictim, HasUnit: true, Unit2: dpFighter, HasUnit2: true},
			constCheck(2, 0),
			constCheck(3, 0xff),
		},
		nil,
		[]ScriptTrigger{
			{Pairs: [3]ScriptPair{pair(0, 2, ScriptCmpEQ)}, Instants: acts(), Once: true},
			{Pairs: [3]ScriptPair{pair(1, 3, ScriptCmpEQ)}, Instants: acts(), Once: true, Latch: 1},
		})
}

func dpRequireDeadAnswers(t *testing.T, w *World, label string) {
	t.Helper()
	if r0, r1 := w.ScriptRegister(0), w.ScriptRegister(1); r0 != 0 || r1 != 0xff {
		t.Errorf("%s: Alive %d, UnitDistance %d; want 0 and 0xff", label, r0, r1)
	}
	if !w.ScriptLatched(0) || !w.ScriptLatched(1) {
		t.Errorf("%s: Alive trigger %v, UnitDistance trigger %v; want both fired", label, w.ScriptLatched(0), w.ScriptLatched(1))
	}
}

func TestAFlyingUnitsDeathReachesAliveAndUnitDistanceAfterItLeaves(t *testing.T) {
	w, err := NewScriptedWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, dpEnts(DomainAir, 10), dpAliveDistanceScript(t))
	if err != nil {
		t.Fatal(err)
	}
	dpAttack(w)
	dpUntilGone(t, w, 64)
	if w.ScriptRegister(0) != 1 || w.ScriptLatched(0) {
		t.Fatalf("the dying window already read dead: Alive %d latched %v", w.ScriptRegister(0), w.ScriptLatched(0))
	}
	loaded := dpReload(t, w)
	dpBoth(t, w, loaded, scriptCycle)
	dpRequireDeadAnswers(t, w, "live")
	dpRequireDeadAnswers(t, loaded, "after LOAD")
}

func TestASourceBoundFlyingUnitLeavesAScriptReadableDeadRow(t *testing.T) {
	w, err := NewScriptedWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, dpBoundEnts(DomainAir, 10), dpAliveDistanceScript(t))
	if err != nil {
		t.Fatal(err)
	}
	dpAttack(w)
	dpUntilGone(t, w, 64)
	rows := w.OriginalDeadActors()
	if len(rows) != 1 || rows[0].ID != dpVictim || rows[0].Source.Identity != 0x10000002 {
		t.Fatalf("source-bound actor left no retained row: %+v", rows)
	}
	loaded := dpReload(t, w)
	dpBoth(t, w, loaded, scriptCycle)
	dpRequireDeadAnswers(t, w, "live")
	dpRequireDeadAnswers(t, loaded, "after LOAD")
}

func TestAGroundUnitPastItsLastDecayStageKeepsItsDeadAnswers(t *testing.T) {
	// -590 after the blow; the ladder walks it past -600.
	w, err := NewScriptedWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, dpEnts(DomainGround, 595), dpAliveDistanceScript(t))
	if err != nil {
		t.Fatal(err)
	}
	dpAttack(w)
	for n := 0; n < 64 && w.entities[indexOfEntity(w.entities, dpVictim)].Decay != decayLast; n++ {
		Step(w, nil)
	}
	if got := w.entities[indexOfEntity(w.entities, dpVictim)]; got.Decay != decayLast {
		t.Fatalf("the body never reached the last stage: %+v", got)
	}
	scriptTicks(w, scriptCycle, nil)
	dpRequireDeadAnswers(t, w, "on the last stage")
	dpUntilGone(t, w, 32*decayCycle)
	loaded := dpReload(t, w)
	dpBoth(t, w, loaded, 1)
	for _, c := range []struct {
		label string
		w     *World
	}{{"live", w}, {"after LOAD", loaded}} {
		if tr := sttPass(c.w); len(tr.Silent) != 0 {
			t.Errorf("%s: a pass past the last stage left checks silent: %+v", c.label, tr.Silent)
		}
		dpRequireDeadAnswers(t, c.w, c.label)
	}
	if w.Hash() != loaded.Hash() {
		t.Error("the reloaded world diverged from the live one")
	}
}

func TestAFlyingVIPsDeathCountsTheLossAfterItLeaves(t *testing.T) {
	s := mustScript(t, []ScriptCheck{{Op: ScriptCheckVIP, Register: 0, Unit: dpVictim, HasUnit: true}}, nil, nil)
	for _, reload := range []bool{false, true} {
		w, err := NewScriptedWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, dpEnts(DomainAir, 10), s)
		if err != nil {
			t.Fatal(err)
		}
		dpAttack(w)
		dpUntilGone(t, w, 64)
		if _, lost := w.ScriptCounters(); lost != 0 {
			t.Fatalf("the dying window counted %d losses", lost)
		}
		if reload {
			w = dpReload(t, w)
		}
		scriptTicks(w, 2*scriptCycle, nil)
		if _, lost := w.ScriptCounters(); lost == 0 || w.Outcome() != OutcomeLost {
			t.Errorf("reload %v: lose counter %d, outcome %d; want a counted loss", reload, lost, w.Outcome())
		}
	}
}

func TestAControlSpiritCorpseStaysOnTheDeadList(t *testing.T) {
	// -12 after the blow: bones.
	ents := dpEnts(DomainGround, 17)
	ents = append(ents, effectMage(3, 1, 2, 1<<25))
	s := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckHealth, Register: 0, Unit: dpVictim, HasUnit: true, Args: [scriptParams]int32{scriptCheckHealthGate}},
		{Op: ScriptCheckAlive, Register: 1, Unit: dpVictim, HasUnit: true},
	}, nil, nil)
	w, err := NewSummoningWorld(51, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, ents, s, Relations{}, nil, nil,
		[]SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}}, hlGhostTemplate())
	if err != nil {
		t.Fatal(err)
	}
	dpAttack(w)
	for n := 0; n < 64 && w.entities[indexOfEntity(w.entities, dpVictim)].Decay != DecayBones; n++ {
		Step(w, nil)
	}
	if got := w.ScriptRegister(0); got >= 0 {
		t.Fatalf("the health check read %d on the body", got)
	}
	spRunCast(w, Cast(3, dpVictim, 25))
	if indexOfEntity(w.entities, dpVictim) >= 0 {
		t.Fatal("Control Spirit left the corpse in the world")
	}
	loaded := dpReload(t, w)
	dpBoth(t, w, loaded, scriptCycle)
	for _, c := range []*World{w, loaded} {
		if hp, alive := c.ScriptRegister(0), c.ScriptRegister(1); hp != controlSpiritCorpseHP || alive != 0 {
			t.Errorf("consumed corpse reads health %d, alive %d; want %d and 0", hp, alive, controlSpiritCorpseHP)
		}
	}
}
