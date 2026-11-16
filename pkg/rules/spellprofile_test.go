package rules

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestGameProfileIsTheBookTheGameDraws(t *testing.T) {
	for _, g := range []SpellGame{SpellGameFirst, SpellGameSecond} {
		p := GameProfile(g)
		if p.BookSize != 24 || len(p.BookRows()) != 24 || SpellBookPositions() != 24 {
			t.Errorf("game %d book size %d rows %d", g, p.BookSize, len(p.BookRows()))
		}
		q, err := p.Edited(0, nil)
		if err != nil || !slices.Equal(q.BookRows(), p.BookRows()) || !slices.Equal(q.Spells, p.Spells) {
			t.Errorf("game %d: an empty edit changes the profile: %v", g, err)
		}
	}
}

func TestEditedSwitchesMovesAndSizesTheBook(t *testing.T) {
	edits := []SpellEdit{{Name: "Fire_Ball", Off: true}, {Name: "Wall of Fire", Position: 1}, {Name: "Fire_Arrow", Position: 3}}
	for _, g := range []SpellGame{SpellGameFirst, SpellGameSecond} {
		p := GameProfile(g)
		q, err := p.Edited(23, edits)
		if err != nil {
			t.Fatal(err)
		}
		rows := q.BookRows()
		want := slices.Clone(p.BookRows()[:23])
		want[0], want[1], want[2] = 3, 0, 1
		if !slices.Equal(rows, want) {
			t.Errorf("game %d rows %v, want %v", g, rows, want)
		}
		last := p.BookRows()[23]
		for _, id := range []int{2, int(last)} {
			if q.ShopBook(id) || q.ShopScroll(id) || q.Spells[id-1].Slot != 0 {
				t.Errorf("game %d spell %d stays in the book or a shop: %+v", g, id, q.Spells[id-1])
			}
			if q.Spells[id-1].Arm != p.Spells[id-1].Arm || q.Spells[id-1].Name != p.Spells[id-1].Name {
				t.Errorf("game %d spell %d lost its arm or name", g, id)
			}
		}
		for id := 1; id <= len(p.Spells); id++ {
			if id == 2 || id == int(last) {
				continue
			}
			if q.ShopBook(id) != p.ShopBook(id) || q.ShopScroll(id) != p.ShopScroll(id) {
				t.Errorf("game %d spell %d shop admission moved", g, id)
			}
		}
	}
}

func TestEditedRefusesNameTheEditAndTheLimit(t *testing.T) {
	p := GameProfile(SpellGameFirst)
	for name, c := range map[string]struct {
		size  int
		edits []SpellEdit
		edit  int
		want  string
	}{
		"size above":        {25, nil, -1, "book_size is 25, outside 1..24"},
		"size below":        {-1, nil, -1, "book_size is -1, outside 1..24"},
		"position past":     {23, []SpellEdit{{Name: "Heal"}, {Name: "Bless", Position: 24}}, 1, "book_position of Bless is 24, outside 1..23"},
		"off with position": {0, []SpellEdit{{Name: "Bless", Off: true, Position: 2}}, 0, "Bless is off, so book_position does not apply"},
		"off then position": {0, []SpellEdit{{Name: "Bless", Off: true}, {Name: "Bless", Position: 2}}, 1, "Bless is off"},
		"collision":         {0, []SpellEdit{{Name: "Bless", Position: 2}}, 0, "book_position 2 holds both Fire_Ball and Bless"},
		"unslotted":         {0, []SpellEdit{{Name: "Drain_Life", On: true}}, 0, "Drain_Life has no place in this game's book"},
		"unknown":           {0, []SpellEdit{{Name: "Ice_Missile", Position: 1}}, 0, "Ice_Missile is not a spell of this game's catalog"},
		"on past the size":  {23, []SpellEdit{{Name: "Shield", On: true}}, 0, "Shield is on, but its book position 24 is past book_size 23"},
	} {
		_, err := p.Edited(c.size, c.edits)
		var e *SpellEditError
		if !errors.As(err, &e) || e.Edit != c.edit || !strings.Contains(e.Msg, c.want) {
			t.Errorf("%s: %v, want edit %d %q", name, err, c.edit, c.want)
		}
	}
}

// An explicit on is judged against the cell the edits leave: a move before or
// after it, in the same row or another, keeps it valid; with none it overflows.
func TestEditedOnFollowsTheMovedCell(t *testing.T) {
	for _, g := range []SpellGame{SpellGameFirst, SpellGameSecond} {
		p := GameProfile(g)
		var name, first string
		for _, s := range p.Spells {
			switch s.Slot {
			case 1:
				first = s.Name
			case 20:
				name = s.Name
			}
		}
		clear := SpellEdit{Name: first, Off: true}
		for label, edits := range map[string][]SpellEdit{
			"one row":  {clear, {Name: name, On: true, Position: 1}},
			"move, on": {clear, {Name: name, Position: 1}, {Name: name, On: true}},
			"on, move": {clear, {Name: name, On: true}, {Name: name, Position: 1}},
		} {
			q, err := p.Edited(10, edits)
			if err != nil || q.BookRows()[0] == 0 {
				t.Errorf("%v %s: %v", g, label, err)
			}
		}
		_, err := p.Edited(10, []SpellEdit{{Name: name, On: true}})
		var e *SpellEditError
		if !errors.As(err, &e) || e.Edit != 0 || e.Msg != name+" is on, but its book position 20 is past book_size 10" {
			t.Errorf("%v overflow: %v", g, err)
		}
	}
}
