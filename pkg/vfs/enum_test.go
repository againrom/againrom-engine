package vfs_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/res"
	"againrom/pkg/vfs"
)

// ---------------------------------------------------------------- fixtures

// archAt and dirAt are the two source shapes an answer can take.
func archAt(identity string, index int) vfs.Source {
	return vfs.Source{Tier: vfs.TierArchive, Identity: identity, Index: index}
}

func dirAt(index int) vfs.Source {
	return vfs.Source{Tier: vfs.TierDir, Index: index}
}

// entryOf is the listing row one fixture file must produce: its address, the
// archive serving it, and the size of its bytes.
func entryOf(address, identity string, index int, data []byte) vfs.Entry {
	return vfs.Entry{Address: address, Source: archAt(identity, index), Size: int64(len(data))}
}

// looseTree records every path under root — a regular file by its bytes, a
// directory by a marker, so a file replaced by a directory of the same name reads
// as a difference. Two of these compared say that nothing was created, removed or
// changed, and root here holds the archive hosts as well as the loose tier.
func looseTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		if d.IsDir() {
			out[filepath.ToSlash(rel)] = "<dir>"
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return out
}

// ---------------------------------------------------------------- assertions

func assertAscendingAndUnique(t *testing.T, got []vfs.Entry) {
	t.Helper()
	for i := 1; i < len(got); i++ {
		if got[i-1].Address >= got[i].Address {
			t.Errorf("Entries()[%d].Address = %q does not ascend past [%d] = %q",
				i, got[i].Address, i-1, got[i-1].Address)
		}
	}
}

func assertListedSourcesAnswerTheReads(t *testing.T, f *vfs.FS, got []vfs.Entry, byAddress map[string][]byte) {
	t.Helper()
	for _, e := range got {
		want, known := byAddress[e.Address]
		if !known {
			t.Errorf("Entries() lists %q, which the fixture does not place anywhere", e.Address)
			continue
		}
		if e.Size != int64(len(want)) {
			t.Errorf("Entries() gives %q size %d, want %d", e.Address, e.Size, len(want))
		}
		assertBytes(t, f, e.Address, want)
		src, ok := f.Locate(e.Address)
		if !ok {
			t.Errorf("Locate(%q) reports absent, though the listing serves it from %+v", e.Address, e.Source)
			continue
		}
		if src != e.Source {
			t.Errorf("Locate(%q) = %+v, the listing says %+v", e.Address, src, e.Source)
		}
	}
}

// ---------------------------------------------------------------- AC-11

func TestEntriesListsEveryAddressOnceInOrder(t *testing.T) {
	root := t.TempDir()
	gXY := []byte("G:x/y.bmp")
	gDirt := []byte("G:terrain.3d/dirt.bmp")
	gA := []byte("G:a.bin")
	mXY := []byte("M:x/y.bmp")
	mUp := []byte("M:sub/up.bin")

	// The entries are given in an order that is not the sorted one — buildArchive
	// emits each directory's files before the root's — so the ascent below is the
	// listing's doing and not the fixture's. Sub/UP.BIN is written capitalised: an
	// address is the folded one.
	graphics := writeArchive(t, filepath.Join(root, "arc"), "graphics.res",
		resFile{path: "x/y.bmp", data: gXY},
		resFile{path: "terrain.3d/dirt.bmp", data: gDirt},
		resFile{path: "a.bin", data: gA})
	main := writeArchive(t, filepath.Join(root, "arc"), "main.res",
		resFile{path: "x/y.bmp", data: mXY},
		resFile{path: "Sub/UP.BIN", data: mUp})

	// The loose tier holds an address under a registered identity and one under no
	// identity at all; neither may be listed.
	loose := filepath.Join(root, "loose")
	writeLoose(t, loose, "graphics/loose.bin", []byte("D:graphics/loose.bin"))
	writeLoose(t, loose, "world/data.bin", []byte("D:world/data.bin"))

	f := mustOpen(t, []string{graphics, main}, []string{loose})

	want := []vfs.Entry{
		entryOf("graphics/a.bin", "graphics", 0, gA),
		entryOf("graphics/terrain.3d/dirt.bmp", "graphics", 0, gDirt),
		entryOf("graphics/x/y.bmp", "graphics", 0, gXY),
		entryOf("main/sub/up.bin", "main", 1, mUp),
		entryOf("main/x/y.bmp", "main", 1, mXY),
	}
	byAddress := map[string][]byte{
		"graphics/a.bin":               gA,
		"graphics/terrain.3d/dirt.bmp": gDirt,
		"graphics/x/y.bmp":             gXY,
		"main/sub/up.bin":              mUp,
		"main/x/y.bmp":                 mXY,
	}

	got := f.Entries()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Entries() =\n%+v\nwant\n%+v", got, want)
	}
	assertAscendingAndUnique(t, got)
	for _, e := range got {
		if e.Source.Tier != vfs.TierArchive {
			t.Errorf("Entries() lists %q from tier %v; the loose tier is never listed", e.Address, e.Source.Tier)
		}
	}
	assertListedSourcesAnswerTheReads(t, f, got, byAddress)

	if second := f.Entries(); !reflect.DeepEqual(second, got) {
		t.Errorf("a second Entries() =\n%+v\ndiffers from the first\n%+v", second, got)
	}
	assertBytes(t, f, "graphics/x/y.bmp", gXY)
	assertBytes(t, f, "graphics/x/y.bmp", gXY)
	assertBytes(t, f, "MAIN/SUB/UP.BIN", mUp)
}

