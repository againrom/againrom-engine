package game

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// At the first town-1 inn visit patron 0 is selected and every seat holds
// the Humans row whose server id is its NPC key: the card shows the values
// R2-SESSION-203 read, the pools by the five school skills (R2-ENGINE-579),
// the heading by the sheet's rule (R2-ENGINE-581) and SPELLCASTER on the
// mage's card only (R2-ENGINE-580); the seat draws inn/unit<type id>
// frames, and the left oval draws the person's figure. A seat click selects the seat and opens no talk; Talk talks to
// the selected person (R2-SESSION-213).
func TestReleaseSecondInnPatrons(t *testing.T) {
	f, app, s, now := secondTownOne(t)
	c := f.Town.second
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	s.TownSquareActive(true)
	if s.composed.seat != 0 {
		t.Fatalf("the inn opens with seat %d selected, want 0", s.composed.seat)
	}
	d, room, _ := s.composedRoom()
	r := d.Room(room)
	page := s.roomArt(d, room).page
	tbl := f.Table
	type card struct {
		npc                         int
		typeID                      int32
		ownName, spellcaster        bool
		body, agility, mind, spirit int
		hp, mana                    int
		dmg                         [2]int
		attack, defence, absorb     int
		skills                      [5]int
		protection                  [5]int
		sight256, speed             int
	}
	for i, want := range []card{
		{207, 15, false, false, 32, 32, 28, 22, 156, 0, [2]int{23, 37}, 156, 138, 1,
			[5]int{12, 23, 4, 9, 48}, [5]int{11, 11, 36, 11, 11}, 1638, 18},
		{2108, 1, true, false, 25, 20, 25, 20, 55, 0, [2]int{0, 0}, 3, 6, 0,
			[5]int{}, [5]int{10, 10, 10, 10, 10}, 1484, 16},
		{517, 24, false, true, 35, 40, 45, 50, 158, 604, [2]int{21, 43}, 311, 39, 0,
			[5]int{0, 99, 99, 0, 0}, [5]int{25, 25, 25, 25, 25}, 1894, 20},
	} {
		list, _ := c.innSeats()
		if list[i].npc != want.npc {
			t.Fatalf("seat %d holds NPC %d, want %d", i, list[i].npc, want.npc)
		}
		p := s.innPatron(r, page, want.npc)
		if !p.known {
			t.Fatalf("NPC %d resolves no Humans row", want.npc)
		}
		member, _ := mapload.CharacterMember(tbl, mapload.CharacterSpec{Row: data.FindHumanByServerID(tbl.Humans, int32(want.npc))}, nil)
		if member.Class != want.typeID {
			t.Errorf("NPC %d type %d, want %d", want.npc, member.Class, want.typeID)
		}
		sub := p.subject
		name, _ := ui.PanelSubjectName(sub)
		// A key of 1000 or more heads the card with its npcnames entry alone;
		// any other with Mercenary above its type's unitname entry (DIV-4087).
		role, wantName := f.Words.PanelCaptions[mainSlotMercenary], f.Words.UnitNames[want.typeID]
		if want.ownName {
			role, wantName = "", tbl.NPCNames[want.npc-1]
		}
		if name == "" || name != wantName || sub.Role != role {
			t.Errorf("NPC %d heading %q / %q, want %q / %q", want.npc, sub.Role, name, role, wantName)
		}
		t.Logf("NPC %d heading %q / %q", want.npc, sub.Role, name)
		caption := strings.ToUpper(f.Words.PanelCaptions[190])
		spellcaster := false
		for _, row := range ui.PanelStatement(ui.AuthoredPanelLayout(), sub) {
			spellcaster = spellcaster || strings.Contains(strings.ToUpper(row), caption)
		}
		if caption == "" || spellcaster != want.spellcaster {
			t.Errorf("NPC %d states %q: %v, want %v", want.npc, caption, spellcaster, want.spellcaster)
		}
		ch := sub.Char
		got := card{npc: want.npc, typeID: member.Class, ownName: want.ownName, spellcaster: want.spellcaster,
			body: ch.Body, agility: ch.Reaction, mind: ch.Mind, spirit: ch.Spirit, hp: sub.MaxHP, mana: sub.MaxMana,
			attack: sub.Combat.ToHit, defence: sub.Combat.Defence, absorb: sub.Combat.Absorption,
			protection: ch.Protection, sight256: int(ch.Sight256), speed: sub.Speed}
		got.dmg = [2]int{sub.Combat.DamageBase, sub.Combat.DamageBase + sub.Combat.DamageSpread}
		if sub.Combat.WeaponSpellDamageKnown {
			got.dmg = [2]int{int(sub.Combat.SpellDamageBase), int(sub.Combat.SpellDamageBase + sub.Combat.SpellDamageSpread)}
		}
		copy(got.skills[:], ch.Skills[1:])
		if got != want || sub.HP != sub.MaxHP || sub.Mana != sub.MaxMana {
			t.Errorf("NPC %d card\n got %+v\nwant %+v", want.npc, got, want)
		}
		if !ch.ExperienceWithheld || sub.WeightKnown && sub.OriginalPanel.Flags&1 != 0 {
			t.Errorf("NPC %d card states its experience or weight", want.npc)
		}
		bound, err := screenArtOf(f.Archives.artSource()).Bind(r.Name, r.Art, "patron", want.typeID)
		if err != nil || len(bound) == 0 || len(p.frames) != len(bound) || p.frames[0] == nil || !sameImage(p.frames[0], bound[0]) {
			t.Errorf("NPC %d seat frames are not inn/unit%d: %v", want.npc, want.typeID, err)
		}
		if p.figure == nil {
			t.Errorf("NPC %d has no figure", want.npc)
		}
	}
	if a, b := s.innPatron(r, page, 207).frames, s.innPatron(r, page, 2108).frames; sameImage(a[0], b[0]) {
		t.Error("the crossbowman's seat draws the peasant's frames")
	}

	// The oval draws the selected person's figure over the left picture.
	paint := func() *image.RGBA {
		dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
		s.TownSquareView().Scene.Paint(dst)
		return dst
	}
	dst := paint()
	p0 := s.innPatron(r, page, 207)
	figure := p0.figure
	p0.figure = nil
	bare := paint()
	p0.figure = figure
	oval := image.Rect(0, 238, 160, 480)
	if differingPixels(dst, bare) != differingPixels(dst.SubImage(oval).(*image.RGBA), bare.SubImage(oval).(*image.RGBA)) {
		t.Error("the figure draws outside the left oval")
	}
	if differingPixels(dst.SubImage(oval).(*image.RGBA), bare.SubImage(oval).(*image.RGBA)) < 1000 {
		t.Error("the left oval draws no figure")
	}

	// A seat click selects; it opens no talk. Talk talks to the selection.
	click := func(p image.Point) {
		t.Helper()
		for _, edge := range []string{"hover", "press", "release"} {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
		*now = now.Add(10)
		for range 2 {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
	}
	click(image.Pt(247, 445))
	if s.composed.seat != 1 || c.talk.payload != nil {
		t.Fatalf("seat 1 click: seat %d, talk open %v", s.composed.seat, c.talk.payload != nil)
	}
	click(image.Pt(553, 65))
	if s.composed.seat != 1 || c.talk.payload != nil {
		t.Fatal("the blank first button acted on a talk seat")
	}
	click(image.Pt(553, 112))
	if c.talk.payload == nil {
		t.Fatal("Talk with seat 1 selected opened no talk")
	}
	if o := c.speakers()[1]; secondTalkKey(o.npc, o.topic) != "npc2108talk8" {
		key := secondTalkKey(o.npc, o.topic)
		t.Errorf("seat 1 talk key %s", key)
	}
}
