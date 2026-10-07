package game

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// corpseBundle is a bundle of five classes, each frame slice its own pointer
// so identity distinguishes them, and the dying map beside it:
//
//	10 -> 11   a sibling: the human-class shape, dying into another class
//	11 -> 11   itself
//	12 -> 99   a class this bundle does not hold
//	13 -> 10   two hops if anything walked them: 13 -> 10 -> 11
//	14 -> 14   itself, and frameless — a link is a fact about the class
func corpseBundle() (map[int32]*terrain.UnitClass, map[int32]int32) {
	classes := map[int32]*terrain.UnitClass{
		10: {Frames: []*terrain.StaticFrame{{Width: 1, Height: 1}}},
		11: {Frames: []*terrain.StaticFrame{{Width: 1, Height: 1}}},
		12: {Frames: []*terrain.StaticFrame{{Width: 1, Height: 1}}},
		13: {Frames: []*terrain.StaticFrame{{Width: 1, Height: 1}}},
		14: {},
	}
	return classes, map[int32]int32{10: 11, 11: 11, 12: 99, 13: 10, 14: 14}
}

// Each class's link is exactly the class its own key names — and the two-hop
// case stops where the key points, not one class beyond it.
func TestTheCorpseLinkIsOneHop(t *testing.T) {
	classes, dying := corpseBundle()
	linkCorpses(classes, dying)
	// Read through the bundle type the loader hands over, not through the map
	// the helper was handed: what a consumer sees is what is asserted.
	classes = (&terrain.UnitSet{Classes: classes}).Classes

	for _, tc := range []struct {
		label string
		of    int32
		want  int32 // 0 means "no link at all"
	}{
		{"a class dying into a sibling", 10, 11},
		{"a class naming itself", 11, 11},
		{"a class naming one the bundle does not hold", 12, 0},
		{"two hops: the link is the class named, not the one beyond it", 13, 10},
		{"a frameless class still carries its link", 14, 14},
	} {
		got := classes[tc.of].Corpse
		if tc.want == 0 {
			if got != nil {
				t.Errorf("%s: class %d links to something, want nothing", tc.label, tc.of)
			}
			continue
		}
		if got != classes[tc.want] {
			t.Errorf("%s: class %d does not link to class %d", tc.label, tc.of, tc.want)
		}
	}

	// Said again as the thing a chain walk would get wrong: 13's link is NOT
	// 11, which is where a walk of two hops would land.
	if classes[13].Corpse == classes[11] {
		t.Errorf("class 13 links to class 11 — the key names 10, so the resolution walked a chain")
	}
}

// The link is a fact about the class, so every class in the bundle is visited
// and none is skipped, dropped or replaced.
func TestLinkingTouchesEveryClassAndReplacesNone(t *testing.T) {
	classes, dying := corpseBundle()
	before := make(map[int32]*terrain.UnitClass, len(classes))
	frames := make(map[int32]int, len(classes))
	for id, c := range classes {
		before[id], frames[id] = c, len(c.Frames)
	}

	linkCorpses(classes, dying)

	if len(classes) != len(before) {
		t.Fatalf("the bundle holds %d classes, held %d — linking added or dropped one",
			len(classes), len(before))
	}
	for id, c := range classes {
		if c != before[id] {
			t.Errorf("class %d is a different value than before linking", id)
		}
		if len(c.Frames) != frames[id] {
			t.Errorf("class %d holds %d frames, held %d", id, len(c.Frames), frames[id])
		}
	}
	// Only 12 names a class the bundle lacks, so exactly one link is absent.
	absent := 0
	for _, c := range classes {
		if c.Corpse == nil {
			absent++
		}
	}
	if absent != 1 {
		t.Errorf("%d classes carry no link, want exactly 1 — only class 12 names a missing one", absent)
	}
}

// Nothing in the walk depends on a Go map's per-process order: the links are
// the same however many times it is run, and running it again over an already
// linked bundle changes nothing.
func TestLinkingIsOrderIndependentAndIdempotent(t *testing.T) {
	classes, dying := corpseBundle()
	linkCorpses(classes, dying)
	first := make(map[int32]*terrain.UnitClass, len(classes))
	for id, c := range classes {
		first[id] = c.Corpse
	}
	for i := 0; i < 8; i++ {
		linkCorpses(classes, dying)
		for id, c := range classes {
			if c.Corpse != first[id] {
				t.Fatalf("run %d: class %d's link moved", i+2, id)
			}
		}
	}

	// And an id in the dying map naming no class in the bundle is ignored
	// rather than minting one.
	dying[77] = 10
	linkCorpses(classes, dying)
	if _, ok := classes[77]; ok {
		t.Errorf("linking created class 77 out of a dying entry")
	}
}
