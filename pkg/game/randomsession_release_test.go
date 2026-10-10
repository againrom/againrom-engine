package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/random"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The random session witness plays one headless session from a launch seed:
// the town square, tavern, shop and school with their idle extras under
// Random Order music, then a new game whose mission takes a command voice, a
// Lightning bolt and a Meteor Storm, a SAVE, a LOAD of that SAV and further
// ticks. It records every frame hash and sound request and the World hash.
// No draw source is injected; every presentation stream is the session's.

const (
	sessionLightning = 13
	sessionMeteor    = 21
)

type sessionRun struct {
	lines []string
	world []uint64
	save  []byte
	// state is the World's stream state at SAVE.
	state uint64
}

func playRandomSession(t *testing.T, seed uint64, original bool) sessionRun {
	t.Helper()
	root := releaseRoot(t)
	f, err := NewFrontEnd(root)
	if err != nil {
		t.Fatal(err)
	}
	if f.Base().Profile.Limits.NoCharacterGeneration {
		t.Skip("the base opens without generation; the session walks the first game's town and chargen")
	}
	cleanupFrontAudio(t, f)
	f.SetRandomLaunch(seed, true, original)
	t.Cleanup(func() { ui.SetOriginalItemStars(false) })
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), optionsFileName)}
	if err := f.Options.SetMusicPreferences(ui.MusicPreferences{Enabled: true, RandomOrder: true}); err != nil {
		t.Fatal(err)
	}
	f.SetDeterministicFrames(true)
	now := time.Unix(5000, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.PersistenceContext.tipsOff = true
	sounds := &traceSounds{}
	f.SoundPlayer, f.SpeechPlayer, f.AmbientPlayer = sounds, sounds, sounds
	f.SoundBank = OpenSounds(f.Archives.Root)
	fighter := f.ChargenParty(ui.ChargenResult{Name: "Session fighter", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}})
	mage := f.ChargenParty(ui.ChargenResult{Name: "Session mage", Choices: []int{1, 1, 0}, Stats: []int{20, 20, 30, 30}})
	mage[0].StartingHero = false
	f.Carried = append(fighter, mage[0])
	f.arriveInTown()
	f.Town.gold = 100000
	points, off := traceMaskPoints(t, f)

	a := f.App("random session")
	t.Cleanup(a.StopAudio)
	a.Layout(640, 480)
	a.SetMusic(traceMusicSource{inner: f.MusicBank, log: sounds}, traceMusicDevice{log: sounds}, f.randomService().Stream(random.Music))
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.sav", Label: "Town"}} },
		func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	tr := &townTrace{t: t, f: f, a: a, now: &now, sound: sounds}
	tr.record("town")
	idle := image.Pt(320, 120)
	tr.hover(off, 41*time.Millisecond, 200)
	for _, door := range []byte{0x80, 0x90, 0xc0} {
		tr.hover(points[door][0], 34*time.Millisecond, 3)
		tr.click(points[door][0])
		tr.closeDialogue()
		tr.hover(idle, 47*time.Millisecond, 300)
		tr.key("escape")
		tr.closeDialogue()
		tr.hover(off, 41*time.Millisecond, 20)
	}
	tr.expect("square")

	// The new game: chargen's opener over mission 41, the session's fresh
	// seed committed with it.
	res := ui.ChargenResult{Difficulty: 2, Name: "Session mage", Choices: []int{1, 1, 0}, Stats: []int{20, 20, 30, 30}}
	if err := a.OpenMission(f.NewGameOpener(41, res)); err != nil {
		t.Fatal(err)
	}
	for n := 0; a.HeadlessNoticeOpen() && n < 16; n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	mw := f.live
	hero := mw.mission.ids[0]
	sessionGrant(t, mw, hero, sessionLightning, sessionMeteor)
	step := func(kind string, n int) {
		for i := 0; i < n; i++ {
			if a.HeadlessNoticeOpen() {
				if err := a.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
			} else if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			live := f.live
			e, _ := live.entity(hero)
			pix, _, err := live.view.HeadlessMapFrame(image.Pt(int(e.X), int(e.Y)), 320, 224)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(pix.Pix)
			tr.lines = append(tr.lines, fmt.Sprintf("%s tick=%d world=%016x map=%x snd=[%s]",
				kind, live.world.Tick(), live.world.Hash(), sum[:8], sounds.take()))
		}
	}
	if err := a.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	step("select", replyClockFrames)
	x, y, err := a.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	step("command", 30)
	victim := launchIssue(t, mw, hero)
	step(fmt.Sprintf("bolt@%d", victim), 40)
	h, _ := mw.entity(hero)
	mw.pending = append(mw.pending, sim.CastAt(hero, sessionMeteor, sim.CellPoint{X: h.X + 3, Y: h.Y}))
	step("meteor", 80)

	snap, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := f.ExportCurrentSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	out := sessionRun{save: saved, world: []uint64{f.live.world.Hash()}, state: f.live.world.RandomState()}
	open, town, err := f.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatalf("LOAD of the session's SAV: town %t, %v", town, err)
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	out.world = append(out.world, f.live.world.Hash())
	step("loaded", 120)
	out.world = append(out.world, f.live.world.Hash())
	out.lines = tr.lines
	return out
}

