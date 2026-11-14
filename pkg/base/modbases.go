package base

// AllBases is the applies-to entry that names every base.
const AllBases = "common"

// ModBase is one applies-to entry and its game, empty for every base.
type ModBase struct {
	Name string
	Game Game
}

// ModBases is the one table of applies-to entries, in the editor's order.
var ModBases = []ModBase{
	{Name: AllBases},
	{Name: ROM1, Game: GameROM1}, {Name: ROM1EN, Game: GameROM1}, {Name: ROM1RU, Game: GameROM1},
	{Name: ROM2, Game: GameROM2}, {Name: ROM2EN, Game: GameROM2}, {Name: ROM2RU, Game: GameROM2},
}

// ModFamilyOf is game g's first family word.
func ModFamilyOf(g Game) string {
	return g.Edition().ModFamily[0]
}
