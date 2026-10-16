package game

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestReleaseMinimapLootDotsFollowTheOption opens the install's first
// mission, puts one ground sack on a visible cell with no unit and one on an
// unseen cell, and reads the framed minimap through the headless route. The
// loot colour stands on the visible sack's cell with the option on and nowhere
// on the unseen one; with the option off neither shows it. The World hash and
// the SAV bytes are equal with the option on and off. AGAINROM_LOOT_WITNESS_DIR
// receives the numbers as JSON when set; it writes no picture.
func TestReleaseMinimapLootDotsFollowTheOption(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.speech.player = nil, nil, nil, nil, nil
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	app := f.App("loot dots")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.DirectNewGame(f.Base().Profile.Mission())); err != nil {
		t.Fatal(err)
	}
	for n := 0; app.HeadlessNoticeOpen() && n < 32; n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	mw := f.live
	if app.Screen() != ui.ScreenMap || mw == nil {
		t.Fatal("mission did not reach the map")
	}
	plane := mw.fog.project()
	cols := mw.fog.cols
	occupied := map[image.Point]bool{}
	for _, e := range mw.world.Entities() {
		occupied[image.Pt(int(e.X), int(e.Y))] = true
	}
	for _, s := range mw.world.Sacks() {
		occupied[image.Pt(int(s.X), int(s.Y))] = true
	}
	var hero image.Point
	for _, e := range mw.world.Entities() {
		if e.Owner == sim.SelfSlot {
			hero = image.Pt(int(e.X), int(e.Y))
			break
		}
	}
	pick := func(state uint8, near bool) image.Point {
		best, bestD := image.Point{-1, -1}, 1<<30
		for i, v := range plane {
			c := image.Pt(i%cols, i/cols)
			if v != state || occupied[c] {
				continue
			}
			d := (c.X-hero.X)*(c.X-hero.X) + (c.Y-hero.Y)*(c.Y-hero.Y)
			if near && d < bestD || !near && (best.X < 0 || d > bestD) {
				best, bestD = c, d
			}
		}
		return best
	}
	seen := pick(ui.FogVisible, true)
	if seen.X < 0 || mw.world.Purse(sim.SelfSlot) < 7 {
		t.Fatal("no visible cell or no gold to drop", seen, mw.world.Purse(sim.SelfSlot))
	}
	mw.pending = append(mw.pending, sim.DropGold(sim.SelfSlot, 7, sim.CellPoint{X: int32(seen.X), Y: int32(seen.Y)}))
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	placed := false
	for _, s := range mw.world.Sacks() {
		c := image.Pt(int(s.X), int(s.Y))
		if c == seen {
			placed = true
		}
	}
	if !placed {
		t.Fatal("the dropped sack is not on the requested cell")
	}
	// An existing sack on a cell the fog hides is the unseen control, when the
	// mission has one.
	unseen := image.Pt(-1, -1)
	for _, s := range mw.world.Sacks() {
		c := image.Pt(int(s.X), int(s.Y))
		if i := c.Y*cols + c.X; i >= 0 && i < len(plane) && mw.fog.project()[i] != ui.FogVisible {
			unseen = c
		}
	}
	// Both games export the current state as SAV. When a second-game mission
	// cannot export in this route the witness compares the World hash alone and
	// says so.
	savable := true
	state := func() (uint64, []byte) {
		if !savable {
			return mw.world.Hash(), nil
		}
		snap, _, err := f.Snapshot(true)
		if err == nil {
			var raw []byte
			if raw, err = f.ExportCurrentSave(snap, "loot"); err == nil {
				return mw.world.Hash(), raw
			}
		}
		if f.Archives.Game() == base.GameROM1 {
			t.Fatal(err)
		}
		t.Logf("second-game SAV export unavailable in this route (%v); World hash only", err)
		savable = false
		return mw.world.Hash(), nil
	}
	read := func(on bool) (*image.RGBA, bool, bool) {
		v := 0
		if on {
			v = 1
		}
		if err := f.setGameOption(true, ui.GameOptionLootDots, v); err != nil {
			t.Fatal(err)
		}
		pic, err := app.HeadlessMinimap()
		if err != nil {
			t.Fatal(err)
		}
		a, err := app.HeadlessMinimapLootAt(seen.X, seen.Y)
		if err != nil {
			t.Fatal(err)
		}
		b := false
		if unseen.X >= 0 {
			if b, err = app.HeadlessMinimapLootAt(unseen.X, unseen.Y); err != nil {
				t.Fatal(err)
			}
		}
		return pic, a, b
	}
	onPic, onSeen, onUnseen := read(true)
	hashOn, savOn := state()
	offPic, offSeen, offUnseen := read(false)
	hashOff, savOff := state()
	if !onSeen {
		t.Error("option on: the visible sack has no loot dot")
	}
	if onUnseen {
		t.Error("option on: the unseen sack has a loot dot")
	}
	if offSeen || offUnseen {
		t.Error("option off: a loot dot is drawn")
	}
	if bytes.Equal(onPic.Pix, offPic.Pix) {
		t.Error("the option changed no minimap pixel")
	}
	if hashOn != hashOff || !bytes.Equal(savOn, savOff) {
		t.Error("the option changed the World hash or the SAV bytes")
	}
	t.Logf("sack on %v (visible) and %v (unseen, -1 when the mission has none); on: %v/%v, off: %v/%v; World hash %x on and off; SAV %d bytes equal",
		seen, unseen, onSeen, onUnseen, offSeen, offUnseen, hashOn, len(savOn))
	if dir := os.Getenv("AGAINROM_LOOT_WITNESS_DIR"); dir != "" && filepath.IsAbs(dir) {
		raw, _ := json.MarshalIndent(map[string]any{"visibleSack": seen, "unseenSack": unseen,
			"onVisible": onSeen, "onUnseen": onUnseen, "offVisible": offSeen, "offUnseen": offUnseen,
			"worldHash": hashOn, "savBytes": len(savOn), "savEqual": bytes.Equal(savOn, savOff)}, "", "  ")
		name := filepath.Join(dir, filepath.Base(f.Archives.Root)+".json")
		if err := os.MkdirAll(dir, 0o700); err == nil {
			_ = os.WriteFile(name, raw, 0o600)
		}
	}
}

