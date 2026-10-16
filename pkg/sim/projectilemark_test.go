package sim

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func markedProjectileWorld(t *testing.T, picture, segments int32) *World {
	t.Helper()
	w := shotWorld(t, shotPair(4)...)
	w.SetSavedProjectiles(SavedProjectiles{FreeIndex: 8, IDs: []uint16{7}, Items: []SavedProjectile{
		{ID: 7, Picture: picture, X: 384, Y: 1152, Action: 1, ActionTarget: 2,
			ActionX: 1408, ActionY: 1152, ActionSegments: segments},
	}})
	w.savedWorldEffects.Projectiles = []SavedProjectileDriver{{ID: 7, Phases: 4, Target: 2, HasTarget: true}}
	return w
}

func TestProjectileMarksFollowPositiveResolvedActorCalls(t *testing.T) {
	for _, picture := range []int32{20, 30} {
		t.Run(fmt.Sprint(picture), func(t *testing.T) {
			w := markedProjectileWorld(t, picture, 2)
			want := ProjectileMark{Target: 2, Picture: picture}
			for call := 1; call <= 3; call++ {
				report := StepReported(w, nil)
				if call <= 2 {
					if !reflect.DeepEqual(report.ProjectileMarks, []ProjectileMark{want}) {
						t.Fatalf("positive call %d: marks %+v, want %+v", call, report.ProjectileMarks, want)
					}
				} else if len(report.ProjectileMarks) != 0 {
					t.Fatalf("removal call replayed mark %+v", report.ProjectileMarks)
				}
			}
		})
	}
}

func TestProjectileMarksExcludeCallsWithoutActorCallback(t *testing.T) {
	for _, name := range []string{"zero segments", "other picture", "second game", "no target", "unresolved", "detached", "structure"} {
		t.Run(name, func(t *testing.T) {
			w := markedProjectileWorld(t, 20, 2)
			p, d := w.savedProjectile(7), &w.savedWorldEffects.Projectiles[0]
			switch name {
			case "zero segments":
				p.ActionSegments = 0
			case "other picture":
				p.Picture = 18
			case "second game":
				w.rom2 = &rom2ScriptState{}
			case "no target":
				d.HasTarget, d.Target, p.ActionTarget = false, 0, 0
			case "unresolved":
				d.Target, p.ActionTarget = 9, 9
			case "detached":
				w.detachSavedProjectileTargets(2)
			case "structure":
				w.DeclareStructures([]Structure{{ID: 40, Col: 5, Row: 4, Width: 1, Height: 1, MaxHealth: 50}})
				d.Target, d.TargetStructure, p.ActionTarget = 40, true, 40
			}
			if got := StepReported(w, nil).ProjectileMarks; len(got) != 0 {
				t.Fatalf("call without actor callback emitted %+v", got)
			}
		})
	}
}

func TestProjectileMarkReportKeepsCanonicalState(t *testing.T) {
	for _, picture := range []int32{20, 30} {
		t.Run(fmt.Sprint(picture), func(t *testing.T) {
			observed := markedProjectileWorld(t, picture, 3)
			var silent World
			if err := silent.UnmarshalBinary(mustMarshal(t, observed)); err != nil {
				t.Fatal(err)
			}
			for call := 1; call <= 5; call++ {
				report := StepReported(observed, nil)
				Step(&silent, nil)
				if !bytes.Equal(mustMarshal(t, observed), mustMarshal(t, &silent)) || observed.Hash() != silent.Hash() ||
					!reflect.DeepEqual(observed.SavedProjectiles(), silent.SavedProjectiles()) {
					t.Fatalf("call %d: report changed world bytes, hash or projectile leaves", call)
				}
				if call <= 3 && len(report.ProjectileMarks) != 1 {
					t.Fatalf("positive call %d lost its mark", call)
				}
			}
		})
	}
}

func TestProjectileMarkReleaseCallbackAcceptsActorZero(t *testing.T) {
	for _, picture := range []int32{20, 30} {
		t.Run(fmt.Sprint(picture), func(t *testing.T) {
			target := shotPair(4)[1]
			target.ID, target.SourceBinding = 0, SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 101, RuntimeID: 91}
			target.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
			observed, silent := shotWorld(t, target), shotWorld(t, target)
			var marks []ProjectileMark
			record := CastRecord{Picture: picture, HasTarget: true, Target: 0, Segments: 2, Phases: 4,
				Mark: func(mark ProjectileMark) { marks = append(marks, mark) }}
			if _, ok := observed.ReleaseCast(record); !ok {
				t.Fatal("actor-zero homing record refused")
			}
			record.Mark = nil
			if _, ok := silent.ReleaseCast(record); !ok {
				t.Fatal("silent record refused")
			}
			want := ProjectileMark{Target: 0, Picture: picture, Binding: target.SourceBinding}
			if !reflect.DeepEqual(marks, []ProjectileMark{want}) {
				t.Fatalf("first driver call marks %+v, want %+v", marks, want)
			}
			if !bytes.Equal(mustMarshal(t, observed), mustMarshal(t, silent)) || observed.Hash() != silent.Hash() {
				t.Fatal("release callback changed world bytes or hash")
			}
			if got := StepReported(observed, nil).ProjectileMarks; !reflect.DeepEqual(got, []ProjectileMark{want}) {
				t.Fatalf("last positive call marks %+v, want %+v", got, want)
			}
			if len(marks) != 1 {
				t.Fatal("release callback was retained past its first call")
			}
		})
	}
}

func TestProjectileMarkProducerReportsActorZeroOnRelease(t *testing.T) {
	caster := spMage(1, 2, 4, 40, 100, 100, 1<<6)
	target := spEnt(0, 6, 4)
	target.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 101, RuntimeID: 91}
	target.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	w := spWorld(t, 1, []SpellRule{{ID: 6, School: 1, TargetsUnit: true, Damaging: true, DamageMin: 1, DamageMax: 1, MaxRange: 8}}, caster, target)
	w.SetFlightInputs(FlightInputs{Pictures: map[int32]FlightPicture{20: {Present: true, Phases: 4, Homes: true, LifeTicks: 2}}})
	for tick := 0; tick < 24; tick++ {
		var commands []Command
		if tick == 0 {
			commands = []Command{Cast(1, 0, 6)}
		}
		report := StepReported(w, commands)
		if len(report.Casts) == 0 {
			continue
		}
		want := []ProjectileMark{{Target: 0, Picture: 20, Binding: target.SourceBinding}}
		if !reflect.DeepEqual(report.ProjectileMarks, want) {
			t.Fatalf("cast release marks %+v, want %+v", report.ProjectileMarks, want)
		}
		return
	}
	t.Fatal("book cast never released")
}
