package sim

import "testing"

func TestTheTargetIDArmAnswersZeroForAnAcquisitionTurnAndAnIdleOrder(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		acquire    bool
		idle       bool
		wantTarget bool
	}{
		{"control: pathing pursuit", false, false, true},
		{"acquisition turn", true, false, false},
		{"idle order holding the victim", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := mustScript(t, []ScriptCheck{targetIDNode(caSubject)}, nil, nil)
			w, err := NewScriptedWorld(1, caBounds, ModeCanonical, nil,
				[]Entity{
					{ID: caSubject, X: 1, Y: 1, HP: 5, MaxHP: 5, MapUnitID: caSubjectMapID},
					{ID: caTarget, X: 40, Y: 40, HP: 5, MaxHP: 5, MapUnitID: caTargetMapID},
				}, s)
			if err != nil {
				t.Fatalf("NewScriptedWorld: %v", err)
			}
			scriptTicks(w, 1, []Command{Attack(caSubject, caTarget)})
			i := indexOfEntity(w.entities, caSubject)
			w.entities[i].AcquirePursuit, w.entities[i].PursuitIdle = tc.acquire, tc.idle
			scriptTicks(w, caTicks, nil)
			if e := w.entities[indexOfEntity(w.entities, caSubject)]; !e.HasAttackTarget || e.AttackTarget != caTarget {
				t.Fatalf("the subject holds victim %v/%d, want the victim held", e.HasAttackTarget, e.AttackTarget)
			}
			want := int32(0)
			if tc.wantTarget {
				want = int32(caTargetMapID)
			}
			if got := w.ScriptRegister(caRegister); got != want {
				t.Errorf("register %d is %d, want %d", caRegister, got, want)
			}
		})
	}
}
