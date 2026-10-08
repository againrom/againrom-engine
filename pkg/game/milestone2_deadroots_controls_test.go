package game

import (
	"encoding/binary"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func dead1163Fixture(t *testing.T) ([]byte, dead1163Source, *Mission) {
	t.Helper()
	f := poolFixtureFront(t, 91, 92)
	runtimeID := uint32(73)
	a := &poolFixtureActor{cell: 0x0807, hp: 0xff13, maxHP: 30, stage: 4, human: true, runtime: &runtimeID}
	b := *a // Equal values, distinct constructed archive identity.
	zero := uint32(0)
	terminal := &poolFixtureActor{mapID: 92, cell: 0x0a09, hp: 0xd8ef, maxHP: 30, stage: 5, runtime: &zero}
	raw := completeDocumentTail1115(t, f, savedContainer(poolFixtureBody([]*poolFixturePlayer{{}, {}}, []*poolFixtureActor{a, &b, a, terminal})))
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := dead1163Expected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	return raw, want, ms
}

func TestDeadRoots1163OrderMultiplicityAndIdentityControls(t *testing.T) {
	_, want, ms := dead1163Fixture(t)
	if len(want.roots) != 4 || len(want.actors) != 3 || want.roots[0] != want.roots[2] || want.roots[0] == want.roots[1] {
		t.Fatal("literal repeated/equal actor roots lost", want.roots)
	}
	if want.actors[0].State != want.actors[1].State || want.actors[0].Identity == want.actors[1].Identity || want.actors[2].State.HP != -10001 {
		t.Fatal("literal identity/state anchors changed")
	}
	if d := want.documentDifferences(ms.savedDocument); len(d) != 0 {
		t.Fatal("positive Document", d)
	}
	if d := want.liveDifferences(ms.World.OriginalDeadActors(), ms.World.Entities()); len(d) != 0 {
		t.Fatal("positive live", d)
	}
	for _, name := range []string{"omit-repeat", "permute", "collapse-equal"} {
		t.Run(name, func(t *testing.T) {
			bad := cloneSavedDocumentFixture(t, ms.savedDocument)
			r := bad.Document.DeadActors
			switch name {
			case "omit-repeat":
				bad.Document.DeadActors = append(r[:2], r[3:]...)
			case "permute":
				r[0], r[1] = r[1], r[0]
			case "collapse-equal":
				r[1] = r[0]
			}
			if d := want.documentDifferences(bad); len(d) == 0 {
				t.Fatal("equal World fields hid root loss")
			}
		})
	}
	for _, name := range []string{"omit-virtual", "collapse-native", "swap-order", "normalize-terminal", "source-ref", "held-weapon"} {
		t.Run(name, func(t *testing.T) {
			bad := slices.Clone(ms.World.OriginalDeadActors())
			switch name {
			case "omit-virtual":
				bad = bad[1:]
			case "collapse-native":
				bad[1].ID = bad[0].ID
			case "swap-order":
				bad[0], bad[1] = bad[1], bad[0]
			case "normalize-terminal":
				bad[2].Source.State.HP = -10017
			case "source-ref":
				bad[0].Source.References[2] ^= 1
			case "held-weapon":
				bad[2].Source.HeldWeapon = sim.OriginalDeadWeapon{Present: true}
			}
			if d := want.liveDifferences(bad, ms.World.Entities()); len(d) == 0 {
				t.Fatal("accepted dead source/identity loss")
			}
		})
	}
}

func TestDeadRoots1163CountAndStartControls(t *testing.T) {
	raw, _, _ := dead1163Fixture(t)
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	loc, err := source.DocumentDeadRootLocation()
	if err != nil {
		t.Fatal(err)
	}
	// This structural API returns a detached slice.
	first := loc.RefOffs[0]
	loc.RefOffs[0]++
	again, err := source.DocumentDeadRootLocation()
	if err != nil || again.RefOffs[0] != first {
		t.Fatal("locator aliases cached state", err)
	}
	for _, value := range []uint32{0, 3, 5, 0xffffffff} {
		bad := *source
		bad.Body = slices.Clone(source.Body)
		binary.LittleEndian.PutUint32(bad.Body[loc.CountOff:], value)
		if _, err := dead1163Expected(&bad, raw); err == nil {
			t.Fatal("accepted changed raw root count", value)
		}
	}
}
