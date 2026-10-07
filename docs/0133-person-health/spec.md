# spec — 0133 a placed person's health is derived, not read

A map places a person and he arrives at the health his definition row's health column names. That
column is not his health: his health maximum is a number the engine derives from his statistics and
his training, and the column is overwritten before anything reads it. This story derives it.

**Intensity:** spec-anchored / static. **Terrain:** brownfield in the derived-stat graph, in the
placement arm and in the definition-table report; nothing here is new code beside existing code.

## Terms

* **Person** — a placement resolving on one of the three rungs that search the `Humans` collection,
  as against a **creature**, which resolves on the collection of unit definitions.
* **Health column** — the `Humans` row cell this tree streams into a definition's health maximum.
* **Derived maximum** — what the derived-stat graph yields for a character from his four statistics
  and six skill levels alone. Its health arm has three terms in order: the capped Body times the
  class multiplier; a logarithmic term in the summed skill experience, itself scaled by that
  multiplier; and an exponential growth term in the capped Body. Each is truncated toward zero and
  narrowed to a word before the next reads it. The tree carries this arm already and this story does
  not change its arithmetic.
* **Party member** — a character the player brings to a mission. He is not a placement: no map
  places him, he is minted by a separate path, and he may carry no definition row at all.
* **Class multiplier** — the factor the health arm of that graph applies, 2 for a row stating no
  mana column and 1 otherwise.
* **Capped statistic** — a statistic after the graph's own ceiling is applied to it.
* **Column flag** — the mark a character's profile carries saying that the row he came from states a
  health maximum at all. A generated character with no row carries it clear.

## 1. Problem / current behaviour

A person's health pair — his live health and his maximum, which are equal at placement — is the
health column, carried whole. On both lawful roots the three mission-20 placements resolving to the
`NPC14_1` row are minted at 20 and 20; the same row is placed three times in mission 10 and is minted
there at 20 as well.

The health arm of the derived-stat graph is gated on the **column flag**: a character whose profile
states no health column skips the arm's first term and keeps the other two, so he derives a small
positive maximum out of the experience and growth terms alone — 2 for the party hero this tree ships.
The party mint therefore carries a second condition, on the same flag, to keep such a member from
being born at that 2.

The definition-table report prints one line per placement carrying the health maximum the world was
built with, and does not say which roster slot owns the placement. It reports nothing about the
person collection as a whole.

A `Humans` row too short to stream its slots is already refused where a search reaches it, and the
refusal already takes the whole world with it. AC-8 and P-2 preserve that; they ask for no new
validation, only for evidence that it still holds once the health beside it is derived.

## 2. Functional requirements

* **FR-1** A person's health maximum MUST be the derived maximum of the definition row he resolved
  to. It MUST NOT be that row's health column.
* **FR-2** There MUST be exactly one implementation of the derivation in the tree: the one a
  generated character already goes through. A person's maximum and a generated character's MUST NOT
  come from two expressions.
* **FR-3** A person's live health at placement MUST equal his derived maximum.
* **FR-4** The health arm of the derived-stat graph MUST run all three of its terms when the product
  of the capped Body and the class multiplier is nonzero, and MUST yield a maximum of 0 otherwise.
  **Inside the graph** the column flag MUST NOT decide it, at either end. What a caller does with the
  number afterwards is FR-8's business, not this one's.
* **FR-5** A row stating no mana column MUST take the doubled class multiplier on that arm; a row
  stating one MUST take 1.
* **FR-6** The difficulty setting MUST NOT reach a person's health.
* **FR-7** A creature placement and a placement resolving to nothing MUST keep the health each has
  today — the creature its own definition's maximum at the chosen difficulty, the unresolved one the
  loader's own provisional value.
* **FR-8** A party member whose profile carries the column flag clear MUST still be minted at the
  same provisional health FR-7 gives an unresolved placement, rather than at a derived number. This
  is a decision the mint makes about the number, not a condition inside the graph.
* **FR-9** The definition-table report MUST name, per placement, the roster slot that owns it.
* **FR-10** The definition-table report MUST census the whole person collection: the sum of the
  health columns, the sum of the derived maxima, how many rows derive above, below and equal to
  their own column, and the row standing in the largest ratio of derived to column. A row whose
  column is 0 has no ratio: it MUST be excluded from that comparison and counted separately, so the
  report never divides by it and never lets one such row stand for the collection.
* **FR-11** No field is added to a simulation entity, the serialized byte form does not change, and
  its version constant is not spent.

## 3. Acceptance criteria

| # | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a person definition with a nonzero Body and a health column unequal to its derived maximum | a world is built from a map placing it | the entity's health and maximum are both the derived maximum |
| AC-2 | unit | a definition whose capped Body is 0 | its maximum is derived | the maximum is 0 |
| AC-3 | unit | two profiles differing only in the column flag, over one character | each is derived | both yield the same maximum |
| AC-4 | unit | a definition with a nonzero Body and a health column of 0 | its maximum is derived | the maximum is positive |
| AC-5 | unit | one person placement | worlds are built at each of the three difficulties | the health is the same number in all three |
| AC-6 | unit | a creature placement and a placement resolving to nothing | a world is built | each carries the health FR-7 names for it |
| AC-7 | unit | a party member carrying no base row | a mission is started | his health is the party's own starting health |
| AC-8 | unit | a `Humans` row the placement's own search reaches, too short to stream its slots | a world is built from a map placing it | the build is refused, the world is not returned, and the error names the row |
| AC-9 | developer-run | either lawful root | the definition-table report is run over mission 20 | each `NPC14_1` placement reads its derived maximum, and the report names the roster slot owning it |
| AC-10 | developer-run | either lawful root | the collection census is run | it prints the two sums, the three counts and the largest-ratio row |
| AC-11 | unit | one person placement | its maximum is derived twice | the two numbers are equal |

**Error cases:** AC-8 is the error case. A row that cannot be streamed is the one input on this path
that has no answer, and it already has none.

## 4. Derived properties

* **P-1** *(invariant)* For any person definition, the maximum the world mints for a placement of it
  equals the maximum the derived-stat graph yields for it. There is one number, not two that agree.
* **P-2** *(negative-invariant)* For any `Humans` row a placement's search reaches and the definition
  builder then refuses, no entity is minted from that placement and no world is returned. A row so
  short that the search reaches nothing at all is **no match**, not a refusal, and takes FR-7's
  provisional value like any other placement that resolves to nothing.
* **P-3** *(idempotence)* For any definition, deriving its maximum twice yields the same number: the
  derivation reads nothing it writes.
* **P-4** *(completeness)* For any map, every placement resolving on the person arm carries a derived
  maximum, and none carries its row's column.

## 5. I/O examples

The definition-table report's per-placement line gains one field, the owning roster slot, and keeps
the fields it has:

```
   49 key 0x000b/0x0000  arm server-id entry   58  owner  5  healthMax   120  speed   16  domain ground
```

A placement the map gives no owner carries slot 0, which names no roster entry; the field prints it
as the 0 it is rather than as a blank.

The collection census is a block of its own:

```
humans health: N row(s), column sum N, derived sum N
  derived above column N, below N, equal N; zero column N, zero derived N
  largest ratio: <row name> derived N against column N
```

## 6. Constraints

* The derived maximum crosses into hashed simulation state. It is computed outside the determinism
  wall, as the graph already is, and reaches the wall as an integer.
* Tests are synthetic and read no install. The corpus evidence AC-9 and AC-10 ask for comes from a
  developer-run tool.
* The health arm's gate — the one choice this contract makes that a reader could reasonably make
  differently:

| | Gate | Observable consequence |
|---|---|---|
| A | the column flag decides the arm's first term (today) | a row's column changes a maximum derived from statistics; a character with no row derives 2 |
| B | the product of capped Body and the class multiplier decides the whole arm | **chosen** — a maximum is a function of the character alone; a character with no Body has none |
| C | no gate | a character with no Body derives a positive maximum from the growth term alone |

## 7. Out of scope

* **A person's mana maximum**, which stays the row's mana column. The same routine derives it and
  this story does not, so a person's two pools come from two different places until one does.
* **The armour fold** — what a worn piece contributes to protection, absorption and the health
  addend. A person wears his row already; nothing he wears moves any number here.
* **The placement record's per-actor statistic overrides**, which this tree's map decode does not
  carry at all.
* **Which archetype the class multiplier names.** FR-5 fixes the condition and the direction; it
  does not name the two halves.
* **Mercenary hiring and any transfer of a placement's owner**, mission 20's win condition, and the
  campaign chain.

## Verification mapping

| AC | How |
|---|---|
| AC-1…AC-8, AC-11 | CI-automatable, synthetic fixtures |
| AC-9, AC-10 | developer-run against both lawful roots; output recorded |

## Gate check

FR-1 → AC-1, P-1, P-4 · FR-2 → AC-1, P-1 · FR-3 → AC-1 · FR-4 → AC-2, AC-3, AC-4 · FR-5 → AC-1 ·
FR-6 → AC-5 · FR-7 → AC-6 · FR-8 → AC-7 · FR-9 → AC-9 · FR-10 → AC-10 · FR-11 → AC-1 (the byte form
is unchanged, so every existing form test stands unedited) · P-2 → AC-8 · P-3 → AC-11.
