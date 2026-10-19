# Usable mission structures

Usable structures receive the existing pixel hit, town cursor, structure
readout and use order even when their maximum word is zero. The readout
substitutes 1000/1000 without changing either simulation word.

## Authority

Knowledge k330: UNIT-153, UNIT-154, UNIT-155, UNIT-156, AI-484, AI-485,
UNIT-STRUCTHIT-138, UNIT-STRUCTUSE-137, AI-CURSOR-231, AI-CURSOR-226,
AI-STRUCTUSE-306, MENU-072, REG-STR-080, UNIT-STRUCTUSE-090,
UNIT-STRUCTUSE-091, UNIT-STRUCTZERO-080, UNIT-STRUCTCONT-078, MENU-071,
ITEM-USE-113 and MAGIC-CONSUME-142. The publication is
knowledge/formats/unit/structures.md, Installed usable population and pool.
These are ROM1 claims. Engine witnesses establish this implementation only.

## Behaviour

| Class | Simulation maximum | Readout | One completed use |
|---|---|---|---|
| 15 | 3 | current/3 | Positive charge spends one; installed Big Healing potion restores up to 100 health |
| 16 | 3 | current/3 | Positive charge spends one; installed Big Mana potion restores up to 100 mana |
| 28, 29 | 1 | current/1 | Zero becomes one; any nonzero state becomes zero |
| 34, 35 | 0 | 1000/1000 | Approach and finish; mission Shop entry remains DIV-284 |
| 39, 40 | 0 | 1000/1000 | Approach and finish; no invented local effect, DIV-3689 |
| 66 | 0 | 1000/1000 | Approach and finish; unresolved receiver effect, DIV-3690 |

All nine use the town cursor for an eligible selected player character.
The existing cursor's actor, modifier, fog and selection gates still apply.
Empty wells remain usable and cannot spend below zero. Their existing counter-60
recharge policy is unchanged. Full actor pools still spend a positive charge.
Bridges, decorations, nonusable structures and town Shop controls keep their routes.

UI use admission calls sim.Structure.Usable through UseAdmission, a read-only
interface in MapStructure. game.mapWorld.structureDraws supplies a value copy
of the live sim.Structure, including its Kind, Width and Height. The rule admits
the nine selectors above with nonzero width and height.
The registry flag supplies no separate admission rule. The same answer gates
zero-maximum inspection, usable hover capability, town cursor and click.

ROM2 registry-usable classes outside the nine remain off the town cursor and
cannot send a use order. Zero-maximum classes outside the nine have no readout,
as at base 55048dcf. IDs 67-77, 93-95, 99-101, 105-107 and 111-113 contain 23
selectors. The installed registry witness
enumerates every usable class on each supplied root. Their second-game use
effects have no evidence here and are not admitted in the simulation.
An existing nonzero health pool keeps the base physical hover and readout;
it grants no use capability. The pooled indestructible class-70 fixture keeps
the select cursor and plain-click selection result without a move or use order.

The 1000/1000 substitution never supplies attack health. Structure attack
admission reads live signed Health and requires a nonzero MaxHealth. Real Ctrl
input also raises AttackHeld: the cursor is attack at base and here, but a
zero-maximum target produces the ground Move order rather than AttackStructure.
That holds even when a script has set its current word to one.

## Builder census

| Responsibility | Single production route | Change |
|---|---|---|
| Installed class flags | game.LoadStructures, structures.go | Unchanged; registry flags remain metadata, not use admission |
| State and maximum | mapload.Structures, fromalm.go | Unchanged; Buildings parameter 3 already yields 3/3/1/1/0 |
| Potion metadata | mapload.StructureUseMetadata | Unchanged; installed named potion operands |
| Pixel hit and subject | ui.inspectionAt, inspectionPanel, inspection.go | Ask usableStructure when maximum is zero; project 1000/1000 |
| Capability mask | ui.hoverMask and structureHoverHit, missioninput.go | Keep base pooled physical hits; admit zero-maximum uses through usableStructure |
| Usable capability | ui.usableStructure, structureuse.go | Ask UseAdmission.Usable on the simulation snapshot; no selector list |
| Admission projection | game.mapWorld.structureDraws | Supply the live sim.Structure value through MapStructure.UseAdmission; no UI-to-sim import |
| Cursor and click | ui.missionHoverCursor, hoverCursorName, command.go; App dispatch | Same town slot and target; attack admission reads live Health/MaxHealth |
| Order sink | game.mapWorld.useStructure | Unchanged; typed sim.UseStructure |
| Admission | sim.Structure.Usable | Admit all nine selectors independently of charge |
| Approach and completion | sim.beginStructureUse, advanceStructureUse | Unchanged; one mover and one completion |
| Effect | sim.applyStructureUse | Restrict potion branch to 15/16 with a resolved positive operand |
| SAV producer | game.projectSavedStructures and projectSavedStructureRecord | Unchanged; current Field42 and MaxHealth |
| SAV consumer | original structure import/current bindings | Unchanged; restored words take precedence over placement |

