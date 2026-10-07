# Analysis — the object layer's own cycle, and the gate nobody opens

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-first / static** — the profile names rendering work at this tier; no watcher tool exists, so it is discipline |
| Terrain — the object timeline, the phase, the four-corner gate, the animated frame arm | **greenfield**: none of it exists in any spelling |
| Terrain — the class loader, the placement builder, the anchor, the cull, both front-ends' object passes | **brownfield**: every one is shipped and drawn from today, and each is touched here |

## The baseline names a build this tree does not contain

Each sentence was re-derived against the code rather than translated:

| The baseline says | What is actually here |
|---|---|
| the cycle rides `animClock()` / `phaseStep`, "the cadence the unit animation uses" | **neither symbol exists.** There are two counters: `terrain.Ticker.Count()`, advanced from wall-clock microseconds (`pkg/render/terrain/water.go:225`) and read by the water phase (`pkg/ui/viewer.go:1253`), and `mapWorld.scene`, `+1` per map-screen tick (`pkg/game/world.go:406`), read by the unit selector as `scene + id` (`world.go:501`) |
| a GPU cache keyed `objTexKey{classID, frame}` | **no such key.** The window keys `spriteTextureKey{frame *terrain.StaticFrame, row int}` (`pkg/ui/statics.go:233`) — a frame *pointer* and 0044's ramp row. A per-frame pointer already gives a per-frame texture, so no class id is needed |
| the anchor is `ObjectAnchor` and does not depend on `w/h` | **no `ObjectAnchor`.** It is `StaticAnchor` (`pkg/render/terrain/statics.go:140`) and it takes `frameW, frameH` as two of its ten required arguments — the drawn frame's size is *half the anchor* |
| the unit-sprite precedent lives in `unit_sprites.go` | no such file. `pkg/render/terrain/units.go` + `unitanim.go` |
| "byte-identical to 0018", "exactly 0018" (five times) | **there is no `docs/0018-*`.** 0018 was folded into 0017 (`scripts/check-doc-budget.sh`, `select_ceilings` 0017) |
| `docs/0016-data-classes/spec.md` tags the paired arrays `[R]` | **that spec carries no `[R]` tag at all.** It lists `AnimationTime`/`AnimationFrame` as `[]int` keys and asserts no draw-time meaning (`docs/0016-data-classes/spec.md:101`) |
| the existing idiom is a nil sentinel meaning "cache miss => invisible" | there is no such idiom. A nil `StaticClass.Frame` means the placement is **never made** and the cell is counted `NoFrame` (`terrain/statics.go:315`); the tree's one frame selector answers **sheet frame 0** for an out-of-range index (`unitanim.go:110`) |
| `cleandocs/0017-map-objects/spec.md` R-1 is inherited unresolved | our 0017 **removed** R-1 and R-2 as research items and says why: both were answered at High by `TERR-SPR-040` and `TERR-SPR-042` (`docs/0017-map-objects/provenance.md:59`) |

One baseline clause that declines work by pointing elsewhere was checked and **the thing it points at
is really there**: "water-strip terrain animation (already handled separately in the terrain layer)"
— `WaterPhase`/`ResolveAnimated` (`water.go:53`, `:66`), read per cell by the windowed viewer
(`viewer.go:1253`). It stays a non-goal on a true premise.

## The gate the baseline omits, and it decides the story

The baseline's whole animate-gate is "a non-empty `AnimationFrame`/`AnimationTime` pair". That is
**half** of the decoded condition. The other half is a test on the cell's own and its three
neighbours' tile words, and **0 of 880 704 shipped cells satisfy it**, with no writer of those bits
anywhere in the image. Importing the baseline's gate would have made every foliage class sway on
every map, in a renderer whose research says the original draws them still — a fidelity break that
fails no test, because there is nothing on our side to compare against.

So the reachability question is genuinely open on research's side, and the contract must not settle
it by accident in either direction. What this story can do is build the mechanism, gate it as
decoded, and give the gate a **diagnostic override of our own** so the arm is drawable, reviewable
and testable without asserting that the original ever enters it.

## Which counter, and the three that were candidates

The decoded answer is not "a render clock" and not the unit clock: the object arm reads the very
counter the water phase reads, incremented two lines above the unit advance in one pass of the game
tick. In this tree that is `Ticker.Count()` — and it is the one 0041 already re-rated, so taking it
adds no ticker and reopens no cadence decision. It follows that objects keep cycling while the
world is stopped, because 0041 decided the stop does not reach that counter and its `SetRate`
deliberately takes no stop (`pkg/ui/viewer.go:566`). That is inherited, not re-decided.

Two arithmetic traps sit next to each other and were checked apart. Water reduces
`animCtr >> 2` and staggers by `(col+1)*row`; the object arm takes the counter **unshifted** and
staggers by `col*(row+1)`. The two stagger terms are transposes, so a copy-paste between them is
silent on the diagonal and wrong everywhere else.

## What the disable switch actually does to objects

`-noanimation` today reaches water alone: it freezes `Ticker` and forces phase 0
(`viewer.go:746`, `:1252`). The decoded flag is shared — with animations off, **every** static
object draws sheet frame 0 rather than its own `Index`. That is a visible behaviour change this
story ships on shipped data, unlike the animated arm, and it is the only part of the frame rule a
lawful install exercises today.

## Left open for the plan to settle

The engine reads the three neighbour words unguarded, so a cell on the last column or row reads past
its own grid; there is no decoded behaviour there to reproduce. The frame a bad selection falls back
to is likewise ours — the corpus puts 0 of 82 classes outside their sheet, so nothing measures it.
Both are decided in `plan.md`, not here.
