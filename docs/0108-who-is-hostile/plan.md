# Plan — 0108

## The shape of the change

Two new pieces of code and nothing else. A gate on the relation type, and one call from the blow.

`pkg/sim/relations.go` gains `turnHostile(from, to uint32)`, a pointer method beside `Set`.
`pkg/sim/combat.go`'s `resolveBlow` gains one line, and `pkg/sim/world.go` gains the entity-to-slot
bridge that line calls. No file outside `pkg/sim` is opened.

## FR-1, FR-2, FR-3 — where the call goes

`resolveBlow` already has the shape the contract wants and the whole of FR-1..FR-3 is *which
statement the call sits after*:

```
if !inReach(*a, *t) || t.MaxHP <= 0 { return }     <- FR-2, out of reach: nothing
dmg  := base + uniform(spread)
roll := uniform(span) + floor
if !(alwaysHits || toHit+roll > defence || roll >= auto) { return }   <- FR-2, a miss: nothing
                                                    <- THE CALL GOES HERE (FR-1)
dmg -= absorption
if dmg <= 0 { return }                              <- FR-1: already flipped
```

Placing it on that line and not one lower is the whole of FR-1's "before absorption", and placing
it below the to-hit test is the whole of FR-2. FR-3 needs no code at all: `resolveBlow` tests
`MaxHP`, never `HP`, so a victim below 1 health reaches the same line.

**The two draws are above the call and stay above it.** Nothing added here consumes a position in
the stream, which is P-1 and is what keeps every landed test's rolls where they are.

## FR-4, FR-5, FR-7 — the gate

`Relations.turnHostile(from, to)` is the whole rule: find the cell; if it has one and its low two
bits are both clear, set bit 0; otherwise return having written nothing. A slot the matrix does not
hold falls out of `relationIndex` returning false, which is FR-7 and is `Set`'s own behaviour, so
there is no second out-of-range rule to keep in agreement (P-4).

FR-4 is **two calls at the bridge**, not one symmetric method: `[a][t]` then `[t][a]`. Each call
re-reads its own cell and neither is conditioned on the other's outcome.

## FR-6 — the diagonal, and the world named no relation

Nothing is written for FR-6's first arm. Both calls name the same cell, the gate reads 2 there on
any loaded map, and the pair is declined. Its second arm is a consequence to *test*, not to
prevent: a world constructed with no relation has an all-zero matrix, so the same blow sets the
diagonal. **This is the one place the change can reach a landed test.** Any test where two
same-owner entities exchange blows in a world built without a relation now has a self-hostile slot
afterwards, and if that test also runs an engagement it may move. The task hunts those explicitly
rather than waiting for the suite to find them.

## FR-8 — the flip is bare

Nothing is added. The bridge writes the matrix and returns; no engagement is re-issued, no order
touched, no candidate list rebuilt. The next group decision rebuilds its list from a freshly
cleared stamp and re-filters it through the relation, which is where the change becomes visible.
This is a property to state in a comment at the call, because "we did not call anything here" is
invisible to a later reader and is the kind of absence that gets helpfully repaired.

## FR-9 — the boundary

`pkg/sim/binary.go` is not opened. The relation is already the byte form's last block, so a flipped
relation encodes and hashes with no new field and no version change.

## DD-1 — the gate is written once, on the type

The `low two bits clear` test lives in `turnHostile` and nowhere else. The alternative — read
through `Byte`, test at the call site, write through `Set` — spreads one rule over two files and
makes `Set` the writer of a value it did not decide. One method also keeps P-4 true for free:
nothing outside `relations.go` indexes the matrix.

## DD-2 — one call site, and it is in `resolveBlow`

The flip is fired from the blow and from nothing else. Not from `advanceAttack`, which decides
whether a swing happens; not from `orderAttack`, which is an order and not a strike. A second call
site is how "being struck" and "the relation changed" come apart.

## DD-3 — no same-slot special case

Written down because its absence is deliberate. The game's flip has no self-test either; what
protects it is the loader forcing the diagonal to 2, which this tree already does. Adding a guard
here would be a rule the original does not have, and it would hide FR-6's second arm rather than
disclose it.

## DD-4 — two calls, not a symmetric helper

A `makeMutuallyHostile(a, b)` that wrote both cells under one test would be wrong on exactly the
pairs that matter: a pair already hostile one way and clear the other. The directions are gated
separately in the game and separately here.

## DD-5 — the byte form is not touched

Version 24 stays. **This story's allocated form version 25 is not spent** and is left free for the
story that adds the remembered attacker, which does need per-entity fields.

## DD-6 — what is not built, and why each

The remembered attacker and its twenty-tick clock: two per-entity fields, a version, and a notion
of human participation this world lacks — a story, not a clause. The turn-to-face flag: its only
consumer is `rand()`-gated and was cut on determinism grounds two stories ago. The script's action
opcode 10: measured absent from every map either root ships, so it changes nothing that exists and
belongs to customisation. The mission joins and session command: no consumer here. Any notify: not
simulation. Each of these is named in `spec.md`'s scope so a reader need not infer it from silence.

## The two comments this makes false

`relations.go`'s bit-1 paragraph says its writers are unimplemented; `world.go`'s `Relations()`
says the type has no writer to find. Both are corrected in the task that makes them false, in the
voice of the ones already there — a comment asserting what the rest of the tree does is a premise
and goes stale silently.

## Verification steps

**SC-1** — the local gate on a clean tree: build, vet, `gofmt`, `go test -count=1 -trimpath ./...`,
then `check-no-game-assets.sh`, `check-doc-budget.sh`, `check-sdd-audit.sh`. Exit codes captured
directly, not through a pipe.

**SC-2** — the campaign census over both installed roots (AC-1), through a throwaway reader that is
in no commit, reporting the three counts and requiring the roots to agree.

**SC-3** — the milestone on both roots, before and after the last commit (AC-8), with the trace
quoted whichever way it comes out. The recorded prediction is that it does not move.

**SC-4** — a mutation battery: drop the bit-1 half of the gate; make it an assignment rather than
an `OR`; write one direction only; move the call below the absorption test; remove the slot-0
guard. Each must be killed by a named test.

**SC-5** — the form: version and record lengths unchanged, a world encoded before the change still
decodes, and a digest that **differs** across a connecting blow on a flippable pair (AC-7).

## Traceability

| FR | built by | decided by | witnessed by |
|---|---|---|---|
| FR-1, FR-2, FR-3 | T2 | DD-2 | AC-5, AC-6, SC-4 |
| FR-4, FR-5, FR-7 | T1 | DD-1, DD-4 | AC-2, AC-3, AC-4, SC-4 |
| FR-6 | T1, T2 | DD-3 | AC-5 |
| FR-8 | T2 | DD-6 | AC-2 |
| FR-9 | T1, T2 | DD-5 | AC-7, SC-5 |

P-1..P-5 are properties of the shape above rather than of a statement: P-1 from the call sitting
below both draws, P-2 and P-3 from the gate only ever setting one bit, P-4 from DD-1, P-5 from the
bridge writing nothing but the matrix.
