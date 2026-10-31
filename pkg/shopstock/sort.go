package shopstock

import (
	"slices"
	"sort"
)

// SortBy is the one shelf sort of both games: it orders items in place by
// their keys, compared in turn and each ascending. Each item's keys are read
// once; items equal on every key keep their order.
func SortBy[I any](items []I, keys func(I) []int64) {
	if len(items) < 2 {
		return
	}
	type entry struct {
		item I
		keys []int64
	}
	sorted := make([]entry, len(items))
	for i, it := range items {
		sorted[i] = entry{it, keys(it)}
	}
	sort.SliceStable(sorted, func(a, b int) bool { return slices.Compare(sorted[a].keys, sorted[b].keys) < 0 })
	for i := range sorted {
		items[i] = sorted[i].item
	}
}
