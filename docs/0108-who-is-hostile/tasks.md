# Tasks — 0108

Two tasks, both in `pkg/sim`. T1 is the gate on the relation type, T2 the call from the blow.
Each leaves the tree green on its own.

| Task | FRs | DDs | ACs | Ps |
|---|---|---|---|---|
| T1 | FR-4, FR-5, FR-6, FR-7, FR-9 | DD-1, DD-4, DD-5 | AC-3, AC-4, AC-7 | P-2, P-3, P-4 |
| T2 | FR-1, FR-2, FR-3, FR-6, FR-8 | DD-2, DD-3, DD-6 | AC-2, AC-5, AC-6 | P-1, P-5 |

AC-1 and AC-8 are measurements the evidence stage takes against an installed root, and SC-1..SC-5
are its own steps; a task neither asserts them nor reads an install.

## T1 — the gate

**Files:** `pkg/sim/relations.go`, `pkg/sim/relations_test.go`.

Add one pointer method beside `Set`: `turnHostile(from, to uint32)`. It finds the cell through
`relationIndex`, returns silently when the pair has none (FR-7), and sets **bit 0 only if the
cell's low two bits are both clear** (FR-5). Nothing else is written and no bit is ever cleared.
Materialise on write as `Set` does.

It takes **one direction**. Do not add a symmetric form — read `plan.md`'s DD-4 first.

Nothing else in the file changes, and nothing outside it indexes `cells` (DD-1, P-4). Do not open
`binary.go`: the version stays 24 and this story's allocated 25 is unspent (DD-5, FR-9).

**Correct the bit-1 paragraph's last sentence** — it says bit 1's writer is unimplemented, and
after this task it is not. Say what bit 1 now does: it is the one thing that stops a blow.

Tests: the four low-bit shapes in and out; bits above 1 preserved; twice equals once (P-3); no
other cell moves; slot 0 and a slot past the matrix are no-ops; no run of calls lowers a byte
(P-2).

## T2 — the call from the blow

**Files:** `pkg/sim/{combat,world,combat_test,engage_test}.go`.

Add `func (w *World) flipOnBlow(ai, ti int)` in `world.go` beside `hostileTo`, for its reason.
It returns unless **both** entities have a non-zero `Owner` (FR-2), then calls `turnHostile`
twice — `[attacker][victim]`, then `[victim][attacker]` (FR-4). Nothing else is written and
nothing is called (FR-8, P-5). **No same-slot guard** (DD-3): both calls naming one cell is FR-6
working.

In `resolveBlow`, put `w.flipOnBlow(ai, ti)` immediately after the to-hit `if` and immediately
before `dmg -= int64(t.Absorption)` (FR-1, FR-3, DD-2). Read `plan.md`'s FR-1 paragraph for why
that exact line. It is the only call site.

Correct `Relations()`'s comment: the type has one writer now, and the copy-on-read keeps it the
only one.

Tests: AC-2 through a struck flippable pair, ending with the struck slot's group acquiring the
striker where before it saw nothing; AC-5's three no-ops; AC-6's two blows; and both arms of FR-6 —
the diagonal set in a world with no relation, a loaded 2 left alone.

**Sweep the landed suite for FR-6's second arm first.** Any existing test where two entities of one
owner exchange blows in a world built with no relation now ends self-hostile. If one moves, say so
in the commit message; do not adjust the rule to spare it.
