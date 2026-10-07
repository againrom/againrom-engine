package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentAnimationKeepsMeleeClocksAcrossColdLoads(t *testing.T) {
	f := currentArchiveFixture(t)
	entities := f.live.world.Entities()
	a, b := entities[0].ID, entities[1].ID
	f.live.swing = map[sim.EntityID]int{a: 7, b: 0}
	f.live.phase = map[sim.EntityID]sim.AttackPhase{a: sim.AttackCharging, b: sim.AttackReady}
	for cycle := 0; cycle < 2; cycle++ {
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		cold := openCurrentArchive(t, raw)
		if !reflect.DeepEqual(f.live.swing, cold.live.swing) || !reflect.DeepEqual(f.live.phase, cold.live.phase) || f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatal("current melee clocks or World changed", cycle, cold.live.swing, cold.live.phase)
		}
		for tick := 0; tick < 20; tick++ {
			f.live.tick()
			cold.live.tick()
			if !reflect.DeepEqual(f.live.swing, cold.live.swing) || !reflect.DeepEqual(f.live.phase, cold.live.phase) || f.live.world.Hash() != cold.live.world.Hash() {
				t.Fatal("next melee animation or World changed", cycle, tick)
			}
		}
		f = cold
	}
}

func TestCurrentAnimationRejectsMalformedClockWithoutCommit(t *testing.T) {
	f := currentArchiveFixture(t)
	id := f.live.world.Entities()[0].ID
	f.live.swing[id] = 7
	f.live.phase[id] = sim.AttackCharging
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"repeated", "negative", "phase", "empty", "cast overlap", "unbound"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || len(a.Animation) == 0 {
				t.Fatal(err)
			}
			row := &a.Animation[0]
			switch change {
			case "repeated":
				a.Animation = append(a.Animation, *row)
			case "negative":
				n := -2
				row.Swing = &n
			case "phase":
				p := sim.AttackPhase(255)
				row.Phase = &p
			case "unbound":
				row.Entity = 4000000
			case "empty":
				row.Swing, row.Phase = nil, nil
			case "cast overlap":
				a.Runs = append(a.Runs, SnapshotCastRun{Entity: row.Entity, Left: 1, Span: 1})
			}
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			candidate, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			before, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := f.RestoreOriginal(candidate); err == nil {
				t.Fatal("malformed animation accepted")
			}
			after, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("failed LOAD changed current state")
			}
		})
	}
}

func TestCurrentAnimationCastOwnsItsClockOnce(t *testing.T) {
	residue := SnapshotResidue{Swing: map[uint32]int{3: 6, 4: 8}, Phase: map[uint32]uint8{3: 1}, CastRuns: []SnapshotCastRun{{Entity: 3, Left: 2, Span: 3, Swing: 6, Phase: 1}}}
	rows := captureCurrentAnimation(residue)
	if len(rows) != 1 || rows[0].Entity != 4 || *rows[0].Swing != 8 || rows[0].Phase != nil {
		t.Fatal("cast acquired a second clock", rows)
	}
}
