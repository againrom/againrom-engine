package sav

import "testing"

func TestActorFacingReadsCurrentMoverByteWithoutQuantization(t *testing.T) {
	for _, class := range []string{"Unit", "Humanoid", "Human"} {
		mover := make([]byte, 180)
		mover[0], mover[1] = 171, 64
		a := &Record{Off: 100, Class: class, Value: map[string]uint32{"U4B": 7},
			Raw: map[string][]byte{"Block12": make([]byte, 12), "U154": mover}}
		p := &Record{Class: "Player", Refs: map[string][]*Record{"Actors": {a}}}
		got, err := ownerActors([]*Record{p})
		if err != nil || len(got) != 1 || got[0].Facing != 171 {
			t.Fatalf("%s current mover facing: %+v %v", class, got, err)
		}
		a.Raw["U154"] = mover[:179]
		if _, err := ownerActors([]*Record{p}); err == nil {
			t.Fatal("truncated mover accepted")
		}
	}
}
