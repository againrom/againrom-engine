package sim

import (
	"bytes"
	"testing"
)

func TestProjectilePhaseObservationKeepsCanonicalState(t *testing.T) {
	w := shotWorld(t)
	id, ok := w.ReleaseAreaBurst(AreaBurst{CellX: 3, CellY: 4, Picture: 51, Segments: 16, Phases: 16, Owner: 7})
	if !ok {
		t.Fatal("burst producer refused")
	}
	var silent World
	if err := silent.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	for phase := int32(1); phase <= 20; phase++ {
		report := StepReported(w, nil)
		Step(&silent, nil)
		if !bytes.Equal(mustMarshal(t, w), mustMarshal(t, &silent)) || w.Hash() != silent.Hash() {
			t.Fatalf("phase %d: observation changed World bytes or hash", phase)
		}
		if phase == 15 {
			if p, ok := savedProjectileOf(w, id); !ok || p.ActionSegments != 0 || p.ActionPhase != 15 {
				t.Fatalf("last positive driver call left %+v, present=%v", p, ok)
			}
		}
		if phase >= 16 {
			if p, ok := savedProjectileOf(w, id); ok {
				t.Fatalf("record outlived its removal call: %+v", p)
			}
		}
		if phase != 8 {
			if len(report.ProjectilePhases) != 0 {
				t.Fatalf("phase %d emitted %+v", phase, report.ProjectilePhases)
			}
			continue
		}
		want := ProjectilePhase{ID: id, Picture: 51, ActionPhase: 8, Owner: 7, CellX: 3, CellY: 4}
		if len(report.ProjectilePhases) != 1 || report.ProjectilePhases[0] != want {
			t.Fatalf("phase boundary %+v, want %+v", report.ProjectilePhases, want)
		}
	}
}

func TestProjectilePhaseSoundRunsOnTheLastPositiveSegment(t *testing.T) {
	w := shotWorld(t)
	id, _ := w.ReleaseAreaBurst(AreaBurst{CellX: 3, CellY: 4, Picture: 51, Segments: 16, Phases: 16})
	p := w.savedProjectile(id)
	p.ActionPhase, p.Phase, p.ActionSegments = 7, 7, 1
	report := StepReported(w, nil)
	if len(report.ProjectilePhases) != 1 || report.ProjectilePhases[0].ID != id {
		t.Fatalf("last positive segment lost its callback: %+v", report.ProjectilePhases)
	}
	if p, ok := savedProjectileOf(w, id); !ok || p.ActionPhase != 8 || p.ActionSegments != 0 {
		t.Fatalf("sound boundary retired too early: %+v, present=%v", p, ok)
	}
	if report := StepReported(w, nil); len(report.ProjectilePhases) != 0 {
		t.Fatalf("removal call replayed its callback: %+v", report.ProjectilePhases)
	}
}

func TestProjectilePhaseObservationRequiresALiveFirstGameRecord(t *testing.T) {
	for _, tc := range []struct {
		name              string
		second            bool
		picture, segments int32
	}{
		{"expired", false, 51, 0},
		{"other picture", false, 17, 16},
		{"other game", true, 51, 16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := shotWorld(t)
			if tc.second {
				w.rom2 = &rom2ScriptState{}
			}
			w.ReleaseAreaBurst(AreaBurst{CellX: 3, CellY: 4, Picture: tc.picture, Segments: tc.segments, Phases: 16})
			for range 20 {
				if report := StepReported(w, nil); len(report.ProjectilePhases) != 0 {
					t.Fatalf("unvoiced record emitted %+v", report.ProjectilePhases)
				}
			}
		})
	}
}
