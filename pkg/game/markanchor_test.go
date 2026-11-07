package game

import (
	"fmt"
	"image"
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func markAnchorWorld(t *testing.T, game base.Game, side int) *mapWorld {
	t.Helper()
	rules := []sim.SpellRule{{ID: 6}, {ID: 11}}
	var script *sim.Script
	if game == base.GameROM2 {
		rules = []sim.SpellRule{{ID: 24}, {ID: 26}}
		sim.AssignSecondGameArms(rules)
		var err error
		script, err = sim.NewROM2Script(nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	e := sim.Entity{ID: 1, X: 3, Y: 3, HP: 50, MaxHP: 100, TokenSize: uint8(side), Owner: 1}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, nil,
		[]sim.Entity{e}, script, rules)
	if err != nil {
		t.Fatal(err)
	}
	edition := game.Edition()
	frame := &terrain.StaticFrame{Width: 64, Height: 64, Pixels: make([]terrain.StaticPixel, 64*64)}
	art := &terrain.UnitClass{Width: 64, Height: 64, CenterX: 32, CenterY: 32, TileSize: side,
		Frames: []*terrain.StaticFrame{frame}}
	return &mapWorld{world: w, mission: &missionNotices{table: &mapload.Table{Edition: &edition}},
		units:       &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{0: art}},
		projectiles: emSheets(20, 30, 56, 60)}
}

func TestHealAndDrainMarksUseTheActorAnchorInBothGames(t *testing.T) {
	for _, game := range []base.Game{base.GameROM1, base.GameROM2} {
		for _, side := range []int{1, 2, 3} {
			for _, arm := range []uint16{6, 11} {
				t.Run(fmt.Sprintf("%v/side%d/arm%d", game, side, arm), func(t *testing.T) {
					mw := markAnchorWorld(t, game, side)
					var spell uint16
					for _, row := range mw.world.Spells() {
						if row.ArmID() == arm {
							spell = row.ID
						}
					}
					if spell == 0 {
						t.Fatal("spell arm absent")
					}
					ev := sim.CastEvent{Caster: 1, Target: 1, Spell: spell, Weapon: true, Produced: true,
						FromX: 3, FromY: 3, ToX: 3, ToY: 3, TargetOwner: 1}
					mw.advanceEffectMarks(mw.observeCasts([]sim.CastEvent{ev})...)
					marks := mw.markDraws(1, mw.effectTileSize(1))
					if len(marks) == 0 || len(mw.healBursts) != 0 {
						legacy := mw.healSpriteDraws()
						t.Fatalf("actor marks=%d, fixed-cell bursts=%d, fixed-cell sprites=%d; want shared actor mark at footprint centre %v",
							len(marks), len(mw.healBursts), len(legacy), image.Pt(3*32+side*16, 3*32+side*16))
					}
					v, err := ui.NewViewer("mark anchor", terrain.Grid{Width: 12, Height: 12, Tiles: make([]uint16, 144)}, &terrain.Tileset{})
					if err != nil {
						t.Fatal(err)
					}
					art := mw.units.Classes[0]
					v.SetFogReveal(true)
					v.Camera().ViewW, v.Camera().ViewH = 384, 384
					v.Camera().SetZoom(1)
					v.SetEntities([]ui.MapEntity{{ID: 1, Cell: image.Pt(3, 3), TokenSize: side,
						Art: art, Frame: art.Frames[0], Marks: marks}})
					draws, err := v.HeadlessArtDraws()
					if err != nil {
						t.Fatal(err)
					}
					var body image.Point
					for _, d := range draws {
						if d.Frame == art.Frames[0] && d.Kind == "sprite" {
							x, y := d.Geometry.Apply(32, 32)
							body = image.Pt(int(x), int(y))
						}
					}
					if body == (image.Point{}) {
						t.Fatal("drawn body absent")
					}
					want, got := map[image.Point]int{}, map[image.Point]int{}
					for _, m := range marks {
						want[body.Add(image.Pt(m.Mark.DX-m.Sheet.CenterX, m.Mark.DY-m.Mark.Depth-m.Sheet.CenterY))]++
					}
					for _, d := range draws {
						if d.Kind == "effect" {
							x, y := d.Geometry.Apply(0, 0)
							got[image.Pt(int(x), int(y))]++
						}
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("mark submissions %v, want %v around body %v", got, want, body)
					}
					before := marks
					rows := mw.captureTargetMarks()
					if err := validateTargetMarks(rows); err != nil {
						t.Fatalf("SAVE rejects actor mark: %v", err)
					}
					mw.markElements = nil
					mw.restoreTargetMarks(rows)
					if got := mw.markDraws(1, mw.effectTileSize(1)); !reflect.DeepEqual(got, before) {
						t.Fatal("restored actor mark changed its records")
					}
					ev.AtCell = true
					mw.observeCasts([]sim.CastEvent{ev})
					if len(mw.healBursts) != 1 {
						t.Fatal("cell-targeted shower lost its fixed anchor")
					}
				})
			}
		}
	}
}
