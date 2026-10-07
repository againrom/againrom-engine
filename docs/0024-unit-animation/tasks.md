# Tasks — unit animation (idle and move, with facing)

Legend: **files** the task may change · **done when** the observable it must leave behind. Every
entry is an implementation task; they land in ascending order, and every dependency of a task is
a task before it. A task marked *small* is mechanical enough to batch with an adjacent one.

## T1 — `pkg/data`: the units.reg default table

**files** `pkg/data/keys.go`, `pkg/data/defaults_test.go`

DD-1. *Small, mechanical.*

**done when** SC-1 passes (FR-1). `load.go` is not edited; only the two tests DD-1 names
change, every other suite green unedited; the table's rows are exactly DD-1's inventory —
`InMapEditor` row-less, no row `noInherit`; the no-`File` case asserts through `SpritePath()`
on a class whose chain never sets the key.

## T2 — `pkg/data`: the descriptor

**files** new `pkg/data/anim.go`, new `pkg/data/anim_test.go`

DD-2. *Small.*

**done when** SC-2 passes (FR-2). `Anim()` performs no IO, reads no sheet, names no float; a
fixture whose `MovePhases` exceeds its pair's expanded length keeps the track's period — a
`Phases` scalar is never a length; a sentinel `Dying` changes nothing; every expected base,
stride, total and track is a hand-written literal, the spec's example class among them.

## T3 — `pkg/render/terrain`: the selector

**files** new `pkg/render/terrain/unitanim.go`, new `pkg/render/terrain/unitanim_test.go`

DD-3 — `terrain.UnitAnim`, the mirror type, and `SelectUnitFrame`.

**done when** SC-3 passes (FR-3, P-1). No import enters the package; the matrix covers both
layouts, 8 octants and both states, plus a tick run over two ids; every expectation is a
literal, the boundary octants matching DD-3's tables verbatim; the guard cases include an
over-reaching index, `frameCount` 0 and a negative tick, none panicking.

## T4 — the bundle: whole sheets, descriptor aboard, mirror-carrying placement

**files** `pkg/render/terrain/units.go`, `pkg/render/terrain/statics.go` (the `Mirror` field
alone), their tests; `pkg/game/statics.go` (the `sheetCache` memo), `pkg/game/units.go`,
`pkg/game/census.go`, `pkg/game/world.go`, `pkg/ui/overlay.go`, the tests the shape change
reaches; `internal/synth` if a fixture widens

DD-4, and DD-6's type halves: `Frames`/`Anim`, `UnitPlace`'s new signature,
`StaticPlacement.Mirror`, `MapEntity`'s new shape.

**done when** SC-4 passes (FR-5) and the whole suite is green with `entityDraws` handing
`Frames[0]` unmirrored — 0022's selection, an explicit interim T5 replaces. `LoadStatics` and
`frame` are not edited; the census keys off `len(Frames)`; the two `-check` cases run; no
ebiten import enters `terrain` or `game`.

## T5 — `pkg/game`: the resolution seam

**files** `pkg/game/world.go`, `pkg/game/world_test.go`

DD-5, and DD-7's digest witness.

**done when** SC-5 and SC-7 pass (FR-4, FR-6's fallback chain, FR-7's digest half, P-4). The
scene tick and facing memory live on `mapWorld` alone and drop with it; `pkg/sim` is not
edited in any byte — its tests, scan and DAG unedited; selection is reached only through
`SelectUnitFrame`; the driven digest test asserts sprites held mid-run and compares at every
k; AC-4's three seam cases and AC-5's independent memories are covered.

## T6 — `pkg/ui`: the mirrored draw and the negative witnesses

**files** `pkg/ui/statics.go`, `pkg/ui/viewer.go` (the sprite paint), `pkg/ui` tests

DD-6's draw half, and DD-7's P-3 witness.

**done when** SC-6 and SC-8 pass (FR-6, P-2, P-3). The mirror bit rides `staticScreenRect`
from the placement; the reflection is GeoM-only — `staticImage` is not edited and no new
cache appears; the square pass, glyphs, colours and pass order are unchanged; `cmd/mapview`
is not edited; the nil-bundle comparison runs at tick 0 and after k driven ticks.

## T7 — the corpus instrument

**files** new `pkg/game/animaudit.go`, new `pkg/game/animaudit_test.go`;
`cmd/terraintool/main.go`, new `cmd/terraintool/unitanim_test.go`

DD-7's instrument. *Small.*

**done when** SC-9's unit half passes (FR-2, FR-3's corpus reach). The audit sweeps both
states, 8 octants and every track step per class, reporting predicted total, frame count,
in-range and guarded; `render`'s and `units`' flags and output are unchanged; the verb exits
non-zero only on a load failure; no install path in source.

## T8 — `pkg/data`: an absent phase contributes no block

**files** `pkg/data/anim.go`, `pkg/data/anim_test.go`

DD-2's clamp. *Small.*

**done when** SC-2's clamp half passes (FR-2). The clamp sits where each scalar enters a base, a
stride or the total and nowhere else: the gates keep the resolved scalar and answer exactly as
they did. A synthetic class resolving `-1` for a phase derives, field for field, the descriptor
its 0-resolving twin derives; the bases and totals either side of an absent phase are
hand-written literals, never read back from `Anim()`, and no fixture carries a value read off an
install. The selector, the mirror type and the loader are not edited.

## T9 — `pkg/game`: the paced advance

**files** `pkg/game/world.go`, `pkg/game/frontend.go`, `pkg/game/world_test.go`

FR-8, DD-8.

**done when** SC-10 passes (FR-8). `paceTo` takes its `now` as an argument, so the whole
criterion runs off hand-written timestamps and reads no clock; the elapsed runs, the baseline
call and the stall are literals. `tick()` is not edited, and neither is `pkg/sim`, `pkg/ui` or
`pkg/render` in any byte — 0020's counting witness passes unedited, which is the point of the
signature staying as it is. The digest half drives the paced path against direct `tick()` calls.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1 | DD-1, SC-1 |
| T2 | FR-2 | DD-2, SC-2 |
| T3 | FR-3 | DD-3, SC-3 |
| T4 | FR-5 | DD-4, DD-6, SC-4 |
| T5 | FR-4, FR-6, FR-7 | DD-5, DD-7, SC-5, SC-7 |
| T6 | FR-6, FR-7 | DD-6, DD-7, SC-6, SC-8 |
| T7 | FR-2, FR-3 | DD-7, SC-9 |
| T8 | FR-2 | DD-2, SC-2 |
| T9 | FR-8 | DD-8, SC-10 |
