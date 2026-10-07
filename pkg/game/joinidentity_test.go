package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

func localizedJoinFront(t *testing.T, names map[int]string) *FrontEnd {
	t.Helper()
	lines := make([]string, 64)
	for i, name := range names {
		lines[i] = name
	}
	root := t.TempDir()
	dir := filepath.Join(root, "main", "text")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "npcnames.txt"), []byte(strings.Join(lines, "\r\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open(nil, []string{root})
	if err != nil {
		t.Fatal(err)
	}
	return &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Containers: fsys}}}
}

func TestJoinedScenarioHeroesUseLocalizedCanonicalNamesInTheLivePanelAndCarry(t *testing.T) {
	for _, tc := range []struct {
		locale string
		brian  string
		naira  string
	}{
		{"EN", "Brian", "Naira"},
		{"RU", "Брайан", "Наира"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			f := localizedJoinFront(t, map[int]string{21: tc.naira, 24: tc.brian})
			ms := &Mission{Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{
				6: {ID: "join:6", Name: "PC_Paladin", PlayerCharacter: true, CompanionNPC: 25, Class: 42},
				7: {ID: "join:7", Name: "PC_Naira", PlayerCharacter: true, CompanionNPC: 23, Class: 14},
			}}}
			f.canonicalizeJoinedRoster(ms)
			if ms.Start.Roster[6].Name != tc.brian || ms.Start.Roster[7].Name != tc.naira {
				t.Fatalf("localized roster names = %q/%q, want %q/%q",
					ms.Start.Roster[6].Name, ms.Start.Roster[7].Name, tc.brian, tc.naira)
			}
			chars := missionCharacters(nil, nil, ms)
			if chars[6].Name != tc.brian || chars[7].Name != tc.naira {
				t.Fatalf("live panel names = %q/%q, want %q/%q",
					chars[6].Name, chars[7].Name, tc.brian, tc.naira)
			}

			w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
				[]sim.Entity{
					{ID: 6, X: 1, Y: 1, HP: 20, MaxHP: 20, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID},
					{ID: 7, X: 2, Y: 1, HP: 20, MaxHP: 20, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID},
				})
			if err != nil {
				t.Fatal(err)
			}
			carried := mapload.CarryRoster(nil, w, nil, ms.Start.Roster)
			if len(carried) != 2 || carried[0].Name != tc.brian || carried[1].Name != tc.naira {
				t.Fatalf("boundary identities = %+v, want %q then %q", carried, tc.brian, tc.naira)
			}
			encoded, err := EncodeSave(Snapshot{Party: carried}, "joined heroes")
			if err != nil {
				t.Fatal(err)
			}
			decoded, _, err := DecodeSave(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if len(decoded.Party) != 2 || decoded.Party[0].Name != tc.brian || decoded.Party[1].Name != tc.naira ||
				decoded.Party[0].CompanionNPC != 25 || decoded.Party[1].CompanionNPC != 23 {
				t.Fatalf("save/load identities = %+v, want localized canonical companion rows 25 and 23", decoded.Party)
			}
		})
	}
}
