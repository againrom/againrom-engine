# Tasks — a placed unit's combat numbers

Legend: **Kind** is `impl` (one coherent product change, one trailered commit) or `test` where a
task's whole product is test code. `Done when:` is the entry's own exit condition. Criterion
numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 the seam, and the two pins it moves

Kind: impl. Carries FR-1, FR-2, FR-3, FR-4, FR-6, FR-7, P-4, DD-1, DD-2, DD-3, DD-4, DD-5, DD-6,
R-1, R-4. Criteria SC-6, SC-7.
Files: `pkg/mapload/fromalm.go` MODIFY, `pkg/mapload/spawn.go` MODIFY, `pkg/mapload/start.go`
MODIFY, `pkg/mapload/fromalm_test.go` MODIFY.

Boundary: the production write and the two pins that break because of it. No new test file.

Scope fence: no file under `pkg/sim`, `pkg/data` or `pkg/formats` is touched — if one appears to
need touching, the design is wrong and this task stops. The health an unresolved placement carries
does not move. The difficulty adjustment's own code is not edited: FR-3 is already satisfied by it
and this task adds no arm to it.

Done when: a resolved placement's entity carries the eight off its definition and an unresolved one
carries the constructor's; the movement-column helper's doc no longer argues from a zero definition;
the party literal reaches the fallback through the same source the placement loop does; both pinned
digests assert their new values; and a test shows that zeroing the eight fields' bytes in every
record of each new form reproduces that fixture's pre-story digest exactly.

## T2 the witnesses

Kind: test. Carries FR-1, FR-2, FR-3, FR-4, AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-9, P-1, P-2,
P-3, R-4. Criteria SC-1, SC-2, SC-3, SC-4, SC-5, SC-8.
Files: `pkg/mapload/combat_test.go` ADD, `pkg/mapload/start_test.go` MODIFY.

Boundary: test code only. No production file changes; if one has to, T1 was incomplete.

Fixtures are synthetic definition tables built in test code. Choose the eight source values so no
two are equal and none equals a constructor default, so a field crossed with another field and a
field left unwritten are two different failures.

Scope fence: the difficulty rows compare **two built worlds** rather than asserting arithmetic on a
definition, so a change to the adjustment that this contract forbids fails here rather than passing
by construction. The party row compares against a placement **in the same world**, never against a
literal 8 and 4.

Done when: each of the eight has its own failing assertion; every admitted value of the routing
column has a row and the always-hits mark is asserted set on exactly one; the hard rows assert the
six that must not move as well as the two that must; the three unresolved shapes are asserted equal
to each other and not only to the expected pair; the refusal row asserts a nil world and a message
naming the entry; and a world advanced some ticks is asserted to carry the eight the load wrote.

## T3 the dump the owner reads

Kind: impl. Carries AC-10, DD-7, R-3. Criterion SC-10.
Files: `cmd/classdump/databin.go` MODIFY, `cmd/classdump/main.go` MODIFY,
`cmd/classdump/databin_test.go` MODIFY.

*Corrected 2026-08-03: this line said `campaign.go`, a different verb taking an asset root and no
difficulty. The boundary below is right; only the file names were wrong.*

Boundary: one added report on the existing table verb, which already takes an archive, an optional
map and a difficulty. No new verb and no new binary.

The report prints one row per placement of the named map: the class key pair, the entry name the
placement resolved to or that it resolved to none, and the eight numbers. Its heading states that
these are the template's numbers and that equipment is not applied (DD-7, R-3).

Scope fence: the verb's existing rows, totals and exit codes do not change. Numbers and names only —
no raw parameter array and no byte dump.

Done when: the verb prints the report for a named map at each of the three difficulties; a
placement that resolved to nothing is shown as such rather than as a row of zeros; the caveat is in
the output and not only in this repo; and the verb exits non-zero on a failure to open, read or
resolve.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1 | SC-1, SC-6 | T1, T2 |
| FR-2 | SC-2 | T1, T2 |
| FR-3 | SC-3 | T1, T2 |
| FR-4 | SC-4, SC-5 | T1, T2 |
| FR-5 | — (a requirement met by adding nothing; witnessed by T1's scope fence and by the unchanged form) | T1 |
| FR-6 | SC-6, SC-7 | T1 |
| FR-7 | SC-1 | T2 |
| AC-1…AC-6, AC-9 | SC-1…SC-5, SC-8 | T2 |
| AC-7, AC-8 | SC-6, SC-7 | T1 |
| AC-10 | SC-10 | T3 |
| P-1, P-2, P-3 | SC-1, SC-4, SC-8 | T2 |
| P-4 | SC-6, SC-7 | T1 |
| P-5 | SC-6 | T1 |
| SC-9, SC-11 | — | the Verify stage, which is not a task |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above
and neither commit carries a trailer.
