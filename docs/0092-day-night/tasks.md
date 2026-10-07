# Tasks — 0092 day and night

Legend: **kind** is `impl` (one commit). Every entry states its `Done when:`.

## T1 — the sun model `impl`

Covers FR-1, FR-2, FR-3, FR-4, FR-5, FR-7, DD-1, DD-2, DD-9.
Files: `pkg/render/terrain/sun.go` (new), `pkg/render/terrain/sun_test.go` (new),
`pkg/render/terrain/light.go` (comment only).

Pure functions over integers and one bool; nothing holds a value between calls. The switch-off arm
returns the existing daytime `Light` rather than rebuilding one.

Scope fence: no viewer, no world, no draw path. `DefaultDaytime` and `DefaultTheta` do not move and
keep their names. Two sentences of `DefaultTheta`'s comment are corrected in the same commit —
they assert the retracted double-width sweep and call the switch's default an open question, and
both are this task's own input.

**Done when:** the endpoint table and the band census are pinned as literals; the four constants'
bit patterns are pinned **on the package's own identifiers**, not on the test's copies; `720 x` the
day step and the threefold relation are asserted separately; the switch-off arm equals the existing
daytime `Light` whole at every minute; and the predicate fires 72 times per in-game day, with
neither half of its gate alone reproducing that.
`go build ./... && go vet ./... && go test -count=1 -trimpath ./...` green.

## T2 — the cache, the switch, the offset, and the one relight `impl`

Covers FR-6, FR-7, FR-8, FR-9, FR-10, FR-11, DD-3, DD-3a, DD-4, DD-5, DD-6, DD-10.
Files: `pkg/ui/viewer.go`, `pkg/ui/statics.go`, `pkg/ui/daynight_test.go` (new).

The viewer's own sun, seeded with the value the constructor already used, and the single place a
relight decides, rebuilds and writes. The sprite layer's two reads of the package-level daytime
light are repointed at it, and the comment saying the viewer holds no light is corrected.

Scope fence: no key is bound here and no front-end is opened. The sprite texture cache's key is NOT
widened — neither of its inputs moves under FR-5, and that cost belongs to whoever closes D-1.
`cornerScales` and the tile cache are untouched.

**Done when:** a clock pushed between relight instants leaves the sun where it was and one pushed on
an instant moves it; the switch and the offset each force a relight off-cadence, with the dusk
band's no-op asserted as such; **all four** altitude-length fixtures that distinguish the two guards
stay unlit through repeated relights, the two overlong ones included; 24 hourly steps return both
the sun and the grid; minutes 0, 360 and 720 give three different literal grids while a
single-height grid is 46 at all three; and the sprite row and tint are the cache's. Gates as T1.

## T3 — the world's tick becomes the lighting clock `impl`

Covers FR-1, FR-12, DD-8, DD-11.
Files: `pkg/game/world.go`, `pkg/game/daynight_test.go` (new).

One push, INSIDE the per-tick entity push rather than beside its two call sites, carrying the
world's own tick as sub-ticks.

Scope fence: no simulation type gains a field, no byte form is opened and no version is moved.
Nothing in `pkg/sim` is edited at all — the two pins that witness that are read, never touched. The
paced advance, the cadence and the readout are not touched.

**Done when:** a world opened and never ticked has already pushed once, so the map is lit for its
opening minute rather than for the seed; the clock the viewer receives after N ticks is N; the
number of ticks one paced call can run is shown to be below the relight period at every rung of the
cadence ladder, so no frame can fire two relights; and a run with the switch moved and the offset
stepped produces the same digest and the same byte form as one without. Gates as T1.

## T4 — the two keys `impl`

Covers FR-9, FR-11, FR-13, DD-7.
Files: `pkg/ui/app.go`, `pkg/ui/daynight_input_test.go` (new).

`N` moves the switch; `F3` steps the offset by one in-game hour. Both are press edges, both are read
on the map arm alone. The front-end's authored letter/function-key rule is narrowed in the same
commit to cover only bindings that are ours, because `N` is a decoded binding that changes only what
is drawn and is otherwise a silent counterexample to it.

Scope fence: no other binding moves, and neither key does anything on the menu or picker screen.

**Done when:** a snapshot naming neither key changes nothing; `N` flips the switch once per press
and not per frame held; `F3` advances the drawn sun and leaves the world's tick and digest where
they were; both do nothing on every screen but the map; and the whole suite is green. Gates as T1.
