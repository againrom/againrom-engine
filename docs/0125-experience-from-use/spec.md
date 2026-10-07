# 0125 — experience from use: spec

**Intensity:** rigor Medium, autonomy High. **Terrain:** simulation state, its byte form, and
the readout above it.

A unit that lands blows must **earn experience for landing them**, and the panel must state
**every** skill slot a unit has, including the ones at zero. The two are one contract: a
number that grows is invisible while the row that would show it is suppressed.

## Vocabulary

- **Slot** — one of the **six** skill positions a character carries, numbered 0..5. Slot 0 is
  `General`; slots 1..5 are `Blade`, `Axe`, `Bludgen`, `Pike`, `Shooting`. Six is the data
  model's own count, not a choice made here. **The original's character sheet shows only
  1..5** — which is why five numbers is what a player expects — but slot 0 exists, holds a
  level, and is counted in the experience total, so this build states all six.
- **Slot experience** — a non-negative integer per slot. It is the state that grows.
- **The curve** `S(n)` — what one slot standing at level `n` accounts for. `S` is
  non-decreasing, `S(0) = 0`, and a level is confined to `[0, 100]`.
- **Level of a slot** — the largest `n` in `[0, 100]` with `S(n)` at most that slot's
  experience.
- **Total experience** — the sum of the six slots' experience.

## Functional requirements

### The state

**FR-1** An entity carries **six slot-experience integers**, one per slot, in slot order.
They are simulation state: hashed, carried in the byte form, and equal in every process that
replays the same world.

**FR-2** An entity carries the three further numbers a gain reads, none of which may be a
literal in this build's source: its own **Mind**, its own **experience value** (the units
table's column, read for the unit it is), and the **slot a gain it earns is credited to**.

**FR-3** An entity carries one flag saying whether its class **gains experience at all**. A
class that does not gain is not merely paid zero — no gain is computed for it.

### The gain

**FR-4** Paying experience is the **last act of a landed blow**, after the target's health has
been reduced, and it happens nowhere else.

**FR-5** Nothing is paid unless **all** of the following hold. Each is a separate refusal and
any one of them ends the payment:

1. the blow removed a **positive** amount of health;
2. the target was **alive before the blow** — a blow on an already-dead body pays nothing,
   whatever it removes;
3. the attacker's class **gains** (FR-3);
4. the attacker and the target **do not stand on the same owner slot**;
5. the relation the attacker holds toward the target does **not** have bit 1 set.

**FR-6** The amount before scaling is

```
raw = targetExperienceValue * removed / (2 * targetMaxHealth) + 1
```

with the division truncating toward zero, and every term evaluated wide enough not to wrap.
So a unit killed outright pays half its experience value, a unit killed over several blows
pays the same half split by the fraction of its **maximum** health each blow removed, and
**every** landed blow pays one on top of that — including one against a unit whose row states
no experience value.

**FR-7** The amount is then scaled by the attacker's Mind and truncated toward zero:

```
gain = raw * (4 * mind + 30) / 120
```

An attacker at Mind 0 therefore earns nothing from a single-point blow, and this is the
scaling's own arithmetic rather than a floor added here.

**FR-8** The gain is added to **exactly one** slot: the one FR-2 names. Nothing else on the
attacker changes and nothing at all on the target changes.

*Amended by `0135-skill-moves` FR-6 and FR-8.* The gain is **capped** at one level's worth of
the credited slot's current level, `S(n+1) - S(n)`, and a gain that carries the slot's
experience strictly past `S` of its current level also raises that **level** by one. So one
further thing on the attacker does change, and by at most one.

**FR-9** A blow that is refused for any reason in FR-5 draws no randomness and changes no
state, so the outcome of every other rule is unaffected by whether experience was paid.

### The level

**FR-10** A slot's level is derived from its experience by the definition in *Vocabulary*, and
by nothing else. The derivation is exact: no search tolerance, no rounding, no float compared
against a float. Experience above `S(100)` yields level 100.

*Amended by `0135-skill-moves` FR-1.* **A level is not derived from an experience.** It is its
own stored value, equal to `S`'s inverse only at creation and free of it from the first raise
on. This inverse survives as a definition and is still exact; nothing in a running mission
calls it.

**FR-11** The simulation itself never converts between a level and an experience. It carries
experience; a level is produced only where a level is read.

*Amended by `0135-skill-moves` FR-1, FR-8, FR-14.* **The simulation carries the level too**, and
compares an experience against `S` of a level at every award. What it never does is *produce* a
level from an experience: a level is read where one is read, and raised where one is raised.

### Where the numbers come from

**FR-12** A unit placed by a map is born with the experience value, Mind and gain slot its own
definition states. A placement that resolves to nothing takes the definition tier's defaults,
exactly as every other number it carries already does.

**FR-13** A party member is minted with **the experience his levels already account for** —
the sum over each slot of `S(level)` — so the total the panel states at the first tick is the
total his character sheet already implied, and the first blow moves it rather than resetting
it.

