package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"testing"
)

// Literal independent adapters keep frozen predecessor fixtures unchanged.
func widenedSavedWorldEffectsPin(old []byte) []byte {
	out := append(bytes.Clone(old), 0, 0, 0, 0)
	out[0] = 92
	return widenedAttackNoticePin(out)
}

func strippedSavedWorldEffectsPin(form []byte) []byte {
	out := strippedAttackNoticePin(form)
	if len(out) > 0 && out[0] >= 91 {
		n := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-n]
		out[0] = 90
	}
	return out
}

func retainedArea1162(t *testing.T, spell uint16, mode, stage, direction byte, timer uint16) *World {
	t.Helper()
	w := mustWorld(t, 1162, Bounds{96, 96}, nil)
	w.SetSavedSpellEffects([]SavedSpellEffect{{Class: "AreaEffect", AE48: [4]byte{199, 2, direction, stage}, AE4C: timer, AE44: &SavedEffect{Class: "Effect_DirectDamage", E0C: uint8(spell), E3C: 6, DirectDamage: [24]byte{99}}}})
	d := SavedAreaDriver{ID: 1, Root: 0, Identity: 17, Key: 0x2828, Mode: mode, Spell: spell, Layer: 255}
	if err := w.ImportOriginalWorldEffectDrivers(&SavedWorldEffects{Areas: []SavedAreaDriver{d}}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestRetainedAreas1162KnownStagePrograms(t *testing.T) {
	for _, tc := range []struct {
		spell uint16
		dir   byte
		limit int
	}{{4, 0, 2}, {9, 0, 6}, {9, 32, 6}, {21, 0, 32}} {
		t.Run(fmt.Sprintf("spell%d-dir%d", tc.spell, tc.dir), func(t *testing.T) {
			w := retainedArea1162(t, tc.spell, 2, 0, tc.dir, 0)
			seed := w.rng.state
			var fresh World
			for tick := 0; tick <= 3*(tc.limit-1); tick++ {
				report := StepReported(w, nil)
				stage := tick / 3
				if tick%3 != 0 && len(report.AreaPaints) != 0 {
					t.Fatal("stage ran on countdown tick", tick)
				}
				if tick%3 == 0 {
					count := 1
					if tc.spell == 4 {
						count = []int{8, 12}[stage]
					}
					if tc.spell == 9 {
						if tc.dir == 0 {
							count = 2*stage + 1
							if stage == 5 {
								count = 0
							}
						} else {
							count = stage + 1
						}
					}
					got := 0
					if len(report.AreaPaints) > 0 {
						if len(report.AreaPaints) != 1 {
							t.Fatal("duplicate stage")
						}
						got = len(report.AreaPaints[0].Cells)
					}
					if got != count {
						t.Fatal("stage cell count", stage, got, count)
					}
					if tc.spell == 4 && stage == 0 {
						want := []CellPoint{{39, 41}, {39, 40}, {39, 39}, {40, 41}, {40, 39}, {41, 41}, {41, 40}, {41, 39}}
						if !slices.Equal(report.AreaPaints[0].Cells, want) {
							t.Fatal("fixed stage geometry/order")
						}
					}
					if tc.spell == 9 && count > 0 {
						for i, c := range report.AreaPaints[0].Cells {
							want := CellPoint{40 + int32(i-stage), 40 - int32(stage)}
							if tc.dir == 32 {
								want = CellPoint{40 + int32(stage-i), 40 - int32(i)}
							}
							if c != want {
								t.Fatal("orientation transform/order", stage, i, c, want)
							}
						}
					}
					if tc.spell == 21 {
						c := report.AreaPaints[0].Cells[0]
						if c.X < 38 || c.X > 43 || c.Y < 38 || c.Y > 43 || w.rng.state != seed+uint64(2*(stage+1))*gamma {
							t.Fatal("Meteor range/two RNG draws")
						}
					}
				}
				if tick == 0 {
					if err := fresh.UnmarshalBinary(mustMarshal(t, w)); err != nil {
						t.Fatal(err)
					}
				} else {
					Step(&fresh, nil)
					if fresh.Hash() != w.Hash() {
						t.Fatal("changed native continuation", tick)
					}
				}
			}
			if len(w.SavedSpellEffects()) != 0 || w.SavedWorldEffectDrivers() != nil {
				t.Fatal("terminal stage did not reap on same tick")
			}
		})
	}
	// Blast ignores its counter, executes once, and does not register a layer.
	w := retainedArea1162(t, 7, 1, 0, 0, 65535)
	Step(w, nil)
	if len(w.SavedSpellEffects()) != 0 {
		t.Fatal("blast counter incorrectly delays one-shot completion")
	}
}

func TestRetainedProjectile1162SignedTravelClocksAndCompletion(t *testing.T) {
	for _, picture := range []int32{10, 15, 18, 34, 36, 51, 60, 65} {
		t.Run(fmt.Sprint(picture), func(t *testing.T) {
			w := mustWorld(t, 1, Bounds{32, 32}, nil)
			steps := int32(3)
			if picture == 34 || picture == 36 {
				steps = 13
			}
			p := SavedProjectile{ID: 7, Picture: picture, X: 10, Y: 2, ActionX: -4, ActionY: -5, ActionSegments: steps, ActionPhase: 0, Action: 1, Dir: 9, ActionDir: 11}
			if err := w.ImportOriginalProjectiles(SavedProjectiles{FreeIndex: 8, IDs: []uint16{7}, Items: []SavedProjectile{p}}); err != nil {
				t.Fatal(err)
			}
			if err := w.ImportOriginalWorldEffectDrivers(&SavedWorldEffects{Projectiles: []SavedProjectileDriver{{ID: 7, Phases: 4}}}); err != nil {
				t.Fatal(err)
			}
			var cold World
			for tick := int32(1); tick <= steps; tick++ {
				Step(w, nil)
				got := w.SavedProjectiles().Items[0]
				phase := (tick / 2) % 4
				switch picture {
				case 34, 36:
					phase = []int32{4, 3, 2, 1, 0, 1, 2, 1, 0, 1, 2, 3, 4}[tick-1]
				case 51:
					phase = tick
				case 60:
					phase = tick - 1
				}
				if got.ActionPhase != tick || got.Phase != phase || got.ActionSegments != steps-tick || got.LastAction != 1 || got.Dir != 11 || got.ActionDir != 11 {
					// ANIM-139: an action-1 call with no target keeps actiondir and copies it to dir.
					t.Fatal("clock/lifetime/retained directions", tick, got)
				}
				if picture == 34 || picture == 36 {
					if got.X != 10 || got.Y != 2 {
						t.Fatal("stationary bolt moved")
					}
				} else if picture == 18 {
					if got.X != -4 || got.Y != -5 {
						t.Fatal("attached picture did not snap")
					}
				} else if picture == 60 {
					// ANIM-149: the picture-60 arm writes no position.
					if got.X != 10 || got.Y != 2 {
						t.Fatal("picture 60 moved", got)
					}
				} else if tick == 1 && (got.X != 6 || got.Y != 0) {
					t.Fatal("signed division does not truncate toward zero", got)
				}
				if tick == 1 {
					if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
						t.Fatal(err)
					}
				} else {
					Step(&cold, nil)
					if cold.Hash() != w.Hash() {
						t.Fatal("native next-step mismatch")
					}
				}
			}
			Step(w, nil)
			if len(w.SavedProjectiles().Items) != 0 || len(w.SavedProjectiles().IDs) != 0 || w.SavedProjectiles().FreeIndex != 8 {
				t.Fatal("completion changed allocator or retained projectile")
			}
		})
	}
}

func TestRetainedContinuation1162LiteralWireAndAtomicControls(t *testing.T) {
	w := retainedArea1162(t, 4, 2, 0, 0, 1)
	b := mustMarshal(t, w)
	// counts4 + one22-byte area + zero projectile count4 =30, plus span4.
	want := []byte{1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 17, 0, 0, 0, 40, 40, 255, 2, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 30, 0, 0, 0}
	// entityIDFloor (form94) closes the form outside this section entirely.
	end := len(b) - entityIDFloorLen - spellDeliverySpanLen
	start := end - 4 - len(want)
	if !bytes.Equal(b[start:end-4], want) {
		t.Fatalf("literal continuation footer %x", b[start:end-4])
	}
	for _, at := range []int{start, start + 4, start + 8, start + 19, start + 20, start + 22, end - 8} {
		bad := bytes.Clone(b)
		bad[at] = 255
		if at == start+4 {
			bad[at] = 0
		}
		before := w.Hash()
		if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != before {
			t.Fatal("corrupt footer admitted/partly published", at-start, err)
		}
	}
}

// The source operand is a template, not power. The untimed arm consumes all32
// bits and stores no attachment. Pulse/removal are restricted to the retained
// radius and identity; flying occupants belong to a different source slot.
func TestRetainedCloud1162PermanentOperandLayerAndRadius(t *testing.T) {
	ground, air, outside, other := spEnt(1, 5, 5), spEnt(2, 5, 5), spEnt(3, 7, 5), spEnt(4, 5, 6)
	air.Domain = DomainAir
	for _, e := range []*Entity{&ground, &air, &outside, &other} {
		e.HP, e.MaxHP = 5, 100000
	}
	w := hlWorld(t, 1162, Relations{}, []SpellRule{{ID: 12, TargetsUnit: true}}, ground, air, outside, other)
	w.SetSavedSpellEffects([]SavedSpellEffect{{Class: "AreaEffect", AE48: [4]byte{199, 1, 0, 0}, AE4C: 17, AE44: &SavedEffect{Class: "Effect", E0C: 12, E3C: 6, E40: 65537}}})
	w.SetSavedCellRecords([]SavedCellRecord{
		{Cell: 0x0505, LayerCount: 2, SpellEffects: [6]uint32{0, 99, 0, 0, 17, 0}},
		{Cell: 0x0507, LayerCount: 1, SpellEffects: [6]uint32{0, 0, 0, 0, 17, 0}},
		{Cell: 0x0605, LayerCount: 1, SpellEffects: [6]uint32{0, 0, 0, 0, 88, 0}},
	})
	retainedCloudCellOwners1162(t, w)
	if err := w.ImportOriginalWorldEffectDrivers(&SavedWorldEffects{Areas: []SavedAreaDriver{{ID: 1, Root: 0, Identity: 17, Key: 0x0505, Layer: 4, Mode: areaModeCloud, Spell: 12, Cells: []uint16{0x0505, 0x0507}}}}); err != nil {
		t.Fatal(err)
	}
	if w.entities[0].HP != 5 {
		t.Fatal("LOAD replayed the template")
	}
	Step(w, nil)
	if w.entities[0].HP != 65542 || w.entities[1].HP != 5 || w.entities[2].HP != 5 || w.entities[3].HP != 65542 || len(w.attached) != 0 {
		t.Fatal("untimed32-bit operand / ground layer / radius / any same-spell presence", w.entities[0].HP, w.entities[1].HP, w.entities[2].HP, w.entities[3].HP, len(w.attached))
	}
	// Instant29 follows the live layer pointer, compares the full spell dword,
	// and writes the low16 duration bits. It must reach retained areas too.
	w.setCellEffectTime(5, 5, 268, 60000)
	w.setCellEffectTime(5, 6, 12, 60000)
	if w.SavedSpellEffects()[0].AE4C != 16 {
		t.Fatal("retime ignored spell width or layer identity")
	}
	w.setCellEffectTime(5, 5, 12, 60000)
	var retimed World
	if err := retimed.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	Step(&retimed, nil)
	if retimed.SavedSpellEffects()[0].AE4C != 59999 {
		t.Fatal("retimed native area froze")
	}
	w.setCellEffectTime(5, 5, 12, 65537)
	if w.SavedSpellEffects()[0].AE4C != 1 {
		t.Fatal("retime did not truncate duration to word")
	}
	Step(w, nil)
	if len(w.SavedSpellEffects()) != 1 || w.SavedSpellEffects()[0].AE4C != 0 {
		t.Fatal("zero boundary removed early")
	}
	Step(w, nil)
	cells := w.SavedCellRecords()
	if cells[0].SpellEffects[4] != 0 || cells[0].LayerCount != 1 || cells[0].SpellEffects[1] != 99 || cells[1].SpellEffects[4] != 17 || cells[2].SpellEffects[4] != 88 {
		t.Fatal("radius-bounded removal touched another layer or out-of-radius residue", cells)
	}
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal("retired driver rejected preserved residue", err)
	}
}

