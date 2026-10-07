package sim

import (
	"slices"
	"testing"
)

func TestNativeTerminalTransitionSynchronizesSavedMotionCellAndAction(t *testing.T) {
	for _, relocate := range []bool{false, true} {
		name := "already aligned"
		if relocate {
			name = "living superseded motion"
		}
		t.Run(name, func(t *testing.T) {
			actor := Entity{ID: 41, X: 4, Y: 5, HP: 30, MaxHP: 30}
			w, err := NewWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, []Entity{actor})
			if err != nil {
				t.Fatal(err)
			}
			motion := SavedActorMotion{Entity: actor.ID, Position: SavedActorPosition{
				Cell: 0x0504, PackedCell: 0x0504, FineX: 128, FineY: 128,
				Residue: 0x136a, TerrainKey: 0x56473829,
			}}
			motion.Mover[10], motion.Mover[31] = 18, 0xa7
			if err := w.ImportOriginalActorMotions([]SavedActorMotion{motion}, nil, nil); err != nil {
				t.Fatal(err)
			}
			if relocate {
				if err := w.HeadlessPlace(actor.ID, 6, 6); err != nil {
					t.Fatal(err)
				}
			}
			before := *w.motionFor(actor.ID)
			current, ok := w.Entity(actor.ID)
			if !ok || !current.Alive() {
				t.Fatal("control is not a living actor")
			}
			cell := uint16(current.Y)<<8 | uint16(current.X)
			if relocate {
				if before.Current || before.Position.Cell == cell || before.Position.PackedCell == cell {
					t.Fatalf("living superseded control changed frozen position: actor=%#04x motion=%+v", cell, before)
				}
			} else if before.Position.Cell != cell || before.Position.PackedCell != cell {
				t.Fatalf("already aligned control actor=%#04x motion=%+v", cell, before)
			}
			if err := w.HeadlessKill(actor.ID); err != nil {
				t.Fatal(err)
			}
			after := *w.motionFor(actor.ID)
			current, ok = w.Entity(actor.ID)
			if !ok || current.Alive() || current.Decay < DecayBones {
				t.Fatalf("ordinary death did not reach terminal teardown: %+v held=%v", current, ok)
			}
			if after.Position.Cell != cell || after.Position.PackedCell != cell {
				t.Fatalf("dying motion cell=%#04x packed=%#04x, actor=%#04x", after.Position.Cell, after.Position.PackedCell, cell)
			}
			wantPosition := before.Position
			wantPosition.Cell, wantPosition.PackedCell = cell, cell
			if after.Position != wantPosition || after.Mover != before.Mover || !slices.Equal(after.StaticRoute, before.StaticRoute) || !slices.Equal(after.DynamicRoute, before.DynamicRoute) {
				t.Fatalf("death changed frozen motion outside its cell: before=%+v after=%+v", before, after)
			}
			if after.ActorAction != 16 {
				t.Fatalf("terminal teardown kept ordinary action %d", after.ActorAction)
			}
			if after.Current || after.Active || after.Issue == "" {
				t.Fatalf("death did not supersede motion: %+v", after)
			}
		})
	}
}
