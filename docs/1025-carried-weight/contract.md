# 1025 — carried weight

Contract. Written at the seat 2026-08-22, before the lane opens. Normative for scope.

## The result

A character's carried weight is a real quantity in this build, derived from the items he actually
holds, and the statistics card draws it as a `WEIGHT` row with one fractional digit. Overloading a
character slows him down.

Today the tree has no weight anywhere. `pkg/sim/carry.go` and `pkg/sim/equip.go` both say so in
their own doc comments: a container is a slot count and nothing else. The card has a vertical space
where the row belongs and draws nothing in it.

Pointable result: the `WEIGHT` row appears on all three statistics surfaces in `builds/current/`,
and a character loaded past his capacity moves at a lower speed.

## Why this story exists separately

Story `1022` composed the card and dropped the `WEIGHT` row, recording the drop as accepted. The
owner's ruling of 2026-08-21 is that he never accepted it (`pipeline/archive/owner-ruling-2026-08-21-stats-card.md`,
part 1). `DIV-209` carries that correction and names this story.

The row was not folded back into `1022` because reproducing the quantity needs a `Data.bin` column
the loader does not read, a sum over the inventory container, two fields on the actor, and a change
to what the simulation hashes. `1022` was at its own ceiling. The owner's split rule names this
case: split where one piece reaches hashed simulation state and the others do not.

## Behaviours

Five, B1 to B5. They are one causal chain: each is the input to the next, and the last is the only
one the owner can see.

**B1. The weight column is read.** `Data.bin`'s `Armors`, `Shields` and `Weapons` share one title
array, and slot 3 of it is `weight`. `pkg/data` exposes that value per item definition, in the
column's own units, alongside the fields it already reads from the same rows.

**B2. A carried item has a weight and a count, and a container has their sum.** Per-unit weight and
count are properties of the carried item, and the container's load is the sum of weight times count
over its elements. A stack of five arrows weighs five arrows.

**B3. An actor has a load and a capacity.** The load is the actor's own weight plus half the
container's sum, truncated toward zero, with a saturation arm at the top of the range. The capacity
is derived from the actor's body statistic. Both are actor state and both are hashed.

**B4. Overload is a speed penalty and never a refusal.** The penalty applies only when the load
reaches the capacity; below that it does nothing at all. Nothing anywhere refuses a pick-up, a
purchase or an equip because of weight. This is the only consumer of the load in the original, and
adding a second one would be this project inventing a rule.

**B5. The card draws the `WEIGHT` row.** One fractional digit, in the vertical space `1022` left,
between the last resistance row and the `XP` / `SIGHT` / `SPEED` block. It appears on every surface
that draws a character sheet, which is three call sites, all through `RenderCharacterPanel`.

## Claims

Read every one whole through the pinned reader before you build against it. Do not take this
section's summary as the fact; it is a pointer, not a quotation, and a digest is a selection.

```sh
cd research && go run ./tools/claim <ID>
```

- **`ITEM-LOAD-005`** (High) — the load derivation, its truncation, its saturation, and its single
  consumer. It also corrects a label another row uses for the same field.
- **`ITEM-STACK-003`** (High for the arithmetic; one Unknown on an unrelated field) — the per-unit
  weight field, the count beside it, the container's running sum, and the complete set of sites that
  maintain it.
