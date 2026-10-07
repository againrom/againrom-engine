package game

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/vfs"
)

// mapExt is the map file extension, matched without regard to case: a stock
// install ships both `Kids.alm` and `Tomb.ALM`.
const mapExt = ".alm"

// scenarioIdentity is scenario.res's address identity and scenarioPrefix is
// that identity with its separator: the prefix of every map address this
// package resolves out of the campaign container.
//
// Both are DERIVED from the archive name this package already owns, through
// the one rule the filesystem itself applies to a host path — not spelt as
// a second literal beside ScenarioArchive. A second spelling is how a
// renamed container leaves the map list asking for an identity nothing in
// the set answers: every row would still list, off the enumeration, and
// every read of one would fail.
//
// ScenarioArchive is a constant whose stem is non-empty, so the derivation cannot
// fail here; the panic is what says so, in the MustCompile shape, and it can only
// fire if that constant is edited into something no address could ever name.
var (
	scenarioIdentity = mustIdentity(ScenarioArchive)
	scenarioPrefix   = scenarioIdentity + "/"
)

func mustIdentity(archive string) string {
	identity, err := vfs.Identity(archive)
	if err != nil {
		panic("game: " + err.Error())
	}
	return identity
}

// NamedReader is one place maps come from: the names it holds, and the bytes
// behind a name.
//
// The map list is built from two of these rather than from a directory path and
// an archive, so the whole of the listing contract — the ordering, the metadata
// read, which rows are choosable — is decidable against in-memory stubs. The
// cases that matter most are the awkward ones: the same source text present in
// both sources, and two names differing only in case.
type NamedReader interface {
	Names() []string
	Read(name string) ([]byte, error)
}

// MapEntry is one row of the map list.
//
// Name is the map's own recorded name, which is EMPTY on most campaign maps —
// normal input, not a failure. Err is non-nil when the map's metadata would not
// decode; such a row is still listed, because dropping it would hide part of an
// install rather than report it.
type MapEntry struct {
	Source      string // the file name, or the entry name inside the archive
	FromArchive bool
	Name        string
	Err         error

	// Width, Height and Description are the map's own decoded size and
	// authored prose (alm.Info), zero/empty for a row Err marks unreadable.
	// They are the map-selection list's row-owned metadata
	// (TEXT-HOVERTEXT-052); DIV-1270 carries what this build shows of it.
	Width, Height int
	Description   string

	Word70, Word74 uint32

	// Mission is the campaign mission number this row starts, or ZERO for an
	// ordinary map row. A mission row is built OVER a map row by WithMissions
	// and carries that row's own Source, Name and Err unchanged — there is no
	// second read and no second rule for whether it can be chosen: Choosable()
	// below reads Err exactly as it always has, so a mission row is choosable
	// exactly when the map row it was made from is, by construction rather than
	// by a second arm here.
	Mission int
}

// Choosable reports whether the map can be picked.
func (e MapEntry) Choosable() bool { return e.Err == nil }

// Text renders the row: its source, the recorded name when there is one, and a
// mark when the metadata would not read.
//
// It is ASCII because this screen's text is placeholder debug text drawn with
// the engine's built-in font; rendering the game's own fonts is a later story.
// The source comes first because it is the only thing every row has — a row
// leading with an empty recorded name would start with a blank.
//
// A MISSION ROW LEADS WITH ITS NUMBER. The number is the one fact about a
// mission row that is not already true of the map row it was built over, so
// it is what has to come first for the two to read apart by their text alone
// — everything after the leading branch is the ordinary rendering,
// unchanged, so a mission row still names the entry it starts from and still
// carries the unreadable mark when its map row does.
func (e MapEntry) Text() string {
	s := ""
	if e.Mission > 0 {
		s = fmt.Sprintf("Mission %d: ", e.Mission)
	}
	s += e.Source
	if e.Name != "" {
		s += " - " + e.Name
	}
	if e.Err != nil {
		s += "  [unreadable]"
	}
	return s
}

