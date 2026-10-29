package game

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The first game's tavern, shop and school carry the town's pane; the shop's
// has the book.
func TestFirstTownRoomsCarryTheHeroPane(t *testing.T) {
	d := ROM1TownDescription()
	want := town.Rect{480, 238, 640, 480}
	if got := want.Rectangle(); got != ui.TownCharacterRegion {
		t.Fatalf("the data's pane region %v is not the town's %v", got, ui.TownCharacterRegion)
	}
	for _, room := range []string{"tavern", "shop", "school"} {
		r := d.Room(room)
		if r == nil || r.Pane == nil || r.Pane.Rect != want || r.Pane.Book != (room == "shop") ||
			r.Pane.BookOffset != (town.Point{}) || r.Pane.Font != town.PaneFontInstall {
			t.Errorf("%s pane = %+v", room, r)
		}
	}
}

// A rectangle or book flag changed in the data moves the pane in both games.
func TestHeroPaneFollowsTheRoomData(t *testing.T) {
	_, first, _ := sourceCityEquipment(t)
	_, _, second := secondCampaignFixtureFiles(t, false)
	second.composed = &townSceneState{}
	for name, screen := range map[string]*townScreen{"first": first, "second": second} {
		at := &town.PaneSpec{Rect: town.Rect{480, 238, 640, 480}}
		moved := &town.PaneSpec{Rect: town.Rect{0, 0, 160, 242}, Book: true}
		a, _, _ := screen.heroPane(at)
		b, _, _ := screen.heroPane(moved)
		if a.PaneRect != at.Rect.Rectangle() || b.PaneRect != moved.Rect.Rectangle() {
			t.Errorf("%s: pane rectangles %v and %v, want the data's %v and %v", name, a.PaneRect, b.PaneRect,
				at.Rect.Rectangle(), moved.Rect.Rectangle())
		}
		if !a.NoBook || b.NoBook {
			t.Errorf("%s: book corners %v and %v, want none and one", name, !a.NoBook, !b.NoBook)
		}
		if !a.HasSubject || !b.HasSubject {
			t.Errorf("%s: the pane shows no party member", name)
		}
		pa, pb := image.NewRGBA(image.Rect(0, 0, 640, 480)), image.NewRGBA(image.Rect(0, 0, 640, 480))
		ui.DrawTownCharacterRegion(pa, a)
		ui.DrawTownCharacterRegion(pb, b)
		if bytes.Equal(pa.Pix, pb.Pix) {
			t.Errorf("%s: the moved pane draws the same frame", name)
		}
	}
}

// With no party member either game's pane shows nobody.
func TestHeroPaneWithNoPartyMember(t *testing.T) {
	_, first, _ := sourceCityEquipment(t)
	_, _, second := secondCampaignFixtureFiles(t, false)
	second.composed = &townSceneState{}
	for name, screen := range map[string]*townScreen{"first": first, "second": second} {
		screen.shop.fresh = nil
		screen.sess.Carried = nil
		v, mask, _ := screen.heroPane(&town.PaneSpec{Rect: town.Rect{480, 238, 640, 480}, Book: true})
		if v.HasSubject || v.Figure != nil || mask != nil || v.MemberCount != 0 || v.Member != 0 {
			t.Errorf("%s: has %v figure %v mask %v member %d/%d", name, v.HasSubject, v.Figure != nil, mask != nil, v.Member, v.MemberCount)
		}
	}
}

// The first game's rooms read their pane from the room data.
func TestFirstTownPaneIsTheRoomData(t *testing.T) {
	_, screen, _ := sourceCityEquipment(t)
	screen.shopBook = true
	for _, room := range []string{"tavern", "shop", "school"} {
		v, _, _ := screen.heroPane(screen.chapterPane(room))
		book := room == "shop"
		if v.NoBook == book || v.BookOpen != book || v.PaneRect != ui.TownCharacterRegion {
			t.Errorf("%s: no book %v open %v rect %v", room, v.NoBook, v.BookOpen, v.PaneRect)
		}
	}
}

// modTownPane sets the pane record of room in a description.
func modTownPane(room string, rect []any, book bool) func(d map[string]any) {
	return func(d map[string]any) {
		editRoom(d, room, func(r map[string]any) { r["pane"] = map[string]any{"rect": rect, "book": book} })
	}
}

