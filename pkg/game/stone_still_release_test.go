package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type stoneStillPicture struct {
	Frame    int
	Mirror   bool
	Kind     string
	Geometry [6]float64
	Position [2]float64
}

type stoneStillTick struct {
	Tick                  uint64
	PhaseUS               int
	Remaining             uint16
	WorldHash, LoadedHash string
	Cell, Step            image.Point
	Transit               int
	Facing                uint8
	Picture, Loaded       stoneStillPicture
}

func stoneStillFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SoundPlayer, f.speech.player, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil, nil
	f.SetDeterministicFrames(true)
	return f
}

func stoneStillCold(t *testing.T, store SaveStore) *FrontEnd {
	t.Helper()
	f := stoneStillFront(t)
	a := f.App("Stone Curse cold LOAD")
	a.Layout(1024, 768)
	f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
	if err := a.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	rows := a.HeadlessRows()
	if a.Screen() != ui.ScreenLoad || len(rows) != 1 {
		t.Fatal("cold LOAD must list the one written SAV", a.Screen(), rows)
	}
	if err := a.HeadlessActivate(rows[0].Text); err != nil || a.Screen() != ui.ScreenMap {
		t.Fatal("cold LOAD refused the written SAV", err, a.Screen(), a.HeadlessMessage())
	}
	return f
}

func stoneStillDraw(t *testing.T, f *FrontEnd, id sim.EntityID, phase int) stoneStillPicture {
	t.Helper()
	mw := f.live
	entity := swingDraw(t, mw, id)
	if entity.Art == nil || entity.Frame == nil {
		t.Fatal("Stone Curse witness target has no installed unit art")
	}
	frame := -1
	for index, source := range entity.Art.TierFrames(mw.tiers[id]) {
		if source == entity.Frame || mw.ownerFrame(entity.Art, source, entity.Owner) == entity.Frame {
			frame = index
			break
		}
	}
	if frame < 0 {
		t.Fatal("Stone Curse witness frame is absent from the target's installed sheet")
	}
	mw.view.SetPhase(phase, mw.clock.Period())
	draws, err := mw.view.HeadlessArtDraws()
	if err != nil {
		t.Fatal(err)
	}
	var out stoneStillPicture
	found := 0
	for _, draw := range draws {
		if draw.Frame != entity.Frame || draw.Kind != "sprite" && draw.Kind != "stone" {
			continue
		}
		found++
		out.Frame, out.Mirror, out.Kind = frame, entity.Mirror, draw.Kind
		for row := range 2 {
			for col := range 3 {
				out.Geometry[row*3+col] = draw.Geometry.Element(row, col)
			}
		}
		x0, y0 := draw.Geometry.Apply(0, 0)
		x1, y1 := draw.Geometry.Apply(float64(draw.Pixels.Bounds().Dx()), float64(draw.Pixels.Bounds().Dy()))
		out.Position = [2]float64{math.Min(x0, x1), math.Min(y0, y1)}
	}
	if found != 1 {
		t.Fatalf("target frame reached %d production sprite submissions, want one", found)
	}
	return out
}

func stoneStillRemaining(w *sim.World, id sim.EntityID) uint16 {
	for _, effect := range w.ActiveEffects() {
		if effect.Target == id && effect.Spell == 20 {
			return effect.Remaining
		}
	}
	return 0
}

func stoneStillWitnessOutput(t *testing.T, f *FrontEnd, name string, rows []stoneStillTick, raw []byte) {
	t.Helper()
	output := os.Getenv("AGAINROM_STONE_WITNESS_DIR")
	if output == "" {
		return
	}
	dir, err := effectRimOutputPath(f.Archives.Root, output)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	trace, err := json.MarshalIndent(struct {
		Root, Case string
		Mission    int
		SAVSHA256  string
		Ticks      []stoneStillTick
	}{f.Archives.Root, name, 101, fmt.Sprintf("%x", sha256.Sum256(raw)), rows}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), trace, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("trace=%s SAV=%s sha256=%x bytes=%d", filepath.Join(dir, name+".json"), filepath.Join(dir, name+".sav"), sha256.Sum256(raw), len(raw))
}

