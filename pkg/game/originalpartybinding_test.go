package game

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func TestOriginalCurrentPartyActorKeepsSourceAndNativeIdentity(t *testing.T) {
	actor := sav.ActorRecord{
		Actor:        sav.Actor{Off: 17, Class: "Human", MapUnitID: 91, RuntimeID: 77, HP: 11, Stage: 1},
		ArchiveIndex: 4, Identity: 501, CurrentEntity: true,
	}
	for _, tc := range []struct {
		name     string
		actor    sav.ActorRecord
		admitted bool
	}{
		{name: "current source actor", actor: actor, admitted: true},
		{name: "absent source actor", actor: func() sav.ActorRecord { a := actor; a.CurrentEntity = false; return a }()},
		{name: "terminal source actor", actor: func() sav.ActorRecord { a := actor; a.TerminalActor = true; return a }()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &alm.Map{Units: []alm.Unit{{UnitID: 91}, {UnitID: 92}}}
			registry, units, err := planOriginalActors(m, sav.SavedActorGraph{Actors: []sav.ActorRecord{tc.actor}, CurrentPopulation: true},
				[]mapload.PartyMember{{Name: "source member"}}, []int{17})
			if !tc.admitted {
				if err == nil {
					t.Fatal("absent party actor acquired a native identity")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			binding, ok := registry.actor(17)
			if !ok || !binding.Party || binding.New || binding.ID != 1 || binding.Source.ArchiveIndex != 4 || binding.Source.Identity != 501 {
				t.Fatalf("source party identity changed: %+v, found %t", binding, ok)
			}
			if len(units) != 1 || units[0].UnitID != 92 {
				t.Fatalf("source party actor kept a duplicate ALM placement: %+v", units)
			}
		})
	}
}
