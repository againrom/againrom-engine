# Overload penalty before the speed modifier

## Intent

An overloaded hero with a speed bonus or penalty walks and turns at the speed
the original derives. Before this story a native Human held its speed with
equipment modifiers and effects already added, and the load penalty and its
floor applied to that sum (`DIV-1421`). The two orders differ when the floor
binds or the sum is negative. Base: `228ad64a`, merged with main `f7f7dc24`.

## Authority

- `SAV-1116` (High for the derive stores): the Human derive subtracts
  load/capacity from its base speed when load reaches capacity, floors the
  result at 6, then adds the signed modifier speed word. A negative sum clears
  the modifier, not the speed.
- `MOVE-RATE-053` and `MOVE-106`: the mover rate and the turn rate take the
  low byte of the derived speed.
- `HERO-SPEED-008`: the penalty and the floor.
- `MOVE-RATE-029` and `TERR-MOVE-056`: walking reads the signed speed word and
  clamps it to the rate range.
- `DIV-036` (the effect speed floor) and `DIV-052` (sibling speed records)
  stay as they are.

## As built

### One derive

`rules.HumanSpeed(base, modifier, load, capacity)` (`pkg/rules/humanspeed.go`)
is the only Human speed derive. It returns the speed word and the modifier
the derive keeps. Three routes call it:

- `data.HumanState.derive`: original-SAV Humans and source-backed re-derives
  (`mapload.deriveSourceActor`), and town Humans.
- `sim.humanSpeedWord` (`pkg/sim/weight.go`): native Humans. `aloneSpeed`,
  `ActorLoadSnapshot.DisplaySpeed` and `deriveNativeHumanSpeed` read it.

### Native state

A native Humanoid's `Entity.Speed` stays the unencumbered sum, base plus
modifier; `Entity.SpeedModifier` is the signed modifier inside it. A Unit and
a source-backed Human hold zero: rearm and speed effects write it only on a
native Humanoid, and an actor that becomes source-backed (original-living
import, a source-class LOAD restore, the original-Human spawn route) drops it.
The base is `Speed - SpeedModifier`.

`deriveNativeHumanSpeed` (`pkg/sim/turnstate.go`) runs after every producer.
It derives the word, sets the turn rate to the word's low byte, and on a
negative sum clears the modifier (`Speed` drops by the cleared amount). A word
at or below zero is kept as a retained Human word, so the hero walks at the
rate floor and turns at the low byte, as a source Human does.

### Producers

| producer | route | writes |
|---|---|---|
| equipment and standing effects at rearm | `SetDerived` (`pkg/sim/rearm.go`) from the loadout's `Mod.Speed` (`data.Derived.SpeedModifier`, `pkg/game/rearm.go`) | `SpeedModifier` = loadout modifier plus the standing effects inside `Speed` |
| speed effects: Haste, Slow, item and potion speed arms | `effectLanding`, `EffectSpeed` (`pkg/sim/effect.go`) | adds the landed amount to `Speed` and `SpeedModifier` |
| inventory load | `finishLoadMutation` and the actor-load apply (`pkg/sim/actorload.go`) | re-derives only |
| spawn: party and placed persons | `mapload` party definition and `spawnBlock` (`pkg/mapload/start.go`, `fromalm.go`) | seeds `SpeedModifier` from the loadout; a party member's turn rate is the low byte of the derive over his spawn load (`spawnSpeedWord`) |
| source Human publication | `publishSource` (`pkg/sim/sourceactor.go`) | zero; the source derive owns the modifier |

### Persistence

- World byte form: an optional tail section, version 119, tag `SPM1`, one
  record (entity ID, modifier) per nonzero modifier. A world without one
  writes no section; corrupt records and a modifier on a non-native or
  non-Humanoid entity are refused.
- SAV: a native Human's speed word holds the derived word (`Entity.SpeedWord`);
  its mover byte holds the low byte. A word retained from an imported record
  does not replace the derive there; only a non-positive word the derive
  itself kept does. The modifier speed word holds each byte
  the native basis knows, else the live modifier (`Entity.SpeedModifierWord`).
  When either word differs from the live `Speed` or `SpeedModifier`, the
  engine-state leaf carries the operand `SpeedSplit` with both wires and both
  live values.
- LOAD of an engine SAV: the modifier comes from the modifier speed word. An
  operand applies each live value while its wire matches the record, so an
  edited record wins.
- An engine SAV written before this story has no operand. Its speed word is
  the unencumbered sum, which LOAD reads as `Speed`, and its modifier speed
  word is the modifier. This split is deterministic and is the state that SAV
  described.
- The actor-load snapshot carries `SpeedModifier` (zero when absent).

### Card

The character card shows the derived word (`humanSpeedStat`).

## Proof

- `TestHumanSpeedAppliesTheOverloadBeforeTheModifier`: zero load, below and at
  capacity, the floor binding before a positive modifier, the floor raising a
  slow base, a negative sum, a zero sum and the word wrap.
- `pkg/sim/overloadorder_test.go`: the floor binding with the step rate, zero
  load, Slow on an overloaded Human (word -9, turn 247, rate floor; expiry
  adds to the cleared modifier), a load change re-deriving the word, the byte
  form and its corruption refusals, and the LOAD split with and without the
  operand.
- `TestOverloadedNativeHumanSpeedSurvivesSAVE` (fixture): SAV word 11,
  modifier word 5, mover byte 11; cold LOAD equals the saved World; 40 ticks
  of walking keep equal hashes.
- `TestReleaseOverloadedHastedHeroMovesInTheClaimedOrder` (EN and RU
  installs): a native companion carrying the table's heaviest item past
  sixteen times his capacity, under Haste. His word is 6 plus the Haste
  magnitude, not the former order's value. SAVE writes the word, modifier and
  mover byte; cold LOAD keeps speed, modifier, word, turn rate and step rate,
  and 48 ticks of walking match. The same file without its engine-state leaf
  loads him as a source Human with the same word, and a dropped item
  re-derives the same word. That control writes the pack's running weight
  into the hero record (`DIV-2761`) and a mana period into the fixture
  caster's record, which an original profile requires.
- Before this story the same hero spawned with turn rate 15 against a
  derived word of 6: party spawn took the low byte of the unencumbered
  speed.

## Ledger

- `DIV-1421` closed.
- `DIV-1660` closed: the mover byte follows the derived word under a speed
  effect.
- `DIV-2760` opened: the modifier after a cleared negative sum, and the
  prelude.
- `DIV-2761` opened: a native pack's running weight omits table-weighted
  items in the SAV.

## Open debt

- `DIV-2760`: a native rearm rebuilds the modifier from items and standing
  effects, so a modifier cleared by a negative sum returns at the next rearm;
  the live modifier does not apply the above-24 prelude clear.
- `DIV-2761`: the SAV running weight of a native pack holding
  table-weighted items is 0; a reload without the engine-state leaf derives a
  lighter load.
- Placed persons take their spawn turn rate from the unencumbered speed; a
  placement overloaded at spawn is not covered (`DIV-2568`).
- `DIV-036`: an effect may not take the summed speed below 1.
