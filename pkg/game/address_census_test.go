package game_test

// The address census (0027 spec AC-15, SC-8): every container entry a
// library caller resolves, resolved through the filesystem, carrying the
// identity of the container it lives in.
//
// WHERE THE ADDRESSES COME FROM IS THE TEST. Each one enters through the surface
// its own consumer reads — menu.Entries(), the tile grid as LoadTileset walks it,
// the addresses the two loaders and their sheet cache actually ask for, the
// enumeration's own rows — and THIS FILE SPELLS NO ADDRESS OF ITS OWN. A census
// listing addresses it wrote itself would witness only that the test agrees with
// the test; the constant is what shipped code reads, so the constant is what has
// to be asked. The recorder below is what makes that possible for the loaders:
// their addresses are built inside them, from a registry's own [Files] table, and
// cannot be enumerated from outside without rebuilding the rule.
//
// TWO PASSES, because the set is not known until the consumers have asked. Pass
// one runs each consumer against a recorder over a discovery install and collects
// what it asked for. Pass two writes a FRESH install in which every collected
// address's remainder is an entry of the container its identity names — AND of the
// two containers it does not, holding different bytes. The decoys are what make
// the identity segment load-bearing rather than decorative: a resolution that
// dropped it would read the first listed container's copy, and every comparison
// below would fail. Pass two decodes nothing, which is the point — it is about
// addresses, not about formats.
//
// LOOSE FILES ARE OUT OF THE CENSUS BY CONTRACT, not by omission.

import (
	"bytes"
	"errors"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/res"
	"againrom/pkg/game"
	"againrom/pkg/render/menu"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// censusRecorder answers a consumer's reads out of fs and records every address it
// was asked for, in the order it asked.
//
// It satisfies terrain.EntrySource, which is the whole seam the tileset and both
// loaders take — so a consumer handed one behaves exactly as it does in the
// front-end while naming, for the record, every address it resolves.
type censusRecorder struct {
	fs   *vfs.FS
	seen []string
}

func (r *censusRecorder) ReadFile(address string) ([]byte, error) {
	r.seen = append(r.seen, address)
	return r.fs.ReadFile(address)
}

// censusSplit cuts an address into its leading segment and the remainder. ok
// is false for a string holding no separator at all, which carries no
// identity segment and is therefore not an address of a container entry.
func censusSplit(address string) (head, rest string, ok bool) {
	i := strings.IndexByte(address, '/')
	if i < 0 {
		return address, "", false
	}
	return address[:i], address[i+1:], true
}

func censusIdentity(t *testing.T, archive string) string {
	t.Helper()
	identity, err := vfs.Identity(archive)
	if err != nil {
		t.Fatalf("vfs.Identity(%q): %v", archive, err)
	}
	return identity
}

// censusBytes is the pass-two entry payload: bytes that name the container they
// were written into as well as the entry, so no two containers hold equal bytes at
// one remainder and a read can be traced to the container that served it.
func censusBytes(archive, remainder string) []byte {
	return []byte(archive + "|" + remainder)
}

func censusWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func censusOpen(t *testing.T, path string) *vfs.FS {
	t.Helper()
	f, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatalf("vfs.Open(%s): %v", path, err)
	}
	return f
}

