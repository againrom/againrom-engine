# Story `1039` - closure

As-built evidence. `spec.md` owns the behaviour; this file owns the aspect
matrix, the shipped witness, the research reconciliation and the open work.

Base `e3466269`, research pin `d7ee0c6`. The pin did not move. Simulation form
version 59 is this story's reserved and shipped form.

## The result

A physical blow now consumes the target's Blade, Axe, Bludgeon, Pike or
Shooting resistance byte selected by the attacker's active melee kind. The
reduced value is the value that changes health and reaches attribution,
weapon-rider and experience decisions.

The pointable result is campaign mission 90 on both lawful installs. Its real
`F_KnightLeader2` placement lands the same first Blade blow on its real
`Catapult` placement at tick 10. With only the Catapult's decoded 40 percent
Blade resistance zeroed, the blow removes 19 health; with the shipped byte
kept, it removes 12. The production test prints the same line on `en` and `ru`:

```text
scenario/90.alm: u42 Blade blow at tick 10 dealt 19 without resistance and 12 against u290's 40% Blade resistance
```

That is a player-visible result: after this branch lands and `builds/current`
is rebuilt, the same kind of resistant target loses less health from the same
physical blow. The lane did not replace the shared build or drive the owner's
desktop.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `data.DamageKindResistance` narrows the five folded `int32` values to their low bytes once, with no 0..100 clamp. `FoldWeapon` assigns only supported melee kinds 1..5 and clears bare, ranged and unsupported rows to zero. Units, Humans and derived modifier values keep Blade, Axe, Bludgeon, Pike, Shooting order. |
| Runtime state | PASS | `sim.Entity` carries `[5]uint8 Resistance`; the existing canonical `XPSlot` carries the selector. `CombatBlock`, `DerivedBlock` and `GhostTemplate` carry the same state. The common world boundary validates both initial Entities and the retained ghost template; decode and the two live setters refuse a selector outside 0..5 before a partial write. |
| Simulation | PASS | Every landed physical fighter blow reaches `resolveBlow`. Ordinary hits subtract absorption, `AlwaysHits` skips it, the physical component floors at zero, then slots 1..5 apply `(damage*(100-resistance)+75)/100` in signed `int64`; slot zero bypasses it. No RNG draw moves. The reduced result is the health, attribution, rider and experience input. |
| Player input | PASS | The existing `KindAttack` command and its attack-cycle state still reach `advanceAttack` and the one `resolveBlow`. This story adds no command, cursor arm or alternate damage route. The release witness issues that production command rather than calling the arithmetic helper. |
| AI | PASS | AI and player ownership schedule the same attack cycle and share the same resolver; there is no AI-side damage estimate to update. AI observes the resulting canonical health, death and target state. The witness changes the shipped attacker's owner only after loading, because mission AI otherwise replaces a direct test order; every loaded combat field is retained. |
| UI/HUD | PASS | `mapWorld.entityDraws` overlays the character sheet's five weapon resistances from the live entity, beside the existing live elemental-protection overlay. A resumed or re-armed actor therefore shows the bytes the next blow consumes, in the existing Blade-to-Shooting order. |
| Triggers/scripts | N/A | No opcode, trigger field or script dispatch is introduced. A trigger or AI arm that causes an ordinary fighter attack still reaches the shared resolver. The complete script census is unchanged and measured below. |
| Inventory/equipment | PASS | Creature weapon folding, generated-party derivation and live `Rearm` update the active kind and all five bytes. Armour and shields retain their existing defence and absorption contributions and add no weapon resistance. Invalid and ranged weapon kinds clear the selector instead of indexing outside the family. |
| Persistence/save-load | PASS | Form 59 appends five bytes at entity offsets `+277..+281`, growing a record from 277 to 282 bytes without moving an earlier field. Encode, decode and `World.Hash` consume them. The explicit upgrade path widens readable forms with zero bytes and reports `SaveFormWeaponResistance`; version 58 keeps its already-present structure and script sections. A production Control Spirit raise is encoded, decoded into fresh worlds and then used as the resistance-selected attacker. |
| Campaign/session | PASS | Placement worlds, generated party members, carried party members, siege mercenaries, raised ghosts and live recomputes all reach the same canonical fields. Mission 90 is loaded with `game.StartMission` on each root; the witness then steps two worlds through the ordinary session command and combat path. The two ghost-bearing mission constructors reject selector 6 and 255 before a session can retain them. |
| Shipped content | PASS | `scenario/90.alm` is 166994 bytes with SHA256 `1af4b4ff040d48f4293020101d5042092328f2794254f438cfd8253aa6a006e1` on both roots. u42 loads at `(106,14)` with the worn Blade sword and 21..30 damage; u290 loads at `(102,14)` with absorption 7 and resistance `{40,0,20,60,80}`. Both roots produce 19 -> 12 at tick 10. |
| Interactions with existing mechanics | PASS | Tests distinguish absorption from the `AlwaysHits` bypass, slots zero through five, resistance 0 through 255, zero and negative results, negative source fields and the maximum reachable signed-field combination. Ranged physical selection is zero; book spells and weapon-spell riders keep their elemental path. A cancelled landed component changes no health, attribution or experience but still flips both relation cells; a positive reduction writes fighter attribution and pays XP from the reduced amount. The absent ranged second component and item-effect producer remain typed rows. |

