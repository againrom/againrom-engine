package game

import (
	"crypto/sha256"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type packDragTownRig struct {
	f   *FrontEnd
	app *ui.App
	s   *townScreen
}

func packDragTownFixture(t *testing.T, count uint32, extra bool) packDragTownRig {
	t.Helper()
	f := releaseFront(t)
	f.Carried = f.NextParty()
	f.arriveInTown()
	f.Town.gold = 100000
	screen := f.TownScreen().(*townScreen)
	bow := mapload.SourceConstructedItem(mapload.ItemInstanceFromCode(0x8114, f.Table), f.Table)
	other := mapload.SourceConstructedItem(mapload.ItemInstanceFromCode(0x8115, f.Table), f.Table)
	stacks := []sim.ItemStack{sim.StackItem(bow, 6), sim.StackItem(bow, 2), sim.StackItem(bow, count), sim.StackItem(other, 2)}
	if extra {
		for _, code := range []uint16{0x8134, 0x8116, 0x1104, 0x1106} {
			stacks = append(stacks, sim.StackItem(mapload.SourceConstructedItem(mapload.ItemInstanceFromCode(code, f.Table), f.Table), 1))
		}
	}
	cityShopSetTestPack(t, f, screen.shopMemberIndex(), stacks...)
	cityShopGraph(t, f)
	r := packDragTownRig{f: f, s: screen}
	return r.cold(t, "seed")
}

func (r packDragTownRig) cells(t *testing.T) ([]sim.ItemStack, []sim.SavedObjectID) {
	t.Helper()
	pack := r.s.shopPackStacks()
	ids := slices.Clone(cityShopRoots(t, r.f, r.s.shopMemberIndex()).Pack)
	if len(pack) != len(ids) {
		t.Fatal("town pack and graph root populations differ", len(pack), len(ids))
	}
	return pack, ids
}

func packDragTownRequire(t *testing.T, r packDragTownRig, want []sim.ItemStack, ids []sim.SavedObjectID) {
	t.Helper()
	got, gotIDs := r.cells(t)
	if !slices.Equal(gotIDs, ids) || !slices.EqualFunc(got, want, sim.StackStateEqual) {
		t.Fatalf("town pack IDs %v, want %v; cells\n got %+v\nwant %+v", gotIDs, ids, got, want)
	}
}

func (r packDragTownRig) point(t *testing.T, index int) image.Point {
	t.Helper()
	x, y, err := r.app.HeadlessShopPoint("pack", index+1-r.s.packBase)
	if err != nil {
		t.Fatal(err)
	}
	return image.Pt(x, y)
}

func (r packDragTownRig) drag(t *testing.T, source, target int, whole bool) {
	t.Helper()
	from, to := r.point(t, source), r.point(t, target)
	prefix := ""
	if whole {
		prefix = "shift-"
	}
	for _, edge := range []struct {
		name string
		at   image.Point
	}{{"press", from}, {"move", to}, {"release", to}} {
		if source == target && edge.name == "move" {
			edge.at.X += 6
		}
		if err := r.app.HeadlessPointer(prefix+edge.name, edge.at.X, edge.at.Y); err != nil {
			t.Fatal(err)
		}
	}
}

func (r packDragTownRig) cold(t *testing.T, label string) packDragTownRig {
	t.Helper()
	dir := filepath.Join(packDragOutput(t), label)
	save, _, _ := r.f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("town ordinary SAV %q: %v", name, err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	_, actions, indices := cityProjectionWire(t, raw)
	_, ids := r.cells(t)
	for _, id := range ids {
		if indices[id] == 0 {
			t.Fatal("ordinary town SAV omitted a current pack identity", id)
		}
	}
	if actions == nil {
		t.Fatal("ordinary town SAV omitted current state")
	}
	g := releaseFront(t)
	app := openLocalTownSAV(t, g, dir, name)
	app.Layout(640, 480)
	screen := kitOwnerShop(t, g, app)
	closeShopTip(t, app, screen)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: town SAV %s SHA256=%x bytes=%d roots=%v", label, filepath.Join(dir, name), sha256.Sum256(raw), len(raw), ids)
	return packDragTownRig{f: g, app: app, s: screen}
}

func TestReleasePackStackDragTownGestures(t *testing.T) {
	for _, tc := range []struct {
		name           string
		source, target int
		whole, single  bool
	}{
		{"merge-one", 2, 1, false, false},
		{"merge-whole", 2, 1, true, false},
		{"unequal-partial", 2, 3, false, false},
		{"unequal-whole", 2, 3, true, false},
		{"unequal-singleton", 2, 3, false, true},
		{"self", 2, 2, true, false},
		{"onto-purse", 2, -1, true, false},
		{"from-purse", -1, 2, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			count := uint32(3)
			if tc.single {
				count = 1
			}
			r := packDragTownFixture(t, count, false)
			before, ids := r.cells(t)
			want, wantIDs := slices.Clone(before), slices.Clone(ids)
			gold, table := r.f.Town.Gold(), r.f.Shop.Table()
			var total uint32
			for _, stack := range before {
				total += stack.Count
			}
			switch {
			case tc.source == 2 && tc.target == 1:
				take := uint32(1)
				if tc.whole {
					take = count
				}
				want[1].Count += take
				want[2].Count -= take
				if want[2].Count == 0 {
					want, wantIDs = slices.Delete(want, 2, 3), slices.Delete(wantIDs, 2, 3)
				}
			case tc.target == 3 && (tc.whole || tc.single):
				want[2], want[3] = want[3], want[2]
				wantIDs[2], wantIDs[3] = wantIDs[3], wantIDs[2]
			}
			r.drag(t, tc.source, tc.target, tc.whole)
			packDragTownRequire(t, r, want, wantIDs)
			if r.f.Town.Gold() != gold || !reflect.DeepEqual(r.f.Shop.Table(), table) {
				t.Fatal("town pack drag changed money or trade table")
			}
			var after uint32
			for _, stack := range r.s.shopPackStacks() {
				after += stack.Count
			}
			if after != total {
				t.Fatal("town pack drag lost quantity", total, after)
			}
			cold := r.cold(t, "after-gesture")
			packDragTownRequire(t, cold, want, wantIDs)
			want = slices.Clone(want)
			want[0].Count++
			want[1].Count--
			cold.drag(t, 1, 0, false)
			packDragTownRequire(t, cold, want, wantIDs)
		})
	}
}

func packDragTownArrow(t *testing.T, app *ui.App, kind ui.ShopControlKind) {
	t.Helper()
	var hits []image.Point
	for y := range 480 {
		for x := range 640 {
			if control, ok := ui.ShopControlAt(image.Pt(x, y)); ok && control.Kind == kind {
				hits = append(hits, image.Pt(x, y))
			}
		}
	}
	if len(hits) == 0 {
		t.Fatal("shop arrow has no production hit pixels", kind)
	}
	at := hits[len(hits)/2]
	shopTap(t, app, at, "press", "release")
}

func TestReleasePackStackDragTownControlsAndTrading(t *testing.T) {
	for _, tc := range []struct {
		name           string
		source, target int
		whole          bool
	}{
		{"unequal partial", 2, 3, false},
		{"self", 2, 2, true},
		{"onto purse", 2, -1, true},
		{"from purse", -1, 2, true},
	} {
		t.Run("absent graph "+tc.name, func(t *testing.T) {
			r := packDragTownFixture(t, 3, false)
			r.f.Town.cityObjects = nil
			party, shop, gold := mapload.CloneParty(r.f.Carried), cloneShopMutation(r.f.Shop), r.f.Town.Gold()
			r.drag(t, tc.source, tc.target, tc.whole)
			if r.f.Town.cityObjects != nil || !reflect.DeepEqual(r.f.Carried, party) || !reflect.DeepEqual(cloneShopMutation(r.f.Shop), shop) || r.f.Town.Gold() != gold {
				t.Fatal("refused town drag published a graph or changed party, shop or purse")
			}
		})
	}
	t.Run("initial graph construction", func(t *testing.T) {
		r := packDragTownFixture(t, 3, false)
		pack, _ := r.cells(t)
		want := slices.Clone(pack)
		want[2], want[3] = want[3], want[2]
		r.f.Town.cityObjects = nil
		r.drag(t, 2, 3, true)
		if r.f.Town.cityObjects == nil {
			t.Fatal("initial town pack drag did not construct its graph")
		}
		_, ids := r.cells(t)
		packDragTownRequire(t, r, want, ids)
		cold := r.cold(t, "constructed-graph")
		packDragTownRequire(t, cold, want, ids)
	})
	t.Run("scrolled selected cells", func(t *testing.T) {
		r := packDragTownFixture(t, 3, true)
		want, ids := r.cells(t)
		want = slices.Clone(want)
		at := r.point(t, 1)
		if err := r.app.HeadlessPointer("wheel-down", at.X, at.Y); err != nil {
			t.Fatal(err)
		}
		if r.s.packBase != 1 {
			t.Fatal("shop pack wheel did not move one position", r.s.packBase)
		}
		r.drag(t, 2, 1, false)
		want[1].Count++
		want[2].Count--
		packDragTownRequire(t, r, want, ids)
		packDragTownArrow(t, r.app, ui.ShopControlPackRight)
		if r.s.packBase != 2 {
			t.Fatal("right arrow did not scroll the pack", r.s.packBase)
		}
		packDragTownArrow(t, r.app, ui.ShopControlPackLeft)
		if r.s.packBase != 1 {
			t.Fatal("left arrow did not scroll the pack", r.s.packBase)
		}
		packDragTownRequire(t, r, want, ids)
	})
	t.Run("existing sell and buy", func(t *testing.T) {
		r := packDragTownFixture(t, 3, false)
		pack, _ := r.cells(t)
		code := pack[2].Code
		quantity := func() uint32 {
			var count uint32
			for _, stack := range r.s.shopPackStacks() {
				if stack.Code == code {
					count += stack.Count
				}
			}
			return count
		}
		before, gold := quantity(), r.f.Town.Gold()
		from := r.point(t, 2)
		x, y, err := r.app.HeadlessShopPoint("table", 0)
		if err != nil {
			t.Fatal(err)
		}
		shopDrag(t, r.app, from, image.Pt(x, y), "press", "move", "release")
		places := r.f.Shop.Table()
		if quantity() != before-1 || len(places) != 1 || places[0].ShopItem.Count != 1 || !places[0].Mine {
			t.Fatal("existing pack-to-table staging changed", quantity(), places)
		}
		payout := (places[0].ShopItem.Price + 1) / 2
		cityShopPointer(t, r.app, "button", 2, "press", "release")
		if r.f.Town.Gold() != gold+int(payout) || len(r.f.Shop.Table()) != 0 {
			t.Fatal("existing sell payment changed", r.f.Town.Gold(), gold, payout)
		}
		closeShopTip(t, r.app, r.s)
		cityShopPointer(t, r.app, "shelf_pick", roomWeapons, "press", "release")
		if r.s.shopChosen != roomWeapons {
			t.Fatal("weapons shelf did not open", r.s.shopChosen)
		}
		index := slices.IndexFunc(r.f.Shop.Shelf(ShelfWeapons), func(item ShopItem) bool { return uint16(item.Code) == code })
		if index < 0 {
			t.Fatal("sold bow absent from weapons shelf")
		}
		var shelf image.Point
		for range 32 {
			x, y, err = r.app.HeadlessShopPoint("shelf", index-r.s.shelfBase)
			if err == nil {
				shelf = image.Pt(x, y)
				break
			}
			x, y, err = r.app.HeadlessShopPoint("shelf", 0)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.app.HeadlessPointer("wheel-down", x, y); err != nil {
				t.Fatal(err)
			}
		}
		if shelf == (image.Point{}) {
			t.Fatal("sold bow never reached a visible shelf cell")
		}
		shopDrag(t, r.app, shelf, r.point(t, 0), "press", "move", "release")
		places = r.f.Shop.Table()
		if len(places) != 1 || places[0].Mine || places[0].ShopItem.Count != 1 || uint16(places[0].ShopItem.Code) != code || quantity() != before-1 {
			t.Fatal("existing shelf-to-pack staging changed", places, quantity())
		}
		cost := int(places[0].ShopItem.Price)
		closeShopTip(t, r.app, r.s)
		cityShopPointer(t, r.app, "button", 1, "press", "release")
		if quantity() != before || r.f.Town.Gold() != gold+int(payout)-cost || len(r.f.Shop.Table()) != 0 {
			t.Fatal("existing buy quantity or payment changed", quantity(), before, r.f.Town.Gold(), gold, payout, cost, r.f.Shop.Table())
		}
	})
}