func TestRetainedProjectile1162RemovedTargetKeepsLastCurrentPosition(t *testing.T) {
	for _, originalDead := range []bool{false, true} {
		t.Run(fmt.Sprint(originalDead), func(t *testing.T) {
			var w *World
			if originalDead {
				w = deadWorld(t)
				if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(1, 3, -40)}); err != nil {
					t.Fatal(err)
				}
			} else {
				w = sourceMutationWorld(t, PlainItem(0xe01))
				w.entities[0].Humanoid = true
				w.entities[0].SourceBinding = SourceBinding{Class: 2, ArchiveIndex: 17, Identity: 0x123456, RuntimeID: 31}
			}
			if err := w.ImportOriginalProjectiles(SavedProjectiles{FreeIndex: 8, IDs: []uint16{7}, Items: []SavedProjectile{{ID: 7, Picture: 10, X: 10, Y: 2, Action: 1, ActionSegments: 3, ActionTarget: 31, ActionX: 30, ActionY: 40}}}); err != nil {
				t.Fatal(err)
			}
			if err := w.ImportOriginalWorldEffectDrivers(&SavedWorldEffects{Projectiles: []SavedProjectileDriver{{ID: 7, Phases: 4, Target: 1, HasTarget: true}}}); err != nil {
				t.Fatal(err)
			}
			beforeImport := w.Hash()
			wrong := w.SavedWorldEffectDrivers()
			wrong.Projectiles[0].TargetDetached = true
			if err := w.ImportOriginalWorldEffectDrivers(wrong); err == nil || w.Hash() != beforeImport {
				t.Fatal("premature target detachment admitted")
			}
			mustMarshal(t, w)
			Step(w, nil)
			before := w.SavedProjectiles().Items[0]
			if err := w.HeadlessPlace(1, 6, 5); err != nil {
				t.Fatal(err)
			}
			placed, _ := w.Entity(1)
			if !w.remove([]EntityID{1}) {
				t.Fatal("test removal incomplete")
			}
			d := w.SavedWorldEffectDrivers().Projectiles[0]
			p := w.SavedProjectiles().Items[0]
			if !d.TargetDetached || p.ActionTarget != 31 || p.ActionX != placed.X*256+128 || p.ActionY != placed.Y*256+128 || (p.ActionX == before.ActionX && p.ActionY == before.ActionY) {
				t.Fatal("removal captured stale aim or lost source key", d, p.ActionX, p.ActionY)
			}
			wrong = w.SavedWorldEffectDrivers()
			wrong.Projectiles[0].TargetDetached = false
			beforeImport = w.Hash()
			if err := w.ImportOriginalWorldEffectDrivers(wrong); err == nil || w.Hash() != beforeImport {
				t.Fatal("lost target detachment admitted")
			}
			var fresh World
			if err := fresh.UnmarshalBinary(mustMarshal(t, w)); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				Step(w, nil)
				Step(&fresh, nil)
				if w.Hash() != fresh.Hash() {
					t.Fatal("detached target native continuation")
				}
			}
			if len(w.SavedProjectiles().Items) != 0 {
				t.Fatal("detached target prevented completion")
			}
		})
	}
}
