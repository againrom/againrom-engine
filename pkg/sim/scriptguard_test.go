package sim

import "testing"

func scriptGuardWorld(t *testing.T, second bool) *World {
	t.Helper()
	w := mustWorld(t, 113, Bounds{Width: 32, Height: 32}, []Entity{
		{ID: 10, Owner: 1, Group: 19, X: 8, Y: 8, HP: 100, MaxHP: 100, Speed: 20, ScanRange: 2},
		{ID: 20, Owner: 1, Group: 19, X: 9, Y: 8, HP: 100, MaxHP: 100, Speed: 20, ScanRange: 2},
	})
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	g := SavedGroup{ID: 71, Selector: 19}
	g.Members = []SavedGroupMember{{1, 10, true}, {2, 20, true}}
	g.AI[0x20], g.AI[0x45], g.AI[0x39] = 4, 1, 5
	if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{{Entity: 10}, {Entity: 20}}); err != nil {
		t.Fatal(err)
	}
	return w
}

// TestScriptGuardTakesItsRange: the widest sum is 1 cell + sight 2 = 3
// (AI-793, R2-ENGINE-1253).
func TestScriptGuardTakesItsRange(t *testing.T) {
	for _, tc := range []struct {
		name      string
		second    bool
		par1      int32
		radius    uint8
		programme uint8
	}{
		{"first n=3", false, 3, 3, orderGuard},
		{"first n=2 below the sum", false, 2, 3, orderGuard},
		{"first n=12", false, 12, 12, orderGuard},
		{"first n=0", false, 0, minimalGuardRange, orderGuard},
		{"first n low byte 0", false, 0x100, minimalGuardRange, orderGuard},
		{"second n=0", true, 0, 20, secondGuardRoute},
		{"second n=6", true, 6, 6, secondGuardRoute},
		{"second n=12", true, 12, 12, secondGuardRoute},
	} {
		w := scriptGuardWorld(t, tc.second)
		w.cmdSavedGroupOrder(ScriptInstant{HasGroup: true, Group: 19, Args: [scriptParams]int32{1, tc.par1}})
		g := w.savedGroupFor(10)
		if g.AI[0x2c] != 3 || g.AI[0x2d] != tc.radius || g.AI[0x38] != tc.radius || g.AI[0x39] != 0 || g.AI[0x20] != tc.programme {
			t.Errorf("%s: +0x2c=%d +0x2d=%d +0x38=%d +0x39=%d +0x20=%d, want 3 %d %d 0 %d", tc.name, g.AI[0x2c],
				g.AI[0x2d], g.AI[0x38], g.AI[0x39], g.AI[0x20], tc.radius, tc.radius, tc.programme)
		}
	}
}
