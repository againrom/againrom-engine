# Tasks — 0098

One task. The whole contract is one function and two call sites in a single file, and the tests that
witness it read that file's own package internals. A second task would either split a function from
its own tests or put two agents in one file.

`verification.md` and the build are pipeline stages, not tasks, and carry no trailer.

| Task | FRs | ACs | DDs |
|---|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10, FR-11, FR-12, FR-13, FR-14, FR-15 | AC-1 … AC-17 | DD-1, DD-2, DD-3, DD-4, DD-5, DD-6, DD-7, DD-8, DD-9, DD-10, DD-11, DD-12, DD-13 |

## T1 — the release, and the file's own account of it

**Files:** `pkg/sim/engage.go`, `pkg/sim/engage_test.go`, new `pkg/sim/release_test.go`, and nothing
else — `internal/archtest` in particular is not edited (DD-13).

Add `releaseAttack` beside `orderAttack` as its inverse and the only clearer a decision reaches: it
returns at once on a member holding no victim, then calls `clearAttack` and `clearOrder`. In
`decide`, replace the `len(cands) == 0` early return with the two narrowed count tests, releasing
every member and returning; and give `if at >= 0` an `else` that releases. Both sites gate on guard.

**Exactly two landed tests go red and are meant to** —
`TestAFriendlyRelationDoesNotEndAnAttackAlreadyIssued` and
`TestReEngagingTheSameVictimDoesNotRestartTheCycle`, measured red already and handled by DD-12.
**No other test may be edited**; a third going red is a stop-and-report.

Rewrite the file header's closing paragraphs, `decide`'s doc comment and `aiGroup`'s doc — each
states what the pin or this build contradicts (plan step 0, DD-9).

Write `pkg/sim/release_test.go` for every row of the plan's table.

**Done when:** `go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test
-trimpath -count=1 ./...` are clean, `sh scripts/check-no-game-assets.sh` and
`check-doc-budget.sh` pass, and every FR, AC and DD above has a named witness.
