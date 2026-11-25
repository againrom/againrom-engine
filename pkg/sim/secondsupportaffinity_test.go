package sim

import (
	"slices"
	"testing"
)

// R2-ENGINE-1793: the relation byte decides; the first game sorts by owner.
func TestSecondSupportCandidatesFollowTheRelationByte(t *testing.T) {
	t.Parallel()
	build := func(second bool, diagonal, toAlly byte) *World {
		caster := acCaster(1, 2, 2, 50, 1<<6, 6)
		caster.Owner, caster.ScanRange = SelfSlot, 6
		own := spEnt(2, 4, 2)
		own.Owner, own.HP = SelfSlot, 20
		ally := spEnt(3, 2, 4)
		ally.Owner, ally.HP = 2, 20
		var rel Relations
		rel.Set(SelfSlot, SelfSlot, diagonal)
		rel.Set(2, 2, diagonal)
		rel.Set(SelfSlot, 2, toAlly)
		rel.Set(2, SelfSlot, toAlly)
		w := hlWorld(t, 29, rel, []SpellRule{hlHeal()}, caster, own, ally)
		if second {
			w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
		}
		return w
	}
	for _, tc := range []struct {
		name             string
		second           bool
		diagonal, toAlly byte
		setting          AutoCastSetting
		own, ally        bool
	}{
		{name: "second, Allies alone, diagonal 0x12", second: true, diagonal: 0x12, toAlly: relationLocked,
			setting: AutoCastAllies, own: true, ally: true},
		{name: "second, Neutral alone, diagonal 0x12", second: true, diagonal: 0x12, setting: AutoCastNeutral, ally: true},
		{name: "second, Neutral alone, diagonal 0", second: true, setting: AutoCastNeutral, own: true, ally: true},
		{name: "second, Allies alone, hostile and allied", second: true, diagonal: 0x12, toAlly: relationHostile | relationLocked,
			setting: AutoCastAllies, own: true},
		{name: "first, Allies alone, diagonal 0x12", diagonal: 0x12, toAlly: relationLocked, setting: AutoCastAllies, ally: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := build(tc.second, tc.diagonal, tc.toAlly).autoCastCandidates(0, tc.setting, false)
			if !slices.Contains(got, 0) {
				t.Fatalf("caster left out: %v", got)
			}
			if slices.Contains(got, 1) != tc.own || slices.Contains(got, 2) != tc.ally {
				t.Fatalf("candidates %v, want own %v, ally %v", got, tc.own, tc.ally)
			}
		})
	}
}
