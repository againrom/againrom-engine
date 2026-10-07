package ui

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"
)

func TestMissionTabKeepsFigureAndStatisticsVisible(t *testing.T) {
	v := missionPaneViewer(t)
	v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 12, 34)})
	v.sel = selection{1}
	v.SetInventorySubject(InventorySubject{ID: 1, Figure: solidPic(160, 240, color.RGBA{R: 255, A: 255})})
	a := newTestApp(t, appRows(1), okLoader(t))
	a.flow.viewer, a.flow.screen = v, ScreenMap
	a.Layout(1024, 768)
	pane := missionPaneCompose(t, v)
	card, _, ok := v.missionCardPresent()
	if !ok {
		t.Fatal("missing initial stats card")
	}
	for _, key := range []string{"tab", "shift-tab", "tab"} {
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
		if v.characterPaneStatistics() {
			t.Fatal("Tab replaced the mission figure")
		}
		after, _, ok := v.missionCardPresent()
		if !ok || !bytes.Equal(after.Pix, card.Pix) || !bytes.Equal(missionPaneCompose(t, v).Pix, pane.Pix) {
			t.Fatal("Tab changed the figure or lower statistics card")
		}
	}
}

func TestShopStatisticsAcceptEquipDrops(t *testing.T) {
	for _, origin := range []struct {
		kind  ShopControlKind
		point image.Point
	}{
		{ShopControlShelfCell, shopDragShelfPoint},
		{ShopControlPackCell, ShopPackCellRect(0).Min.Add(image.Pt(4, 4))},
		{ShopControlTableCell, ShopTableCellRect(0).Min.Add(image.Pt(4, 4))},
	} {
		a, town := shopDragTestApp(t)
		town.stats = true
		now := time.Unix(1_700_000_000, 0)
		a.step(appInput{CursorX: origin.point.X, CursorY: origin.point.Y, PrimaryPressed: true}, now)
		a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y}, now)
		a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryReleased: true}, now)
		want := [][2]ShopControl{{{Kind: origin.kind}, {Kind: ShopControlDoll}}}
		if !reflect.DeepEqual(town.dragged, want) {
			t.Fatalf("origin %v: %v, want %v", origin.kind, town.dragged, want)
		}
	}
}
