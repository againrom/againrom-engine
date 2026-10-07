package sim

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
)

func TestOriginalActionEscortAdmissionKeepsExactTargetAndPendingOrder(t *testing.T) {
	for _, state := range []uint8{actorStateDefend, actorStateFollow} {
		for _, mode := range []string{"bound", "missing-key", "stage", "absent-order"} {
			t.Run(fmt.Sprintf("%d/%s", state, mode), func(t *testing.T) {
				w := savedIncomingEscortWorld(t)
				o := w.savedOrder(10)
				o.State = uint32(state)
				switch mode {
				case "missing-key":
					o.EscortTarget, o.EscortBound = 0, false
					binary.LittleEndian.PutUint32(o.Raw[0x10:], 0xdeadbeef)
				case "stage":
					o.RepairStage, o.EscortTarget, o.EscortBound = 2, 0, false
				case "absent-order":
					w.savedGroups.Orders = w.savedGroups.Orders[1:]
				}
				_, orders, _ := w.SavedGroups()
				before := w.entities[0]
				if err := w.ImportOriginalActorActions([]OriginalActorAction{{Entity: 10, ActorState: state}}); err != nil {
					t.Fatal(err)
				}
				e := w.entities[0]
				if mode == "bound" {
					if e.ActorState != state || !e.HasEscortTarget || e.EscortTarget != 30 || e.EscortRange != 1 || e.HasTarget || e.X != before.X || e.Y != before.Y {
						t.Fatal("escort admission changed target or dispatched movement", e)
					}
				} else if !reflect.DeepEqual(before, e) {
					t.Fatal("unresolved escort changed native state", before, e)
				}
				_, afterOrders, _ := w.SavedGroups()
				if !reflect.DeepEqual(orders, afterOrders) {
					t.Fatal("admission changed ordinary source order")
				}
				form, err := w.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				var fresh World
				if err := fresh.UnmarshalBinary(form); err != nil {
					t.Fatal(err)
				}
				if fresh.Hash() != w.Hash() {
					t.Fatal("escort admission is not a complete native state")
				}
				for range 20 {
					Step(w, nil)
					Step(&fresh, nil)
					if fresh.Hash() != w.Hash() {
						t.Fatal("escort continuation changed")
					}
				}
				if mode == "bound" && w.entities[0].X == before.X && w.entities[0].Y == before.Y {
					t.Fatal("restored escort did not take its actual move step")
				}
			})
		}
	}
}