func TestEntriesNeverListsTheLooseTier(t *testing.T) {
	root := t.TempDir()
	loose := filepath.Join(root, "loose")
	data := []byte("D:world/data.bin")
	writeLoose(t, loose, "world/data.bin", data)

	f := mustOpen(t, nil, []string{loose})
	if got := f.Entries(); len(got) != 0 {
		t.Errorf("Entries() = %+v, want an empty listing", got)
	}
	assertBytes(t, f, "world/data.bin", data)
	if src, ok := f.Locate("world/data.bin"); !ok || src != dirAt(0) {
		t.Errorf("Locate(%q) = %+v, %v; want %+v, true", "world/data.bin", src, ok, dirAt(0))
	}

	if got := mustOpen(t, nil, nil).Entries(); len(got) != 0 {
		t.Errorf("Entries() over empty lists = %+v, want an empty listing", got)
	}
}

func TestEntriesEmitsTheReadsWinnerWithinOneIdentity(t *testing.T) {
	root := t.TempDir()
	firstShared := []byte("A:shared.bin")
	secondShared := []byte("B:shared.bin, and longer")
	onlyA := []byte("A:only-a.bin")
	onlyB := []byte("B:only-b.bin")
	a := writeArchive(t, filepath.Join(root, "a"), "graphics.res",
		resFile{path: "shared.bin", data: firstShared},
		resFile{path: "only-a.bin", data: onlyA})
	b := writeArchive(t, filepath.Join(root, "b"), "graphics.res",
		resFile{path: "shared.bin", data: secondShared},
		resFile{path: "only-b.bin", data: onlyB})

	for _, c := range []struct {
		name     string
		order    []string
		shared   []byte
		aAt, bAt int
	}{
		{"a then b", []string{a, b}, firstShared, 0, 1},
		{"b then a", []string{b, a}, secondShared, 1, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := mustOpen(t, c.order, nil)

			want := []vfs.Entry{
				entryOf("graphics/only-a.bin", "graphics", c.aAt, onlyA),
				entryOf("graphics/only-b.bin", "graphics", c.bAt, onlyB),
				// The winner is the first archive of this identity holding the
				// path, so its position is 0 in either order.
				entryOf("graphics/shared.bin", "graphics", 0, c.shared),
			}
			byAddress := map[string][]byte{
				"graphics/only-a.bin": onlyA,
				"graphics/only-b.bin": onlyB,
				"graphics/shared.bin": c.shared,
			}

			got := f.Entries()
			// Not fatal: a listing that names the wrong archive of this identity
			// should also be caught below, by the read and the report disagreeing
			// with it, and both readings of the defect are worth seeing.
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Entries() =\n%+v\nwant\n%+v", got, want)
			}
			assertAscendingAndUnique(t, got)
			assertListedSourcesAnswerTheReads(t, f, got, byAddress)
		})
	}
}

func TestEntriesOmitsAnEntryNoAddressReaches(t *testing.T) {
	reachable := []byte("G:x.bin")
	host := writeArchive(t, t.TempDir(), "graphics.res",
		resFile{path: "x.bin", data: reachable},
		resFile{path: "", data: []byte("G:the unnamed entry")})

	// The entry is really there: the reader indexes both records, and it is the
	// address grammar that cannot express the second one.
	a, err := res.Open(host)
	if err != nil {
		t.Fatalf("res.Open(%s): %v", host, err)
	}
	unnamed := 0
	for _, e := range a.Entries() {
		if e.Path == "" {
			unnamed++
		}
	}
	if unnamed != 1 {
		t.Fatalf("the fixture archive holds %d entries with an empty path, want 1: %+v", unnamed, a.Entries())
	}

	f := mustOpen(t, []string{host}, nil)
	want := []vfs.Entry{entryOf("graphics/x.bin", "graphics", 0, reachable)}
	if got := f.Entries(); !reflect.DeepEqual(got, want) {
		t.Errorf("Entries() = %+v, want %+v", got, want)
	}
	for _, address := range []string{"graphics/", "graphics", "graphics//"} {
		if src, ok := f.Locate(address); ok {
			t.Errorf("Locate(%q) = %+v, want absent", address, src)
		}
		assertAbsent(t, f, address)
	}
	assertBytes(t, f, "graphics/x.bin", reachable)
}

