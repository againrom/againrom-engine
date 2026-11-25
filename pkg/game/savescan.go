package game

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/savename"
)

// saveFacts is what a directory scan needs from one file: whether it opens as a
// SAV, its label bytes, and per game asked whether the document belongs to
// that game. sav.Open decompresses and indexes the whole body, so the label
// cannot come from a shorter read without changing which files count as
// corrupt. The facts are computed once per file version and kept for the
// process.
type saveFacts struct {
	size   int64
	mod    time.Time
	opens  bool
	label  []byte
	ofGame bool
	// envelope is the game the envelope names; empty for Asg&.
	envelope string
}

var saveFactsCache = struct {
	sync.Mutex
	byKey map[string]*saveFacts
	reads int // files read in full; a test proves a repeat scan reads none
}{byKey: map[string]*saveFacts{}}

func saveFactsReads() int {
	saveFactsCache.Lock()
	defer saveFactsCache.Unlock()
	return saveFactsCache.reads
}

func factsOfBytes(b []byte, size int64, mod time.Time) *saveFacts {
	f := &saveFacts{size: size, mod: mod}
	if sf, err := sav.Open(b); err == nil {
		f.opens = true
		f.label = append([]byte(nil), sf.Label...)
		f.envelope = sf.EnvelopeGame()
	}
	return f
}

// readSaveFacts answers the facts of dir/name from the cache while the file's
// size and modification time are unchanged. bounded applies the save reader's
// size limit; the install reader is unbounded. withGame adds the game check;
// its entry is kept apart per game.
func readSaveFacts(dir, name string, fi os.FileInfo, bounded bool, game base.Game, withGame bool) (saveFacts, bool) {
	key := filepath.Join(dir, name)
	if withGame {
		key += "|" + fmt.Sprint(game)
	}
	saveFactsCache.Lock()
	if e := saveFactsCache.byKey[key]; e != nil && e.size == fi.Size() && e.mod.Equal(fi.ModTime()) {
		out := *e
		saveFactsCache.Unlock()
		return out, true
	}
	saveFactsCache.Unlock()

	var b []byte
	var err error
	if bounded {
		b, err = (SaveStore{Dir: dir}).Read(name)
	} else {
		b, err = os.ReadFile(filepath.Join(dir, name))
	}
	if err != nil {
		return saveFacts{}, false
	}
	fresh := factsOfBytes(b, fi.Size(), fi.ModTime())
	fresh.ofGame = fresh.opens && withGame && validateOriginalGame(b, game) == nil
	saveFactsCache.Lock()
	saveFactsCache.reads++
	saveFactsCache.byKey[key] = fresh
	out := *fresh
	saveFactsCache.Unlock()
	return out, true
}

// originalSaveLabelOf is a save's list row: its description alone, as the
// original's Load list draws it, with no mission or town suffix.
func originalSaveLabelOf(raw []byte, selector int) string {
	label := asciiLabel(raw, selector)
	if label == "" {
		label = "(no label)"
	}
	return label
}

// saveScan is a save directory's row of listProfileFiles. Filters: the save
// extension and non-directories always; Bounded applies the save reader's size
// limit and bounded read (the install reader has neither); Regular admits
// regular files only; OfGame keeps the saves whose document belongs to Game;
// First stops at the first save that passes. A file whose envelope names a
// game other than Game, the zero Game being the first game's, never passes.
type saveScan struct {
	Dir     string
	Bounded bool
	Regular bool
	OfGame  bool
	Game    base.Game
	First   bool
}

// ownsEnvelope reports that f's envelope names no game or Game.
func (s saveScan) ownsEnvelope(f *saveFacts) bool {
	_, err := ownsEnvelope(f.envelope, s.Game)
	return err == nil
}

// scannedSave is one save that passed the scan, newest first in the result.
type scannedSave struct {
	Name  string
	Label []byte
	Mod   time.Time
}

// rowLabel is the list row text OriginalSaveLabel gives the same file.
func (s scannedSave) rowLabel(selector int) string {
	return originalSaveLabelOf(s.Label, selector)
}

// scanSaves lists the saves that pass the scan, newest first, ties by name
// descending. A missing or unreadable directory yields nothing; a file that
// will not read or open is skipped.
func scanSaves(s saveScan) []scannedSave {
	out, _, _ := listProfileFiles(profileListing[scannedSave]{dir: s.Dir, exts: savename.Extensions(), regular: s.Regular,
		first: s.First, read: s.read, order: newestSaveFirst})
	return out
}

// newestSaveFirst orders saves by modification time, newest first, then by
// file name descending.
func newestSaveFirst(a, b scannedSave) int {
	if c := b.Mod.Compare(a.Mod); c != 0 {
		return c
	}
	return cmp.Compare(b.Name, a.Name)
}

// read is the scan's row of the file e in dir: a file that will not read or
// open is an error; one the scan's game or envelope filter refuses is passed
// over.
func (s saveScan) read(dir string, e os.DirEntry) (scannedSave, bool, error) {
	fi, err := e.Info()
	if err != nil {
		// The install reader lists a file whose stat fails, read uncached
		// and without a modification time.
		if s.Bounded {
			return scannedSave{}, false, err
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return scannedSave{}, false, err
		}
		f := factsOfBytes(b, 0, time.Time{})
		if !f.opens {
			return scannedSave{}, false, errSaveWillNotOpen
		}
		return scannedSave{Name: e.Name(), Label: f.label}, s.ownsEnvelope(f), nil
	}
	if s.Bounded && (fi.Size() < 0 || fi.Size() > maxSaveBytes) {
		return scannedSave{}, false, errSaveTooLarge
	}
	f, ok := readSaveFacts(dir, e.Name(), fi, s.Bounded, s.Game, s.OfGame)
	if !ok || !f.opens {
		return scannedSave{}, false, errSaveWillNotOpen
	}
	if (s.OfGame && !f.ofGame) || !s.ownsEnvelope(&f) {
		return scannedSave{}, false, nil
	}
	return scannedSave{Name: e.Name(), Label: f.label, Mod: fi.ModTime()}, true, nil
}

var (
	errSaveWillNotOpen = errors.New("save will not read or open")
	errSaveTooLarge    = errors.New("save exceeds the size bound")
)
