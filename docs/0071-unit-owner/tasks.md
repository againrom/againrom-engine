# Tasks — the owner a script hands over

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria.

## T1 the owner reaches the world, and the instant reaches its references

Kind: impl. Carries FR-1, FR-2, FR-3, FR-6, P-3, DD-1, DD-2, DD-3, DD-4, DD-5, DD-10.
Criteria SC-1, SC-2, SC-3, SC-4.
Files MODIFY: `pkg/formats/alm/alm.go`, `pkg/sim/{world,script,binary,scriptbinary,step}.go`,
`pkg/mapload/{fromalm,script}.go`, and the affected `*_test.go` of those three packages.

Boundary: the state and its carriage. Both arms stay outside the supported set — SC-4 measures that,
and it is what makes this task independently coherent.

Scope fence: no dispatch arm, no edit to the supported-arm table. **Take the byte-form version number
from the orchestrator; do not read the next one off the tree.** A node's plain parameters coming out
different than before means DD-4 is broken and this task stops. The `step.go` edit is prose only. A
pinned digest changes only where a widened record makes it a different world.

Done when: a decoded placed record carries its owner word and the document writes back byte-identical;
an entity built from a placement carries it, one built from no map carries zero, and a felled one keeps
it in both; a compiled instant carries three references with presence apart from value; both widened
records cross the form and the digest; the previous version, a truncated record and a bad presence byte
are refused; both arms are still in the unsupported report.

## T2 the two arms hand ownership over

Kind: impl. Carries FR-4, FR-5, FR-7, P-1, P-2, DD-6, DD-7, DD-8, DD-9.
Criteria SC-5, SC-6, SC-7, SC-8.
Files: `pkg/sim/script.go` MODIFY, `pkg/sim/script_test.go` MODIFY, `pkg/sim/scriptowner_test.go` ADD.

Boundary: the table that decides what this build runs, the two dispatch arms behind it, and the
witnesses. No record changes width and no loader is edited.

Scope fence: nothing outside `pkg/sim`, and **no arm reads an owner** — a branch on one is the fence
being crossed. No check arm is implemented, stubbed or dropped from the report; if any trigger's
inertness changes, stop.

Build SC-8's choreography as a world and a hand-built script, not from a map: two one-shot triggers,
the first handing a group of three to one player, the second handing one of those three to another,
each gated on a condition this build already evaluates.

Done when: the group arm writes every member including a dead one and nothing outside the group; the
unit arm writes its one entity and nothing on a miss; each no-reference case leaves every owner as it
stood; the report names neither opcode; the differential agrees on all three vectors; and the suite,
the import graph and the determinism-wall scan are clean.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1, FR-2 | SC-1 | T1 |
| FR-3 | SC-2 | T1 |
| FR-6 | SC-3, SC-4 | T1 |
| FR-4 | SC-5, SC-6 | T2 |
| FR-5 | SC-7 | T2 |
| FR-7 | SC-4, SC-8 | T1, T2 |
| AC-1, AC-2 | SC-1 | T1 |
| AC-3 | SC-2 | T1 |
| AC-7 | SC-3 | T1 |
| AC-4 | SC-5 | T2 |
| AC-5 | SC-6 | T2 |
| AC-6 | SC-7 | T2 |
| AC-8, AC-9 | SC-8 | T2 |
| P-3 | SC-3 | T1 |
| P-1, P-2 | SC-5, SC-6 | T2 |
| DD-1, DD-2, DD-3 | SC-1, SC-3 | T1 |
| DD-4, DD-5 | SC-2, SC-3 | T1 |
| DD-10 | SC-4 | T1 |
| DD-6 | SC-5 | T2 |
| DD-7 | SC-6 | T2 |
| DD-8, DD-9 | SC-8 | T2 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above and
neither commit carries a trailer.
