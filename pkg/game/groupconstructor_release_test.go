package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// groupConstructorWrites lists the AI bytes a mission-start Group may hold
// other than zero: the constructor's +0x45 (AI-449), the stance byte +0x20
// (AI-451, AI-450), the guard setter's post, centre, spread, sight, reach and
// notice bytes (AI-450, SAV-1218) and the trigger binder's +0x48 (AI-452).
func groupConstructorWrites(owner uint32) map[int]bool {
	writes := map[int]bool{0x45: true, 0x48: true, 0x49: true, 0x4a: true, 0x4b: true}
	switch owner {
	case 0:
	case sim.SelfSlot:
		writes[0x20] = true
	default:
		for _, at := range []int{0x00, 0x01, 0x20, 0x38, 0x39} {
			writes[at] = true
		}
		for at := 0x24; at <= 0x2d; at++ {
			writes[at] = true
		}
	}
	return writes
}

// missionStartGroupAIFaults names every registry Group whose AI bytes differ
// from the constructor and its order: Stand Ground +0x20 = 3 and +0x38 = 0 on
// the controlled Player, Guard +0x20 = 1 and +0x38 = +0x2d on every other
// Player, no stance on owner 0, the constructor's +0x45 = 1 on each, and zero
// in every byte none of them writes.
func missionStartGroupAIFaults(groups []sim.SavedGroup) []string {
	var faults []string
	for _, g := range groups {
		owner := g.Owner.Owner
		var want20, want38 uint8
		switch owner {
		case 0:
		case sim.SelfSlot:
			want20 = uint8(sim.OrderStandGround)
		default:
			want20, want38 = uint8(sim.OrderGuard), g.AI[0x2d]
		}
		if g.AI[0x45] != 1 || g.AI[0x20] != want20 || g.AI[0x38] != want38 {
			faults = append(faults, fmt.Sprintf("Group %d owner %d: +0x45=%d +0x20=%d +0x38=%d, want 1, %d, %d", g.ID, owner, g.AI[0x45], g.AI[0x20], g.AI[0x38], want20, want38))
		}
		writes := groupConstructorWrites(owner)
		for at, b := range g.AI {
			if b != 0 && !writes[at] {
				faults = append(faults, fmt.Sprintf("Group %d owner %d: +0x%02x=%d, which no constructor or setter writes", g.ID, owner, at, b))
			}
		}
	}
	return faults
}

// TestReleaseMissionStartGroupAIIsTheConstructors opens missions 10, 20, 41
// and 80 fresh and checks every registry Group's AI bytes against the
// constructor and its order, and the SAV's G3C bytes against the live ones.
// The loss control LOADs the same SAV as an older engine's, without the Group
// carrier: the registry is then built from the native Group records, carries
// their notice base into the controlled Player's +0x38, and the check fails.
func TestReleaseMissionStartGroupAIIsTheConstructors(t *testing.T) {
	for _, mission := range []int{10, 20, 41, 80} {
		t.Run(fmt.Sprint(mission), func(t *testing.T) {
			f := releaseFront(t)
			f.SetTipsOff(true)
			a := f.App("mission-start Group AI")
			a.SetCutscenes(nil)
			if err := a.OpenMission(f.MissionOpener(mission)); err != nil {
				t.Fatal(err)
			}
			groups, _, present := f.live.world.SavedGroups()
			if !present || len(groups) == 0 {
				t.Fatal("the mission World holds no Group registry")
			}
			self := 0
			for _, g := range groups {
				if g.Owner.Owner == sim.SelfSlot {
					self++
				}
			}
			if self == 0 {
				t.Fatal("the registry holds no controlled Player's Group")
			}
			for _, fault := range missionStartGroupAIFaults(groups) {
				t.Error(fault)
			}
			saved, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.ExportCurrentSave(saved, "mission-start Group AI")
			if err != nil {
				t.Fatal(err)
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			actions, err := readCurrentActions(&doc)
			if err != nil || actions == nil {
				t.Fatal("SAV supplement", err)
			}
			if !actions.GroupsConstructed || actions.GroupsNative {
				t.Fatalf("SAV supplement GroupsConstructed=%v GroupsNative=%v, want true, false", actions.GroupsConstructed, actions.GroupsNative)
			}
			live := map[uint32][76]byte{}
			for _, g := range groups {
				live[g.ID] = g.AI
			}
			for _, r := range actions.Groups {
				ai, held := live[r.ID]
				if wire := crossingRawField(t, &doc.Objects[r.Object-1].Groups[r.Inline], "G3C"); !held || len(wire) < len(ai) || string(wire[:len(ai)]) != string(ai[:]) {
					t.Errorf("Group %d: SAV G3C differs from the live AI bytes", r.ID)
				}
			}

			older := withNativeActions(t, raw, func(m map[string]any) {
				delete(m, "GroupsConstructed")
				m["Policy"].(map[string]any)["GroupCarrier"] = false
			})
			cold := groupCarrierColdLoad(t, older)
			if !cold.live.world.GroupsNative() {
				t.Fatal("loss control: an older engine's SAV did not build its registry from the native Group records")
			}
			restored, _, _ := cold.live.world.SavedGroups()
			copied := 0
			for _, g := range restored {
				if g.Owner.Owner == sim.SelfSlot && g.AI[0x38] != 0 {
					copied++
				}
			}
			if copied == 0 || len(missionStartGroupAIFaults(restored)) == 0 {
				t.Fatalf("loss control: the native copy left %d controlled Group(s) with a notice base, and the check found no fault", copied)
			}
			t.Logf("mission %d: %d Groups, %d controlled; the native copy sets +0x38 on %d controlled Group(s)", mission, len(groups), self, copied)
		})
	}
}
