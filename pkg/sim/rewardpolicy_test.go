package sim

import "testing"

func TestRewardOutputsAreRecorded(t *testing.T) {
	type output struct {
		gold uint32
		loot bool
		paid bool
		xp   int32
	}
	cases := []struct {
		name     string
		dialect  ScriptDialect
		marked   bool
		humanoid bool
		want     output
	}{
		{"first/plain/creature", ScriptROM1, false, false, output{8, true, true, 100}},
		{"first/plain/human", ScriptROM1, false, true, output{8, true, true, 100}},
		{"first/marked/creature", ScriptROM1, true, false, output{8, true, true, 100}},
		{"first/marked/human", ScriptROM1, true, true, output{8, true, true, 100}},
		{"second/plain/creature", ScriptROM2, false, false, output{8, true, true, 100}},
		{"second/plain/human", ScriptROM2, false, true, output{8, true, true, 100}},
		{"second/marked/creature", ScriptROM2, true, false, output{0, false, true, 100}},
		{"second/marked/human", ScriptROM2, true, true, output{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attacker, source := skAwarder(1, 2), skSource(2, 3)
			attacker.Humanoid = tc.humanoid
			source.TypeID, source.GoldChance = 0x41, 100
			source.TreasureMin, source.TreasureMax = 8, 8
			source.SuppressRewards = tc.marked
			w := cbWorld(t, 7, attacker, source)
			w.game = newGameProfile(tc.dialect)
			got := output{gold: w.deathGold(1), loot: w.hasTerminalLoot(1)}
			w.awardSkill(0, 0, 100, 1)
			got.xp = w.entities[0].SkillXP[3]
			got.paid = got.xp != 0
			t.Logf("seed=7 gold=%d loot=%t paid=%t xp=%d", got.gold, got.loot, got.paid, got.xp)
			if got != tc.want {
				t.Fatalf("reward output = %+v, want %+v", got, tc.want)
			}
		})
	}
}
