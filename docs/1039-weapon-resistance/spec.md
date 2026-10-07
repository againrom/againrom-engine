# Story `1039` - weapon damage-kind resistance

The behaviour this story ships. This file is canonical as-built; `contract.md`
records the premise measured before implementation.

Base `e3466269`, research pin `d7ee0c6`. Simulation form version 59 is reserved
to this story and is the form described below.

## 1. The state

Every simulation entity carries two parts of one physical-damage selector:

- `XPSlot` is the attacker's active kind. Zero means no weapon-kind selection;
  1 through 5 mean Blade, Axe, Bludgeon, Pike and Shooting.
- `Resistance` is the target's five unsigned bytes in that same order.

The implementation reuses `XPSlot` rather than adding a second kind byte.
`HERO-DAMAGE-022` indexes resistance by the active-skill slot, and the existing
field already carries that slot through construction, re-arm, serialization and
hashing for physical attribution and experience. Two canonical fields for the
same selection could disagree after an equipment change.

Only 0 through 5 are legal canonical slots. The shared world-construction
boundary refuses another value on both an initial Entity and the retained
Control Spirit template; decode and `SetCombat` make the same refusal without a
partial write. Data producers normalize a bare weapon, every ranged row, and a
melee attack type outside 1 through 5 to zero. The meaning of unsupported melee
types remains `DIV-364`; the normalization prevents an index outside the
researched six-byte family.

The data tier keeps resistance as `[5]int32` while modifiers are being folded.
`data.DamageKindResistance` converts that array once, at the simulation
boundary, to `[5]uint8`. This is low-byte narrowing, not a clamp: 256 becomes
0, -1 becomes 255, and both signed extremes retain their low byte. That is the
modulo-256 store established by `HERO-FOLD-033` and `UNIT-WIDTH-016`.

## 2. Every production producer

The five bytes and active kind reach simulation through all actor-minting paths:

- a Units placement takes `UnitDef.Resistance`; its resolved equipped weapon is
  folded into `UnitDef.SkillSlot` before `UnitDef.Combat` is consumed;
- a Humans placement takes the derived family and its resolved weapon kind;
- an unresolved placement takes the defaults through the same spawn block;
- a generated party member takes the current hero recompute;
- Catapult and Ballista party mercenaries take the Units placement block;
- the Control Spirit ghost template takes the Ghost Units row and the kind of
  its resolved equipped weapon, and every raised ghost copies both;
- an ordinary live equipment change passes the recomputed family and kind
  through `game.Rearm` and `sim.SetCombat` atomically.

`UNIT-COMBAT-015` is the source for the Units-row values. The array had already
been decoded and shown on the character sheet before this story; the missing
part was the production consumer recorded by `DIV-361`.

## 3. One physical resolution

Every fighter attack released by the input path or by AI reaches
`sim.resolveBlow`; there is no second physical-damage estimate. The resolver
now performs these operations in order:

1. Draw the existing physical spread and apply the existing attached max/min
   damage overrides.
2. Run the existing hit test. `AlwaysHits` bypasses it.
3. For an ordinary hit, subtract the target's current `Absorption` and floor
   the component at zero. An `AlwaysHits` hit skips absorption, as
   `HERO-AUTOHIT-031` establishes; this closes `DIV-362`.
4. If the attacker slot is 1 through 5, select
   `target.Resistance[slot-1]` and compute

   ```text
   trunc(damage * (100 - resistance) / 100 + 0.75)
   ```

   Slot zero bypasses this step.
5. If the complete physical result is not positive, stop before health,
   attribution, ordinary weapon-rider and experience writes. Otherwise subtract
   the reduced result from health, retaining the existing `minHP` clamp and all
   existing downstream ordering.

The production expression is the exact signed-integer equivalent
`(damage*(100-resistance)+75)/100`. Go's signed division truncates toward zero,
as the researched `ftol` does. Every source combat field is `int32` and the
resistance is one byte; the intermediate is `int64`, including the maximum
reachable base plus spread minus minimum absorption, so the multiplication
cannot overflow.

No range clamp is applied to resistance. A value of 100 produces zero. Values
above 100 may produce a negative component, exactly as `HERO-CLAMP-030`
establishes; with the only physical component this build carries, the final
non-positive gate removes no health. When the absent second and third damage
components are implemented, their addition must precede that final gate rather
than treating this story's present one-component equivalence as a new rule.

Resistance consumes no random number. Two worlds that differ only in the
target's five bytes stay on the same hit/miss and damage draws; the release
witness asserts the first successful blow occurs on the same tick.