*Folded from hotfix `dbd5c89` — see `docs/hotfix/ARCHIVE.md#dbd5c89`.* A minted member may be
seeded from carried exact experience values as well as from levels, and the exact value is
carried because the level's inverse floors.

**FR-14** The slot a gain is credited to is the **skill of the weapon in hand** for a unit
whose class fights, and slot 0 for one whose class casts. This is an **authored** choice: the
original branches here on its class flag and which slot each branch credits has not been read.
It is one expression.

*Folded from hotfix `b55f111` — see `docs/hotfix/ARCHIVE.md#b55f111`.* The credited slot follows
the loadout: it is written by the same recompute that writes the combat numbers, through the
same door, all-or-nothing behind that door's own bounds guard.

*Amended by `0135-skill-moves` FR-7, which decodes what this clause authored.* The branch is
**not** "slot 0 for a caster". A unit whose class casts earns **nothing at all** from a blow,
and a unit whose class fights earns into the weapon in its hand and nothing when that is slot
0. So slot 0 is credited by no blow at all, and the authored collapse this clause records is
withdrawn.

### The readout

**FR-15** The unit panel states **all six** slot levels on one row, in slot order, separated by
spaces, **including every zero**. The row is stated whenever the panel knows a character for
the subject at all — it is never suppressed for being all zeros, and never truncated to five.

**FR-16** The total experience is stated beside them on the same row.

**FR-17** Both are read **off the simulation entity on the tick the readout is built**, not off
what the load computed. A blow landed on the previous tick is visible on this one.

## Acceptance criteria

**AC-1** A world in which one gaining attacker lands a blow on a unit it does not share an
owner slot with ends the tick with **more** experience in exactly one slot and the same
experience in the other five.

**AC-2** Replaying that world from its byte form reproduces the six numbers exactly, and the
world's digest changes when any one of them does.

**AC-3** An attacker whose class does not gain ends the tick with all six slots unchanged,
against the same target, the same seed and the same blow.

**AC-4** An attacker sharing an owner slot with its target ends the tick unchanged; so does one
whose relation to the target has bit 1 set.

**AC-5** A blow on a target that was already dead pays nothing.

**AC-6** Killing a unit outright pays `experienceValue / 2 + 1` before the Mind scaling; killing
it in `k` equal blows pays the same halved value plus `k`, up to the truncation each blow takes.
*Amended by `0135-skill-moves` FR-6:* and up to that story's cap, which bounds any one blow at
one level's worth of the credited slot.

**AC-7** The panel's skill row for a hero trained in exactly one slot states six numbers, five
of them zero, and states them for a hero trained in none.

**AC-8** The number the panel states rises across ticks in which the subject lands blows, and
holds still across ticks in which it does not.

## Properties

**P-1** The level derivation is the exact inverse of the curve: for every level `n` in
`[0, 100]`, the level of `S(n)` is `n`, and the level of `S(n) - 1` is below `n` whenever
`S(n)` is above `S(n-1)`.

**P-2** The integer form of FR-6 equals the truncated real-valued form
`experienceValue * 0.5 * removed / maxHealth + 1` over the whole range this build can reach.

**P-3** The integer form of FR-7 is **not** claimed equal to the real-valued form
`amount * (mind/30 + 0.25)`. `mind/30` is inexact in binary, so the two may differ by one at a
boundary. The divergence set is **measured**, not asserted, and the measurement is recorded.

**P-4** `pkg/sim` gains no floating-point identifier, literal or import, and no new import at
all, so the determinism wall is where it was.

**P-5** No new number is written into this build's source that the game reads from data. The
experience value comes from the units table's own column; the slot ordering is the data
model's; the constants that are the game's own compiled numbers are each named once.

## Out of scope, and cut deliberately

- **The loss path.** The original takes 10 % off every slot and inverts each back to a level.
  Nothing in this build reaches it, so it is cut and no field is reserved for it.
- **The type-range restriction.** The original also refuses a gain outside a type-id range
  that, for a person, restates *being a person*. FR-3's flag already carries that refusal, and
  this build has no type id on an entity to test, so the range is cut as redundant.
- **A class bit in the simulation.** The fighter/mage branch is resolved once, where the class
  is known, into FR-2's single slot number. The simulation carries no class.
  *Reversed by `0135-skill-moves` FR-7.* The branch is not resolvable once: the two arms refuse
  different awards rather than choosing different slots. The simulation now reads the class at
  the award, off the mana pool it already carries — still no new field.
- **Gold, sacks and the kill counter**, which the same routine also pays. Loot is another
  story's and is already built.
- **Re-deriving a blow's numbers from a raised level.** A level that rises here does not change
  damage or to-hit until something recomputes, and nothing in this story does.
  *Reversed by `0135-skill-moves` FR-12*, which recomputes for the character whose pack is open
  and states what it still cannot reach.

## Success criteria

**SC-1** Starting a mission, selecting a party member and attacking a hostile unit makes a
number on the panel rise, blow by blow, with no other action.

**SC-2** The panel states six skill numbers for that member from the first frame, zeros
included.
