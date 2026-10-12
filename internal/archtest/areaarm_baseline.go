package archtest

// CommittedAreaArmSites is each production file's arm site count. A fall
// lowers its row in the same commit; the live test logs measured rows.
var CommittedAreaArmSites = map[string]int{
	"pkg/game/mapinspection.go":      4,
	"pkg/game/savworldeffects.go":    1,
	"pkg/game/shopconsumables.go":    2,
	"pkg/game/spell.go":              1,
	"pkg/game/spellbolt.go":          5,
	"pkg/game/world.go":              1,
	"pkg/sim/areasave.go":            1,
	"pkg/sim/castbinary.go":          1,
	"pkg/sim/celleffect.go":          9,
	"pkg/sim/creaturecast.go":        9,
	"pkg/sim/currentareas.go":        1,
	"pkg/sim/currentdeliveries.go":   2,
	"pkg/sim/currentworldpolicy.go":  1,
	"pkg/sim/effect.go":              2,
	"pkg/sim/itemattribution.go":     2,
	"pkg/sim/rom2script.go":          2,
	"pkg/sim/savedcontinuation.go":   2,
	"pkg/sim/savedspellgraph.go":     2,
	"pkg/sim/scorched.go":            2,
	"pkg/sim/sourceeffect.go":        1,
	"pkg/sim/spell.go":               5,
	"pkg/sim/spelldeliverybinary.go": 1,
	"pkg/sim/spelldeliverysave.go":   1,
	"pkg/sim/spellformula.go":        4,
	"pkg/sim/spellvalues.go":         1,
}
