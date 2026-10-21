package game

import (
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
)

// minimalTown is the smallest description of a first-game town the screen
// plays: a square, the four doors, three room scenes and the school page's
// four actors.
const minimalTown = "testdata/minimal-town.json"

// minimalTownFrame is a small picture for a series member.
func minimalTownFrame() *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			pic.SetRGBA(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 16), B: 90, A: 255})
		}
	}
	return pic
}

// minimalEastMods is the east-town fixture mod with towns/east.json replaced
// by the minimal description as edit leaves it, and the pictures it names,
// drawn by code.
func minimalEastMods(t *testing.T, edit func(d map[string]any)) string {
	t.Helper()
	dir := eastTownMods(t)
	raw, err := os.ReadFile(minimalTown)
	if err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		var d map[string]any
		if err := json.Unmarshal(raw, &d); err != nil {
			t.Fatal(err)
		}
		edit(d)
		if raw, err = json.MarshalIndent(d, "", " "); err != nil {
			t.Fatal(err)
		}
	}
	mod := filepath.Join(dir, "east-town")
	files := map[string][]byte{
		"towns/east.json":   raw,
		"art/square.bmp":    synth.BMP24(eastTownSquare()),
		"art/room.bmp":      synth.BMP24(eastTownSquare()),
		"art/mask.bmp":      synth.BMP8(eastTownMask()),
		"art/frame0000.bmp": synth.BMP24(minimalTownFrame()),
		"art/frame0001.bmp": synth.BMP24(minimalTownFrame()),
	}
	for name, b := range files {
		if err := os.WriteFile(filepath.Join(mod, filepath.FromSlash(name)), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// rooms edits the description's room named name.
func editRoom(d map[string]any, name string, edit func(r map[string]any)) {
	for _, r := range d["rooms"].([]any) {
		if room := r.(map[string]any); room["name"] == name {
			edit(room)
		}
	}
}

// The minimal description layers, the party travels to it, and every room it
// opens draws and ticks: the tavern, shop and school pages and the gates.
func TestMinimalChapterTownTravelsAndOpensEveryRoom(t *testing.T) {
	f, overrides, err := layeredFront(t, minimalEastMods(t, nil), "east-town")
	if err != nil || len(overrides) != 0 {
		t.Fatalf("layer: %v, overrides %v", err, overrides)
	}
	if err := checkEastTown(f, false); err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	for _, c := range []struct {
		row  int
		room townRoom
	}{{0, roomTavern}, {1, roomShop}, {2, roomSchool}, {3, roomGates}} {
		s.Choose(c.row)
		if s.room != c.room {
			t.Fatalf("row %d: the party stands in room %v, want %v", c.row, s.room, c.room)
		}
		s.TavernInteriorActive(true)
		s.ShopInteriorActive(true)
		s.SchoolTrainingActive(true)
		s.AdvanceShopInteriorAnimation()
		switch c.room {
		case roomTavern, roomSchool:
			s.TownSurface()
		case roomShop:
			s.ShopScreen()
		}
		s.Back()
		if !s.AtTownSquare() {
			t.Fatalf("row %d: Back leaves the party in room %v", c.row, s.room)
		}
	}
}

// A description of the first game's towns that the screen cannot play is
// refused where the mod layers it, naming the mod, the file and what is
// missing, instead of stopping the game on entry or leaving the party on
// the map.
func TestChapterTownDescriptionRefusals(t *testing.T) {
	const head = `mod "east-town": data/towns.toml:9: town catalog rom1: town 2 has no usable description: east-town:towns/east.json `
	for _, c := range []struct {
		name string
		edit func(d map[string]any)
		want string
	}{
		{"no set-room on the square", func(d map[string]any) { delete(d["square"].(map[string]any), "enter") },
			`square enter steps do not run hook "set-room", so the party never stands on the square`},
		{"no set-room in a room", func(d map[string]any) { editRoom(d, "shop", func(r map[string]any) { delete(r, "enter") }) },
			`room "shop" enter steps do not run hook "set-room", so the party never enters it`},
		{"no navigate on the gates", func(d map[string]any) {
			editRoom(d, "gates", func(r map[string]any) { r["enter"] = []any{map[string]any{"hook": "set-room"}} })
		}, `room "gates" enter steps do not run hook "navigate", so the gates never open the world map`},
		{"no tavern scene", func(d map[string]any) { editRoom(d, "tavern", func(r map[string]any) { delete(r, "scene") }) },
			`room "tavern" has no scene; the tavern page is drawn from it`},
		{"no shop scene", func(d map[string]any) { editRoom(d, "shop", func(r map[string]any) { delete(r, "scene") }) },
			`room "shop" has no scene; the shop page is drawn from it`},
		{"no school actor", func(d map[string]any) {
			editRoom(d, "school", func(r map[string]any) {
				scene := r["scene"].(map[string]any)
				var kept []any
				for _, a := range scene["actors"].([]any) {
					if a.(map[string]any)["name"] != "diamond" {
						kept = append(kept, a)
					}
				}
				scene["actors"] = kept
			})
		}, `room "school" scene has no actor "diamond" running program "bounce"; the school page drives it`},
	} {
		_, _, err := layeredFront(t, minimalEastMods(t, c.edit), "east-town")
		if err == nil || !strings.Contains(err.Error(), head+c.want) {
			t.Errorf("%s: %v\nwant %q", c.name, err, head+c.want)
		}
	}
}

// A description that names an install art entry, a music track or a sound the
// install does not hold is refused where the mod layers it, naming the field
// and the value; a file of the mod is read separately and a name the install
// holds passes.
func TestModTownDescriptionNamesOnlyWhatTheInstallHolds(t *testing.T) {
	track := GameMusic(base.Profile{}).Tracks[0]
	const head = `mod "east-town": data/towns.toml:11: description towns/east.json: `
	art := func(name, key string) func(d map[string]any) {
		return func(d map[string]any) {
			for _, e := range d["art"].([]any) {
				if entry := e.(map[string]any); entry["name"] == name {
					entry["key"] = key
				}
			}
		}
	}
	sound := func(key string) func(d map[string]any) {
		return func(d map[string]any) {
			editRoom(d, "school", func(r map[string]any) {
				slot := r["scene"].(map[string]any)["sounds"].(map[string]any)["slots"].([]any)[0].(map[string]any)
				slot["key"] = key
			})
		}
	}
	for _, c := range []struct {
		name string
		edit func(d map[string]any)
		want string
	}{
		{"an install art key it lacks", art("square", "graphics/none/zzz.bmp"),
			`art "square": key "graphics/none/zzz.bmp" is not an entry of the install; use "mod:path" for a file of the mod`},
		{"an install art key it holds", art("square", shippedTownArtKeys(townBooks["rom1"].descriptions["rom1"])[0]), ""},
		{"a track it lacks", func(d map[string]any) { d["music"] = map[string]any{"track": "zzz"} },
			`music: track "zzz" is not a track of the install's music`},
		{"a track in a list it lacks", func(d map[string]any) {
			editRoom(d, "gates", func(r map[string]any) {
				r["music"] = map[string]any{"tracks": []any{track, "zzz"}, "by": "member-class"}
			})
		}, `room gates music: track "zzz" is not a track of the install's music`},
		{"a track it holds", func(d map[string]any) { d["music"] = map[string]any{"track": track} }, ""},
		{"a sound it lacks", sound("town/none/zzz.wav"), `room school sounds: slot "rotate" key "town/none/zzz.wav" is not a sound of the install`},
		{"a sound it holds", sound("town/school/rotate.wav"), ""},
	} {
		f := layerFront(t)
		f.SoundBank = &SoundBank{src: f.Archives.Containers.Fork()}
		f.SoundBank.src.Overlay("sfx/town/school/rotate.wav", func() ([]byte, error) { return []byte{0}, nil })
		_, err := f.SetModTowns(runTownMods(t, minimalEastMods(t, c.edit), "east-town").Towns)
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), head+c.want)):
			t.Errorf("%s: %v\nwant %q", c.name, err, head+c.want)
		}
	}
}

func TestModsDocExampleIsTheMinimalDescription(t *testing.T) {
	doc, err := os.ReadFile("../../docs/MODS.md")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/minimal-town.json")
	if err != nil {
		t.Fatal(err)
	}
	norm := func(b []byte) string { return strings.TrimSpace(strings.ReplaceAll(string(b), "\r\n", "\n")) }
	if !strings.Contains(norm(doc), "```json\n"+norm(want)+"\n```") {
		t.Fatal("docs/MODS.md does not carry testdata/minimal-town.json as its example")
	}
}
