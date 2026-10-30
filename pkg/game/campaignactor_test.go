package game

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

func TestCampaignHirePolicyDefaultsAndRefusesCorruptMembership(t *testing.T) {
	var old currentPartyPolicy
	if err := json.Unmarshal([]byte(`{"PlayerCharacter":true,"CompanionNPC":22}`), &old); err != nil {
		t.Fatal(err)
	}
	p := old.member()
	if p.Hired() || p.CampaignUnit || p.CampaignHire != 0 || p.CompanionNPC != 22 {
		t.Fatalf("old companion policy changed: %+v", p)
	}
	for _, policy := range []currentPartyPolicy{
		{CampaignHire: 21, Temporary: true},
		{CampaignUnit: true},
		{CampaignHire: 1},
		{CampaignHire: 1, Temporary: true, MercenaryType: 1},
		{CampaignHire: 1, Temporary: true, PlayerCharacter: true},
	} {
		p := currentPartyMember{Policy: &policy}
		if _, err := p.restoreFromState(nil, nil); err == nil || !strings.Contains(err.Error(), "invalid campaign hire membership") {
			t.Fatalf("corrupt policy %+v: %v", policy, err)
		}
	}
}

// Each page of a hire section draws its own speaker (DIV-4199).
func TestTownHireDialogueDrawsEachSpeakersOwnFace(t *testing.T) {
	sections := []struct {
		hire, stage int
		accept      bool
		speakers    []int
	}{
		{1, 90, false, []int{1, 22, 1}},
		{2, 10, true, []int{2, 22}},
		{2, 30, true, []int{22, 2}},
		{2, 40, true, []int{22, 2}},
		{2, 100, true, []int{2, 22}},
		{2, 30, false, []int{22, 2}},
		{2, 70, false, []int{2, 21, 22}},
		{3, 90, true, []int{22, 3}},
		{3, 110, true, []int{3, 22, 3}},
		{5, 10, true, []int{5, 22, 5}},
		{5, 100, true, []int{5, 22, 5}},
		{6, 70, true, []int{6, 23, 6}},
	}
	key := func(hire, stage int, accept bool) string {
		if accept {
			return fmt.Sprintf("npc%daccept%d", hire, stage)
		}
		return fmt.Sprintf("npc%dreject%d", hire, stage)
	}
	var text strings.Builder
	for _, sec := range sections {
		fmt.Fprintf(&text, "#%s\r\n", key(sec.hire, sec.stage, sec.accept))
		for i, speaker := range sec.speakers {
			fmt.Fprintf(&text, "<NPC=%d,PART=%d>\r\nLine %d.\r\n", speaker, i+1, i+1)
		}
	}
	text.WriteString("#end\r\n")
	f, app, s := secondCampaignFixtureFiles(t, false)
	dir := t.TempDir()
	archives := map[string][]synth.File{
		MainArchive: {{Path: "text/town.txt", Data: []byte(text.String())}},
		GraphicsArchive: {
			{Path: "infowindow/Hire.bmp", Data: synthBMP(4, 4, color.RGBA{R: 0xff, A: 0xff})},
			{Path: "infowindow/Iglez.bmp", Data: synthBMP(4, 4, color.RGBA{G: 0xff, A: 0xff})},
			{Path: "infowindow/Dina.bmp", Data: synthBMP(4, 4, color.RGBA{B: 0xff, A: 0xff})},
		},
	}
	var paths []string
	for _, name := range []string{MainArchive, GraphicsArchive} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, synth.Archive(archives[name]), 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	fs, err := vfs.Open(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = fs
	f.Units = &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
		100: {Portrait: "Hire"}, 101: {Portrait: "Iglez"}, 102: {Portrait: "Dina"},
	}}
	own := data.NPCTokens(data.NPCTokenNotHuman)
	f.NPCFaces = map[int32]data.NPCFace{
		22: {Kind: data.NPCPortrait, Class: 101, Tokens: own},
		23: {Kind: data.NPCPortrait, Class: 102, Tokens: own},
	}
	portrait := func(class int32) *image.RGBA {
		pic := classPortrait(f.Archives.Containers, f.Units.Classes[class], 1, map[string]*image.RGBA{})
		if pic == nil {
			t.Fatalf("fixture class %d has no portrait", class)
		}
		return pic
	}
	want := map[int]*image.RGBA{22: portrait(101), 23: portrait(102)}
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	hero := f.Carried[0]
	for _, sec := range sections {
		name := key(sec.hire, sec.stage, sec.accept)
		c := f.Town.second
		for i := 0; i < secondHireSlots; i++ {
			c.bank[secondHireSelection+i], c.bank[secondHireAvailability+i], c.bank[secondHireTown+i] = 0, 0, 0
		}
		availability := int32(2)
		if sec.accept {
			availability = 1
		}
		c.bank[768] = int32(sec.stage)
		c.bank[secondHireAvailability+sec.hire-1], c.bank[secondHireTown+sec.hire-1] = availability, int32(c.current.id)
		c.innBits = nil
		hire := mapload.PartyMember{ID: fmt.Sprintf("hire:%d", sec.hire), Name: "hire", Class: 100,
			Temporary: true, CampaignHire: uint8(sec.hire), CampaignUnit: true}
		f.Carried = []mapload.PartyMember{hero, hire}
		want[sec.hire] = portrait(100)
		for _, target := range []string{"TAVERN", fmt.Sprintf("HIRE %d", sec.hire)} {
			if err := app.HeadlessActivate(target); err != nil {
				t.Fatalf("%s: %s: %v", name, target, err)
			}
		}
		var drawn []int
		for page := 0; page < 16; page++ {
			v := s.view()
			if !v.Open {
				break
			}
			speaker, named := tagSpeaker(s.dialogue.tag)
			if named {
				drawn = append(drawn, speaker)
				if w := want[speaker]; w != nil && (v.Page.Face == nil || !sameImage(v.Page.Face, w)) {
					t.Errorf("%s page %d: speaker %d does not draw its own face", name, page+1, speaker)
				}
				if speaker != sec.hire && v.Page.Face != nil && sameImage(v.Page.Face, want[sec.hire]) {
					t.Errorf("%s page %d: speaker %d draws the hire's face", name, page+1, speaker)
				}
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				break
			}
		}
		if fmt.Sprint(drawn) != fmt.Sprint(sec.speakers) {
			t.Fatalf("%s: pages by speaker %v, want %v", name, drawn, sec.speakers)
		}
		delete(want, sec.hire)
		if err := app.HeadlessActivate("EXIT"); err != nil {
			t.Fatalf("%s: EXIT: %v", name, err)
		}
	}
}
