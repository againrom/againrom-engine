package game

import (
	"image"
	"os"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// releaseShopApp enters a production shop through App's load and town input
// routes. The returned screen is the same TownShopScreen App dispatches to;
// tests using it observe model state but never invoke its mutation methods.
func releaseShopApp(t *testing.T) (*ui.App, *townScreen) {
	t.Helper()
	f := releaseFront(t)
	f.Carried = f.NextParty()
	f.arriveInTown()
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC), payload); err != nil {
		t.Fatal(err)
	}
	app := f.App("1006-release-shop-hit")
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	s := registerTownFront(f.TownScreen().(*townScreen), f)
	for n := 0; s.room == roomTalk && n < 32; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomShop {
		t.Fatalf("App entered room %d, want shop", s.room)
	}
	return app, s
}

func releaseShopMaskPoint(t *testing.T, s *townScreen,
	accept func(ui.ShopScreenView, *image.RGBA, *image.RGBA, int, int) bool) (image.Point, int) {
	t.Helper()
	v := s.ShopScreen()
	if v.SlotMask == nil {
		t.Fatal("production shop has no doll mask")
	}
	frame := ui.ComposeShopScreen(v, image.Point{}, false, nil, false)
	withoutText := v
	withoutText.Font = nil
	withoutText.Character.Font = nil
	plain := ui.ComposeShopScreen(withoutText, image.Point{}, false, nil, false)
	for y := 0; y < v.SlotMask.H; y++ {
		for x := 0; x < v.SlotMask.W; x++ {
			slot, ok := v.SlotMask.At(x, y)
			if ok && accept(v, frame, plain, x, y) {
				return image.Pt(480+x, 240+y), slot - 1
			}
		}
	}
	t.Fatal("production doll has no occupied pixel in the requested region")
	return image.Point{}, 0
}