- **`HERO-SPEED-008`** (High; one Unknown on an arm's reachability) — speed, and the overload
  penalty in full, including the floor. The Unknown is about a class arm, not about the penalty.
- **`HERO-SIGHT-007`** (High, amended and contested) — carry capacity and the same load arithmetic.
  Read its amendment history with it.
- **`DAT-SCHEMA-007`** — the published column list per collection, and the instrument that fixes the
  slot numbering.
- **`HERO-EQUIP-017`** (High, amended) — which slots the weapon fill reads and what each one is.
  This is what identifies the weight column by number.
- **`HERO-104`** (published by `EXP-0209`, at this pin) — the sheet's row formatter and its two
  different fractional conventions. **Read this one especially carefully: its answer for weight is
  negative.**

## What is decoded and what is not

Decoded: the column, the per-item fields, the container sum, the load arithmetic including its
truncation direction and its saturation, the capacity formula, and the penalty with its floor.

**Not decoded: how the drawn `WEIGHT` row gets its number.** `HERO-104` read the sheet's row
formatter whole and found that weight is drawn by neither of the two fractional sites it identified,
and that the character-screen fill routine resolves none of the actor's three weight, load and
capacity fields. So the bridge from the load field to the drawn row is missing, and this story
authors it.

Two things are known about the shape of that bridge and neither settles it. The formatter's second
fractional convention is decimal tenths, printing an integer's quotient and remainder by ten. And
the owner's screenshots show the row carrying one fractional digit with values that vary by
character.

**This is a divergence row, not a hold.** Owner directive outweighs research absence for what to
build. Author the divisor, state it in `spec.md` in the reader's own units, and open the row.

## The acceptance check, and the one thing you may not do with it

The owner's ruling records three observed values from the original, one per named character, in its
own table. **Derive the quantity first, from the claims, and write down what your derivation
produces before you look at his numbers.** Then compare.

If they agree, say so and name it as corroboration. If they disagree, **record the disagreement as a
divergence row and leave the derivation alone.** Do not tune a constant, a divisor, a rounding
direction or a column reading until the output matches a screenshot. A fitted constant reproduces
three numbers and is wrong about every other character, and nothing in this tree would ever catch
it.

One of the three observed values is zero. A character carrying nothing still has his own weight in
the load by `ITEM-LOAD-005`'s first assignment, so a zero on the drawn row is either a character
whose own weight is zero or a sign that the drawn row is not the load field. Say which your
derivation implies, and if you cannot tell, that is a divergence row and a question for research.

## Domains

Five, from `docs/DOMAINS.md`: **Assets** (1), **Sim Core** (2), **Party, Items & Heroes** (5),
**Client** (8), **Persistence** (9).

## The ceiling, stated before the lane opens

**Four adversarial passes**, because this reaches hashed simulation state. Five is absolute.
Reaching the ceiling is a scoping diagnosis, not a failure: land what works, open the remainder as
its own story, and say so in `closure.md`.

**Five behaviours and five domains, with hashed reach, and it does not split. Why:** every split
that has been considered leaves one half worthless. Cut B5 and the story computes a number nothing
shows, which is produced-but-unconsumed data and the first thing an adversarial reviewer looks for.
Cut B4 and the load is dead state that no rule reads. Cut B1 or B2 and there is nothing to sum. The
chain is short and it has one observable end. What keeps the size honest instead is the exclusion
list below, which is longer than the behaviour list.

## Out of scope

- **Any refusal on weight.** No pick-up, purchase, equip, trade or sack transfer is blocked by
  weight. `ITEM-LOAD-005` is explicit that the penalty is the only consumer.
- **The sheet's other fractional rows.** `SIGHT` already draws through its own convention; do not
  change it. `HERO-104` establishes the sheet uses more than one rule, so do not unify them.
- **The card's row order, fonts, rects and chrome.** Those are `1022`'s and `1027`'s, both landed.
  You are filling a space that exists, not composing a card.
- **The mission screen's right column.** That is `1023`, blocked on `1026`.
- **Encumbrance categories, stamina, fatigue** or any other invented mechanic. There is one number
  and one penalty.
- **Money.** `ITEM-LOAD-005` corrects an earlier label: the container's sum is item weight, and money
  is not an item and is not in it. Do not add the purse to the load.

## The twelve aspects

| Aspect | Expected |
|---|---|
| Data | PASS — B1, the weight column |
| Runtime state | PASS — B3, load and capacity on the actor |
| Simulation | PASS — B4, the speed penalty |
| Player input | N/A — no input path changes |
| AI | N/A — the AI issues the same actions; the penalty applies to it through movement without a decision change |
| UI/HUD | PASS — B5, the card row |
| Triggers/scripts | N/A unless a shipped script sets speed or weight; check and say which |
| Inventory/equipment | PASS — B2, the container sum, and every path that adds to or removes from a container |
| Persistence | PASS — the new actor fields are hashed and serialized, so the byte form version moves |
| Campaign/session | PASS — the load survives a mission boundary and a town return |
| Shipped content | PASS — sweep the shipped item tables and say what the weight column's real range is |
| Interactions | PASS — the speed penalty meets the existing movement and pathing rules |

## Expected divergence rows

`DIV-222`..`DIV-229` are reserved for this story. Allocate from that range; record any tail as
returned unused, and a returned id is never reissued.

At least these:

- The authored bridge from the load field to the drawn row: the divisor, the rounding direction and
  which field the row reads. Type `UNKNOWN` or `DEVIATION` by what you find.
- `DIV-209` should CLOSE at this landing if the row is drawn. Check its wording against what you
  shipped before moving it; do not move a row whose text no longer describes what happened.
- Anything the shipped-content sweep turns up that the claims do not cover.

## Gates

The repository chain, both repositories, plus **every** `scripts/check-*.sh` by glob in each. Run
the glob; the glob is the authority on how many there are.

This story changes what the screen draws AND what the simulation hashes, so the install-gated seat
gates are yours and not the seat's alone. From `<seat>`, on **both** roots:

- `pipeline/check-release-tests.sh` — the repository's own `go test` cannot reach these. A skip and
  a pass both print `ok`. Compare against the SELECTED count the script prints, not against a number
  written down anywhere.
- `pipeline/check-scenarios.sh` — both roots.
- `pipeline/check-milestone.sh` **before and after**. A speed change moves how far a unit gets per
  tick, so the escort census may well move, and if it does that is a fact your `closure.md` owes
  with its cause named. **The census drives `builds/current/missionrun.exe`, not your worktree**, so
  build the drive yourself and run it with the script's own arguments, or say plainly which binary
  you measured.

Read every exit code from `$?` on a captured variable, never through a pipe.

Deletion set `git diff --diff-filter=D --name-only origin/master HEAD` must be empty or explained.

## Witness rules that bind here

- A screen-drawing change extends that screen's own geometry test first, and the extension is
  mutation-proved **at the use site** — the composer's own call that reads the rectangle and draws
  with it — not at the rectangle-returning function's return statement. Check the extension fails
  against the pre-change code, then revert it byte-identical.
- Register any new geometry test in `pkg/ui/screenregistry.go`. A test outside `pkg/ui` needs its
  import path in the reference or the registry's own existence test rejects it.
- Expected values are independent of the constant or formula under test. Do not compute the expected
  load with the same expression the production code uses.
- Enumerate producers, not convenient rows. Every site that adds to or removes from a container is a
  producer of the sum, and the population must be stated with what your instrument cannot see.
