package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// heroLabelInspector is an inspector of profile p over an empty archive whose
// font takes selector sel, with speakers 25 and 26 in its NPC catalog.
func heroLabelInspector(t *testing.T, p base.Profile, sel int) *MapInspector {
	t.Helper()
	path := filepath.Join(t.TempDir(), GraphicsArchive)
	if err := os.WriteFile(path, synth.Archive(nil), 0o600); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &MapInspector{tiles: &terrain.Tileset{}, rules: []mapRule{}, font: &text.Font{Selector: sel},
		archives: &Archives{Containers: fsys, Base: base.Match{Profile: p}},
		table: &mapload.Table{NPCCatalog: &data.NPCCatalog{Entries: []data.NPCEntry{
			{ID: 21, Section: "npc21"}, {ID: 25, Section: "npc25", Name: "Oracle"}, {ID: 26, Section: "npc26", Name: "Gatekeeper"}}}}}
}

// heroLoseMap is a map whose lose trigger fires when hero 2 (10002) dies,
// with a check no trigger names on hero 1005 (11005).
func heroLoseMap(t *testing.T) []byte {
	t.Helper()
	actions := []scriptNode{{label: "lose", opcode: 5, id: 3}}
	conditions := []scriptNode{
		{label: "dead", opcode: 5, id: 1, values: []uint32{10002}, tags: []uint32{4}},
		{label: "zero", opcode: 0x10002, id: 2, values: []uint32{0}, tags: []uint32{7}},
		{label: "far", opcode: 5, id: 3, values: []uint32{11005}, tags: []uint32{4}},
	}
	o := synth.ALMOptions{Width: 32, Height: 32, Type7Payload: goalScript(actions, conditions, []goalTrigger{
		{name: "Lose", left: 1, right: 2, actions: []uint32{3}},
	})}
	return built(t, o, nil, nil).Bytes()
}

// openHeroLabelMap opens map b in x's editor.
func openHeroLabelMap(t *testing.T, x *MapInspector, b []byte) *ui.MapEditor {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hero.alm")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	return e
}

// On a first-game map every hero band label names the hero's number and the
// label of speaker 20+N as the message popup builds it, in the install's
// language: the script's unit choices (heroes 1 to 1000 and a hero above it
// the script names) and the arms list's target. Heroes 5 and 6 take the NPC
// catalog's names, as the popup does. On a second-game map the label stays a
// number. The stored value does not change.
func TestHeroLabelsNameTheSpeakerRoleOnAFirstGameMap(t *testing.T) {
	for _, c := range []struct {
		name    string
		profile base.Profile
		sel     int
		choice  map[uint32]string
		goal    string
	}{
		{"en", base.Profile{Language: "english"}, 0, map[uint32]string{
			10001: "Hero 1: Main hero", 10002: "Hero 2: Traitor mage", 10003: "Hero 3: Monk", 10004: "Hero 4: Prisoner",
			10005: "Hero 5: Oracle", 10006: "Hero 6: Gatekeeper", 10007: "hero 7", 11005: "hero 1005"}, "party role 10002: Traitor mage"},
		{"ru", base.Profile{Language: "russian"}, text.SelectorConverting, map[uint32]string{
			10001: "Герой 1: Главный герой", 10002: "Герой 2: Маг-предатель", 10003: "Герой 3: Монах", 10004: "Герой 4: Узник",
			10005: "Герой 5: Oracle", 10006: "Герой 6: Gatekeeper", 10007: "герой 7", 11005: "герой 1005"}, "party role 10002: Маг-предатель"},
		{"rom2", base.Profile{Game: base.GameROM2, Language: "english"}, 0, map[uint32]string{
			10001: "hero 1", 10002: "hero 2", 10005: "hero 5", 11005: "hero 1005"}, "party role 10002"},
	} {
		t.Run(c.name, func(t *testing.T) {
			x := heroLabelInspector(t, c.profile, c.sel)
			e := openHeroLabelMap(t, x, heroLoseMap(t))
			targets := e.Document().Script.Targets[mapload.ScriptUnit]
			for value, want := range c.choice {
				found := false
				for _, target := range targets {
					if target.Value == value {
						found = true
						if target.Name != x.inspectionUTF8(want) {
							t.Errorf("unit choice %d reads %q, want %q", value, target.Name, x.inspectionUTF8(want))
						}
					}
				}
				if !found {
					t.Errorf("no unit choice %d", value)
				}
			}
			goals := e.Document().Edits.(ui.InspectionScript).Goals()
			if len(goals) == 0 || !strings.HasSuffix(goals[0].Text, ", "+c.goal) {
				t.Errorf("the arms list reads %+v, want the target %q", goals, c.goal)
			}
			if n := wizNode(t, wizScript(t, e.Document().SourceBytes).Conditions, 1); n.Value[0] != 10002 {
				t.Errorf("the stored hero reads %d", n.Value[0])
			}
		})
	}
}

// The roles come from the message popup's speaker builder: heroes 5 and 6
// read the popup's own labels for speakers 25 and 26.
func TestHeroRolesAreTheSpeakerListsLabels(t *testing.T) {
	x := heroLabelInspector(t, base.Profile{Language: "english"}, 0)
	roles := x.heroLabels("%d", "%d %s", func(s string) string { return s }).Roles
	for _, choice := range x.npcChoices(255, true) {
		if n := choice.Value - 20; n >= 1 && n <= 6 {
			if want := strings.SplitN(choice.Label, ": ", 2)[1]; roles[n] != want {
				t.Errorf("hero %d role %q, the speaker list says %q", n, roles[n], want)
			}
		}
	}
	if len(roles) != 6 {
		t.Errorf("roles %v, want heroes 1 to 6", roles)
	}
}
