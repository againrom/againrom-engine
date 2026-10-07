package game

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestReleaseReturnedTownSchoolTrainingSavesAndReloads(t *testing.T) {
	f := currentTown(t, nil, nil)
	s := currentTownShop(f, "hero")
	s.room = roomSchool
	member := f.Carried[s.shopMember]
	if member.Carry == nil {
		t.Fatal("returned hero has no current carry")
	}
	slot := 0
	for i := 1; i <= 5; i++ {
		if member.Hero.Skill[i] == 0 {
			slot = i
			break
		}
	}
	gold := f.Town.Gold()
	if slot == 0 || gold < 420 {
		t.Fatalf("returned hero cannot afford two untrained skills: slot=%d gold=%d", slot, gold)
	}
	s.schoolCell = slot - 1
	message := s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false).Msg
	if !strings.HasPrefix(message, "trained ") {
		t.Fatalf("school after mission return: %s", message)
	}
	trained := f.Carried[s.shopMember]
	if trained.Carry == nil || trained.Hero.Skill[slot] != 1 || trained.Carry.SkillXP[slot] != 101 || f.Town.Gold() != gold-200 {
		t.Fatalf("training did not update current native skill and purse: base=%d XP=%d gold=%d", trained.Hero.Skill[slot], trained.Carry.SkillXP[slot], f.Town.Gold())
	}
	worn, pack := memberItemCodes(trained)
	if beforeWorn, beforePack := memberItemCodes(member); worn != beforeWorn || !reflect.DeepEqual(pack, beforePack) {
		t.Fatal("training changed the hero's equipment or pack")
	}
	raw := currentTownSave(t, f)
	dir := t.TempDir()
	input, output := filepath.Join(dir, "trained.sav"), filepath.Join(dir, "continued.sav")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestTrainedCityProcess$", "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_1099_PROCESS=train", "AGAINROM_1099_INPUT="+input,
		"AGAINROM_1099_OUTPUT="+output, fmt.Sprintf("AGAINROM_1099_SLOT=%d", slot),
		"AGAINROM_1099_PRICE=220", fmt.Sprintf("AGAINROM_1099_GOLD=%d", gold-200))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fresh-process LOAD and next training: %v\n%s", err, out)
	}
	continued, err := os.ReadFile(output)
	if err != nil || bytes.Equal(raw, continued) {
		t.Fatalf("next training produced no changed SAV: %v", err)
	}
	g := currentTownReload(t, continued)
	for _, reloaded := range g.Carried {
		if reloaded.ID != member.ID {
			continue
		}
		if reloaded.Carry == nil || reloaded.Hero.Skill[slot] != 2 || reloaded.Carry.SkillXP[slot] != 211 || g.Town.Gold() != gold-420 {
			t.Fatalf("second SAV lost training: base=%d XP=%d gold=%d", reloaded.Hero.Skill[slot], reloaded.Carry.SkillXP[slot], g.Town.Gold())
		}
		gotWorn, gotPack := memberItemCodes(reloaded)
		if gotWorn != worn || !reflect.DeepEqual(gotPack, pack) {
			t.Fatal("cold LOAD lost the trained hero's equipment or pack")
		}
		g.Town.gold = 0
		before := mapload.CloneParty(g.Carried)
		denied := currentTownShop(g, member.ID)
		denied.room, denied.schoolCell = roomSchool, slot-1
		message = denied.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false).Msg
		if strings.HasPrefix(message, "trained ") || !reflect.DeepEqual(before, g.Carried) || g.Town.Gold() != 0 {
			t.Fatal("unaffordable training changed the hero or purse", message)
		}
		return
	}
	t.Fatal("cold LOAD lost the trained hero")
}
