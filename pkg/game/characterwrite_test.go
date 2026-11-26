package game

import (
	"encoding/binary"
	"math/bits"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/a2c"
	"againrom/pkg/formats/alch"
	"againrom/pkg/ui"
)

// newAlchFile is a new first-game file from the hero format page: the short
// form with flag bit 0, the name and the four creation statistics opening the
// first 28-byte block at +0x54, every other byte zero.
func newAlchFile(name string, stats [4]int) []byte {
	f := alchFile(name, 0x8c)
	for i, v := range stats {
		binary.LittleEndian.PutUint32(f[0x54+4*i:], uint32(v))
	}
	return f
}

func pageSum(plain []byte) uint32 {
	var sum uint32
	for _, c := range plain {
		sum = sum*2 + uint32(c)
	}
	return sum
}

// pageSection is a section from the rom2-a2c format page under the key the
// low 15 bits of the plain checksum give.
func pageSection(tag uint32, plain []byte) []byte {
	sum := pageSum(plain)
	key := uint16(sum & 0x7fff)
	f := binary.LittleEndian.AppendUint32(nil, tag)
	f = binary.LittleEndian.AppendUint32(f, uint32(len(plain)))
	f = binary.LittleEndian.AppendUint32(f, uint32(key)<<16)
	f = binary.LittleEndian.AppendUint32(f, sum)
	reg := uint32(key)<<16 | uint32(key)
	for i, c := range plain {
		f = append(f, c^uint8(reg>>16))
		reg <<= 1
		if i%16 == 15 {
			reg |= uint32(key)
		}
	}
	return f
}

// newA2CFile is a new second-game file from the claims: A with the pair
// (1, id1), the list value, the name and unit byte 0; B's 2560 zero bytes as
// 20 repeat packets of 127 and one of 20; C's 72 zero bytes; E with money
// 1000 and the four statistics, each stored field the reader's chain solved
// by hand, a zero pad before each field whose key bit is set.
func newA2CFile(id1 uint32, name string, stats [4]int) []byte {
	a := make([]byte, 0x34)
	binary.LittleEndian.PutUint32(a, 1)
	binary.LittleEndian.PutUint32(a[4:], id1)
	binary.LittleEndian.PutUint32(a[8:], 0x529c0291)
	copy(a[0x0c:], name)
	f := binary.LittleEndian.AppendUint32(nil, 0x04507989)
	f = append(f, pageSection(0xAAAAAAAA, a)...)
	b := binary.LittleEndian.AppendUint32(nil, 2560)
	for range 20 {
		b = append(b, 0xff, 0)
	}
	f = append(f, pageSection(0x55555555, append(b, 0x80|20, 0))...)
	f = append(f, pageSection(0x40A40A40, make([]byte, 72))...)
	plain := make([]byte, 0x34)
	binary.LittleEndian.PutUint32(plain[0x10:], 1000)
	v := [4]uint8{uint8(stats[0]), uint8(stats[1]), uint8(stats[2]), uint8(stats[3])}
	copy(plain[0x14:], v[:])
	sum := pageSum(plain)
	key := uint16(sum & 0x7fff)
	var zero uint32
	stored := []uint32{0x1529251, zero - 0x13141516, zero - 0xabcdef, 0x17ff12aa, 1000 + 0x21524542,
		uint32(v[0] - 1000*0x11&0xff), uint32(v[1] - v[0]*3), uint32(v[2] - v[0] - v[1]*5), uint32(v[3] - v[0]*7 - v[2]*9),
		0x10121974, 0, 0xdadedade, 0, 0, 0, 0}
	pads := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 14, 13, 9, 10, 11, 12, 13}
	f = binary.LittleEndian.AppendUint32(f, 0x41392521)
	f = binary.LittleEndian.AppendUint32(f, uint32(0x34+bits.OnesCount16(key)))
	f = binary.LittleEndian.AppendUint32(f, uint32(key)<<16)
	f = binary.LittleEndian.AppendUint32(f, sum)
	for i, s := range stored {
		if key&(1<<pads[i]) != 0 {
			f = append(f, 0)
		}
		if i >= 5 && i <= 8 {
			f = append(f, uint8(s))
		} else {
			f = binary.LittleEndian.AppendUint32(f, s)
		}
	}
	return f
}

