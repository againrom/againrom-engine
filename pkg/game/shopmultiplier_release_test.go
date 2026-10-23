package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// shopStockActionBound is the longest a player-facing shop action may take at
// the multiplier's bound: a town visit's stock generation, one shelf page
// step, one page draw and one purchase. Each is held to it on a statistic a
// single load spike on a shared machine cannot move: the 95th percentile of
// the page steps, the median of the sampled draws, and the minimum of three
// repetitions of a generation or a purchase. A generation is timed on its
// thread's CPU time, which a machine kept busy by other processes does not
// stretch.
const shopStockActionBound = 250 * time.Millisecond

// shopStockQuantile is the q-quantile of ds by the nearest-rank rule.
func shopStockQuantile(ds []time.Duration, q float64) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	sorted := slices.Clone(ds)
	slices.Sort(sorted)
	return sorted[min(len(sorted)-1, int(q*float64(len(sorted))))]
}

// shopStockMinOfThree times act three times on its thread's CPU time and
// answers the fastest and the slowest run.
func shopStockMinOfThree(act func()) (best, slowest time.Duration) {
	best = time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		took := shopStockThreadTime(act)
		best, slowest = min(best, took), max(slowest, took)
	}
	return best, slowest
}

func TestReleaseShopStockMultiplierDefaultMatchesMain(t *testing.T) {
	f := shopOrderFront(t)
	h := sha256.New()
	for _, ceiling := range shelfTestCeilings(t, f) {
		for _, seed := range []int64{1, 7, 42} {
			s := NewShop(ceiling)
			s.Generate(f.Table, seed)
			explicit := NewShop(ceiling)
			explicit.stockMultiplier = 1
			explicit.Generate(f.Table, seed)
			assertShopRolledTimes(t, s, explicit, 1)
			if err := json.NewEncoder(h).Encode(s.shelves); err != nil {
				t.Fatal(err)
			}
		}
	}
	const mainDigest = "b8b288e13ec90e4a898eb50a59a7d5e89114dace965103d763fe48083e690dda"
	if got := fmt.Sprintf("%x", h.Sum(nil)); got != mainDigest {
		t.Fatalf("default stock digest %s, want main %s", got, mainDigest)
	}
	t.Logf("main stock digest %s; %d ceilings, three seeds, four shelves", mainDigest, len(shelfTestCeilings(t, f)))
}

func TestReleaseShopStockMultiplierRollsEveryShelf(t *testing.T) {
	f := shopOrderFront(t)
	for _, ceiling := range shelfTestCeilings(t, f) {
		for _, seed := range []int64{1, 7, 42} {
			base := NewShop(ceiling)
			base.Generate(f.Table, seed)
			ns := []int32{3}
			if seed == 42 {
				ns = append(ns, MaxShopStockMultiplier)
			}
			for _, n := range ns {
				got := NewShop(ceiling)
				got.stockMultiplier = n
				start := time.Now()
				got.Generate(f.Table, seed)
				took := time.Since(start)
				assertShopRolledTimes(t, base, got, int64(n))
				for _, shelf := range got.shelves {
					for _, item := range shelf {
						if width := f.tipFont().Advance(ui.GroupDigits(int64(item.Count))) + 1; width > 69 {
							t.Fatalf("quantity %d needs %d pixels; cell allows 69", item.Count, width)
						}
					}
				}
				if n == MaxShopStockMultiplier {
					t.Logf("ceiling %d: N=%d generated in %v; entries %d/%d/%d/%d, one roll %d/%d/%d/%d", ceiling, n, took,
						len(got.shelves[ShelfArmour]), len(got.shelves[ShelfWeapons]), len(got.shelves[ShelfMagic]), len(got.shelves[ShelfBooks]),
						len(base.shelves[ShelfArmour]), len(base.shelves[ShelfWeapons]), len(base.shelves[ShelfMagic]), len(base.shelves[ShelfBooks]))
				}
			}
		}
	}
}

