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

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseTownFidelity1192(t *testing.T) {
	t.Run("item-text", func(t *testing.T) {
		f := releaseFront(t)
		for id := uint32(1); id <= 28; id++ {
			book := gameSpellBook(id)
			book.Price = 321
			lines := itemInstanceInfoLines(book, f.Table, f.Words)
			if len(lines) != 1 || f.Words.ItemSpellNames[id] == "" || lines[0] != itemName(data.ItemCode(book.Code), f.Table)+" "+f.Words.ItemSpellOf+" "+f.Words.ItemSpellNames[id]+f.Words.ItemSpellOfSuffix {
				t.Fatalf("book%d: %q", id, lines)
			}
		}
		item := sim.ItemInstance{Code: 0x812d, Price: 150, Effects: []sim.ItemEffect{{Kind: 49, Operand: 5}, {Kind: 21, Operand: 10}}}
		lines := itemInstanceInfoLines(item, f.Table, f.Words)
		magic := -1
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				t.Fatal("blank item line")
			}
			if line == f.Words.ItemMagic {
				if magic != -1 {
					t.Fatal("duplicate magic header")
				}
				magic = i
			}
		}
		if magic < 1 || magic+3 != len(lines) || lines[magic+1] != "#"+f.Words.ItemStats[49]+" +5" ||
			lines[magic+2] != itemEffectInfoLine(item.Effects[1], f.Table, f.Words) {
			t.Fatal(lines)
		}
		plain := item.Clone()
		plain.Effects = nil
		for _, line := range itemInstanceInfoLines(plain, f.Table, f.Words) {
			if line == f.Words.ItemMagic {
				t.Fatal("plain item has magic header")
			}
		}
	})
	t.Run("siege-cards-and-mission", func(t *testing.T) {
		f := releaseFront(t)
		f.Carried = f.NextParty()
		s := f.TownScreen().(*townScreen)
		for typ := 1; typ <= 2; typ++ {
			members, ok := s.buildMercenarySquad(typ, 1)
			if !ok {
				t.Fatal("missing siege", typ)
			}
			f.Carried = append(f.Carried, members[0])
		}
		if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(20)(); err != nil {
			t.Fatal(err)
		}
		for typ := 1; typ <= 2; typ++ {
			m := f.Carried[len(f.Carried)-3+typ]
			card := partyPanelSubject(m, f.Table, f.Words)
			candidate, _, _, ok := s.tavernCandidateDetail(typ)
			if !ok {
				t.Fatal("missing candidate card")
			}
			i := len(f.live.mission.ids) - 3 + typ
			e, ok := f.live.entity(f.live.mission.ids[i])
			if !ok {
				t.Fatal("missing siege entity")
			}
			wantHP := []int{0, 250, 170}[typ]
			if card.HP != wantHP || card.HP != int(e.HP) || card.MaxHP != int(e.MaxHP) || card.Combat.DamageBase != int(e.DamageBase) || card.Combat.DamageSpread != int(e.DamageSpread) || card.Combat.ToHit != int(e.ToHit) || card.Combat.Defence != int(e.Defence) || card.Combat.Absorption != int(e.Absorption) || card.Speed != int(e.Speed) || card.Char.Sight != int(e.ScanRange) || candidate.Subject.HP != wantHP || candidate.Subject.Combat != card.Combat {
				t.Fatalf("siege%d card%+v entity%+v candidate%+v", typ, card, e, candidate.Subject)
			}
		}
	})
	t.Run("siege-potion-card", func(t *testing.T) {
		for _, typ := range []int{1, 2} {
			a, shop := releaseShopApp(t)
			defer a.StopAudio()
			shop.CloseTip()
			members, ok := shop.buildMercenarySquad(typ, 1)
			if !ok {
				t.Fatal("missing siege template")
			}
			shop.sess.Carried, shop.shopMember = members, 0
			item := mapload.ItemInstanceFromCode(0xe01, shop.in.Table)
			if !shop.setShopPackItemInstances([]sim.ItemInstance{item, item}) {
				t.Fatal("missing potion pack")
			}
			before := shop.ShopScreen().Character.Subject.Combat.Absorption
			shopPointer1090(t, a, "pack", 1, "press", "release", "press", "release")
			p := shop.shopPartyMember(0)
			if p.PotionEffect == nil || p.PotionEffect.Kind != sim.EffectAbsorption || p.PotionEffect.Magnitude != 50 || len(shop.shopPackItemInstances()) != 1 {
				t.Fatal("potion was not consumed with its timer")
			}
			if after := shop.ShopScreen().Character.Subject.Combat.Absorption; after != before+50 {
				t.Fatalf("siege%d absorption %d, want %d", typ, after, before+50)
			}
			// Let the real simulation expire the timer, then project the returned hire.
			w, err := sim.NewStockedWorld(0, sim.Bounds{Width: 1, Height: 1}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{{ID: 1, HP: 250, MaxHP: 250, Absorption: int32(before)}}, nil, sim.Relations{}, nil, nil)
			if err != nil || !w.RestorePotionEffect(1, *p.PotionEffect) {
				t.Fatal("cannot restore potion", err)
			}
			for ticks := 0; ticks < int(p.PotionEffect.Remaining); ticks++ {
				sim.Step(w, nil)
			}
			returned := mapload.CarryParty([]mapload.PartyMember{*p}, w, []sim.EntityID{1})
			if len(returned) != 1 || returned[0].PotionEffect != nil {
				t.Fatal("expired effect survived return")
			}
			d, _, _ := mapload.PartyDisplayWithTable(returned[0], shop.in.Table)
			if d.Combat.Absorption != int32(before) {
				t.Fatalf("expired siege%d potion remained: %d", typ, d.Combat.Absorption)
			}
		}
	})
	t.Run("shop-pointer", func(t *testing.T) {
		a, s := releaseShopApp(t)
		defer a.StopAudio()
		s.CloseTip()
		frontOf(s).SetDeterministicFrames(true)
		before := mapload.CloneParty(s.sess.Carried)
		gold := s.sess.Town.Gold()
		s.sess.originalCity = &originalCitySaveState{trade: &originalCityTrade{}}
		binding, trade := s.sess.originalCity, s.sess.originalCity.trade
		for _, index := range []int{1, 2} {
			r := ui.ShopButtonRect(index)
			p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
			frame := func() *image.RGBA {
				pic, _, err := a.HeadlessFrame()
				if err != nil {
					t.Fatal(err)
				}
				return pic
			}
			pointer := func(kind string, at image.Point) {
				if err := a.HeadlessPointer(kind, at.X, at.Y); err != nil {
					t.Fatal(err)
				}
			}
			pointer("hover", image.Pt(290, 300))
			idle := frame()
			pointer("hover", p)
			hover := frame()
			pointer("press", p)
			down := frame()
			pointer("move", image.Pt(290, 300))
			outside := frame()
			pointer("release", p)
			released := frame()
			if sameRegion1192(idle, hover, r) || sameRegion1192(hover, down, r) || !sameRegion1192(idle, outside, r) || !sameRegion1192(hover, released, r) {
				t.Fatal("shop button lost hover/press/cancel/release", index)
			}
			// An ordinary press/release invokes the empty command.
			pointer("press", p)
			pointer("release", p)
			if !s.ShopScreen().Live[index] || !reflect.DeepEqual(before, s.sess.Carried) || s.sess.Town.Gold() != gold || s.sess.originalCity != binding || s.sess.originalCity.trade != trade {
				t.Fatal("empty command changed town state", index)
			}
			for name, pic := range map[string]*image.RGBA{"idle": idle, "hover": hover, "down": down} {
				capture1192(t, fmt.Sprintf("shop%d-%s", index, name), pic)
			}
		}
	})
	t.Run("town-and-generator-plaques", func(t *testing.T) {
		a, s := releaseShopApp(t)
		defer a.StopAudio()
		s.CloseTip()
		// Exit through the real button, then enter each door through App.
		p := ui.ShopButtonRect(3).Min.Add(ui.ShopButtonRect(3).Max).Div(2)
		if err := a.HeadlessPointer("press", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("release", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		for _, door := range []string{"TAVERN", "SCHOOL"} {
			if err := a.HeadlessActivate(door); err != nil {
				t.Fatal(err)
			}
			for n := 0; s.room == roomTalk && n < 32; n++ {
				if err := a.HeadlessActivate("dialogue"); err != nil {
					t.Fatal(err)
				}
			}
			s.CloseTip()
			view := s.TownSurface()
			exit := len(view.Buttons) - 1
			for i, b := range view.Buttons {
				if b.Enabled {
					checkPlaquePointer1192(t, a, fmt.Sprintf("%s%d", door, i), ui.TownSurfaceButtonRect(view.Kind, i))
				}
			}
			p := ui.TownSurfaceButtonRect(view.Kind, exit).Min.Add(ui.TownSurfaceButtonRect(view.Kind, exit).Max).Div(2)
			if err := a.HeadlessPointer("press", p.X, p.Y); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessPointer("release", p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
		f := releaseFront(t)
		gen := f.App("generator plaques")
		gen.SetNewGameChargen(func() *ui.ChargenEntry { return &ui.ChargenEntry{Model: ui.NewChargen(f.ChargenSetup())} })
		defer gen.StopAudio()
		gen.SetCutscenes(nil)
		gen.Layout(640, 480)
		if err := gen.HeadlessActivate("new game"); err != nil {
			t.Fatal(err)
		}
		state, ok := gen.HeadlessChargenState()
		if !ok {
			t.Fatal("generator did not open")
		}
		forward, ok := state.Control(ui.ChargenControlForward, "")
		if !ok {
			t.Fatal("forward control missing")
		}
		for n := 0; state.Focus != forward.Focus && n < 32; n++ {
			if err := gen.HeadlessKey("down"); err != nil {
				t.Fatal(err)
			}
			state, _ = gen.HeadlessChargenState()
		}
		if err := gen.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
		state, ok = gen.HeadlessChargenState()
		if !ok || state.Stage != ui.ChargenStageDetailed {
			t.Fatal("detailed generator did not open", state)
		}
		for i, r := range []image.Rectangle{image.Rect(486, 69, 617, 111), image.Rect(486, 115, 617, 160), image.Rect(504, 163, 601, 208)} {
			checkPlaquePointer1192(t, gen, fmt.Sprintf("generator%d", i), r)
		}
	})
}

func checkPlaquePointer1192(t *testing.T, a *ui.App, name string, r image.Rectangle) {
	t.Helper()
	p := r.Min.Add(r.Max).Div(2)
	frame := func() *image.RGBA {
		pic, _, err := a.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		return pic
	}
	pointer := func(kind string, p image.Point) {
		if err := a.HeadlessPointer(kind, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	outside := image.Pt(400, 450)
	pointer("hover", outside)
	idle := frame()
	pointer("hover", p)
	hover := frame()
	pointer("press", p)
	down := frame()
	pointer("move", outside)
	away := frame()
	pointer("release", outside)
	pointer("hover", p)
	released := frame()
	for state, pic := range map[string]*image.RGBA{"idle": idle, "hover": hover, "down": down} {
		capture1192(t, name+"-"+state, pic)
	}
	if sameRegion1192(idle, hover, r) || sameRegion1192(hover, down, r) || !sameRegion1192(idle, away, r) || !sameRegion1192(hover, released, r) {
		t.Fatal(name, "lost hover/press/cancel/release", sameRegion1192(idle, hover, r), sameRegion1192(hover, down, r), sameRegion1192(idle, away, r), sameRegion1192(hover, released, r))
	}
}

func sameRegion1192(a, b *image.RGBA, r image.Rectangle) bool {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}

func capture1192(t *testing.T, name string, pic *image.RGBA) {
	t.Helper()
	out := os.Getenv("AGAINROM_TOWN_FIDELITY_OUT")
	if out == "" {
		return
	}
	root, err := filepath.Abs(os.Getenv("AGAINROM_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	dest, err := filepath.Abs(out)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, dest)
	if err != nil || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("output must be outside install")
	}
	if err = os.MkdirAll(dest, 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dest, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, pic); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}