// startStats are the generator's start statistics, Body, Reaction, Mind and
// Spirit, which an accept with no change stores.
func startStats(f *FrontEnd) [4]int {
	setup := f.ChargenSetup()
	var out [4]int
	for i, row := range []int{chargenStatBody, chargenStatReaction, chargenStatMind, chargenStatSpirit} {
		out[i] = setup.Stats[row].Start
	}
	return out
}

// In each game the list's New row opens the generator; Cancel returns to
// the list and writes nothing, and Accept writes the claims' file under the
// first name the store's builder gives, then shows the list again with that
// file listed and selected. A second Accept takes the next name and leaves
// the first file as it was.
func TestNewCharacterIsGeneratedStoredAndListed(t *testing.T) {
	for _, tc := range []struct {
		id, dir string
		files   [2]string
		want    func(at int, name string, stats [4]int) []byte
	}{
		{base.ROM1EN, "rom1", [2]string{"char0000.chr", "char0001.chr"},
			func(_ int, name string, stats [4]int) []byte { return newAlchFile(name, stats) }},
		{base.ROM2EN, "rom2", [2]string{"10.a2c", "11.a2c"},
			func(at int, name string, stats [4]int) []byte { return newA2CFile(uint32(at), name, stats) }},
	} {
		t.Run(tc.id, func(t *testing.T) {
			profile := t.TempDir()
			store := CharacterStore{Dir: filepath.Join(profile, "characters")}
			f := storeFront(t, tc.id)
			f.Archives.Containers = townTextFS(t, nil)
			app := ui.NewApp("t", nil, nil, nil)
			f.ConfigureCharacters(app, store)
			app.OpenCharacters()
			key := func(name string) {
				t.Helper()
				if err := app.HeadlessKey(name); err != nil {
					t.Fatal(err)
				}
			}
			key("enter")
			if app.Screen() != ui.ScreenChargen {
				t.Fatalf("the New row opened %v, want the generator", app.Screen())
			}
			key("escape")
			if _, err := os.Stat(store.Dir); app.Screen() != ui.ScreenLoad || !os.IsNotExist(err) {
				t.Fatalf("Cancel left %v and the store %v", app.Screen(), err)
			}
			stats := startStats(f)
			for at, file := range tc.files {
				for range at {
					key("down")
				}
				key("enter")
				key("enter")
				if app.Screen() != ui.ScreenLoad {
					t.Fatalf("Accept left %v: %s", app.Screen(), app.HeadlessMessage())
				}
				b, err := os.ReadFile(filepath.Join(store.Dir, tc.dir, file))
				if want := tc.want(at, fallbackHeroName, stats); err != nil || string(b) != string(want) {
					t.Fatalf("%s: %x, %v\nwant %x", file, b, err, want)
				}
				var rows []string
				for _, r := range app.HeadlessRows() {
					rows = append(rows, r.Text)
				}
				if want := slices.Repeat([]string{fallbackHeroName}, at+1); !slices.Equal(rows, append(want, "New Character")) {
					t.Fatalf("rows %q", rows)
				}
				listed, _ := f.characterFiles(store).List()
				if sel := app.HeadlessListSelection(); sel < 0 || sel >= len(listed) || listed[sel].Name != file {
					t.Fatalf("row %d selected of %v, want %s", sel, listed, file)
				}
			}
			first, err := os.ReadFile(filepath.Join(store.Dir, tc.dir, tc.files[0]))
			if want := tc.want(0, fallbackHeroName, stats); err != nil || string(first) != string(want) {
				t.Fatalf("the second Accept changed %s", tc.files[0])
			}
		})
	}
}

