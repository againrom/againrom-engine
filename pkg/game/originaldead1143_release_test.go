package game

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseOriginalDead1143UnboundHirelingAndEarlyDecayStage(t *testing.T) {
	f := releaseFront(t)
	type want struct {
		identity, mapUnit, runtime uint32
		stage                      uint8
		hp                         int16
		cell                       uint16
		bound                      bool // has a live entity binding (MapUnitID != 0)
	}
	cases := []struct {
		rel  string
		hash string
		w    want
	}{
		{"2027-09-07/game0011.sav", "be50c93ad391eda8e052e520d1ceb0f660b27bc697ebd32b97b161386eed709d",
			want{identity: 0x303ac50, mapUnit: 51, runtime: 90, stage: 2, hp: -14, cell: 0x3332, bound: true}},
		{"2027-09-07/game0032.sav", "7375f0c08c8361b2fa32d20564802acba688d5e5a445ab5ca635c609632499ed",
			want{identity: 0x123fbb48, mapUnit: 0, runtime: 39, stage: 4, hp: -58, cell: 0x3f34, bound: false}},
		{"2027-09-07/game0033.sav", "b6ffd904f44717c76e22f12d8801a21b3dfca3dd8563b7e02aaecbf93a4c740a",
			want{identity: 0x123fbb48, mapUnit: 0, runtime: 39, stage: 4, hp: -86, cell: 0x3f34, bound: false}},
	}
	for _, c := range cases {
		t.Run(c.rel, func(t *testing.T) {
			_, payload := groundCorpusFile(t, c.rel, c.hash)
			ms, report, err := ResumeOriginalSave(f.Archives.Containers, payload, f.Table, f.Difficulty, nil, f.Bodies)
			if err != nil {
				t.Fatalf("ResumeOriginalSave refused: %v", err)
			}
			if report.Mission == 0 {
				t.Fatal("resume did not carry a mission")
			}
			var found bool
			for _, r := range ms.World.OriginalDeadActors() {
				if r.Source.Identity != c.w.identity {
					continue
				}
				found = true
				if r.Source.MapUnitID != uint16(c.w.mapUnit) || r.Source.State.Stage != c.w.stage ||
					r.Source.State.HP != c.w.hp || r.Source.State.RuntimeID != c.w.runtime || r.Source.State.Cell != c.w.cell {
					t.Fatalf("source tuple differs from independent transcription: %+v", r.Source)
				}
				if r.Current != r.Source.State {
					t.Fatalf("current tuple diverged from an untouched import: %+v vs %+v", r.Current, r.Source.State)
				}
				var haveEntity bool
				for _, e := range ms.World.Entities() {
					if e.ID == r.ID {
						haveEntity = true
					}
				}
				if haveEntity != c.w.bound {
					t.Fatalf("entity binding: got %t want %t", haveEntity, c.w.bound)
				}
			}
			if !found {
				t.Fatalf("dead identity %#x not carried into OriginalDeadActors", c.w.identity)
			}
			// Determinism: an independent second resume of the identical
			// bytes must hash identically.
			ms2, _, err := ResumeOriginalSave(f.Archives.Containers, payload, f.Table, f.Difficulty, nil, f.Bodies)
			if err != nil {
				t.Fatalf("second resume refused: %v", err)
			}
			if ms.World.Hash() != ms2.World.Hash() {
				t.Fatal("two independent resumes of the same save disagree")
			}
			records := ms.World.OriginalDeadActors()
			data, err := ms.World.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var w2 sim.World
			if err := w2.UnmarshalBinary(data); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(records, w2.OriginalDeadActors()) {
				t.Fatal("native round trip changed dead-actor state")
			}
			if w2.Hash() != ms.World.Hash() {
				t.Fatal("native round trip changed the hash")
			}
		})
	}
}