// TestReleaseShopDollHitUsesOnlyPaintedFigurePixels is the EN/RU App-level
// draw/hit agreement witness. The shipped Danath mask has occupied pixels in
// source rows 200..239 and in rows 170..199, where the removed doll-name
// overlay used to cover equipment. Both populations are visible and interactive.
func TestReleaseShopDollHitUsesOnlyPaintedFigurePixels(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: production doll-mask witness needs a lawful install")
	}
	t.Run("complete lower rows", func(t *testing.T) {
		app, s := releaseShopApp(t)
		p, slot := releaseShopMaskPoint(t, s, func(v ui.ShopScreenView, frame, _ *image.RGBA, x, y int) bool {
			return y >= 200 && x >= 33 && x < 119 && v.Figure != nil && v.Figure.RGBAAt(x, y).A == 255 &&
				frame.RGBAAt(480+x, 240+y) == v.Figure.RGBAAt(x, y)
		})
		before := *s.shopWornSlots(s.shopMemberIndex())
		if before[slot] == 0 {
			t.Fatalf("mask pixel %v names empty slot %d", p, slot+1)
		}
		if err := app.HeadlessPointer("press", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessPointer("release", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		after := *s.shopWornSlots(s.shopMemberIndex())
		want := before
		want[slot] = 0
		if after != want {
			t.Fatalf("App did not unequip exactly the visible lower-row slot at %v (slot %d): before %v, after %v", p, slot+1, before, after)
		}
	})

	t.Run("former member-name rows", func(t *testing.T) {
		app, s := releaseShopApp(t)
		p, slot := releaseShopMaskPoint(t, s, func(v ui.ShopScreenView, frame, plain *image.RGBA, x, y int) bool {
			return x >= 33 && x < 127 && y >= 170 && y < 200 && v.Figure != nil &&
				frame.RGBAAt(480+x, 240+y) == plain.RGBAAt(480+x, 240+y)
		})
		before := *s.shopWornSlots(s.shopMemberIndex())
		if before[slot] == 0 {
			t.Fatalf("former name-row point %v names empty slot %d", p, slot+1)
		}
		if err := app.HeadlessPointer("press", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessPointer("release", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		after := *s.shopWornSlots(s.shopMemberIndex())
		if after[slot] != 0 {
			t.Fatalf("App did not route former name-row point %v to occupied slot %d: before=%v after=%v", p, slot+1, before, after)
		}
	})
}

// TestReleaseSharedTownRoute runs only when a lawful install is named. It uses
// the production campaign, definition table, screen seams and save envelope;
// the asset-free gate skips it rather than substituting synthetic content.
func TestReleaseSharedTownRoute(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: shared-town release route needs a lawful install")
	}
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil || f.TownSchoolArt.Value().Masks[0] == nil || f.TownSchoolArt.Value().Masks[1] == nil ||
		f.TownSchoolArt.Value().Skills[0][4][2] == nil || f.TownSchoolArt.Value().Skills[1][4][2] == nil {
		t.Fatalf("production school art did not resolve: %v", f.TownSchoolArt.Err())
	}
	if f.TownTavernArt.Value() == nil || f.TownTavernArt.Value().LeftPicture == nil || f.TownTavernArt.Value().LeftStats == nil ||
		f.TownTavernArt.Value().Center == nil || f.TownTavernArt.Value().Units[1] == nil || f.TownTavernArt.Value().Units[15] == nil {
		t.Fatalf("production tavern art did not resolve: %v", f.TownTavernArt.Err())
	}
	target := releaseMercenaryChapter(t, f.Campaign.Value())
	town := NewTown(f.Campaign.Value())
	for mission := range f.Campaign.Value().Chapters {
		if mission < target {
			town.Won(mission)
		}
	}
	if got := town.Chapter(); got != target {
		t.Fatalf("prepared chapter = %d, want %d", got, target)
	}
	town.gold = 1_000_000
	f.Town = town
	f.Carried = f.NextParty()
	f.arriveInTown()
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	s.composeShopFaces()
	if surface := s.TownSurface(); len(surface.Buttons) != 4 ||
		surface.Buttons[tavernButtonSleep].Label != f.Words.TavernSleep || !surface.Buttons[tavernButtonSleep].Enabled {
		t.Fatalf("production tavern buttons = %+v, want enabled Sleep above Hire/Talk/Exit", surface.Buttons)
	}
	if act := s.townSurfaceButton(tavernButtonSleep); act.Msg != "" || f.Shop.restocks != 1 {
		t.Fatalf("production Sleep = %q ordinal %d", act.Msg, f.Shop.restocks)
	}
	npcDialogue := false
	for _, offer := range f.Town.Offers(TownTavern) {
		s.tavernSelection = tavernCandidateKey{kind: tavernCandidateOffer, id: offer.Index}
		inspection := s.TownSurface()
		if inspection.Candidate.HasSubject || inspection.Candidate.Figure == nil || inspection.CandidatePixels == nil {
			t.Fatalf("shipped talk-only candidate %d inspection = subject %v figure %p pixels %p; want a real figure without invented stats",
				offer.Index, inspection.Candidate.HasSubject, inspection.Candidate.Figure, inspection.CandidatePixels)
		}
		if !s.openTownDialogue(TownTavern, offer, offer.NPC) {
			continue
		}
		if pic, ok := s.TownDialogue(); !ok || pic == nil {
			t.Fatal("shipped tavern dialogue opened without a rendered page")
		}
		for n := 0; s.room == roomTalk && n < 32; n++ {
			s.AdvanceTownDialogue()
		}
		if s.room != roomTavern {
			t.Fatal("shipped tavern dialogue did not return within 32 pages")
		}
		if !s.openTownDialogue(TownTavern, offer, offer.NPC) {
			t.Fatal("shipped tavern dialogue was not repeatable")
		}
		s.Back()
		for s.room == roomTalk {
			s.Back()
		}
		npcDialogue = true
		break
	}
	if !npcDialogue {
		t.Fatalf("chapter %d has no readable shipped tavern dialogue", target)
	}

	mercs := s.tavernMercenaries()
	if len(mercs) == 0 {
		t.Fatalf("chapter %d resolved no unlocked stocked mercenary", target)
	}
	s.tavernSelection = tavernCandidateKey{kind: tavernCandidateMercenary, id: mercs[0].Type}
	if act := s.townSurfaceButton(tavernButtonTalk); act.Msg != "" || s.room != roomTalk {
		t.Fatalf("candidate Talk did not open a modal: room %d msg %q", s.room, act.Msg)
	}
	if pic, ok := s.TownDialogue(); !ok || pic == nil {
		t.Fatal("candidate Talk has no production dialogue surface")
	}
	for s.room == roomTalk {
		s.AdvanceTownDialogue()
	}
	if s.room != roomTavern {
		t.Fatal("candidate Talk did not return to the tavern")
	}
	if act := s.townSurfaceButton(tavernButtonTalk); act.Msg != "" || s.room != roomTalk {
		t.Fatal("candidate Talk was not repeatable")
	}
	for s.room == roomTalk {
		s.AdvanceTownDialogue()
	}

	// Keep expected install bytes across the save/restore below, rather than
	// accepting a restored model's possibly reset word set as its own oracle.
	rawCaptions := roomCaptionRaw(t, f, "main/text/main.txt")
	wantHire := roomCaptionRawLine(t, rawCaptions, 258)
	wantFire := roomCaptionRawLine(t, rawCaptions, 259)
	if button := s.TownSurface().Buttons[tavernButtonHire]; button.Label != wantHire || !button.Enabled {
		t.Fatalf("available production squad button = %+v", button)
	}
	if act := s.townSurfaceButton(tavernButtonHire); act.Msg == "" {
		t.Fatalf("hire type %d returned no message", mercs[0].Type)
	}
	if button := s.TownSurface().Buttons[tavernButtonHire]; button.Label != wantFire || !button.Enabled {
		t.Fatalf("hired production squad button = %+v, want enabled Fire", button)
	}
	if len(f.Carried) <= 1 {
		t.Fatalf("whole-squad hire left party at %d member(s)", len(f.Carried))
	}
	snap, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, inTown, err := f.Restore(snap); err != nil || !inTown {
		t.Fatalf("restore hired town = %v, %v", inTown, err)
	}
	s = f.TownScreen().(*townScreen)
	if !f.Town.MercenaryHired(mercs[0].Type) || len(f.Carried) <= 1 {
		t.Fatal("save/load lost the hired squad")
	}
	s.room = roomTavern
	s.tavernSelection = tavernCandidateKey{kind: tavernCandidateMercenary, id: mercs[0].Type}
	if button := s.TownSurface().Buttons[tavernButtonHire]; button.Label != wantFire || !button.Enabled {
		t.Fatalf("restored hired production squad button = %+v, want enabled Fire", button)
	}

	// Shop migration: a live table survives Book and a member change, then the
	// established EXIT path clears it.
	s.room = roomShop
	s.composeShopFaces()
	placed := false
	for i, room := range shopRoomShelves {
		if !room.stocked || len(f.Shop.Shelf(room.shelf)) == 0 {
			continue
		}
		s.chooseRoomShelf(i)
		if act := s.shopClickShelfCell(0, false); act.Msg == "" && len(f.Shop.Table()) > 0 {
			placed = true
			break
		}
	}
	if !placed {
		t.Fatal("production shop offered no item that could reach its table")
	}
	before := append([]ShopPlace(nil), f.Shop.Table()...)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlBook})
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPickerNext})
	if !s.ShopScreen().Book || !reflect.DeepEqual(before, f.Shop.Table()) {
		t.Fatal("Book/member change moved the production trade table")
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 3})
	if s.room != roomSquare || len(f.Shop.Table()) != 0 {
		t.Fatal("shop EXIT did not clear and leave")
	}

	// Return is the same immediate toggle after the save boundary.
	s.room = roomTavern
	if msg, ok := s.toggleMercenary(mercs[0].Type); !ok {
		t.Fatalf("return type %d: %s", mercs[0].Type, msg)
	}
	if len(f.Carried) != 1 {
		t.Fatalf("return left %d members, want the player hero", len(f.Carried))
	}

	// School purchase and its second save/load boundary.
	s.room, s.schoolCell = roomSchool, 0
	if f.Carried[0].Mage {
		s.schoolCell = 5
	}
	slot, price, ok := s.selectedSchoolSlot()
	if !ok {
		t.Fatal("production hero has no selectable class skill")
	}
	level := f.Carried[0].Hero.Skill[slot]
	gold := f.Town.Gold()
	s.trainHeroSkill(slot)
	if f.Carried[0].Hero.Skill[slot] != level+1 || f.Town.Gold() != gold-price {
		t.Fatal("production Train did not mutate exactly one level and the purse")
	}
	snap, _, err = f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.Restore(snap); err != nil {
		t.Fatal(err)
	}
	if f.Carried[0].Hero.Skill[slot] != level+1 || f.Town.Gold() != gold-price {
		t.Fatal("production save/load lost Train")
	}

	// A shipped school mission conversation remains reachable after the room
	// has become a purchase surface.
	dialogueSeen := false
	for _, target := range f.Campaign.Value().Main {
		candidate := NewTown(f.Campaign.Value())
		for mission := range f.Campaign.Value().Chapters {
			if mission < target {
				candidate.Won(mission)
			}
		}
		candidate.Arrive()
		if candidate.Chapter() != target || len(candidate.Offers(TownSchool)) == 0 {
			continue
		}
		f.Town = candidate
		f.townUI = nil
		school := f.TownScreen().(*townScreen)
		school.room = roomSchool
		if act := school.openBuildingDialogue(TownSchool, 1); act.Msg != "" || school.room != roomTalk {
			continue
		}
		if pic, ok := school.TownDialogue(); !ok || pic == nil {
			t.Fatalf("chapter %d school dialogue opened without a page", target)
		}
		dialogueSeen = true
		break
	}
	if !dialogueSeen {
		t.Fatal("no shipped school mission dialogue was reachable")
	}

	// The two siege mercenary types use their exact Units rows at mission
	// minting time, including the row's secondary key.
	for typ := 1; typ <= 2; typ++ {
		members, ok := buildSiegeSquad(f.Table, typ, 1)
		if !ok || len(members) != 1 {
			t.Fatalf("production siege type %d did not resolve", typ)
		}
		party := append(mapload.CloneParty(f.NextParty()), members...)
		if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(10, party)(); err != nil {
			t.Fatalf("production siege type %d could not enter mission 10: %v", typ, err)
		}
	}

	// A fresh production application traverses unchanged pre-create and the
	// migrated detailed stage to a valid Accept.
	cf := releaseFront(t)
	app := cf.App("1006-release-chargen")
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() == ui.ScreenPicker {
		if err := app.HeadlessActivate("@first"); err != nil {
			t.Fatal(err)
		}
	}
	if err := headlessCreateCharacter(app, HeadlessCharacter{Name: "Self"}); err == nil {
		t.Fatal("production detailed chargen accepted a reserved name")
	}
	if app.Screen() != ui.ScreenChargen {
		t.Fatalf("invalid Accept ended on %s, want chargen", app.Screen())
	}
	state, ok := app.HeadlessChargenState()
	if !ok || state.Stage != ui.ChargenStageDetailed {
		t.Fatalf("invalid Accept left unusable state %#v, %v", state, ok)
	}
	if err := headlessChargenPress(app, &state, ui.ChargenControlBack, ""); err != nil {
		t.Fatalf("Back after invalid Accept: %v", err)
	}
	if err := headlessChargenFocus(app, &state, ui.ChargenControlName, ""); err != nil {
		t.Fatalf("focus name after invalid Accept: %v", err)
	}
	for range len(state.Name) {
		if err := app.HeadlessType("", true); err != nil {
			t.Fatalf("clear invalid name: %v", err)
		}
	}
	if err := headlessCreateCharacter(app, HeadlessCharacter{Name: "Witness"}); err != nil {
		t.Fatalf("production detailed chargen: %v", err)
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatalf("Accept ended on %s, want map", app.Screen())
	}
}

func releaseMercenaryChapter(t *testing.T, c Campaign) int {
	t.Helper()
	for _, target := range c.Main {
		if !c.offers(target) {
			continue
		}
		unlocked := make(map[int]bool)
		for mission, ch := range c.Chapters {
			if mission >= target {
				continue
			}
			for _, typ := range ch.EnableMercenary {
				unlocked[typ] = true
			}
		}
		for _, typ := range c.Chapters[target].Mercenaries {
			if unlocked[typ] && typ > 0 && typ <= len(c.MercenaryCount) && c.MercenaryCount[typ-1] > 0 {
				return target
			}
		}
	}
	t.Fatal("campaign has no chapter with a previously unlocked stocked mercenary")
	return 0
}
