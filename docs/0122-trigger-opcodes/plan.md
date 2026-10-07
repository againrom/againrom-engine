# Plan — 0122-trigger-opcodes

## Shape

Five commits. One carries the record change and the form; three carry arms; one is a
comment-only correction. Every arm but the diplomacy write is a *check*, so the order matters in
one place only: the record change must land before the three arms that read it.

```
T1  two stale doc blocks corrected          pkg/sim/script.go
T2  the two player references + form v31    pkg/sim/{script,scriptbinary,binary}.go, pkg/mapload/script.go
T3  checks 8 and 15                         pkg/sim/script.go
T4  check 10 and instant 10                 pkg/sim/{script,relations}.go
T5  the dead arms 11 and 13                 pkg/sim/script.go
```

## Design decisions

**DD-1 (FR-1) — the second player is a slot of its own, named `Player2`, not a general "second
reference".** The record being reconstructed has one `+0x3c` slot holding *whichever kind came
second*, and this build already declines that generality: it carries `Unit` and `Unit2` as two
typed slots. A third spelling of the same idea would be a second convention in one struct. So
`ScriptCheck` gains `Player`, `Player2`, `HasPlayer`, `HasPlayer2`, exactly mirroring the unit
pair beside it.

**DD-2 (FR-7) — `ScriptInstant` is not touched.** The diplomacy write takes its two roster slots as
**plain parameters**, not as references, so instant 10 reads `Args[0]`, `Args[1]`, `Args[2]` and
the instant record keeps the width it has. Only the check record moves.

**DD-3 (FR-2) — the binder's player rule becomes the unit rule.** `bindParams` already parses a player
parameter into `b.player`/`b.hasPlayer` and keeps only the first, on a comment that says no arm
of the vocabulary names two. **That comment is now false** — check 10 names two — so the binder
gains a `seenPlayer` counter shaped exactly like the existing `seenUnit`, and the comment is
rewritten to say what is now true. Instants keep taking the first player only, which is what they
have always done and what every instant arm reads.

**DD-4 (FR-3) — byte form version 31, and the number is disclosed as unallocated.** The check record
goes 63 → 73 bytes: `+0x3f` player, `+0x43` player2, `+0x47` and `+0x48` their flags — a tail
append, so every offset the previous version fixed stays where it was, and the instant and
trigger arrays that follow simply start later. `formatVersion` moves from 26 to **31**. 27 through
30 are skipped deliberately: three of them were reported out to sibling lanes and one is
unaccounted for, and this tree already has the precedent of a skipped number recorded in a
comment. The doc block above `formatVersion` states, in its own paragraph, that **31 was chosen
in the lane without an allocation and may be renumbered by a single edit**.

**DD-5 (FR-3) — the pinned form barely moves, and that is worth knowing before starting.** The pin world
carries **no script**, so the widened check record contributes no bytes to it; only the version
byte at offset 0 changes. What must be recomputed is therefore `pinDigest` in
`pkg/sim/hash_test.go` and any digest taken over `pinBytes`. Take the new value from the failing
test's own output; do not compute it by hand. The previous-version backstop follows the shape
already in `binary_test.go`: a `preScriptPlayerFormVersion` constant holding **26**, and the
refusal message naming both numbers.

**DD-6 (FR-4, FR-5) — one walk, two arms.** Checks 8 and 15 both iterate `w.entities`, filter on
`Owner == c.Player`, and skip the dead with the same `scriptDead` predicate every other arm uses.
They are written as two cases sharing one documented rule rather than as a shared helper: a helper
returning a count *and* a minimum would compute both for every caller, and the two arms answer
different questions about the same set. **No membership index is built** — the group count's own
doc block already says why, and the same reason holds: an index would be a second representation
of a fact nothing in this package changes, and it would put an iteration order into the digest.

**DD-7 (FR-5) — check 15 seeds `0xff` and the seed survives an empty set.** The seed is the arm's own,
and it doubles as the answer for a player with no living unit. That is not the same as writing
nothing: a check naming **no player** writes no register at all. Both paths are exercised by
AC-2, and the trace records the second as a silence.

