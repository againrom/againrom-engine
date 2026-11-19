package game

import (
	"image"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// teleportSoundWorld is a Teleport map of either game cast by book or scroll.
func teleportSoundWorld(t *testing.T, second, scroll, turn bool) (*mapWorld, int) {
	t.Helper()
	row, class, delay := uint16(26), int32(1), int32(2)
	caster := sim.Entity{ID: 1, X: 8, Y: 8, Class: class, HP: 100, MaxHP: 100,
		Owner: 7, Mind: 40, Mana: 100, MaxMana: 100, KnownSpells: 1 << 26,
		AttackCharge: tpCharge, AttackRelax: 4, ScanRange: 30, TokenSize: 1}
	rules := []sim.SpellRule{{ID: row, ManaCost: 3, School: 5, MaxRange: 20}}
	var script *sim.Script
	if second {
		row, class, delay = secondTeleportRow, int32(secondMageClass.id), int32(secondMageClass.delay)
		caster.Class, caster.KnownSpells, caster.Humanoid = class, 1<<row, true
		rules = []sim.SpellRule{{ID: row, ManaCost: 3, School: 5, MaxRange: 20, Delivery: 1}}
		mapload.SecondGameSpellArms(rules)
		s, err := sim.NewROM2Script(missionChecks(), nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		script = s
	}
	if turn {
		caster.Facing, caster.RotationSpeed = 0x80, 16
	}
	var stock []sim.Stock
	cmd := sim.CastAt(1, sim.SpellID(row), sim.CellPoint{X: 12, Y: 8})
	if scroll {
		caster.Mana, caster.MaxMana, caster.KnownSpells = 0, 0, 0
		item := sim.ItemInstance{Code: 0xe10, Kind: 4, Price: 50,
			Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(row) | 50<<16}}}
		stock = []sim.Stock{{ID: 1, ItemInstances: []sim.ItemInstance{item}}}
		cmd = sim.UseScrollAt(1, 0, sim.CellPoint{X: 12, Y: 8})
	}
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{caster}, script, sim.Relations{}, nil, stock, rules)
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	v, err := ui.NewViewer("teleport-sound", terrain.Grid{Width: 32, Height: 32, Tiles: make([]uint16, 32*32)}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	units := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{class: secondTeleportArt(secondMageClass)}}
	mw := newMapWorld(w, nil, units, v)
	if second {
		edition := base.GameROM2.Edition()
		mw.mission = &missionNotices{table: &mapload.Table{Edition: &edition, SpellArms: mapload.SecondGameSpellArms}}
	}
	mw.sounds = map[int32]UnitSound{class: {AttackDelay: delay}}
	mw.projectiles = sbSheets()
	mw.sched = [][]sim.Command{{cmd}}
	return mw, int(delay)
}

// teleportSoundTicks answers the slots heard on each tick.
func teleportSoundTicks(mw *mapWorld, n int) map[int][]int {
	heard := map[int][]int{}
	tick := 0
	mw.spellSound = func(slot int, _ uint32, _ image.Point) { heard[tick] = append(heard[tick], slot) }
	for ; tick < n; tick++ {
		mw.tick()
	}
	return heard
}

// Every Teleport cast path sounds in both games (R2-ENGINE-704).
func TestTeleportSoundOfEveryCastPath(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		second, scroll, turn bool
		want                 map[int][]int
	}{
		{"first game book", false, false, false, map[int][]int{0: {560}, 2: {560}}},
		{"first game scroll", false, true, false, map[int][]int{1: {560}, 3: {560}}},
		{"second game book", true, false, false, map[int][]int{8: {554}}},
		{"second game scroll", true, true, false, map[int][]int{9: {554}}},
		{"first game book turning", false, false, true, map[int][]int{3: {560}, 5: {560}}},
		{"second game book turning", true, false, true, map[int][]int{11: {554}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw, _ := teleportSoundWorld(t, tc.second, tc.scroll, tc.turn)
			got := teleportSoundTicks(mw, 30)
			if len(got) != len(tc.want) {
				t.Fatalf("heard %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if len(got[k]) != len(v) || got[k][0] != v[0] {
					t.Errorf("tick %d heard %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
