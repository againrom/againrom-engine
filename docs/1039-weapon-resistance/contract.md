# Story `1039` - weapon damage-kind resistance reaches a blow

**Contract, lane, 2026-08-23.** Base `e3466269`, research pin `d7ee0c6`.
Branch `story/1039-weapon-resistance`, worktree `wt-impl1039`.

`DIV-361` through `DIV-368` are reserved to this story. The allocator answered
`DIV-369`, with the highest mention at `PIPELINE-STATUS.md:165`, before this
contract was written. When the range is spent, the lane stops and asks the seat.
It does not take another id from an unmerged worktree.

Simulation form version **59** is reserved exclusively to this story. Master at
the base carries version 58. No other open implementation lane may take 59.

## Result

A physical blow is reduced by the target's resistance to the weapon damage kind
the attacker is using. Blade, axe, bludgeon, pike and shooting select five
separate bytes. A bare or ranged attack selects slot zero and bypasses this
family. The reduced amount, rather than the amount before resistance, changes
health, kill attribution, weapon riders and experience.

The pointable result is one shipped campaign target, loaded from each lawful EN
and RU install through the production campaign loader, whose health loss changes
when its decoded resistance byte is kept instead of zeroed. A synthetic witness
beside it gives the exact arithmetic and every boundary.

## The measured absence

Re-measured on base `e3466269`, not inherited from the backlog:

- `data.UnitDef.Resistance` decodes Units columns 24 through 28, and
  `data.Derived.Resistance` carries the corresponding derived family.
- `mapload.Sheet.WeaponKind` exposes those five values to the character pane.
- `mapload.spawnBlock`, `sim.Entity`, `sim.CombatBlock` and `sim.GhostTemplate`
  carry `Protection` but no weapon-kind resistance.
- every placement, party, mercenary, ghost and live re-arm producer therefore
  drops the family before simulation.
- `sim.resolveBlow` subtracts absorption and then health. It reads neither the
  attacker's active damage kind nor the target's resistance.
- the byte form and hash carry neither value.

This is a parsed-but-unconsumed production gap, not a missing table decode.
`DIV-361` records it until closure.

## Research contract

Only the pinned research submodule is authority for ROM1 behaviour.

- `HERO-DAMAGE-022` (High) gives the one physical hit routine and its order:
  roll, hit test, absorption, then the byte at target offset `0xce + active
  skill`, with `ftol(damage * (100 - resistance) / 100 + 0.75)`.
- `HERO-CLAMP-030` (High) gives both clamps: physical damage floors at zero
  after absorption and before resistance; no clamp follows the multiply, but
  the complete result floors at zero. Resistance is unsigned and may exceed
  100.
- `UNIT-COMBAT-006` and `UNIT-COMBAT-015` (High) bind slots 1 through 5 to
  Blade, Axe, Bludgeon, Pike and Shooting, bind slot zero to the unfilled byte
  at `+0xce`, and establish the five Units columns. The ranged equip arm clears
  the active slot.
- `HERO-FOLD-033` and `UNIT-WIDTH-016` (High) establish byte storage: the six
  damage-kind bytes are additive and wrap modulo 256 rather than clamp.
- `HERO-AUTOHIT-031` (High for the absorption bypass) establishes that an
  always-hit source skips absorption as well as the hit roll. The current
  resolver does not; `DIV-362` records that adjacent defect until closure.
- `HERO-ARMOUR-018`, `HERO-EQUIP-017` and `ITEM-ARMFOLD-033` (High) establish
  the existing armour interaction: armour and shields add defence and
  absorption, but not either protection family.
- `HERO-DMG2-029` (High) gives the ranged weapon's second component and its
  elemental protection path. This build has no such component; `DIV-363`
  keeps that known debt out of the physical component implemented here.
- `MAGIC-RESIST-006` (High) gives damaging spells their separate elemental
  protection path. Weapon-kind resistance does not apply to a book spell or a
  weapon-borne spell rider.

The research does not establish a safe meaning for a melee weapon attack type
outside 1 through 5, although the shipped Weapons table contains a row with
`-1`. Production narrows unsupported kinds to slot zero and simulation refuses
an out-of-range canonical byte; `DIV-364` records that decision. The decoded
modifier block can add damage-kind bytes, but this build has no item-effect
fold that populates those additions; `DIV-365` records that separate debt.

