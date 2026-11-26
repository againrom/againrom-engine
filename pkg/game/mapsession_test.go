package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/a2c"
	"againrom/pkg/formats/alch"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
	"againrom/pkg/words"
)

// sessionCase is one game's front end, the loose map its map choice lists,
// one stored character file and what the entry must show of it.
type sessionCase struct {
	name   string
	front  func(t *testing.T) *FrontEnd
	mapAt  string
	mapB   func(t *testing.T) []byte
	dir    string
	file   string
	record func(t *testing.T) []byte
	stats  [4]int32
	purse  uint32
	xp     [data.SkillSlots]int32
	female bool
}

func sessionCases() []sessionCase {
	return []sessionCase{{
		name: base.ROM1EN,
		front: func(t *testing.T) *FrontEnd {
			f := missionFrontEnd(t)
			profile, _ := base.Find(base.ROM1EN)
			f.Archives.Base = base.Match{Profile: profile}
			return f
		},
		mapAt: "arena.alm",
		mapB: func(t *testing.T) []byte {
			b := synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: "Arena",
				Units: []synth.ALMUnit{{X: 0x0280, Y: 0x01c0}}})
			binary.LittleEndian.PutUint32(b[alm.FileHeaderSize+alm.RecordHeaderSize+alm.MetaWord70:], 2)
			return b
		},
		dir: "rom1", file: "char0000.chr",
		record: func(t *testing.T) []byte {
			b, err := alch.Write(alch.Character{Name: []byte("Stored"), Flags: alch.FlagShort, Stats: [4]uint32{31, 27, 22, 26}})
			if err != nil {
				t.Fatal(err)
			}
			return b
		},
		stats: [4]int32{31, 27, 22, 26},
	}, {
		name: base.ROM2EN,
		front: func(t *testing.T) *FrontEnd {
			return secondMissionFront(t)
		},
		mapAt: "duel.alm",
		mapB:  func(t *testing.T) []byte { return secondMissionMapOf(40, 40, secondMissionContent{}) },
		dir:   "rom2", file: "10.a2c",
		record: func(t *testing.T) []byte {
			c := a2c.Character{
				A:       a2c.A{ID0: 1, ID1: 0, ListKey: a2c.ListKeyAny, Name: []byte("Stored"), Unit: 2 << 6},
				Kills:   make([]byte, a2c.KillsSize),
				Hotkeys: make([]byte, a2c.HotkeysSize),
				E:       a2c.E{Money: 777, Base: [4]uint8{31, 27, 22, 26}, SkillXP: [5]uint32{0, 0, 2400, 0, 90}},
			}
			b, err := a2c.Write(c)
			if err != nil {
				t.Fatal(err)
			}
			return b
		},
		stats: [4]int32{31, 27, 22, 26}, purse: 777, female: true,
		xp: [data.SkillSlots]int32{data.SkillBlade + 2: 2400, data.SkillBlade + 4: 90},
	}}
}

// sessionApp wires c's front end with its loose map and stored character to
// a headless application showing the character list.
func sessionApp(t *testing.T, c sessionCase) (*FrontEnd, *ui.App, string) {
	t.Helper()
	f := c.front(t)
	if f.Words.Engine.Text("mapchoice.title") == "" {
		f.Words.Engine = words.For("english")
	}
	maps := t.TempDir()
	if err := os.WriteFile(filepath.Join(maps, c.mapAt), c.mapB(t), 0o600); err != nil {
		t.Fatal(err)
	}
	loose, err := vfs.Open(nil, []string{maps})
	if err != nil {
		t.Fatal(err)
	}
	dm, err := DirMaps(maps, loose)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Loose, f.Maps = loose, BuildMapList(dm, nil)
	store := CharacterStore{Dir: filepath.Join(t.TempDir(), "characters")}
	if err := os.MkdirAll(filepath.Join(store.Dir, c.dir), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir, c.dir, c.file), c.record(t), 0o600); err != nil {
		t.Fatal(err)
	}
	f.SetDeterministicFrames(true)
	app := f.App("session")
	app.Layout(640, 480)
	f.ConfigureCharacters(app, store)
	app.OpenCharacters()
	return f, app, filepath.Join(store.Dir, c.dir)
}

func sessionKey(t *testing.T, app *ui.App, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if err := app.HeadlessKey(k); err != nil {
			t.Fatal(err)
		}
	}
}

