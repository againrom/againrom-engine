package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func openDifficultyChargen(t *testing.T, f *FrontEnd) *ui.App {
	t.Helper()
	app := f.App("1082 difficulty")
	app.Layout(640, 480)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() == ui.ScreenPicker {
		if err := app.HeadlessActivate("@first"); err != nil {
			t.Fatal(err)
		}
	}
	state, ok := app.HeadlessChargenState()
	if !ok || state.Difficulty != 2 {
		t.Fatalf("new game default=%+v showing=%v", state, ok)
	}
	return app
}

func TestReleaseDifficultyCampaignThroughProductionUI(t *testing.T) {
	f := releaseFront(t)
	for _, level := range []int{1, 2, 3} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			app := openDifficultyChargen(t, f)
			// Easy and Hard also cross cmd/againrom's process-start entry:
			// OpenChargen + NewGameOpener, without the map-picker gate.
			if level != 2 {
				if err := app.OpenChargen(ui.NewChargen(f.ChargenSetup()), func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(10, res), nil }); err != nil {
					t.Fatal(err)
				}
			}
			if err := headlessCreateCharacter(app, HeadlessCharacter{Name: "Witness", Difficulty: []string{"easy", "normal", "hard"}[level-1]}); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenMap || int(f.Difficulty) != level || f.liveMission != 10 {
				t.Fatalf("creation: screen=%s level=%d mission=%d", app.Screen(), f.Difficulty, f.liveMission)
			}
			assertDifficultyPlacements(t, f, 10, level)
			// A native mission load starts with an intentionally different live
			// setting. Saved bytes stay exact; non-byte-form Ghost input must use
			// the candidate setting (DIV-037, DIV-041), not the stale live one.
			s, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			ghost := f.live.world.Ghost()
			b, err := EncodeSave(s, "level")
			if err != nil {
				t.Fatal(err)
			}
			back, _, err := DecodeSave(b)
			if err != nil {
				t.Fatal(err)
			}
			f.Difficulty = mapload.Difficulty(4 - level)
			oldTown, oldWorld := f.Town, f.live.world
			open, town, err := f.Restore(back)
			if err != nil || town {
				t.Fatalf("prepare mission: %v %v", town, err)
			}
			if f.Town != oldTown || f.live.world != oldWorld || f.Difficulty != mapload.Difficulty(4-level) {
				t.Fatal("native candidate committed before adoption")
			}
			if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
				t.Fatal(err)
			}
			got, err := f.live.world.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(s.World, got) {
				t.Fatal("native resume modified/rescaled world bytes")
			}
			if f.live.world.Ghost() != ghost || int(f.Difficulty) != level {
				t.Fatal("native candidate inherited stale live difficulty")
			}
			// Invalid candidate must not alter any session-owned difficulty/world.
			oldWorld = f.live.world
			oldTown = f.Town
			bad := back
			bad.Difficulty = 99
			if _, _, err := f.Restore(bad); err == nil {
				t.Fatal("invalid level accepted")
			}
			if f.live.world != oldWorld || f.Town != oldTown || int(f.Difficulty) != level {
				t.Fatal("invalid candidate mutated current game")
			}
			// Production campaign transition: 10 -> 20 -> town -> native town
			// restore -> 30. Completion is direct, not a claim of combat victory.
			f.FinishMission(10, f.liveParty, f.live.world, f.live.mission.ids)
			if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(20)(); err != nil {
				t.Fatal(err)
			}
			assertDifficultyPlacements(t, f, 20, level)
			f.FinishMission(20, f.liveParty, f.live.world, f.live.mission.ids)
			if !f.Town.Open() {
				t.Fatal("mission 20 did not reach town")
			}
			s, _, err = f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			b, err = EncodeSave(s, "town")
			if err != nil {
				t.Fatal(err)
			}
			back, _, err = DecodeSave(b)
			if err != nil {
				t.Fatal(err)
			}
			f.Difficulty = 0
			if _, town, err := f.Restore(back); err != nil || !town {
				t.Fatalf("town restore: %v %v", town, err)
			}
			if int(f.Difficulty) != level {
				t.Fatal("town restore lost difficulty")
			}
			if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(30)(); err != nil {
				t.Fatal(err)
			}
			assertDifficultyPlacements(t, f, 30, level)
		})
	}
	for _, level := range []int{1, 3} {
		saved := originalDifficultyFixture(t, 10, uint32(level))
		// The original import's Head value beats the diagnostic caller flag.
		ms, _, err := loadOriginalMission(f, saved)
		if err != nil {
			t.Fatal(err)
		}
		open, town, err := f.RestoreOriginal(saved)
		if err != nil || town {
			t.Fatalf("original map restore: %v %v", town, err)
		}
		if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
			t.Fatal(err)
		}
		if int(f.Difficulty) != level || f.live.world.Ghost() != ms.World.Ghost() {
			t.Fatal("original difficulty was ignored by an import entry")
		}
		assertDifficultyPlacements(t, f, 10, level)
	}
}