**DD-8 (FR-9) — a new silence kind for the missing player.** `ScriptSilence` gains
`ScriptSilenceNoPlayer`, beside `ScriptSilenceNoUnit` and `ScriptSilenceNoGroup`. The three
existing kinds are a closed set with named values; adding a fourth rather than reusing
`NoUnit` keeps a trace readable, and this value is not serialised anywhere.

**DD-9 (FR-6, FR-7) — the diplomacy write is a method on `Relations`, not a poke into its slice.**
`relations.go` states that one type exists so that **one place computes the offset**; a script arm
reaching into `cells` would be the second. The new method sits beside `turnHostile` and is
deliberately *unlike* it: `turnHostile` declines a locked or already-hostile pair, and this one
declines nothing. Both go through `relationIndex`, so an out-of-range slot is one rule and not
two. Like `Set`, it materialises the matrix on first write; unlike `Set`, it preserves the high
bits.

**DD-10 (FR-8) — the dead arms are a third answer, and `scriptCheckSupported` is where it is given.**
That function is documented as the ONLY place the supported answer is given, so both the compile
report and the runtime dispatch must take the dead arms from it. The dead arms are therefore
**supported** — they reach a `case` in `runCheck` that returns having written nothing. That single
change gives all three of FR-8's consequences at once: no gap is reported, no reader is inerted,
and no register is written. A separate "dead" table would be a second place the answer is given
and could come to disagree with the first.

**DD-11 — the two comment corrections say what is fixed AND what is not.** `TRIG-DIST-014` is
High for the metric and **Medium for the byte-width clause**, so the corrected block must state
that the metric is settled while the mask-before-subtract ordering is not, and the existing
divergence note about masking the result rather than the operands stays. Correcting a comment into
a second overstatement is the failure this task exists to undo.

## Risks

**R-1 — the version number collides with a sibling lane.** Mitigated by DD-4: one place, one
line, and a doc paragraph that says so. If the seat renumbers, only `formatVersion` and
`preScriptPlayerFormVersion`'s companion prose move; the digests are recomputed by running the
tests.

**R-2 — the dead arms silently change which triggers are inert on a real map.** They cannot: no
shipped map on either root authors opcode 11 or 13. The change is reachable only by a customised
map, which is exactly the customisation seam this is for. State it in `verification.md` rather
than asserting a corpus figure this tree cannot measure without an install.

**R-3 — check 15 writing `0xff` where it should write nothing.** The two paths differ in one
condition and a reader could collapse them. AC-2 tests both and the trace distinguishes them.

**R-4 — the binder's second player disturbs the plain-parameter packing.** It cannot: references
have never joined `Args`, and the `seenPlayer` counter changes no branch that touches `next`. A
regression here would show as a shifted parameter on an unrelated arm, so T2's test builds a node
carrying a player *between* two plain parameters and asserts both parameters land where they did.

**R-5 — the arms reach outside `pkg/sim`.** They do not. `internal/archtest` bans floats, `os`,
`time` and `math/rand` in this package; nothing here needs any of them.

## Success criteria

**SC-1.** `scriptCheckSupported` names 8, 10, 11, 13 and 15 beside the nine it already names, and
it is still the only place that answer is given.

**SC-2.** `Script.Unsupported()` reports **no** gap for opcodes 11 and 13, and a trigger reading
one is not in `InertTriggers()`.

**SC-3.** The compiled check record round-trips both player references through the byte form, a
form at version 26 is refused naming both versions, and `pinDigest` is the recomputed value with
no other pin touched.

**SC-4.** `Relations` has exactly two runtime writers and both compute their offset through
`relationIndex`.

**SC-5.** Every arm added is witnessed by reverting it: with the arm's body removed its own test
fails, and no other test fails in its place.

**SC-6.** The full gate is green on the committed tree with no game present:
`go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -trimpath -count=1 ./...`,
plus `scripts/check-no-game-assets.sh`, `check-doc-budget.sh`, `check-sdd-audit.sh`.
