package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"testing"
)

func TestReleaseNativeBodyWritesCurrentPartyBase(t *testing.T) {
	f := releaseFront(t)
	id := openSkillMission(t, f, 90, skillCaseParty(f, skillCases()[0]))
	// Establish a retained Document before changing the authored base.
	baseline := skillExport(t, f, true, "mission SAVE")
	f, id = coldMission(t, baseline)
	base := f.live.mission.party[0].Hero.Body
	f.live.mission.party[0].Hero.Body += 3
	f.live.mission.party[0].Name = "Current party name"
	f.Difficulty = mapload.DifficultyHard
	regions, _ := worldRegions(f.live.world)
	for i, e := range f.live.world.Entities() {
		if e.ID == id {
			regions["World.entities[].PotionStats[0]"].values[i].SetInt(2)
		}
	}
	f.live.recomputeRaisedSkills()
	// The ordinary Body must follow the captured current party even while the
	// last loaded Document holds the previous base.
	for _, how := range []string{"mission SAVE", "autosave"} {
		raw := skillExport(t, f, true, how)
		file, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		if file.Head.Difficulty != 3 {
			t.Errorf("%s stale mission difficulty: %d", how, file.Head.Difficulty)
		}
		characters, err := file.Party()
		if err != nil {
			t.Fatal(err)
		}
		for _, character := range characters {
			if character.Hero && character.Name != "Current party name" {
				t.Errorf("%s stale party name: %q", how, character.Name)
			}
		}
		got := savHero(t, raw).stats[0]
		if got != uint16(base+5) {
			t.Errorf("%s Body %d, current %d", how, got, base+5)
		}
		cold, coldID := coldMission(t, raw)
		if e, ok := cold.live.world.Entity(coldID); !ok || e.PotionStats != ([4]int32{2}) {
			t.Fatalf("%s cold gain changed: %+v", how, e)
		}
		cold.LiveAdvance(2)
		if got := savHero(t, skillExport(t, cold, true, "mission SAVE")).stats[0]; got != uint16(base+5) {
			t.Errorf("%s next SAVE applied permanent gain twice: %d", how, got)
		}
	}
}
