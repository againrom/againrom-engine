package game

import "againrom/pkg/mapload"

// townPickerMembers lists, in party order, the party indices the shop, school
// and tavern pickers stand on: every member except a tavern hire.
//
// The original steps each room's picker over the local player's own units that
// carry the player-character flag, and a hired unit carries the mercenary flag
// instead, so it is in none of the three arrays (TOWN-138, PARTY-FLAG-003). The
// hired squad stays in the party: the next mission takes it along (MERC-HIRE-003,
// MERC-DEATH-006).
func townPickerMembers(party []mapload.PartyMember) []int {
	out := make([]int, 0, len(party))
	for i := range party {
		if !party[i].Hired() {
			out = append(out, i)
		}
	}
	return out
}

// townPickerIndex is the party index the rooms act on for a stored selection.
// A selection outside the party reads as the first member, and one that names a
// hire reads as the first member a picker admits. A party of hires alone has no
// such member and keeps its selection; no route in play builds one.
func townPickerIndex(party []mapload.PartyMember, selected int) int {
	if len(party) == 0 {
		return 0
	}
	if selected < 0 || selected >= len(party) {
		selected = 0
	}
	if !party[selected].Hired() {
		return selected
	}
	for i := range party {
		if !party[i].Hired() {
			return i
		}
	}
	return selected
}

// townPickerPosition is the picker's own numbering for party index i: its place
// among the members the picker steps over, and how many there are.
func townPickerPosition(party []mapload.PartyMember, i int) (position, count int) {
	for j := range party {
		if party[j].Hired() {
			continue
		}
		if j == i {
			position = count
		}
		count++
	}
	return position, count
}
