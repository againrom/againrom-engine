package vfs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"againrom/pkg/formats/res"
)

// identityMax is the byte length an archive identity is cut to. It is a
// property of the address grammar rather than of any container: two host
// filenames agreeing in their first identityMax bytes after the fold address
// one and the same archive set.
const identityMax = 15

// FS is a read-only filesystem over an ordered list of .res archives and an
// ordered list of loose host directories. An address is an archive identity, a
// separator, and the path inside that archive; an address no archive answers is
// resolved as a relative path under each listed directory in turn.
//
// There is no close operation and nothing to release: an archive holds no host
// handle once open (C-2). The zero value is not usable; construct one with Open.
type FS struct {
	archives []archiveSource
	dirs     []string
	// overlay holds addresses a caller added after opening: the bytes served
	// ahead of every archive and directory. Enumeration does not list them.
	overlay map[string]func() ([]byte, error)
}

// archiveSource is one listed archive: the identity derived from its host path,
// the open reader, and the entry snapshot taken at open.
type archiveSource struct {
	identity string
	archive  archiveReader
	snap     snapshot
}

type archiveReader interface {
	Entries() []res.Entry
	ReadFile(string) ([]byte, error)
}

// snapshot is one archive's entry index, taken once at open: the unique
// folded entry paths in registry order, first record per path — the
// archive reader's own resolution — and a position index over them. A read
// needs presence alone; the ordered entries carry the order and the sizes
// any walk over this archive follows.
type snapshot struct {
	entries []snapEntry
	index   map[string]int
}

// snapEntry is one entry of a snapshot: the folded path its archive serves it
// under, and the size of its bytes.
type snapEntry struct {
	path string
	size int64
}

// Tier names which of the two tiers a source belongs to. The zero value is
// TierNone, so a Source no lookup filled in never reads as position 0 of either
// list.
type Tier int

const (
	// TierNone is the absent answer's tier: nothing serves the address.
	TierNone Tier = iota
	// TierArchive is a listed archive, named by its identity and its position.
	TierArchive
	// TierDir is a listed directory, named by its position alone.
	TierDir
)

// String names the tier, for a diagnostic.
func (t Tier) String() string {
	switch t {
	case TierArchive:
		return "archive"
	case TierDir:
		return "dir"
	default:
		return "none"
	}
}

// Source is where an address resolves: the tier, the archive identity —
// empty for the directory tier, which no address names — and the position
// of that source in the list Open was given.
//
// It is comparable, and it is the one shape both the enumeration and the
// per-address report are built from, so "these two answers name the same
// source" is an equality of values rather than two structures to reconcile.
type Source struct {
	Tier     Tier
	Identity string
	Index    int
}

// Entry is one enumerated address: the folded address, the source serving
// it, and the size of the bytes a read of it returns.
type Entry struct {
	Address string
	Source  Source
	Size    int64
}

// Open opens every archive in archives, in list order, and records dirs as the
// loose-file tier beneath them. Either list may be empty.
//
// The open is atomic: the first archive that cannot be opened, or whose
// derived identity is empty, fails the whole call, which returns a nil *FS
// and retains no archive's bytes. A listed directory is not validated here
// — it is read only when an address reaches it.
//
// A failure to open an archive reads open <host>: <err>, naming the host path
// the caller listed.
func Open(archives, dirs []string) (*FS, error) {
	return openArchives(archives, dirs, func(host string) (archiveReader, error) {
		return res.Open(host)
	})
}

// OpenFileBacked opens the same address space as Open while retaining only
// each archive's registry. Payloads are read from disk one at a time. It is for
// large optional containers such as MUSIC.RES, not the five startup archives
// whose existing in-memory lifetime is unchanged.
func OpenFileBacked(archives, dirs []string) (*FS, error) {
	return openArchives(archives, dirs, func(host string) (archiveReader, error) {
		return res.OpenFileIndex(host)
	})
}

