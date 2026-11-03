package shopstock

import (
	"slices"
	"testing"
)

func TestCRTMixedKeysAndPartition(t *testing.T) {
	keys := []int64{2, 1, 1, 3, 2, 1, 3, 2}
	order := []int{0, 1, 2, 3, 4, 5, 6, 7}
	SortBy(order, func(i int) []int64 { return []int64{keys[i]} }, true)
	if !slices.Equal(order, []int{1, 2, 5, 4, 7, 0, 6, 3}) {
		t.Fatalf("short sort %v", order)
	}
	for n := 9; n <= 128; n++ {
		items := make([]int, n)
		for i := range items {
			items[i] = (n - i) * 7 % 19
		}
		SortBy(items, func(i int) []int64 { return []int64{int64(i)} }, true)
		if !slices.IsSorted(items) {
			t.Fatalf("partition n=%d: %v", n, items)
		}
	}
}

func TestCRTEqualItemsExchangeZeroAndHalf(t *testing.T) {
	for _, n := range []int{2, 8, 9, 10, 12, 14, 17} {
		order := make([]int, n)
		for i := range order {
			order[i] = i
		}
		SortBy(order, func(int) []int64 { return []int64{5} }, true)
		want := make([]int, n)
		for i := range want {
			want[i] = i
		}
		if n <= 8 {
			want = append(want[1:], 0)
		} else {
			want[0], want[n/2] = want[n/2], want[0]
		}
		if !slices.Equal(order, want) {
			t.Fatalf("n=%d: got %v want %v", n, order, want)
		}
	}
}
