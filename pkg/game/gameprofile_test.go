package game

import (
	"fmt"

	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// worldProfileRow is a loaded World's answer to which game it runs and
// whether it holds the Group registry, with its hash when loaded and after
// stepping.
type worldProfileRow struct {
	second, registry bool
	loaded, stepped  uint64
}

func (r worldProfileRow) String() string {
	return fmt.Sprintf("{second: %v, registry: %v, loaded: %#x, stepped: %#x}", r.second, r.registry, r.loaded, r.stepped)
}

// playsSecondGame reports whether every profile answer pkg/game reads is the
// second game's.
func playsSecondGame(w *sim.World) bool {
	return w.SpellGame() == rules.SpellGameSecond && w.NativeMovers() && w.KeepsCastRetain() && w.ResidueOverBookCasts() &&
		w.SavesMissionFog() && !w.RestoresCompanionObjective()
}
