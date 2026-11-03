package shopstock

import (
	"slices"
	"sort"
)

// ShelfOrder builds the client keys of both games (SHOP-154, R2-ENGINE-798).
// Group is read only for kind 1. CRT selects the equal-key permutation.
type ShelfOrder[I any] struct {
	Flag        func(I) bool
	Kind, Group func(I) int
	Price       func(I) int64
	CRT         bool
}

func (p ShelfOrder[I]) Sort(items []I) {
	order := p.Indices(items)
	before := slices.Clone(items)
	for i, j := range order {
		items[i] = before[j]
	}
}

func (p ShelfOrder[I]) Indices(items []I) []int {
	order := make([]int, len(items))
	for i := range order {
		order[i] = i
	}
	SortBy(order, func(i int) []int64 {
		it := items[i]
		flag, group := int64(1), int64(0)
		if p.Flag(it) {
			flag = 0
		}
		kind := p.Kind(it)
		if kind == 1 {
			group = int64(p.Group(it))
		}
		return []int64{flag, int64(kind), group, p.Price(it)}
	}, p.CRT)
	return order
}

// SortBy reads keys once and sorts in place, stably unless crt is true.
func SortBy[I any](items []I, keys func(I) []int64, crt ...bool) {
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
	compare := func(a, b int) int { return slices.Compare(sorted[a].keys, sorted[b].keys) }
	if len(crt) > 0 && crt[0] {
		crtSort(0, len(sorted)-1, compare, func(a, b int) { sorted[a], sorted[b] = sorted[b], sorted[a] })
	} else {
		sort.SliceStable(sorted, func(a, b int) bool { return compare(a, b) < 0 })
	}
	for i := range sorted {
		items[i] = sorted[i].item
	}
}

// crtSort follows the short maximum-selection and middle-pivot partition
// of R2-ENGINE-799, including their equal-key permutations.
func crtSort(lo, hi int, compare func(int, int) int, swap func(int, int)) {
	for lo < hi {
		if hi-lo < 8 {
			for hi > lo {
				max := lo
				for i := lo + 1; i <= hi; i++ {
					if compare(i, max) > 0 {
						max = i
					}
				}
				swap(max, hi)
				hi--
			}
			return
		}
		swap(lo, lo+(hi-lo+1)/2)
		left, right := lo, hi+1
		for {
			left++
			for left <= hi && compare(left, lo) <= 0 {
				left++
			}
			right--
			for right > lo && compare(right, lo) >= 0 {
				right--
			}
			if right < left {
				break
			}
			swap(left, right)
		}
		swap(lo, right)
		if right-lo < hi-left {
			crtSort(lo, right-1, compare, swap)
			lo = left
		} else {
			crtSort(left, hi, compare, swap)
			hi = right - 1
		}
	}
}
