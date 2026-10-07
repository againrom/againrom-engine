package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// A second living source survives each loss. Requiring merely one successful
// comparison must not let a missing living or supported dying subject become
// a raw-only exclusion. The fixture writes stage and signed HP into the source;
// no importer-produced population decides which records are required.
func TestUnit1156MissingLiveSubjectControls(t *testing.T) {
	for _, stage := range []byte{0, 1} {
		t.Run(fmt.Sprintf("source-stage-%d", stage), func(t *testing.T) {
			f := poolFixtureFront(t, 91, 92)
			f.Campaign = resolved(saveCampaign(), nil)
			hp := uint16(31)
			if stage == 1 {
				hp = 65505 // signed -31, already dying in the original source
			}
			target := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: hp, maxHP: 101, mana: 23, maxMana: 103, stage: stage, profile: literalProfile1107(), name: "target"}
			survivor := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 31, maxHP: 101, mana: 23, maxMana: 103, profile: literalProfile1107(), name: "survivor"}
			raw := completeDocumentTail1115(t, f, savedContainer(poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{{target, survivor}}}}, nil)))
			source, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			want, err := unit1156Expected(source, raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(want.records) != 2 || want.records[0].values["Stage"] != uint32(stage) || uint16(want.records[0].values["Health"]) != hp {
				t.Fatal("literal source population or stage/HP changed")
			}
			ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
			if err != nil {
				t.Fatal(err)
			}
			unit1156InitialCheck(t, want, ms)
			entities := ms.World.Entities()
			baseline, excluded, n := want.entityDifferences(entities, ms.ActorManifest)
			if len(baseline) != 0 || len(excluded) != 0 || n != 2 {
				t.Fatal("two-source baseline", baseline, excluded, n)
			}
			index := -1
			for i, e := range entities {
				if e.SourceBinding.ArchiveIndex == want.records[0].archive {
					index = i
				}
			}
			if index < 0 {
				t.Fatal("baseline target has no carrier")
			}
			for _, bindingOnly := range []bool{false, true} {
				t.Run(fmt.Sprintf("binding-only-%t", bindingOnly), func(t *testing.T) {
					bad := slices.Clone(entities)
					if bindingOnly {
						bad[index].SourceBinding = sim.SourceBinding{}
					} else {
						bad = slices.Delete(bad, index, index+1)
					}
					differences, excluded, n := want.entityDifferences(bad, ms.ActorManifest)
					prefix := fmt.Sprintf("archive %d ", want.records[0].archive)
					if len(differences) != 1 || !strings.HasPrefix(differences[0], prefix) || !strings.Contains(differences[0], "expected live scalar") || len(excluded) != 0 || n != 1 {
						t.Fatalf("missing required source was not rejected: compared=%d differences=%v exclusions=%v", n, differences, excluded)
					}
				})
			}
		})
	}
}
