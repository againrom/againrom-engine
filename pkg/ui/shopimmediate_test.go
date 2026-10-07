package ui

import (
	"testing"
	"time"
)

func TestShopUsableSingleClickStagesOnTheReleaseFrame(t *testing.T) {
	a, s := shopUseApp(t)
	now := time.Unix(100, 0)
	tapShopUse(t, a, now)
	if len(s.clicked) != 1 || len(s.dragged) != 0 {
		t.Fatalf("release frame deferred the transfer: clicks=%v uses=%v", s.clicked, s.dragged)
	}
	a.step(appInput{}, now.Add(time.Second))
	if len(s.clicked) != 1 || len(s.dragged) != 0 {
		t.Fatal("idle expiry added a second transfer")
	}
}

func TestShopDoubleClickUsesTheStagedUnitAfterPackCompaction(t *testing.T) {
	for _, following := range []ShopCell{{}, {UseItemKey: "different wearable", Count: 1, Back: ShopBackEmpty + 1}} {
		a, s := shopUseApp(t)
		s.view.Pack[1].Count = 1
		s.view.Pack[2] = following
		now := time.Unix(100, 0)
		tapShopUse(t, a, now)
		if s.view.Pack[1].UseItemKey != following.UseItemKey || s.view.Table[0].Count != 1 {
			t.Fatal("the first click did not stage and compact the pack")
		}
		tapShopUse(t, a, now.Add(100*time.Millisecond))
		if len(s.clicked) != 1 || len(s.dragged) != 1 || s.dragged[0][0].Kind != ShopControlTableCell || s.view.Pack[1].UseItemKey != following.UseItemKey || s.view.Pack[1].Count != following.Count {
			t.Fatalf("double click touched the next pack occupant: sale=%v use=%v pack=%+v", s.clicked, s.dragged, s.view.Pack[1])
		}
	}
}

func TestShopDoubleClickUsesOneUnitWhenTheStageJoinsAnExistingStack(t *testing.T) {
	a, s := shopUseApp(t)
	s.view.Table[0] = s.view.Pack[1]
	s.view.Table[0].UseItemKey = "staged:" + s.view.Pack[1].UseItemKey
	s.view.Table[0].Count, s.view.Table[0].Mine = 3, true
	now := time.Unix(100, 0)
	tapShopUse(t, a, now)
	tapShopUse(t, a, now.Add(100*time.Millisecond))
	if s.view.Table[0].Count != 3 || s.view.Pack[1].Count != 1 || len(s.dragged) != 1 {
		t.Fatal("double click did not consume only the newly staged unit")
	}
}

func TestShopDoubleClickStillUsesThePackWhenTheTableIsFull(t *testing.T) {
	a, s := shopUseApp(t)
	for i := range s.view.Table {
		s.view.Table[i] = ShopCell{UseItemKey: "other", Count: 1, Back: ShopBackEmpty + 1}
	}
	now := time.Unix(100, 0)
	tapShopUse(t, a, now)
	tapShopUse(t, a, now.Add(100*time.Millisecond))
	if len(s.dragged) != 1 || s.dragged[0][0].Kind != ShopControlPackCell || len(s.clicked) != 1 {
		t.Fatalf("full table blocked use: sale=%v use=%v", s.clicked, s.dragged)
	}
}
