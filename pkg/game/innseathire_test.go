package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/screenart"
)

func TestInnSeatShowsTheHireItsActionTakes(t *testing.T) {
	g, _, screen := secondCampaignFixtureFiles(t, false)
	g.Table = keyedHireTable(t)
	swordsman := mercenaryHumanRow(5, 1, 0, 10)
	swordsman[24] = 5
	g.Table.Humans = append(g.Table.Humans.(dbCollection), dbEntry{name: "Man_Sword2Handed", params: swordsman})
	c := innAt(3, 80, map[int]int32{534: 1, 554: 3, 535: 1, 555: 3, 774: 1})
	g.Town.second = c
	screen.composed = newTownSceneState(nil)
	screen.enterSeats()
	d := screen.townDescription()
	r := d.Room("inn")
	page := &screenart.Set{}

	seats, hires := screen.innPageSeats()
	var keys []int
	for _, s := range seats {
		keys = append(keys, s.word.npc)
	}
	if hires != 2 || !slices.Equal(keys[:3], []int{3, 4, 5}) || screen.composed.seat != 0 {
		t.Fatalf("seats %v, %d hires, seat %d selected; want hires 3 and 4, then talk 5, seat 0", keys, hires, screen.composed.seat)
	}
	want := map[int]int32{3: 100, 4: 102, 5: 101}
	check := func(when string) {
		t.Helper()
		view := roomPageView{t: screen, d: d, room: "inn"}
		for i, npc := range keys[:3] {
			member, ok := screen.campaignActor(npc)
			if !ok || member.Class != want[npc] {
				t.Errorf("%s: seat %d (key %d) shows type %d, want %d", when, i, npc, member.Class, want[npc])
			}
			if p := screen.innPatron(r, page, npc); !p.known || p.subject.UnitNameIndex != int(want[npc]) {
				t.Errorf("%s: seat %d (key %d) card names type %d, want %d", when, i, npc, p.subject.UnitNameIndex, want[npc])
			}
		}
		if p := view.selectedPatron(r, page); p == nil || p.subject.UnitNameIndex != int(want[keys[screen.composed.seat]]) {
			t.Errorf("%s: the panel of seat %d does not name its hire", when, screen.composed.seat)
		}
	}
	check("before a click")
	for i := range keys[:3] {
		screen.composed.seat = i
		check(fmt.Sprintf("after a click on seat %d", i))
	}

	word, ok := c.innWord(5)
	if !ok {
		t.Fatal("the talk seat of key 5 has no word")
	}
	c.talkTo(word)
	for _, npc := range []int{3, 4, 5} {
		c.seatHire(npc)
	}
	party, err := enterHires(secondCampaignRules{}, nil, g.Town, g.Table)
	if err != nil {
		t.Fatal(err)
	}
	var taken []int32
	for _, m := range party {
		taken = append(taken, m.Class)
	}
	if !slices.Equal(taken, []int32{100, 102, 101}) {
		t.Fatalf("the hires take types %v, want 100, 102, 101 as the seats showed", taken)
	}
	check("after the talk and the hires")
}
