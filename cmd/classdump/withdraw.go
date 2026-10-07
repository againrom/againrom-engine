package main

import (
	"fmt"
	"io"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// withdrawalCensus is the whole population the -withdraw verb measures. The
// definition and placement halves stay separate: one answers what classes can
// carry, the other what shipped missions actually instantiate.
type withdrawalCensus struct {
	maps, placements                         int
	parameterised, withdrawDefs, wimpyDefs   int
	withdrawPlaced, wimpyPlaced              int
	withdrawRangedDefs, withdrawRangedPlaced int
	maxHP, maxThresholdHP                    int32
	classes                                  []withdrawalClass
}

type withdrawalClass struct {
	index           int
	name            string
	withdraw, wimpy int32
	attack          int32
	ranged          bool
}

// runWithdrawal measures all definition rows and all maps under the selected
// root. No install path is embedded: root is the same explicit lawful-install
// argument as -campaign. missionrun's -withdrawal arm owns the tick witness,
// because classdump deliberately stops at mapload's reporting seam and never
// imports the simulation package directly.
func runWithdrawal(root string, out io.Writer) error {
	c, err := openCampaign(root)
	if err != nil {
		return err
	}
	got, err := censusWithdrawal(c, root)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "DEFINITIONS parameterised=%d withdraw-positive=%d wimpy-positive=%d\n",
		got.parameterised, got.withdrawDefs, got.wimpyDefs)
	for _, row := range got.classes {
		fmt.Fprintf(out, "  class=%d name=%q withdraw=%d wimpy=%d attack=%d ranged=%t\n",
			row.index, row.name, row.withdraw, row.wimpy, row.attack, row.ranged)
	}
	fmt.Fprintf(out, "PLACEMENTS maps=%d total=%d withdraw-positive=%d ranged=%d wimpy-positive=%d max-hp=%d threshold-max-hp=%d\n",
		got.maps, got.placements, got.withdrawPlaced, got.withdrawRangedPlaced, got.wimpyPlaced,
		got.maxHP, got.maxThresholdHP)
	return nil
}

func censusWithdrawal(c *campaign, root string) (withdrawalCensus, error) {
	var out withdrawalCensus
	for i := 0; i < c.table.Units.Len(); i++ {
		params := c.table.Units.EntryParams(i)
		if len(params) == 0 {
			continue
		}
		out.parameterised++
		def, err := data.NewUnitDef(c.table.Units.EntryName(i), params)
		if err != nil {
			return out, fmt.Errorf("Units entry %d: %w", i, err)
		}
		if def.Withdraw > 0 {
			out.withdrawDefs++
		}
		if def.Wimpy > 0 {
			out.wimpyDefs++
		}
		if def.Withdraw > 0 || def.Wimpy > 0 {
			attack, ranged := resolvedUnitAttack(c.table, i)
			if def.Withdraw > 0 && ranged {
				out.withdrawRangedDefs++
			}
			out.classes = append(out.classes, withdrawalClass{
				index: i, name: c.table.Units.EntryName(i), withdraw: def.Withdraw,
				wimpy: def.Wimpy, attack: attack, ranged: ranged,
			})
		}
	}

	srcs, err := c.sources(root)
	if err != nil {
		return out, err
	}
	for _, src := range srcs {
		b, err := src.read()
		if err != nil {
			return out, fmt.Errorf("%s: %w", src.name, err)
		}
		m, err := alm.Open(b)
		if err != nil {
			return out, fmt.Errorf("%s: %w", src.name, err)
		}
		world, err := mapload.FromALMWith(m, c.table, mapload.DifficultyNormal)
		if err != nil {
			return out, fmt.Errorf("%s: %w", src.name, err)
		}
		out.maps++
		ents := world.Entities()
		out.placements += len(ents)
		for _, e := range ents {
			if e.MaxHP > out.maxHP {
				out.maxHP = e.MaxHP
			}
			if (e.Withdraw > 0 || e.Wimpy > 0) && e.MaxHP > out.maxThresholdHP {
				out.maxThresholdHP = e.MaxHP
			}
			if e.Withdraw > 0 {
				out.withdrawPlaced++
				if int(e.ID) < 0 || int(e.ID) >= len(m.Units) {
					return out, fmt.Errorf("%s: entity %d has no source placement", src.name, e.ID)
				}
				r := mapload.Resolve(m.Units[int(e.ID)], c.table)
				if r.Arm == mapload.ArmUnits && r.Found() {
					_, ranged := resolvedUnitAttack(c.table, r.Index)
					if ranged {
						out.withdrawRangedPlaced++
					}
				}
			}
			if e.Wimpy > 0 {
				out.wimpyPlaced++
			}
		}
	}
	return out, nil
}

// resolvedUnitWeapon is mapload's EquipItem resolution read at the reporting
// seam: the first trailing equipment literal that ResolveWeapon accepts. The
// census classifies its resolved attack type as SkillShoot; Entity.Reach is a
// folded consequence and is not evidence of the equipment class.
func resolvedUnitWeapon(table *mapload.Table, index int) (data.Weapon, bool) {
	if table == nil || table.Units == nil || table.Shapes == nil || table.Materials == nil || table.Weapons == nil {
		return data.Weapon{}, false
	}
	for _, name := range table.Units.EntryStrings(index) {
		if name == "" {
			continue
		}
		weapon, err := data.ResolveWeapon(name, table.Shapes, table.Materials, table.Weapons)
		if err == nil {
			return weapon, true
		}
	}
	return data.Weapon{}, false
}

func resolvedUnitAttack(table *mapload.Table, index int) (int32, bool) {
	weapon, ok := resolvedUnitWeapon(table, index)
	if !ok {
		return 0, false
	}
	return weapon.AttackType, weapon.AttackType == data.SkillShoot
}