No aspect is `GAP`. The four unbuilt behaviours in `spec.md` section 8 are
outside this contract and remain in the single divergence ledger.

## Shipped integration witness

`TestReleaseMission90BladeResistanceChangesOnePhysicalBlow` starts at the
production archive and definition front, calls `game.StartMission`, and locates
the actors by the authored map ids rather than by a synthetic class.

The attacker is entity 1, map unit u42, class 5, at `(106,14)`: HP 261,
damage base/spread 21/9, to-hit 202, reach 1, no always-hit flag and no weapon
spell. Its worn slot contains item `0101106`, resolved through the production
weapon tables to `Uncommon Bronze Two Handed Sword`, attack kind Blade. The
target is entity 152, map unit u290, class 26, at `(102,14)`: HP 250, defence
150, absorption 7 and the five shipped bytes `{40,0,20,60,80}`.

The test copies those exact loaded entities into two small production
`sim.World` instances with the same seed, spell rules and attack command. It
clears transient order state and makes the attacker player-owned so AI cannot
replace the directed order. Only the control target's resistance is zeroed.
Because resistance draws no random value, misses, cadence and the successful
damage roll stay lockstep; the first health change must occur on the same tick.

The control range is independently bounded by the shipped inputs:
`21..30 - 7 = 14..23`. The observed control damage is 19. The target's Blade
byte then requires `(19*60+75)/100 = 12`, which is the observed resisted damage.
The test fails if either root's map entry, placement, equipment, kind, decoded
resistance, tick, range or arithmetic changes.

The synthetic witness beside it enumerates selector slots 0 through 5 and all
256 resistance bytes. Named boundary cases cover resistance 0, 75, 76, 99,
100, 101 and 255; positive, zero and negative components; positive and
negative absorption; `AlwaysHits`; and the largest component reachable from
the three signed `int32` combat fields. The arithmetic stays within `int64`.

Two downstream cases expose the result rather than only the arithmetic. A
100-percent cancellation leaves health, attribution and all six XP banks
unchanged while the landed blow makes both owners hostile. A 50-percent result
from a fixed 100-point blow removes 50, records the fighter as source and pays
the Blade bank from 50 rather than from the unreduced 100.

## Persistence and hash witness

Two nonzero entity records prove that bytes `+277..+281` round-trip in their
own record and order. Peeling exactly those tails restores the complete
version-58 byte fixtures and their prior hashes; the version test is named
`TestThePinIsThePreviousPinPlusWeaponResistance`, with no live version number
in its name. Separate hash fixtures prove that changing a resistance byte
changes canonical `World.Hash`.

Upgrade fixtures cover version 57, which still needs the structure section as
well as resistance, and version 58, which needs only the five entity bytes.
The widened values are zero because the format layer has neither definitions
nor equipment input from which to reconstruct them. The returned loss notice
states that fact rather than claiming a recompute already occurred.

