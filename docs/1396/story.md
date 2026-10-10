# One composer scene runtime

## Intent

The town square and every room page run on one runtime, `town.Scene`, over one
host interface, `town.Host`, from art one manifest loader, `town.LoadArt`,
resolves, through one game adapter, `townSceneHost`. A square program runs in a
room page and a room program runs on the square without new code. Nothing a
player sees or a save holds changes. No CHANGELOG line: nothing is
player-visible. Base: public main `612a359f`, knowledge pin unchanged.

## Authority

Owner decision 5 on the architecture audit
`pipeline/reviews/arch-audit-34ae6dac.md`, row 5 (also
`arch-audit-db226edb.md`, row 5): one composer scene runtime for the square
`View` and the room `Page`, before composer step 2. Owner rule "one builder per
kind": the composer is the one builder of a town screen. The present behaviour
is the specification; the town square, town room and generator traces are the
oracle. No ROM1 behaviour changes, so no claim is cited anew and no divergence
row is added; the reserved DIV-2904..2911 are unused.

## As built

### The one runtime, host, loader and adapter

| Kind | One builder | Replaced |
|---|---|---|
| runtime | `town.Scene` (`pkg/town/scene.go`, `draws.go`, `paint.go`) | `town.View` (`view.go`, deleted) and `town.Page` |
| actor model | one `actor` interface over `*Scene`, 14 programs, `still` answers every call a program takes no part in (`actors.go`, `roomactors.go`) | `actor` over `*View` and `pageActor` over `*Page` |
| host interface | `town.Host` | `town.Host` (square) and `town.PageHost` |
| validator | `Description.Validate` with one `validateScene` for the square and every room (`validate.go`) | `validate.go` and `validatescene.go` (deleted) |
| manifest loader | `town.LoadArt(*SceneSpec, Loader)` | `town.LoadArt(*Description, Loader)` and `town.LoadSceneArt` |
| game adapter | `townSceneHost{t, room}` (`pkg/game/townscenehost.go`), one type bound to its room by name | `townSquareHost` and `roomPageHost` |

`town.NewScene(d, room, host, proc)` builds a scene: the square's name
(`d.Square.Name`) builds the square, a room's name its page, and a room without
a scene answers nil. `Description.SquareScene` answers the description's top
level as a `SceneSpec`; `Description.Scene(room)` answers either.

`internal/archtest/composerscene.go` is the ratchet: over production code under
`pkg/` it finds every pkg/town type with both `Advance` and `Paint`, every
pkg/town interface with an `Art` method, every pkg/town function answering
`*Art`, and every type elsewhere with `Art() *town.Art`. Each kind must be
exactly `Scene`, `Host`, `LoadArt` and `townSceneHost`; a second one or a
missing one fails `TestComposerScenesHaveOneRuntime`.

### The description

`towns/rom1.json` is unchanged. Its top level is the square's scene in a short
form; `SquareScene` reads it as one process clock named by the square
(`period-ms`, `compare` from `clock`), three step groups (`step.before`
unbound, `step.admitted` bound to that clock, `step.after` unbound),
`lifecycle: shown` and `advance-when: active`. A room scene states the same
fields itself. `SceneSpec` gained the fields the square held at the top level
(`view`, `mask`, `hotspots`, `pointer`) and three lifecycle fields; `PageClock`
is now `SceneClock` with a `process` field.

### Differences between the two former runtimes

Each difference is a data field of the one runtime or became one rule; none
moved a trace golden.

