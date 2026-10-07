package ui

import (
	"image"
	"reflect"
	"testing"
)

func TestStructureAttackProductionPointerIsExplicitAndKeepsSelection(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	var commands []order
	a.flow.attack = func(entity, victim, spell uint32, x, y int, cell bool) {
		commands = append(commands, order{entity: entity, victim: victim, spell: spell, cell: cell})
	}
	ref := InspectionSubject{Kind: InspectionStructure, ID: 7}
	inv := v.invSubject
	inspectionHover(t, a, v, ref)
	if len(commands) != 0 {
		t.Fatal("hover ordered attack")
	}
	x, y, err := v.InspectionPoint(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("attack"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("press", x, y); err != nil {
		t.Fatal(err)
	}
	if len(commands) != 0 {
		t.Fatal("press dispatched before release")
	}
	if err := a.HeadlessPointer("release", x, y); err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].entity != 1 || commands[0].victim != 7 || !commands[0].cell || commands[0].spell != 0 {
		t.Fatalf("commands=%+v", commands)
	}
	if !reflect.DeepEqual(v.sel, selection{1}) || !reflect.DeepEqual(v.invSubject, inv) || v.AttackArmed() {
		t.Fatal("attack changed selection/inventory or retained one-shot mode")
	}
}

func TestStructureAttackPointerRefusesUnarmedHiddenAndRuinedTargets(t *testing.T) {
	for _, mode := range []string{"unarmed", "unseen", "explored", "ruin"} {
		t.Run(mode, func(t *testing.T) {
			a, v := inspectionFixture(t, image.Pt(1024, 768))
			attacks := 0
			a.flow.attack = func(_, _, _ uint32, _, _ int, _ bool) { attacks++ }
			x, y, err := v.InspectionPoint(InspectionSubject{InspectionStructure, 7})
			if err != nil {
				t.Fatal(err)
			}
			if mode != "unarmed" {
				if err := a.HeadlessKey("attack"); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "unseen" || mode == "explored" {
				fog := make([]byte, 1024)
				for i := range fog {
					fog[i] = FogVisible
				}
				fog[6*32+6] = FogUnseen
				if mode == "explored" {
					fog[6*32+6] = FogExplored
				}
				v.SetFog(fog, 32, 32)
			}
			if mode == "ruin" {
				v.SetStructures([]MapStructure{{ID: 7, Health: 0, MaxHealth: 789, Cell: image.Pt(6, 6)}})
			}
			if err := a.HeadlessPointer("press", x, y); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessPointer("release", x, y); err != nil {
				t.Fatal(err)
			}
			if attacks != 0 {
				t.Fatal("refused target issued an attack")
			}
		})
	}
}
