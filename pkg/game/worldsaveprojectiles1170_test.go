package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestWorld1170ProjectileConstructorAndRetirement(t *testing.T) {
	f := spell1152FixtureFront(t)
	doc, err := sav.DecodeDocumentData(completeDocumentFixture1115(t, f))
	if err != nil {
		t.Fatal(err)
	}
	current := sim.SavedProjectiles{FreeIndex: 713, IDs: []uint16{11, 7, 11}, Items: []sim.SavedProjectile{{ID: 7, X: 1, Y: 2, Picture: 3, Phase: 1, Action: 2, ActionX: 5, ActionY: 6, ActionSegments: 4}, {ID: 11, X: 9, Y: 10, ActionTarget: 73, ActionPhase: 2, ActionSegments: 3}}}
	if err := projectSavedProjectiles(&doc, current, []uint16{7, 11}); err != nil {
		t.Fatal(err)
	}
	wire, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(wire)
	if err != nil {
		t.Fatal(err)
	}
	got, present, err := file.Projectiles()
	if err != nil || !present || got.FreeIndex != 713 || !slices.Equal(got.IDs, current.IDs) || len(got.Items) != 2 {
		t.Fatal("current allocator/order/multiplicity/sections lost", got, err)
	}
	for _, p := range got.Items {
		if p.ID == 11 && (p.X != 9 || p.Y != 10 || p.ActionTarget != 73 || p.ActionSegments != 3) {
			t.Fatal("current fields lost", p)
		}
	}
	current.Items = current.Items[:1]
	current.IDs = []uint16{7}
	if err := projectSavedProjectiles(&doc, current, []uint16{7, 11}); err != nil {
		t.Fatal(err)
	}
	for _, r := range doc.State.DirectoryRecords {
		if foldStatePath(r.Path) == "/prj11" {
			t.Fatal("retired projectile section resurrected")
		}
	}
	for i, r := range doc.State.ValueRecords {
		if foldStatePath(r.Path) == "/prj7/x" {
			doc.State.ValueRecords = append(doc.State.ValueRecords[:i], doc.State.ValueRecords[i+1:]...)
			break
		}
	}
	if err := projectSavedProjectiles(&doc, current, []uint16{7, 11}); err == nil {
		t.Fatal("partial existing record accepted as a constructor")
	}
}
