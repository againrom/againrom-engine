package shopstock

import (
	"reflect"
	"sort"
	"testing"
)

type shelfItem struct{ key, tag, n int }

func shelfStack(order bool) Stack[shelfItem, int] {
	s := Stack[shelfItem, int]{
		Key:   func(it shelfItem) int { return it.key },
		Same:  func(a, b shelfItem) bool { return a.tag == b.tag },
		Takes: func(entry, it shelfItem) bool { return it.tag >= 0 || entry.n == 0 },
		Add: func(entry, it shelfItem) shelfItem {
			entry.n += it.n
			return entry
		},
	}
	if order {
		s.Order = func(list []shelfItem) {
			sort.SliceStable(list, func(i, j int) bool { return list[i].key < list[j].key })
		}
	}
	return s
}

// TestInsertJoinsTheFirstEntryItMerges: key, Same, Takes, placeholder; Order once.
func TestInsertJoinsTheFirstEntryItMerges(t *testing.T) {
	s := shelfStack(false)
	list := []shelfItem{{3, 1, 2}, {3, 2, 1}, {3, -1, 0}, {3, -1, 4}}
	got := Insert(append([]shelfItem(nil), list...), s, shelfItem{3, 2, 5}, shelfItem{3, -1, 1}, shelfItem{3, -1, 1}, shelfItem{1, 1, 1})
	want := []shelfItem{{3, 1, 2}, {3, 2, 6}, {3, -1, 1}, {3, -1, 4}, {3, -1, 1}, {1, 1, 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unordered insert %v, want %v", got, want)
	}
	if i := Find(list, s, shelfItem{3, -1, 9}); i != 2 {
		t.Fatalf("Find placeholder at %d, want 2", i)
	}
	ordered := shelfStack(true)
	sorted := 0
	inner := ordered.Order
	ordered.Order = func(l []shelfItem) { sorted++; inner(l) }
	got = Insert([]shelfItem{{2, 0, 1}}, ordered, shelfItem{2, 0, 1})
	if sorted != 0 || !reflect.DeepEqual(got, []shelfItem{{2, 0, 2}}) {
		t.Fatalf("a merge sorted %d times: %v", sorted, got)
	}
	got = Insert(got, ordered, shelfItem{1, 0, 1}, shelfItem{0, 0, 1}, shelfItem{2, 0, 1})
	if sorted != 1 || !reflect.DeepEqual(got, []shelfItem{{0, 0, 1}, {1, 0, 1}, {2, 0, 3}}) {
		t.Fatalf("an open sorted %d times: %v", sorted, got)
	}
}
