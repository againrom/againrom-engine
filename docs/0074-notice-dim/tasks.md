# Tasks — the dim behind a notice is the one the original applies

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 the dim's strength

Kind: impl. Carries FR-1, FR-2, FR-3, DD-1, DD-2, DD-3, DD-4, DD-5, R-1, R-2, R-3.
Criteria SC-1, SC-2, SC-3, SC-4.
Files: `pkg/ui/notice.go` MODIFY, `pkg/ui/noticedim_test.go` ADD.

Boundary: the value one function returns, and the assertions that pin it. The comment above that
function is part of this commit rather than a follow-up — it is where R-1 is answered.

Scope fence: `Viewer.Draw` is not edited, the frame decision is not edited, the setter and the field
are not edited, and no existing test is rewritten. If the composition order, the rectangle or the
gate has to move, the boundary was crossed and this task stops. Nothing under `pkg/sim` is touched
and no serialized version is taken.

Done when: the shipped value is the closest strength its type can carry, witnessed by a search rather
than by a literal; its residual is inside the stated bound; every assertion the previous story made
about the dim still holds untouched; and the whole gate is clean.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1 | SC-1 | T1 |
| FR-2 | SC-2 | T1 |
| FR-3 | SC-3, SC-4 | T1 |
| AC-1, AC-4, P-1 | SC-1 | T1 |
| AC-2, AC-3 | SC-2 | T1 |
| AC-5, AC-6 | SC-3 | T1 |
| P-2 | SC-4 | T1 |
| DD-1, DD-3 | SC-1, SC-3 | T1 |
| DD-2 | SC-2 | T1 |
| DD-4 | SC-2 | T1 |
| DD-5 | SC-1, SC-2 | T1 |
| R-1 | SC-2 | T1 |
| R-2 | SC-3 | T1 |
| R-3 | SC-2 | T1 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above
and neither commit carries a trailer.