func TestReleaseShopStockMultiplierListsAndSellsAtTheBound(t *testing.T) {
	f := currentTown(t, nil, nil)
	f.SoundPlayer, f.speech.player, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil, nil
	f.SetDeterministicFrames(true)
	base := cloneShopMutation(f.Shop)
	raw := currentTownSave(t, f)
	if err := f.SetShopStockMultiplier(1); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, currentTownSave(t, f)) {
		t.Fatal("explicit default changed SAV bytes")
	}
	if err := f.SetShopStockMultiplier(MaxShopStockMultiplier); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, currentTownSave(t, f)) {
		t.Fatal("launch option entered SAV")
	}
	visit := NewShop(f.Shop.Ceiling())
	visit.stockMultiplier = MaxShopStockMultiplier
	generated, generatedWorst := shopStockMinOfThree(func() { visit.Generate(f.Table, f.Shop.drawSeed) })

	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.WriteOriginal("", raw); err != nil {
		t.Fatal(err)
	}
	app := f.App("shop stock multiplier")
	app.Layout(640, 480)
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
	start := time.Now()
	for _, action := range []string{"load game", "@first"} {
		if err := app.HeadlessActivate(action); err != nil {
			t.Fatal(err)
		}
	}
	loaded := time.Since(start)
	assertShopRolledTimes(t, base, f.Shop, MaxShopStockMultiplier)
	t.Logf("N=%d: town stock at ceiling %d generated in %v; load and town arrival %v", MaxShopStockMultiplier, f.Shop.Ceiling(), generated, loaded)

	// The campaign's largest shop ceiling gives the longest Magic Items
	// shelf; the loaded shop is regenerated at it for the screen checks.
	var top int32
	for _, ch := range f.Campaign.Value().Chapters {
		top = max(top, int32(ch.ShopMax))
	}
	topBase := NewShop(top)
	topBase.Generate(f.Table, f.Shop.drawSeed)
	f.Shop.ceiling = top
	topTook, topWorst := shopStockMinOfThree(func() { f.Shop.Generate(f.Table, f.Shop.drawSeed) })

	assertShopRolledTimes(t, topBase, f.Shop, MaxShopStockMultiplier)
	if n := len(f.Shop.Shelf(ShelfMagic)); n < 100*len(topBase.Shelf(ShelfMagic)) {
		t.Fatalf("magic shelf holds %d entries at ceiling %d, one roll %d", n, top, len(topBase.Shelf(ShelfMagic)))
	}
	t.Logf("N=%d: stock at ceiling %d generated in %v; entries %d/%d/%d/%d", MaxShopStockMultiplier, top, topTook,
		len(f.Shop.Shelf(ShelfArmour)), len(f.Shop.Shelf(ShelfWeapons)), len(f.Shop.Shelf(ShelfMagic)), len(f.Shop.Shelf(ShelfBooks)))
	// samples holds every timed page step and draw, and for generation and
	// purchase the minimum of each three repetitions.
	samples := map[string][]time.Duration{"generate": {generated, topTook}}
	worst := map[string]time.Duration{"generate": max(generatedWorst, topWorst)}
	note := func(what string, d time.Duration) {
		samples[what] = append(samples[what], d)
		worst[what] = max(worst[what], d)
	}

	s := shopOrderEnter(t, f, app)
	pointer := shopPointer(t, app)
	down := shopOrderControl(t, ui.ShopControlArrowDown)
	for room, rack := range shopRoomShelves {
		pointer("shelf_pick", room, "press", "release")
		items := f.Shop.Shelf(rack.shelf)
		pages := 0
		for v := s.ShopScreen(); ; v = s.ShopScreen() {
			for i, cell := range v.Shelf {
				k := v.ShelfOffset + i
				if k >= len(items) {
					break
				}
				if cell.Price != items[k].Price || cell.Count != uint32(items[k].Count) {
					t.Fatalf("shelf %s cell %d paints %d x%d, model %d x%d", rack.shelf, k, cell.Price, cell.Count, items[k].Price, items[k].Count)
				}
			}
			if v.ShelfOffset+len(v.Shelf) >= len(items) {
				break
			}
			start := time.Now()
			shopTap(t, app, down, "press", "release")
			note("page", time.Since(start))
			if s.ShopScreen().ShelfOffset <= v.ShelfOffset {
				t.Fatalf("the down arrow left shelf %s at offset %d", rack.shelf, v.ShelfOffset)
			}
			pages++
			if pages%500 == 0 || v.ShelfOffset+2*len(v.Shelf) >= len(items) {
				start := time.Now()
				if _, _, err := app.HeadlessFrame(); err != nil {
					t.Fatal(err)
				}
				note("draw", time.Since(start))
			}
		}
		// Buy three stacks from the last page's first cell, then three from
		// the shelf's first cell.
		bought := 0
		for _, last := range []bool{true, false} {
			if !last {
				pointer("shelf_pick", (room+1)%len(shopRoomShelves), "press", "release")
				pointer("shelf_pick", room, "press", "release")
			}
			fastest := time.Duration(1<<63 - 1)
			for rep := 0; rep < 3 && len(f.Shop.Shelf(rack.shelf)) != 0; rep++ {
				v := s.ShopScreen()
				want := f.Shop.Shelf(rack.shelf)[v.ShelfOffset]
				units := len(s.shopPackItemInstances())
				f.Town.gold = 1 << 30
				start := time.Now()
				pointer("shelf", 0, "shift-press", "shift-release")
				if held := f.Shop.Table(); len(held) != 1 || !shopItemStateEqual(held[0].ShopItem, want) {
					t.Fatal("screen did not stage the complete stack", held, want)
				}
				pointer("button", 1, "press", "release")
				took := time.Since(start)
				fastest = min(fastest, took)
				worst["purchase"] = max(worst["purchase"], took)
				if len(f.Shop.Table()) != 0 || len(s.shopPackItemInstances())-units != int(want.Count) || f.Town.Gold() != (1<<30)-int(want.Price)*int(want.Count) {
					t.Fatal("screen did not sell the complete stack at its unit price", want)
				}
				bought++
			}
			samples["purchase"] = append(samples["purchase"], fastest)
		}
		t.Logf("shelf %s: %d entries listed over %d page steps; %d stacks bought through App pointer controls", rack.shelf, len(items), pages, bought)
	}
	held := map[string]time.Duration{
		"generate": slices.Max(samples["generate"]),
		"page":     shopStockQuantile(samples["page"], 0.95),
		"draw":     shopStockQuantile(samples["draw"], 0.5),
		"purchase": slices.Max(samples["purchase"]),
	}
	statistic := map[string]string{
		"generate": "minimum of three on thread CPU time, larger of the two ceilings",
		"page":     "95th percentile",
		"draw":     "median",
		"purchase": "minimum of three, largest over the eight buy positions",
	}
	for _, what := range []string{"generate", "page", "draw", "purchase"} {
		t.Logf("N=%d %s: %s %v (median %v, worst %v, %d samples)", MaxShopStockMultiplier, what, statistic[what], held[what],
			shopStockQuantile(samples[what], 0.5), worst[what], len(samples[what]))
		if held[what] > shopStockActionBound {
			t.Errorf("%s took %v (%s) at the bound, over %v", what, held[what], statistic[what], shopStockActionBound)
		}
	}
}