// TestReleaseSecondGameLootDotsLeaveTheSaveUntouched enters the second game's
// first campaign mission through its town route, draws the minimap with the
// option on and off and compares the World hash and the exported current SAV
// bytes. The mission need not hold a sack; the loot colour itself is proved by
// the first-mission witness above. Loss control: an empty export fails.
func TestReleaseSecondGameLootDotsLeaveTheSaveUntouched(t *testing.T) {
	f := releaseFront(t)
	if f.Archives.Game() != base.GameROM2 {
		t.Skip("second-game current SAV producer")
	}
	f.SetDeterministicFrames(true)
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.speech.player = nil, nil, nil, nil, nil
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.LoadOptions()
	app := f.App("second game loot save")
	t.Cleanup(app.StopAudio)
	app.SetCutscenes(nil)
	app.Layout(1024, 768)
	enterSecondCampaignMission(t, app)
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, f.live)
	capture := func(on bool) (*image.RGBA, uint64, []byte) {
		v := 0
		if on {
			v = 1
		}
		if err := f.setGameOption(true, ui.GameOptionLootDots, v); err != nil {
			t.Fatal(err)
		}
		pic, err := app.HeadlessMinimap()
		if err != nil {
			t.Fatal(err)
		}
		snap, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snap, "loot option")
		if err != nil {
			t.Fatal(err)
		}
		return pic, f.live.world.Hash(), raw
	}
	_, onHash, on := capture(true)
	_, offHash, off := capture(false)
	if onHash != offHash || !bytes.Equal(on, off) || len(on) == 0 {
		t.Fatal("the option reaches the World hash or the SAV", onHash, offHash, len(on), len(off))
	}
	t.Logf("World hash %x on and off; SAV %d bytes equal", onHash, len(on))
}
