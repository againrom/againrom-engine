package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// bonusSwordParty is a fighter trained to Sword 100, wearing an armor piece
// whose skill effect adds bonus to Sword (zero builds the control).
func bonusSwordParty(f *FrontEnd, bonus int16) []mapload.PartyMember {
	party := f.ChargenParty(ui.ChargenResult{Name: "bonus", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	party[0].Hero.Skill[1] = 100
	if bonus != 0 {
		item := mapload.SourceConstructedItem(sim.PlainItem(0xec7e), f.Table)
		item.Effects = []sim.ItemEffect{{Kind: 27, Operand: uint32(bonus)}}
		party[0].WornItems[11], party[0].Worn[11] = item, item.Code
	}
	party[0].Carry = &mapload.Carry{Equipped: party[0].Worn, EquippedItems: party[0].WornItems}
	return party
}

func savedHumanLevels(t *testing.T, raw []byte) (level, base, bonus int32) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i := range doc.Objects {
		r := &doc.Objects[i]
		lv, bs, md := modBlock(r, "UA6"), modBlock(r, "U114"), modBlock(r, "UD4")
		if r.Class != "Human" || len(lv) != 24 || len(bs) != 24 || len(md) != 64 {
			continue
		}
		l := int32(int16(binary.LittleEndian.Uint16(lv[2+2*1:])))
		if found && l <= level {
			continue
		}
		found = true
		level, base, bonus = l, int32(int16(binary.LittleEndian.Uint16(bs[2+2*1:]))), int32(int16(binary.LittleEndian.Uint16(md[20+2*1:])))
	}
	if !found {
		t.Fatal("no Human record in the save")
	}
	return
}

// A worn bonus lifts the effective level past 100 in the base game, the save
// keeps the original's fields, and a cold load derives the level again.
func TestReleaseSkillBonusLiftsPast100AndTheSaveStaysOriginal(t *testing.T) {
	for _, tc := range []struct {
		name  string
		bonus int16
		want  int32
	}{{"item bonus 10", 10, 110}, {"no bonus control", 0, 100}} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			app := f.App("bonus")
			app.Layout(1024, 768)
			if err := app.OpenMission(f.MissionOpenerWith(20, bonusSwordParty(f, tc.bonus))); err != nil {
				t.Fatal(err)
			}
			id := f.live.mission.ids[0]
			if got := releaseEntity(t, f.live, id).Skill[1]; got != tc.want {
				t.Fatalf("live Sword = %d, want %d", got, tc.want)
			}
			subject, rows := cardLoadRows(t, f, app, id)
			if subject.Char.Skills[1] != int(tc.want) {
				t.Fatalf("the character panel holds Sword %d, want %d", subject.Char.Skills[1], tc.want)
			}
			seen := false
			for _, r := range rows {
				if strings.Contains(" "+r.FullValue+" ", fmt.Sprintf(" %d ", tc.want)) {
					seen = true
					if r.Value != r.FullValue {
						t.Fatalf("the layout shortened %q to %q", r.FullValue, r.Value)
					}
				}
			}
			if !seen {
				t.Fatalf("no panel row states %d", tc.want)
			}
			cardLoadShot(t, app, "skill-"+strings.ReplaceAll(tc.name, " ", "-"))
			path := saveCorpseMission(t, f, t.TempDir())
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(raw, modMarkName) {
				t.Fatal("a save without a mod set carries a mod mark")
			}
			level, base, bonus := savedHumanLevels(t, raw)
			if level != 100 || base != 100 || bonus != int32(tc.bonus) {
				t.Fatalf("saved level %d base %d bonus %d, want 100 100 %d", level, base, bonus, tc.bonus)
			}
			cold := loadAreaContinuation(t, path)
			cid := cold.live.mission.ids[0]
			if got := releaseEntity(t, cold.live, cid).Skill[1]; got != tc.want {
				t.Fatalf("cold Sword = %d, want %d", got, tc.want)
			}
			// The next action: time passes and the level holds; a second save
			// writes the same ordinary fields.
			for n := 0; n < 64; n++ {
				cold.live.tick()
			}
			if got := releaseEntity(t, cold.live, cid).Skill[1]; got != tc.want {
				t.Fatalf("Sword after the next ticks = %d, want %d", got, tc.want)
			}
			path2 := saveCorpseMission(t, cold, t.TempDir())
			raw2, _ := os.ReadFile(path2)
			if l, b, m := savedHumanLevels(t, raw2); l != 100 || b != 100 || m != int32(tc.bonus) {
				t.Fatalf("second save level %d base %d bonus %d", l, b, m)
			}
		})
	}
}
