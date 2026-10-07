# 0102 — tasks

Two tasks, serial. T1 is the model, T2 the screen pass; T2 compiles only after T1. One commit each,
trailer `SDD-Task: 0102-shadow-geometry/T<n>`. `verification.md` and the build are stages, not tasks.

## T1 — the shadow model

**File:** `pkg/render/terrain/shadow.go` and `pkg/render/terrain/shadow_test.go`. Touch nothing else.

Rewrite the geometry half to spec FR-1..FR-13 and plan DD-1..DD-4. Add `ShadowAngle`,
`ShadowPivotRow`, `ShadowPivotShift`; `ShadowSlope` becomes the tangent of `ShadowAngle`; change
`UnitShadowPlace`'s X term; change `ObjectShadowPlace`'s signature and X term; leave
`StructureShadowShift` and `StructureShadowPlace`'s guards alone. Delete `floorDiv` and
`UnitShadowShift`. Leave the recolour half — `ShadowLevel`, `ShadowChannel`, `ShadowRGBA`,
`ShadowAlpha`, `ShadowMask`, `BlitShadow` — untouched apart from comments naming the wrong pivot.

The header comment discloses two things this story removes: that the sense of the lean is ours,
and that the shear angle's routine was not read. Both go. Cite no claim id.

Tests: AC-1, AC-2, AC-3, AC-4, AC-5, AC-6's numeric half, AC-7, AC-9's first half. AC-1 sweeps
`SunAngle(m, true)` over 1440 minutes. AC-4 builds its strip list by hand — `dstY = row0*CellSize
- (rowTop-k)*CellSize`, `rowTop = row0 - TileHeight + FullHeight`, `k` from `rowTop` down — not
through the placement builder.

`pkg/ui` will not compile until T2. Run `go build ./pkg/render/... && go test -trimpath -count=1
./pkg/render/...` and stop there.

## T2 — the screen pass

**File:** `pkg/ui/shadow.go` and `pkg/ui/shadow_test.go`; `pkg/ui/structures_test.go` only where it
holds a stale constant. Touch nothing else.

Bring the structure and unit arms of `shadowDraws` to the object arm's shape (plan DD-6) so all
three submit a slope and a pivot, and rebuild `drawShadows`' local matrix so the mirror composes
inside it (plan DD-5). `ObjectShadowPlace` now takes `theta` and returns two values, not three.
`shadowWiden` does not change; its comment about serving one caster does.

`structures_test.go` hard-codes `planeStructureShadowShift = 18`, the submitted X of a structure
shadow before this story. Recompute it from the story's own law, not from a run, and say in its
comment which two terms it is now the sum of.

Tests: AC-8. AC-9's `-check`/`missionrun` half is a pipeline stage, not yours. AC-8's mirror clause
is checkable with no graphics context by comparing two `shadowDraw` entries and the matrix
`drawShadows` would build for each; if reaching the matrix needs an export, add an unexported
helper both it and the test call rather than exporting anything.

Gate before committing: `go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go
test -trimpath -count=1 ./...`.

## Traceability

| Task | Requirements | Decisions |
|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10, FR-11, FR-12, FR-13 | DD-1, DD-2, DD-3, DD-4, DD-7, DD-8 |
| T2 | FR-14, FR-15, FR-16 | DD-5, DD-6, DD-7 |
