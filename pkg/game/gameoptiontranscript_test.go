package game

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

var updateGameOptionTranscript = flag.Bool("update-gameoption-transcript", false,
	"rewrite testdata/gameoptiontranscript.txt from the current code")

// TestGameOptionTranscriptIsUnchanged records, for both games, every Game
// Options answer the game gives: the new-profile values off and on the map,
// each value set off the map and on the map (the page's values, the stored
// keys, the session flags, the live map's flags, the queued commands, the
// pending options a SAVE keeps), a cold restart and a fresh map door after
// it, and the page's captions with and without install text. The recorded
// transcript is the answer before the option rows existed.
func TestGameOptionTranscriptIsUnchanged(t *testing.T) {
	var b strings.Builder
	for _, g := range []base.Game{base.GameROM1, base.GameROM2} {
		writeGameOptionTranscript(t, &b, g)
	}
	got := b.String()
	path := filepath.Join("testdata", "gameoptiontranscript.txt")
	if *updateGameOptionTranscript {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := 0; i < max(len(gl), len(wl)); i++ {
			var g, w string
			if i < len(gl) {
				g = gl[i]
			}
			if i < len(wl) {
				w = wl[i]
			}
			if g != w {
				t.Fatalf("transcript line %d:\n got %s\nwant %s", i+1, g, w)
			}
		}
	}
}

func writeGameOptionTranscript(t *testing.T, b *strings.Builder, g base.Game) {
	t.Helper()
	m := worldFixtureMap()
	archives := &Archives{Base: base.Match{Profile: base.Profile{Game: g}}}
	newFront := func(path string) *FrontEnd {
		f := &FrontEnd{InstallResources: InstallResources{Archives: archives},
			PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
		f.LoadOptions()
		return f
	}
	openMap := func(f *FrontEnd) *mapWorld {
		mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
		if err := mw.world.RestoreCurrentPlayers([]sim.SavedGroupPlayer{{ID: 1, Slot: 0}, {ID: 2, Slot: sim.SelfSlot}}, []sim.PlayerParticipant{{PlayerID: 1, Value: 1}, {PlayerID: 2}}, true); err != nil {
			t.Fatal(err)
		}
		mw.view.SetPathfinding(f.showPathfinding)
		mw.view.SetGraphicsOptions(f.graphics)
		if err := f.applyFreshGameOptions(mw, nil); err != nil {
			t.Fatal(err)
		}
		f.live = mw
		return mw
	}
	session := func(f *FrontEnd) string {
		return fmt.Sprintf("graphics=%+v wimpy=%d pathfinding=%v", f.graphics, f.wimpyMode, f.showPathfinding)
	}
	stored := func(f *FrontEnd) string {
		all, err := f.Options.readAll()
		if err != nil {
			return "unreadable"
		}
		keys := make([]string, 0, len(all))
		for k := range all {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			keys[i] = k + "=" + all[k]
		}
		return strings.Join(keys, " ")
	}
	live := func(mw *mapWorld) string {
		damage, _ := mw.view.DamageNumerals()
		healing, healingSet := mw.world.AutoHealing(sim.SelfSlot)
		pending := pendingGameOptions(mw.pending)
		return fmt.Sprintf("timeflow=%v health=%v damage=%v path=%v graphics=%+v healing=%d/%v autocast=%#x formation=%d record=%v pending=%v options=%v valid=%v",
			mw.view.TimeFlow(), mw.view.HealthBarsShown(), damage, mw.view.PathfindingShown(), mw.view.GraphicsOptions(),
			healing, healingSet, uint32(mw.world.AutoCast(sim.SelfSlot)), mw.world.FormationMode(sim.SelfSlot),
			mw.applicationState != nil, mw.pending, pending, validatePendingGameOptions(pending) == nil)
	}
	fmt.Fprintf(b, "game %s column %q\n", g, archives.Base.Profile.Edition().Options)
	dir := t.TempDir()
	fresh := newFront(filepath.Join(dir, "new.txt"))
	fmt.Fprintf(b, "new off=%v %s stored[%s]\n", fresh.gameOptionValues(false), session(fresh), stored(fresh))
	mw := openMap(fresh)
	fmt.Fprintf(b, "new on=%v %s live[%s]\n", fresh.gameOptionValues(true), session(fresh), live(mw))
	for _, o := range ui.GameOptions() {
		for v := 0; v < o.Choices(); v++ {
			path := filepath.Join(dir, fmt.Sprintf("off-%d-%d.txt", o, v))
			f := newFront(path)
			err := f.setGameOption(false, o, v)
			fmt.Fprintf(b, "%s=%d off err=%v off=%v %s stored[%s]\n", o.Key(), v, err != nil, f.gameOptionValues(false), session(f), stored(f))
			cold := newFront(path)
			mw := openMap(cold)
			fmt.Fprintf(b, "%s=%d cold off=%v on=%v %s live[%s]\n", o.Key(), v, cold.gameOptionValues(false), cold.gameOptionValues(true), session(cold), live(mw))

			on := newFront(filepath.Join(dir, fmt.Sprintf("on-%d-%d.txt", o, v)))
			mw = openMap(on)
			err = on.setGameOption(true, o, v)
			fmt.Fprintf(b, "%s=%d on err=%v on=%v off=%v %s stored[%s] live[%s]\n", o.Key(), v, err != nil, on.gameOptionValues(true), on.gameOptionValues(false), session(on), stored(on), live(mw))
		}
	}
	// A failed store write changes nothing on the live map.
	broken := newFront(filepath.Join(dir, "broken.txt"))
	mw = openMap(broken)
	broken.Options.Path = t.TempDir()
	for _, o := range ui.GameOptions() {
		err := broken.setGameOption(true, o, o.Offers(archives.Base.Profile.Edition().Options)[0])
		fmt.Fprintf(b, "%s broken err=%v on=%v %s live[%s]\n", o.Key(), err != nil, broken.gameOptionValues(true), session(broken), live(mw))
	}
	writeGameOptionCaptions(t, b, g)
}

// writeGameOptionCaptions records the page's captions without an install and
// over a synthetic install whose dialogs line n reads "D<n>" and whose patch
// line n reads "P<n>".
func writeGameOptionCaptions(t *testing.T, b *strings.Builder, g base.Game) {
	t.Helper()
	words := func(f *FrontEnd) ui.GameOptionWords {
		a := ui.NewApp("options", nil, nil, nil)
		f.wireGameOptions(a)
		return a.GameOptionWords()
	}
	plain := &FrontEnd{InstallResources: InstallResources{Archives: nil}}
	fmt.Fprintf(b, "words plain %q\n", words(plain))
	dir := t.TempDir()
	lines := func(prefix string) []byte {
		var s strings.Builder
		for i := 0; i < 200; i++ {
			fmt.Fprintf(&s, "%s%d\r\n", prefix, i)
		}
		// The second game's automatic-casting label line ends in its own
		// colon, which the page drops.
		return []byte(strings.Replace(s.String(), "\r\nD170\r\n", "\r\n D170: \r\n", 1))
	}
	main := filepath.Join(dir, MainArchive)
	if err := os.WriteFile(main, synth.Archive([]synth.File{{Path: "text/dialogs.txt", Data: lines("D")}}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, patchArchive), synth.Archive([]synth.File{{Path: "patch.txt", Data: lines("P")}}), 0o600); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open([]string{main}, nil)
	if err != nil {
		t.Fatal(err)
	}
	installed := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: dir, Containers: fsys, Base: base.Match{Profile: base.Profile{Game: g}}}}}
	fmt.Fprintf(b, "words installed %q\n", words(installed))
}
