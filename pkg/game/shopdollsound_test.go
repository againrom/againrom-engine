package game

import (
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// dollSoundRecord points the shop page's sound hook at a list and answers the
// events it requests, in order.
func dollSoundRecord(s *townScreen) *[]string {
	var events []string
	s.shop.sound = func(on string) { events = append(events, on) }
	return &events
}

// The shop requests doll-on once for every gesture that puts an item on the
// shown member's figure and doll-off once for every gesture that takes one
// off, by drag, by tap and by double click, from and to the pack and the
// trade table. A refused gesture requests neither, and the table's own drop
// and pick sounds do not join a gesture that ends on the figure.
func TestShopDollRequestsPutOnAndPutOff(t *testing.T) {
	doll := func(slot int) ui.ShopControl { return ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1} }
	table := ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0}
	steps := []struct {
		name string
		do   func(s *townScreen)
		want []string
	}{
		{"wear from the pack by drag", func(s *townScreen) { s.ShopDrag(secondDollPack(s, 0), doll(1)) }, []string{"doll-on"}},
		{"take off into the pack by drag", func(s *townScreen) { s.ShopDrag(doll(2), secondDollPack(s, 0)) }, []string{"doll-off"}},
		{"wear again, take off by tap", func(s *townScreen) {
			s.ShopDrag(secondDollPack(s, 0), doll(1))
			s.ShopClick(doll(1))
		}, []string{"doll-on", "doll-off"}},
		{"take off onto the table by drag", func(s *townScreen) {
			s.ShopDrag(secondDollPack(s, 0), doll(1))
			s.ShopDrag(doll(1), table)
		}, []string{"doll-on", "doll-off"}},
		{"double click wears", func(s *townScreen) {
			s.ShopClick(secondDollPack(s, 0))
			s.ShopDrag(table, doll(1))
		}, []string{"table-drop", "doll-on"}},
		{"a refused wear", func(s *townScreen) { s.ShopDrag(secondDollPack(s, 1), doll(1)) }, nil},
		{"an empty slot taken off", func(s *townScreen) { s.ShopDrag(doll(5), secondDollPack(s, 0)) }, nil},
	}
	for _, st := range steps {
		s, _ := secondDollShop(t, secondDollAxe, secondDollStaff)
		events := dollSoundRecord(s)
		st.do(s)
		if !reflect.DeepEqual(*events, st.want) {
			t.Errorf("%s: requested %v, want %v", st.name, *events, st.want)
		}
	}
}

// The second game's town descriptions bind doll-on to Put_On and doll-off to
// Put_Off in the shop; the first game's binds neither, so its shop stays
// silent on the figure.
func TestShopDollSoundsAreRoomData(t *testing.T) {
	slotOf := func(town string, on string) string {
		for _, d := range townDescriptions {
			if d.Town != town {
				continue
			}
			room := d.Room("shop")
			if room == nil || room.Scene == nil {
				t.Fatalf("%s has no shop scene", town)
			}
			for _, tr := range room.Scene.Sounds.Triggers {
				if tr.On == on {
					return tr.Slot
				}
			}
			return ""
		}
		t.Fatalf("no description %s", town)
		return ""
	}
	for _, town := range []string{"rom2-town1", "rom2-town2", "rom2-town3"} {
		if got := slotOf(town, "doll-on"); got != "put-on" {
			t.Errorf("%s binds doll-on to %q", town, got)
		}
		if got := slotOf(town, "doll-off"); got != "put-off" {
			t.Errorf("%s binds doll-off to %q", town, got)
		}
	}
	for _, on := range []string{"doll-on", "doll-off"} {
		if got := slotOf("rom1-square", on); got != "" {
			t.Errorf("rom1-square binds %s to %q", on, got)
		}
	}
}
