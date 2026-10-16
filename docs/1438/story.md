# Simulation-owned flight records

## Intent

Every first-game flight record is created by the simulation step. Ordinary play
and headless play use the same installed phase and launch inputs. The window
consumes current records for display, and SAVE writes them through the existing
current-state producer.

## Authority

Owner direction. This is the projectile-producer result deferred by story1395,
after the Group, Mover and cell-plane carriers. The simulation owns the one
flight-record builder and lifecycle. Installed phase counts and launch offsets
become World inputs; appearance remains in the window layer.

## As built

`sim.FlightInputs` carries immutable picture phase/lifetime/homing data and
class launch data. The shared definition loader binds it to a first-game
World before its first step and after LOAD; raw registry rows do not depend on
rendered sheets. Picture enumeration follows the installed IDs. Hero class
selection reads the current actor and equipment through a pure installed rule.

The simulation creates records from cast, cell-cast, staged-area and physical
attack events through the existing record builders. The Fire Ball burst keeps
its simulation builder. Events name the created IDs and initial trail points;
the window attaches looks and trails without another release. ROM2 keeps its
existing producers.

`Entity.FlightRun` holds the physical release clock that the window previously
owned. Actor continuation writes it to SAV, and the optional FLR1 byte-form
wrapper preserves it through World decoding. Cancellation clears it. Legacy
absence seeds a current wind-up through the existing action clock. The input
catalog is rebound independently of persisted state.

`ClientClassFixed` retains an actor constructor's fixed body-class policy. A
hired actor whose valid type also lies in the hero type band keeps its class;
an actual hero with that type still derives its class from current equipment.
The typed policy uses the same current continuation and optional FLR1 wrapper,
without interpreting an original record's unproved marker as a hire.

Base: main48ad5680, knowledge k315. Existing record SAVE and LOAD remain the
current-state producer. Render-only appearance remains outside World state.

## Proof

The regression first failed on tick2: ordinary Attack and StepReported without
the window held no record, while the ordinary window tick held ID0/picture1.

Focused simulation checks cover physical delays0/2/6 against units and
structures, a decode before release, book/staff/rider and cell-entry arms13/14.
Controlled direct-simulation and ordinary window runs compare record stores,
IDs and World hashes by tick; consuming their observations again changes no
state. An empty class produces no physical release clock.

EN and RU run an installed StartScenarioMission cast without a mapWorld
consumer. Eight ordinary SAVE/fresh-process families cover book, staff, staged
burst, Fire Ball, client Lightning/Prismatic Spray, unit and structure targets.
Each fresh child uses the ordinary headless FrontEnd tick and compares49 ticks
of record stores, IDs, drivers and World hashes. Installed delay21 shots also
save before release. Raw SAV fields retain the existing dropped-record and
changed-leaf loss controls; a structure target is checked against independently
decoded Building identities. The separate installed weapon-rider case is not
claimed; its ordinary-command coverage is synthetic.

Raw catalog tests cover picture1007 without art, missing registries, live
equipment/class selection, constructor entry before tick0 and a render sheet
that disagrees with raw burst phases. A valid high-band hire keeps its fixed
class across current SAV/cold LOAD; clearing the typed policy gives the same
type's ordinary hero rule. The existing profile witness remains unchanged on
EN and RU. No existing behavioral comparison was weakened.

Binary-copy continuation probes bind the same installed catalog as their
source. The decoded-install isolation probe compares picture and class data
and checks the class rule beside its other function inputs. Cell-entry probes
retain exact report/hash equality and check the new record identity. The
destination-writer guard distinguishes an Entity order from a record recipe;
corrupt FLR1 input produces a bounded LOAD refusal without a package prefix.

The sole adversarial review passed on candidate `921354d9`. Final chain
receipts belong to the landing commit. Local command receipts and retained
failed attempts are outside Git under the story's owner artifact directory.

## Open debt

No original runtime result is claimed. This change does not close the entire
C4/C5 remainder: original structure-as-shooter behaviour, missing-offset
inheritance, native zero-delay chronology, mission-transition allocation,
burst-tail timing and the broader effect/callback graph retain their existing
bounded divergence rows. DIV-1786 and DIV-1878 were already closed before this
story. ROM2 behaviour is preserved; no ROM2 feature is part of this result.
