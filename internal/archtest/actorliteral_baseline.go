package archtest

// CommittedActorLiterals is the falling-only count of sim.Entity composite
// literals per non-test file outside pkg/sim/actorconstructor.go. None of
// them creates an actor: each is a probe entity handed to a rule function or
// a one-actor scratch world, a witness tool's fixture, or the snapshot
// decoder restoring an encoded actor field by field. Regenerate with:
//
//	go run ./internal/archtest/cmd/actorliteral
var CommittedActorLiterals = map[string]int{
	"cmd/areaoverlaycheck/main.go":   1,
	"cmd/weaponspellcheck/main.go":   1,
	"pkg/game/chargen.go":            1,
	"pkg/game/iteminfo.go":           2,
	"pkg/game/originalspellbook.go":  3,
	"pkg/game/shopview.go":           1,
	"pkg/game/world.go":              1,
	"pkg/mapload/potion.go":          3,
	"pkg/mapload/sourceequipment.go": 1,
	"pkg/sim/binary.go":              1,
	"pkg/sim/carry.go":               1,
	"pkg/sim/effectwitness.go":       2,
	"pkg/sim/nativebasismutation.go": 2,
	"pkg/sim/savedstride.go":         1,
	"pkg/sim/script.go":              1,
	"pkg/sim/structurecombat.go":     1,
}