// Accepting the character row opens the game's map choice; Cancel returns to
// the character list on that row; Ok enters the chosen map with the stored
// character as the party, holding its statistics, purse and experience.
func TestCharacterRowEntersAMultiplayerMapWithTheStoredCharacter(t *testing.T) {
	for _, c := range sessionCases() {
		t.Run(c.name, func(t *testing.T) {
			f, app, stored := sessionApp(t, c)
			before := snapshotDir(t, stored)
			if !app.CharacterList() || len(app.HeadlessRows()) != 2 {
				t.Fatalf("setup: character list %v rows %d", app.CharacterList(), len(app.HeadlessRows()))
			}
			sessionKey(t, app, "enter")
			if rows := app.HeadlessRows(); app.CharacterList() || len(rows) != 1 {
				t.Fatalf("the character row left the list %v with %d rows, want the one map: %s",
					!app.CharacterList(), len(rows), app.HeadlessMessage())
			}
			sessionKey(t, app, "escape")
			if !app.CharacterList() || app.HeadlessListSelection() != 0 {
				t.Fatalf("Cancel left list %v selection %d, want the character row", app.CharacterList(), app.HeadlessListSelection())
			}
			sessionKey(t, app, "enter", "enter")
			if app.Screen() != ui.ScreenMap || f.live == nil {
				t.Fatalf("Ok on the map left %v: %s", app.Screen(), app.HeadlessMessage())
			}
			if got := f.live.mission.state.Address; got != c.mapAt {
				t.Fatalf("entered %q, want the loose %q", got, c.mapAt)
			}
			party := f.LiveParty()
			if len(party) != 1 || party[0].Name != "Stored" {
				t.Fatalf("party %d %v, want the stored character alone", len(party), party)
			}
			h := party[0].Hero
			if got := [4]int32{h.Body, h.Reaction, h.Mind, h.Spirit}; got != c.stats {
				t.Fatalf("statistics %v, want %v", got, c.stats)
			}
			if got := f.live.world.Purse(sim.SelfSlot); got != c.purse {
				t.Fatalf("purse %d, want %d", got, c.purse)
			}
			hero := secondMissionActor(t, f.live.world)
			if want := data.FigureDirFor(false, c.female); party[0].Mage || party[0].FigureDir != string(want) {
				t.Fatalf("mage %v figure %q, want a fighter of %q", party[0].Mage, party[0].FigureDir, want)
			}
			if c.xp != ([data.SkillSlots]int32{}) && hero.SkillXP != c.xp {
				t.Fatalf("experience %v, want %v", hero.SkillXP, c.xp)
			}
			if f.liveMission != 0 || !reflect.DeepEqual(snapshotDir(t, stored), before) {
				t.Fatalf("the entry is campaign mission %d or rewrote the store", f.liveMission)
			}
			if _, _, err := f.Snapshot(true); err == nil {
				t.Fatal("SAVE was offered on the session's map")
			}
		})
	}
}

// A campaign's new game opens the same world it opened before a character
// row could enter a map: the hashes are the base engine's.
func TestCampaignNewGameWorldIsUnchanged(t *testing.T) {
	for _, c := range []struct {
		name  string
		front func(t *testing.T) *FrontEnd
		want  uint64
	}{
		{base.ROM1EN, missionFrontEnd, 0xa20a27160800cec1},
		{base.ROM2EN, secondMissionFront, 0xa9b355532c0155c6},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := c.front(t)
			f.SetDeterministicFrames(true)
			res := ui.ChargenResult{Name: "Campaign", Choices: []int{1, 0, 2}, Stats: []int{30, 25, 20, 25}}
			if _, _, _, _, _, _, _, _, _, _, err := f.NewGameOpener(10, res)(); err != nil {
				t.Fatal(err)
			}
			if got := f.live.world.Hash(); got != c.want || f.liveMission != 10 {
				t.Fatalf("mission %d world hash %#x, want %#x", f.liveMission, got, c.want)
			}
		})
	}
}

// The same stored character entering the same map twice gives equal world
// hashes.
func TestMultiplayerMapEntryIsDeterministic(t *testing.T) {
	for _, c := range sessionCases() {
		t.Run(c.name, func(t *testing.T) {
			var hashes [2]uint64
			for i := range hashes {
				f, app, _ := sessionApp(t, c)
				sessionKey(t, app, "enter", "enter")
				if f.live == nil {
					t.Fatalf("run %d entered no map: %s", i, app.HeadlessMessage())
				}
				hashes[i] = f.live.world.Hash()
			}
			if hashes[0] != hashes[1] {
				t.Fatalf("hashes %x and %x differ", hashes[0], hashes[1])
			}
		})
	}
}