func openArchives(archives, dirs []string, open func(string) (archiveReader, error)) (*FS, error) {
	f := &FS{
		archives: make([]archiveSource, 0, len(archives)),
		dirs:     append([]string(nil), dirs...),
	}
	for _, host := range archives {
		identity, err := Identity(host)
		if err != nil {
			return nil, err
		}
		a, err := open(host)
		if err != nil {
			return nil, &fs.PathError{Op: "open", Path: host, Err: err}
		}
		f.archives = append(f.archives, archiveSource{
			identity: identity,
			archive:  a,
			snap:     snapshotOf(a),
		})
	}
	return f, nil
}

// Identity derives an archive's address identity from its host path, and
// from nothing else: the text after the last separator, truncated before the
// first '.', folded, and cut to identityMax bytes.
//
// A host path yielding an empty identity is refused. Nothing could address
// such an archive: an address whose leading segment is empty is not
// addressable at all, so the archive would be unreachable rather than merely
// oddly named.
func Identity(hostPath string) (string, error) {
	stem := hostPath
	if i := strings.LastIndexAny(stem, `\/`); i >= 0 {
		stem = stem[i+1:]
	}
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	stem = fold(stem)
	if len(stem) > identityMax {
		stem = stem[:identityMax]
	}
	if stem == "" {
		return "", fmt.Errorf("vfs: %s: the host filename yields an empty archive identity", hostPath)
	}
	return stem, nil
}

// ReadFile returns the bytes of the entry at address. The returned slice is
// the caller's own: mutating it changes nothing a later read of the same
// address returns, and no read, of either tier, writes anything.
//
// An address held by no archive and no listed directory yields no bytes and
// an error satisfying errors.Is(err, fs.ErrNotExist) that names the address.
// A source that holds the address but fails to produce its bytes surfaces
// its own error instead, and resolution stops there.
func (f *FS) ReadFile(address string) ([]byte, error) {
	folded := fold(address)
	if read, ok := f.overlay[folded]; ok {
		return read()
	}
	head, rest := cut(folded)
	if head == "" {
		// An address whose leading segment is empty — the empty address, or one
		// beginning with a separator — is not addressable, and is absent before
		// either tier is consulted.
		return nil, notExist(address)
	}
	if b, served, err := f.readArchive(head, rest); err != nil || served {
		return b, err
	}
	if b, served, err := f.readDirs(folded); err != nil || served {
		return b, err
	}
	return nil, notExist(address)
}

// Overlay serves address from read ahead of every archive and directory. It is
// how a launch adds a file the install does not hold: the read function runs on
// each read and returns the caller's own slice. Overlay is not safe to call
// while other goroutines read, so a caller adds its addresses before any read.
// Entries and Locate report the archives and directories only.
func (f *FS) Overlay(address string, read func() ([]byte, error)) {
	if f.overlay == nil {
		f.overlay = map[string]func() ([]byte, error){}
	}
	f.overlay[fold(address)] = read
}

// Fork is a filesystem over the same archives and directories with its own
// overlay, so a caller can add addresses without touching the filesystem it
// was forked from. The archives are shared and read-only.
func (f *FS) Fork() *FS {
	c := &FS{archives: f.archives, dirs: f.dirs}
	if len(f.overlay) > 0 {
		c.overlay = make(map[string]func() ([]byte, error), len(f.overlay))
		for k, v := range f.overlay {
			c.overlay[k] = v
		}
	}
	return c
}