// Expected scaling is computed here, never through mapload.Adjust. Normal
// placement values and party identity come from the same installed map/table;
// this witnesses difficulty transport, not the table decoder's separate proof.
func assertDifficultyPlacements(t *testing.T, f *FrontEnd, mission, level int) {
	t.Helper()
	m := releaseMissionMap(t, f, mission)
	normal, err := StartMissionFrom(m, "difficulty witness", mission, f.Table, mapload.DifficultyNormal, f.liveParty)
	if err != nil {
		t.Fatal(err)
	}
	base, got := normal.World.Entities(), f.live.world.Entities()
	if len(got) != len(base) {
		t.Fatalf("entity count %d vs %d", len(got), len(base))
	}
	units, humans, party := 0, 0, 0
	for i, e := range got {
		want := base[i]
		if e.ID != want.ID {
			t.Fatal("difficulty changed entity identity")
		}
		if i < len(m.Units) && m.Units[i].ClassID >= 26 {
			units++
			if level == 1 {
				want.MaxHP = want.MaxHP * 66 / 100
			}
			if level == 3 {
				want.MaxHP = want.MaxHP * 3 / 2
				want.ToHit += 50
				want.Defence += 50
			}
			want.HP = want.MaxHP
			if hp := m.Units[i].CurrentHP; hp != -1 {
				want.HP = int32(hp)
			}
		} else if i < len(m.Units) {
			humans++
		} else {
			party++
		}
		if e.HP != want.HP || e.MaxHP != want.MaxHP || e.ToHit != want.ToHit || e.Defence != want.Defence {
			t.Fatalf("m%d level%d entity%d class%d HP/max/tohit/def=%d/%d/%d/%d want %d/%d/%d/%d", mission, level, e.ID, e.Class, e.HP, e.MaxHP, e.ToHit, e.Defence, want.HP, want.MaxHP, want.ToHit, want.Defence)
		}
	}
	if units == 0 || humans == 0 || party == 0 {
		t.Fatalf("missing class arm: units=%d humans=%d party=%d", units, humans, party)
	}
	t.Logf("m%d level%d: %d Units scaled once, %d Humans and %d party entities unchanged", mission, level, units, humans, party)
}