// BuildMapList assembles the one map list from a loose-file source and an
// archive source.
//
// Ordering puts every LOOSE row before every archive row, then orders each group
// case-insensitively by the source text that row displays, ties inside a group
// broken by comparing that same text case-sensitively. Nothing is deduplicated:
// a loose file and an archive entry whose source texts are equal are two rows,
// the loose one first.
//
// The source kind is the primary key on purpose, and it used to be only a
// tiebreak. On a stock install scenario.res holds 28 campaign maps named
// 10.alm..91.alm which record no name of their own, while the 10 loose files are
// named Beast.ALM..Waters.alm and every one of them records a name. Digits sort
// before letters, so ranking by source text alone put all 28 nameless rows ahead
// of all 10 named ones and the picker's visible window held nothing but bare
// numbers. Sorting on "does this row have a name" instead would be worse: the
// name is decoded at runtime, an unreadable row has none, and one campaign map
// does have one -- so the groups would depend on bytes rather than on where a
// row came from.
//
// Each map is read only as far as its own metadata. A stock install holds 38
// maps and this runs on every start, including the headless one, so decoding
// every grid and content table to learn a name would be paid on every launch.
// It also keeps two outcomes apart that a full decode would merge: a map whose
// metadata will not read is listed and unchoosable here, while a map that reads
// fine and then fails to load is something only choosing it can discover.
func BuildMapList(loose, archive NamedReader) []MapEntry {
	var out []MapEntry
	add := func(src NamedReader, fromArchive bool) {
		if src == nil {
			return
		}
		for _, name := range src.Names() {
			e := MapEntry{Source: name, FromArchive: fromArchive}
			data, err := src.Read(name)
			if err != nil {
				e.Err = err
			} else if info, err := alm.OpenInfo(data); err != nil {
				e.Err = err
			} else {
				e.Name = info.Name
				e.Width, e.Height, e.Description = info.Width, info.Height, info.ListDescription
				e.Word70, e.Word74 = info.Word70, info.Word74
			}
			out = append(out, e)
		}
	}
	add(loose, false)
	add(archive, true)

	// SliceStable, so even two rows equal on all three keys keep the order they
	// were appended in and the list is reproducible run to run.
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.FromArchive != b.FromArchive {
			return !a.FromArchive // loose first
		}
		if fa, fb := strings.ToLower(a.Source), strings.ToLower(b.Source); fa != fb {
			return fa < fb
		}
		return a.Source < b.Source
	})
	return out
}

// WithMissions returns the mission rows built from list's own campaign
// entries, followed by list UNCHANGED.
//
// BuildMapList answers one question — what maps does this install hold — and
// this asks a second one OVER its result rather than folding into it: which
// of those maps also open as a mission, and in what order. Composing the two
// at the call site is what keeps BuildMapList's own ordering rule, and every
// assertion already written against it, untouched: WithMissions never
// rewrites list, only reads it and prepends what it derives, so every row
// that was listed before is still listed, in its own order, with its own
// text.
//
// A row YIELDS a mission row exactly when it is FromArchive and its stem —
// the source with its path and extension removed — parses as a positive
// decimal integer: the campaign container's own rule, and MissionMap's own
// inverse (strconv.Itoa there, strconv.Atoi here), so no mission number, map
// name or campaign extent is ever spelled as a constant here. Add a map to
// the container and its row appears; rename one and its row moves; a loose
// file mints no mission row no matter what its name reads as, because it is
// not a campaign entry regardless.
//
// Ordering is ascending by the parsed number, sort.SliceStable so two rows
// that (in a modded install) claimed the same number keep the order the
// underlying list already put them in. Ordering by text instead would put
// mission 100 before mission 20.
func WithMissions(list []MapEntry) []MapEntry {
	missions := make([]MapEntry, 0, len(list))
	for _, e := range list {
		n, ok := missionNumber(e)
		if !ok {
			continue
		}
		missions = append(missions, MapEntry{
			Source:      e.Source,
			Name:        e.Name,
			Err:         e.Err,
			Mission:     n,
			Width:       e.Width,
			Height:      e.Height,
			Description: e.Description,
			Word70:      e.Word70,
			Word74:      e.Word74,
		})
	}
	sort.SliceStable(missions, func(i, j int) bool {
		return missions[i].Mission < missions[j].Mission
	})

	out := make([]MapEntry, 0, len(missions)+len(list))
	out = append(out, missions...)
	out = append(out, list...)
	return out
}

