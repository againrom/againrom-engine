# Story `1045` — closure

As-built evidence. `spec.md` owns behaviour; this file owns aspect closure, shipped witnesses,
research reconciliation and the remaining review surface.

Research pin `23daf74f`; reviewed story-1044 master `69c0985b` merged at `13ede1c7` without changing
its additive outer campaign fields. Later implementation master `8f6e68e3` merged at `ebb8ec30`,
then exact master `5e5cc97a` merged at `1c5615b1`; neither merge changed this story's simulation
state. The cadence production commits are `544394cd`, `c62eb890`, `90c1ed4e`, `d78369e6`,
`6abcc6da`, `d3c1ee18`, `c32e97d0` and pass-2 correction `743885a8`. Simulation form 62 is current.

## Result

Retained physical attacks now repeat after decoded charge, ranged flight, recovery, inclusive
random jitter, the equipped-Humanoid weight/Reaction penalty and two scheduler boundaries.
Ordinary book casts add Complication and retain their owner-authored eight-tick visible-swing floor.
Mage weapon diversion and fighter weapon riders keep their separate cadence identities.

A completed cast with insufficient mana attempts three times, skips once when completion is
consumed and repeats; an incomplete cast attempts every tick. Later mana admits the same retained
Spell and target without another command. Mana lost during charge returns a retained record to
pending state and removes a one-shot record. Physical application revalidates presence, linkage and
reach but does not reject a target merely because another actor already reduced its health to zero
or below.

Repeated explicit unit and cell commands cannot replace a retained record during recovery or its
two scheduler boundaries. Insufficient one-shot autocasts store no pending target. Command damage,
later-ID combat, equipment changes and derived-health recomputation clear every action residue in
the death transition. Book, autocast, mage-diverted and fighter-rider area self-kills do not restore
recovery afterward. Target health alone does not terminate approach before physical or fighter-rider
application.

An area release may remove book records below, at or above the releasing record while the sorted
book sweep is active. The sweep rebinds the releasing record by `EntityID` after application. Each
surviving record advances once, a dead caster retains no record, and a completed one-shot release
removes only its own record.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | The Spell loader preserves the Complication byte independently. Human, party and raised-actor producers provide explicit Humanoid values. Runtime weapon weight remains the declared item-weight input. |
| Runtime state | PASS | Physical phase/countdown and sorted retained book records carry charge, recovery, completion, retry progress and retention through every boundary. One-shot pending, dead `CastWait` and a book record on a dead caster are non-canonical. |
| Simulation | PASS | Focused interval tests isolate every physical and book term, both weapon-Spell routes, retained and one-shot insufficient-mana paths at admission and release, later mana, the complete death-producer population, side-effectful area-release record removal and target revalidation. |
| Player input | PASS | One `KindCast` remains armed through insufficient mana and later regeneration. Existing attack and cast commands converge on the lifecycle without a second input mode. |
| AI | PASS | Offensive and armed restorative autocasts return to current selection after insufficient mana. Idle Heal keeps its affordability gate. Attack AI enters the common physical executor. |
| UI/HUD | N-A | No drawing or animation clock changes. Existing blow and cast events remain the client boundary. |
| Triggers/scripts | PASS | Script and player commands retain established priority. Script casts remain their separate immediate producer and do not acquire retained-order state. |
| Inventory/equipment | PASS | The equipped primary item selects armed/unarmed Humanoid penalty and fighter-rider versus mage-diversion routes. Replacement, unequip and worn-item drop normalize a health-loss death after the item move completes. |
| Persistence/save-load | PASS | Form 62 stores every later-affecting input and lifecycle byte, hashes them, refuses non-retained pending, dead recovery and absent/dead caster residue, and migrates every supported older form without guessing absent facts. |
| Campaign/session | PASS | The story adds no campaign field. Form 62 nests unchanged inside story 1044's additive outer snapshot and resumes through the production save path. |
| Shipped content | PASS | EN and RU release drives load mission 90's placed Humanoid and weapon plus mission 10's generated mage and installed positive-cost Spell. |
| Interactions with existing mechanics | PASS | Busy priority, repeated explicit input, movement during casts, current perception, mana regeneration, death, target teardown, item instances and deterministic random order retain their established boundaries. |

No in-scope aspect is `GAP`.

## Exact shipped witnesses

On each lawful root, `TestReleaseActionCadenceUsesShippedHumanoidWeaponAndSpellInputs` starts
mission 90 through the production archive and map loader. Its UnitID 42 is explicitly Humanoid and
holds a declared runtime-weight weapon. Three retained adjacent applications fall inside the
literal interval

`charge + relax + humanoidPenalty + U[0,3] + 2`.

The same test starts mission 10 with the production generated mage, requires a non-empty installed
spellbook and chooses an admissible positive-cost unit Spell from the normalized installed table.
The first ordinary release exhausts mana. No second command is supplied; the retained order waits
through insufficient-mana retries and releases the same Spell after ordinary mana regeneration.
Both EN and RU focused drives pass. No GUI or game process is launched.

## Research reconciliation

- `HERO-CADENCE-112` supplies the physical interval and two scheduler-boundary ticks.
- `HERO-CADENCE-113` supplies the Humanoid gate, signed division and clamped runtime-weight and
  Reaction formula. Its shipped population remains witness evidence, not canonical arithmetic.
- `HERO-CADENCE-114` supplies convergence after arming and the mage-diversion versus fighter-rider
  distinction. Upstream selection remains bounded.
- `HERO-CADENCE-115` supplies death cancellation, physical presence/link/reach revalidation and the
  absence of a target-health precheck. External replacement and general Spell target loss remain
  Unknown.
- `MAGIC-CADENCE-126` supplies Complication, pointer-clear ordering and book versus weapon-diverted
  recovery. `DIV-425` records the accepted conflict for the owner-authored eight-tick book floor.
- `MAGIC-CADENCE-127` supplies incomplete retry, completed three-attempt/one-skip retry and later
  admission without failed-cast recovery.

`DIV-022` still owns cross-producer priority, facing and current perception. `DIV-028` still owns
movement admitted during a pending cast. Neither row owns the separate eight-tick floor mismatch;
`DIV-425` owns that conflict with `MAGIC-CADENCE-126`'s decoded-charge interval.

## Remaining surface

The remaining-surface list is empty across canonical inputs; physical, book, mage-diverted and
fighter-rider formulas; repeated unit/cell input; retained and one-shot insufficient-mana boundaries
at admission and release; actor death before, during and after application through command, combat,
equipment and derived-health producers; physical target presence, linkage, reach and non-positive
health through approach and teardown; lower, releasing and higher book-record removal during one
area application, including multiple removals and surviving or dead releasing casters; every
form-62 field; malformed
lifecycle classes; all supported migrations; hash separation; save/resume; and EN/RU shipped
inputs. External progress replacement and Spell-specific target loss remain expressly outside the
contract rather than untested in-scope behaviour. No in-scope production or administrative surface
remains.