// sessionGrant writes spells into the hero's book and fills his mana.
func sessionGrant(t *testing.T, mw *mapWorld, hero sim.EntityID, spells ...int) {
	t.Helper()
	e, _ := mw.entity(hero)
	book, known := e.Book, e.KnownSpells
	book.State = sim.BookPresent
	for _, spell := range spells {
		rule, ok := mw.world.Spell(uint32(spell))
		if !ok {
			t.Fatalf("no installed row for spell %d", spell)
		}
		book.Slots[spell-1] = sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
		known |= 1 << spell
	}
	if err := mw.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: hero, KnownSpells: known, Book: book}}); err != nil {
		t.Fatal(err)
	}
	launchRefill(t, mw, hero)
}

func releaseRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the session witness needs a lawful install")
	}
	return root
}

func sessionDiff(a, b []string) string {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return fmt.Sprintf("line %d:\n  %s\n  %s", i, a[i], b[i])
		}
	}
	if len(a) != len(b) {
		return fmt.Sprintf("%d lines against %d", len(a), len(b))
	}
	return ""
}

func countSounds(lines []string, prefix string) int {
	n := 0
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) || strings.Contains(l, prefix) {
			n++
		}
	}
	return n
}

// TestReleaseRandomSessionReplaysFromItsSeed plays the session twice from one
// seed and once from another. One seed replays every frame, sound request,
// World hash and SAV byte; another seed moves presentation draws and leaves
// the World where it was.
func TestReleaseRandomSessionReplaysFromItsSeed(t *testing.T) {
	root := releaseRoot
	first := playRandomSession(t, 7, false)
	again := playRandomSession(t, 7, false)
	if d := sessionDiff(first.lines, again.lines); d != "" {
		t.Fatalf("one seed played two sessions: %s", d)
	}
	if fmt.Sprint(first.world) != fmt.Sprint(again.world) || string(first.save) != string(again.save) {
		t.Fatal("one seed wrote two Worlds or two SAVs")
	}
	if first.world[0] != first.world[1] {
		t.Fatalf("LOAD changed the World hash: %016x then %016x", first.world[0], first.world[1])
	}
	for _, want := range []string{"music-start", "unit-command", "bolt@", "meteor", "loaded"} {
		if countSounds(first.lines, want) == 0 {
			t.Errorf("the session never reached %q", want)
		}
	}
	other := playRandomSession(t, 8, false)
	if sessionDiff(first.lines, other.lines) == "" {
		t.Fatal("a different seed drew the same presentation")
	}
	if fmt.Sprint(first.world) != fmt.Sprint(other.world) {
		t.Fatalf("a different seed moved the World: %x against %x", first.world, other.world)
	}
	if dir := os.Getenv("AGAINROM_SESSION_WITNESS_DIR"); dir != "" {
		for name, lines := range map[string][]string{"seed7": first.lines, "seed8": other.lines} {
			path := filepath.Join(dir, filepath.Base(root(t))+"-"+name+".txt")
			if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Logf("%d lines replayed; World %016x at SAVE and after LOAD, %016x after the continuation", len(first.lines), first.world[0], first.world[2])
}

// loadSessionSave LOADs saved into a fresh front end launched in the given
// mode and runs ticks, answering the loaded World's mode and state and its
// hash after the ticks.
func loadSessionSave(t *testing.T, saved []byte, original bool, ticks int) (random.Mode, uint64, uint64) {
	t.Helper()
	f, err := NewFrontEnd(releaseRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	cleanupFrontAudio(t, f)
	f.SetRandomLaunch(99, true, original)
	t.Cleanup(func() { ui.SetOriginalItemStars(false) })
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatalf("LOAD: town %t, %v", town, err)
	}
	a := f.App("random session load")
	t.Cleanup(a.StopAudio)
	a.Layout(640, 480)
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	mode, state := f.live.world.RandomMode(), f.live.world.RandomState()
	for i := 0; i < ticks; i++ {
		f.live.tick()
	}
	return mode, state, f.live.world.Hash()
}

// TestReleaseOriginalRandomSession plays the session on the original
// generator: the World runs on the shared stream, its SAV records the mode
// the seed and the reseed count, a LOAD continues through the load path's
// reseeds at that count, and a SAV of either mode loads in the other.
func TestReleaseOriginalRandomSession(t *testing.T) {
	first := playRandomSession(t, 7, true)
	doc, err := sav.DecodeDocumentData(first.save)
	if err != nil {
		t.Fatal(err)
	}
	leaf, ok, err := sav.ReadNativeSession(doc.State)
	if err != nil || !ok || leaf.Mode != uint32(random.Original) || leaf.Seed != 7 || leaf.Reseeds == 0 {
		t.Fatalf("original SAV session leaf %+v present %t: %v", leaf, ok, err)
	}
	// The LOAD reseeds at the saved count (SESS-083, DIV-2730); the mission
	// screen's own entry draws, its music among them, then continue that
	// stream.
	mode, state, _ := loadSessionSave(t, first.save, true, 60)
	probe := random.MSVC{State: random.MissionLoadState(7, leaf.Reseeds)}
	draws := 0
	for ; draws < 4096 && uint64(probe.State) != state; draws++ {
		probe.Rand()
	}
	if mode != random.Original || uint64(probe.State) != state {
		t.Fatalf("original SAV loaded as mode %d state %#x, not a continuation of the load reseeds at the saved count %d", mode, state, leaf.Reseeds)
	}
	seeded := playRandomSession(t, 7, false)
	toSeeded, _, _ := loadSessionSave(t, first.save, false, 60)
	toOriginal, _, _ := loadSessionSave(t, seeded.save, true, 60)
	if toSeeded != random.Seeded || toOriginal != random.Original {
		t.Fatalf("cross-mode LOADs ran in modes %d and %d", toSeeded, toOriginal)
	}
	t.Logf("original World %016x at SAVE, reseed count %d; the LOAD continued the reseeded stream after %d entry draws; cross-mode LOADs ran 60 ticks each", first.world[0], leaf.Reseeds, draws)
}

// TestReleaseOriginalRandomMissionOne runs the first mission on the original
// generator for a fixed number of ticks. On a game whose edition has no
// evidence for its original generator the switch runs the default mode, says
// so once, and the first mission runs seeded (DIV-2748).
func TestReleaseOriginalRandomMissionOne(t *testing.T) {
	f, err := NewFrontEnd(releaseRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	cleanupFrontAudio(t, f)
	wantMode := random.Original
	notice := f.SetRandomLaunch(1, true, true)
	if f.Base().Profile.Edition().OriginalGenerator == "" {
		wantMode = random.Seeded
		if notice == "" || f.randomService().Mode() != random.Seeded || f.randomService().LaunchSettings().Mode != random.Seeded {
			t.Fatalf("the switch on a game without original-generator evidence: notice %q, mode %d", notice, f.randomService().Mode())
		}
	} else if notice != "" {
		t.Fatalf("the first game's switch gave a notice: %q", notice)
	}
	t.Cleanup(func() { ui.SetOriginalItemStars(false) })
	f.SetDeterministicFrames(true)
	a := f.App("original mission one")
	t.Cleanup(a.StopAudio)
	a.Layout(640, 480)
	res := ui.ChargenResult{Difficulty: 2, Name: "Original", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}}
	if err := a.OpenMission(f.NewGameOpener(f.Base().Profile.Mission(), res)); err != nil {
		t.Fatal(err)
	}
	if f.live.world.RandomMode() != wantMode {
		t.Fatalf("the first mission started in mode %d, want %d", f.live.world.RandomMode(), wantMode)
	}
	const ticks = 2000
	for i := 0; i < ticks; i++ {
		f.live.tick()
	}
	if f.live.world.Tick() < ticks {
		t.Fatalf("the first mission stopped at tick %d", f.live.world.Tick())
	}
	t.Logf("the first mission ran %d ticks in mode %d; World %016x", ticks, wantMode, f.live.world.Hash())
}