// Entries lists every address the archives serve, each exactly once, in
// ascending order of the folded address — plain byte order, which is what
// comparing the folded strings is. Every entry names the archive a read of
// that address resolves to and the size of the bytes that read returns.
//
// The listing is the read's winner. Archives are walked in list order and
// each snapshot in registry order, so within an archive the first record of
// a twice-held path is the one listed — the snapshot already holds only
// that — and across two archives sharing one identity the earlier
// archive's record is, which is the archive a read resolves to.
//
// The directory tier is never listed: nothing was indexed for it and nothing
// is walked to build one.
//
// The returned slice is the caller's own; mutating it changes no later listing.
func (f *FS) Entries() []Entry {
	n := 0
	for i := range f.archives {
		n += len(f.archives[i].snap.entries)
	}
	out := make([]Entry, 0, n)
	for i := range f.archives {
		src := &f.archives[i]
		for _, e := range src.snap.entries {
			if e.path == "" {
				// No address reaches it: an address is an identity, a separator and a
				// NON-EMPTY remainder, so a bare "<identity>/" is refused.
				continue
			}
			if f.shadowed(i, src.identity, e.path) {
				continue
			}
			out = append(out, Entry{
				Address: src.identity + "/" + e.path,
				Source:  Source{Tier: TierArchive, Identity: src.identity, Index: i},
				Size:    e.size,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Address < out[j].Address })
	return out
}

// Locate reports which source serves address, and reads no bytes to do it:
// the archive named by its identity and its position, the directory named by
// its position, or the zero Source and false for an address nothing serves.
//
// It stops where a read stops, because it asks the same two questions of the
// same data: which archive answers, through the one function that decides
// that, and which directory holds the address, by the one regular-file test
// that decides that. A directory holding the address as a file whose read
// fails is still the source named here — the read then surfaces that
// failure rather than continuing past it.
func (f *FS) Locate(address string) (Source, bool) {
	folded := fold(address)
	head, rest := cut(folded)
	if head == "" {
		// Not addressable, and so absent before either tier.
		return Source{}, false
	}
	if i, _, ok := f.findArchive(head, rest); ok {
		return Source{Tier: TierArchive, Identity: f.archives[i].identity, Index: i}, true
	}
	if i, _, ok := f.findDir(folded, 0); ok {
		return Source{Tier: TierDir, Index: i}, true
	}
	return Source{}, false
}

// readArchive serves an address from the archive tier. served reports that
// an archive answered. A non-nil error is a source that held the address and
// failed to produce its bytes, which stops resolution rather than letting a
// copy in a later archive, or under a listed directory, mask it.
func (f *FS) readArchive(head, rest string) (b []byte, served bool, err error) {
	i, key, ok := f.findArchive(head, rest)
	if !ok {
		return nil, false, nil
	}
	data, rerr := f.archives[i].archive.ReadFile(key)
	if rerr != nil {
		// Unreachable through the reader this tier is built on — it fails
		// not-found alone, and the snapshot has just said the entry is there. It
		// is here so that no source failure can ever be masked, whatever a later
		// reader learns to fail with.
		return nil, false, rerr
	}
	return data, true, nil
}

// findArchive decides which archive answers an address, and is the only
// thing that decides it: the read, the report and the enumeration all
// resolve through here, so none of them can name an archive another would
// not. It returns that archive's position and the snapshot key the read
// uses.
func (f *FS) findArchive(head, rest string) (int, string, bool) {
	// The archive reader keys on the path with its outer separators trimmed, so
	// the probe below applies that same rule and cannot disagree with the read
	// that follows it. An empty remainder is no entry at all: the address is a
	// bare identity, which no archive answers.
	key := strings.Trim(rest, "/")
	if key == "" {
		return 0, "", false
	}
	for i := range f.archives {
		src := &f.archives[i]
		// Equality, never a prefix relation: with graphics.res listed,
		// "graphicsx/x.bin" and "graphic/x.bin" name no archive.
		if src.identity != head {
			continue
		}
		if _, held := src.snap.index[key]; !held {
			continue
		}
		// The first archive of this identity holding the remainder answers, and it
		// is the only archive consulted.
		return i, key, true
	}
	return 0, "", false
}

// shadowed reports whether an archive listed before position i, sharing
// identity, already holds path. It is the same question findArchive answers
// — same snapshots, same order, same key — so the record the enumeration
// drops is exactly the record a read passes over.
func (f *FS) shadowed(i int, identity, path string) bool {
	for j := 0; j < i; j++ {
		if f.archives[j].identity != identity {
			continue
		}
		if _, held := f.archives[j].snap.index[path]; held {
			return true
		}
	}
	return false
}

// readDirs serves an address from the loose-file tier: the whole folded
// address, identity segment included, as a relative path under each listed
// directory in order. A directory holds the address iff a regular file
// stands at that path; the first holder answers, and anything else standing
// there is not held, so the walk continues to the next directory.
func (f *FS) readDirs(folded string) (b []byte, served bool, err error) {
	for from := 0; ; {
		i, host, ok := f.findDir(folded, from)
		if !ok {
			return nil, false, nil
		}
		from = i + 1
		data, rerr := os.ReadFile(host)
		if rerr != nil {
			if errors.Is(rerr, fs.ErrNotExist) {
				// It stood there a moment ago and does not now: this directory
				// does not hold the address, so the walk continues.
				continue
			}
			return nil, false, rerr
		}
		return data, true, nil
	}
}

// findDir walks the listed directories from position from onwards and
// returns the position and host path of the first one holding folded. It is
// the only thing that decides whether a directory holds an address, so the
// read and the report stop at the same directory.
//
// The host path is joined from the folded segments verbatim, never through
// filepath.Clean, and an address carrying a ".", ".." or empty segment is
// held by no directory at all: cleaning "a/../b" would resolve outside the
// root the caller listed.
func (f *FS) findDir(folded string, from int) (int, string, bool) {
	segs, ok := segments(folded)
	if !ok {
		return 0, "", false
	}
	for i := from; i < len(f.dirs); i++ {
		host := f.dirs[i]
		for _, seg := range segs {
			host += string(filepath.Separator) + seg
		}
		info, serr := os.Stat(host)
		if serr != nil || !info.Mode().IsRegular() {
			continue
		}
		return i, host, true
	}
	return 0, "", false
}

// snapshotOf takes one archive's entry snapshot. The reader's entry paths are
// already in its canonical key form, so nothing is folded again here.
func snapshotOf(a archiveReader) snapshot {
	entries := a.Entries()
	s := snapshot{index: make(map[string]int, len(entries))}
	for _, e := range entries {
		if _, seen := s.index[e.Path]; seen {
			// A path held twice is served by its first record, in the reader and
			// so in the snapshot over it.
			continue
		}
		s.index[e.Path] = len(s.entries)
		s.entries = append(s.entries, snapEntry{path: e.Path, size: e.Size})
	}
	return s
}

// fold maps an address, or a host filename stem, to the one form everything
// is compared in: '\' becomes '/', ASCII 'A'-'Z' becomes 'a'-'z', and every
// other byte — including every byte of a multi-byte sequence decoded from
// a high CP866 entry name — stands for itself.
//
// Nothing is trimmed.
func fold(s string) string {
	b := []byte(s)
	for i, c := range b {
		switch {
		case c == '\\':
			b[i] = '/'
		case c >= 'A' && c <= 'Z':
			b[i] = c - 'A' + 'a'
		}
	}
	return string(b)
}

// cut splits a folded address into its leading segment and the remainder
// past the first separator. An address carrying no separator is all leading
// segment, and is refused by neither this function nor the caller: it names
// no archive entry, and the loose tier is consulted for it as for every
// address.
func cut(folded string) (head, rest string) {
	if i := strings.IndexByte(folded, '/'); i >= 0 {
		return folded[:i], folded[i+1:]
	}
	return folded, ""
}

// segments splits a folded address into the path segments a listed directory is
// probed with, reporting whether the address is addressable under one at all.
//
// The segments are joined onto the listed root verbatim, never through
// filepath.Clean, and an address carrying a ".", ".." or empty segment is
// not addressable by this tier: cleaning "a/../b" would resolve outside the
// root the caller listed.
func segments(folded string) ([]string, bool) {
	if folded == "" {
		return nil, false
	}
	segs := strings.Split(folded, "/")
	for _, seg := range segs {
		switch seg {
		case "", ".", "..":
			return nil, false
		}
	}
	return segs, true
}

// notExist is the absent answer: an error naming the address as the caller
// wrote it, and satisfying errors.Is(err, fs.ErrNotExist).
func notExist(address string) error {
	return &fs.PathError{Op: "readfile", Path: address, Err: fs.ErrNotExist}
}
