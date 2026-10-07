package ui

import (
	"testing"
	"time"
)

type consumableShopFixture struct {
	*fakeShopTown
	view ShopScreenView
	open bool
}

func (s *consumableShopFixture) ShopScreen() ShopScreenView { return s.view }
func (s *consumableShopFixture) AtTownShop() bool           { return s.open }

func (s *consumableShopFixture) ShopClick(c ShopControl) TownAction {
	action := s.fakeShopTown.ShopClick(c)
	if c.Kind != ShopControlPackCell || c.Index < 0 || c.Index >= len(s.view.Pack) {
		return action
	}
	item := s.view.Pack[c.Index]
	if item.Count == 0 {
		return action
	}
	for i, held := range s.view.Table {
		if held.Count != 0 && held.UseItemKey != "staged:"+item.UseItemKey {
			continue
		}
		if held.Count == 0 {
			held = item
			held.UseItemKey, held.Count, held.Mine = "staged:"+item.UseItemKey, 0, true
		}
		held.Count++
		s.view.Table[i] = held
		s.view.Pack[c.Index].Count--
		if s.view.Pack[c.Index].Count == 0 {
			copy(s.view.Pack[c.Index:], s.view.Pack[c.Index+1:])
			s.view.Pack[len(s.view.Pack)-1] = ShopCell{}
		}
		break
	}
	return action
}

func (s *consumableShopFixture) ShopDrag(from, to ShopControl) TownAction {
	action := s.fakeShopTown.ShopDrag(from, to)
	if from.Kind == ShopControlTableCell && s.view.Table[from.Index].Count > 0 {
		s.view.Table[from.Index].Count--
		if s.view.Table[from.Index].Count == 0 {
			s.view.Table[from.Index] = ShopCell{}
		}
	}
	return action
}

func shopUseApp(t *testing.T) (*App, *consumableShopFixture) {
	a, base := shopDragTestApp(t)
	s := &consumableShopFixture{fakeShopTown: base, view: base.ShopScreen(), open: true}
	s.view.InputEpoch = new(byte)
	s.view.Pack[1] = ShopCell{UseItemKey: "potion health 30", Count: 2, Back: ShopBackEmpty + 1}
	a.SetTown(s)
	if !a.flow.showTown("") {
		t.Fatal("shop open")
	}
	return a, s
}

func tapShopUse(t *testing.T, a *App, at time.Time) {
	t.Helper()
	x, y, err := a.HeadlessShopPoint("pack", 1)
	if err != nil {
		t.Fatal(err)
	}
	a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, at)
	a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, at)
}

func TestShopConsumableDoubleUseAndImmediateSingleTrade(t *testing.T) {
	now := time.Unix(100, 0)
	for _, double := range []bool{false, true} {
		a, s := shopUseApp(t)
		tapShopUse(t, a, now)
		if len(s.clicked) != 1 || len(s.dragged) != 0 || s.view.Table[0].Count != 1 || s.view.Pack[1].Count != 1 {
			t.Fatal("first click did not stage exactly one unit")
		}
		if double {
			tapShopUse(t, a, now.Add(100*time.Millisecond))
		}
		a.step(appInput{}, now.Add(time.Second))
		if double {
			if len(s.dragged) != 1 || len(s.clicked) != 1 || s.dragged[0][0].Kind != ShopControlTableCell || s.dragged[0][1].Kind != ShopControlDoll || s.view.Table[0].Count != 0 {
				t.Fatalf("double click sale=%v use=%v", s.clicked, s.dragged)
			}
		} else if len(s.controls) != 1 || s.controls[0].Kind != ShopControlPackCell || len(s.dragged) != 0 {
			t.Fatalf("single click sale=%v use=%v", s.controls, s.dragged)
		}
	}
}

func TestShopConsumablePendingTapCannotCrossIdentityOrContext(t *testing.T) {
	for _, name := range []string{"item", "quantity", "staged item", "staged quantity", "staged owner", "member", "pack page", "shelf tab", "character tab", "stock refresh", "modal", "menu", "map", "escape", "right cancel"} {
		t.Run(name, func(t *testing.T) {
			a, s := shopUseApp(t)
			now := time.Unix(100, 0)
			tapShopUse(t, a, now)
			switch name {
			case "item":
				s.view.Pack[1].UseItemKey = "another potion"
			case "quantity":
				s.view.Pack[1].Count++
			case "staged item":
				s.view.Table[0].UseItemKey = "another staged potion"
			case "staged quantity":
				s.view.Table[0].Count++
			case "staged owner":
				s.view.Table[0].Mine = false
			case "member":
				s.view.Member++
			case "pack page":
				s.view.PackOffset++
			case "shelf tab":
				s.view.Chosen++
			case "character tab":
				s.view.Character.Statistics = !s.view.Character.Statistics
			case "stock refresh":
				s.view.InputEpoch = new(byte)
			case "modal":
				s.open = false
			case "menu":
				a.flow.screen = ScreenGameMenu
			case "map":
				a.flow.screen = ScreenMap
			case "escape":
				a.step(appInput{Escape: true}, now.Add(time.Millisecond))
			case "right cancel":
				a.step(appInput{SecondaryPressed: true}, now.Add(time.Millisecond))
			}
			a.step(appInput{}, now.Add(time.Second))
			if a.shopUseTap != nil || len(s.clicked) != 1 || len(s.dragged) != 0 {
				t.Fatalf("stale action pending=%v sale=%v use=%v", a.shopUseTap, s.clicked, s.dragged)
			}
		})
	}
}
