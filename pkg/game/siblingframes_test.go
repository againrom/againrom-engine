package game_test

import (
	"image/color"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

func frameSibling(f any) *terrain.StaticFrame {
	v := reflect.ValueOf(f).Elem().FieldByName("Boundary")
	if !v.IsValid() || v.IsNil() {
		return nil
	}
	return v.Interface().(*terrain.StaticFrame)
}

func TestSiblingSheetsFollowEveryBaseFrame(t *testing.T) {
	for _, family := range []string{"unit", "hero", "object", "structure", "shot", "catap1", "catap2", "sack"} {
		for _, state := range []string{"paired", "missing", "mismatched", "broken"} {
			t.Run(family+"/"+state, func(t *testing.T) {
				i := func(k string, n int32) synth.RegNode { return synth.RegNode{Name: k, Kind: 2, Int: n} }
				basePath, regPath := "units/test/sprites.256", game.UnitRegistry
				registry := synth.UnitsReg([]string{`test\sprites`}, []synth.RegNode{i("ID", 3), i("File", 0)})
				switch family {
				case "hero":
					basePath = "units/heroes_l/swordsman/sprites.256"
				case "object":
					basePath, regPath = "objects/test/sprites.256", game.ObjectRegistry
					registry = synth.ObjectsReg([]string{`test\sprites`}, []synth.RegNode{i("ID", 0), i("File", 0), i("Index", 1)})
				case "structure":
					basePath, regPath = "structures/test.256", game.StructureRegistry
					registry = synth.StructuresReg(structSection(1, "test", 1, 1, 1))
				case "shot", "catap1", "catap2":
					basePath, regPath = "projectiles/goblin/arrow.256", game.ProjectileRegistry
					file := `goblin\arrow`
					if family != "shot" {
						file = family + `\sprites`
						basePath = "projectiles/" + family + "/sprites.256"
					}
					registry = synth.ProjectilesReg(1, []synth.RegNode{projFile(file), i("ID", 1), i("A16", 0), i("Palette", 1)})
				case "sack":
					basePath = "backpack/sprites.256"
				}
				palette := []color.RGBA{{R: 77, G: 119, B: 201}, {R: 203, G: 41, B: 83}}
				sheet := func(n int, own bool) []byte {
					frames := make([]synth.Frame256, n)
					for k := range frames {
						pixels := make([]synth.Pixel256, 32*32)
						pixels[k] = synth.Pixel256{Index: uint8(k), Opaque: true}
						frames[k] = synth.Frame256{Width: 32, Height: 32, Pixels: pixels}
					}
					return synth.Sheet256(synth.Sheet256Options{Palette: palette, NoPalette: !own, Frames: frames})
				}
				files := []synth.File{{Path: graphicsEntry(t, regPath), Data: registry}, {Path: basePath, Data: sheet(2, true)}}
				if state != "missing" {
					n := 2
					if state == "mismatched" {
						n = 1
					}
					bytes := sheet(n, false)
					if state == "broken" {
						bytes = []byte{0}
					}
					files = append(files, synth.File{Path: basePath[:len(basePath)-4] + "b.256", Data: bytes})
				}
				src := openContainers(t, synth.Archive(files))
				var frames []*terrain.StaticFrame
				switch family {
				case "unit", "hero":
					set, err := game.LoadUnits(src)
					if err != nil {
						t.Fatal(err)
					}
					frames = set.Classes[3].Frames
					if family == "hero" {
						game.LoadHeroBody(src, set, "heroes_l", data.BodySwordsman)
						frames = set.Bodies[data.HeroBodyKey("heroes_l", data.BodySwordsman)].Frames
					}
				case "object":
					set, err := game.LoadStatics(src)
					if err != nil {
						t.Fatal(err)
					}
					frames = set.Classes[1].Frames
				case "structure":
					set, err := game.LoadStructures(src)
					if err != nil {
						t.Fatal(err)
					}
					frames = set.Classes[1].Frames
				case "shot", "catap1", "catap2":
					set, err := game.LoadProjectiles(src)
					if err != nil {
						t.Fatal(err)
					}
					if len(set.Sheets[1].Frames) != 2 {
						t.Fatal("base frames changed")
					}
					for k, f := range set.Sheets[1].Frames {
						checkSiblingFrame(t, frameSibling(f), k, state, palette)
					}
					return
				case "sack":
					frames = game.LoadSackFrames(src)
					paired := game.LoadSackBoundaries(src)
					for k := range frames {
						var b *terrain.StaticFrame
						if k < len(paired) {
							b = paired[k]
						}
						checkSiblingFrame(t, b, k, state, palette)
					}
					return
				}
				if len(frames) != 2 {
					t.Fatalf("base has %d frames", len(frames))
				}
				for k, f := range frames {
					checkSiblingFrame(t, frameSibling(f), k, state, palette)
				}
			})
		}
	}
}

func checkSiblingFrame(t *testing.T, b *terrain.StaticFrame, k int, state string, palette []color.RGBA) {
	t.Helper()
	if state != "paired" {
		if b != nil {
			t.Fatal("invalid sibling admitted")
		}
		return
	}
	if b == nil {
		t.Fatal("paired frame was not attached")
	}
	if !b.Pixels[k].Opaque || b.Pixels[k].Index != uint8(k) {
		t.Fatal("wrong sibling index")
	}
	want := palette[k]
	want.A = 255
	if b.Palette[k] != want {
		t.Fatalf("sibling palette %v, want base %v", b.Palette[k], want)
	}
}