func TestReleaseStoneCurseStaysStillThroughSAVColdLoadAndExpiry(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, name := range []string{"mid-step", "standing"} {
		t.Run(name, func(t *testing.T) {
			f := stoneStillFront(t)
			party := hasteParty()
			party[0].Hero = data.Hero{Body: 60, Reaction: 60, Spirit: 100}
			party[0].KnownSpells = 1 << 20
			party[1].Hero = data.Hero{Body: 26, Reaction: 15}
			a := f.App("Stone Curse witness")
			a.Layout(1024, 768)
			if err := a.OpenMission(f.MissionOpenerWith(101, party)); err != nil {
				t.Fatal(err)
			}
			w := f.live.world
			w.SetRawSessionHead([48]byte{16: 0x80, 17: 0x96, 18: 0x98, 32: 0x10, 33: 0x27})
			caster, target := f.live.mission.ids[0], f.live.mission.ids[1]
			initial := releaseEntity(t, f.live, target)
			inspectionCentre(f.live, int(initial.X), int(initial.Y))
			f.live.view.Camera().SetZoom(1)
			var trace []stoneStillTick
			cursePhases := make(map[uint64]uint8)
			capture := func(front, loaded *FrontEnd, phase int) stoneStillTick {
				t.Helper()
				e := releaseEntity(t, front.live, target)
				d := swingDraw(t, front.live, target)
				row := stoneStillTick{Tick: front.live.world.Tick(), PhaseUS: phase,
					Remaining: stoneStillRemaining(front.live.world, target), WorldHash: fmt.Sprintf("%016x", front.live.world.Hash()),
					Cell: image.Pt(int(e.X), int(e.Y)), Step: d.Step, Transit: int(e.Transit), Facing: e.DrawnFacing(),
					Picture: stoneStillDraw(t, front, target, phase)}
				if loaded != nil {
					row.LoadedHash, row.Loaded = fmt.Sprintf("%016x", loaded.live.world.Hash()), stoneStillDraw(t, loaded, target, phase)
				}
				if row.Remaining > 0 {
					mask := uint8(1)
					if phase == front.live.clock.Period()/2 {
						mask = 2
					} else if phase == front.live.clock.Period() {
						mask = 4
					}
					cursePhases[row.Tick] |= mask
				}
				trace = append(trace, row)
				return row
			}
			capture(f, nil, 0)
			goalX := initial.X + 2
			if name == "mid-step" {
				f.live.enqueue(uint32(target), int(goalX), int(initial.Y))
			}
			f.live.attackOrCast(uint32(caster), uint32(target), 20, int(initial.X), int(initial.Y), false)
			for tick := 0; tick < 256 && !w.HasEffectSpell(target, 20); tick++ {
				f.live.tick()
				capture(f, nil, 0)
				if name == "mid-step" && !w.HasEffectSpell(target, 20) {
					e := releaseEntity(t, f.live, target)
					if e.X == goalX && e.Y == initial.Y && e.Transit == 0 {
						if goalX == initial.X {
							goalX = initial.X + 2
						} else {
							goalX = initial.X
						}
						f.live.enqueue(uint32(target), int(goalX), int(initial.Y))
					}
				}
			}
			remaining := stoneStillRemaining(w, target)
			if remaining != 160 {
				t.Fatal("controlled installed Stone Curse must attach with 160 ticks remaining", remaining, w.BookSpellRefusal(caster, target, 20))
			}
			attachedAt := w.Tick()
			caught := releaseEntity(t, f.live, target)
			if name == "mid-step" && caught.Transit == 0 && !w.ActorMotionActive(target) {
				t.Fatal("moving loss control was not caught mid-step", caught)
			}
			baseline := stoneStillDraw(t, f, target, 0)
			phases := []int{0, f.live.clock.Period() / 2, f.live.clock.Period()}
			var cold *FrontEnd
			var raw []byte
			frameChanges, positionChanges, hashChanges, loadedDrawChanges := 0, 0, 0, 0
			var firstChange, firstHash string
			checkHash := func(row stoneStillTick) {
				if row.LoadedHash != "" && row.WorldHash != row.LoadedHash {
					hashChanges++
					if firstHash == "" {
						firstHash = fmt.Sprintf("tick%d %s/%s", row.Tick, row.WorldHash, row.LoadedHash)
					}
				}
			}
			captureHeld := func() {
				for _, phase := range phases {
					row := capture(f, cold, phase)
					if row.Picture.Frame != baseline.Frame || row.Picture.Mirror != baseline.Mirror || row.Picture.Kind != "stone" {
						frameChanges++
					}
					if row.Picture.Geometry != baseline.Geometry {
						positionChanges++
					}
					if firstChange == "" && row.Picture != baseline {
						firstChange = fmt.Sprintf("tick%d phase%d: frame%d mirror%t position%v -> frame%d mirror%t position%v", row.Tick, phase,
							baseline.Frame, baseline.Mirror, baseline.Position, row.Picture.Frame, row.Picture.Mirror, row.Picture.Position)
					}
					checkHash(row)
					if cold != nil && (row.Picture != row.Loaded || row.Remaining != stoneStillRemaining(cold.live.world, target)) {
						loadedDrawChanges++
					}
				}
			}
			expired := false
			for elapsed := 0; elapsed <= 600; elapsed++ {
				left := stoneStillRemaining(w, target)
				if left == 0 {
					expired = true
					checkHash(capture(f, cold, 0))
					break
				}
				if elapsed == int(remaining)/2 {
					captureHeld()
					var store SaveStore
					store, _, raw = menuSAVE(t, f, a, OriginalStore{})
					cold = stoneStillCold(t, store)
					adoptLoadedAllocator(t, f.live.world, cold.live.world)
					*cold.live.view.Camera() = *f.live.view.Camera()
				}
				captureHeld()
				f.live.tick()
				if cold != nil {
					cold.live.tick()
				}
			}
			if cold == nil {
				t.Fatal("mid-curse ordinary SAV checkpoint was never reached")
			}
			if !expired || stoneStillRemaining(cold.live.world, target) != 0 {
				t.Error("paired Stone Curse did not expire within 600 ticks")
			}
			if len(cursePhases) != int(remaining) || w.Tick() != attachedAt+uint64(remaining) {
				t.Errorf("curse coverage: %d unique ticks, attachment %d expiry %d; want %d consecutive ticks", len(cursePhases), attachedAt, w.Tick(), remaining)
			}
			for offset := range int(remaining) {
				tick := attachedAt + uint64(offset)
				if cursePhases[tick] != 7 {
					t.Errorf("curse tick %d captured phase mask %d, want all three phases", tick, cursePhases[tick])
				}
			}
			thawed := releaseEntity(t, f.live, target)
			if name == "standing" {
				for _, front := range []*FrontEnd{f, cold} {
					front.live.enqueue(uint32(target), int(thawed.X+1), int(thawed.Y))
				}
			}
			resumed := false
			for range 128 {
				f.live.tick()
				cold.live.tick()
				row := capture(f, cold, 0)
				checkHash(row)
				e := releaseEntity(t, f.live, target)
				resumed = resumed || e.X != thawed.X || e.Y != thawed.Y || row.Picture.Position != baseline.Position
			}
			stoneStillWitnessOutput(t, f, name, trace, raw)
			t.Logf("mission101 target%d curseTicks%d observedCurseTicks%d caughtTransit%d standingFrame%d at%v samples%d frameChanges%d positionChanges%d pairedHashChanges%d loadedDrawChanges%d resumed%t SAVbytes%d SAVsha256%x firstChange=%s firstHash=%s",
				target, remaining, len(cursePhases), caught.Transit, baseline.Frame, baseline.Position, len(trace), frameChanges, positionChanges,
				hashChanges, loadedDrawChanges, resumed, len(raw), sha256.Sum256(raw), firstChange, firstHash)
			if frameChanges != 0 || positionChanges != 0 || hashChanges != 0 || loadedDrawChanges != 0 || !resumed {
				t.Error("Stone Curse changed its drawn picture, cold continuation, or post-expiry motion")
			}
		})
	}
}