## Behaviour groups

**G1 - one data boundary.** The five signed table or derived integers narrow
once to five unsigned bytes with modulo-256 semantics. Every production entity
producer carries them. The equipped weapon's supported melee attack type fills
the existing canonical `XPSlot`: research gives one active-skill byte for both
the resistance index and physical attribution, and a second field would create
two sources for one fact. Ranged, bare and unsupported kinds become zero.
Creatures carry the slot even though `GainsXP` keeps them from earning.

**G2 - the complete physical arithmetic.** A landed ordinary blow subtracts
absorption and floors at zero. An always-hit blow skips absorption. A positive
physical component is multiplied by `100 - resistance`, rounded by the
published `+0.75`/truncate rule, and the result floors at zero before any
health, attribution, rider or experience write. Slot zero bypasses the array;
slots 1 through 5 index it exactly.

All intermediate arithmetic is `int64`. The input damage and absorption are
`int32`; a resistance byte is at most 255. Their product therefore cannot
overflow `int64`. Health retains the resolver's existing `minHP` clamp. No RNG
draw is added, removed or reordered.

**G3 - canonical state.** The existing `XPSlot` and five new resistance bytes
are part of `Entity`, `CombatBlock`, the ghost template and every live
recompute. They are serialized and hashed. Form 59 appends five bytes to each
entity record. `UnmarshalBinary` refuses every prior version; the existing
explicit `UpgradeSaveForm` path widens readable versions with zero resistance
and a player-facing loss disclosure, because the format layer has no definition
or loadout input from which to recover the missing bytes. Constructor, decoder
and setter already refuse `XPSlot` outside 0 through 5 without a partial write,
and this story normalizes every new producer before it reaches that gate.

**G4 - witnesses at both scales.** Synthetic simulation tests enumerate slots
0 through 5, every resistance-byte boundary, modulo narrowing, rounding,
zero/negative outcomes, absorption, always-hit, riders, experience, setter
atomicity, form offsets, round trip and digest. The release witness uses real
campaign data on both installs and invokes the ordinary production combat
step. Spells prove the negative interaction: they keep using elemental
protection and do not consult this family.

These four behaviours are one vertical slice. Splitting after G1 would land
parsed-but-unused data; splitting before G3 would let save/load and hash erase
a combat fact; splitting the witness from G2 would leave the shipped path
unmeasured.

## Domains

Assets and content, Simulation Core, Combat and Magic, Party Items and Heroes,
and Persistence. The reviewer walks every producer at those interfaces:
placements, humans, generated party, siege mercenaries, raised ghosts, live
re-arm, binary decode and physical resolution.

## Twelve-aspect obligations

Closure must answer all twelve aspects explicitly. Data, runtime state,
simulation, persistence, shipped content and interactions are direct PASS
candidates. AI must show that it reaches the same resolver and observes the
changed health rather than owning a second damage estimate. UI/HUD must show
that its existing derived readout remains in the same Blade-to-Shooting order.
Input and triggers must show that no alternate hit path bypasses the resolver.
Inventory/equipment must cover live re-arm and the armour non-interaction.
Campaign/session must cover the production loader on both roots. An aspect may
be N/A only with a named boundary; an in-scope GAP fails closure.

## Outside this contract

- the ranged second damage component and its ranged to-hit source (`DIV-363`);
- decoding item or magic-effect contributions into `EquipMod.Resistance`
  (`DIV-365`);
- changing the simulation RNG;
- widening forms below the existing readable floor.

They remain visible in the single divergence ledger rather than only here.

## Review ceiling and stopping condition

This story reaches hashed simulation state and touches five domains, so its
ceiling is **four adversarial passes**. Five is the project absolute ceiling,
not this story's target.

The named surface is exhausted when a fresh reviewer has walked all G1
producers, G2's arithmetic and downstream writes, G3's form/hash/setter
boundary, G4's synthetic and EN/RU witnesses, and every twelve-aspect row. The
chain ends on the first pass after that surface is exhausted which finds no P
finding. W is added to the ledger and D is corrected in place; neither returns
the story. If four passes still find P, the story was cut too large: the working
slice lands only under the pipeline's ceiling rule and the remainder becomes a
separate defect story.
