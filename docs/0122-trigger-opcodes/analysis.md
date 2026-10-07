# Analysis — 0122-trigger-opcodes

## The question

A mission script arm this build does not implement is not a small inaccuracy. A check arm writes
no register, every trigger reading that register is marked inert and never evaluated, and the
authored behaviour behind it cannot happen on any map. The vocabulary is 22 check arms and 34
instant arms; this build evaluates nine and seven. The census that opened this story counted the
cost over the shipped corpus: of 545 runtime condition nodes, 191 take no arm, and **123 of those
191 are one arm** — check 15, the most-used runtime check in the whole corpus.

So the question is not "which arms are missing" but **which missing arms are worth the most nodes
per unit of work, and which of them the decoded law actually fixes**.

## What the law fixes, and what it does not

Five arms are settled well enough to build, and they divide into two readings.

**Reading one — walk a player's units.** `TRIG-DIST-014` (High) reads the distance helper whole:
fifteen instructions returning `max(|dx|,|dy|)`, with no multiply, no `FSQRT`, no `FILD` and no
addition of the two terms, so Euclidean and Manhattan are excluded by the instructions present
rather than by preference. It then names all five arms that use it, check 15 among them:
`R2056`, the **minimum** over every member of every group of the named player, seeded
`0xff`. `TRIG-COUNT-015` (High) reads check arm 8 in the same routine: it walks every group of
the player and every member of every group, and its complete per-member body is `INC EDI` — a
population count with no filter. Both arms need one thing this build's compiled check record does
not carry: **a player reference**.

**Reading two — the diplomacy pair.** `TRIG-DIPLO-019` (High) transcribes action arm 10
instruction by instruction: index `50*p0 + p1`, one direction, no mirrored store, bits 2..7
preserved, and the parameter **ADDed rather than ORed**. `TRIG-DIPLO-020` (High) reads check
arm 10 as thirteen instructions yielding `matrix[A][B] & 3`. `TRIG-DIPLO-021` (Medium) censuses
the corpus: eleven action nodes over seven maps, five check nodes over five on EN, and the map
authors' own labels show a mutual change costing two nodes — which is what a single index
expression predicts and what a symmetric write would make redundant.

**And two arms that are dead in the original.** `TRIG-COND-003` (High) reads the 22-entry table
out of the PE and finds arms **11 and 13 pointing at `L12305`**, the dispatch loop's own
continue label. They are dispatched, write nothing, and leave the slot at its previous value —
which is a different behaviour from an unimplemented arm, because the trigger reading that slot
is still evaluated.

## What the law does not fix, and is therefore not built

The census sized the "units enter and leave" family — instants 16, 17, 18, 32, 33, 81 nodes, 116
with the spawn — as the natural second group. It is not built here, and the reason is not effort.
`TRIG-ACT-004` names those arms' callees (`R0121`, `R0123`, `R0122`) and one
of their error strings; nothing published says **what an off-map unit is**. Whether it ticks,
whether it can be struck, whether it holds its cell, whether check 15 or the group count still
sees it — every one of those is a behaviour a consumer would have to author, and each is reachable
by a shipped map. Building the arm without them would not be a partial implementation, it would be
an invented one.

The spawn arm 21 fails twice over: `TRIG-ACT-004` states in its own grading that the
six-argument order of `R1138` is **not interpreted**, and `pkg/sim` holds no unit template
table from which an entity's health, speed, reach and sight could be built.

The item arms — checks 12, 16, 17 and actions 12, 13, 20, 28 — need a per-actor container that can
resolve a *named* item. This tree's `pkg/sim/carry.go` holds item codes rather than identified
items, and the binder drops a node's item reference outright.

Group sub-commands 11 and 15 are the one refusal that is *not* a gap in the law:
`AI-FOLLOWSET-116` and `AI-FOLLOWAUTH-117` (both High) read Defend and Follow whole, down to the
three bytes by which the two setters differ. They are refused because the per-actor states they
write — 8 and `0x11`, beside this build's `guard` and `patrol` — do not exist in this build's
actor machine at all, and the subject and range they carry would be a second widening of the
entity record. That is a story, not a task.

## The enabling change, and its price

Four of the five arms built here need a player on the compiled check record, and that record is
**serialised in the hashed byte form** (`pkg/sim/scriptbinary.go`, `scriptCheckLen = 63`). So this
story cannot be the pure-read story it was expected to be: it widens the check record and moves
the form's version. That is the single largest fact this analysis contributes, because it was
believed otherwise when the story was written.

The widening is small and it is at a tail: four fields on a record that already carries three
references on exactly these terms, and the pinned form the tests fix carries no script, so the
pin's own bytes move only at the version byte.

## What is bought

Four new arms and two dead arms reproduced. Over the shipped EN corpus: check 15's **123** nodes,
check 8's **4**, check 10's **5** and action 10's **11** — **143 nodes**, and the condition side
goes from 354 of 545 evaluated to **486 of 545**. Arms 11 and 13 buy no node and close the one
divergence in this subsystem that nothing disclosed.
