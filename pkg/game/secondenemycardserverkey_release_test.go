package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// secondKeyedRows lists the live non-own units of the mission, humanoid or
// not, by the Units row their placement resolves to.
func secondKeyedRows(f *FrontEnd, live *mapWorld) (map[int][]sim.EntityID, map[int]sim.Entity) {
	ids, first := map[int][]sim.EntityID{}, map[int]sim.Entity{}
	for _, u := range live.mission.state.Map.Units {
		r := mapload.Resolve(u, f.Table)
		if !r.Found() || u.UnitID == 0 {
			continue
		}
		for _, e := range live.world.Entities() {
			if e.MapUnitID == u.UnitID && e.Owner != sim.SelfSlot {
				ids[r.Index] = append(ids[r.Index], e.ID)
				if _, ok := first[r.Index]; !ok {
					first[r.Index] = e
				}
			}
		}
	}
	return ids, first
}

// The second game's card is graded by the unit's server id alone: every kind
// whose row states a server id in 1..0x9ff reads level 7 at a count of 14 and a
// row with none reads level 0, and a humanoid creature of a high row counts a
// kill (R2-ENGINE-772, R2-ENGINE-1710, R2-ENGINE-1712).
func TestReleaseSecondGameEnemyCardFollowsServerIdForEveryKind(t *testing.T) {
	f, _ := secondMissionSettledAt(t, 20)
	live := f.live
	w := live.world
	ids, first := secondKeyedRows(f, live)
	rows := make([]int, 0, len(first))
	for row := range first {
		rows = append(rows, row)
	}
	slices.Sort(rows)
	weakest, outside, looks := -1, 0, map[int]int{}
	for _, row := range rows {
		e := first[row]
		look := row
		if typeID, face, ok := data.UnitKeyPair(f.Table.Units.EntryParams(row)); ok {
			if r := data.FindUnit(f.Table.Units, typeID, face); r > 0 {
				look = r
			}
		}
		looks[row] = look
		sid, sok := data.UnitServerID(f.Table.Units.EntryParams(look))
		keyed := sok && uint16(sid) >= 1 && uint16(sid) <= 0x9ff
		secondCardSetCount(w, look, 14)
		want := 0
		if keyed {
			want = 7
		}
		if l := w.KnowledgeLevel(mustEntity(t, live, e.ID)); l != want {
			t.Fatalf("row %d type %d serverID %d: level %d, want %d", row, e.TypeID, sid, l, want)
		}
		if !keyed {
			outside++
		}
		if e.Humanoid && keyed && row >= 64 && (weakest < 0 || e.MaxHP < first[weakest].MaxHP) {
			weakest = row
		}
	}
	t.Logf("rows read %d, without a server id %d, humanoid row %d", len(rows), outside, weakest)
	if weakest < 0 {
		t.Fatal("no hostile humanoid creature of a high row in the mission")
	}
	look := looks[weakest]
	secondCardSetCount(w, look, 0)
	enemyCardKill(t, live, ids[weakest][0])
	if got := enemyCardCount(w, look); got != 1 {
		t.Fatalf("humanoid row %d type %d: one kill gave count %d, want 1", weakest, first[weakest].TypeID, got)
	}
}
