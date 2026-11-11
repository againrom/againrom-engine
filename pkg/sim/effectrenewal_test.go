package sim

import "testing"

// renewalStep is one attach of a renewal pair: its magnitude and duration.
type renewalStep struct {
	magnitude int32
	duration  uint16
}

// renewSpeed attaches the two steps of rule, as kind EffectSpeed, to a target
// of base speed 10 and returns the World after both.
func renewSpeed(t *testing.T, rule SpellRule, steps [2]renewalStep) *World {
	t.Helper()
	target := spEnt(2, 3, 2)
	target.Speed = 10
	w := spWorld(t, 1, []SpellRule{rule}, spEnt(1, 2, 2), target)
	for i, s := range steps {
		if !w.attachEffect(2, 1, rule, EffectSpeed, s.magnitude, s.duration, EffectDuration) {
			t.Fatalf("attach %d of %+v did not land", i+1, s)
		}
	}
	return w
}

// expireAll runs every attached record out and reports the target's speed.
func expireAll(w *World) int32 {
	for i := range w.attached {
		w.attached[i].Remaining = 1
	}
	w.stepAttachedEffects()
	return w.entities[1].Speed
}

func TestSecondGameRenewalKeepsTheLargerMagnitudeAndDuration(t *testing.T) {
	haste := secondRow(SpellRule{ID: 21, TargetsUnit: true, SpellDuration: 5, EffectKind: EffectSpeed, EffectMode: EffectDuration})
	slow := secondRow(SpellRule{ID: 29, TargetsUnit: true, SpellDuration: 5, EffectKind: EffectSpeed, EffectMode: EffectDuration})
	for _, tc := range []struct {
		name      string
		rule      SpellRule
		steps     [2]renewalStep
		speed     int32
		remaining uint16
	}{
		{"weaker Haste after a stronger", haste, [2]renewalStep{{5, 100}, {2, 300}}, 15, 300},
		{"stronger Haste after a weaker", haste, [2]renewalStep{{2, 300}, {5, 100}}, 15, 300},
		{"weaker Slow after a stronger", slow, [2]renewalStep{{-5, 300}, {-2, 100}}, 8, 300},
		{"stronger Slow after a weaker", slow, [2]renewalStep{{-2, 100}, {-5, 300}}, 8, 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := renewSpeed(t, tc.rule, tc.steps)
			got := w.ActiveEffects()
			if len(got) != 1 || w.entities[1].Speed != tc.speed || got[0].Remaining != tc.remaining || got[0].Magnitude != tc.speed-10 {
				t.Fatalf("speed %d, records %+v; want speed %d, one record of magnitude %d and remainder %d",
					w.entities[1].Speed, got, tc.speed, tc.speed-10, tc.remaining)
			}
			if speed := expireAll(w); speed != 10 || len(w.attached) != 0 {
				t.Fatalf("after expiry speed %d with %d records, want base 10 and none", speed, len(w.attached))
			}
		})
	}
}

func TestSecondGameRenewalUnderAClampReturnsToTheBase(t *testing.T) {
	shield := secondRow(SpellRule{ID: 4, TargetsUnit: true, SpellDuration: 5, EffectKind: EffectProtectionFire, EffectMode: EffectDuration})
	target := spEnt(2, 3, 2)
	target.Protection[0] = 90
	w := spWorld(t, 1, []SpellRule{shield}, spEnt(1, 2, 2), target)
	w.attachEffect(2, 1, shield, EffectProtectionFire, 20, 100, EffectDuration)
	w.attachEffect(2, 1, shield, EffectProtectionFire, 15, 300, EffectDuration)
	if got := w.entities[1].Protection[0]; got != 100 || len(w.attached) != 1 || w.attached[0].Remaining != 300 {
		t.Fatalf("protection %d, records %+v; want 100 and one record of remainder 300", got, w.ActiveEffects())
	}
	w.attached[0].Remaining = 1
	w.stepAttachedEffects()
	if got := w.entities[1].Protection[0]; got != 90 {
		t.Fatalf("after expiry protection %d, want base 90", got)
	}
}

