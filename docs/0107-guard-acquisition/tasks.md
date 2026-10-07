# Tasks — 0107

Two tasks, both inside `candidateCost` in `pkg/sim/engage.go`. T1 is the member's own reach, T2
the candidate's. Each leaves the tree green on its own.

| Task | FRs | DDs | ACs | Ps |
|---|---|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-6 | DD-1, DD-3, DD-4 | AC-2, AC-3, AC-4 | P-1, P-5 |
| T2 | FR-4, FR-5, FR-7, FR-8, FR-9 | DD-2, DD-5, DD-6 | AC-5, AC-7 | P-2, P-3, P-4 |

AC-1 and AC-6 are measurements the verification stage takes against an installed root; a task
neither asserts them nor reads an install.

## T1 — the member's own reach

**Files:** `pkg/sim/engage.go`, `pkg/sim/engage_test.go`, one new `_test.go` file.

Two statements, both gated on `m.Reach > 1`, both in `candidateCost` and nowhere else.

The **row choice** (FR-1, FR-3): a ranged member's `pref` comes from `preference[0][...]`, the
row index written as the literal `0` — not `lawDomain` of anything.

The **distance rewrite** (FR-2), immediately after: `d <= int32(m.Reach)` sets `d = 1`,
otherwise `d = d + 1 - int32(m.Reach)`. Two arms in that order, **not** a `max` (DD-3), and no
clamp (DD-4).

Then **move** the existing `order == orderStandGround && d > groupScorerReach` refusal **below**
the rewrite (FR-6). `groupScorerReach` stays the literal 1, unwired from `Entity.Reach`. Read
`plan.md`'s FR-6 paragraph first: that ordering is derived, graded Medium, and the comment must
say so.

Do not split the function in two (DD-1).

Tests: a ground member scoring a flier — seed at reach 1, a finite cost at reach 4, and that
cost the row-0 value (AC-2); the distance term over separations 0 to 8 at reach 4, flat at 1
through 4 then rising (AC-3, P-1); stand ground at reach 4 taking a candidate at 4 and refusing
at 5, at reach 1 taking only what is adjacent (AC-4).

## T2 — the candidate's reach, and the boundary

**Files:** `pkg/sim/engage.go`, `pkg/sim/engage_test.go`.

The **column choice**, computed before T1's row choice because it feeds it: the candidate's
index is `0` when `c.Reach > 1` **and** it is a ground mover (FR-4), and `0` when `c.Reach > 1`
whatever its domain under the stand-ground order (FR-5). One expression with an `||` on the
order. Keep it a separate statement from the row choice (DD-2).

The veto line stays where it is, above everything both tasks add (FR-7).

**Nothing outside `candidateCost` is edited** (FR-8, FR-9). Do not open `binary.go`: the version
stays 24 and this story's allocated one is not spent. Do not touch `candidates`,
`clipToNotice`, `noticeBase`, `noticeRadius`, `freezeGroups` or `decide`; leave
`minimalGuardRange` a constant (DD-5) and the notice circle alone (DD-6).

DD-1..DD-6 get a comment at the statement each explains, in the voice of the ones already here.

Tests: a reach-4 ground candidate scored by a reach-1 member takes the immobile column under the
ordinary order; a reach-4 **flier** does not; under stand ground it does (FR-5). The landed
suite green with no pin re-taken (AC-5); a decode of a world encoded before the change (AC-7);
the determinism scan unchanged (P-2); two greps — `scoreSeed` at its two uses (P-3),
`candidateCost` the only scorer (P-4).