func TestLocateAnswersExactlyOneSourcePerAddress(t *testing.T) {
	root := t.TempDir()
	gX := []byte("G:x.bin")
	mY := []byte("M:y.bin")
	graphics := writeArchive(t, filepath.Join(root, "arc"), "graphics.res", resFile{path: "x.bin", data: gX})
	main := writeArchive(t, filepath.Join(root, "arc"), "main.res", resFile{path: "y.bin", data: mY})

	d1 := filepath.Join(root, "d1")
	d2 := filepath.Join(root, "d2")
	d1Loose := []byte("D1:graphics/loose.bin")
	d1World := []byte("D1:world/data.bin")
	d2Beast := []byte("D2:beast.alm")
	writeLoose(t, d1, "graphics/x.bin", []byte("D1:graphics/x.bin, the decoy"))
	writeLoose(t, d1, "graphics/loose.bin", d1Loose)
	writeLoose(t, d1, "world/data.bin", d1World)
	writeLoose(t, d2, "world/data.bin", []byte("D2:world/data.bin"))
	writeLoose(t, d2, "beast.alm", d2Beast)
	outside := []byte("OUTSIDE-THE-LISTED-ROOTS")
	if err := os.WriteFile(filepath.Join(root, "outside.bin"), outside, 0o600); err != nil {
		t.Fatalf("write outside.bin: %v", err)
	}

	f := mustOpen(t, []string{graphics, main}, []string{d1, d2})
	listed := map[string]bool{}
	for _, e := range f.Entries() {
		listed[e.Address] = true
	}

	for _, c := range []struct {
		address string
		source  vfs.Source
		data    []byte // nil where nothing serves the address
		why     string
	}{
		{"graphics/x.bin", archAt("graphics", 0), gX, "the archive, over the decoy loose copy of it"},
		{`GRAPHICS\X.BIN`, archAt("graphics", 0), gX, "the same address, through the fold"},
		{"main/y.bin", archAt("main", 1), mY, "the second archive"},
		{"graphics//x.bin", archAt("graphics", 0), gX, "the reader's own trim of the remainder"},
		{"graphics/loose.bin", dirAt(0), d1Loose, "a registered identity, an entry the archive lacks"},
		{"world/data.bin", dirAt(0), d1World, "the first directory holding it, not the copy in the second"},
		{"BEAST.ALM", dirAt(1), d2Beast, "no separator at all"},
		{"nosuch/x.bin", vfs.Source{}, nil, "no such identity, and no loose copy"},
		{"graphics/nosuch.bin", vfs.Source{}, nil, "the identity is registered, the entry is nowhere"},
		{"graphics", vfs.Source{}, nil, "a bare identity, and a directory stands where the loose tier looks"},
		{`\graphics\x.bin`, vfs.Source{}, nil, "an empty leading segment"},
		{"", vfs.Source{}, nil, "the empty address"},
		{"../outside.bin", vfs.Source{}, nil, "a file that is there, reached by a segment the tier refuses"},
		{"world/../world/data.bin", vfs.Source{}, nil, "cleaning it would resolve, inside the root"},
		{"world/./data.bin", vfs.Source{}, nil, "a . segment"},
		{"world//data.bin", vfs.Source{}, nil, "an empty segment"},
	} {
		src, ok := f.Locate(c.address)
		if want := c.data != nil; ok != want {
			t.Errorf("Locate(%q) ok = %v, want %v (%s)", c.address, ok, want, c.why)
			continue
		}
		if src != c.source {
			t.Errorf("Locate(%q) = %+v, want %+v (%s)", c.address, src, c.source, c.why)
		}
		if c.data == nil {
			assertAbsent(t, f, c.address)
			continue
		}
		// The bytes name their own source, so this is the report checked against
		// the source the read actually used.
		assertBytes(t, f, c.address, c.data)
		if src.Tier == vfs.TierDir && listed[c.address] {
			t.Errorf("Entries() lists %q, which the loose tier serves", c.address)
		}
	}

	if got, err := os.ReadFile(filepath.Join(root, "outside.bin")); err != nil || !bytes.Equal(got, outside) {
		t.Errorf("outside.bin = %q, %v after the sample; want %q unchanged", got, err, outside)
	}
}

