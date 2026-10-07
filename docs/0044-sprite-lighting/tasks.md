# Tasks — the ramp, the blit, and the two front-ends

Legend: **files** what an entry may change — a permission, not a prediction · **fences** what it must
not do · **done when** the observable it leaves behind. All four are *implementation* entries, in
ascending order, each depending only on entries before it. SC-8's mutants are one to an entry,
applied to production code, measured over the whole tree and reverted by the entry that owns one; a
kill is claimed only where it was run.

## T1 — the ramp and the row a sun gives

**files** ADD `pkg/render/terrain/spriteshade.go`, `pkg/render/terrain/spriteshade_test.go`

DD-1, DD-3, DD-4 — FR-1, FR-2, FR-3.

**fences** no existing file is opened and no caller is converted, so this entry moves no pixel
anywhere in the tree. `LevelCount`, `shadeDivisor`, `ShadeChannel`, `ShadeScale` and `Light` are read
and neither edited nor generalised, and the 16 rows are not expressed through the 96-row transform.
No table is allocated, nothing is cached, and no package-level variable appears. The row is derived
from a whole `Light`, and no second entry point lets a caller name a row without one.

**done when** AC-1, AC-3 and AC-4 hold and SC-1 and SC-3 with them, AC-3's terrain side read from the
existing transform rather than restated beside it, and AC-1's independent expression written from
the contract by hand. Then SC-8's numerator mutant is applied, the whole tree run with the failing
tests named, reverted, and the tree confirmed byte-identical.

## T2 — one walk, two rows to index

**files** MODIFY `pkg/render/terrain/blit.go`, `pkg/render/terrain/blit_test.go`;
ADD `pkg/render/terrain/blitlit_test.go`

DD-1, DD-2 — FR-1, FR-4, FR-5.

**fences** the two existing entry points keep their signatures, and their pixels are asserted
unchanged rather than argued. The lit and unshaded cases differ in nothing but which 256-entry row
the loop indexes: no branch, bound test or shading call enters that loop, and the clip stays one
rectangle intersection taken once. No caller is converted — both front-ends still draw unshaded when
this entry lands. Nothing is cached and no row outlives the call that built it. A palette entry's
alpha stays unread.

**done when** AC-2, AC-5, AC-6 and AC-10 hold and SC-2 and SC-4 with them, P-1 and P-2 checked by
comparing whole destinations rather than the painted rectangle. Then SC-8's standalone-image mutant
is applied, the whole tree run with the failing tests named, reverted, and byte-identity confirmed.

## T3 — the raster tool's sun reaches its sprites

**files** MODIFY `cmd/terraintool/main.go`, `cmd/terraintool/statics_test.go`

DD-7 — FR-5, FR-6, FR-7.

**fences** no flag is added, renamed or given a second meaning, and the render descriptor's tokens
keep their order and spelling. The four geometry-and-light combinations stay reachable and the four
compositors are called with the arguments they take today, so an unshaded run's PNG is unchanged
outside sprite pixels. The statics loop stays at scale 1, gains no second pass, and neither the
placement list nor the ground check is opened.

**done when** AC-8 holds and SC-6 with it, each of the three renders pinned at its own row rather
than compared only for inequality. Then SC-8's pinned-sun mutant is applied, the whole tree run with
the failing tests named, reverted, and byte-identity confirmed.

## T4 — the window: both passes, one builder, one key

**files** MODIFY `pkg/ui/statics.go`, `pkg/ui/viewer.go`, `pkg/ui/statics_test.go`;
ADD `pkg/ui/spritelight_test.go`

DD-5, DD-6, DD-8 — FR-2, FR-3, FR-5, FR-6.

**fences** `Lit()` is not edited and gains no clause; the level grid, the vertex colour scales and
every terrain draw are untouched. Both sprite passes reach pixels through the one builder and
neither learns a row of its own; `spriteGeoM` and the mirror are not opened, and no key is widened
for a mirror. No texture is uploaded outside a draw, so a headless run still builds none. The sun
stays the fixed daytime one and the viewer gains no light setter and no second switch.

**done when** AC-7 and AC-9 hold and SC-5 and SC-7 with them, AC-9's third state compared with its
first byte for byte. Then SC-8's narrowed-key mutant is applied, the whole tree run with the failing
tests named, reverted, and byte-identity confirmed.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, FR-2, FR-3, AC-1, AC-3, AC-4 | DD-1, DD-3, DD-4, SC-1, SC-3 |
| T2 | FR-1, FR-4, FR-5, AC-2, AC-5, AC-6, AC-10, P-1, P-2 | DD-1, DD-2, SC-2, SC-4 |
| T3 | FR-5, FR-6, FR-7, AC-8 | DD-7, SC-6 |
| T4 | FR-2, FR-3, FR-5, FR-6, AC-7, AC-9, P-3, P-4 | DD-5, DD-6, DD-8, SC-5, SC-7 |
