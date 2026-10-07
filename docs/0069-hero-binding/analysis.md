# 0069 — what was measured before the contract was written

## The question

A map's `Target_Unit` parameter names one of three id spaces (`ALM-TRIG-046`). Below 10001 it is a
placed unit the map itself carries; 10001..11000 is a **hero ordinal** resolved against the live
player list; above that is a static name table. The binder takes the hero from its caller, because a
campaign map places no hero — the party already exists and the load only positions it.

The caller on the mission path never supplies one. So the question was whether that matters: how
many references land in the hero band, and whether anything a player can reach depends on them.

## The corpus, measured

Both lawful installs, every map: 38 in `en` (28 campaign + 10 loose), 34 in `ru` (28 + 6). The
campaign maps are byte-identical between the two installs, so the campaign figures below are one
measurement, not two agreeing ones.

| | en | ru |
|---|---|---|
| hero-band references | 196 | 196 |
| static-name-band references | 0 | 0 |
| references to a placed unit the map lacks | 0 | 0 |

Ordinal distribution, identical on both installs:

| ordinal | 10001 | 10002 | 10003 | 10004 | 10005 | 10006 |
|---|---|---|---|---|---|---|
| count | 135 | 22 | 11 | 4 | 13 | 11 |

24 of the 38 `en` maps carry at least one. Binding ordinal 1 alone takes the corpus's unresolved
count from **196 to 61** — the residue is every reference above ordinal 1.

## The tension in the tree's own comments, resolved

The compile's own note says *"nothing in the shipped corpus fails to resolve — 0 of 1304 buildable
nodes"*, while the band note says 196 references are hero-band and the resolver refuses every one of
them unless a hero is supplied. Measured rather than reasoned about: hero-band references **are**
unit-typed, **do** reach the resolver, and **do** land in the compile's unresolved list. On the
mission path mission 10 reports **8** unresolved and 0 discarded actions; supplying a hero takes
that to **0**.

The note is a bad paraphrase. `ALM-TRIG-046`'s figure counts all four reference kinds across the 38
maps and says that **with the bands honoured** none would be rejected **by the original's builder** —
which has a live player list. Compressed into "nothing in the shipped corpus fails to resolve" it
reads as a statement about *this* build, where it is false. The sentence is a defect in the code,
not merely an imprecision, because it is the reason nobody looked.

It is reproducible for mission 10 once a hero is supplied. It is **not** reproducible corpus-wide by
any compile this tree can make: 61 references remain, all in *campaign* maps. What the original
resolves them to is the claim's business and not measured here.

**Open, and deliberately assigned no meaning here:** what ordinals 2..6 name. The tree reads the
band as a subscript into the live *player* list; that the residue is exactly 61 and lives only in
single-player maps is consistent with a subscript into the *party* instead. This story does not
decide it and does not need it — mission 10 uses ordinal 1 only. It is a question, not a verdict.

## Mission 10, and why the story is on the critical path

Mission 10 carries 8 hero-band references, all ordinal 1: four conditions, all "distance from point
to unit", and four actions this build does not run.

Its **only** win trigger compares two clauses and, when both hold, forces the mission-complete
state. The first clause measures the distance from a fixed point to the hero. The second reads a
mission variable another trigger increments when a placed unit reaches its own point.

An unresolved reference measures nothing and **writes no register**. The hero clause's register is
therefore never written and holds zero — and zero satisfies the clause. The win does not become
unreachable; it becomes **falsely armed**, hanging entirely on the second clause.

One reading worth killing before it is formed again: the four conditions are **not** silently
measuring the distance to entity 0. The resolver returns "no entity" rather than id zero, and the
runtime's own arm returns before writing anything when a reference did not resolve. The damage is
the unwritten register, not a measurement of the wrong unit — a different mechanism reaching the
same class of defect, and the reason the fix is a binding rather than a guard.

Measured on mission 10's real compiled script, driving the world to its outcome:

| hero supplied | hero at the objective | outcome |
|---|---|---|
| no | **no** | **won** |
| no | yes | won |
| yes | no | undecided |
| yes | yes | won |

The first row is the defect. Today the mission announces victory with the hero sixty cells away.

Six of the corpus's 32 win triggers are hero-dependent, on maps 10, 30, 50, 61, 81 and 141. No VIP
(protect-this-unit) check anywhere in the corpus names the hero band.

Every arm the win chain needs is already implemented — the two distance checks, the variable read,
the constant preset, the increment and the mission-complete instant. Nothing undecoded blocks it.

## The ordering problem, and why it is not one

The hero's entity id is assigned inside the start, after the world is built, as the count of
entities the map's own placements produced. The compile happens before that and takes an
already-compiled program, so the binding cannot be read back off the built world without either
building it twice or mutating a validated script.

It does not have to be. The world builder emits exactly one entity per placed record, id equal to
index, skipping none; the start appends the party. So party member *i*'s entity id is the map's
placement count plus *i*, which is a property of **the map alone** and needs no world.

Checked on every map of both installs — 72 loads, 0 mismatches: the built world holds one entity per
placement, party member 0 receives the placement count as its id, and that entity stands on the
start's first cell.

## What the change costs

The compiled script is part of the world's canonical byte form and therefore of its digest. A
started mission's world digest moves — measured on mission 10, `0x1cfea58bd60be74d` to
`0xb46bf8105de39d2d`. The form itself does not move: the two fields whose values change are already
in it, at their existing widths.
