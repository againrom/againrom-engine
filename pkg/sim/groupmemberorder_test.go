package sim

import "testing"

// A pass takes members in group list order (AI-905, R2-ENGINE-1984).
func TestAGroupPassWalksItsMembersHeadToTailThroughTheGroupList(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 5, 5, 5), engFighter(2, 5, 6, 5), engFighter(3, 5, 7, 5))
	g := regGroup(t, w, 5, 0)
	if len(g.Members) != 3 {
		t.Fatalf("fixture: the Group holds %d members, want 3", len(g.Members))
	}
	g.Members[0], g.Members[2] = g.Members[2], g.Members[0]

	var seen []EntityID
	w.walkSavedMembers(g, func(i int) { seen = append(seen, w.entities[i].ID) })

	if len(seen) != 3 || seen[0] != 3 || seen[1] != 2 || seen[2] != 1 {
		t.Errorf("the pass visited members %v, want [3 2 1] in list order", seen)
	}
}
