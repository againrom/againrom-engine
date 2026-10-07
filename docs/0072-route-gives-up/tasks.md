# Tasks — the route the hero does not take

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 the near search settles

Kind: impl. Carries FR-1, FR-2, FR-3, FR-4, FR-5, P-1, P-2, P-3, DD-1, DD-2, DD-3, DD-4, DD-5, DD-6,
R-1, R-2, R-3. Criteria SC-1, SC-2, SC-3, SC-4, SC-5.
Files: `pkg/sim/route.go` MODIFY, `pkg/sim/step.go` MODIFY, `pkg/sim/optimised.go` MODIFY,
`pkg/sim/waypoint_test.go` ADD, and the affected `pkg/sim/*_test.go` and `pkg/mapload/*_test.go`
MODIFY.

Boundary: the search parameter, the bound it carries, the two call sites in the tick, and the
witnesses. No field is added anywhere, no record changes width, and the byte-form version does not
move — if one has to, the boundary was crossed and this task stops.

Scope fence: the stall count, its limit and the clearing it triggers are not edited. The far search's
relation, window, budget and bound are not edited. No product file outside `pkg/sim` is touched.

The moved pins live in `approach_test.go`, `domain_test.go`, `downed_test.go` and
`pkg/mapload/routing_test.go`.

Done when: a mover walks past a body standing on its waypoint and arrives, and does not when the
fixture's body is left in a sealed corridor; the near bound is witnessed as eight against the far
rule's on one world; a boxed-in mover still counts to the limit and ends with no residue; every pinned
digest is unmoved or re-derived forward from pre-change bytes; and the suite, the import graph and the
determinism-wall scan are clean.

## T2 a mission is driven to its outcome

Kind: impl. Carries FR-6, DD-7, DD-8. Criteria SC-6.
Files: `cmd/missionrun/main.go` ADD, `cmd/missionrun/main_test.go` ADD,
`internal/archtest/dag.go` MODIFY, `docs/ARCHITECTURE.md` MODIFY.

Boundary: the tool and its place in the import graph. Nothing under `pkg/` is edited — a change there
means T1 was incomplete and this task stops.

Scope fence: the tool issues plain move orders and reads the outcome. It does not reach into the
script, does not set a register, and does not write any file.

Its own tests are synthetic except one: the campaign drive, which skips unless an asset root is
configured. Everything the tool can be asked that does not need a map — argument parsing, waypoint
syntax, unit references, a mission number that is not one, a missing asset root — is witnessed
without one.

Done when: `missionrun` starts a named campaign mission from a configured asset root, drives each
waypoint's unit in turn, and prints the outcome, the tick it was decided on and each unit's final
cell; the tenth mission reaches the won outcome; and `go test ./...` is green with no game present.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1 | SC-1 | T1 |
| FR-2 | SC-2 | T1 |
| FR-3, FR-4 | SC-4 | T1 |
| FR-5 | SC-3 | T1 |
| FR-6 | SC-6 | T2 |
| AC-1 | SC-1 | T1 |
| AC-2 | SC-2 | T1 |
| AC-3, AC-4 | SC-3 | T1 |
| AC-5 | SC-4 | T1 |
| AC-6 | SC-4 | T1 |
| AC-7 | SC-6 | T2 |
| P-1, P-3 | SC-5 | T1 |
| P-2 | SC-4 | T1 |
| DD-1, DD-2, DD-3 | SC-2 | T1 |
| DD-4, DD-5, DD-6 | SC-3, SC-4 | T1 |
| DD-7, DD-8 | SC-6 | T2 |
| R-1 | SC-1, SC-6 | T1, T2 |
| R-2 | SC-4 | T1 |
| R-3 | SC-5 | T1 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above and
neither commit carries a trailer.
