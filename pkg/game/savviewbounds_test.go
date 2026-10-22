package game

import (
	"math"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func viewAfterSAVUpperBounds(v ui.SaveApplicationState, width, height int) ui.SaveApplicationState {
	v.ViewX -= math.Max(0, math.Round(v.ViewX)-math.Max(0, float64(width-35)))
	rows := 22
	if v.SpellBookOpen {
		rows = 19
	}
	v.ViewY -= math.Max(0, math.Round(v.ViewY)-math.Max(0, float64(height-8-rows)))
	return v
}

func TestSAVViewBounds(t *testing.T) {
	f, snapshot, _ := applicationSnapshotFixture(t)
	for _, tc := range []struct {
		name               string
		town               bool
		book               int32
		width, height      int
		x, y, wantX, wantY int32
	}{
		{"open bottom boundary", false, 1, 80, 80, 8, 53, 8, 53},
		{"open bottom past boundary", false, 1, 80, 80, 8, 54, 8, 53},
		{"closed bottom boundary", false, 0, 80, 80, 8, 50, 8, 50},
		{"closed bottom past boundary", false, 0, 80, 80, 8, 51, 8, 50},
		{"right", false, 1, 80, 80, 60, 20, 45, 20},
		{"corner", false, 0, 96, 112, 96, 112, 61, 82},
		{"interior", false, 1, 80, 80, 12, 13, 12, 13},
		{"no camera", false, 0, 80, 80, 0, 0, 0, 0},
		{"town corner", true, 0, 40, 40, 35, 35, 5, 10},
		{"town interior", true, 1, 40, 40, 5, 13, 5, 13},
		{"town negative source", true, 0, 40, 40, 5, -32, 5, -32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := sav.CloneDocumentData(*snapshot.SavedDocument.Document)
			if err != nil {
				t.Fatal(err)
			}
			f.live.mission.state.Map.Width, f.live.mission.state.Map.Height = tc.width, tc.height
			set := func(d *sav.DocumentData, path string, value int32) {
				t.Helper()
				if _, err := patchStateLeaf(d.State.RootKind, d.State.DirectoryRecords, d.State.ValueRecords, path, func(v *sav.CityStateValueData) error {
					v.Int32 = value
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			set(&doc, "/View/X", tc.x)
			set(&doc, "/View/Y", tc.y)
			set(&doc, "/SpellBook/IsOpen", tc.book)
			want, err := sav.CloneDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			if tc.town {
				doc.World, want.World = nil, nil
			}
			set(&want, "/View/X", tc.wantX)
			set(&want, "/View/Y", tc.wantY)
			if err := f.clampOriginalView(&doc); err != nil {
				t.Fatal(err)
			}
			x, err := readOriginalStateLeaf(&doc, "/View/X", 2)
			if err != nil {
				t.Fatal(err)
			}
			y, err := readOriginalStateLeaf(&doc, "/View/Y", 2)
			if err != nil {
				t.Fatal(err)
			}
			if x.Int32 != tc.wantX || y.Int32 != tc.wantY {
				t.Fatalf("saved view (%d,%d), want (%d,%d)", x.Int32, y.Int32, tc.wantX, tc.wantY)
			}
			if !reflect.DeepEqual(doc, want) {
				t.Fatal("clamp changed fields outside View/X and View/Y")
			}
		})
	}
}
