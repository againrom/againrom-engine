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

// secondHeroClass is the type a hero casts as: 14 updates, the sound at the
// first (R2-ENGINE-704).
var secondHeroClass = secondTeleportClass{id: 1, run: 14, delay: 0, centerX: 64, centerY: 78, selection: image.Rect(48, 48, 80, 90)}

// teleportSoundWorld is a Teleport map of either game cast by book or scroll.
// A queued scroll is used one tick after a book cast of the same caster, so it
// waits behind that cast and its recovery.
func teleportSoundWorld(t *testing.T, second, scroll, turn, queued bool, art secondTeleportClass) (*mapWorld, int) {
	t.Helper()
	row, class, delay := uint16(26), int32(1), int32(2)
	caster := sim.Entity{ID: 1, X: 8, Y: 8, Class: class, HP: 100, MaxHP: 100,
		Owner: 7, Mind: 40, Mana: 100, MaxMana: 100, KnownSpells: 1 << 26,
		AttackCharge: tpCharge, AttackRelax: 4, ScanRange: 30, TokenSize: 1}
	rules := []sim.SpellRule{{ID: row, ManaCost: 3, School: 5, MaxRange: 20}}
	var script *sim.Script
	if second {
		row, class, delay = secondTeleportRow, int32(art.id), int32(art.delay)
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
		if !queued {
			caster.Mana, caster.MaxMana, caster.KnownSpells = 0, 0, 0
		}
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
	units := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{class: secondTeleportArt(art)}}
	mw := newMapWorld(w, nil, units, v)
	if second {
		edition := base.GameROM2.Edition()
		mw.mission = &missionNotices{table: &mapload.Table{Edition: &edition, SpellArms: mapload.SecondGameSpellArms}}
	}
	mw.sounds = map[int32]UnitSound{class: {AttackDelay: delay}}
	mw.projectiles = sbSheets()
	mw.sched = [][]sim.Command{{cmd}}
	if queued {
		mw.sched = [][]sim.Command{{sim.CastAt(1, sim.SpellID(row), sim.CellPoint{X: 12, Y: 8})}, {cmd}}
	}
	return mw, int(delay)
}

// teleportSoundTicks answers the slots heard on each tick.
func teleportSoundTicks(mw *mapWorld, n int) map[int][]int {
	heard := map[int][]int{}
	tick := 0
	mw.semanticSound = func(_ string, slot int, _ uint32, _, _ image.Point) { heard[tick] = append(heard[tick], slot) }
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
		queued               bool
		want                 map[int][]int
	}{
		{"first game book", false, false, false, false, map[int][]int{0: {560}, 2: {560}}},
		{"first game scroll", false, true, false, false, map[int][]int{1: {560}, 3: {560}}},
		{"second game book", true, false, false, false, map[int][]int{8: {554}}},
		{"second game scroll", true, true, false, false, map[int][]int{9: {554}}},
		{"first game book turning", false, false, true, false, map[int][]int{3: {560}, 5: {560}}},
		{"second game book turning", true, false, true, false, map[int][]int{11: {554}}},
		{"second game queued scroll", true, true, false, true, map[int][]int{8: {554}, 24: {554}}},
		{"second game scroll turning", true, true, true, false, map[int][]int{9: {554}}},
		{"second game queued scroll turning", true, true, true, true, map[int][]int{9: {554}, 25: {554}}},
		{"first game queued scroll", false, true, false, true, map[int][]int{0: {560}, 2: {560}, 16: {560}, 18: {560}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw, _ := teleportSoundWorld(t, tc.second, tc.scroll, tc.turn, tc.queued, secondMageClass)
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

// A hero casts as type 1: its sound is at the first update of its run.
func TestTeleportSoundHeroClass(t *testing.T) {
	for _, tc := range []struct {
		scroll bool
		tick   int
	}{{false, 0}, {true, 1}} {
		mw, _ := teleportSoundWorld(t, true, tc.scroll, false, false, secondHeroClass)
		got := teleportSoundTicks(mw, 30)
		if len(got) != 1 || len(got[tc.tick]) != 1 || got[tc.tick][0] != 554 {
			t.Errorf("scroll=%v heard %v, want tick %d once", tc.scroll, got, tc.tick)
		}
	}
}

// The second game voices a Teleport cast at the cell the cast began in; the
// first game keeps the caster's live cell for its animation sound (DIV-4289).
func TestTeleportSoundCell(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		second, scroll, queue bool
		// cells is the cell of each cast's start, in the order the casts sound.
		cells []image.Point
	}{
		{"second game book", true, false, false, []image.Point{{X: 8, Y: 8}}},
		{"second game scroll", true, true, false, []image.Point{{X: 8, Y: 8}}},
		{"second game queued scroll", true, true, true, []image.Point{{X: 8, Y: 8}, {X: 12, Y: 8}}},
		{"first game book", false, false, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw, _ := teleportSoundWorld(t, tc.second, tc.scroll, false, tc.queue, secondMageClass)
			type heard struct{ cell, live image.Point }
			var got []heard
			mw.semanticSound = func(_ string, slot int, _ uint32, cell, _ image.Point) {
				e, _ := mw.entity(1)
				got = append(got, heard{cell, image.Pt(int(e.X), int(e.Y))})
			}
			for i := 0; i < 30; i++ {
				mw.tick()
			}
			if len(got) == 0 {
				t.Fatal("no sound heard")
			}
			if !tc.second {
				for _, h := range got {
					if h.cell != h.live {
						t.Errorf("first game sound at %v, want the live cell %v", h.cell, h.live)
					}
				}
				return
			}
			if len(got) != len(tc.cells) {
				t.Fatalf("heard %d sounds, want %d", len(got), len(tc.cells))
			}
			jumped := false
			for i, h := range got {
				jumped = jumped || h.live != tc.cells[0]
				if h.cell != tc.cells[i] {
					t.Errorf("sound %d at %v (caster now at %v), want the start cell %v", i, h.cell, h.live, tc.cells[i])
				}
			}
			if !jumped {
				t.Fatal("the caster never left the first start cell; the test proves nothing")
			}
		})
	}
}
