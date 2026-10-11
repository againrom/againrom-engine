package screenart

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
)

// fakeSource answers pictures, masks and sheets from maps and counts every
// read by key.
type fakeSource struct {
	pictures map[string]image.Image
	masks    map[string]*image.Paletted
	sheets   map[string][]image.Image
	reads    map[string]int
}

func newFakeSource() *fakeSource {
	return &fakeSource{pictures: map[string]image.Image{}, masks: map[string]*image.Paletted{},
		sheets: map[string][]image.Image{}, reads: map[string]int{}}
}

func (f *fakeSource) Picture(key string) (image.Image, error) {
	f.reads[key]++
	if p, ok := f.pictures[key]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("%s: not found", key)
}

func (f *fakeSource) Mask(key string) (*image.Paletted, error) {
	f.reads[key]++
	if m, ok := f.masks[key]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("%s: not found", key)
}

func (f *fakeSource) Sprites(key string) []image.Image {
	f.reads[key]++
	return f.sheets[key]
}

func solid(w, h int, c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(pic.Pix); i += 4 {
		pic.Pix[i], pic.Pix[i+1], pic.Pix[i+2], pic.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return pic
}

func size(w, h int) *Point { return &Point{w, h} }

// A required entry whose picture has the wrong size fails the load with one
// error naming the key and both sizes.
func TestLoadRefusesAWrongSizeWithTheKey(t *testing.T) {
	src := newFakeSource()
	src.pictures["a.bmp"] = solid(4, 3, color.RGBA{A: 255})
	_, err := Load("panel", []Entry{{Name: "a", Format: "picture", Key: "a.bmp", Size: size(4, 4), Required: true}}, src)
	if err == nil || !strings.Contains(err.Error(), "a.bmp") || !strings.Contains(err.Error(), "size 4x3, want 4x4") {
		t.Fatalf("error = %v, want one naming a.bmp and its size", err)
	}
}

// A required entry whose picture is missing fails the load with the key; an
// optional one is named in Problems and leaves the others loaded.
func TestLoadRefusesAMissingRequiredPictureAndSkipsAnOptionalOne(t *testing.T) {
	src := newFakeSource()
	src.pictures["b.bmp"] = solid(2, 2, color.RGBA{R: 9, A: 255})
	if _, err := Load("panel", []Entry{{Name: "a", Format: "picture", Key: "a.bmp", Required: true}}, src); err == nil || !strings.Contains(err.Error(), "a.bmp") {
		t.Fatalf("missing required picture error = %v, want one naming a.bmp", err)
	}
	set, err := Load("panel", []Entry{
		{Name: "a", Format: "picture", Key: "a.bmp"},
		{Name: "b", Format: "picture", Key: "b.bmp", Required: true},
	}, src)
	if err != nil {
		t.Fatal(err)
	}
	if set.Picture("a") != nil || set.Picture("b") == nil || len(set.Problems) != 1 || !strings.Contains(set.Problems[0], "a.bmp") {
		t.Fatalf("set = %v %v, problems %v", set.Picture("a"), set.Picture("b"), set.Problems)
	}
}

// Two screens naming one picture under one rule read and validate it once,
// through one library, and draw the same decoded picture.
func TestLibraryValidatesASharedPictureOnce(t *testing.T) {
	src := newFakeSource()
	src.pictures["shared.bmp"] = solid(8, 4, color.RGBA{A: 255})
	src.pictures["own.bmp"] = solid(2, 2, color.RGBA{G: 1, A: 255})
	lib := NewLibrary(src)
	shared := Entry{Name: "menu", Format: "picture", Key: "shared.bmp", Size: size(8, 4), Transparent: "black", Required: true}
	first, err := lib.Load("tavern", []Entry{shared})
	if err != nil {
		t.Fatal(err)
	}
	second, err := lib.Load("shop", []Entry{{Name: "own", Format: "picture", Key: "own.bmp"}, shared})
	if err != nil {
		t.Fatal(err)
	}
	if src.reads["shared.bmp"] != 1 || lib.Reads() != 2 {
		t.Fatalf("shared.bmp read %d times, library read %d keys; want 1 and 2", src.reads["shared.bmp"], lib.Reads())
	}
	if first.Picture("menu") != second.Picture("menu") {
		t.Fatal("the two screens hold different decodes of one picture")
	}
	if a := first.Picture("menu").(*image.RGBA).Pix[3]; a != 0 {
		t.Fatalf("pure black alpha = %d, want the key colour cleared", a)
	}
}

// Reread gives a fresh library every key another has read, under its rule,
// so the two hold equal pictures.
func TestRereadReadsWhatAnotherLibraryRead(t *testing.T) {
	src := newFakeSource()
	src.pictures["a.bmp"] = solid(2, 2, color.RGBA{A: 255})
	src.masks["m.bmp"] = &image.Paletted{Pix: []uint8{1}, Stride: 1, Rect: image.Rect(0, 0, 1, 1)}
	src.sheets["s.16a"] = []image.Image{solid(1, 1, color.RGBA{A: 255})}
	used := NewLibrary(src)
	if _, err := used.Load("screen", []Entry{
		{Name: "a", Format: "picture", Key: "a.bmp", Size: size(2, 2), Transparent: "black"},
		{Name: "m", Format: "mask", Key: "m.bmp"},
		{Name: "s", Format: "frame", Key: "s.16a"},
		{Name: "gone", Format: "picture", Key: "gone.bmp"},
	}); err != nil {
		t.Fatal(err)
	}
	fresh := NewLibrary(src)
	fresh.Reread(used)
	if fresh.Reads() != used.Reads() || len(fresh.memo) != 4 {
		t.Fatalf("reads %d, want %d; memo %d keys", fresh.Reads(), used.Reads(), len(fresh.memo))
	}
	for id, m := range used.memo {
		f := fresh.memo[id]
		if f == nil || f.rule.String() != m.rule.String() || f.screen != m.screen || (f.err == nil) != (m.err == nil) {
			t.Fatalf("%s: fresh %+v, used %+v", id, f, m)
		}
	}
	if a := fresh.memo["picture|a.bmp"].frames[0].(*image.RGBA).Pix[3]; a != 0 {
		t.Fatalf("reread picture alpha = %d, want its key colour applied", a)
	}
}

// A second screen naming a read picture under another rule fails with an
// error naming both screens and both rules.
func TestLibraryRefusesASecondRuleForOnePicture(t *testing.T) {
	src := newFakeSource()
	src.pictures["shared.bmp"] = solid(8, 4, color.RGBA{A: 255})
	lib := NewLibrary(src)
	if _, err := lib.Load("tavern", []Entry{{Name: "m", Format: "picture", Key: "shared.bmp", Size: size(8, 4), Required: true}}); err != nil {
		t.Fatal(err)
	}
	_, err := lib.Load("shop", []Entry{{Name: "m", Format: "picture", Key: "shared.bmp", Transparent: "black", Required: true}})
	if err == nil || !strings.Contains(err.Error(), "tavern") || !strings.Contains(err.Error(), "shop") || !strings.Contains(err.Error(), "shared.bmp") {
		t.Fatalf("error = %v, want one naming the picture and both screens", err)
	}
	if src.reads["shared.bmp"] != 1 {
		t.Fatalf("shared.bmp read %d times, want 1", src.reads["shared.bmp"])
	}
}

// Every format and placement the screens use: an indexed family by printed
// number, a series, a sheet and one of its frames, a mask with its required
// indices, a crop, a placement inside a rectangle, and a bound entry.
func TestLoadReadsEveryFormat(t *testing.T) {
	src := newFakeSource()
	for _, n := range []int{3, 7} {
		src.sheets[fmt.Sprintf("u%d.16a", n)] = []image.Image{solid(2, 2, color.RGBA{A: 255})}
	}
	src.pictures["s1.bmp"] = solid(3, 3, color.RGBA{A: 255})
	src.pictures["s2.bmp"] = solid(3, 3, color.RGBA{A: 255})
	src.sheets["f.256"] = []image.Image{solid(1, 1, color.RGBA{A: 255}), solid(5, 6, color.RGBA{A: 255})}
	src.masks["m.bmp"] = &image.Paletted{Pix: []uint8{1, 2, 2, 1}, Stride: 2, Rect: image.Rect(0, 0, 2, 2)}
	src.pictures["wide.bmp"] = solid(10, 4, color.RGBA{R: 1, A: 255})
	src.pictures["page7.bmp"] = solid(2, 2, color.RGBA{A: 255})
	entries := []Entry{
		{Name: "unit", Format: "sprites", Key: "u%d.16a", Index: []Index{{Values: []int{3, 7}}}, Required: true},
		{Name: "loop", Format: "series", Key: "s%d.bmp", First: 1, Count: 2, Size: size(3, 3), At: &Point{1, 1}, Inside: &Rect{0, 0, 4, 4}, Required: true},
		{Name: "piece", Format: "frame", Key: "f.256", Frame: 1, Size: size(5, 6), Required: true},
		{Name: "mask", Format: "mask", Key: "m.bmp", Size: size(2, 2), RequiredBytes: []int{1, 2}, Required: true},
		{Name: "cap", Format: "picture", Key: "wide.bmp", Size: size(10, 4), Crop: &Rect{6, 0, 10, 4}, Required: true},
		{Name: "page", Format: "picture", Key: "page%d.bmp", Bound: true},
	}
	lib := NewLibrary(src)
	set, err := lib.Load("screen", entries)
	if err != nil {
		t.Fatal(err)
	}
	if got := set.Numbered("unit"); len(got) != 2 || got[3] == nil || got[7] == nil {
		t.Fatalf("numbered unit sheets = %v", got)
	}
	if len(set.Pictures("loop")) != 2 || set.Picture("piece").Bounds().Size() != image.Pt(5, 6) || set.Mask("mask") == nil {
		t.Fatal("series, frame or mask did not load")
	}
	if b := set.Picture("cap").Bounds(); b != image.Rect(0, 0, 4, 4) {
		t.Fatalf("crop bounds = %v", b)
	}
	if src.reads["page%d.bmp"] != 0 || src.reads["page7.bmp"] != 0 {
		t.Fatal("Load read a bound entry")
	}
	frames, err := lib.Bind("screen", entries, "page", 7)
	if err != nil || len(frames) != 1 {
		t.Fatalf("bound page = %v, %v", frames, err)
	}
	if _, err := lib.Bind("screen", entries, "page", 8); err == nil || !strings.Contains(err.Error(), "page8.bmp") {
		t.Fatalf("missing bound page error = %v", err)
	}

	src.masks["m.bmp"] = &image.Paletted{Pix: []uint8{1, 1, 1, 1}, Stride: 2, Rect: image.Rect(0, 0, 2, 2)}
	if _, err := Load("screen", entries[3:4], src); err == nil || !strings.Contains(err.Error(), "missing required mask index 2") {
		t.Fatalf("blank mask error = %v", err)
	}
	outside := Entry{Name: "loop", Format: "series", Key: "s%d.bmp", First: 1, Count: 2, At: &Point{2, 2}, Inside: &Rect{0, 0, 4, 4}, Required: true}
	if _, err := Load("screen", []Entry{outside}, src); err == nil || !strings.Contains(err.Error(), "lies outside") {
		t.Fatalf("outside patch error = %v", err)
	}
}

// Keys and Rule describe what an entry reads, so shipped manifests can be
// checked for one rule per picture without an install.
func TestEntryKeysAndRule(t *testing.T) {
	e := Entry{Name: "b", Format: "series", Key: "b%d.bmp", Index: nil, First: 1, Count: 2, Size: size(4, 4), Transparent: "black"}
	if got := e.Keys(); len(got) != 2 || got[0] != "b1.bmp" || got[1] != "b2.bmp" {
		t.Fatalf("keys = %v", got)
	}
	if got := e.Rule(); got != "picture size 4x4 transparent black" {
		t.Fatalf("rule = %q", got)
	}
	if (Entry{Format: "picture", Key: "p%d.bmp", Bound: true}).Keys() != nil {
		t.Fatal("a bound entry lists a key")
	}
}

// Check refuses an unknown format, an unnamed entry and a name used twice.
func TestCheckRefusesMalformedManifests(t *testing.T) {
	for _, bad := range [][]Entry{
		{{Name: "a", Format: "video", Key: "a"}},
		{{Format: "picture", Key: "a"}},
		{{Name: "a", Format: "picture", Key: "a"}, {Name: "a", Format: "mask", Key: "b"}},
	} {
		if Check(bad) == nil {
			t.Errorf("Check accepted %+v", bad)
		}
	}
}

// An indexed series takes its first number by its last index: member 0 of
// series 1 is s1000.bmp and member 0 of series 2 is s2001.bmp.
func TestIndexedSeriesTakesItsFirstNumberByIndex(t *testing.T) {
	src := newFakeSource()
	for _, k := range []string{"s1000.bmp", "s1001.bmp", "s1002.bmp", "s2001.bmp", "s2002.bmp"} {
		src.pictures[k] = solid(2, 2, color.RGBA{A: 255})
	}
	e := Entry{Name: "girl", Format: "series", Key: "s%d%%03d.bmp", Index: []Index{{Count: 2, Base: 1}},
		Counts: []int{3, 2}, Firsts: []int{0, 1}, Required: true}
	if got := strings.Join(e.Keys(), ","); got != "s1000.bmp,s1001.bmp,s1002.bmp,s2001.bmp,s2002.bmp" {
		t.Fatalf("keys = %s", got)
	}
	set, err := Load("screen", []Entry{e}, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Pictures("girl/0")) != 3 || len(set.Pictures("girl/1")) != 2 || src.reads["s2000.bmp"] != 0 {
		t.Fatalf("series %d and %d frames, s2000 read %d times", len(set.Pictures("girl/0")), len(set.Pictures("girl/1")), src.reads["s2000.bmp"])
	}
}

func TestSpritesRejectsAnyEmptyFrame(t *testing.T) {
	for _, tc := range []struct {
		name  string
		frame image.Image
	}{
		{"zero-width", image.NewRGBA(image.Rect(0, 0, 0, 2))},
		{"zero-height", image.NewRGBA(image.Rect(0, 0, 2, 0))},
		{"zero-area", image.NewRGBA(image.Rectangle{})},
		{"nil", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := newFakeSource()
			good := solid(2, 2, color.RGBA{R: 7, A: 255})
			src.sheets["sheet.256"] = []image.Image{good, tc.frame, good}
			lib := NewLibrary(src)
			set, err := lib.Load("whole", []Entry{{Name: "whole", Format: "sprites", Key: "sheet.256", Required: true}})
			if err == nil || set != nil || !strings.Contains(err.Error(), "sheet.256") {
				t.Errorf("empty sheet member returned set %v, error %v", set, err)
			}
			set, err = lib.Load("selected", []Entry{{Name: "one", Format: "frame", Key: "sheet.256", Frame: 2, Required: true}})
			if err != nil || set.Picture("one") != good {
				t.Fatalf("whole-sheet rejection removed a readable selected frame: %v", err)
			}
			if src.reads["sheet.256"] != 1 {
				t.Fatalf("sheet reads = %d, want 1", src.reads["sheet.256"])
			}
		})
	}
}