// missionNumber answers the mission number a map row's own name yields, and
// whether it yields one at all.
//
// Only a FromArchive row is asked at all: a loose file is never a campaign
// entry, whatever its name reads as. The stem asked about is the source's
// file-name component with its extension removed — exactly what MissionMap
// composes back into an address with strconv.Itoa, so the two are inverses
// of one another and neither can drift from the other's idea of what a
// mission number looks like. A stem that does not parse, or parses to zero
// or a negative number, answers no row: campaign maps are numbered from one,
// and a modded container's stray `bonus.alm` is silence here rather than a
// row nothing could ever open.
func missionNumber(e MapEntry) (int, bool) {
	if !e.FromArchive {
		return 0, false
	}
	base := e.Source
	if i := strings.LastIndexAny(base, "/\\"); i >= 0 {
		base = base[i+1:]
	}
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	n, err := strconv.Atoi(stem)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// dirMaps lists the loose maps in one directory and reads them through the loose
// filesystem over it.
type dirMaps struct {
	loose *vfs.FS
	names []string
}

// DirMaps lists the map files directly inside root, and reads them through loose,
// the read-only filesystem over that same root.
//
// The LISTING IS STILL A SCAN, and legitimately so: the filesystem never
// enumerates its directory tier, so nothing else can say which loose files
// an install ships. The scan is NOT recursive: it covers the asset root
// itself and not its subdirectories, so an install's `Allods` or `Help`
// folder cannot contribute rows. A name is kept when its extension
// case-folds to `.alm`, which is what lets `Kids.alm` and `Tomb.ALM` both
// list while a near-miss like `KIDS.LM` does not.
//
// The READ is the flip: a row's bytes come from its own bare file name
// resolved as an address on the loose filesystem — a separator-less
// address, which names no container identity and so reaches the loose tier
// exactly as every address does. This package holds no second code path onto
// the host beside it.
//
// An address is FOLDED before it is looked up, so the row a scan listed as
// `Tomb.ALM` is read as `tomb.alm` under the root. That is the disclosed
// case-folding limitation (0027 C-3) reaching shipped behaviour: on a
// case-sensitive host such a file stops resolving and its row lists as unreadable
// rather than vanishing (0027 R-2). Shipped installs live on hosts that fold.
func DirMaps(root string, loose *vfs.FS) (NamedReader, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", root, err)
	}
	d := &dirMaps{loose: loose}
	for _, ent := range ents {
		if ent.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(ent.Name()), mapExt) {
			continue
		}
		d.names = append(d.names, ent.Name())
	}
	return d, nil
}

func (d *dirMaps) Names() []string { return d.names }

func (d *dirMaps) Read(name string) ([]byte, error) {
	if d.loose == nil {
		return nil, fmt.Errorf("no loose filesystem")
	}
	return d.loose.ReadFile(name)
}

// archiveMaps lists the maps the campaign container holds, over the container
// filesystem the whole set is addressed through.
type archiveMaps struct {
	containers *vfs.FS
	names      []string
}

// ArchiveMaps lists the map entries scenario.res holds, over containers.
//
// The listing is the filesystem's own ENUMERATION filtered to the scenario
// identity, not a scan of one archive handle. The identity is what selects
// the container — equality on the address's leading segment, so a `.alm`
// entry in another listed container is not a campaign map and is not listed
// — and this package names scenario.res once, in that identity, holding no
// reader of its own on the file.
//
// A row's source text is the address's REMAINDER: the entry path inside the
// container, which is exactly the path the archive reader reported before the
// flip, so every displayed row, every ordering key and the picker's whole
// appearance are unchanged.
//
// The scan is flat: an archive stored as an entry inside another archive is one
// opaque blob and is not descended into. On the shipped scenario.res that is
// exactly right — its 28 maps sit at the root beside three registry files, and
// those 31 records are the whole container: it nests no archive at all.
func ArchiveMaps(containers *vfs.FS) NamedReader {
	m := &archiveMaps{containers: containers}
	if containers == nil {
		return m
	}
	for _, e := range containers.Entries() {
		if e.Source.Identity != scenarioIdentity {
			continue
		}
		name := strings.TrimPrefix(e.Address, scenarioPrefix)
		if !strings.EqualFold(filepath.Ext(name), mapExt) {
			continue
		}
		m.names = append(m.names, name)
	}
	return m
}

func (m *archiveMaps) Names() []string { return m.names }

func (m *archiveMaps) Read(name string) ([]byte, error) {
	if m.containers == nil {
		return nil, fmt.Errorf("no container filesystem")
	}
	return m.containers.ReadFile(scenarioPrefix + name)
}
