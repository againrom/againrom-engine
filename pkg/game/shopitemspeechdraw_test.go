package game

import (
	"math/rand"
	"testing"
)

// shopDrawSeam answers the gate draw from values in turn.
func shopDrawSeam(values []int, calls *int) func(int) int {
	return func(n int) int {
		if n != 100 {
			return 0
		}
		v := values[*calls%len(values)]
		*calls++
		return v
	}
}

// A draw below 30 requests a recording; 30 and above requests nothing and
// leaves a playing one (SHOP-109).
func TestShopItemResponseContinuesBelowThirty(t *testing.T) {
	f, app, s, r := shopCommentFixture(t)
	item := f.Shop.shelves[ShelfArmour][0]
	values := []int{0, 29, 30, 31, 99, 15}
	want := []int{1, 1, 0, 0, 0, 1}
	calls := 0
	f.ShopRandom = shopDrawSeam(values, &calls)
	for i, v := range values {
		base := len(r.samples)
		s.shopItemResponse(item)
		if got := shopSpeechDrain(t, app, r, base); got != want[i] {
			t.Fatalf("draw %d: %d recording(s), want %d", v, got, want[i])
		}
	}
	if calls != len(values) {
		t.Fatalf("%d gate draws for %d placements, want one each", calls, len(values))
	}

	calls = 0
	f.ShopRandom = shopDrawSeam([]int{5, 60}, &calls)
	base := len(r.samples)
	s.shopItemResponse(item)
	if len(r.samples) != base+1 || !r.voices[base].Playing() {
		t.Fatalf("draw 5 started %d recording(s)", len(r.samples)-base)
	}
	s.shopItemResponse(item)
	if len(r.samples) != base+1 || !r.voices[base].Playing() {
		t.Fatal("a failed draw stopped or replaced the recording that was playing")
	}
}

// Requests over 400 seeded placements equal the draws below 30.
func TestShopItemResponseCountOverFixedSeed(t *testing.T) {
	f, app, s, r := shopCommentFixture(t)
	item := f.Shop.shelves[ShelfArmour][0]
	rng := rand.New(rand.NewSource(7))
	values := make([]int, 400)
	want := 0
	for i := range values {
		values[i] = rng.Intn(100)
		if values[i] < 30 {
			want++
		}
	}
	calls := 0
	f.ShopRandom = shopDrawSeam(values, &calls)
	got := 0
	for range values {
		base := len(r.samples)
		s.shopItemResponse(item)
		got += shopSpeechDrain(t, app, r, base)
	}
	if got != want || want == 0 || want == len(values) {
		t.Fatalf("%d requests over %d placements, want %d", got, len(values), want)
	}
	t.Logf("%d of %d placements spoke", got, len(values))
}
