package game

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// dollHoverFrame writes the composed frame under AGAINROM_DOLLHOVER_DIR when
// that variable names a directory, for a person to read.
func dollHoverFrame(t *testing.T, name string, frame *image.RGBA) {
	t.Helper()
	dir := os.Getenv("AGAINROM_DOLLHOVER_DIR")
	if dir == "" || frame == nil {
		return
	}
	file, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, frame); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

// dollHoverPixel is the middle pixel, in raster order, that the mask marks
// with slot.
func dollHoverPixel(mask *ui.SlotMask, slot int) (image.Point, bool) {
	var marked []image.Point
	for y := 0; y < mask.H; y++ {
		for x := 0; x < mask.W; x++ {
			if int(mask.Slot[y*mask.W+x]) == slot {
				marked = append(marked, image.Pt(x, y))
			}
		}
	}
	if len(marked) == 0 {
		return image.Point{}, false
	}
	return marked[len(marked)/2], true
}

// TestReleaseGeneratorDollHoverShowsWornItemInformation hovers every worn item
// of the detailed generator's doll for each of the four heroes and requires
// the delayed hover to carry that item's information.
func TestReleaseGeneratorDollHoverShowsWornItemInformation(t *testing.T) {
	f := releaseFront(t)
	if f.ChargenAssets == nil {
		t.Skip("the install ships no character generation")
	}
	l := f.generator()
	heroes := len(l.PreCreate.Heroes)
	for hero := 0; hero < heroes; hero++ {
		f := releaseFront(t)
		f.Options = OptionsStore{}
		f.MusicPlayer = &secondTestMusic{}
		f.SetDeterministicFrames(true)
		app := f.App("doll hover")
		app.Layout(640, 480)
		var c *ui.Chargen
		app.SetNewGameChargen(func() *ui.ChargenEntry {
			c = ui.NewChargen(f.ChargenSetup())
			return &ui.ChargenEntry{Model: c, Begin: f.NewGameBegin(f.Base().Profile.Mission())}
		})
		if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenChargen {
			t.Fatalf("NEW GAME: screen %s, error %v", app.Screen(), err)
		}
		pre := l.PreCreate
		masks := []byte{byte(pre.Heroes[hero].Mask), byte(pre.Forward.Mask)}
		at := chargenTraceMaskPoints(t, f, pre.Mask.Key, image.Point{}, masks)
		secondGeneratorClick(t, app, at[masks[0]])
		secondGeneratorClick(t, app, at[masks[1]])
		if secondGeneratorStage(app) != ui.ChargenStageDetailed {
			t.Fatalf("hero %d: OK did not open the detail page", hero)
		}
		res, ok := c.Result()
		if !ok {
			t.Fatalf("hero %d: no result", hero)
		}
		preview := f.ChargenPreview(res)
		if preview.Doll == nil || preview.Worn == nil || preview.Worn.Mask == nil {
			t.Fatalf("hero %d: the preview has no doll hover data", hero)
		}
		member := f.ChargenParty(res)[0]
		items := mapload.MemberItemEquipment(member, f.Table)
		d := l.Detail
		origin := d.Doll.Rect.Rectangle().Min.Add(d.Preview.FigureOffset.Pt())
		worn := 0
		for slot := 1; slot <= len(items); slot++ {
			if items[slot-1].Empty() {
				continue
			}
			want := itemInstanceInfoLines(items[slot-1], f.Table, f.Words)
			if got := preview.Worn.Info[slot-1]; !reflect.DeepEqual(got, want) {
				t.Errorf("hero %d slot %d information %q, want the item's lines %q", hero, slot, got, want)
			}
			px, ok := dollHoverPixel(preview.Worn.Mask, slot)
			if !ok {
				continue
			}
			p := origin.Add(px)
			frame, state := generatorWitnessHover(t, app, p)
			wantTarget := fmt.Sprintf("chargen-item/%d/%d", c.Stage(), slot)
			if !state.Visible || state.Target != wantTarget {
				t.Errorf("hero %d slot %d at %v: hover visible %v target %q, want %q", hero, slot, p, state.Visible, state.Target, wantTarget)
				continue
			}
			worn++
			if worn == 1 {
				dollHoverFrame(t, fmt.Sprintf("generator-%s-hero%d-slot%d", f.Base().Profile.Language, hero, slot), frame)
			}
		}
		if worn == 0 {
			t.Errorf("hero %d wears no item the doll shows", hero)
		}
		for i, slot := range preview.Worn.Mask.Slot {
			if slot != 0 {
				continue
			}
			blank := origin.Add(image.Pt(i%preview.Worn.Mask.W, i/preview.Worn.Mask.W))
			if _, state := generatorWitnessHover(t, app, blank); strings.HasPrefix(state.Target, "chargen-item") {
				t.Errorf("hero %d: a blank doll pixel %v shows item information", hero, blank)
			}
			break
		}
	}
}

