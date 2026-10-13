package sim

import (
	"bytes"
	"fmt"
	"testing"
)

func TestAuthoredControlAttackNoticeAndColdRestore(t *testing.T) {
	for _, owner := range []uint32{SelfSlot, 7} {
		for _, control := range []uint32{0, 1, 0x100} {
			t.Run(fmt.Sprintf("owner-%d-control-%x", owner, control), func(t *testing.T) {
				attacker, victim := cbEnt(1, 5, 6), cbEnt(2, 9, 9)
				attacker.Owner, victim.Owner = 2, owner
				w := cbWorld(t, 1, attacker, victim)
				if err := w.RestoreCurrentPlayers([]SavedGroupPlayer{{ID: 91, Slot: 2}, {ID: 103, Slot: owner}},
					[]PlayerParticipant{{PlayerID: 91, Value: 0}, {PlayerID: 103, Value: control}}, true); err != nil {
					t.Fatal(err)
				}
				w.rememberAttacker(0, 1)
				wantNotice := attackNotice{Cell: 0x0605}
				if got := w.attackNoticeAt(1); got != wantNotice {
					t.Fatalf("damage notice = %+v, want %+v", got, wantNotice)
				}
				var cold World
				if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil || cold.Hash() != w.Hash() {
					t.Fatal("notice or control changed on cold restore", err)
				}
				wantStamp := make([]byte, 16*16)
				if control != 0 {
					wantStamp[6*16+5] = 1
				}
				for _, world := range []*World{w, &cold} {
					before := world.Hash()
					stamp := make([]byte, 16*16)
					world.stampAttackNotices(stamp, []int{1}, false)
					if !bytes.Equal(stamp, wantStamp) || world.attackNoticeAt(1) != wantNotice || world.Hash() != before {
						t.Fatal("perception query changed the notice or used the roster ordinal as control")
					}
				}
				for scan := 1; scan <= 22; scan++ {
					if scan == 22 {
						clear(wantStamp)
					}
					if control != 0 && scan <= 21 {
						wantNotice.Scans = uint8(scan)
						if scan == 21 {
							wantNotice.Cell = 0
						}
					}
					for _, world := range []*World{w, &cold} {
						before := world.Hash()
						stamp := make([]byte, 16*16)
						world.stampAttackNotices(stamp, []int{1}, true)
						if !bytes.Equal(stamp, wantStamp) || world.attackNoticeAt(1) != wantNotice {
							t.Fatalf("scan %d: notice %+v, want %+v; stamp equal %t", scan, world.attackNoticeAt(1), wantNotice, bytes.Equal(stamp, wantStamp))
						}
						if changed := world.Hash() != before; changed != (control != 0 && scan <= 21) {
							t.Fatalf("scan %d: hashed notice change = %t", scan, changed)
						}
					}
					if cold.Hash() != w.Hash() {
						t.Fatalf("notice continuation diverged after scan %d", scan)
					}
				}
			})
		}
	}
}

func TestAuthoredControlSpellDamageExperienceAndColdRestore(t *testing.T) {
	for _, owner := range []uint32{SelfSlot, 7} {
		for _, control := range []uint32{0, 1, 0x100} {
			t.Run(fmt.Sprintf("owner-%d-control-%x", owner, control), func(t *testing.T) {
				caster, victim := xpAttacker(2), xpTarget(owner)
				caster.MaxMana = 100
				caster.Skill[3], caster.SkillXP[3] = 30, skillXPFor(30)
				victim.XPValue = 1000
				w := cbWorld(t, 1, caster, victim)
				if err := w.RestoreCurrentPlayers([]SavedGroupPlayer{{ID: 91, Slot: 2}, {ID: 103, Slot: owner}},
					[]PlayerParticipant{{PlayerID: 91, Value: 0}, {PlayerID: 103, Value: control}}, true); err != nil {
					t.Fatal(err)
				}
				var cold World
				if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil || cold.Hash() != w.Hash() {
					t.Fatal("experience or control changed on cold restore", err)
				}
				wantXP := caster.SkillXP
				for award := 1; award <= 2; award++ {
					if control != 0 {
						wantXP[3] += 114 // (1000*10/(2*100)+1)*(4*60+30)/120, truncated.
					}
					for _, world := range []*World{w, &cold} {
						before := world.Hash()
						world.awardSpellDamage(0, 1, SpellRule{School: 3}, 10)
						if got := world.entities[0].SkillXP; got != wantXP {
							t.Fatalf("award %d: skill XP = %v, want %v", award, got, wantXP)
						}
						if changed := world.Hash() != before; changed != (control != 0) {
							t.Fatalf("award %d: hashed experience change = %t", award, changed)
						}
					}
					if cold.Hash() != w.Hash() {
						t.Fatalf("experience continuation diverged after award %d", award)
					}
					if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil || cold.Hash() != w.Hash() {
						t.Fatal("awarded experience changed on cold restore", err)
					}
				}
			})
		}
	}
}
