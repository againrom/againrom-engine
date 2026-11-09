package game

import (
	"crypto/sha256"
	"os"
	"sync"
	"testing"
)

// sharedOriginalFront resumes one original save once per process and root, for
// every corpus instrument that only reads the result. Each caller receives the
// same front and Mission and must not write to them: resumes through
// loadOriginalFront stay private, which a writer keeps using. The World hash
// recorded at the first resume is compared when each caller's subtest ends, so
// an instrument that writes to the shared World fails where it wrote.
func sharedOriginalFront(t *testing.T, saved []byte) (*FrontEnd, error) {
	t.Helper()
	assets := os.Getenv("AGAINROM_ASSETS")
	key := sharedOriginalKey{root: assets, saved: sha256.Sum256(saved)}
	sharedOriginals.Lock()
	if sharedOriginals.slots == nil {
		sharedOriginals.slots = map[sharedOriginalKey]*sharedOriginalSlot{}
	}
	slot := sharedOriginals.slots[key]
	if slot == nil {
		slot = &sharedOriginalSlot{}
		sharedOriginals.slots[key] = slot
	}
	sharedOriginals.Unlock()
	slot.once.Do(func() {
		base, err := sharedResumeBase(assets)
		if err != nil {
			slot.err = err
			return
		}
		slot.front, slot.err = loadOriginalFront(base, saved)
		if slot.err == nil {
			slot.hash = slot.front.CurrentMission().World.Hash()
		}
	})
	if slot.err != nil {
		return nil, slot.err
	}
	world := slot.front.CurrentMission().World
	t.Cleanup(func() {
		if got := world.Hash(); got != slot.hash {
			t.Errorf("an instrument wrote to the shared resumed World: hash %x, resumed %x", got, slot.hash)
		}
	})
	return slot.front, nil
}

// sharedOriginalMission is sharedOriginalFront for a caller that needs only
// the Mission and its report.
func sharedOriginalMission(t *testing.T, saved []byte) (*Mission, OriginalSaveResume, error) {
	t.Helper()
	front, err := sharedOriginalFront(t, saved)
	if err != nil {
		return nil, OriginalSaveResume{}, err
	}
	mission := front.CurrentMission()
	return mission, mission.OriginalSaveReport(), nil
}

// sharedResumeBase is the one front per root every shared resume copies, so
// the memo holds one decoded install however many saves it holds.
func sharedResumeBase(assets string) (*FrontEnd, error) {
	sharedOriginals.Lock()
	defer sharedOriginals.Unlock()
	if base, ok := sharedOriginals.bases[assets]; ok {
		return base, nil
	}
	base, err := decodedInstallFront(assets)
	if err != nil {
		return nil, err
	}
	base.SetDeterministicFrames(true)
	if sharedOriginals.bases == nil {
		sharedOriginals.bases = map[string]*FrontEnd{}
	}
	sharedOriginals.bases[assets] = base
	return base, nil
}

type sharedOriginalKey struct {
	root  string
	saved [sha256.Size]byte
}

type sharedOriginalSlot struct {
	once  sync.Once
	front *FrontEnd
	hash  uint64
	err   error
}

var sharedOriginals struct {
	sync.Mutex
	slots map[sharedOriginalKey]*sharedOriginalSlot
	bases map[string]*FrontEnd
}