func TestReleaseDifficultyLevelsInstalledArtAndPointer(t *testing.T) {
	f := releaseFront(t)
	// Expectations name installed entries directly. They do not read the
	// production Loaded Levels array or production geometry/state helpers.
	read := func(path string) *bmp.Image {
		b, err := f.Archives.Containers.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		p, err := bmp.Decode(b)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	bg := read("graphics/interface/chrgen/precreate/mainarea.bmp")
	origins := [3]image.Point{{60, 196}, {0, 110}, {48, 65}}
	for i := 0; i < 3; i++ {
		pic := read(fmt.Sprintf("graphics/interface/chrgen/precreate/levels/level%don.bmp", i))
		best, bestAt := int64(1<<62), image.Point{}
		for y := 30; y < 230; y++ {
			for x := 0; x < 150; x++ {
				var score int64
				for py := 0; py < pic.Height; py += 6 {
					for px := 0; px < pic.Width; px += 6 {
						c := pic.At(px, py)
						if c == (bmp.Color{}) {
							continue
						}
						d := bg.At(x+px, y+py)
						for _, delta := range []int{int(c.R) - int(d.R), int(c.G) - int(d.G), int(c.B) - int(d.B)} {
							if delta < 0 {
								delta = -delta
							}
							score += int64(delta)
						}
					}
				}
				if score < best {
					best, bestAt = score, image.Pt(x, y)
				}
			}
		}
		t.Logf("installed art fit level%d: origin=%v score=%d", i, bestAt, best)
		if bestAt != origins[i] {
			t.Errorf("level%d measured fit %v vs chosen %v", i, bestAt, origins[i])
		}
	}
	app := f.App("1082 installed overlap")
	app.Layout(640, 480)
	setup := f.ChargenSetup()
	// This witness compares the static level art layer pixel-for-pixel.
	precreate, art := *setup.PreCreate, *setup.PreCreate.Art
	art.Sparkles = nil
	precreate.Art, setup.PreCreate = &art, &precreate
	c := ui.NewChargen(setup)
	c.CloseTip()
	if err := app.OpenChargen(c, func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(10, res), nil }); err != nil {
		t.Fatal(err)
	}
	// Independently decoded art and mask identify the visible owner, including
	// the overlap excluded from the Levels-only pixel comparison below.
	hero := read("graphics/interface/chrgen/precreate/heroes/ffon.bmp")
	queen := read("graphics/interface/chrgen/precreate/levels/level2on.bmp")
	maskBytes, err := f.Archives.Containers.ReadFile("graphics/interface/chrgen/precreate/mask.bmp")
	if err != nil {
		t.Fatal(err)
	}
	mask, err := bmp.DecodePaletted(maskBytes)
	if err != nil {
		t.Fatal(err)
	}
	overlap := 0
	for y := 166; y < 217; y++ {
		for x := 124; x < 148; x++ {
			if hero.At(x-124, y-166) == (bmp.Color{}) || queen.At(x-48, y-65) == (bmp.Color{}) || mask.ColorIndexAt(x, y) != 100 {
				continue
			}
			overlap++
			if owner, ok := ui.PreCreateControlAt(c, image.Pt(x, y)); !ok || owner != "choice 2" {
				t.Errorf("visible hero overlap at (%d,%d) owned by %q, want choice 2", x, y, owner)
			}
		}
	}
	if overlap != 175 {
		t.Fatalf("independent installed overlap census=%d want175", overlap)
	}
	frame, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	if got := frame.RGBAAt(147, 174); got != (color.RGBA{40, 31, 18, 255}) {
		t.Fatalf("visible hero counterexample pixel=%v", got)
	}
	for _, action := range []string{"press", "release"} {
		if err := app.HeadlessPointer(action, 147, 174); err != nil {
			t.Fatal(err)
		}
	}
	if c.PreChoice() != 2 || c.Difficulty() != 2 {
		t.Fatalf("visible hero click: choice=%d difficulty=%d, want choice2 Normal", c.PreChoice(), c.Difficulty())
	}
	t.Logf("%d visible hero overlap pixels owned by choice2; (147,174) selects female fighter, retains Normal", overlap)
	for _, tc := range []struct {
		level int
		point image.Point
	}{{1, image.Pt(86, 226)}, {3, image.Pt(107, 129)}, {2, image.Pt(30, 164)}} {
		if err := app.HeadlessPointer("press", tc.point.X, tc.point.Y); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessPointer("release", tc.point.X, tc.point.Y); err != nil {
			t.Fatal(err)
		}
		state, _ := app.HeadlessChargenState()
		if state.Difficulty != tc.level {
			t.Fatalf("pointer at %v selected %d want%d", tc.point, state.Difficulty, tc.level)
		}
		if c.PreChoice() != 2 {
			t.Fatalf("level%d center changed hero choice to %d", tc.level, c.PreChoice())
		}
		// Move to background so the selected picture must be on.bmp, not lon.
		if err := app.HeadlessPointer("move", 630, 470); err != nil {
			t.Fatal(err)
		}
		frame, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		pic := read(fmt.Sprintf("graphics/interface/chrgen/precreate/levels/level%don.bmp", tc.level-1))
		origin := origins[tc.level-1]
		checked := 0
		for y := 0; y < pic.Height; y++ {
			for x := 0; x < pic.Width; x++ {
				c := pic.At(x, y)
				if c == (bmp.Color{}) {
					continue
				}
				p := origin.Add(image.Pt(x, y))
				want := color.RGBA{c.R, c.G, c.B, 255}
				// The female-fighter portrait is painted after Levels (TOWN-223).
				// Its independently named origin is (124,166); exclude that overlap.
				if p.X >= 124 && p.Y >= 166 {
					continue
				}
				if got := frame.RGBAAt(p.X, p.Y); got != want {
					t.Fatalf("level%d source pixel %v got%v want%v", tc.level, p, got, want)
				}
				checked++
			}
		}
		t.Logf("pointer-selected level%d: %d installed pixels", tc.level, checked)
		if dir := os.Getenv("AGAINROM_DIFFICULTY_WITNESS"); dir != "" {
			out, err := os.Create(filepath.Join(dir, fmt.Sprintf("difficulty-%d.png", tc.level)))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(out, frame); err != nil {
				out.Close()
				t.Fatal(err)
			}
			if err := out.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}