| Rule | Square (`View`) | Room (`Page`) | Now |
|---|---|---|---|
| clock scope | one clock in `Process`: outlives every reset; while unstamped a paint only stamps it | clocks reset with the page, stamped on entry at now plus `entry-ms` | `SceneClock.Process`; true for the square |
| paint gate | no step while inactive, the whole paint | per group, `when: active` | `SceneSpec.AdvanceWhen`; `active` for the square; groups keep `when` |
| lifecycle | made ready on first activation (actors' `prepare`); entered each time shown, left each time hidden (host `LeaveSquare`, actors' `leave`) | entered by the game (`Enter`) or on first activation; a pause leaves it entered | `SceneSpec.Lifecycle`; `shown` for the square, empty for rooms |
| step structure | `before`, `admitted` on the clock, `after` | ordered groups, each optionally bound to a clock | groups; the square's short form reads as three |
| sound request | key given by the request (op, cue, stepper sound) | the slot's own key, `loop`, `untracked` | one `sound(slot, key)`: an empty key takes the slot's `key`; `loop` and `untracked` are slot data |
| draws | host sources and description generators (`lcg`) with `scaled`/`masked` forms | host sources with `raw`, `divide`, `times` | one draw path; `PickSpec` evaluates both forms in either scene |
| page entry reseed | none | `PageHost.Reseed` per host source; its one host did nothing | removed from `Host` |
| pause | slots, hover and entry loop cleared on the change to inactive | slots stopped on every inactive call | every inactive call stops slots, clears the hover and stops the loop. On the square these are already clear while it is inactive: it requests sound only while active |
| paint | every layer, with `when` conditions, no clip | one named group, with clip, no conditions | `Paint(dst, group)`; the square's layers are the unnamed group; `when` and `clip` apply to any layer |
| no art | `Advance` skipped; `Paint` drew nothing | `Advance` and `Paint` skipped | both skip |
| validation | square programs only; `lcg`, sprites, masks, indexed art; keyless slots | room programs only; host sources only; picture and series art without index; every slot keyed; layer group required, no conditions | one check for every program and form in any scene; a slot must hold a key only where a request names none (`sound <slot>` steps, state sounds, training slot) |

### The adapter

`townSceneHost` answers each room's art (`TownSquareArt`, the tavern, shop and
school scene art), one draw table (`animation`, `ambient`, `tender`, `idle`,
`training`, each a runtime seam and the session stream it falls back to), the
square's conditions and hooks and the rooms' values. One per-room rule is the
adapter's and not the runtime's: a room page requests no sound while the screen
has no sound device; the square still opens the room audio scope then
(`PlaySound`), as before.

## Proof

| Test or gate | Result |
|---|---|
| `TestReleaseTownSquareTraceIsUnchanged`, `TestReleaseTownRoomTraceIsUnchanged`, `TestReleaseChargenTraceIsUnchanged`, RU then EN | pass; no golden written or moved |
| `go test -run 'TestRelease.*(Town\|Tavern\|Shop\|School\|Square\|Room)' ./pkg/game` (160 listed), RU then EN | 150 pass, 10 skip each (second-game tests and tests needing an absent input); 0 fail |
| `go test -run TestReleaseSecond ./pkg/game` (53 listed), `rom2-ru` then `rom2-en`, `AGAINROM_SECOND_SAVE_OUTPUT` and `AGAINROM_FIRST_ASSETS` set | 52 pass each; `TestReleaseSecondPhysicalSyntheticCityAppRoute` is a first-game test and passes on RU and EN |
| `TestEveryProgramRunsOnTheSquareAndInARoomPage` (`pkg/town/scene_test.go`) | pass; does not build on the base (no `NewScene`) |
| `TestComposerScenesHaveOneRuntime` and its synthetic cases | pass |
| ordinary `go test` of `pkg/town`, `pkg/game`, `pkg/ui`, `cmd/againrom`, `internal/archtest`, `internal/storyguard`, `internal/gatedtests`, `internal/divledger` | pass |
| `gofmt -l`, `git diff --check`, `go vet ./pkg/town/... ./pkg/game/...`, `check-no-game-assets.sh`, `check-div-claims.sh` | clean |

## Open debt

- The screen art loader is the next story: the room scenes' art still reaches
  the adapter through `TownTavernArt`, the shop screen art and `TownSchoolArt`.
- Process clocks, flock waits and episode end counters share one name space per
  `Process`; two scenes naming the same process clock share its stamp.
- The page kinds (inn, shop, school widgets) remain game code; composer step 2
  moves the ROM2 town list onto the composer.
