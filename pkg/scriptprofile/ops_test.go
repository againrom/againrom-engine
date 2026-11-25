package scriptprofile

import (
	"strconv"
	"testing"
)

// TestOpRowsCarryTheOldWordKeysAndRegisterSlots lists every row of each game
// and compares its editor word key and register flag with the ids the old
// word tables and the register test named.
func TestOpRowsCarryTheOldWordKeysAndRegisterSlots(t *testing.T) {
	words := map[Ops]map[bool][]uint32{
		OpsFirst: {
			true:  {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 21, 22, 65538, 16641, 16642},
			false: {2, 3, 4, 5, 6, 8, 9, 10, 11, 12, 13, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 32, 33, 34, 65538, 16640, 16643, 16644, 16646},
		},
		OpsSecond: {
			true:  {1, 2, 4, 5, 6, 7, 8, 9, 10, 14, 15, 16, 17, 18, 19, 21, 23, 24, 25, 26, 27, 65538},
			false: {2, 3, 4, 5, 6, 8, 10, 12, 13, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 28, 29, 30, 31, 32, 33, 34, 35, 36, 38, 39, 65538},
		},
	}
	prefix := map[Ops]map[bool]string{
		OpsFirst:  {true: "editor.check.op", false: "editor.action.op"},
		OpsSecond: {true: "editor.second.check.", false: "editor.second.action."},
	}
	for ops, families := range words {
		for condition, ids := range families {
			want := map[uint32]string{}
			for _, id := range ids {
				want[id] = prefix[ops][condition] + strconv.Itoa(int(id))
			}
			for _, row := range OpRows(ops, condition) {
				if row.Word != want[row.ID] {
					t.Errorf("ops %d condition %v id %d: word %q, want %q", ops, condition, row.ID, row.Word, want[row.ID])
				}
				delete(want, row.ID)
				register := condition && row.ID == 19 || !condition && (row.ID == 3 || row.ID == 8)
				if row.Register != register {
					t.Errorf("ops %d condition %v id %d: register %v, want %v", ops, condition, row.ID, row.Register, register)
				}
			}
			if len(want) > 0 {
				t.Errorf("ops %d condition %v: no row for words %v", ops, condition, want)
			}
		}
	}
	for _, c := range []struct {
		condition bool
		id        uint32
		want      bool
	}{{true, 19, true}, {true, 3, false}, {false, 3, true}, {false, 8, true}, {false, 19, false}} {
		if got := AddressesRegister(c.condition, c.id); got != c.want {
			t.Errorf("AddressesRegister(%v, %d) = %v", c.condition, c.id, got)
		}
	}
}

func TestOpRowsHoldConditionShapes(t *testing.T) {
	wantFirst := map[uint32]Shape{
		2:  ShapeBox,
		3:  ShapeSquare,
		6:  ShapeUnitSquare,
		7:  ShapeDistance,
		15: ShapeDistance,
		16: ShapeDistance,
	}
	for _, ops := range []Ops{OpsFirst, OpsSecond} {
		for _, condition := range []bool{true, false} {
			for _, row := range OpRows(ops, condition) {
				var want Shape
				if ops == OpsFirst && condition {
					want = wantFirst[row.ID]
				}
				if row.Shape != want {
					t.Errorf("ops %d condition %v id %d: shape %d, want %d", ops, condition, row.ID, row.Shape, want)
				}
			}
		}
	}
}

func TestSecondGameActionsUnchangedByFirstGameTeleport(t *testing.T) {
	if _, ok := OpRow(OpsSecond, false, TeleportCellOp); ok {
		t.Fatal("the second game's action table carries the first game's teleport action")
	}
	want := []uint32{2, 3, 4, 5, 6, 8, 10, 12, 13, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 28, 29, 30, 31, 32, 33, 34, 35, 36, 38, 39, 65538}
	rows := OpRows(OpsSecond, false)
	if len(rows) != len(want) {
		t.Fatalf("second actions count %d, want %d", len(rows), len(want))
	}
	for i, id := range want {
		if rows[i].ID != id {
			t.Errorf("second action %d: id %d, want %d", i, rows[i].ID, id)
		}
	}
}
