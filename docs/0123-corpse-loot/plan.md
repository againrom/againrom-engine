# 0123-corpse-loot — plan

## Design decisions

**DD-1 — The drop goes in `clearFelled`, not in the decay pass.** `clearFelled`
(`pkg/sim/step.go`) is the one death transition: it is reached from the kill arm, the
damage arm and the blow resolver, and its `Decay == DecayNone` guard already carries "this
happens once per death" for the stage, the dwell and the halved defence. Putting the drop
inside that guard makes FR-4 the guard's own property rather than a second rule to keep in
agreement with it. The decay pass is the wrong place twice over: it walks bodies that are
already dead, so a drop there would fire on a rung of the ladder rather than on the death,
and it runs at the end of an advance, so a body felled in phase 1 would spend the tick
dead with its goods still on it.

**DD-2 — Merge-or-insert by binary search, no re-sort.** FR-3 is answered by one small
method on the world: `sort.Search` over the `(Y, X)`-ascending sack list — the same search
`sackAt` and `TakeSack` already run — then either an append onto the found sack's codes or
an insert at the search's own index. Inserting at that index preserves the order by
construction, so nothing re-sorts and nothing re-scans. A `sort.Slice` here would be a
second place that decides what the list's order is.

**DD-3 — The container is moved, not copied.** The drop hands the entity's own slice to
the sack and sets the entity's to `nil`. That is FR-1's "identity, not a copy" and it also
makes double-drop unrepresentable: after the move there is nothing left to drop again.
Nothing else in the package aliases a container slice — `Carried` and `Stock` both copy
out — so handing the slice over cannot reach a caller.

**DD-4 — `remove` rebuilds three parallel slices, not two.** FR-6 is a one-line repair with
a load-bearing shape: `carried` joins `entities` and `routes` in the single compaction loop
rather than being swept separately afterwards. Separately would be a second walk that could
come to disagree about which entities survived, which is exactly how the two came apart.

**DD-5 — The bounds guard reuses `sackFault`.** FR-5 asks the one per-sack predicate the
constructor and the decoder already share, instead of a fresh bounds test. A cell that
predicate refuses is a cell no world here may hold a sack on, whichever door it came
through.

**DD-6 — No new saved state, and the version stays put.** P-1 is a design decision and not
an outcome: the drop was scoped to what the byte form already carries. The gold roll and
the suppression mark are declined for exactly this reason, and their cost is recorded in
`spec.md`'s "Out of scope" so the story that takes them knows what it is buying.

**DD-7 — The random-source doc block is corrected in the same story that declines the
draw.** FR-7 is a comment, and it belongs here rather than anywhere else because this
story is the one that had to decide whether a death draws — and answers no (AC-7, P-2).
Leaving a doc block asserting that nothing describes the original's generator, in the file
the next story that *does* need a draw will read first, is how a stale premise gets
believed.

## Traceability

| FR | Decisions | Witnessed by |
|---|---|---|
| FR-1 | DD-1, DD-3 | AC-1, SC-2 |
| FR-2 | DD-1 | AC-2 |
| FR-3 | DD-2 | AC-3, AC-4 |
| FR-4 | DD-1, DD-3 | AC-5 |
| FR-5 | DD-5 | SC-5 |
| FR-6 | DD-4 | AC-6, SC-3 |
| FR-7 | DD-7 | SC-6 |

`P-1` and `P-3` fall out of DD-6 and DD-3, and AC-9 is what measures P-1 from the outside;
`P-2` falls out of DD-1 and DD-7.

## Success criteria

**SC-1** `go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test
-trimpath -count=1 ./...` all clean on the committed tree, plus
`scripts/check-no-game-assets.sh`, `check-doc-budget.sh` and `check-sdd-audit.sh`.

**SC-2** Reverting the drop's body — leaving the call site and the guard in place — turns
AC-1 red. Recorded with the failure text, not with a reading of the assertion.

**SC-3** Reverting `carried` out of `remove`'s compaction turns AC-6 red, with the panic
this story found still in it.

**SC-4** The world hash and the byte form are unaffected for any world in which nothing
dies: the existing form and digest tests pass unchanged, and the form version constant is
the same value it held before this story.

**SC-5** A death at a cell outside the world's bounds is exercised and leaves both the sack
list and the container untouched.

**SC-6** `pkg/sim`'s generator doc block names the original's generator and states our
divergence; `internal/archtest`'s source scan over `pkg/sim` still passes, so the
correction introduced no banned import, identifier or literal.

## Risks

- **A drop mid-advance changing an existing test's expectations.** The drop happens inside
  a tick, so any test that kills a carrying entity and then reads the sack list will see
  one more entry. Mitigated by the fact that no test before this story gives a dying entity
  a container; if one does, it is a real behaviour change and belongs in `verification.md`.
- **The move aliasing a caller's slice.** Mitigated by DD-3's own note: every path out of
  the package copies.
