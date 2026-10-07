package winicon

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
)

// swatch is the k-th of 256 distinct opaque colours: the red channel alone
// tells them apart, since 37 is odd.
func swatch(k int) color.NRGBA {
	return color.NRGBA{R: byte(k*37 + 11), G: byte(k*91 + 5), B: byte(200 - k*3), A: 255}
}

// art draws a size by size picture from `colours` distinct opaque colours, with
// its corners cut away: the pixels outside the inscribed circle are transparent.
func art(size, colours int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := 2*x+1-size, 2*y+1-size
			if dx*dx+dy*dy > size*size {
				continue
			}
			img.SetNRGBA(x, y, swatch((x*3+y*5+x*y)%colours))
		}
	}
	return img
}

// translucent returns img with a diagonal of half-transparent pixels, which
// only an alpha channel can carry.
func translucent(img *image.NRGBA) *image.NRGBA {
	out := image.NewNRGBA(img.Bounds())
	copy(out.Pix, img.Pix)
	for i := 0; i < out.Bounds().Dx(); i++ {
		c := out.NRGBAAt(i, i)
		c.A = 128
		out.SetNRGBA(i, i, c)
	}
	return out
}

func coloursAt(depth int) int {
	if depth >= 24 {
		return 256
	}
	return 1 << depth
}

func iconEntry(id uint16, size, depth int, data []byte) synth.IconEntry {
	return synth.IconEntry{ID: id, Width: size, Height: size, Depth: depth, Data: data}
}

// iconExecutable builds an executable with one group numbered id and the icon
// resources for entries, all in one language.
func iconExecutable(id, language uint16, entries []synth.IconEntry) []byte {
	res := []synth.PEResource{{Type: 14, ID: id, Language: language, Data: synth.IconGroup(entries)}}
	return synth.PE(append(res, synth.IconResources(entries, language)...))
}