The constructor population is complete and explicit. `NewWorld`,
`NewTerrainWorld`, `NewRelatedWorld`, `NewScriptedWorld`, `NewSpelledWorld`,
`NewLootWorld`, `NewStockedWorld`, `NewStockedSpelledWorld`,
`NewSummoningWorld` and `NewStructuredWorld` all reach `newWorld`.
`NewSummoningWorld` and `NewStructuredWorld` are the only two that accept a
raisable GhostTemplate; the shared table witness accepts selector 0 and 5 and
rejects 6 and 255 through each constructor.

Control Spirit's `w.entities = append(w.entities, ghost)` is the sole runtime
population append. Its production cast consumes a bones-stage corpse, copies
selector 4 into the raised Entity, marshals the world and unmarshals the real
form into two fresh worlds. The decoded form is byte-identical and keeps the
same canonical hash. Its ghost then deals 100 against the zeroed control and 50
against the target's 50-percent Pike byte. The adjacent older Domain-255 path
was reproduced separately: construction and the raise accept it, encoding
writes it, and decoding refuses `movement domain is 255`. It is disclosed as
`DIV-366`, not folded into this selector correction.

## Script census and game drive

This story was not meant to move the script-node census. A lane-local
`missionrun` was built from the code commit and passed to
`pipeline/check-milestone.sh` through `AGAINROM_MILESTONE_DRIVE`; it reported
that the complete script gap and drive remain at the recorded baseline on both
roots.