// The campaign's starts keep their setup: the second game's accept shows the
// town and the first game's opens the mission; the character start shows
// neither and opens no map.
func TestCharacterStartLeavesTheCampaignStarts(t *testing.T) {
	for _, tc := range []struct {
		id   string
		town bool
	}{{base.ROM1EN, false}, {base.ROM2EN, true}} {
		f := storeFront(t, tc.id)
		f.Archives.Containers = townTextFS(t, nil)
		if got := f.ChargenSetup().AcceptShowsTown; got != tc.town {
			t.Errorf("%s: the campaign accept shows the town %v, want %v", tc.id, got, tc.town)
		}
		if f.chargenSetup(generatorStartCharacter).AcceptShowsTown {
			t.Errorf("%s: the character start shows the town", tc.id)
		}
		store := CharacterStore{Dir: filepath.Join(t.TempDir(), "characters")}
		open, err := f.generatorBegin(generatorStartCharacter, generatorTarget{store: store})(ui.ChargenResult{Name: "Aeryn"})
		if open != nil || err != nil {
			t.Errorf("%s: the character start opens %v, %v", tc.id, open != nil, err)
		}
	}
}

// The store reads back what the generator made: the name, the statistics
// and, in the second game, the unit row of each sex and class in the
// claims' order Start_MF, Start_MM, Start_FF, Start_FM. A taken name is
// skipped and its file kept; with no free name the write is refused.
func TestStoredCharacterReadsBackAsGenerated(t *testing.T) {
	res := func(sex, class int) ui.ChargenResult {
		r := ui.ChargenResult{Name: "Aeryn", Choices: make([]int, 3), Stats: []int{21, 22, 23, 24}}
		r.Choices[chargenChoiceSex], r.Choices[chargenChoiceClass] = sex, class
		r.Stats[chargenStatBody], r.Stats[chargenStatReaction] = 21, 22
		r.Stats[chargenStatMind], r.Stats[chargenStatSpirit] = 23, 24
		return r
	}
	profile := t.TempDir()
	store := CharacterStore{Dir: filepath.Join(profile, "characters")}
	taken := filepath.Join(store.Dir, "rom1", "char0000.chr")
	writeCharacters(t, filepath.Dir(taken), []storedFile{{"char0000.chr", []byte("kept"), 0}})
	first := storeFront(t, base.ROM1EN)
	name, err := first.storeCharacter(store, res(1, 1))
	if err != nil || name != "char0001.chr" {
		t.Fatalf("first game stored %q, %v", name, err)
	}
	if b, err := os.ReadFile(taken); err != nil || string(b) != "kept" {
		t.Fatalf("the taken name's file is %q, %v", b, err)
	}
	b, _ := os.ReadFile(filepath.Join(store.Dir, "rom1", name))
	if c, err := alch.Read(b); err != nil || string(c.Name) != "Aeryn" || c.Stats != [4]uint32{21, 22, 23, 24} || c.Flags != alch.FlagShort {
		t.Fatalf("first game read %+v, %v", c, err)
	}
	row := firstCharacterFiles
	row.fileName = func(at int) (string, bool) { return "char0000.chr", at == 0 }
	if name, _, err := first.writeCharacter(row, store, storedCharacter{result: res(0, 0)}, ""); err == nil ||
		err.Error() != "No free character file name is left. Delete a character first." || name != "" {
		t.Fatalf("a folder with no free name stored %q, %v", name, err)
	}
	if b, err := os.ReadFile(taken); err != nil || string(b) != "kept" {
		t.Fatalf("the refused write changed the taken file: %q, %v", b, err)
	}
	second := storeFront(t, base.ROM2EN)
	for i, want := range []struct {
		sex, class int
		unit       byte
	}{{0, 0, 0x00}, {0, 1, 0x40}, {1, 0, 0x80}, {1, 1, 0xc0}} {
		name, err := second.storeCharacter(store, res(want.sex, want.class))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(filepath.Join(store.Dir, "rom2", name))
		c, err := a2c.Read(b)
		if err != nil || string(c.A.Name) != "Aeryn" || c.A.Unit != want.unit || c.A.ID0 != 1 || c.A.ID1 != uint32(i) ||
			c.E.Base != [4]uint8{21, 22, 23, 24} || c.E.Money != 1000 {
			t.Fatalf("%s read %+v, %v", name, c, err)
		}
	}
}
