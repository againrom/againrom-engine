package game_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/vfs"
)

// ---------------------------------------------------------------------------
// An in-memory NamedReader
// ---------------------------------------------------------------------------

// stubNames is an in-memory NamedReader. Names() answers in the slice's own
// order, so a test can permute it and show the result does not depend on it.
type stubNames struct {
	names []string
	data  map[string][]byte
}

func newStub() *stubNames { return &stubNames{data: make(map[string][]byte)} }

func (s *stubNames) add(name string, data []byte) *stubNames {
	if _, dup := s.data[name]; dup {
		panic("stubNames: duplicate name " + name)
	}
	s.names = append(s.names, name)
	s.data[name] = data
	return s
}

func (s *stubNames) Names() []string {
	out := make([]string, len(s.names))
	copy(out, s.names)
	return out
}

func (s *stubNames) Read(name string) ([]byte, error) {
	b, ok := s.data[name]
	if !ok {
		return nil, fmt.Errorf("stubNames: no entry %q", name)
	}
	return append([]byte(nil), b...), nil
}

// permuted returns a reader over the same entries whose Names() order is the
// given permutation of this one's.
func (s *stubNames) permuted(order []int) *stubNames {
	out := &stubNames{names: make([]string, 0, len(order)), data: s.data}
	for _, i := range order {
		out.names = append(out.names, s.names[i])
	}
	return out
}

// ---------------------------------------------------------------------------
// Synthetic map bytes
// ---------------------------------------------------------------------------

// mapBytes is a small, fully valid map recording the given name. An empty name
// is the normal case on campaign maps.
func mapBytes(name string) []byte {
	return synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: name})
}

// mapBrokenGrid is well framed — the record walk and the type-0 metadata both
// read — but its type-1 payload length disagrees with 2*W*H, so a full decode
// would fail later. The listing pass reads no more than the metadata, so this
// map is listed AND choosable.
func mapBrokenGrid(name string) []byte {
	return synth.ALM(synth.ALMOptions{
		Width: 4, Height: 4, Name: name,
		Type1Payload: make([]byte, 2*4*4+5),
	})
}

// mapTruncated is a valid map cut short, so the ten records no longer tile to
// EOF and the metadata read itself fails.
func mapTruncated() []byte {
	b := mapBytes("Never Read")
	return b[:len(b)-40]
}

// ---------------------------------------------------------------------------
// The fixture and the hand-derived expectation
// ---------------------------------------------------------------------------

// mapListFixture holds, between the two sources:
//   - maps recording a name and maps whose recorded name is empty;
//   - one source text present in BOTH sources ("dup.alm");
//   - two names differing only by letter case within one source
//     ("Beast.alm" / "beast.alm", both loose);
//   - a cross-source case difference whose byte order disagrees with the
//     loose-before-archive rule ("zone.alm" loose, "Zone.alm" archive: byte-wise
//     "Zone.alm" sorts first, but the loose row must come first anyway);
//   - bytes whose metadata will not decode: garbage ("junk.alm") and a truncated
//     map ("cut.alm");
//   - a well-framed map with a broken grid ("grid.alm").
func mapListFixture() (loose, archive *stubNames) {
	loose = newStub().
		add("Beast.alm", mapBytes("Beast")).
		add("beast.alm", mapBytes("")).
		add("dup.alm", mapBytes("Loose Dup")).
		add("zone.alm", mapBytes("Zone Loose")).
		add("cut.alm", mapTruncated()).
		add("Waters.alm", mapBytes(""))

	archive = newStub().
		add("dup.alm", mapBytes("Arch Dup")).
		add("Zone.alm", mapBytes("Zone Arch")).
		add("junk.alm", []byte("not a map at all, just bytes")).
		add("grid.alm", mapBrokenGrid("Broken Grid")).
		add("aaa.alm", mapBytes("")).
		add("Middle.ALM", mapBytes("Middle"))

	return loose, archive
}

// rowSpec is the observable content of one row.
type rowSpec struct {
	Source      string
	FromArchive bool
	Name        string
	Choosable   bool
	Text        string
}

