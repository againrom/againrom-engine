# Analysis — the player can order an attack

## What the tree already has, measured rather than assumed

- `pkg/sim/combat.go` resolves a blow end to end: a three-phase cycle with a jitter, a to-hit roll
  over `[-100,100]` with an auto-hit band, damage base plus spread, flat absorption, a saturating
  subtraction and a killing blow. `pkg/sim/step.go` defines `KindAttack = 4` and applies it.
- `grep -rn KindAttack --include=*.go .` outside `pkg/sim` returns **nothing**. The only producers
  are two test files. So no press, no key and no seam in this tree issues one, and no unit has ever
  attacked in a running game.
- `ui.MapOrder` is `func(entity uint32, x, y int)` — a cell, and only a cell. `ui.MapAffect` is
  `func(entity uint32, kill bool)`, the two debug blow keys, which apply damage directly and are
  not an order at all.
- The secondary press is the front-end's only order gesture (`decide`'s fourth branch). The primary
  button selects.

## What we did not know, and where it was answered

**How a click becomes an attack.** Assumed: by what it hit, probably with a hostility test.
`AI-CLICK-050` says otherwise — the CURSOR decides, the attack cursor is the only one producing two
different orders (attack when a unit is under it, a plain move to the cell when not), and the
click-time test is a runtime-CLASS test with **no diplomacy consulted at click time at all**.

**Whether one click is enough.** `AI-CLICK-051`/`AI-PANEL-053`: no — a panel arms a mode first, and
only if the selection's capability mask permits. Guard and aggressive issue on the spot; attack does
not.

**Whether being struck orders anything.** `AI-RETAL-056`: no. Retaliation sets one flag whose single
consumer makes the victim turn. Nothing here builds retaliation.

**Whether an attack order can survive being out of reach.** `AI-CMD-054`: the player's `0x19` sets
`actor+0x50 = 3` with `ord+0x0c = target` and `ord+0x14 = actor+0x12c`; arm 3 is the engage routine,
and `AI-GUARD-007` reads that routine as setting `order+8 = 5`, `order+0xc = target`,
`order+0x14 = actor+0x12c` — *"a target with a stop distance, which is the only construct in this
area that survives being farther away than reach."* So the original's attack order carries a
movement order in the same block, and our `KindAttack` arm — which ends the walk — cannot reach
anything the player did not first walk the unit next to.

## What is still open, and who is closing it

`EXP-0110` is running now over the three cursor gate globals `[L00627]`/`[L00670]`/
`[L00632]` (`AI-CURSOR-052` is capped **Medium** on them) and over the selection capability
mask `R0093` that `AI-PANEL-053` names but does not decode. Neither is invented here: the
gates are not modelled at all, and the mask is authored behind one predicate.

## What was looked at and not used

`pkg/ui/panel.go` and the readout layout were read to see what a command panel would cost. There is
no panel widget, no button art and no decoded button geometry in this tree, so the arming input is a
key on the map screen's existing letter register rather than a panel this story would have to invent
whole.

## Seat correction, mid-story — and its reconciliation

The orchestrator corrected its own brief mid-story, after `EXP-0110` landed on
research master past this story's pin. It was recorded here as a seat-supplied
correction with no id, because a frozen pin cannot cite one. **Master then bumped
the pin at the 0074 boundary and this branch was rebased onto it, so the
obligation is discharged rather than carried:** every clause below is now a row of
the submodule at its current pin, cited in `provenance.md`, and nothing in the
contract moved when the ids arrived.

What changed, and what it saved:

1. **There is no capability mask over the selection.** `AI-PANEL-053`'s clause is
   struck through in its own row and `AI-PANEL-060` reads the routine it pointed
   at: twenty instructions over one view flags word, touching the selection array
   **not at all**, returning `0xef` — every mode but cast. So attack is armable
   for any owned selection whatever classes it holds. A control greyed out by unit
   class would show the original withholding something it does not.
2. **The gate is OWNERSHIP**, and `AI-PANEL-061` names the bit: `view+0x144 & 0x4`
   is set when the **primary** selected object's player differs from the local
   one, and `R0093` returns zero on it. That is the comparison this story
   builds, on the primary present id.
3. **This is not the only producer.** `AI-SURFACE-063` counts exactly three input
   surfaces reaching the order-builder family. Nothing here builds the other two,
   and no artifact of this story says the attack order has one producer.
4. **The fan-out ceiling is 253** (`AI-FANOUT-064`) over a hash container rather
   than an array (`AI-SELECT-065`), which is why an overflow is an unpredictable
   subset. Examined and diverged from deliberately — spec FR-9, plan DD-14.

The three cursor gate globals `AI-CURSOR-052` was capped Medium on are closed too
(`AI-KEYMOD-059`: modifier-key latches). This story models none of them, which is
unchanged by their closing.