func decode(t *testing.T, exe []byte) *Group {
	t.Helper()
	g, err := FromExecutable(bytes.NewReader(exe))
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func sameImage(t *testing.T, label string, got, want *image.NRGBA) {
	t.Helper()
	if got.Bounds() != want.Bounds() {
		t.Errorf("%s: bounds %v, want %v", label, got.Bounds(), want.Bounds())
		return
	}
	for y := 0; y < want.Bounds().Dy(); y++ {
		for x := 0; x < want.Bounds().Dx(); x++ {
			if g, w := got.NRGBAAt(x, y), want.NRGBAAt(x, y); g != w {
				t.Errorf("%s: pixel %d,%d is %v, want %v", label, x, y, g, w)
				return
			}
		}
	}
}

// Every size and depth decodes to the picture it was encoded from, transparent
// corners included. Sizes 16 and 48 leave padding bits at the end of each mask
// row, which the encoder sets, so a reader that counted them would shift or
// darken pixels. The images come back in the order the group names them.
func TestEverySizeAndDepthDecodesToItsSource(t *testing.T) {
	type spec struct{ size, depth int }
	specs := []spec{
		{32, 4}, {32, 8}, {48, 8}, {16, 4}, {48, 4}, {16, 8},
		{16, 1}, {32, 1}, {16, 24}, {48, 24}, {16, 32}, {32, 32}, {48, 32},
	}
	var entries []synth.IconEntry
	var want []*image.NRGBA
	for i, s := range specs {
		src := art(s.size, coloursAt(s.depth))
		if s.depth == 32 {
			src = translucent(src)
		}
		want = append(want, src)
		entries = append(entries, iconEntry(uint16(10+i), s.size, s.depth, synth.IconDIB(src, s.depth)))
	}
	pngSrc := translucent(art(256, 256))
	want = append(want, pngSrc)
	entries = append(entries, iconEntry(99, 256, 32, synth.IconPNG(pngSrc)))

	g := decode(t, iconExecutable(119, 0x419, entries))
	if g.Name != "#119" || g.Language != 0x419 {
		t.Errorf("group %q language %#x, want #119 language 0x419", g.Name, g.Language)
	}
	if len(g.Images) != len(entries) {
		t.Fatalf("%d images, want %d", len(g.Images), len(entries))
	}
	for i, img := range g.Images {
		wantDepth := 32
		if i < len(specs) {
			wantDepth = specs[i].depth
		}
		if img.Depth != wantDepth {
			t.Errorf("image %d: depth %d, want %d", i, img.Depth, wantDepth)
		}
		sameImage(t, fmt.Sprintf("image %d (%d wide at %d bits)", i, want[i].Bounds().Dx(), wantDepth), img.Pix, want[i])
	}
}

// A 32-bit image written before alpha channels existed has zero in every alpha
// byte, and its mask alone says what is transparent.
func TestThirtyTwoBitImageWithoutAlphaTakesItsMask(t *testing.T) {
	src := art(32, 256)
	g := decode(t, iconExecutable(1, 0x409, []synth.IconEntry{iconEntry(1, 32, 32, synth.IconDIBMasked32(src))}))
	sameImage(t, "masked 32-bit image", g.Images[0].Pix, src)
	if g.Images[0].Pix.NRGBAAt(0, 0).A != 0 || g.Images[0].Pix.NRGBAAt(16, 16).A != 255 {
		t.Error("the corner is not transparent or the centre is not opaque")
	}
}

// A 32-bit image with an alpha channel ignores its mask, whatever the mask says.
func TestThirtyTwoBitImageWithAlphaIgnoresItsMask(t *testing.T) {
	src := translucent(art(16, 256))
	data := synth.IconDIB(src, 32)
	for i := 40 + 16*16*4; i < len(data); i++ {
		data[i] = 0xff
	}
	g := decode(t, iconExecutable(1, 0x409, []synth.IconEntry{iconEntry(1, 16, 32, data)}))
	sameImage(t, "alpha image under a full mask", g.Images[0].Pix, src)
}

// A masked pixel decodes to zero in every channel. The encoder puts a colour
// that is not black under the mask, so a reader that kept it would show it.
func TestMaskedPixelsDecodeToZero(t *testing.T) {
	for _, depth := range []int{1, 4, 8, 24} {
		src := art(16, coloursAt(depth))
		g := decode(t, iconExecutable(1, 0x409, []synth.IconEntry{iconEntry(1, 16, depth, synth.IconDIB(src, depth))}))
		if got := g.Images[0].Pix.NRGBAAt(0, 0); got != (color.NRGBA{}) {
			t.Errorf("depth %d: corner pixel %v, want zero in every channel", depth, got)
		}
		if got := g.Images[0].Pix.NRGBAAt(8, 8); got.A != 255 {
			t.Errorf("depth %d: centre pixel %v, want opaque", depth, got)
		}
	}
}

// The first group in directory order is the icon: a named group before any
// numeric one, and the lowest number among numeric ones however the file lists
// them. Each image is read in the group's language when it has one.
func TestFirstGroupInDirectoryOrder(t *testing.T) {
	one, two := art(16, 2), art(16, 3)
	a := iconEntry(1, 16, 8, synth.IconDIB(one, 8))
	b := iconEntry(2, 16, 8, synth.IconDIB(two, 8))
	icons := synth.IconResources([]synth.IconEntry{a, b}, 0x409)

	t.Run("numeric groups by ascending id", func(t *testing.T) {
		res := append([]synth.PEResource{
			{Type: 14, ID: 7, Language: 0x409, Data: synth.IconGroup([]synth.IconEntry{b})},
			{Type: 14, ID: 3, Language: 0x409, Data: synth.IconGroup([]synth.IconEntry{a})},
		}, icons...)
		g := decode(t, synth.PE(res))
		if g.Name != "#3" || len(g.Images) != 1 {
			t.Fatalf("group %q with %d images, want #3 with one", g.Name, len(g.Images))
		}
		sameImage(t, "group #3", g.Images[0].Pix, one)
	})

	t.Run("a named group before numeric ones", func(t *testing.T) {
		res := append([]synth.PEResource{
			{Type: 14, ID: 1, Language: 0x409, Data: synth.IconGroup([]synth.IconEntry{a})},
			{Type: 14, Name: "MainIcon", Language: 0x409, Data: synth.IconGroup([]synth.IconEntry{b})},
		}, icons...)
		g := decode(t, synth.PE(res))
		if g.Name != "MainIcon" {
			t.Fatalf("group %q, want MainIcon", g.Name)
		}
		sameImage(t, "group MainIcon", g.Images[0].Pix, two)
	})

	t.Run("an image in the group's language wins, else its first language", func(t *testing.T) {
		res := []synth.PEResource{
			{Type: 14, ID: 1, Language: 0x409, Data: synth.IconGroup([]synth.IconEntry{a, b})},
			{Type: 3, ID: 1, Language: 0x419, Data: b.Data}, // icon 1 in Russian, other art
			{Type: 3, ID: 1, Language: 0x409, Data: a.Data},
			{Type: 3, ID: 2, Language: 0x419, Data: b.Data}, // icon 2 only in Russian
		}
		g := decode(t, synth.PE(res))
		if len(g.Images) != 2 {
			t.Fatalf("%d images, want 2", len(g.Images))
		}
		sameImage(t, "icon 1 in the group's language", g.Images[0].Pix, one)
		sameImage(t, "icon 2 in its only language", g.Images[1].Pix, two)
	})
}

func patched(b []byte, edits map[int]uint32, width int) []byte {
	out := append([]byte(nil), b...)
	for off, v := range edits {
		if width == 2 {
			binary.LittleEndian.PutUint16(out[off:], uint16(v))
		} else {
			binary.LittleEndian.PutUint32(out[off:], v)
		}
	}
	return out
}

// Damaged input is an error that names what is wrong, never a panic and never
// an allocation sized by a number the file states.
func TestRefusals(t *testing.T) {
	valid := synth.IconDIB(art(16, 200), 8)
	one := func(data []byte) []byte {
		return iconExecutable(1, 0x409, []synth.IconEntry{iconEntry(1, 16, 8, data)})
	}
	minus := func(n uint32) uint32 { return -n }
	group := func(edit func([]byte)) []byte {
		g := synth.IconGroup([]synth.IconEntry{iconEntry(1, 16, 8, valid)})
		edit(g)
		res := []synth.PEResource{{Type: 14, ID: 1, Language: 0x409, Data: g}}
		return synth.PE(append(res, synth.IconResources([]synth.IconEntry{iconEntry(1, 16, 8, valid)}, 0x409)...))
	}
	bare := synth.PE(nil)
	resDirAt := 0x40 + 4 + 20 + 96 + 2*8 // the resource slot of the data directories

	cases := []struct {
		name string
		exe  []byte
		want string
	}{
		{"empty input", nil, "not a Windows executable"},
		{"not an executable", []byte(strings.Repeat("not an executable ", 20)), "not a Windows executable"},
		{"no resource directory", patched(bare, map[int]uint32{resDirAt: 0}, 4), "no resource directory"},
		{"resource directory outside every section", patched(bare, map[int]uint32{resDirAt: 0x9000}, 4), "lies in no section"},
		{"no icon group", bare, "no icon group"},
		{"a group and no icon", synth.PE([]synth.PEResource{{Type: 14, ID: 1, Language: 0x409, Data: synth.IconGroup([]synth.IconEntry{iconEntry(1, 16, 8, valid)})}}), "no icons"},
		{"a group naming an icon that is absent", synth.PE([]synth.PEResource{
			{Type: 14, ID: 1, Language: 0x409, Data: synth.IconGroup([]synth.IconEntry{iconEntry(5, 16, 8, valid)})},
			{Type: 3, ID: 1, Language: 0x409, Data: valid},
		}), "does not hold"},

		{"a cursor group", group(func(g []byte) { binary.LittleEndian.PutUint16(g[2:], 2) }), "not an icon group"},
		{"a group of no image", group(func(g []byte) { binary.LittleEndian.PutUint16(g[4:], 0) }), "lists no image"},
		{"a group of too many images", group(func(g []byte) { binary.LittleEndian.PutUint16(g[4:], MaxImages+1) }), "more than the 64 allowed"},
		{"a group that lies about its count", group(func(g []byte) { binary.LittleEndian.PutUint16(g[4:], 3) }), "cannot hold its 3 entries"},

		{"a bitmap header of another size", one(patched(valid, map[int]uint32{0: 108}, 4)), "header of 108 bytes"},
		{"a bitmap with two planes", one(patched(valid, map[int]uint32{12: 2}, 2)), "2 planes"},
		{"a compressed bitmap", one(patched(valid, map[int]uint32{16: 1}, 4)), "compression 1"},
		{"a bitmap of 16 bits per pixel", one(patched(valid, map[int]uint32{14: 16}, 2)), "16 bits per pixel"},
		{"a bitmap of no width", one(patched(valid, map[int]uint32{4: 0}, 4)), "width 0"},
		{"a bitmap wider than 256", one(patched(valid, map[int]uint32{4: 257}, 4)), "width 257"},
		{"a bitmap of odd height", one(patched(valid, map[int]uint32{8: 33}, 4)), "height 33"},
		{"a bitmap taller than 256", one(patched(valid, map[int]uint32{8: 514}, 4)), "height 514"},
		{"a top-down bitmap", one(patched(valid, map[int]uint32{8: minus(32)}, 4)), "height -32"},
		{"a bitmap that stops short", one(valid[:len(valid)-1]), "its header needs"},
		{"a colour table larger than the depth", one(patched(valid, map[int]uint32{32: 300}, 4)), "colour table of 300 entries"},
		{"a pixel beyond its colour table", one(patched(valid, map[int]uint32{32: 2}, 4)), "names colour"},
		{"a bitmap of a few bytes", one([]byte{40, 0, 0, 0}), "shorter than its 40-byte header"},
		{"a PNG that is cut off", one(synth.IconPNG(art(16, 200))[:40]), "PNG image"},
		{"a PNG larger than 256", one(synth.IconPNG(art(300, 4))), "outside 1 to 256"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g, err := FromExecutable(bytes.NewReader(c.exe))
			if err == nil {
				t.Fatalf("decoded %d images, want an error naming %q", len(g.Images), c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not name %q", err, c.want)
			}
		})
	}
}

// Truncating the file anywhere, and overwriting any single byte with 0x00 or
// 0xff, ends in an error or a decoded icon and never in a panic or a stall.
func TestDamagedExecutableNeverPanics(t *testing.T) {
	entries := []synth.IconEntry{
		iconEntry(1, 16, 4, synth.IconDIB(art(16, 16), 4)),
		iconEntry(2, 32, 8, synth.IconDIB(art(32, 200), 8)),
		iconEntry(3, 16, 32, synth.IconPNG(translucent(art(16, 256)))),
	}
	exe := iconExecutable(1, 0x409, entries)
	if _, err := FromExecutable(bytes.NewReader(exe)); err != nil {
		t.Fatalf("the undamaged executable does not decode: %v", err)
	}
	for n := 0; n < len(exe); n++ {
		_, _ = FromExecutable(bytes.NewReader(exe[:n]))
	}
	for at := range exe {
		for _, v := range []byte{0x00, 0xff} {
			damaged := append([]byte(nil), exe...)
			damaged[at] = v
			_, _ = FromExecutable(bytes.NewReader(damaged))
		}
	}
}

// The installed executable holds one icon group of six bitmap images. Sizes,
// depths and pixel fingerprints were measured with an independent decoder over
// the executable's icon resources. Every mask bit that belongs to a pixel is
// clear, so every pixel is opaque; the padding bits that end the mask rows of
// the 16 and 48 pixel images are set and belong to none. The installed file is
// read and nothing is written.
func TestReleaseInstalledExecutableIconImages(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the installed executable's icon needs a lawful install")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var path string
	for _, e := range entries {
		if strings.EqualFold(e.Name(), "rom.exe") {
			path = filepath.Join(root, e.Name())
		}
	}
	if path == "" {
		t.Fatalf("no rom.exe under %s", root)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := FromExecutable(f)
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "#119" || g.Language != 0x419 {
		t.Errorf("group %q language %#x, want #119 language 0x419", g.Name, g.Language)
	}
	want := []struct {
		size, depth int
		sha256      string
	}{
		{32, 4, "10c927aa4aba44533999eb4141adc3bbd5c308c5841ecf33699306315e6ef962"},
		{32, 8, "5f9b6f80730751fbc68120b6b944c8b909d940bd8ad18a4a6ce6e1acebfaf9ad"},
		{48, 8, "60d63672e4dc91bb6ab19ff442051491ca6e5ed5880d532083449efb5733bbd6"},
		{16, 4, "02814ddc55163b21657d113a6ab3550fcf58b8c4a5883bb2997935d5dea6d3dc"},
		{48, 4, "b61322f15137eaeea64983c669ae0f02599d7b84020335766c251460b5b3e36c"},
		{16, 8, "face43024b5b5724bccf6f36b3aba5a6624f1bcd08ea3a8a7ff38ca1a189ecbb"},
	}
	if len(g.Images) != len(want) {
		t.Fatalf("%d images, want %d", len(g.Images), len(want))
	}
	for i, w := range want {
		img := g.Images[i]
		b := img.Pix.Bounds()
		if b.Dx() != w.size || b.Dy() != w.size || img.Depth != w.depth {
			t.Errorf("image %d is %dx%d at %d bits, want %dx%d at %d", i, b.Dx(), b.Dy(), img.Depth, w.size, w.size, w.depth)
			continue
		}
		sum := sha256.Sum256(img.Pix.Pix)
		if got := hex.EncodeToString(sum[:]); got != w.sha256 {
			t.Errorf("image %d (%d at %d bits): pixel fingerprint %s, want %s", i, w.size, w.depth, got, w.sha256)
		}
		for p := 3; p < len(img.Pix.Pix); p += 4 {
			if img.Pix.Pix[p] != 255 {
				t.Errorf("image %d: pixel %d is not opaque", i, p/4)
				break
			}
		}
	}
}