// TestReleaseTownHeroDollHoverShowsWornItemInformation puts a fighter and a
// mage on the tavern and school panes and requires the hover over each worn
// item's doll pixel to carry the item's information, the lines the shop shows.
func TestReleaseTownHeroDollHoverShowsWornItemInformation(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	if f.ChargenAssets == nil {
		t.Skip("the install ships no character generation")
	}
	for _, class := range []int{0, 1} {
		res := ui.ChargenResult{Name: "Doll hover", Choices: []int{0, class, 3}, Stats: []int{31, 27, 24, 29}}
		party := f.ChargenParty(res)
		f.Carried = party
		f.Town = NewTown(f.Campaign.Value())
		for _, room := range []townRoom{roomTavern, roomSchool} {
			s := f.TownScreen().(*townScreen)
			s.room = room
			s.composeShopFaces()
			before := mapload.CloneParty(f.Carried)
			surface := s.TownSurface()
			if !reflect.DeepEqual(before, f.Carried) {
				t.Errorf("class %d room %v: composing the surface changed the party", class, room)
			}
			if surface.HeroSlotMask == nil || !surface.Hero.HasSubject {
				t.Fatalf("class %d room %v: no hero doll on the surface", class, room)
			}
			// The shop's own view of the same member, built after the surface
			// because the shop's paint fills the member's worn items.
			shop := s.ShopScreen().SlotInfo
			worn := 0
			for slot := 1; slot <= len(shop); slot++ {
				want := shop[slot-1]
				if got := surface.HeroSlotInfo[slot-1]; !reflect.DeepEqual(got, want) {
					t.Errorf("class %d room %v slot %d information %q, want the shop's lines %q", class, room, slot, got, want)
				}
				if len(want) == 0 {
					continue
				}
				px, ok := dollHoverPixel(surface.HeroSlotMask, slot)
				if !ok {
					continue
				}
				p := image.Pt(480, 240).Add(px)
				if got, ok := ui.TownHeroHoverLines(surface, p); !ok || !reflect.DeepEqual(got, want) {
					t.Errorf("class %d room %v slot %d at %v: hover %q, %v; want %q", class, room, slot, p, got, ok, want)
					continue
				}
				worn++
			}
			if worn == 0 {
				t.Errorf("class %d room %v: no worn item reached the hover", class, room)
			}
		}
	}
}

// TestReleaseTownHeroDollHoverThroughAppFrame hovers a worn item on the school
// pane of a loaded town through the application and requires the hover to
// carry that item. It needs a town SAV named by AGAINROM_CITY_SAV.
func TestReleaseTownHeroDollHoverThroughAppFrame(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	path := os.Getenv("AGAINROM_CITY_SAV")
	if path == "" {
		t.Skip("AGAINROM_CITY_SAV is not set: the school pane needs an existing town SAV")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, app := campaignCold(t, serviceStore(t, raw))
	app.Layout(640, 480)
	s := serviceOpen(t, f, app, "SCHOOL", roomSchool)
	surface := s.TownSurface()
	if surface.HeroSlotMask == nil || surface.Hero.Statistics {
		t.Fatal("the school pane does not show the hero doll")
	}
	shown := 0
	for slot := 1; slot <= len(surface.HeroSlotInfo); slot++ {
		if len(surface.HeroSlotInfo[slot-1]) == 0 {
			continue
		}
		px, ok := dollHoverPixel(surface.HeroSlotMask, slot)
		if !ok {
			continue
		}
		p := image.Pt(480, 240).Add(px)
		if _, ok := ui.TownHeroHoverLines(surface, p); !ok {
			continue
		}
		if err := app.HeadlessPointer("move", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 40; i++ {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		state, _ := app.HeadlessTooltip()
		if !state.Visible || !strings.HasPrefix(state.Target, "hero-item/") {
			t.Errorf("slot %d at %v: hover visible %v target %q", slot, p, state.Visible, state.Target)
			continue
		}
		shown++
		if shown == 1 {
			frame, _, err := app.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			dollHoverFrame(t, fmt.Sprintf("school-%s-slot%d", f.Base().Profile.Language, slot), frame)
		}
	}
	if shown == 0 {
		t.Fatal("no worn item of the school pane reached the hover")
	}
}