## 4. Existing damage consequences

The reduced value is the value handed to the existing health, attribution and
physical-XP paths. A positive component records its fighter source and computes
the experience award from the reduced health removal, not from the damage roll
before resistance. A resistance-cancelled component writes no physical kill
attribution and pays no experience. It reaches the weapon-rider gate in the same
state as a fully absorbed hit: ordinary riders are suppressed, while any
rider's existing special refusal/fallback semantics remain owned by that rider.

Relationships still flip when a blow lands, before absorption and resistance,
because this story does not move the existing landed-hit boundary. Attack
cadence, target selection, pursuit, death clearing and AI decisions continue to
observe the canonical health that the resolver writes.

Armour and shields keep their decoded interaction. They add Defence and
Absorption (`HERO-ARMOUR-018`, `HERO-EQUIP-017`, `ITEM-ARMFOLD-033`) and do not
add either five-value protection family. A timed Absorption effect is included
because `SetCombat` preserves its live delta before the hit; resistance then
runs after that resulting absorption value.

## 5. Ranged and spell boundaries

A ranged weapon clears `XPSlot` to zero and therefore bypasses weapon-kind
resistance. Its missing second damage component, ranged to-hit source and
elemental reduction are a separate vertical slice recorded by `DIV-363` and
`HERO-DMG2-029`; this story does not fabricate them.

Attack type 5 is still a supported active kind when it occurs on the melee arm,
even at long reach, and selects Shooting resistance. Reach does not decide
whether a row is ranged; the weapon row's own ranged predicate does.

Book spells, area effects, Drain Life and weapon-borne spell effects continue
to use their spell school's elemental `Protection`. They never consult the new
array (`MAGIC-RESIST-006`). Tests place 255 in all five weapon resistances while
checking an elemental spell result, so the negative interaction is observable.

The decoded `EquipMod.Resistance` addition is carried if a caller supplies it,
but no production item-effect decoder populates it. Ordinary armour is correctly
neutral; the missing magic-item/effect producer remains `DIV-365`.

## 6. Persistence and replay

Form version 59 grows every entity record from 277 to 282 bytes. The five new
bytes are appended at offsets `+277` through `+281`, in Blade through Shooting
order. No earlier entity offset and no later section layout is changed.

Encoding copies all five bytes whole; decoding reconstructs them whole.
`World.Hash` hashes the encoded form, so a resistance change is a replay-state
change. Independent byte pins peel the five-byte tail back to the exact
version-58 hashes, and nonzero offset fixtures prove every byte belongs to the
right record. The version-free pin test name remains valid across future form
bumps.

`UpgradeSaveForm` accepts its existing readable versions through 58. Every old
entity record is widened with five zero bytes. Version 58 keeps its structure
section and every other current-width section; only its entity records grow.
The upgrade returns a player-facing `SaveFormWeaponResistance` disclosure,
because the format layer has neither definition nor loadout inputs from which
to reconstruct the lost state. A current version-59 form passes through with no
notice.

The ghost template itself remains mission-side definition state, as it was
before this story. All ten public world constructors funnel through one
`newWorld` boundary; only `NewSummoningWorld` and `NewStructuredWorld` can pass
a nonzero template, and both validate its selector there. Control Spirit is the
only runtime population append. A raised ghost becomes an ordinary entity and
its copied resistance and kind are then serialized and hashed; a real
encode/decode round-trip precedes the physical-blow witness.

The older template-domain boundary is not repaired by this selector story.
`DIV-366` records that a raisable template with an undefined movement domain can
still become an Entity that encoding writes and decoding refuses.

## 7. Presentation

The character sheet keeps the established Blade, Axe, Bludgeon, Pike, Shooting
order. `entityDraws` now overlays `ui.UnitCharacter.Resistance` from the live
entity beside the existing elemental-protection overlay. A re-armed or resumed
actor therefore shows the same canonical bytes the next physical blow will
consume, rather than the mission-start cache.

No new player command, script opcode or HUD control is introduced. Existing
move/attack input and AI ownership decide that a blow is attempted; this story
changes only the shared resolution after it lands.

## 8. Open fidelity work

- `DIV-363`: the ranged second damage component and its elemental path.
- `DIV-364`: ROM1 behaviour for unsupported melee attack types.
- `DIV-365`: production magic-item/effect contributions to resistance.
- `DIV-366`: the pre-existing invalid GhostTemplate movement-domain boundary.

`DIV-361` and `DIV-362` close with this story. Reserved ids `DIV-367` and
`DIV-368` are unused.