No new persisted field or save version is introduced. Legacy deterministic defaults
remain the existing metadata restoration law. No command kind or constructor changes.
The headless cursor seam returns mapCursorPresent's actual emitted picture and
position for the component frame. Comment bytes fall by 118: the obsolete
zero-pool inspection comment is removed, the state-word description is corrected
and the new headless seam has one short doc comment. The shorter MapStructure
snapshot description removes 20 bytes; the physical-hit comment removes 8. Three reused test helpers
lose their numeric suffixes, removing 19 references. The combined TestIdentCount
is 4519.
Other storyguard counters and the composition/command literal baselines do not change.

## Proof

TestUsableStructureHoverReadoutAndOrder covers all nine classes through the
production pointer, readout and order sink. TestZeroMaximumStructureApproachAndBoundedUse
covers approach, completion, cold simulation continuation and the absence of a
potion effect for all five zero-maximum selectors, including a positive script word.
TestStructureUse1150FountainPoolsAndCharges and
TestStructureUse1150ApproachFacingAndCancellation cover positive/empty/full pools,
the one-use boundary and both switch directions.

TestUsableStructureAdmissionMatchesSimulation compares hit, cursor, readout,
use sink and town acknowledgment with simulation admission, including empty
dimensions and class 70's explicit refusal. TestReleaseUsableStructureRegistryAdmission
applies that matrix to every installed registry-usable class. Restoring the
registry flag as the UI rule fails both tests, including installed class 70.
TestUnadmittedStructureKeepsPooledPhysicalHover pins the base pooled class-70
hover, readout and click without admitting use. Removing the pooled physical
fallback fails it. TestZeroMaximumStructureCtrlClickUsesGroundOrder drives real Ctrl/AttackHeld
press and release for all five zero-maximum classes at current words zero and
one. Restoring panel.HP attack admission fails it. The same test passes with
the relevant inspection and command sources from base 55048dcf in a compile
overlay. These controls leave the worktree sources unchanged.

TestReleaseStructureUse1150 enumerates every readable shipped mission in 1..160
on each root. It verifies each of the nine table maxima and logs every placement
count. It drives one placed structure per class through the installed pointer,
approach, effect, SAVE, cold LOAD, next-tick hash equality and completed-use hash
equality. The actor population and initial actor position are controlled; terrain,
structure placements, script and potion data come from the mission. Cell tails
are retained when constructing this controlled witness.

| Class | Placements per root | Witness mission / structure | Completion tick |
|---|---|---|---|
| 15 | 9 | 90 / 38 | 45 |
| 16 | 6 | 90 / 27 | 57 |
| 28 | 8 | 91 / 9 | 63 |
| 29 | 9 | 101 / 11 | 68 |
| 34 | 0 | No placement in the 28 readable shipped maps | - |
| 35 | 1 | 120 / 71 | 61 |
| 39 | 2 | 61 / 15 | 44 |
| 40 | 8 | 20 / 20 | 70 |
| 66 | 0 | No placement in the 28 readable shipped maps | - |

Each hover is at tick 0. EN and RU agree on these counts and completion ticks.
Loss controls temporarily remove zero-maximum inspection admission, the usable
cursor capability, the 1000 fallback, the five-selector admission, the charge
decrement, the zero-to-one switch store, or the current SAV B42 producer. Each
must fail its named test; the mutations are restored before the ordinary gates.
The SAV witness reads the actual decoded Building body bytes 60..63 and checks
the completed world's hash against cold LOAD, in addition to pending-order replay.

Optional AGAINROM_USABLE_STRUCTURE_WITNESS_DIR receives cropped CPU readout
and installed town-cursor art. These are component composites, not GPU screenshots.
Generated images and logs stay outside Git in review/story1470-usable-structures.
The same directory holds loss controls, final gates, scenario/hash census and
the exact base/head receipt in PROGRESS.md.

## Open debt

DIV-284 retains mission Shop entry. DIV-2106 closes the zero-maximum hit/readout/use
omission. DIV-3689 bounds the grave route to AI-485's local action; mission scripts,
other nearby receivers and native visible effects remain Unknown. DIV-3690 leaves
class 66 receiver mutations and their persistence Unknown. No native executable,
window, audio or original use/SAVE/LOAD observation is part of this implementation.
The installed GPU pause witness is explicitly skipped under AGAINROM_HEADLESS_ONLY.
