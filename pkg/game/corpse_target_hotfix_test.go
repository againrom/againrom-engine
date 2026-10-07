package game

import "testing"

func TestEntityDrawCarriesTheSimulationTargetBoundary(t *testing.T) {
	finishable := deathUnit(1, 1, 2, 2)
	finishable.HP = -9
	corpse := deathUnit(2, 1, 3, 2)
	corpse.HP = -10
	mw := deathWorld(t, nil, finishable, corpse)
	if draw := deathDraw(t, mw, 1); draw.Untargetable {
		t.Fatal("HP -9 crossed the UI seam as untargetable")
	}
	if draw := deathDraw(t, mw, 2); !draw.Untargetable {
		t.Fatal("HP -10 crossed the UI seam as targetable")
	}
}
