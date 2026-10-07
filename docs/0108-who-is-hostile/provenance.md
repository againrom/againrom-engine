# Provenance — 0108

Research pin for this story: submodule at `53f8bb7`, unchanged for its whole length.

## The rows the contract rests on

| Claim | Confidence | What it carries here |
|---|---|---|
| `AI-DIPLO-004` | High | The relation is a 50×50 byte matrix and bit 0 is what acquisition tests. Already built; cited for the bit. |
| `AI-DIPLO-005` | High / Medium on the value space | The map's own store, the 1-based column arithmetic and the forced diagonal — already built by an earlier story, cited here because FR-6 leans on the diagonal being 2. |
| `AI-DIPLO-082` | High per writer, **Medium** that six is all of them | The writer set of the matrix after construction. This is the row that made the story: we hold one of the six. |
| `AI-RETAL-056` | High | The hook. Being struck sets a turn flag, records the attacker's cell, clears the memory clock, and calls the flip — and issues no order and produces no target. FR-8's "notifies nothing" starts here. |
| `HERO-AGGRO-028` | High for the hook and the arithmetic; ✖ on one clause | The flip writes `[attacker][victim]` **and** the mirror, and sets bit 0 only where the low two bits are clear. FR-4, FR-5. |
| `AI-DIPLO-084` | High | The gate is `TEST AL,0x3` before `OR AL,0x1`, **separately gated for each direction**, so a pair can flip one way and not the other. This is FR-4's "neither reads the other" and it is the reason the two directions are not one test. |
| `AI-DIPLO-086` | High for the five bare writers, Medium for the reach | A change is observed, never propagated: no notify, no re-scan, no order invalidation, and a consumer sees it when it next rebuilds its candidate list. FR-8. |
| `AI-GROUPSEE-068` | High | That the candidate list is rebuilt and re-filtered through the diplomacy row on every group evaluation — which is what makes FR-8's "at its next decision" a mechanism rather than a hope. |

**Confidence weighed rather than quoted.** Every clause the contract turns into a rule is High and
is a transcribed instruction: the two indices, the `TEST`, the `OR`, the per-direction gates, the
guard on both actors having a player. The Mediums attached to these rows are about *completeness* —
whether six writers are all of them, whether the value space is `{0,1,2}` — and nothing here
depends on either being closed. A seventh writer would be a further story; a cell with a bit above
1 is already carried untouched.

**One retraction, read.** `HERO-AGGRO-028`'s "one blow on a neutral turns his whole faction
hostile, symmetrically and permanently" is retracted as a *consequence* while its instructions
stand. What was wrong is exactly what FR-5 encodes: bit 1 stops the flip, and 3 of the campaign's
198 ordered pairs carry it. The contract is written from the amended row, not the headline.

## What is ours by choice

- **FR-1's boundary — the flip fires when the swing connects, before absorption.** The published
  hook sits in the tail of the routine that *applies* damage, so the swing has already been decided
  to connect when it runs; no row read here separates "entered the routine" from "removed health",
  and no row says whether a miss reaches it at all. The reading chosen is the one in which *being
  struck* and *the relation changed* cannot come apart — a victim whose armour ate the blow was
  still struck. It is a choice and it is disclosed below.
- **The name and shape of the accessor.** The gate is written once, on the relation type, because
  a second copy of `low two bits clear` is exactly the kind of thing that survives one story and
  not two.
- **The census in the acceptance.** No row publishes a per-map split of flippable against locked
  cells; the numbers in `analysis.md` are this tree's own reading of the shipped files.

## What is open

- **D-1** — whether a **missed** swing reaches the hook. Unpublished. FR-2 says it does not.
- **D-2** — whether a blow whose damage absorption reduces to nothing reaches it. Unpublished.
  FR-1 says it does. Both would be settled by reading the damage routine's own entry conditions.
- **D-3** — bits above bit 1. Carried, never interpreted, never written. The loader's own mask
  admits a third bit that no shipped map uses; nothing here narrows or widens that.
- **D-4** — the five writers not built: the two mission joins, session command `0x45`, the script's
  action opcode 10, and the spell-cast entry point that reaches the same flip and additionally
  re-scans in the same instruction stream. This build has no spells, so the last is out of reach
  rather than deferred.
- **D-5** — `AI-FILTER-001` drops a candidate the diplomacy test would keep when the candidate
  carries an invisibility bit, unless a group member is close enough. No row read here names a
  writer of that bit. **A question for research, not a gap to work around.**

## Corrections this story owes the tree

Two comments in shipped source were true when written and are made false by this contract. Both
were read before they were believed, and both are the tree asserting what the rest of the tree
does — a premise, not documentation.

- `pkg/sim/relations.go` says of bits 1 and 2 that "both belong to writers this story does not
  implement". One of them is now implemented, and bit 1 acquires a live consequence.
- `pkg/sim/world.go`'s `Relations()` says the copy-on-read "keeps the type having no writer to
  find". The type now has exactly one writer inside the world, and the copy-on-read is what keeps
  it the only one.

## What was measured rather than cited

Against **both** installed roots, through this tree's own loader and reader, at the pin above:
mission 10's five-record roster and its effective matrix; the interceptor's slot, group, class,
health and scan range; the campaign-wide split of ordered off-diagonal cells; the action-opcode
census over every map either root can read; and the milestone's baseline trace. The figures are in
`analysis.md`. No probe used for them is in any commit.