func TestSecondGamePoisonCloudRenewalAppliesNothing(t *testing.T) {
	poison := secondRow(SpellRule{ID: 6, School: 2, Area: true, Distribution: 3, SpellDuration: 5,
		EffectKind: EffectHealth, EffectMode: EffectContinuous, EffectMagnitude: -2, EffectDuration: 8})
	w := spWorld(t, 1, []SpellRule{poison}, spEnt(1, 0, 0), spEnt(2, 1, 0))
	w.attachEffect(2, 1, poison, EffectHealth, -6, 100, EffectContinuous)
	hp := w.entities[1].HP
	w.attachEffect(2, 1, poison, EffectHealth, -3, 300, EffectContinuous)
	got := w.ActiveEffects()
	if w.entities[1].HP != hp || len(got) != 1 || got[0].Magnitude != -3 || got[0].Remaining != 300 {
		t.Fatalf("health %d (was %d), records %+v; want health unchanged and one record -3 for 300", w.entities[1].HP, hp, got)
	}
}

func TestSecondGameSourceRenewalKeepsTheLargerMagnitudeAndDuration(t *testing.T) {
	haste := secondRow(SpellRule{ID: 21, TargetsUnit: true, SpellDuration: 5, EffectKind: EffectSpeed, EffectMode: EffectDuration})
	for _, tc := range []struct {
		name  string
		steps [2]renewalStep
	}{
		{"weaker after a stronger", [2]renewalStep{{5, 100}, {2, 300}}},
		{"stronger after a weaker", [2]renewalStep{{2, 300}, {5, 100}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := func(steps ...renewalStep) *World {
				target := spEnt(2, 3, 2)
				target.ActorLoad = ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 1}}
				target.ActorLoad.Source.Stats[4] = 10
				target.Capacity = 100
				w := spWorld(t, 1, []SpellRule{haste}, spEnt(1, 2, 2), target)
				w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
				for i, s := range steps {
					if !w.attachEffect(2, 1, haste, EffectSpeed, s.magnitude, s.duration, EffectDuration) {
						t.Fatalf("attach %d did not land", i+1)
					}
				}
				return w
			}
			// The control holds the kept magnitude from one attach.
			w, control := world(tc.steps[:]...), world(renewalStep{5, 300})
			speed := func(w *World) uint16 { return w.entities[1].ActorLoad.Source.Stats[4] }
			got := w.ActiveEffects()
			if speed(w) != speed(control) || len(got) != 1 || got[0].Magnitude != 5 || got[0].Remaining != 300 {
				t.Fatalf("source speed %d, records %+v; want %d (one attach of 5) and one record 5 for 300", speed(w), got, speed(control))
			}
			w.attached[0].Remaining, control.attached[0].Remaining = 1, 1
			w.stepAttachedEffects()
			control.stepAttachedEffects()
			if speed(w) != speed(control) || len(w.attached) != 0 {
				t.Fatalf("after expiry source speed %d with %d records, want %d and none", speed(w), len(w.attached), speed(control))
			}
		})
	}
}

func TestFirstGameRenewalStillReplaces(t *testing.T) {
	haste := SpellRule{ID: 24, TargetsUnit: true, SpellDuration: 5, EffectKind: EffectSpeed, EffectMode: EffectDuration}
	w := renewSpeed(t, haste, [2]renewalStep{{5, 300}, {2, 100}})
	got := w.ActiveEffects()
	if len(got) != 1 || w.entities[1].Speed != 12 || got[0].Remaining != 100 {
		t.Fatalf("first-game Haste repeat left speed %d, records %+v; want the replacing record 2 for 100", w.entities[1].Speed, got)
	}
	if speed := expireAll(w); speed != 10 {
		t.Fatalf("after expiry speed %d, want 10", speed)
	}
}
