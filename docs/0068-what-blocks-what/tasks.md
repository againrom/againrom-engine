# Tasks — what blocks what, and what stands in front of what

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 the collection that was never loaded

Kind: impl. Carries FR-1, FR-2, FR-3, FR-4, FR-6, FR-7, P-1, P-2, DD-1, DD-2, R-1.
Criteria SC-1, SC-2, SC-3, SC-4, SC-7.
Files: `pkg/game/table.go` MODIFY, `pkg/game/table_test.go` MODIFY, `pkg/game/blocking_test.go` ADD.

Boundary: the front-end's table load and the tests that measure what it now reaches. No world
builder, no plane derivation and no footprint resolver is edited — all three are already correct and
already take the argument.

Scope fence: no file under `pkg/sim`, `pkg/mapload`, `pkg/render` or `pkg/ui` is touched. If one
appears to need touching, DD-1 is wrong and this task stops. The viewer's own grid plane is not
handed a table (FR-6).

The fixture map is one whose ingest plane leaves two open regions separated by a blocked strip, with
a structure over the strip attaching it and closing none of it. Assert reachability **both** ways
round: the second is what gives the first its meaning.

Done when: the loaded table carries the buildings collection with the archive's own rows and an
entry's parameters; the world the front-end hands out for a map screen and the world a started
mission carries both close the set blocking bits and open the clear ones; the crossing appears with
the table and is absent without it; and the whole suite passes with no pinned digest edited.

## T2 units in the plane

Kind: impl. Carries FR-5, FR-7, P-3, DD-3, DD-4, DD-5, DD-6, DD-7, R-2.
Criteria SC-5, SC-6, SC-7.
Files: `pkg/render/terrain/structures.go` MODIFY, `pkg/render/terrain/structures_test.go` MODIFY,
`pkg/ui/statics.go` MODIFY, `pkg/ui/overlay.go` MODIFY, `pkg/ui/viewer.go` MODIFY, the affected
`pkg/ui/*_test.go` MODIFY.

Boundary: the ordering function and the two draw sites that read it. No placement is re-anchored, no
builder output is sorted in place, and no geometry is recomputed.

Scope fence: the instrument passes keep their order and their positions relative to each other
(DD-5). The entity **square** stays in the instrument slice. The blocked tint, the lattice, the three
crosses, the health bars, the selection rim and the marquee are not edited.

Done when: the render tier carries one three-way merge beside the two-way one it already carries; the
window tier paints the merged band through the one cull and the one transform, mirror included; a
recorded target shows a unit before a later-row drawable, after an earlier-row one and after every
flat one; a viewer holding no entities records the art sequence it recorded before; and a viewer
holding no bundle records the entity sequence it recorded before.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1 | SC-1 | T1 |
| FR-2 | SC-2 | T1 |
| FR-3 | SC-3 | T1 |
| FR-4 | SC-4 | T1 |
| FR-5 | SC-5 | T2 |
| FR-6 | — (met by changing nothing; witnessed by T1's scope fence) | T1 |
| FR-7 | SC-7 | T1, T2 |
| AC-1…AC-5 | SC-1…SC-4 | T1 |
| AC-6, AC-7 | SC-5 | T2 |
| AC-8, AC-9 | SC-6 | T2 |
| P-1, P-2 | SC-7 | T1 |
| P-3 | SC-5, SC-6 | T2 |
| DD-1, DD-2 | SC-1…SC-4 | T1 |
| DD-3, DD-4, DD-5, DD-6, DD-7 | SC-5, SC-6 | T2 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above
and neither commit carries a trailer.
