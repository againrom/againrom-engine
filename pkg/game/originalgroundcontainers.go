package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// liveGroundContainerTail resolves one file Sack's own container tail
// through the SavedObjectContainer ledger importSavedSackObjects already
// populates (savsackobjects.go) — the single live carrier DIV-962 keeps a
// ground Sack's own +0x1c/+0x20 pair on, joined here by the archive Identity
// both sides key on, exactly like the file-vs-live corpus audit
// (originalgroundcontainers1136_corpus_test.go).
//
// present is false when importSavedSackObjects declined to adopt this file
// Sack at all (ambiguous cell, non-unique identity, no matching empty native
// Sack, and so on: savsackobjects.go's own coverage reasons) — a real,
// pre-existing outcome this function does not treat as an error or paper
// over with an invented value.
func liveGroundContainerTail(registry *sim.SavedObjects, identity uint32) (sav.GroundContainerTail, bool) {
	if registry == nil {
		return sav.GroundContainerTail{}, false
	}
	for _, s := range registry.Sacks {
		if s.Token.Identity != identity {
			continue
		}
		container, ok := savedSackContainer(registry, s.ID)
		if !ok {
			return sav.GroundContainerTail{}, false
		}
		return sav.GroundContainerTail{Identity: identity, InsertIndex: container.InsertIndex, Accumulator: container.Accumulator}, true
	}
	return sav.GroundContainerTail{}, false
}

// exportOriginalGroundContainers writes a world's live ground-container
// tails back into a decoded original save's own world half.
//
// It reads the file's own current Sack list first and rewrites only the two
// tail dwords each already has a recorded offset for (sav.SetGroundContainerTails),
// on exportOriginalCellRecords' own contract: no Sack, Item or Effect record
// is added, removed or relocated. A Sack importSavedSackObjects declined to
// adopt has no live tail to re-derive, so export refuses rather than fall
// back to the file's own stale copy — DIV-962's own corpus census finds zero
// such Sacks over the full preserved corpus, EN and RU.
//
// NO PRODUCTION CALLER INVOKES THIS. This project has no production mission
// SAVE path yet (owner rule); it is an unshipped round-trip building block,
// checked only by the corpus audit
// (originalgroundcontainers1136_corpus_test.go).
//
// IT NEVER TOUCHES THE BETWEEN-MISSION FORM, on exportOriginalCellRecords'
// own rule: a city save has no world half, so f.World == nil there refuses.
func exportOriginalGroundContainers(f *sav.File, w *sim.World) error {
	if f == nil || f.World == nil {
		return fmt.Errorf("original ground containers export: this save has no world session")
	}
	if w == nil {
		return fmt.Errorf("original ground containers export: nil world")
	}
	source, present, err := f.GroundSacks()
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("original ground containers export: this save has no world session")
	}
	registry := w.SavedObjects()
	tails := make([]sav.GroundContainerTail, len(source))
	for i, sack := range source {
		tail, ok := liveGroundContainerTail(registry, sack.Identity)
		if !ok {
			return fmt.Errorf("original ground containers export: Sack %#x has no live container", sack.Identity)
		}
		tails[i] = tail
	}
	return f.SetGroundContainerTails(tails)
}
