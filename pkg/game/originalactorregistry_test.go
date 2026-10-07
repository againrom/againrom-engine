package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func registrySource1111(index uint16, off int, mapID uint16) sav.ActorRecord {
	return sav.ActorRecord{Actor: sav.Actor{Off: off, Class: "Unit", RuntimeID: uint32(index), HP: 7, MapUnitID: mapID},
		ArchiveIndex: index, Identity: uint32(0xa000) + uint32(index)}
}

func TestOriginalActorRegistryKeepsExistingIDsAndAppendsSourceOnly(t *testing.T) {
	m := &alm.Map{Units: []alm.Unit{{UnitID: 11}, {UnitID: 22}, {UnitID: 33}}}
	original := append([]alm.Unit(nil), m.Units...)
	graph := sav.SavedActorGraph{Actors: []sav.ActorRecord{
		registrySource1111(9, 90, 500), registrySource1111(4, 40, 22),
		registrySource1111(8, 80, 500), registrySource1111(6, 60, 33), registrySource1111(7, 70, 0),
	}}
	party := []mapload.PartyMember{{ID: "hero", Saved: &mapload.Saved{MapUnitID: 22}}}
	r, units, err := planOriginalActors(m, graph, party, []int{40})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.Units, original) || len(units) != 2 || units[0].UnitID != 11 || units[1].UnitID != 33 {
		t.Fatalf("detached withdrawal: original=%v surviving=%v", m.Units, units)
	}
	for _, want := range []struct {
		off          int
		id           sim.EntityID
		party, added bool
	}{
		{40, 2, true, false}, {60, 1, false, false}, {70, 3, false, true},
		{80, 4, false, true}, {90, 5, false, true},
	} {
		got, ok := r.actor(want.off)
		if !ok || got.ID != want.id || got.Party != want.party || got.New != want.added {
			t.Fatalf("source%d: %+v present=%t", want.off, got, ok)
		}
	}
	if r.actors[3].Source.MapUnitID != 500 || r.actors[4].Source.MapUnitID != 500 || r.actors[2].Source.MapUnitID != 0 {
		t.Fatalf("map IDs were fabricated or collapsed: %+v", r.actors)
	}
}

func TestOriginalActorRegistryRefusesTwoALMUnitsWithOneIDAtomically(t *testing.T) {
	m := &alm.Map{Units: []alm.Unit{{UnitID: 22}, {UnitID: 22}}}
	graph := sav.SavedActorGraph{Actors: []sav.ActorRecord{registrySource1111(4, 40, 22)}}
	before := append([]alm.Unit(nil), m.Units...)
	registry, units, err := planOriginalActors(m, graph, nil, nil)
	if err == nil || registry != nil || units != nil || !reflect.DeepEqual(before, m.Units) {
		t.Fatalf("ambiguous ALM units published: %+v %v %v", registry, units, err)
	}
}

// Several actors may claim one ALM unit; exactly one binds to it and the rest
// stay distinct source-only actors. The claimant whose owner and type match
// the unit wins over a lower archive index.
func TestOriginalActorRegistryBindsOneOfSeveralClaimants(t *testing.T) {
	claim := func(index uint16, off int, owner, typeID uint16) sav.ActorRecord {
		a := registrySource1111(index, off, 22)
		a.OwnerSlot, a.TypeID = owner, typeID
		return a
	}
	for _, tc := range []struct {
		name   string
		actors []sav.ActorRecord
		bound  int
	}{
		{"owner and type", []sav.ActorRecord{claim(4, 40, 1, 26), claim(6, 60, 1, 26), claim(8, 80, 3, 80)}, 80},
		{"owner only", []sav.ActorRecord{claim(4, 40, 1, 26), claim(6, 60, 3, 7), claim(8, 80, 3, 9)}, 60},
		{"lowest archive index", []sav.ActorRecord{claim(6, 60, 1, 26), claim(4, 40, 1, 26)}, 40},
	} {
		m := &alm.Map{Units: []alm.Unit{{UnitID: 22, ClassID: 80, Owner: 3}}}
		r, units, err := planOriginalActors(m, sav.SavedActorGraph{Actors: tc.actors}, nil, nil)
		if err != nil || len(units) != 1 || len(r.actors) != len(tc.actors) {
			t.Fatalf("%s: %v units%d actors%d", tc.name, err, len(units), len(r.actors))
		}
		for _, a := range tc.actors {
			got, _ := r.actor(a.Off)
			if want := a.Off == tc.bound; got.New == want {
				t.Fatalf("%s: source%d New=%t, bound source is %d", tc.name, a.Off, got.New, tc.bound)
			}
		}
	}
}

func TestOriginalActorRegistryRejectsLateDuplicateSourceAndPartyAlias(t *testing.T) {
	m := &alm.Map{Units: []alm.Unit{{UnitID: 10}}}
	for _, offsets := range [][]int{nil, {40, 40}} {
		graph := sav.SavedActorGraph{Actors: []sav.ActorRecord{registrySource1111(4, 40, 0), registrySource1111(6, 60, 0)}}
		party := make([]mapload.PartyMember, len(offsets))
		if offsets == nil {
			graph.Actors[1].Identity = graph.Actors[0].Identity
		}
		registry, units, err := planOriginalActors(m, graph, party, offsets)
		if err == nil || registry != nil || units != nil || len(m.Units) != 1 {
			t.Fatalf("invalid source published: %+v %v %v", registry, units, err)
		}
	}
}

func TestOriginalActorRegistryReservesRetiredNativeIdentities(t *testing.T) {
	m := &alm.Map{Units: []alm.Unit{{UnitID: 10}}}
	graph := sav.SavedActorGraph{Actors: []sav.ActorRecord{registrySource1111(4, 40, 0)}}
	r, _, err := planOriginalActors(m, graph, nil, nil, 27, 11)
	if err != nil || r.actors[0].ID != 28 {
		t.Fatalf("reserved IDs: %+v %v", r, err)
	}
	if r, units, err := planOriginalActors(m, graph, nil, nil, ^sim.EntityID(0)); err == nil || r != nil || units != nil {
		t.Fatalf("namespace overflow admitted: %+v %v %v", r, units, err)
	}
}

// A current terminal root may carry Stage 1; it is never planned as dying.
func TestOriginalActorRegistrySkipsTerminalRootWithDyingStage(t *testing.T) {
	m := &alm.Map{Units: []alm.Unit{{UnitID: 11}}}
	for _, terminal := range []bool{false, true} {
		root := sav.ActorRecord{Actor: sav.Actor{Off: 40, Class: "Unit", RuntimeID: 4, Stage: 1, HP: -1000, MapUnitID: 11},
			ArchiveIndex: 4, Identity: 0xa004, TerminalActor: terminal}
		r, units, err := planOriginalActors(m, sav.SavedActorGraph{Actors: []sav.ActorRecord{root}}, nil, nil)
		if err != nil || len(units) != 1 {
			t.Fatalf("terminal=%t: %v %v", terminal, units, err)
		}
		if _, planned := r.actor(40); planned == terminal {
			t.Fatalf("terminal=%t: Stage 1 root planned=%t", terminal, planned)
		}
	}
}