func TestFrameValidatesOnlyItsSelectedFrame(t *testing.T) {
	for _, tc := range []struct {
		name  string
		frame image.Image
	}{
		{"zero-area", image.NewRGBA(image.Rectangle{})},
		{"nil", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := newFakeSource()
			good := solid(2, 2, color.RGBA{R: 7, A: 255})
			src.sheets["sheet.256"] = []image.Image{good, tc.frame}
			lib := NewLibrary(src)
			set, err := lib.Load("optional", []Entry{
				{Name: "empty", Format: "frame", Key: "sheet.256", Frame: 1},
				{Name: "good", Format: "frame", Key: "sheet.256", Frame: 0, Required: true},
			})
			if err != nil {
				t.Fatal(err)
			}
			if set.Picture("empty") != nil || set.Picture("good") != good || len(set.Problems) != 1 || !strings.Contains(set.Problems[0], "sheet.256") {
				t.Fatalf("optional frame results: empty %v, good %v, problems %v", set.Picture("empty"), set.Picture("good"), set.Problems)
			}
			set, err = lib.Load("required", []Entry{{Name: "empty", Format: "frame", Key: "sheet.256", Frame: 1, Required: true}})
			if err == nil || set != nil || !strings.Contains(err.Error(), "sheet.256") {
				t.Fatalf("empty required frame returned set %v, error %v", set, err)
			}
			if src.reads["sheet.256"] != 1 {
				t.Fatalf("sheet reads = %d, want 1", src.reads["sheet.256"])
			}
		})
	}
}
