package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// installShare is the install data every front end over one root reads the
// same way and never writes. Archives and decoded pictures are immutable;
// each front end owns its class descriptors. TestMain compares every share
// with a fresh read after the package's tests.
type installShare struct {
	archives      *Archives
	statics       *terrain.StaticSet
	staticsErr    error
	structures    *terrain.StructureSet
	structuresErr error
	units         *terrain.UnitSet
	unitsErr      error
	unitSounds    map[int32]UnitSound
	townSchool    *ui.TownSchoolArt
	townSchoolErr error
	townTavern    *ui.TownTavernArt
	townTavernErr error
	townSquare    *ui.TownSquareArt
	townSquareErr error
}

// installShareSlot builds one share at most once, even under concurrent
// front-end construction.
type installShareSlot struct {
	once  sync.Once
	share *installShare
	err   error
}

var installShares struct {
	mu    sync.Mutex
	slots map[string]*installShareSlot
}

// installShareKey names an install by its root and each required archive's
// size and modification time, so a rewritten archive is a new share. A root
// whose archives cannot be stated has no key, and its front end reads its
// own copy, reporting whatever OpenArchives reports.
func installShareKey(root string) (string, bool) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	key := root + "|" + abs
	for _, name := range RequiredArchives() {
		st, err := os.Stat(filepath.Join(abs, name))
		if err != nil || !st.Mode().IsRegular() {
			return "", false
		}
		key += fmt.Sprintf("|%s %d %d", name, st.Size(), st.ModTime().UnixNano())
	}
	return key, true
}

// loadInstallShare reads a share with no cache.
func loadInstallShare(root string) (*installShare, error) {
	archives, err := OpenArchives(root)
	if err != nil {
		return nil, err
	}
	match, err := DetectBase(root)
	if err != nil {
		return nil, err
	}
	archives.Base = match
	s := &installShare{archives: archives}
	s.statics, s.staticsErr = LoadStatics(archives.Containers)
	s.structures, s.structuresErr = LoadStructures(archives.Containers)
	s.units, s.unitSounds, s.unitsErr = loadUnitRegistry(archives.Containers)
	s.townSchool, s.townSchoolErr = LoadTownSchoolArt(archives.Containers)
	s.townTavern, s.townTavernErr = LoadTownTavernArt(archives.Containers)
	s.townSquare, s.townSquareErr = LoadTownSquareArt(archives.Containers)
	return s, nil
}

func cloneCandidateStatics(source *terrain.StaticSet) *terrain.StaticSet {
	if source == nil {
		return nil
	}
	out := *source
	classes := map[*terrain.StaticClass]*terrain.StaticClass{}
	var clone func(*terrain.StaticClass) *terrain.StaticClass
	clone = func(class *terrain.StaticClass) *terrain.StaticClass {
		if class == nil {
			return nil
		}
		if copy := classes[class]; copy != nil {
			return copy
		}
		copy := *class
		classes[class] = &copy
		copy.Frames = slices.Clone(class.Frames)
		copy.Timeline = slices.Clone(class.Timeline)
		copy.Dead = clone(class.Dead)
		return &copy
	}
	for i, class := range source.Classes {
		out.Classes[i] = clone(class)
	}
	return &out
}

func cloneCandidateStructures(source *terrain.StructureSet) *terrain.StructureSet {
	if source == nil {
		return nil
	}
	out := *source
	for i, class := range source.Classes {
		if class != nil {
			copy := *class
			copy.Frames = slices.Clone(class.Frames)
			copy.Timeline = slices.Clone(class.Timeline)
			copy.Rank = slices.Clone(class.Rank)
			out.Classes[i] = &copy
		}
	}
	return &out
}

// sharedInstall returns the process's share for root, building it on first
// use.
func sharedInstall(root string) (*installShare, error) {
	key, ok := installShareKey(root)
	if !ok {
		return loadInstallShare(root)
	}
	installShares.mu.Lock()
	if installShares.slots == nil {
		installShares.slots = map[string]*installShareSlot{}
	}
	slot := installShares.slots[key]
	if slot == nil {
		slot = &installShareSlot{}
		installShares.slots[key] = slot
	}
	installShares.mu.Unlock()
	slot.once.Do(func() { slot.share, slot.err = loadInstallShare(root) })
	return slot.share, slot.err
}