// wantMapList is the whole expected list, derived by hand from FR-2a and DD19.
//
// Sort key per row: (0 for loose / 1 for archive, strings.ToLower(Source),
// Source compared byte-wise) — the source KIND first, which is the term the
// owner's defect report promoted from a tiebreak to the primary key.
//
// Loose group, by folded key: beast.alm, beast.alm, cut.alm, dup.alm,
// waters.alm, zone.alm.
// Archive group, by folded key: aaa.alm, dup.alm, grid.alm, junk.alm,
// middle.alm, zone.alm.
//
// The three collision cases the fixture exists for resolve as:
//   - "beast.alm": both loose, same folded key, so the case-sensitive term
//     decides — "Beast.alm" ('B' = 0x42) before "beast.alm" ('b' = 0x62);
//   - "dup.alm": equal source texts from the two sources — two rows, and the
//     loose one now leads its whole group rather than merely preceding its twin;
//   - "zone.alm"/"Zone.alm": byte-wise "Zone.alm" would sort first, and the
//     loose row still comes first, because kind outranks every text term.
var wantMapList = []rowSpec{
	// loose
	{"Beast.alm", false, "Beast", true, "Beast.alm - Beast"},
	{"beast.alm", false, "", true, "beast.alm"},
	{"cut.alm", false, "", false, "cut.alm  [unreadable]"},
	{"dup.alm", false, "Loose Dup", true, "dup.alm - Loose Dup"},
	{"Waters.alm", false, "", true, "Waters.alm"},
	{"zone.alm", false, "Zone Loose", true, "zone.alm - Zone Loose"},
	// archive
	{"aaa.alm", true, "", true, "aaa.alm"},
	{"dup.alm", true, "Arch Dup", true, "dup.alm - Arch Dup"},
	{"grid.alm", true, "Broken Grid", true, "grid.alm - Broken Grid"},
	{"junk.alm", true, "", false, "junk.alm  [unreadable]"},
	{"Middle.ALM", true, "Middle", true, "Middle.ALM - Middle"},
	{"Zone.alm", true, "Zone Arch", true, "Zone.alm - Zone Arch"},
}

func rowSpecs(entries []game.MapEntry) []rowSpec {
	out := make([]rowSpec, len(entries))
	for i, e := range entries {
		out[i] = rowSpec{
			Source:      e.Source,
			FromArchive: e.FromArchive,
			Name:        e.Name,
			Choosable:   e.Choosable(),
			Text:        e.Text(),
		}
	}
	return out
}

// fingerprint renders a list including each row's error text, so a determinism
// comparison covers the error state without depending on error identity.
func fingerprint(entries []game.MapEntry) string {
	var b strings.Builder
	for _, e := range entries {
		msg := ""
		if e.Err != nil {
			msg = e.Err.Error()
		}
		fmt.Fprintf(&b, "%q|%v|%q|%v|%q|%q\n",
			e.Source, e.FromArchive, e.Name, e.Choosable(), e.Text(), msg)
	}
	return b.String()
}

func reportRows(t *testing.T, got, want []rowSpec) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	if len(got) != len(want) {
		t.Errorf("map list has %d rows, want %d", len(got), len(want))
	}
	n := len(got)
	if len(want) > n {
		n = len(want)
	}
	for i := 0; i < n; i++ {
		g, w := "<missing>", "<missing>"
		if i < len(got) {
			g = fmt.Sprintf("%+v", got[i])
		}
		if i < len(want) {
			w = fmt.Sprintf("%+v", want[i])
		}
		if g == w {
			continue
		}
		t.Errorf("row %2d:\n\tgot  %s\n\twant %s", i, g, w)
	}
}

