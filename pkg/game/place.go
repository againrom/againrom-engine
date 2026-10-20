package game

// partyPlace is where the party stands, in both games: a town or a mission,
// by its ID. The zero place is none: the party is on the world map or has not
// reached a place yet. The second game's SAV leaf stores it as its Current
// and Available records, kind and ID as they are.
type partyPlace struct {
	kind int
	id   int
}

// The kinds of a place, numbered as the second game's scenario bank numbers
// its records (mission 1, town 2).
const (
	placeMission = 1
	placeTown    = 2
)

// town answers the town ID when the place is a town.
func (p partyPlace) town() (int, bool) { return p.id, p.kind == placeTown }