// ---------------------------------------------------------------- SC-5

func TestLooseTierRefusesDotDotDotAndEmptySegments(t *testing.T) {
	root := t.TempDir()
	outside := []byte("OUTSIDE-THE-LISTED-ROOT")
	outsidePath := filepath.Join(root, "outside.bin")
	if err := os.WriteFile(outsidePath, outside, 0o600); err != nil {
		t.Fatalf("write %s: %v", outsidePath, err)
	}
	d1 := filepath.Join(root, "d1")
	inRoot := []byte("D1:world/data.bin")
	writeLoose(t, d1, "world/data.bin", inRoot)

	if got, err := os.ReadFile(outsidePath); err != nil || !bytes.Equal(got, outside) {
		t.Fatalf("os.ReadFile(%s) = %q, %v; the escape target must be readable for the refusals below to mean anything",
			outsidePath, got, err)
	}

	f := mustOpen(t, nil, []string{d1})
	for _, address := range []string{
		`..\outside.bin`,          // d1/../outside.bin, which is the file above
		"../outside.bin",          // the same, through the other separator
		"world/../../outside.bin", // the same, from one segment deeper
		"world/../world/data.bin", // cleaning it would resolve, inside the root
		"world/./data.bin",
		"./world/data.bin",
		"world//data.bin",
		"world/..",
		"..",
		".",
	} {
		if src, ok := f.Locate(address); ok {
			t.Errorf("Locate(%q) = %+v, want absent", address, src)
		}
		assertAbsent(t, f, address)
	}

	// The control: with no such segment the same tier resolves, and the escape
	// target is still exactly where it was.
	assertBytes(t, f, "world/data.bin", inRoot)
	if got, err := os.ReadFile(outsidePath); err != nil || !bytes.Equal(got, outside) {
		t.Errorf("os.ReadFile(%s) = %q, %v after the refusals; want %q unchanged", outsidePath, got, err, outside)
	}
}

func TestReadsAndEnumerationsLeaveEveryListedSourceUnchanged(t *testing.T) {
	root := t.TempDir()
	inArchive := []byte("G:x.bin")
	deeper := []byte("G:d/y.bin")
	host := writeArchive(t, filepath.Join(root, "arc"), "graphics.res",
		resFile{path: "x.bin", data: inArchive},
		resFile{path: "d/y.bin", data: deeper})
	loose := filepath.Join(root, "loose")
	looseData := []byte("D:world/data.bin")
	writeLoose(t, loose, "world/data.bin", looseData)

	f := mustOpen(t, []string{host}, []string{loose})
	before := looseTree(t, root)

	// The listing is the caller's: mutating what one enumeration returned
	// changes nothing a later one returns, and the two are equal.
	first := f.Entries()
	kept := append([]vfs.Entry(nil), first...)
	for i := range first {
		first[i] = vfs.Entry{Address: "MUTATED", Size: -1}
	}
	if second := f.Entries(); !reflect.DeepEqual(second, kept) {
		t.Errorf("Entries() after the returned listing was mutated =\n%+v\nwant\n%+v", second, kept)
	}

	// The bytes are the caller's too, on both tiers.
	for _, c := range []struct {
		address string
		want    []byte
	}{
		{"graphics/x.bin", inArchive},
		{"graphics/d/y.bin", deeper},
		{"world/data.bin", looseData},
	} {
		got, err := f.ReadFile(c.address)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", c.address, err)
		}
		for i := range got {
			got[i] ^= 0xFF
		}
		assertBytes(t, f, c.address, c.want)
		if _, ok := f.Locate(c.address); !ok {
			t.Errorf("Locate(%q) reports absent after a successful read of it", c.address)
		}
	}

	// The failing answers write nothing either.
	for _, address := range []string{"nosuch/x.bin", "graphics/nosuch.bin", `\graphics\x.bin`, "../outside.bin", "world/../world/data.bin"} {
		assertAbsent(t, f, address)
		if src, ok := f.Locate(address); ok {
			t.Errorf("Locate(%q) = %+v, want absent", address, src)
		}
	}

	if after := looseTree(t, root); !reflect.DeepEqual(after, before) {
		t.Errorf("the tree under %s changed:\nbefore %v\nafter  %v", root, before, after)
	}
	if got := f.Entries(); !reflect.DeepEqual(got, kept) {
		t.Errorf("Entries() after every read =\n%+v\nwant\n%+v", got, kept)
	}
}
