package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestCurrentCellResidueFollowsSackAndCellLifecycle(t *testing.T) {
	for _, mode := range []string{"sack", "actor", "retained", "absent", "refused"} {
		t.Run(mode, func(t *testing.T) {
			payload := sackPayload1115(0)
			w := sackCellWorld1115(t, &payload)
			rows := []SavedCellRecord{{Cell: sackCell1115 - 1, Residue0: 17}, {Cell: sackCell1115, Residue0: 23, Residue1: [2]byte{7, 8}}, {Cell: sackCell1115 + 1, Residue0: 29}}
			if mode == "absent" {
				rows = append(rows[:1], rows[2:]...)
			}
			w.SetSavedCellRecords(rows)
			if mode == "refused" {
				w.savedCellPlanes.Dynamic[sackCell1115] |= 1
				before := w.Hash()
				if ok, issue := w.registerSavedSackCell(sackCell1115, sackKey1115); ok || issue != "" || w.Hash() != before {
					t.Fatal("refused Sack registration changed current residue", ok, issue)
				}
				return
			}
			if ok, issue := w.registerSavedSackCell(sackCell1115, sackKey1115); !ok || issue != "" {
				t.Fatal(ok, issue)
			}
			if mode != "absent" && w.savedCellRecords[1].Sack != sackKey1115 {
				t.Fatal("Sack registration missed current residue")
			}
			if mode == "retained" {
				c := w.motionCell(sackCell1115)
				c.Ground = SavedActorSlot{Key: 77}
				binary.LittleEndian.PutUint32(c.Payload[4:], 77)
			}
			if mode == "actor" {
				binary.LittleEndian.PutUint32(w.motionCell(sackCell1115).Payload[16:], 0)
				if !w.deleteEmptySavedCell(sackCell1115) {
					t.Fatal("empty actor cell not deleted")
				}
			} else if ok, issue := w.removeSavedSackCell(sackCell1115); !ok || issue != "" {
				t.Fatal(ok, issue)
			}
			want := []SavedCellRecord{rows[0], rows[len(rows)-1]}
			if mode == "retained" {
				want = rows
				want[1].Sack = 0
				if w.motionCell(sackCell1115) == nil {
					t.Fatal("occupied current cell was deleted")
				}
			} else if w.motionCell(sackCell1115) != nil {
				t.Fatal("empty current cell survived")
			}
			if !reflect.DeepEqual(w.savedCellRecords, want) {
				t.Fatal("cell lifecycle changed unrelated or absent residue", w.savedCellRecords, want)
			}
		})
	}
}