The brief's direct count, `missionrun -mission N -trace -ticks 1 | grep -c
UNSUPPORTED`, is 0 for mission 10 and 0 for mission 20 on `en`, and 0 and 0 on
`ru`. `pipeline/milestone-baseline.txt` carries no `cannot run` row for either
mission, so those values are unchanged before and after. The full drive remains
`outcome lost at tick 240`, with 4 of 36 units moved and 1 fallen over 240
ticks, on both roots; the outcome is recorded rather than asserted.

## Research reconciliation

Every rule below was read through the pinned `tools/claim` reader, including
its current retraction state.

- `HERO-DAMAGE-022` and its amendments in `HERO-AUTOHIT-031` and
  `HERO-CLAMP-030` are reproduced in order: hit, conditional absorption,
  immediate zero floor, active-slot resistance with the published rounding,
  then the complete nonnegative result.
- `UNIT-COMBAT-006` and `UNIT-COMBAT-015` bind the six selector states and the
  five Units columns. The ranged arm writes zero. The row's retraction concerns
  only its former 30-versus-38 reach statistic; this story relies on its
  field/column bindings and melee/ranged stores, which the corrected active row
  keeps High.
- `HERO-FOLD-033`, `UNIT-WIDTH-016` and `HERO-MOD-016` establish byte-width
  modulo storage and the additive modifier family. This story implements the
  consumer; the absent production item-effect contributor remains `DIV-365`.
- `HERO-ARMOUR-018`, `HERO-EQUIP-017` and `ITEM-ARMFOLD-033` keep ordinary
  armour and shields on defence and absorption, outside both protection
  families.
- `HERO-DMG2-029` gives a separately gated second component and elemental
  protection path. This build has no state for it and no shipped class takes
  that arm from its class row, so it is not fabricated inside the first
  component. `DIV-363` owns the complete later slice.
- `MAGIC-RESIST-006` keeps spells on elemental `Protection`. Synthetic spell
  witnesses set all five weapon bytes to 255 and still observe the existing
  elemental result.

`pipeline/check-div-claims.sh` lists `DIV-363` and `DIV-364` among rows whose
citations have some retraction record. Both were re-read. `DIV-363` relies on
`HERO-DMG2-029`'s active component routing; `DIV-364` relies on
`HERO-DAMAGE-022`'s active index and `UNIT-COMBAT-006`'s corrected bindings.
Neither relies on the retracted reach count.

## Divergence ledger

| Id | State | Reconciliation |
|---|---|---|
| `DIV-361` | CLOSED | The decoded family now reaches every production entity producer, the resolver, form and hash, and the EN/RU shipped witness changes a blow. |
| `DIV-362` | CLOSED | `AlwaysHits` now skips absorption as well as the hit comparison; the 100/20/50 witness distinguishes 50 from the ordinary arm's 40. |
| `DIV-363` | OPEN / FIDELITY-DEBT | The ranged second component, its to-hit source and elemental protection path still have no complete state or resolver slice. |
| `DIV-364` | OPEN / UNKNOWN | Pinned research gives no safe target byte for unsupported melee attack types. Production normalizes them to slot zero. |
| `DIV-365` | OPEN / FIDELITY-DEBT | The derived modifier array has a consumer, but no production magic-item/effect fold populates it. Ordinary armour remains correctly neutral. |
| `DIV-366` | OPEN / FIDELITY-DEBT | Both ghost-bearing constructors accept an undefined template domain; the raised Entity encodes but cannot decode. This pre-existing adjacent invariant is reproduced and left to its own slice. |

Reserved ids `DIV-367` and `DIV-368` are unused. Before the contract, the
allocator answered `DIV-369`, with highest mention `DIV-368` at
`PIPELINE-STATUS.md:165`. The post-reservation/current rerun answered
`DIV-401`, with highest mention `DIV-400` at `PIPELINE-STATUS.md:193`; other
live reservations moved the shared namespace while this lane was open. This
story spent only its reserved `DIV-366` after reproducing the adjacent defect,
took no unreserved id and did not edit `PIPELINE-STATUS.md`.

## Gates on the code commit

The implementation gates below ran in `wt-impl1039` at corrected code commit
`030a8875`.

| Gate | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l .` | no output |
| `go test -trimpath -count=1 ./...` | exit 0; every selected package passed |
| `scripts/check-claim-citations.sh` | exit 0; 1285 distinct citations resolve against 1479 claims and 220 experiments under 787 prefixes |
| `scripts/check-no-game-assets.sh` | exit 0; clean tree scan |
| `pipeline/check-div-claims.sh` with `AGAINROM_IMPL` | exit 0; the printed checkout is `wt-impl1039 @ 030a8875`, 236 live rows selected, 0 malformed |
| `pipeline/check-pin-forward.sh story/1039-weapon-resistance` | exit 0; branch and master both pin `d7ee0c6` |
| `pipeline/check-release-tests.sh` on `en` | exit 0; 47 of 47 install-gated tests passed, 0 skipped |
| `pipeline/check-release-tests.sh` on `ru` | exit 0; 47 of 47 install-gated tests passed, 0 skipped |
| `pipeline/check-scenarios.sh` on `en` | exit 0; 15 of 15 |
| `pipeline/check-scenarios.sh` on `ru` | exit 0; 15 of 15 |
| `pipeline/check-milestone.sh` with the lane-local drive | exit 0; complete census and drive match the baseline on both roots |
| `pipeline/check-preserved-installs.sh` | exit 0; 162 files, both roots as recorded |

The two focused release runs on the same commit each printed tick 10 and
19 -> 12. The constructor, downstream consequence and Control Spirit
round-trip tests also passed in the complete suite. The final
documentation-only commit is followed by the full Go test, the citation and
ledger readers, the asset scan, `git diff --check`, and the trailer audit before
push.

## Review scope and stopping

The independent reviewer receives five touched domains: Assets and Content,
Simulation Core, Combat and Magic, Party Items and Heroes, and Persistence. The
named surface is every producer, the selector boundary, exact physical
arithmetic and downstream writes, form/hash/upgrade, live sheet overlay, the
EN/RU witness and every matrix row above.

The story reaches hashed simulation state and touches five domains. Its ceiling
is four adversarial passes; five is the project absolute, not a target. The
chain ends on the first pass with no P finding after the named surface is
exhausted. W becomes a ledger row and D is corrected in place. If the fourth
pass still finds P, the remaining surface becomes a separate defect story under
the pipeline ceiling rule.

No adversarial pass was self-run. The pushed branch is frozen for a fresh
reviewer; this lane does not merge it or rebuild the shared `builds/current`.
