package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// adSheets is a synthetic art bundle carrying only Wall of Fire's burst sheet.
// It opens no archive and decodes no pixel: a frame's size is all this file
// asserts on.
func adSheets(picture int) *terrain.EffectSet {
	frames := make([]*terrain.EffectFrame, 12)
	for i := range frames {
		frames[i] = &terrain.EffectFrame{Width: 8, Height: 8, Pixels: make([]color.RGBA, 64)}
	}
	return &terrain.EffectSet{Sheets: map[int]*terrain.EffectSheet{
		picture: {Frames: frames, Phases: 12, RotationPhases: 1, CenterX: 32, CenterY: 32},
	}}
}

func TestAnAreaEffectSpriteStandsOnItsOwnCell(t *testing.T) {
	t.Parallel()

	const cx, cy int32 = 76, 109
	caster := sim.Entity{ID: 1, X: cx - 4, Y: cy, HP: 100, MaxHP: 100, Owner: 1, TokenSize: 1,
		Mind: 30, Mana: 500, MaxMana: 500, KnownSpells: 1 << 3, ScanRange: 19,
		AttackCharge: 4, AttackRelax: 2}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 128, Height: 128}, sim.ModeCanonical, nil,
		[]sim.Entity{caster}, nil, []sim.SpellRule{{ID: 3, ManaCost: 5, School: 1, MaxRange: 8,
			Area: true, Distribution: 4, Radius: 2, AreaDuration: 15,
			DamageMin: 4, DamageMax: 4, Damaging: true}})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindCastAt, Entity: 1, X: cx, Y: cy, Spell: 3}})
	for i := 0; i < 64 && len(w.CellEffects()) == 0; i++ {
		sim.Step(w, nil)
	}
	effects := w.CellEffects()
	if len(effects) != 1 {
		t.Fatalf("the cast left %d area effects, want 1", len(effects))
	}
	if len(effects[0].Cells) != 10 {
		t.Fatalf("the wall covers %d cells, want the table's 10", len(effects[0].Cells))
	}

	mw := &mapWorld{world: w, projectiles: adSheets(data.BurstPicture(3))}
	got := mw.areaEffectDraws(w.CellEffects(), nil)
	if len(got) != len(effects[0].Cells) {
		t.Fatalf("%d covered cells produced %d sprites, want one each", len(effects[0].Cells), len(got))
	}
	far := 0
	for _, b := range got {
		want := image.Pt(b.Cell.X*ui.ShotScale, b.Cell.Y*ui.ShotScale)
		if b.Pos != want {
			t.Errorf("the sprite for cell %v stands at %v, want %v", b.Cell, b.Pos, want)
			continue
		}
		// The consumer's own arithmetic, so the assertion is in the units it
		// divides by rather than in the ones this side multiplies in.
		// 1003: the half cell is the decoded ground point, StaticAnchor's own
		// `col*CellSize + CellSize/2`. Without it every sprite here stood on
		// its cell's top-left corner, 16 px left and 16 px up of its cell.
		px := b.Pos.X*terrain.CellSize/ui.ShotScale + terrain.CellSize/2 - b.Sheet.CenterX
		py := b.Pos.Y*terrain.CellSize/ui.ShotScale + terrain.CellSize/2 - b.Sheet.CenterY
		if px != int(b.Cell.X)*terrain.CellSize+terrain.CellSize/2-b.Sheet.CenterX ||
			py != int(b.Cell.Y)*terrain.CellSize+terrain.CellSize/2-b.Sheet.CenterY {
			t.Errorf("cell %v draws at world pixel (%d,%d)", b.Cell, px, py)
		}
		if px > terrain.CellSize && py > terrain.CellSize {
			far++
		}
	}
	if far != len(got) {
		t.Errorf("%d of %d sprites stand within one cell of the map origin", len(got)-far, len(got))
	}
}
