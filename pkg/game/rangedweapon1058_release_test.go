package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestReleaseEveryDragonUsesFlameDamageAndGeneralAccuracy(t *testing.T) {
	f := releaseFront(t)
	flame, err := data.ResolveWeapon("Flame Thrower", f.Table.Shapes, f.Table.Materials, f.Table.Weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon(Flame Thrower): %v", err)
	}
	if flame.AttackType != 11 || flame.DamageBase != 1 || flame.DamageSpread != 7 || flame.Range != 8 {
		t.Fatalf("Flame Thrower = %+v, want type 11, damage 1+U[0,7], range 8", flame)
	}
	wantSecondary := sim.SecondaryDamage{Base: 1, Spread: 7, Selector: 0}

	missions := releaseMissionPopulation(t, f)
	if len(missions) != 28 {
		t.Fatalf("campaign map population = %d, want 28: %v", len(missions), missions)
	}
	definitions := make(map[int]bool)
	placements := 0
	for _, mission := range missions {
		m := releaseMissionMap(t, f, mission)
		ms, err := StartMissionFrom(m, fmt.Sprintf("scenario%d.alm", mission), mission,
			f.Table, mapload.DifficultyNormal, nil)
		if err != nil {
			t.Fatalf("StartMissionFrom(%d): %v", mission, err)
		}
		entities := ms.World.Entities()
		if len(entities) < len(m.Units) {
			t.Fatalf("mission %d: %d entities for %d placements", mission, len(entities), len(m.Units))
		}
		for i, unit := range m.Units {
			resolution := mapload.Resolve(unit, f.Table)
			if resolution.Arm != mapload.ArmUnits {
				continue
			}
			name := f.Table.Units.EntryName(resolution.Index)
			if !strings.Contains(name, "Dragon") {
				continue
			}
			carriesFlame := false
			for _, cell := range f.Table.Units.EntryStrings(resolution.Index) {
				carriesFlame = carriesFlame || strings.Contains(cell, "Flame Thrower")
			}
			if !carriesFlame {
				t.Fatalf("mission %d %s equipment %q does not name Flame Thrower",
					mission, name, f.Table.Units.EntryStrings(resolution.Index))
			}
			def, err := data.NewUnitDef(name, f.Table.Units.EntryParams(resolution.Index))
			if err != nil {
				t.Fatalf("mission %d %s definition: %v", mission, name, err)
			}
			entity := entities[i]
			placements++
			definitions[resolution.Index] = true

			if entity.DamageBase != def.DamageBase || entity.DamageSpread != def.DamageSpread ||
				entity.Defence != def.Defence {
				t.Errorf("mission %d %s: physical/Defence = %d/%d/%d, want row %d/%d/%d",
					mission, name, entity.DamageBase, entity.DamageSpread, entity.Defence,
					def.DamageBase, def.DamageSpread, def.Defence)
			}
			if entity.ToHit != def.ToHit+def.ToHit {
				t.Errorf("mission %d %s: ToHit = %d, want authored %d + copied General %d",
					mission, name, entity.ToHit, def.ToHit, def.ToHit)
			}
			if entity.SecondaryDamage != wantSecondary {
				t.Errorf("mission %d %s: SecondaryDamage = %+v, want Flame Thrower Fire %+v",
					mission, name, entity.SecondaryDamage, wantSecondary)
			}
			wantCharge, wantRelax := flame.ChargeTime, flame.RelaxTime
			if wantCharge < 0 {
				wantCharge = def.AttackChargeTime
			}
			if wantRelax < 0 {
				wantRelax = def.AttackRelaxTime
			}
			if entity.XPSlot != 0 || int32(entity.Reach) != flame.Range ||
				entity.AttackCharge != wantCharge || entity.AttackRelax != wantRelax {
				t.Errorf("mission %d %s: slot/reach/cadence = %d/%d/%d/%d, want 0/%d/%d/%d",
					mission, name, entity.XPSlot, entity.Reach, entity.AttackCharge, entity.AttackRelax,
					flame.Range, wantCharge, wantRelax)
			}
		}
	}

	if releaseDragonDefinitions(f.Table.Units) != 4 || len(definitions) != 4 || placements != 21 {
		t.Fatalf("Dragon population = %d table definitions/%d placed definitions/%d placements; want 4/4/21",
			releaseDragonDefinitions(f.Table.Units), len(definitions), placements)
	}
	t.Logf("%s: 28 maps, 4 Dragon definitions, 21 placements; Flame Thrower type=%d damage=%d+U[0,%d] Fire; General folded into ToHit",
		filepath.Base(strings.TrimRight(os.Getenv("AGAINROM_ASSETS"), `\/`)),
		flame.AttackType, flame.DamageBase, flame.DamageSpread)
}
