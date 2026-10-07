# Tasks — the group a check counts

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 the identifier reaches the register file's edge

Kind: impl. Carries FR-1, FR-2, FR-5, FR-6, P-3, DD-1, DD-2, DD-3, DD-6, R-2, R-3.
Criteria SC-1, SC-2, SC-3, SC-4.
Files: `pkg/sim/world.go` MODIFY, `pkg/sim/script.go` MODIFY, `pkg/sim/binary.go` MODIFY,
`pkg/sim/scriptbinary.go` MODIFY, `pkg/mapload/fromalm.go` MODIFY, `pkg/mapload/script.go` MODIFY,
and the affected `pkg/sim/*_test.go` and `pkg/mapload/*_test.go` MODIFY.

Boundary: the state and its carriage only. The arm stays outside the supported set, which is what
SC-4 measures and what makes this task independently coherent.

Scope fence: no dispatch arm is added and the supported-arm table is not edited. If a node carrying
a group parameter comes out with different plain parameters than before, DD-2 has been broken and
this task stops. A pinned digest is edited only where a widened record makes it a different world,
and SC-3's differential is what justifies each such edit.

Done when: an entity carries its record's group identifier and one built from no map carries the
zero; a compiled check carries the group its node named, with presence distinguishable from that
zero and no other parameter moved; both widened records cross the byte form and the digest; the
previous version, a truncated record and an out-of-alphabet presence byte are each refused; and the
group arm is still named by the unsupported report.

## T2 the arm counts

Kind: impl. Carries FR-3, FR-4, FR-6, P-1, P-2, DD-4, DD-5, DD-7, R-1.
Criteria SC-5, SC-6, SC-7, SC-8.
Files: `pkg/sim/script.go` MODIFY, `pkg/sim/script_test.go` MODIFY,
`pkg/sim/scriptgroup_test.go` ADD.

Boundary: the one table that decides what this build evaluates, the dispatch arm behind it, and the
witnesses. No record changes width and no loader is edited.

Scope fence: the sack arm is not implemented, not stubbed and not removed from the report — a change
that makes its readers live means the fence was crossed. Nothing outside `pkg/sim` is touched.

Build the win-chain witness as a world and a hand-built script, not from a map: three triggers, the
first gated on the count reaching zero, the second on a unit reaching a cell and incrementing a
variable, the third on that variable and a distance, ending in the win arm. Drive it by killing the
named group's members and moving the unit.

Done when: the arm writes the living-member count and the transition through zero is asserted rather
than a single value; two checks of the arm in one pass write their own registers; a check with no
group reference writes none; a script mixing this arm with the sack arm reports only the sack arm;
the win chain reaches a win; and the suite, the import graph and the determinism-wall scan are clean.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1, FR-2, FR-5 | SC-1, SC-2, SC-3 | T1 |
| FR-3 | SC-5, SC-7 | T2 |
| FR-4 | SC-4 | T2 |
| FR-6 | SC-4, SC-6 | T1, T2 |
| AC-1, AC-2, AC-5 | SC-1, SC-2, SC-3 | T1 |
| AC-3 | SC-5 | T2 |
| AC-4 | SC-4 | T2 |
| AC-6 | SC-6 | T2 |
| AC-7 | SC-7 | T2 |
| P-1, P-2 | SC-8 | T2 |
| P-3 | SC-3 | T1 |
| DD-1, DD-2, DD-3, DD-6 | SC-1, SC-2, SC-3 | T1 |
| DD-4, DD-5, DD-7 | SC-5, SC-6, SC-7 | T2 |
| R-1 | SC-5 | T2 |
| R-2, R-3 | SC-3 | T1 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above
and neither commit carries a trailer.
