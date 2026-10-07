package game

import (
	"bytes"
	"fmt"
	"testing"

	"againrom/pkg/sim"
)

func TestResumeLegacyUnitCapacityBeforeExposure(t *testing.T) {
	for _, retained := range []bool{false, true} {
		t.Run(fmt.Sprint("retained_document=", retained), func(t *testing.T) {
			entities := []sim.Entity{
				{ID: 3, X: 1, Y: 1, Class: 75, TypeID: 75, MaxHP: 40, HP: 31},
				{ID: 71, X: 2, Y: 1, Class: 91, TypeID: 91, MaxHP: 40, HP: 0},
				{ID: 82, X: 3, Y: 1, Humanoid: true, Capacity: 0},
				{ID: 91, X: 4, Y: 1, Capacity: 271},
			}
			makeWorld := func(rows []sim.Entity) *sim.World {
				t.Helper()
				w, err := sim.NewWorld(19, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, rows)
				if err != nil {
					t.Fatal(err)
				}
				return w
			}
			raw, err := makeWorld(entities).MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			snapshot := Snapshot{World: bytes.Clone(raw)}
			if retained {
				snapshot.SavedDocument = &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Unavailable: "retained source unavailable"}
			} else {
				entities[0].Capacity, entities[1].Capacity = 300, 300
			}
			want := makeWorld(entities)
			mission := &Mission{World: makeWorld([]sim.Entity{{ID: 3, Humanoid: true, Capacity: 901}, {ID: 71, Capacity: 777}})}
			if err := resumeWorld(mission, &snapshot, nil); err != nil {
				t.Fatal("resume", err)
			}
			assertCurrentWorldEqual(t, want, mission.World, "legacy admission")
			if !bytes.Equal(snapshot.World, raw) {
				t.Fatal("admission changed source bytes")
			}
			if err := resumeWorld(mission, &snapshot, nil); err != nil {
				t.Fatal(err)
			}
			assertCurrentWorldEqual(t, want, mission.World, "repeated legacy admission")
		})
	}
}
