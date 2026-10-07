# Tasks — the hero the script cannot see

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 the id the start will assign, answerable from the map

Kind: impl. Carries FR-5, FR-6, DD-1, DD-2, DD-7, P-1, R-1.
Criteria SC-3.
Files: `pkg/mapload/party.go` ADD, `pkg/mapload/party_test.go` ADD, `pkg/mapload/script.go` MODIFY.

Boundary: the rule, its pinning, and the one comment DD-7 corrects. No caller is rewired here, so
this task changes no behaviour — the `script.go` edit is prose only.

Scope fence: `pkg/mapload/start.go` is **not** edited — the agreement is pinned by test, not by a
shared expression (DD-2). No file under `pkg/sim`, `pkg/game`, `pkg/render` or `pkg/ui` is touched.

The pin is the load-bearing half: for a synthetic map over a range of placement counts including
zero, and a party over a range of sizes including zero, assert the published id equals the id the
started world actually gives that party member, and that the entity carrying it stands on the
start's own cell for that member. An assertion against a recomputed count instead of against the
started world proves nothing.

Done when: the rule is published from `pkg/mapload`, answers for a nil map and for a map with no
placements, the pin above passes for every combination in its range, and the binder's
corpus-resolution sentence says what it is a claim about and what this build does instead.

## T2 the mission binds its hero

Kind: impl. Carries FR-1, FR-2, FR-3, FR-4, FR-7, FR-8, FR-9, FR-10, P-2, P-3, DD-3, DD-4, DD-5,
DD-6, R-2, R-3.
Criteria SC-1, SC-2, SC-4.
Files: `pkg/game/mission.go` MODIFY, `pkg/game/mission_test.go` MODIFY.

Boundary: the mission start's compile call and the tests that measure what it now resolves. The
resolver, the band constants, the report and the start are all already correct and none is edited.

Scope fence: no file under `pkg/sim` or `pkg/mapload` is touched, the map-inspection tool under
`cmd/` is not edited (FR-10), and no serialized form version is changed (FR-9, DD-5). If any of the
three appears to need it, DD-1 or DD-5 is wrong and this task stops.

Write SC-2's negative half **first** (R-3): the fixture's winning trigger, driven with the hero away
from its point, must not reach the won outcome — and that assertion must **fail against the
pre-change code**, because the register it reads holds zero and zero satisfies the comparison. Only
then add the positive half. Give the fixture enough script passes for a variable one trigger writes
to be read by a later one.

Done when: every criterion above holds on a mission driven through the front end's own start, not
through a private compile; and the suite passes with no byte-form version edited.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1, FR-2 | SC-1, SC-2 | T2 |
| FR-3, FR-4 | SC-1 | T2 |
| FR-5, FR-6 | SC-3 | T1 |
| FR-7 | SC-1 | T2 |
| FR-8, FR-9 | SC-4 | T2 |
| FR-10 | — (met by changing nothing; witnessed by T2's scope fence) | T2 |
| AC-1, AC-2 | SC-1, SC-2 | T2 |
| AC-3 | SC-2 | T2 |
| AC-4, AC-5 | SC-1 | T2 |
| AC-6 | SC-3 | T1 |
| AC-7 | SC-1 | T2 |
| AC-8 | SC-4 | T2 |
| P-1 | SC-3 | T1 |
| P-2, P-3 | SC-4, SC-2 | T2 |
| DD-1, DD-2, DD-7 | SC-3 | T1 |
| DD-3, DD-4, DD-5, DD-6 | SC-1, SC-2, SC-4 | T2 |
| R-1 | SC-3 | T1 |
| R-2, R-3 | SC-2, SC-4 | T2 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above
and neither commit carries a trailer.
