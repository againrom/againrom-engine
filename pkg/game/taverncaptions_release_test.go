package game

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// TestReleaseTavernCaptionsMatchInstalledSelectionAndPixels checks the
// TOWN-391/392 source selection through the real tavern model. The four-button
// shop-derived geometry and Sleep remain owner policy, not original-frame
// equivalence. Expected words, font and bitmaps use independent raw readers.
func TestReleaseTavernCaptionsMatchInstalledSelectionAndPixels(t *testing.T) {
	f := releaseFront(t)
	raw := roomCaptionRaw(t, f, "main/text/main.txt")
	hire, fire := roomCaptionRawLine(t, raw, 258), roomCaptionRawLine(t, raw, 259)
	talk, exit := roomCaptionRawLine(t, raw, 242), roomCaptionRawLine(t, raw, 232)
	constructor := roomCaptionRawLine(t, raw, 243)
	if constructor == hire || constructor == fire || hire == fire {
		t.Fatal("installed constructor, Hire and Fire captions must be distinct")
	}
	font := roomCaptionRawFont(t, f)
	wells := [4]image.Rectangle{
		image.Rect(494, 15, 614, 67), image.Rect(483, 67, 623, 113),
		image.Rect(483, 114, 623, 160), image.Rect(494, 160, 614, 212),
	}
	key := func(picture *image.RGBA) *image.RGBA {
		for y := picture.Rect.Min.Y; y < picture.Rect.Max.Y; y++ {
			for x := picture.Rect.Min.X; x < picture.Rect.Max.X; x++ {
				p := picture.RGBAAt(x, y)
				if p.R == 0 && p.G == 0 && p.B == 0 {
					picture.SetRGBA(x, y, color.RGBA{})
				}
			}
		}
		return picture
	}
	panel := key(roomCaptionRawBMP(t, f, "graphics/interface/shopmenu.bmp"))
	if panel.Bounds().Size() != image.Pt(176, 238) {
		t.Fatalf("raw shopmenu size %v, want 176x238", panel.Bounds().Size())
	}
	// The panel body carries the plaques at rest; each native ShopButton
	// bitmap is drawn only while its button is held (TOWN-260).
	var pressedArt [4]image.Image
	for i := range pressedArt {
		pic := key(roomCaptionRawBMP(t, f, fmt.Sprintf("graphics/interface/shopbutton%d.bmp", i+1)))
		if pic.Bounds().Size() != ui.ShopButtonRect(i).Size() {
			t.Fatalf("raw shopbutton%d size %v, want %v", i+1, pic.Bounds().Size(), ui.ShopButtonRect(i).Size())
		}
		pressedArt[i] = pic
	}

	f.Carried = f.NextParty()
	f.Town.Won(10)
	f.Town.Won(20)
	f.arriveInTown()
	const purse, typ = 1000000, 14
	f.Town.gold = purse
	if f.Town.Chapter() != 30 || len(f.Campaign.Value().MercenaryCount) < typ {
		t.Fatal("the first town chapter lacks the type-14 source population")
	}
	count := f.Campaign.Value().MercenaryCount[typ-1]
	terms, ok := f.Table.NPC.Mercenary(typ)
	if !ok || count <= 0 {
		t.Fatal("the installed type-14 squad has no count or price terms")
	}
	// Independent input arithmetic: the first town chapter's unit factor is
	// 10. Neither the candidate quote nor the production price helper is read.
	price := (int(terms.PriceA) + count*int(terms.PriceB)) * 10
	if price <= 0 || price >= purse {
		t.Fatalf("type-14 source price %d cannot exercise both affordability arms", price)
	}
	s := f.TownScreen().(*townScreen)
	s.Choose(0)
	for i := 0; s.room == roomTalk && i < 64; i++ {
		s.AdvanceTownDialogue()
	}
	if s.room != roomTavern {
		t.Fatal("production town entry did not reach the tavern")
	}
	s.CloseTip()
	mercenaryCell, talkCell := -1, -1
	for i, candidate := range s.tavernCandidates() {
		if candidate.key == (tavernCandidateKey{kind: tavernCandidateMercenary, id: typ}) {
			mercenaryCell = i
		}
		if candidate.key.kind == tavernCandidateOffer && talkCell < 0 {
			talkCell = i
		}
	}
	if mercenaryCell < 0 {
		t.Fatal("type-14 squad never reached the live first-chapter candidate stream")
	}

	states, comparisons, negativeControls := 0, 0, 0
	check := func(name, caption, value string, enabled, selected bool, gold int) {
		t.Helper()
		v := s.TownSurface()
		want := [4]ui.TownSurfaceButton{
			{Label: f.Words.TavernSleep, Enabled: true},
			{Label: caption, Value: value, Enabled: enabled},
			{Label: talk, Enabled: selected},
			{Label: exit, Value: ui.GroupDigits(int64(gold)), Enabled: true},
		}
		if len(v.Buttons) != 4 {
			t.Fatalf("%s: production tavern has %d buttons", name, len(v.Buttons))
		}
		for i := range want {
			if v.Buttons[i] != want[i] {
				t.Fatalf("%s button %d: got %+v, want independent %+v", name, i, v.Buttons[i], want[i])
			}
		}
		if f.Town.Gold() != gold {
			t.Fatalf("%s: live purse %d, want %d", name, f.Town.Gold(), gold)
		}
		var expected [4]*image.RGBA
		for i, well := range wells {
			expected[i] = oracleTownButtonWell(panel, image.Pt(464, 0), nil, well, font, want[i], false)
		}
		// A held enabled button draws its native bitmap and moves both text
		// rows one pixel down. Caption sources stay unchanged.
		for press := -1; press < 4; press++ {
			actual := v
			if press >= 0 {
				actual.Press = ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: press}
			}
			pixels := ui.ComposeTownSurface(actual)
			for i, well := range wells {
				var art image.Image
				if press == i && want[i].Enabled {
					art = pressedArt[i]
				}
				pressedWell := oracleTownButtonWell(panel, image.Pt(464, 0), art, well, font, want[i], press == i)
				if n, points := diffWells(pixels, pressedWell, well); n != 0 {
					t.Fatalf("%s press %d button %d: %d pixel differences, first %v", name, press, i, n, points)
				}
				comparisons++
			}
		}
		// Mutations are confined to copies of the view. The constructor's
		// combined caption is explicitly rejected in both active hire states.
		for _, defect := range []string{"constructor-243", "wrong-hire-state", "english", "no-caption", "wrong-selector"} {
			if defect == "wrong-selector" && font.Selector != 1 {
				continue
			}
			bad := v
			bad.Buttons = append([]ui.TownSurfaceButton(nil), v.Buttons...)
			switch defect {
			case "constructor-243":
				bad.Buttons[1].Label = constructor
			case "wrong-hire-state":
				bad.Buttons[1].Label = hire
				if caption == hire {
					bad.Buttons[1].Label = fire
				}
			case "english":
				if caption != "" {
					bad.Buttons[1].Label = "Hire"
					if caption == fire {
						bad.Buttons[1].Label = "Fire"
					}
				}
				bad.Buttons[2].Label, bad.Buttons[3].Label = "Talk", "EXIT"
			case "no-caption":
				bad.Buttons[2].Label = ""
			case "wrong-selector":
				wrongFont := *v.Font
				wrongFont.Selector = 0
				bad.Font = &wrongFont
			}
			pixels := ui.ComposeTownSurface(bad)
			differences := 0
			for i, well := range wells {
				n, _ := diffWells(pixels, expected[i], well)
				differences += n
			}
			if differences == 0 {
				t.Fatalf("%s: oracle failed to reject %s", name, defect)
			}
			negativeControls++
		}
		states++
	}

	// Entry stores position 0 when the roster has a mercenary (TOWN-468), and
	// the type-14 squad is the chapter's only mercenary cell.
	if mercenaryCell != 0 {
		t.Fatalf("type-14 squad sits at position %d, want 0", mercenaryCell)
	}
	check("entry", hire, ui.GroupDigits(int64(price)), true, true, purse)
	if talkCell >= 0 {
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: talkCell}, false)
		check("talk-only", "", "", false, true, purse)
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: mercenaryCell}, false)
	check("available", hire, ui.GroupDigits(int64(price)), true, true, purse)
	partyCount := len(f.Carried)
	f.Town.gold = price - 1
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 1}, false)
	check("unaffordable-refusal", hire, ui.GroupDigits(int64(price)), false, true, price-1)
	if f.Town.MercenaryHired(typ) || len(f.Carried) != partyCount || f.Town.MercenaryPool(typ) != count {
		t.Fatal("unaffordable Hire changed the squad or party")
	}
	f.Town.gold = purse
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 1}, false)
	check("hired", fire, ui.GroupDigits(int64(price)), true, true, purse-price)
	if !f.Town.MercenaryHired(typ) || len(f.Carried) != partyCount+count || f.Town.MercenaryPool(typ) != count {
		t.Fatal("Hire did not move the whole installed squad into the party")
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 1}, false)
	check("returned", hire, ui.GroupDigits(int64(price)), true, true, purse)
	if f.Town.MercenaryHired(typ) || len(f.Carried) != partyCount || f.Town.MercenaryPool(typ) != count {
		t.Fatal("Fire did not return the whole installed squad")
	}
	beforeParty, beforeSelection := mapload.OwnParty(f.Carried), s.tavernSelection
	restocks := f.Shop.restocks
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false)
	check("after-sleep", hire, ui.GroupDigits(int64(price)), true, true, purse)
	if f.Shop.restocks != restocks+1 || !reflect.DeepEqual(f.Carried, beforeParty) ||
		s.tavernSelection != beforeSelection || s.room != roomTavern || f.Town.Chapter() != 30 {
		t.Fatal("Sleep changed its restock-only action or the selected tavern session")
	}
	t.Logf("selector %d: type %d count %d price %d; %d states, %d full-well comparisons, %d rejected caption controls; Hire/Fire/refusal/Sleep preserved", font.Selector, typ, count, price, states, comparisons, negativeControls)
}
