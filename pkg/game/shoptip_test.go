package game

import (
	"image"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// The tip widget's own load (1011 spec; SHOP-TIP-045; DIV-132).

func TestReadShopTipReadsTheWholeFileWithNoLineSplit(t *testing.T) {
	payload := []byte("Buy low\rand sell high.")
	fsys := townTextFS(t, []synth.File{{Path: "text/tips/shop1.txt", Data: payload}})

	got, ok := ReadShopTip(fsys, roomTip(roomShop).Text)
	if !ok {
		t.Fatal("ReadShopTip reported the file absent")
	}
	if got != string(payload) {
		t.Fatalf("ReadShopTip = %q, want the payload unsplit: %q", got, string(payload))
	}
}

// A missing file, or a nil source, answers false — no widget draws when
// there is nothing to draw (matching every other install text reader).
func TestReadShopTipMissingFileAnswersFalse(t *testing.T) {
	fsys := townTextFS(t, nil)
	if _, ok := ReadShopTip(fsys, roomTip(roomShop).Text); ok {
		t.Error("ReadShopTip found a file that was never shipped")
	}
	if _, ok := ReadShopTip(nil, roomTip(roomShop).Text); ok {
		t.Error("ReadShopTip with a nil source answered true")
	}
}

func TestShopTipPathIsMainTextTipsShop1(t *testing.T) {
	if got := roomTip(roomShop).Text; got != "main/text/tips/shop1.txt" {
		t.Fatalf("shop room tip = %q, want main/text/tips/shop1.txt", got)
	}
	if ShopTip2Path != "main/text/tips/shop2.txt" {
		t.Fatalf("ShopTip2Path = %q, want main/text/tips/shop2.txt", ShopTip2Path)
	}
}

// Entering the shop loads shop1.txt and hands it to the screen as Tip
// (DIV-132: the gate SHOP-TIP-045 leaves Unknown is authored open, so every
// entry reads the file with no toggle of its own).
func TestEnteringTheShopLoadsTheTipFromTheInstall(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: shopTable(), Archives: &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/tips/shop1.txt", Data: []byte("Mind your purse.")}})}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	s := f.TownScreen().(*townScreen)

	s.Choose(1) // the shop door, townDoors[1]
	if got := s.ShopScreen().TipPanel.Text; got != "Mind your purse." {
		t.Fatalf("ShopScreen().Tip after entering the shop = %q, want the install's own shop1.txt", got)
	}
}

// EVERY ENTRY RE-READS THE FILE (SHOP-TIP-045: activation loads it, with no
// cache of its own). Swapping the install's own Archives between two visits
// and seeing the second visit's text change proves this is a fresh read each
// time, not a value latched on the front end's first shop entry.
func TestEveryShopEntryRereadsTheTip(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: shopTable(), Archives: &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/tips/shop1.txt", Data: []byte("first visit")}})}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	s := f.TownScreen().(*townScreen)

	s.Choose(1)
	if got := s.ShopScreen().TipPanel.Text; got != "first visit" {
		t.Fatalf("first entry Tip = %q, want %q", got, "first visit")
	}

	s.Back()
	f.Archives = &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/tips/shop1.txt", Data: []byte("second visit")}})}
	s.Choose(1)
	if got := s.ShopScreen().TipPanel.Text; got != "second visit" {
		t.Fatalf("second entry Tip = %q, want %q — the load did not re-run", got, "second visit")
	}
}

// An install shipping no shop1.txt draws no widget: ShopScreen().Tip is
// empty and nothing in ComposeShopScreen has anything to wrap.
func TestEnteringTheShopWithNoTipFileLeavesTipEmpty(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: shopTable(), Archives: &Archives{Containers: townTextFS(t, nil)}}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	s := f.TownScreen().(*townScreen)

	s.Choose(1)
	if got := s.ShopScreen().TipPanel.Text; got != "" {
		t.Fatalf("ShopScreen().Tip with no shop1.txt shipped = %q, want empty", got)
	}
}

// Entering the shop with a front end that carries no Archives at all does not
// panic, the same "every picture is optional" contract shopArt() already
// keeps (shopart.go): the nil guard at the load site is exercised, not
// bypassed by constructing the room directly.
func TestEnteringTheShopWithNoArchivesDoesNotPanic(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: shopTable()}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	s := f.TownScreen().(*townScreen)

	s.Choose(1)
	if got := s.ShopScreen().TipPanel.Text; got != "" {
		t.Fatalf("ShopScreen().Tip with no Archives = %q, want empty", got)
	}
}

// The room tips the description gives are the panels the ui lays out: the
// same text addresses and rectangles (TOWN-015, TOWN-021, SHOP-TIP-045).
func TestRoomTipsAreTheDescribedTextsAndRectangles(t *testing.T) {
	for _, tc := range []struct {
		room townRoom
		text string
		rect image.Rectangle
	}{
		{roomTavern, "main/text/tips/inn.txt", ui.TavernTipRect},
		{roomShop, "main/text/tips/shop1.txt", ui.ShopTipRect()},
		{roomSchool, "main/text/tips/training.txt", ui.SchoolTipRect},
	} {
		if tip := roomTip(tc.room); tip.Text != tc.text || tip.Rect.Rectangle() != tc.rect {
			t.Errorf("room %d tip = %q %v, want %q %v", tc.room, tip.Text, tip.Rect.Rectangle(), tc.text, tc.rect)
		}
	}
}