// censusRecorded is pass one: every name a library caller asks a container for,
// collected from the caller itself.
func censusRecorded(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()

	// graphics.res is the object loader's fixture archive and the unit loader's
	// side by side, so each registry's [Files] table resolves against sheets this
	// container actually holds and both loaders reach their sheet addresses.
	gpath := filepath.Join(dir, game.GraphicsArchive)
	censusWrite(t, gpath, synth.Archive(append(staticArchiveFiles(t), unitArchiveFiles(t)...)))
	gfs := censusOpen(t, gpath)

	spath := filepath.Join(dir, game.ScenarioArchive)
	censusWrite(t, spath, synth.Archive([]synth.File{
		{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{Width: 4, Height: 4})},
		{Path: "sub/91.ALM", Data: synth.ALM(synth.ALMOptions{Width: 5, Height: 4, Name: "Nested"})},
	}))
	sfs := censusOpen(t, spath)

	// The menu enumerates its own eighteen entries, so nothing has to be read to
	// learn them and no fixture stands in for main.res here.
	got := append([]string{}, menu.Entries()...)

	// The tileset grid, recorded as LoadTileset walks it rather than rebuilt: the
	// count is checked against the render tier's own slot constant, so a grid that
	// silently shrank would fail here instead of quietly shortening the census.
	tiles := &censusRecorder{fs: gfs}
	terrain.LoadTileset(tiles)
	if want := terrain.SlotCount + 1; len(tiles.seen) != want {
		t.Fatalf("the tileset asked for %d entries, want the %d-slot grid plus dirt", len(tiles.seen), want)
	}
	got = append(got, tiles.seen...)

	// Both loaders: their registry constants, and every sheet address the cache's
	// own prefixing rule builds off a registry's [Files] table.
	statics := &censusRecorder{fs: gfs}
	if _, err := game.LoadStatics(statics); err != nil {
		t.Fatalf("LoadStatics over the discovery install: %v", err)
	}
	got = append(got, statics.seen...)

	units := &censusRecorder{fs: gfs}
	if _, err := game.LoadUnits(units); err != nil {
		t.Fatalf("LoadUnits over the discovery install: %v", err)
	}
	got = append(got, units.seen...)

	// The campaign rows. The listing is the enumeration's, so the address is the
	// row's own identity and remainder — and it is CROSS-CHECKED against the
	// reader the picker uses, which is what says the address censused below is the
	// address a row is really read by.
	scenario := censusIdentity(t, game.ScenarioArchive)
	rows := game.ArchiveMaps(sfs)
	if len(rows.Names()) == 0 {
		t.Fatal("the discovery install lists no campaign rows, so the map addresses are not censused")
	}
	for _, name := range rows.Names() {
		address := scenario + "/" + name
		want, err := rows.Read(name)
		if err != nil {
			t.Fatalf("the listed row %q does not read: %v", name, err)
		}
		through, err := sfs.ReadFile(address)
		if err != nil {
			t.Fatalf("the row %q does not resolve at %q: %v", name, address, err)
		}
		if !bytes.Equal(through, want) {
			t.Errorf("the row %q reads bytes the address %q does not", name, address)
		}
		got = append(got, address)
	}
	return got
}

