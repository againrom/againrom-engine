package ui

import (
	"bytes"
	"image"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

// TestFogScale is spec AC-8's own table: exactly 1, 0.5, 0 for the three
// states, and a defensive 0 for a value none of the plane's writers can
// produce.
func TestFogScale(t *testing.T) {
	cases := []struct {
		state uint8
		want  float32
	}{
		{FogUnseen, 0},
		{FogExplored, 0.5},
		{FogVisible, 1},
		{3, 0}, // out of range: fogScale's own bounds guard, not fogAt's
	}
	for _, c := range cases {
		if got := fogScale(c.state); got != c.want {
			t.Errorf("fogScale(%d) = %v, want %v", c.state, got, c.want)
		}
	}
}

func TestFogAtWithNoPlanePushedAnswersVisible(t *testing.T) {
	v, err := NewViewer("fog", grid(3, 3), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	for _, c := range []image.Point{{0, 0}, {2, 2}, {-1, -1}, {50, 50}} {
		if got := v.fogAt(c.X, c.Y); got != FogVisible {
			t.Errorf("fogAt(%d,%d) with no plane pushed = %d, want FogVisible", c.X, c.Y, got)
		}
	}
}

// TestFogAtOutOfBoundsAnswersUnseen is plan R-4's own guard: a cell outside
// the PLANE's own cols/rows answers unseen, whether it is negative, past the
// edge, or the plane is simply shorter than cols*rows claims.
func TestFogAtOutOfBoundsAnswersUnseen(t *testing.T) {
	v, err := NewViewer("fog", grid(3, 3), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetFog([]byte{FogVisible, FogVisible}, 2, 2) // 2x2 claimed, only 2 bytes stored

	cases := []image.Point{{-1, 0}, {0, -1}, {2, 0}, {0, 2}, {1, 1}}
	for _, c := range cases {
		if got := v.fogAt(c.X, c.Y); got != FogUnseen {
			t.Errorf("fogAt(%d,%d) = %d, want FogUnseen", c.X, c.Y, got)
		}
	}
}

// TestFogAtAnswersTheStoredByte is fogAt's ordinary case: inside the
// plane's own bounds and within its actual length, the byte the plane holds
// comes back unchanged.
func TestFogAtAnswersTheStoredByte(t *testing.T) {
	v, err := NewViewer("fog", grid(2, 2), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	plane := []byte{FogUnseen, FogExplored, FogVisible, FogUnseen}
	v.SetFog(plane, 2, 2)

	want := map[image.Point]uint8{
		{0, 0}: FogUnseen,
		{1, 0}: FogExplored,
		{0, 1}: FogVisible,
		{1, 1}: FogUnseen,
	}
	for c, w := range want {
		if got := v.fogAt(c.X, c.Y); got != w {
			t.Errorf("fogAt(%d,%d) = %d, want %d", c.X, c.Y, got, w)
		}
	}
}

func TestFogRevealAnswersVisibleWithoutWritingThePlane(t *testing.T) {
	v, err := NewViewer("fog", grid(2, 2), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	plane := []byte{FogUnseen, FogExplored, FogVisible, FogUnseen}
	before := append([]byte(nil), plane...)
	v.SetFog(plane, 2, 2)

	if v.FogRevealed() {
		t.Fatal("a fresh viewer starts with the reveal off")
	}

	if got := v.ToggleFogReveal(); !got {
		t.Fatal("ToggleFogReveal did not turn the reveal on")
	}
	if !v.FogRevealed() {
		t.Fatal("FogRevealed disagrees with ToggleFogReveal's own return")
	}
	for row := 0; row < 2; row++ {
		for col := 0; col < 2; col++ {
			if got := v.fogAt(col, row); got != FogVisible {
				t.Errorf("revealed fogAt(%d,%d) = %d, want FogVisible", col, row, got)
			}
		}
	}
	if !reflect.DeepEqual(plane, before) {
		t.Fatalf("the reveal wrote the plane: got %v, want %v (unchanged)", plane, before)
	}

	if got := v.ToggleFogReveal(); got {
		t.Fatal("a second toggle did not turn the reveal back off")
	}
	for row := 0; row < 2; row++ {
		for col := 0; col < 2; col++ {
			want := before[row*2+col]
			if got := v.fogAt(col, row); got != want {
				t.Errorf("un-revealed fogAt(%d,%d) = %d, want %d (the plane's own answer)", col, row, got, want)
			}
		}
	}
	if !reflect.DeepEqual(plane, before) {
		t.Fatalf("turning the reveal off wrote the plane: got %v, want %v (unchanged)", plane, before)
	}
}

func TestFogGatesEntitiesByOwnerAndVisibility(t *testing.T) {
	v, err := NewViewer("fog", grid(3, 3), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetLocalOwner(1)

	plane := make([]byte, 3*3) // zero value is FogUnseen throughout
	plane[0] = FogVisible      // cell (0,0)
	v.SetFog(plane, 3, 3)

	v.SetEntities([]MapEntity{
		{ID: 1, Cell: image.Pt(1, 1), Owner: 2}, // non-local, unseen: must be absent
		{ID: 2, Cell: image.Pt(0, 0), Owner: 2}, // non-local, visible: must be present
		{ID: 3, Cell: image.Pt(1, 1), Owner: 1}, // local, unseen: must be present anyway
	})

	_, squares, _ := v.entityLayer()
	seen := make(map[uint32]bool, len(squares))
	for _, e := range squares {
		seen[e.ID] = true
	}
	if seen[1] {
		t.Error("a non-local entity on an unseen cell reached the drawn set")
	}
	if !seen[2] {
		t.Error("a non-local entity on a visible cell did not reach the drawn set")
	}
	if !seen[3] {
		t.Error("a local entity on an unseen cell did not reach the drawn set")
	}
}

func TestFogGatesSacksByVisibility(t *testing.T) {
	v, err := NewViewer("fog", grid(2, 2), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetSackFrames([]*terrain.StaticFrame{staticsFrame(4, 4, 1)})
	plane := []byte{FogVisible, FogExplored, FogUnseen, FogUnseen}
	v.SetFog(plane, 2, 2)
	v.SetSacks([]MapSack{
		{Cell: image.Pt(0, 0), FrameIndex: 0}, // visible: kept
		{Cell: image.Pt(1, 0), FrameIndex: 0}, // explored, not visible: dropped
	})

	out := v.sackLayer()
	if len(out) != 1 {
		t.Fatalf("sackLayer() = %d placements, want 1: %+v", len(out), out)
	}
	if out[0].Cell != (image.Point{X: 0, Y: 0}) {
		t.Errorf("sackLayer() kept cell %v, want (0,0)", out[0].Cell)
	}
}

// TestFogGatesStaticAndStructurePlacementsByAnythingButUnseen is spec D-5,
// over the real builder path — staticsGrid's own three placements at
// (0,0), (2,0) and (1,1) (statics_test.go) — so the gate is exercised
// through staticPlacements() and not merely through fogGateGround in
// isolation.
func TestFogGatesStaticAndStructurePlacementsByAnythingButUnseen(t *testing.T) {
	v := newStaticsViewer(t, staticsBundle(), true, true)

	plane := make([]byte, cliffW*cliffH) // zero value is FogUnseen throughout
	plane[0*cliffW+2] = FogExplored      // cell (2,0): stays drawn once explored
	plane[1*cliffW+1] = FogVisible       // cell (1,1): stays drawn, visible
	// cell (0,0) is left FogUnseen and must be dropped.
	v.SetFog(plane, cliffW, cliffH)

	got := v.staticPlacements()
	cells := make(map[image.Point]bool, len(got))
	for _, p := range got {
		cells[p.Cell] = true
	}
	if cells[image.Pt(0, 0)] {
		t.Error("a static placement on an unseen cell reached the drawn set")
	}
	if !cells[image.Pt(2, 0)] {
		t.Error("a static placement on an explored cell did not reach the drawn set")
	}
	if !cells[image.Pt(1, 1)] {
		t.Error("a static placement on a visible cell did not reach the drawn set")
	}
}

// TestFogGateStaticPlacementsPreservesIdentityWithNoPlanePushed guards the
// pointer-identity contract TestSetFlatSelectsAListAndRebuildsNeither
// (statics_test.go) already pins: staticPlacements must keep returning the
// very slice the builder produced when nothing is fog-dropped, and nothing
// is dropped while no plane has been pushed (fogAt is FogVisible
// everywhere). fogGateStaticPlacements' own doc states this is by design
// rather than by accident; this test is what would catch a version that
// filtered unconditionally and broke the promise.
func TestFogGateStaticPlacementsPreservesIdentityWithNoPlanePushed(t *testing.T) {
	v := newStaticsViewer(t, staticsBundle(), true, true)
	want := reflect.ValueOf(v.staticsDisplaced).Pointer()
	if got := reflect.ValueOf(v.staticPlacements()).Pointer(); got != want {
		t.Fatalf("staticPlacements() with no fog plane pushed returned a new slice (pointer %v), want the builder's own slice (%v)", got, want)
	}
}

// TestFogLeavesTerrainLightingForTheFinalShroud is spec AC-8, over the SAME cell's shading
// at two fog states: an unseen answer is all-0, and an explored answer is
// exactly half of what the SAME cell gives when visible — not merely a
// smaller number, half of it, corner by corner. cliffGrid's cell (1,1) is
// interior to the 3x4 fixture and, per
// TestCornerScalesMatchesShadeScaleOfCornerLevels, resolves real shading
// rather than the placeholder case.
func TestFogLeavesTerrainLightingForTheFinalShroud(t *testing.T) {
	v, err := NewViewer("fog", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit")
	}

	const tx, ty = 1, 1
	plane := make([]byte, cliffW*cliffH)
	setAll := func(state byte) {
		for i := range plane {
			plane[i] = state
		}
	}
	setAll(FogVisible)
	v.SetFog(plane, cliffW, cliffH)

	visible := v.cornerScales(tx, ty)
	if visible == ([4]float32{0, 0, 0, 0}) {
		t.Fatal("fixture cell is all-0 even when visible; the comparison below would be vacuous")
	}

	setAll(FogExplored)
	explored := v.cornerScales(tx, ty)
	for i := range visible {
		if want := visible[i]; explored[i] != want {
			t.Errorf("corner %d: explored cornerScales = %v, want half of visible (%v) = %v",
				i, explored[i], visible[i], want)
		}
	}

	setAll(FogUnseen)
	if got := v.cornerScales(tx, ty); got != visible {
		t.Errorf("unseen cornerScales = %v, want unchanged lighting %v", got, visible)
	}
}

// Terrain keeps its lighting. A separate final shroud submits independent
// vertex alpha over the exact same flat or displaced mesh (TERR-FOG-083/084).
// The plane is not changed by either draw pass.
func TestFogFrontierSubmitsFourIndependentLatticeCorners(t *testing.T) {
	const tx, ty = 1, 1
	states := [4]byte{FogVisible, FogExplored, FogUnseen, FogVisible}
	factors := [4]float32{1, 0.5, 0, 1}
	vertices := [4]image.Point{
		image.Pt(tx, ty), image.Pt(tx+1, ty),
		image.Pt(tx, ty+1), image.Pt(tx+1, ty+1),
	}

	for _, flat := range []bool{false, true} {
		name := "displaced"
		if flat {
			name = "flat"
		}
		t.Run(name, func(t *testing.T) {
			v, err := NewViewer("fog-frontier", cliffGrid(), litTileset())
			if err != nil {
				t.Fatalf("NewViewer: %v", err)
			}
			v.SetFlat(flat)
			base := v.cornerShading(tx, ty)
			if base == ([4]float32{}) {
				t.Fatal("fixture terrain has no pre-fog colour; the frontier comparison would be vacuous")
			}

			plane := make([]byte, cliffW*cliffH)
			for i := range plane {
				plane[i] = FogVisible
			}
			for i, p := range vertices {
				plane[p.Y*cliffW+p.X] = states[i]
			}
			before := append([]byte(nil), plane...)
			v.SetFog(plane, cliffW, cliffH)

			var order []image.Point
			v.forEachDrawnTile(func(x, y int) {
				order = append(order, image.Pt(x, y))
			})
			at := -1
			for i, p := range order {
				if p == (image.Point{X: tx, Y: ty}) {
					at = i
					break
				}
			}
			if at < 0 {
				t.Fatalf("fixture %s draw walk does not visit target tile (%d,%d): %v", name, tx, ty, order)
			}

			rec := &recordingTarget{}
			if flat {
				v.drawFlat(rec)
			} else {
				v.drawDisplaced(rec)
			}
			if len(rec.calls) != len(order) {
				t.Fatalf("%s draw submitted %d quads, want draw walk's %d", name, len(rec.calls), len(order))
			}
			got := rec.calls[at].verts
			if len(got) != 4 {
				t.Fatalf("target tile submitted %d vertices, want 4", len(got))
			}
			for i := range got {
				want := base[i]
				if got[i].ColorR != want || got[i].ColorG != want || got[i].ColorB != want {
					t.Errorf("%s corner %d at lattice %v submitted RGB (%v,%v,%v), want pre-fog %v * gain %v = %v",
						name, i, vertices[i], got[i].ColorR, got[i].ColorG, got[i].ColorB, base[i], factors[i], want)
				}
			}
			shroud := &recordingTarget{}
			v.drawShroudTiles(shroud, nil)
			if len(shroud.calls) != len(order) {
				t.Fatal("shroud and terrain use different mesh walks")
			}
			for i, q := range shroud.calls[at].verts {
				if q.ColorA != 1-factors[i] || q.DstX != got[i].DstX || q.DstY != got[i].DstY {
					t.Fatalf("shroud corner %d=%+v, terrain=%+v", i, q, got[i])
				}
			}
			if !bytes.Equal(plane, before) {
				t.Fatalf("%s fog draw rewrote its input plane: got %v, want unchanged %v", name, plane, before)
			}
		})
	}
}

func TestFogRevealKeyTogglesOncePerPress(t *testing.T) {
	load := func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("fog-key", grid(3, 3), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}
	a := newTestApp(t, appRows(3), load)
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, cadenceAt(0))
	if a.Screen() != ScreenMap || a.flow.viewer == nil {
		t.Fatalf("setup: screen = %v, viewer = %v", a.Screen(), a.flow.viewer)
	}
	v := a.flow.viewer
	if v.FogRevealed() {
		t.Fatal("an opened map starts with the reveal off")
	}

	press := func(at int, pressed bool) {
		in := cadenceInput(false, false, false)
		in.Reveal = pressed
		a.step(in, cadenceAt(at))
	}

	press(1, true)
	if !v.FogRevealed() {
		t.Error("the key did not turn the reveal on")
	}
	press(2, true)
	if v.FogRevealed() {
		t.Error("the key did not turn the reveal back off")
	}

	// A HELD key delivers one true and then false, so three frames of
	// holding it act once — F1's own precedent, over F4. The reveal is OFF
	// going into this block (the press above left it there), so one toggle
	// takes it back ON; two more toggles would take it back OFF, which is
	// the failure this asserts against.
	press(3, true)
	press(4, false)
	press(5, false)
	if !v.FogRevealed() {
		t.Error("a held key over three frames toggled more than once (or never toggled at all)")
	}
}
