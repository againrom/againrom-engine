package ui

import (
	"image"
	"slices"
	"testing"
	"time"
)

// shopPaneTestView is a shop view whose pane stands at rect, with or without
// the book corner, over shopTestMask's doll.
func shopPaneTestView(rect image.Rectangle, noBook bool) ShopScreenView {
	mask := shopTestMask()
	return ShopScreenView{SlotMask: mask, OrdinaryDollMask: mask, Book: true,
		Character: TownCharacterView{HasSubject: true, MemberCount: 2, PaneRect: rect, NoBook: noBook, Session: CharacterPaneShop}}
}

var shopPaneKinds = []ShopControlKind{ShopControlPickerPrev, ShopControlPickerNext, ShopControlCharacterMode, ShopControlBook}

// A pane the room data moves takes every hit test with it: each picker,
// mode and book pixel answers at the moved place and no longer at the
// region, and so do the doll, the item drop, the town hover, the town
// corners and the headless doll target.
func TestShopPaneHitTestsFollowThePaneRectangle(t *testing.T) {
	moved := image.Rect(200, 20, 360, 262)
	delta := moved.Min.Sub(TownCharacterRegion.Min)
	at, away := shopPaneTestView(moved, false), shopPaneTestView(TownCharacterRegion, false)
	kinds := map[ShopControlKind]int{}
	for y := TownCharacterRegion.Min.Y; y < TownCharacterRegion.Max.Y; y++ {
		for x := TownCharacterRegion.Min.X; x < TownCharacterRegion.Max.X; x++ {
			p := image.Pt(x, y)
			want, ok := shopScreenControlAt(away, p)
			if !ok || !slices.Contains(shopPaneKinds, want.Kind) {
				continue
			}
			kinds[want.Kind]++
			if got, ok := shopScreenControlAt(at, p.Add(delta)); !ok || got != want {
				t.Fatalf("moved pane at %v answers %+v %v, the region at %v answers %+v", p.Add(delta), got, ok, p, want)
			}
			if got, ok := shopScreenControlAt(at, p); ok && slices.Contains(shopPaneKinds, got.Kind) {
				t.Fatalf("moved pane still answers %+v at the region's %v", got, p)
			}
		}
	}
	for _, k := range shopPaneKinds {
		if kinds[k] == 0 {
			t.Fatalf("the region answers no pixel for control kind %d", k)
		}
	}
	doll := characterPaneFigureOrigin(moved).Add(image.Pt(50, 100))
	if got, ok := shopGridControlAt(at, doll); !ok || got != (ShopControl{Kind: ShopControlDoll, Index: 3}) {
		t.Fatalf("the moved doll's marked pixel answers %+v %v", got, ok)
	}
	if got, ok := shopGridControlAt(at, doll.Sub(delta)); ok && got.Kind == ShopControlDoll {
		t.Fatalf("the region's doll pixel still answers %+v", got)
	}
	if !shopPaneTakesItemAt(at, moved.Min.Add(image.Pt(80, 120))) || shopPaneTakesItemAt(at, TownCharacterRegion.Min.Add(image.Pt(80, 120))) {
		t.Fatal("the item drop does not follow the pane")
	}
	if r := shopWantRegion(at, ShopControl{Kind: ShopControlDoll}); r != characterPaneFigureRect(moved) {
		t.Fatalf("the headless doll region is %v, want %v", r, characterPaneFigureRect(moved))
	}
	surface := TownSurfaceView{Hero: at.Character, HeroSlotMask: shopTestMask()}
	surface.HeroSlotInfo[3] = []string{"a helm"}
	if lines, ok := TownHeroHoverLines(surface, doll); !ok || len(lines) != 1 {
		t.Fatalf("the town hover at the moved doll answers %v %v", lines, ok)
	}
	if _, ok := TownHeroHoverLines(surface, doll.Sub(delta)); ok {
		t.Fatal("the town hover still answers at the region")
	}
	next := CharacterPaneCornerRect(moved, CharacterPaneNext)
	if got, ok := TownSurfaceControlAt(surface, next.Min.Add(image.Pt(2, 2))); !ok || got.Kind != TownSurfaceControlNext {
		t.Fatalf("the town's next arrow on the moved pane answers %+v %v", got, ok)
	}
}

// A pane without the book corner has no book control: no pixel answers it,
// the strip stays closed while the view asks for the book, and the B key
// sends nothing; with the corner, the key sends the book control.
func TestShopPaneWithoutBookHasNoBookControl(t *testing.T) {
	for _, noBook := range []bool{false, true} {
		v := shopPaneTestView(TownCharacterRegion, noBook)
		answered := false
		for y := TownCharacterRegion.Min.Y; y < TownCharacterRegion.Max.Y; y++ {
			for x := TownCharacterRegion.Min.X; x < TownCharacterRegion.Max.X; x++ {
				if c, ok := shopScreenControlAt(v, image.Pt(x, y)); ok && c.Kind == ShopControlBook {
					answered = true
				}
			}
		}
		if answered == noBook || v.bookControl() == noBook || v.bookOpen() == noBook {
			t.Fatalf("noBook %v: corner answers %v, control %v, open %v", noBook, answered, v.bookControl(), v.bookOpen())
		}
		a, town := shopDragTestApp(t)
		town.character = v.Character
		a.stepTownAt(appInput{Book: true}, time.Now())
		sent := slices.ContainsFunc(town.controls, func(c ShopControl) bool { return c.Kind == ShopControlBook })
		if sent == noBook {
			t.Fatalf("noBook %v: the B key sent the book control %v (%+v)", noBook, sent, town.controls)
		}
	}
}
