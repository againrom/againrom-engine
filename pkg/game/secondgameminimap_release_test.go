package game

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestReleaseSecondGameMinimapMarks reads the second game's first-mission
// minimap at 1024x768 against R2-ENGINE-593: the hero's light blue 2x2 mark,
// a still 2x2 mark at the one-cell signpost structure, a 2x2 mark on a
// hostile creature while the hero is near and none after he leaves, and a
// 4x2 mark on a two-cell mound. Each owner colour is computed here from data:
// entry 164 of owner table (player-table colour & 15) of human.pal, the
// table the first game's marks use (MISSION-085). The loss control is the
// second game's manifest without the minimap lookup: the symbolic minimap
// draws no structure mark.
func TestReleaseSecondGameMinimapMarks(t *testing.T) {
	f := releaseFront(t)
	if f.Archives.Game() != base.GameROM2 {
		t.Skip("the second game's minimap marks")
	}
	f.SetDeterministicFrames(true)
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.speech.player = nil, nil, nil, nil, nil
	app := f.App("second game minimap marks")
	t.Cleanup(app.StopAudio)
	app.SetCutscenes(nil)
	app.Layout(1024, 768)
	enterSecondCampaignMission(t, app)
	step := func(n int) {
		t.Helper()
		for range n {
			advance := app.HeadlessStep
			if app.HeadlessNoticeOpen() {
				advance = func() error { return app.HeadlessKey("enter") }
			}
			if err := advance(); err != nil {
				t.Fatal(err)
			}
		}
	}
	step(40)
	if app.Screen() != ui.ScreenMap || f.live == nil || len(f.live.mission.ids) == 0 {
		t.Fatal("the first mission did not settle on the map")
	}
	mw := f.live
	groups := mw.mission.state.Map.Groups
	ownerColour := func(owner uint32) color.RGBA {
		if owner == 0 || int(owner) > len(groups) {
			t.Fatalf("owner %d has no player-table row", owner)
		}
		c := f.Units.OwnerPalettes[groups[owner-1].Color&15][164]
		return color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff}
	}
	// mark reads the picture over cells: every pixel the colour, and the
	// pixel extent.
	mark := func(cells image.Rectangle, want color.RGBA) (bool, image.Point) {
		t.Helper()
		pic, r, err := app.HeadlessMinimapCells(cells)
		if err != nil {
			t.Fatal(err)
		}
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if pic.RGBAAt(x, y) != want {
					return false, r.Size()
				}
			}
		}
		return true, r.Size()
	}
	cell := func(e sim.Entity) image.Rectangle { return image.Rect(int(e.X), int(e.Y), int(e.X)+1, int(e.Y)+1) }

	hero, ok := mw.entity(mw.mission.ids[0])
	if !ok {
		t.Fatal("the hero has no entity")
	}
	local := hero.Owner
	heroBlue := color.RGBA{R: 82, G: 121, B: 231, A: 0xff}
	if on, size := mark(cell(hero), heroBlue); !on || size != image.Pt(2, 2) {
		t.Errorf("hero mark at %v: drawn=%v size=%v, want light blue 2x2", cell(hero), on, size)
	}

	// The signpost is the map's one-cell structure nearest the hero.
	owners := mw.structureOwnerSlots()
	var post sim.Structure
	best := -1
	var mounds []sim.Structure
	for _, s := range mw.world.Structures() {
		dx, dy := int(s.Col)-int(hero.X), int(s.Row)-int(hero.Y)
		if s.Width == 1 && s.Height == 1 && (best < 0 || dx*dx+dy*dy < best) {
			post, best = s, dx*dx+dy*dy
		}
		if s.Width == 2 && s.Height == 1 && mw.world.Relations().Hostile(local, owners[s.ID]) {
			mounds = append(mounds, s)
		}
	}
	if best < 0 || best > 100 {
		t.Fatalf("no one-cell structure near the hero (distance^2 %d)", best)
	}
	postCells := image.Rect(int(post.Col), int(post.Row), int(post.Col)+1, int(post.Row)+1)
	postColour := ownerColour(owners[post.ID])
	t.Logf("signpost kind %d owner %d at %v colour %v", post.Kind, owners[post.ID], postCells.Min, postColour)
	if on, size := mark(postCells, postColour); !on || size != image.Pt(2, 2) {
		t.Errorf("signpost mark at %v: drawn=%v size=%v, want %v 2x2", postCells.Min, on, size, postColour)
	}

	// The hero walks south-east into the first fight. A hostile creature's
	// mark shows while it is in sight; a mound's mark shows once its cells
	// are in sight.
	start := image.Pt(int(hero.X), int(hero.Y))
	mw.enqueue(uint32(hero.ID), start.X+14, start.Y+3)
	moundMark := func() []string {
		var out []string
		for _, s := range mounds {
			r := image.Rect(int(s.Col), int(s.Row), int(s.Col)+2, int(s.Row)+1)
			if on, size := mark(r, ownerColour(owners[s.ID])); on && size == image.Pt(4, 2) {
				out = append(out, fmt.Sprint(r.Min))
			}
		}
		return out
	}
	var seen sim.Entity
	var shown []string
	found := false
	for n := 0; n < 1500 && (!found || len(shown) == 0); n++ {
		step(1)
		for _, e := range mw.world.Entities() {
			if found || !e.Alive() || !mw.world.Relations().Hostile(local, e.Owner) {
				continue
			}
			if on, size := mark(cell(e), ownerColour(e.Owner)); on {
				if size != image.Pt(2, 2) {
					t.Errorf("creature %d mark size %v, want 2x2", e.ID, size)
				}
				seen, found = e, true
				t.Logf("creature %d owner %d at %v: %v 2x2 mark", e.ID, e.Owner, cell(e).Min, ownerColour(e.Owner))
			}
		}
		if len(shown) == 0 {
			shown = moundMark()
		}
	}
	if !found {
		t.Fatal("no hostile creature's mark showed in the first fight")
	}
	if len(shown) == 0 {
		t.Error("no two-cell mound showed a 4x2 mark near the first fight")
	}
	t.Logf("mounds with a 4x2 mark: %v", shown)

	// The hero holds a place the creatures come to; the creature's mark goes
	// once its body is past the first corpse stage (MISSION-085).
	mw.enqueue(uint32(hero.ID), start.X+13, start.Y)
	gone := false
	for n := 0; n < 3000 && !gone; n++ {
		e, ok := mw.entity(seen.ID)
		if ok && e.Alive() {
			step(1)
			continue
		}
		if !ok {
			gone = true
			break
		}
		_, other, err := app.HeadlessMinimapMarkAt(int(e.X), int(e.Y))
		if err != nil {
			t.Fatal(err)
		}
		on, _ := mark(cell(e), ownerColour(e.Owner))
		if gone = !on && !other; !gone {
			step(1)
		}
	}
	if !gone {
		t.Errorf("creature %d's mark stayed: it did not die, or its body kept the mark", seen.ID)
	}

	// The hero walks back to the start: the mounds' marks go with the sight,
	// and the signpost's mark is where it was.
	mw.enqueue(uint32(hero.ID), start.X, start.Y)
	for n := 0; n < 1500; n++ {
		step(1)
		if h, ok := mw.entity(hero.ID); ok && image.Pt(int(h.X), int(h.Y)) == start {
			break
		}
	}
	step(25)
	if h, ok := mw.entity(hero.ID); !ok || !h.Alive() {
		t.Fatal("the hero did not come back")
	}
	if left := moundMark(); len(left) != 0 {
		t.Errorf("mound marks %v stayed after the hero left", left)
	}
	if on, _ := mark(postCells, postColour); !on {
		t.Error("the signpost mark is not at its place with the hero back at the start")
	}
}