func TestAddressCensus(t *testing.T) {
	identities := map[string]string{}
	for _, archive := range []string{game.MainArchive, game.GraphicsArchive, game.ScenarioArchive} {
		identities[censusIdentity(t, archive)] = archive
	}

	// AC-15's first clause, over every name pass one recorded: it carries an
	// identity segment, and that segment is one a required archive answers.
	//
	// bare collects the one shape that is NOT an address: a class resolving no
	// File at all reaches the sheet cache as the empty path, so the cache asks for
	// an identity with no remainder. The grammar answers it absent, which is the
	// absent-sheet exclusion needing no branch of its own — asserted below rather
	// than dropped, because "not an address" is a claim about it.
	seen := map[string]bool{}
	var addresses, bare []string
	for _, name := range censusRecorded(t) {
		if seen[name] {
			continue
		}
		seen[name] = true
		head, rest, ok := censusSplit(name)
		if !ok || head == "" {
			t.Errorf("a library caller resolves %q, which carries no identity segment (FR-12)", name)
			continue
		}
		if _, known := identities[head]; !known {
			t.Errorf("%q names the identity %q, which no required archive answers", name, head)
			continue
		}
		if rest == "" {
			bare = append(bare, name)
			continue
		}
		addresses = append(addresses, name)
	}
	sort.Strings(addresses)
	t.Logf("censused %d addresses over %d containers, plus %d identity with no remainder",
		len(addresses), len(identities), len(bare))
	if min := terrain.SlotCount + 1 + len(menu.Entries()); len(addresses) < min {
		t.Fatalf("the census holds %d addresses, fewer than the tile grid and the menu alone (%d)", len(addresses), min)
	}

	// Pass two. Every remainder goes into EVERY container, each with bytes naming
	// the container it was written into, so exactly one of the three is the right
	// answer for a given address and the other two are decoys.
	remainders := make([]string, 0, len(addresses))
	held := map[string]bool{}
	for _, address := range addresses {
		_, rest, _ := censusSplit(address)
		if held[rest] {
			continue
		}
		held[rest] = true
		remainders = append(remainders, rest)
	}

	dir := t.TempDir()
	var hosts []string
	direct := map[string]*res.Archive{}
	for identity, archive := range identities {
		files := make([]synth.File, 0, len(remainders))
		for _, rest := range remainders {
			files = append(files, synth.File{Path: rest, Data: censusBytes(archive, rest)})
		}
		path := filepath.Join(dir, archive)
		censusWrite(t, path, synth.Archive(files))
		a, err := res.Open(path)
		if err != nil {
			t.Fatalf("res.Open(%s): %v", path, err)
		}
		direct[identity] = a
	}
	for _, archive := range []string{game.MainArchive, game.GraphicsArchive, game.ScenarioArchive} {
		hosts = append(hosts, filepath.Join(dir, archive))
	}
	containers, err := vfs.Open(hosts, nil)
	if err != nil {
		t.Fatalf("vfs.Open over the census install: %v", err)
	}

	listed := map[string]vfs.Entry{}
	for _, e := range containers.Entries() {
		if _, dup := listed[e.Address]; dup {
			t.Errorf("the enumeration lists %q more than once", e.Address)
		}
		listed[e.Address] = e
	}

	for _, address := range addresses {
		head, rest, _ := censusSplit(address)

		// The bytes a DIRECT READ OF THAT CONTAINER'S OWN PATH yields: the archive
		// reader, opened on the host file the identity names, asked for the
		// remainder. That is AC-15's standard of correctness, and it is a different
		// code path from the one under test.
		want, err := direct[head].ReadFile(rest)
		if err != nil {
			t.Errorf("direct read of %q inside %s: %v", rest, identities[head], err)
			continue
		}
		got, err := containers.ReadFile(address)
		if err != nil {
			t.Errorf("the library address %q does not resolve: %v", address, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%q resolved to %q, want %s's own %q", address, got, identities[head], want)
		}

		// The decoys are checked to BE decoys, so the comparison above is a claim
		// about the identity segment and not about three copies of one file.
		for otherID, other := range direct {
			if otherID == head {
				continue
			}
			decoy, err := other.ReadFile(rest)
			if err != nil {
				t.Errorf("the decoy copy of %q in %s is missing: %v", rest, identities[otherID], err)
				continue
			}
			if bytes.Equal(decoy, want) {
				t.Errorf("%s and %s hold equal bytes at %q, so %q's identity segment decides nothing",
					identities[head], identities[otherID], rest, address)
			}
		}

		if src, ok := containers.Locate(address); !ok || src.Identity != head || src.Tier != vfs.TierArchive {
			t.Errorf("Locate(%q) = %+v ok=%v, want the %q archive", address, src, ok, head)
		}
		e, ok := listed[address]
		if !ok {
			t.Errorf("%q is not in the enumeration, so no listing names the source that serves it", address)
			continue
		}
		if e.Source.Identity != head || e.Size != int64(len(want)) {
			t.Errorf("the enumeration lists %q as %+v size %d, want identity %q size %d",
				address, e.Source, e.Size, head, len(want))
		}
	}

	for _, name := range bare {
		if _, err := containers.ReadFile(name); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%q has an identity but no remainder, so it must be absent; got %v", name, err)
		}
		if _, ok := containers.Locate(name); ok {
			t.Errorf("%q has an identity but no remainder, so nothing may report a source for it", name)
		}
	}
}

// TestGameProductionReachesNoArchiveReader is SC-8's second half: this package's
// production files import pkg/formats/res nowhere.
//
// It is an assertion rather than a note because the property is what the whole
// migration was for — every container entry reached through one dispatch tier —
// and a re-added handle would otherwise pass every other test in the package. The
// import is the right thing to check: the reader's types cannot be named without
// it, and the dependency check next door polices tiers rather than this tier's own
// choice to stop holding a container open.
func TestGameProductionReachesNoArchiveReader(t *testing.T) {
	const reader = "againrom/pkg/formats/res"

	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	fset := token.NewFileSet()
	checked := 0
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		checked++
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: unquote %s: %v", name, imp.Path.Value, err)
			}
			if path == reader {
				t.Errorf("%s imports %s; every container entry this package reads is an address now (0027 FR-12, SC-8)", name, reader)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no production file was parsed, so the absence proves nothing")
	}
}
