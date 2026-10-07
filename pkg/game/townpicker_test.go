package game

import (
	"fmt"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// townPickerRooms are the three rooms with a character picker: the input that
// presses one chevron and the reader of the numbering the room draws.
var townPickerRooms = []struct {
	name   string
	room   townRoom
	press  func(s *townScreen, next bool) ui.TownAction
	number func(s *townScreen) (position, count int)
}{
	{"tavern", roomTavern, pressSurfacePicker, surfacePickerNumber},
	{"school", roomSchool, pressSurfacePicker, surfacePickerNumber},
	{"shop", roomShop, pressShopPicker, shopPickerNumber},
}

func pressSurfacePicker(s *townScreen, next bool) ui.TownAction {
	kind := ui.TownSurfaceControlPrevious
	if next {
		kind = ui.TownSurfaceControlNext
	}
	return s.TownSurfaceClick(ui.TownSurfaceControl{Kind: kind}, false)
}

func pressShopPicker(s *townScreen, next bool) ui.TownAction {
	kind := ui.ShopControlPickerPrev
	if next {
		kind = ui.ShopControlPickerNext
	}
	return s.ShopClick(ui.ShopControl{Kind: kind})
}

func surfacePickerNumber(s *townScreen) (int, int) {
	v := s.TownSurface().Hero
	return v.Member, v.MemberCount
}

func shopPickerNumber(s *townScreen) (int, int) {
	v := s.ShopScreen()
	return v.Member, v.MemberCount
}

// hiredSquadTown is the tavern fixture with a shop, the type 3 squad hired
// through the tavern's own Hire button, and optionally a second player
// character. squadBetween puts the squad between the two player characters,
// the order a party read back from a save can hold.
func hiredSquadTown(t *testing.T, companion, squadBetween bool) (*FrontEnd, *townScreen) {
	t.Helper()
	f := shellFrontEnd()
	table := shopTable()
	table.Humans, table.NPC = f.Table.Humans, f.Table.NPC
	f.Table = table
	f.Shop = NewShop(1000)
	f.Shop.Generate(f.Table, shopSeed(f.Town.Chapter(), f.Town.finishedCount(), f.Shop.Ceiling()))
	if companion {
		f.Carried = append(f.Carried, mapload.PartyMember{Name: "Second", PlayerCharacter: true, Hero: data.NewCampaignHero(2)})
	}
	s := f.townUI
	s.composeShopFaces()
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	if act := s.townSurfaceButton(tavernButtonHire); act.Msg != "squad 3 hired for 50" {
		t.Fatalf("Hire = %q, want the squad hired", act.Msg)
	}
	if got := s.mercenaryPartyCount(3); got != 2 {
		t.Fatalf("party squad = %d, want 2", got)
	}
	if squadBetween && companion {
		f.Carried = []mapload.PartyMember{f.Carried[0], f.Carried[2], f.Carried[3], f.Carried[1]}
		s.composeShopFaces()
	}
	return f, s
}

// playerCharacters is the party index of every member a picker stands on.
func playerCharacters(f *FrontEnd) []int {
	var out []int
	for i, m := range f.Carried {
		if !m.Hired() {
			out = append(out, i)
		}
	}
	return out
}

// TestTownPickersStepOverPlayerCharactersOnly covers TOWN-138 and
// PARTY-FLAG-003: the original's three pickers step over the local player's
// own units carrying the player-character flag, and a hired unit carries the
// mercenary flag instead. With a squad in the party each room's picker visits
// the player characters alone, in both directions, wraps between them, and
// numbers them without counting the squad.
func TestTownPickersStepOverPlayerCharactersOnly(t *testing.T) {
	for _, between := range []bool{false, true} {
		for _, room := range townPickerRooms {
			t.Run(fmt.Sprintf("%s squad between %v", room.name, between), func(t *testing.T) {
				f, s := hiredSquadTown(t, true, between)
				want := playerCharacters(f)
				if len(want) != 2 || len(f.Carried) != 4 {
					t.Fatalf("fixture party has %d members, player characters %v", len(f.Carried), want)
				}
				s.room = room.room
				check := func(step string, at int) {
					t.Helper()
					got := s.shopMemberIndex()
					if got != want[at] || f.Carried[got].Hired() {
						t.Fatalf("%s: the picker shows party member %d (%q, hired %v), want member %d",
							step, got, f.Carried[got].Name, f.Carried[got].Hired(), want[at])
					}
					if position, count := room.number(s); position != at || count != 2 {
						t.Fatalf("%s: the picker numbers the member %d of %d, want %d of 2", step, position, count, at)
					}
				}
				s.shopMember = want[0]
				check("opening", 0)
				for i, at := range []int{1, 0, 1} {
					if act := room.press(s, true); act.Msg == "" {
						t.Errorf("forward step %d said nothing", i+1)
					}
					check(fmt.Sprintf("forward step %d", i+1), at)
				}
				for i, at := range []int{0, 1, 0} {
					if act := room.press(s, false); act.Msg == "" {
						t.Errorf("backward step %d said nothing", i+1)
					}
					check(fmt.Sprintf("backward step %d", i+1), at)
				}
				if !f.Town.MercenaryHired(3) || len(f.Carried) != 4 || s.mercenaryPartyCount(3) != 2 {
					t.Fatalf("stepping changed the hire: hired %v, party %d, squad %d",
						f.Town.MercenaryHired(3), len(f.Carried), s.mercenaryPartyCount(3))
				}
			})
		}
	}
}

// TestTownPickersStandStillWithOnePlayerCharacterAndASquad is the common route:
// a hero alone hires a squad. The picker has nobody to step to, so it says so
// in every room and the numbering reads one member.
func TestTownPickersStandStillWithOnePlayerCharacterAndASquad(t *testing.T) {
	for _, room := range townPickerRooms {
		t.Run(room.name, func(t *testing.T) {
			f, s := hiredSquadTown(t, false, false)
			s.room = room.room
			if len(f.Carried) != 3 {
				t.Fatalf("fixture party has %d members, want the hero and two hires", len(f.Carried))
			}
			for _, next := range []bool{true, false} {
				act := room.press(s, next)
				if act.Msg != "nobody else is with you" {
					t.Errorf("stepping alone said %q", act.Msg)
				}
				if got := s.shopMemberIndex(); got != 0 || f.Carried[got].Hired() {
					t.Fatalf("the picker moved to party member %d (%q, hired %v)", got, f.Carried[got].Name, f.Carried[got].Hired())
				}
			}
			if position, count := room.number(s); position != 0 || count != 1 {
				t.Fatalf("the picker numbers the member %d of %d, want 0 of 1", position, count)
			}
		})
	}
}

// TestTownRoomsReadAHiredSelectionAsAPlayerCharacter keeps a selection that
// names a hire, from a party that changed under it, off the squad: every room
// acts on the first player character instead.
func TestTownRoomsReadAHiredSelectionAsAPlayerCharacter(t *testing.T) {
	for _, room := range townPickerRooms {
		t.Run(room.name, func(t *testing.T) {
			f, s := hiredSquadTown(t, true, true)
			s.room = room.room
			hired := 1 // the squad sits between the two player characters
			if !f.Carried[hired].Hired() {
				t.Fatalf("fixture party member %d is not hired", hired)
			}
			s.shopMember = hired
			if got := s.shopMemberIndex(); f.Carried[got].Hired() || got != 0 {
				t.Fatalf("a selection on hire %d reads as party member %d (%q)", hired, got, f.Carried[got].Name)
			}
			if position, count := room.number(s); position != 0 || count != 2 {
				t.Fatalf("the picker numbers the member %d of %d, want 0 of 2", position, count)
			}
			if m := s.shopPartyMember(s.shopMemberIndex()); m == nil || m.Hired() {
				t.Fatalf("the room acts on %+v, want a player character", m)
			}
		})
	}
}

// TestSchoolTrainsAPlayerCharacterAfterSteppingPastASquad is the school
// witness: the picker wraps from the last player character to the first, past
// the squad, and Train raises that hero's skill and no hire's.
func TestSchoolTrainsAPlayerCharacterAfterSteppingPastASquad(t *testing.T) {
	f, s := hiredSquadTown(t, true, false)
	s.room = roomSchool
	pressSurfacePicker(s, true)
	pressSurfacePicker(s, true)
	if got := s.shopMemberIndex(); got != 0 {
		t.Fatalf("two steps over a party of two heroes and a squad left the picker on member %d, want 0", got)
	}
	cell := 1
	if f.Carried[0].Mage {
		cell = 6
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: cell}, false)
	slot, price, ok := s.selectedSchoolSlot()
	if !ok {
		t.Fatalf("clicking cell %d selected no skill", cell)
	}
	before := make([]mapload.PartyMember, len(f.Carried))
	copy(before, f.Carried)
	gold := f.Town.Gold()
	if act := s.townSurfaceButton(0); act.Msg == "" {
		t.Fatal("Train said nothing")
	}
	if got, want := f.Carried[0].Hero.Skill[slot], before[0].Hero.Skill[slot]+1; got != want {
		t.Fatalf("Train left the picked hero's skill %d at %d, want %d", slot, got, want)
	}
	if f.Town.Gold() != gold-price {
		t.Fatalf("gold after Train = %d, want %d", f.Town.Gold(), gold-price)
	}
	for i := 1; i < len(f.Carried); i++ {
		if f.Carried[i].Hero != before[i].Hero {
			t.Fatalf("Train changed party member %d (%q, hired %v)", i, f.Carried[i].Name, f.Carried[i].Hired())
		}
	}
}