func TestBuildMapList(t *testing.T) {
	loose, archive := mapListFixture()

	t.Run("one list in FR-2a order", func(t *testing.T) {
		// The whole slice, not spot checks: the order, the row contents and the
		// row count are one assertion.
		reportRows(t, rowSpecs(game.BuildMapList(loose, archive)), wantMapList)
	})

	// FR-2a's primary sort term, asserted on its own so the property survives any
	// later change to the fixture: the two groups do not interleave, and the
	// boundary between them falls exactly where the sources' sizes put it.
	t.Run("every loose row precedes every archive row", func(t *testing.T) {
		got := game.BuildMapList(loose, archive)

		boundary := len(got)
		for i, e := range got {
			if e.FromArchive {
				boundary = i
				break
			}
		}
		for i, e := range got {
			if want := i >= boundary; e.FromArchive != want {
				t.Fatalf("row %d (%q) FromArchive = %v; the list interleaves the two sources, "+
					"but FR-2a puts every loose row before every archive row (boundary at %d)",
					i, e.Source, e.FromArchive, boundary)
			}
		}
		if boundary != len(loose.Names()) {
			t.Errorf("the loose group holds %d rows, want %d — one per loose input",
				boundary, len(loose.Names()))
		}
	})

	t.Run("every input contributes exactly one row", func(t *testing.T) {
		type key struct {
			source  string
			archive bool
		}
		got := game.BuildMapList(loose, archive)

		count := make(map[key]int, len(got))
		for _, e := range got {
			count[key{e.Source, e.FromArchive}]++
		}
		for _, n := range loose.Names() {
			if c := count[key{n, false}]; c != 1 {
				t.Errorf("loose %q contributed %d rows, want 1", n, c)
			}
		}
		for _, n := range archive.Names() {
			if c := count[key{n, true}]; c != 1 {
				t.Errorf("archive %q contributed %d rows, want 1", n, c)
			}
		}
		if want := len(loose.Names()) + len(archive.Names()); len(got) != want {
			t.Errorf("list holds %d rows, want %d — one per input, nothing dropped or duplicated",
				len(got), want)
		}
	})

	t.Run("equal source texts from the two sources are two rows, loose first", func(t *testing.T) {
		got := game.BuildMapList(loose, archive)

		var dup []game.MapEntry
		for _, e := range got {
			if e.Source == "dup.alm" {
				dup = append(dup, e)
			}
		}
		if len(dup) != 2 {
			t.Fatalf("%q yielded %d rows, want 2 — nothing is deduplicated", "dup.alm", len(dup))
		}
		if dup[0].FromArchive || !dup[1].FromArchive {
			t.Errorf("rows for %q are archive=%v then archive=%v, want the loose one first",
				"dup.alm", dup[0].FromArchive, dup[1].FromArchive)
		}
		if dup[0].Name != "Loose Dup" || dup[1].Name != "Arch Dup" {
			t.Errorf("rows for %q record names %q then %q, want %q then %q — the two rows keep their own metadata",
				"dup.alm", dup[0].Name, dup[1].Name, "Loose Dup", "Arch Dup")
		}
	})

	t.Run("an empty recorded name is still listed and still choosable", func(t *testing.T) {
		got := game.BuildMapList(loose, archive)

		// aaa.alm (archive), beast.alm (loose) and Waters.alm (loose) all record
		// an empty name — the normal case on campaign maps.
		for _, want := range []struct {
			source  string
			archive bool
		}{
			{"aaa.alm", true},
			{"beast.alm", false},
			{"Waters.alm", false},
		} {
			var found *game.MapEntry
			for i := range got {
				if got[i].Source == want.source && got[i].FromArchive == want.archive {
					found = &got[i]
					break
				}
			}
			if found == nil {
				t.Errorf("%q (archive=%v) is missing from the list", want.source, want.archive)
				continue
			}
			if found.Name != "" {
				t.Errorf("%q records name %q, want the empty name the fixture wrote", want.source, found.Name)
			}
			if !found.Choosable() {
				t.Errorf("%q has no recorded name and is not choosable; FR-2 requires it stays choosable, identified by its source alone", want.source)
			}
			if found.Text() != want.source {
				t.Errorf("Text() for %q is %q, want the bare source text %q",
					want.source, found.Text(), want.source)
			}
		}
	})

	t.Run("unreadable metadata stays listed, is distinguished and is not choosable", func(t *testing.T) {
		got := game.BuildMapList(loose, archive)

		for _, want := range []struct {
			source  string
			archive bool
		}{
			{"cut.alm", false}, // a truncated map: the record walk fails
			{"junk.alm", true}, // garbage bytes: the file header fails
		} {
			var found *game.MapEntry
			for i := range got {
				if got[i].Source == want.source && got[i].FromArchive == want.archive {
					found = &got[i]
					break
				}
			}
			if found == nil {
				t.Errorf("%q is missing from the list; FR-2 keeps an unreadable map listed", want.source)
				continue
			}
			if found.Err == nil {
				t.Errorf("%q decoded its metadata, but the fixture's bytes cannot", want.source)
			}
			if found.Choosable() {
				t.Errorf("%q is choosable; FR-2 forbids choosing a map whose metadata cannot be read", want.source)
			}
			if found.Text() == want.source {
				t.Errorf("Text() for %q is just its source text %q, so it is not visibly distinguished from the choosable rows",
					want.source, found.Text())
			}
		}
	})

	t.Run("a well-framed map with a broken grid is listed and choosable", func(t *testing.T) {
		// The Constraint: the listing pass reads no more of a map than its own
		// metadata.
		got := game.BuildMapList(loose, archive)

		var found *game.MapEntry
		for i := range got {
			if got[i].Source == "grid.alm" {
				found = &got[i]
				break
			}
		}
		if found == nil {
			t.Fatalf("grid.alm is missing from the list")
		}
		if found.Err != nil {
			t.Errorf("grid.alm reports %v; its metadata is well framed and must read, since the list reads no more than the metadata", found.Err)
		}
		if !found.Choosable() {
			t.Errorf("grid.alm is not choosable, although its metadata reads")
		}
		if found.Name != "Broken Grid" {
			t.Errorf("grid.alm records name %q, want %q", found.Name, "Broken Grid")
		}
	})

	t.Run("Text shows the source and, when present, the recorded name", func(t *testing.T) {
		got := game.BuildMapList(loose, archive)

		for _, e := range got {
			if !strings.Contains(e.Text(), e.Source) {
				t.Errorf("Text() = %q does not show the source %q", e.Text(), e.Source)
			}
			if e.Name != "" && !strings.Contains(e.Text(), e.Name) {
				t.Errorf("Text() = %q does not show the recorded name %q", e.Text(), e.Name)
			}
		}
	})

	t.Run("the same inputs always give the same list", func(t *testing.T) {
		first := fingerprint(game.BuildMapList(loose, archive))
		second := fingerprint(game.BuildMapList(loose, archive))
		if first != second {
			t.Errorf("two builds over the same sources differ:\nfirst:\n%s\nsecond:\n%s", first, second)
		}
	})

	t.Run("the order Names reports does not change the list", func(t *testing.T) {
		want := fingerprint(game.BuildMapList(loose, archive))

		perms := map[string]func(n int) []int{
			"reversed": func(n int) []int {
				out := make([]int, 0, n)
				for i := n - 1; i >= 0; i-- {
					out = append(out, i)
				}
				return out
			},
			"rotated by 3": func(n int) []int {
				out := make([]int, 0, n)
				for i := 0; i < n; i++ {
					out = append(out, (i+3)%n)
				}
				return out
			},
			"odd indices then even": func(n int) []int {
				out := make([]int, 0, n)
				for i := 1; i < n; i += 2 {
					out = append(out, i)
				}
				for i := 0; i < n; i += 2 {
					out = append(out, i)
				}
				return out
			},
		}
		for label, perm := range perms {
			t.Run(label, func(t *testing.T) {
				l := loose.permuted(perm(len(loose.names)))
				a := archive.permuted(perm(len(archive.names)))
				if got := fingerprint(game.BuildMapList(l, a)); got != want {
					t.Errorf("permuting Names() changed the list:\ngot:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	})
}

func missionListFixture() []game.MapEntry {
	return []game.MapEntry{
		{Source: "Beast.alm", FromArchive: false, Name: "Beast"},
		{Source: "10.alm", FromArchive: false}, // loose, numeric name: not a campaign entry
		{Source: "20.alm", FromArchive: true, Name: "Second"},
		{Source: "npc.alm", FromArchive: true}, // campaign entry, non-numeric stem
		{Source: "10.alm", FromArchive: true, Name: "First"},
		{Source: "13.alm", FromArchive: true, Err: errors.New("bad grid")},
	}
}

func TestWithMissions(t *testing.T) {
	t.Run("mission rows lead, ascending by number, followed by the whole list unchanged", func(t *testing.T) {
		list := missionListFixture()
		got := game.WithMissions(list)

		wantMissions := []int{10, 13, 20}
		if len(got) != len(wantMissions)+len(list) {
			t.Fatalf("WithMissions produced %d rows, want %d (%d mission rows + %d list rows)",
				len(got), len(wantMissions)+len(list), len(wantMissions), len(list))
		}
		for i, n := range wantMissions {
			if got[i].Mission != n {
				t.Errorf("row %d: Mission = %d, want %d — ascending order", i, got[i].Mission, n)
			}
		}
		// The tail is exactly the input list, unchanged, in its own order —
		// WithMissions reads list, it does not rewrite it.
		tail := got[len(wantMissions):]
		if !reflect.DeepEqual(tail, list) {
			t.Errorf("the list's tail differs from the input:\n got  %+v\nwant %+v", tail, list)
		}
	})

	t.Run("a non-numeric stem yields no mission row", func(t *testing.T) {
		got := game.WithMissions(missionListFixture())

		count := 0
		for _, e := range got {
			if e.Source != "npc.alm" {
				continue
			}
			count++
			if e.Mission > 0 {
				t.Errorf("npc.alm produced a mission row: %+v", e)
			}
		}
		if count != 1 {
			t.Errorf("npc.alm appears in %d rows, want exactly 1 — the map row alone, never doubled", count)
		}
	})

	t.Run("a loose entry named as a number yields no mission row", func(t *testing.T) {
		got := game.WithMissions(missionListFixture())

		n := 0
		for _, e := range got {
			if e.Mission == 10 {
				n++
			}
		}
		if n != 1 {
			t.Errorf("mission 10 appears %d times, want exactly 1 — the loose 10.alm must not mint a second one", n)
		}
	})

	t.Run("row text names the mission for a mission row, and a map row's text does not change", func(t *testing.T) {
		got := game.WithMissions(missionListFixture())

		var missionRow, mapRow *game.MapEntry
		for i := range got {
			if got[i].Mission == 20 {
				missionRow = &got[i]
			}
			if got[i].Mission == 0 && got[i].Source == "20.alm" {
				mapRow = &got[i]
			}
		}
		if missionRow == nil || mapRow == nil {
			t.Fatalf("fixture setup: missing the mission row or the map row for 20.alm")
		}
		if !strings.Contains(missionRow.Text(), "20") {
			t.Errorf("mission row text %q does not name the mission number", missionRow.Text())
		}
		if !strings.Contains(missionRow.Text(), "20.alm") {
			t.Errorf("mission row text %q does not name the entry it starts", missionRow.Text())
		}
		if missionRow.Text() == mapRow.Text() {
			t.Errorf("the mission row and the map row render the same text %q; FR-6 requires them "+
				"distinguishable by text alone", missionRow.Text())
		}
		if want := "20.alm - Second"; mapRow.Text() != want {
			t.Errorf("the map row's own text is %q, want %q unchanged — FR-1 leaves every existing row as it was",
				mapRow.Text(), want)
		}
	})

	t.Run("a mission row over an undecodable entry is present and unchoosable", func(t *testing.T) {
		got := game.WithMissions(missionListFixture())

		var found *game.MapEntry
		for i := range got {
			if got[i].Mission == 13 {
				found = &got[i]
			}
		}
		if found == nil {
			t.Fatalf("mission 13 is missing; an undecodable campaign entry must still produce a mission row")
		}
		if found.Choosable() {
			t.Errorf("mission 13 is choosable, but the map row it was built over cannot decode")
		}
		if !strings.Contains(found.Text(), "13") {
			t.Errorf("mission row text %q does not name mission 13", found.Text())
		}
	})
}

func TestDirMapsScan(t *testing.T) {
	root := t.TempDir()

	write := func(dir, rel string, data []byte) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	// Kept: a regular file in the root whose extension case-folds to ".alm",
	// in any letter case. The source text is the bare file name as stored.
	keep := map[string][]byte{
		"a.alm": mapBytes("Scanned Lower"),
		"B.ALM": mapBytes("Scanned Upper"),
		"c.AlM": mapBytes("Scanned Mixed"),
	}
	for name, data := range keep {
		write(root, name, data)
	}

	// The loose filesystem is opened over a SECOND directory, and it holds each
	// kept name FOLDED, with bytes of its own. Two things follow that a single
	// directory could not state. And the file it finds is named in the folded
	// form while the row was listed in the stored one, which is the address
	// fold doing the work; the corollary on a case-sensitive host, where only
	// the folded name resolves at all, is 0027 R-2.
	readRoot := t.TempDir()
	read := map[string][]byte{
		"a.alm": mapBytes("Addressed a"),
		"b.alm": mapBytes("Addressed b"),
		"c.alm": mapBytes("Addressed c"),
	}
	for name, data := range read {
		write(readRoot, name, data)
	}
	loose, err := vfs.Open(nil, []string{readRoot})
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}

	// Dropped: near-misses on the extension.
	write(root, "KIDS.LM", mapBytes("Near Miss"))  // the install really holds one of these
	write(root, "x.al", mapBytes("Short"))         // truncated extension
	write(root, "y.almx", mapBytes("Long"))        // extension is ".almx", not ".alm"
	write(root, "noext", mapBytes("No Extension")) // no extension at all

	// Dropped: the scan covers the asset root itself, not its subdirectories.
	write(root, "sub/deep.alm", mapBytes("In A Subdirectory"))

	// Dropped: a directory is not a regular file, whatever it is called.
	if err := os.Mkdir(filepath.Join(root, "dir.alm"), 0o755); err != nil {
		t.Fatalf("mkdir dir.alm: %v", err)
	}

	r, err := game.DirMaps(root, loose)
	if err != nil {
		t.Fatalf("DirMaps: %v", err)
	}

	t.Run("only root-level .alm regular files are listed", func(t *testing.T) {
		got := r.Names()
		want := make([]string, 0, len(keep))
		for name := range keep {
			want = append(want, name)
		}
		sort.Strings(got)
		sort.Strings(want)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Names() = %q, want %q", got, want)
		}
	})

	t.Run("Read resolves the bare name on the loose filesystem, folded", func(t *testing.T) {
		// Each listed name is expected to yield the FOLDED file's bytes from the
		// filesystem's own root — not the identically named file the scan saw.
		for _, name := range r.Names() {
			want, ok := read[strings.ToLower(name)]
			if !ok {
				t.Fatalf("the fixture has no folded copy of %q", name)
			}
			got, err := r.Read(name)
			if err != nil {
				t.Errorf("Read(%q): %v", name, err)
				continue
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Read(%q) did not return the bytes the loose filesystem holds at %q; "+
					"a row's bytes come from its bare name as an address, not from a read beside it",
					name, strings.ToLower(name))
			}
		}
	})

	t.Run("Read errors on a name that is not there", func(t *testing.T) {
		_, err := r.Read("absent.alm")
		if err == nil {
			t.Fatalf("Read(%q) succeeded on a file that does not exist", "absent.alm")
		}
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Read(%q) failed with %v, want an error satisfying fs.ErrNotExist", "absent.alm", err)
		}
	})

	t.Run("a nonexistent directory is an error", func(t *testing.T) {
		if _, err := game.DirMaps(filepath.Join(root, "no-such-dir"), loose); err == nil {
			t.Errorf("DirMaps succeeded on a directory that does not exist")
		}
	})

	t.Run("no filesystem is an error, not a panic", func(t *testing.T) {
		// A hand-assembled reader has no filesystem; it lists what it scanned and
		// reports that it can read none of it.
		nofs, err := game.DirMaps(root, nil)
		if err != nil {
			t.Fatalf("DirMaps: %v", err)
		}
		if len(nofs.Names()) == 0 {
			t.Errorf("Names() is empty; the scan does not need a filesystem")
		}
		if _, err := nofs.Read("a.alm"); err == nil {
			t.Errorf("Read succeeded with no loose filesystem to read through")
		}
	})
}

// ---------------------------------------------------------------------------
// ArchiveMaps — the archive scan
// ---------------------------------------------------------------------------

func TestArchiveMapsScan(t *testing.T) {
	// The .res reader normalises entry paths to lower case with '/' separators,
	// so "A.ALM" is reported as "a.alm" and that is the row's source text.
	data := map[string][]byte{
		"10.alm":        mapBytes("Ten"),
		"world.reg":     []byte("registry bytes, not a map"),
		"100.alm":       mapBytes(""),
		"pic.bmp":       []byte("bitmap bytes, not a map"),
		"A.ALM":         mapBytes("Upper"),
		"maps/deep.alm": mapBytes("Nested"),
	}
	order := []string{"10.alm", "world.reg", "100.alm", "pic.bmp", "A.ALM", "maps/deep.alm"}

	files := make([]synth.File, 0, len(order))
	for _, p := range order {
		files = append(files, synth.File{Path: p, Data: data[p]})
	}

	// The entries above go into the container CONTAINER-RELATIVE, because that is
	// what an archive holds; the identity segment an address carries is derived from
	// the HOST FILENAME, so laying this archive down as scenario.res is what makes
	// `scenario/10.alm` resolve to it.
	dir := t.TempDir()
	lay := func(name string, files []synth.File) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, synth.Archive(files), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		return p
	}
	scenarioPath := lay(game.ScenarioArchive, files)

	// A DECOY container, listed FIRST so list order cannot do the work an identity
	// is supposed to: it holds "10.alm" under a name the campaign container also
	// holds, with other bytes, and a ".alm" entry of its own. A listing filtered on
	// the extension alone would list `decoy.alm`; a read composing the wrong
	// identity would return the decoy's ten.
	decoyTen := mapBytes("Decoy Ten")
	mainPath := lay(game.MainArchive, []synth.File{
		{Path: "10.alm", Data: decoyTen},
		{Path: "decoy.alm", Data: mapBytes("Decoy Only")},
	})

	containers, err := vfs.Open([]string{mainPath, scenarioPath}, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}

	r := game.ArchiveMaps(containers)

	// remainder as the reader reports the entry path -> the bytes packed there
	want := map[string][]byte{
		"10.alm":        data["10.alm"],
		"100.alm":       data["100.alm"],
		"a.alm":         data["A.ALM"],
		"maps/deep.alm": data["maps/deep.alm"],
	}

	t.Run("only the campaign container's .alm entries are listed, by their entry paths", func(t *testing.T) {
		got := r.Names()
		wantNames := make([]string, 0, len(want))
		for p := range want {
			wantNames = append(wantNames, p)
		}
		sort.Strings(got)
		sort.Strings(wantNames)
		if !reflect.DeepEqual(got, wantNames) {
			t.Errorf("Names() = %q, want %q", got, wantNames)
		}
		for _, n := range got {
			if strings.HasSuffix(n, ".reg") || strings.HasSuffix(n, ".bmp") {
				t.Errorf("Names() includes the non-map entry %q", n)
			}
			if n == "decoy.alm" {
				t.Errorf("Names() includes %q, a .alm entry of another listed container; "+
					"the identity selects the container, not the extension", n)
			}
		}
	})

	t.Run("Read resolves the scenario address, not the same name elsewhere", func(t *testing.T) {
		for name, data := range want {
			got, err := r.Read(name)
			if err != nil {
				t.Errorf("Read(%q): %v", name, err)
				continue
			}
			if !reflect.DeepEqual(got, data) {
				t.Errorf("Read(%q) returned %d bytes, want the %d packed into %s",
					name, len(got), len(data), game.ScenarioArchive)
			}
		}
		if got, err := r.Read("10.alm"); err == nil && reflect.DeepEqual(got, decoyTen) {
			t.Errorf("Read(%q) returned the decoy container's bytes; the address names the wrong identity", "10.alm")
		}
	})

	t.Run("a name the campaign container does not hold is not readable", func(t *testing.T) {
		if _, err := r.Read("decoy.alm"); err == nil {
			t.Errorf("Read(%q) succeeded; that entry is another container's", "decoy.alm")
		}
	})

	t.Run("no filesystem lists nothing and reads an error, not a panic", func(t *testing.T) {
		nofs := game.ArchiveMaps(nil)
		if got := nofs.Names(); len(got) != 0 {
			t.Errorf("Names() = %q, want nothing listed with no filesystem to enumerate", got)
		}
		if _, err := nofs.Read("10.alm"); err == nil {
			t.Errorf("Read succeeded with no filesystem to read through")
		}
	})
}