// A first-game mod town's rooms with no pane record take the shipped rooms'
// panes: the shop draws its book corner closed and open and answers the book
// control. A shop pane with "book": false has no book control. A moved pane,
// or a book on the tavern or school, is refused where the mod layers it.
func TestModTownShopPaneFollowsItsRecord(t *testing.T) {
	region := []any{480, 238, 640, 480}
	green, red := color.RGBA{0, 255, 0, 255}, color.RGBA{255, 0, 0, 255}
	book := ui.CharacterPaneCornerRect(ui.TownCharacterRegion, ui.CharacterPaneBook)
	mid := image.Pt((book.Min.X+book.Max.X)/2, (book.Min.Y+book.Max.Y)/2)
	corner := func(v ui.TownCharacterView) color.RGBA {
		v.CornerArt = &ui.CharacterPaneCornerArt{BookOpened: image.NewUniform(red), BookClosed: image.NewUniform(green)}
		dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
		ui.DrawTownCharacterRegion(dst, v)
		return dst.RGBAAt(mid.X, mid.Y)
	}
	shipped := ROM1TownDescription()
	for _, c := range []struct {
		name string
		edit func(d map[string]any)
		book bool
	}{
		{"no pane record", nil, true},
		{"book", modTownPane("shop", region, true), true},
		{"no book", modTownPane("shop", region, false), false},
	} {
		f, _, err := layeredFront(t, minimalEastMods(t, c.edit), "east-town")
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if err := checkEastTown(f, false); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		s := f.TownScreen().(*townScreen)
		if c.edit == nil {
			for _, room := range []string{"tavern", "shop", "school"} {
				if got, want := s.chapterPane(room), shipped.Room(room).Pane; got == nil || !reflect.DeepEqual(*got, *want) {
					t.Errorf("%s: %s pane %+v, want the shipped %+v", c.name, room, got, want)
				}
			}
		}
		s.Choose(1)
		v := s.ShopScreen()
		if s.room != roomShop || !v.Character.HasSubject || v.Character.NoBook == c.book || v.Character.PaneRect != ui.TownCharacterRegion {
			t.Fatalf("%s: room %v subject %v no book %v rect %v", c.name, s.room, v.Character.HasSubject, v.Character.NoBook, v.Character.PaneRect)
		}
		if hit, ok := ui.ShopControlAt(mid); !ok || hit.Kind != ui.ShopControlBook {
			t.Fatalf("%s: the book corner's middle answers %+v %v", c.name, hit, ok)
		}
		closed := corner(v.Character)
		s.ShopClick(ui.ShopControl{Kind: ui.ShopControlBook})
		w := s.ShopScreen()
		opened := corner(w.Character)
		if c.book && (closed != green || !s.shopBook || !w.Book || !w.Character.BookOpen || opened != red) {
			t.Errorf("%s: corner %v then %v, book %v view %v open %v", c.name, closed, opened, s.shopBook, w.Book, w.Character.BookOpen)
		}
		if !c.book && (closed == green || closed == red || s.shopBook || w.Book || w.Character.BookOpen || opened != closed) {
			t.Errorf("%s: corner %v then %v, book %v view %v open %v", c.name, closed, opened, s.shopBook, w.Book, w.Character.BookOpen)
		}
	}
	for _, c := range []struct {
		name string
		edit func(d map[string]any)
		want string
	}{
		{"moved shop pane", modTownPane("shop", []any{0, 0, 160, 242}, true),
			`room "shop" pane rectangle (0,0)-(160,242) is not the character region (480,238)-(640,480)`},
		{"tavern book", modTownPane("tavern", region, true), `room "tavern" pane has the spellbook corner; the tavern page has no spellbook`},
		{"school book", modTownPane("school", region, true), `room "school" pane has the spellbook corner; the school page has no spellbook`},
		{"shop book offset", func(d map[string]any) {
			editRoom(d, "shop", func(r map[string]any) {
				r["pane"] = map[string]any{"rect": region, "book": true, "book_offset": []any{0, 2}}
			})
		}, `room "shop" pane book_offset (0,2) is not zero; the shop page does not move its spellbook`},
	} {
		if _, _, err := layeredFront(t, minimalEastMods(t, c.edit), "east-town"); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v\nwant %q", c.name, err, c.want)
		}
	}
}
