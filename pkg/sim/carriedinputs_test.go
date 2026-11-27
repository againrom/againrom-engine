package sim

import (
	"reflect"
	"testing"
)

func TestFreshReceiverRaisesControlSpiritOnlyWithCarriedInputs(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<25)
	caster.Owner = 7
	corpse := spEnt(2, 2, 2)
	src := hlGhostWorld(t, 33, []SpellRule{csRule()}, hlGhostTemplate(), caster, corpse)
	src.entities[1].HP, src.entities[1].Decay = -10, DecayBones
	raw, err := src.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	for _, copied := range []bool{false, true} {
		var staged World
		if copied {
			staged.CopyCarriedInputs(src)
		}
		if err := staged.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		if got := staged.Ghost().Raisable(); got != copied {
			t.Fatalf("copied=%v: ghost raisable = %v", copied, got)
		}
		spRunCast(&staged, Cast(1, 2, 25))
		raised := false
		for _, e := range staged.entities {
			raised = raised || e.Class == hlGhostTemplate().Class && e.Owner == 7
		}
		if raised != copied {
			t.Errorf("copied=%v: raised Ghost present = %v", copied, raised)
		}
	}
}

func TestImportLivingActorsCarriesNonSerializedInputs(t *testing.T) {
	ghost := hlGhostTemplate()
	rule := blastRule(1)
	w, err := NewStructuredWorld(7, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
		[]Entity{
			{ID: 1, MapUnitID: 91, X: 5, Y: 5, HP: 20, MaxHP: 20, TokenSize: 1},
			{ID: 2, MapUnitID: 92, X: 6, Y: 5, HP: 20, MaxHP: 20, TokenSize: 1},
		}, nil, Relations{}, nil, nil, []SpellRule{rule}, ghost, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.DeclareItemWeights([]ItemWeight{{Code: packMergeBow.Code, Weight: 3}}); err != nil {
		t.Fatal(err)
	}
	mergeRows := []ItemWeight{{Code: packMergeBow.Code, Weight: 3, Constructor: packMergeBuilt()}}
	if err := w.BindMergeOperands(mergeRows); err != nil {
		t.Fatal(err)
	}
	in := FlightInputs{
		Pictures: map[int32]FlightPicture{
			fireBallBurstPicture(): {Present: true, Phases: 11},
			20:                     {Present: true, Phases: 4, Homes: true, LifeTicks: 2},
		},
		Classes: map[int32]FlightClass{
			7: {Projectile: 20, ShootDelay: 2, TileSize: 1},
		},
	}
	w.SetFlightInputs(in)
	w.SetLoadedRecord(LoadedDocument)

	for _, name := range []string{"original", "current"} {
		t.Run(name, func(t *testing.T) {
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			cand := *w
			if err := cand.UnmarshalBinary(raw); err != nil {
				t.Fatal(err)
			}
			e := cand.entities[0]
			e.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 0x01000001}
			e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
			if name == "original" {
				if err := cand.ImportOriginalLivingActors([]OriginalLivingActor{{Entity: e}}); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := cand.ImportCurrentLivingActors([]OriginalLivingActor{{Entity: e}}, []EntityID{e.ID}); err != nil {
					t.Fatal(err)
				}
			}
			if got := cand.Ghost(); got != ghost {
				t.Errorf("ghost = %+v, want %+v", got, ghost)
			}
			if !reflect.DeepEqual(cand.mergeOperands, mergeRows) {
				t.Errorf("mergeOperands = %+v, want %+v", cand.mergeOperands, mergeRows)
			}
			if cand.burst.phases != 11 {
				t.Errorf("burst.phases = %d, want 11", cand.burst.phases)
			}
			if !cand.HasFlightInputs() {
				t.Errorf("flightInputs dropped (HasFlightInputs = false)")
			} else if !reflect.DeepEqual(cand.flightInputs.Pictures, in.Pictures) || !reflect.DeepEqual(cand.flightInputs.Classes, in.Classes) {
				t.Errorf("flightInputs = %+v, want %+v", cand.flightInputs, in)
			}
			if got := cand.LoadedRecord(); got != LoadedDocument {
				t.Errorf("loadedRecord = %v, want %v", got, LoadedDocument)
			}
		})
	}
}
