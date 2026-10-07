# 0100 — night is dark: plan

## DD-1 — the schedule gets its own file, beside `sun.go` and not inside it

`pkg/render/terrain/skylight.go`. `sun.go` owns *where the sun stands* — a clock, four bands and two
angle constants whose relations it asserts against literals. The schedule is *what the light is* —
seven arms, thirty-five values and a ramp. They share only the band selector, and merging them would
put two unrelated sets of literals under one file comment that could no longer describe either.

`SunAt` (FR-4) stays in `sun.go` and becomes the one place they join: `SkyLight` for the four
parameters, `SunAngle` for the fifth.

## DD-2 — seven arms as code, not a table

`TERR-LIGHT-120` is explicit that the routine reads no table: every value is an immediate in the
instruction stream or the output of a divide. A `[7][5]uint8` array here would be a shape the
original does not have, and it would make the two ramp arms per twilight — which differ in their
*formulas*, not in their entries — unrepresentable without a second mechanism beside it.

So: a `switch` on the hour with one arm per band, each arm building a `Light` from its own decimal
literals (FR-2). `q` is one two-argument helper, `func ramp(k, m int) int { return k * m / 120 }`,
and every `k` is passed at the call site. Nothing is spelled `3 * daySunStep`-style: the relations
between arms are what AC-3 and AC-4 check, and a derived constant turns a check into a tautology —
the rule `sun.go`'s own file comment states and this file inherits.

The ramp phase is `minute % 120` after the modulo-1440 reduction (FR-2, P-1). That reduction is
lossless for both the band and the phase because 60 and 120 both divide 1440, and the argument's
`+360` is a multiple of both — which is why the phase is written on the raw minute here while the
band is written on `minute + 360`.

## DD-3 — `Light` gains the two shroud fields, `DefaultDaytime` gains their cycle-off values

`ShroudObject` and `ShroudUnit`, `uint8`, on `Light` (FR-5). `DefaultDaytime` takes 4 and 2 —
`TERR-LIGHT-119`'s cycle-off arm — so a `Light` built from the constant and one built from
`SkyLight`'s cycle-off arm remain equal, which is what AC-3 compares.

Widening a struct that several packages construct by literal is the risk here, and it is contained
by the field being additive with a zero value that is never read: `LightFromFields` and every test
literal keep compiling and keep meaning what they meant. `SkyLight` and `SunAt` are the only writers
of a non-zero value, and no drawing reads either field — FR-5's "nothing reads them" is therefore a
property of there being no reader in the tree, checkable by grep, not a discipline.

## DD-4 — the ground tint rides the texture, the level stays on the vertex

The viewer draws terrain with `DrawTriangles` and per-corner vertex colour multipliers. A vertex
colour multiplies; it cannot add. The decoded transform is

    out = clamp( (chan + tint) * (96 - level) / 32, 0, 255 )

and it factors exactly into a texture holding `chan + tint` and a multiplier of `(96 - level)/32` —
which is precisely `ShadeScale(level)`, already what `withScales` applies. So FR-6 needs no new pass
and no shader: build the cell texture through `clamp(chan + tint, 0, 255)` and change nothing else.

That clamp is where D-2 comes from and it is the only place the two disagree. `ShadeChannel(ch,
tint, 64)` is `(ch + tint) * 32 / 32`, i.e. the tint applied at the identity row, so the tinted
texture is expressible in the shipped transform at its own identity level rather than in a new one.
The alternative — dropping the vertex path for a CPU composite per relight — was rejected on cost:
it is the whole map's pixels 72 times an in-game day, against one texture per resolved cell per
band.

Nothing about the *unshaded* path changes (FR-9): a diagnostic is not terrain, and `Lit()` already
decides it one place, so the tint the cache key carries is zero whenever `Lit()` is false. The
placeholder fill is built by its own function that never sees a tint.

## DD-5 — both caches take the tint into the key

The sprite cache is keyed `{frame, row}` and bakes the tint in without it — 0092 DD-3a names the
consequence and defers it to the story that makes the tint vary. This is that story, so the key
gains `tint [3]uint8` (FR-7). The ground cache is keyed `{slot, sub}` and gains the same field
(FR-6).

Keying rather than dropping, for both, on the reasoning 0044 already recorded for the row: an
invalidation is a second thing to keep true and is invisible at the call site. The cost this buys is
bounded and worth stating — `TERR-LIGHT-128` measures **24** distinct `(R,G,B,ambient,range)` tuples
reachable through the unforced relight cadence, so both caches are bounded by 24 entries per cell or
frame in the worst case, and by one in the common one, since day and cycle-off both carry a zero
tint and eleven of a day's twenty-four hours are day. The forced path (the diagnostic hour step) can
reach any of 170 and is a developer's key, not a session's.

## DD-6 — `SunAt` stops reading `DefaultDaytime`, and the agreement becomes a test

Today `SunAt` copies `DefaultDaytime` so that "with the cycle off this is the light the tree drew
before" is an identity. Under FR-4 it composes two decoded functions instead, and that identity
becomes an *agreement between two readings* — which is the stronger form and the one AC-3 asserts.
The constant stays where it is: it is still the seed a viewer holds before its first relight, and
still what `cmd/terraintool` lights a still frame with.

## DD-7 — AC-5 is measured on a level, not on a byte

The assertion the owner would make is that a night pixel is darker, so it is made on the level a
pixel actually gets: a flat synthetic height field through `LevelGrid` at `SunAt(noon, true)` and at
`SunAt(midnight, true)`, then `ShadeScale` of the result. 46 and 64, 1.5625 and 1.0. Both come out
of arithmetic this tree already shipped, driven by the new schedule — no new transform is introduced
to make the number true.

## Success criteria

- **SC-1** `go build ./...`, `go vet ./...`, `gofmt -l` over tracked Go files, and
  `go test -trimpath -count=1 ./...` are all clean.
- **SC-2** `check-no-game-assets.sh`, `check-doc-budget.sh` and `check-sdd-audit.sh` exit 0 with an
  empty FAIL set.
- **SC-3** The continuity property holds over all 1440 minutes, as a test that fails on a
  transcription slip in any of the thirty-five values.
- **SC-4** `againrom -check` prints the same line as before this story on both the ru and the en
  root.
- **SC-5** The mission-10 census ends `lost at tick 272` with 2 movers on both roots.
- **SC-6** Flat ground levels 46 at noon and 64 at midnight, multipliers 1.5625 and 1.0; the sprite
  row runs 3 to 8 and takes all six values.
- **SC-7** `builds/0100-night-is-dark/` ships a binary and a README whose commands were run from that
  directory.

## Traceability

| FR | DD | SC |
|---|---|---|
| FR-1 | DD-1, DD-2 | SC-1, SC-3 |
| FR-2 | DD-2 | SC-3 |
| FR-3 | DD-2, DD-6 | SC-3 |
| FR-4 | DD-1, DD-6 | SC-6 |
| FR-5 | DD-3 | SC-1 |
| FR-6 | DD-4, DD-5 | SC-1, SC-7 |
| FR-7 | DD-5 | SC-1, SC-7 |
| FR-8 | DD-4 | SC-3 |
| FR-9 | DD-4 | SC-1 |
| FR-10 | DD-5 | SC-7 |
| FR-11 | DD-1 | SC-4, SC-5 |
