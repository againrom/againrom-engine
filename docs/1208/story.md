# 1208 — internal/archtest's two ratchets become type-aware

## Result

Both ratchets now ask `go/types` what an expression's real type is, instead of
matching how the source happens to spell it. Both numbers move as a
measurement correction on the same tree, with no structural change beneath
either.

| ratchet | population | before | after |
|---|---|---|---|
| `CommandLiteral` | production literals outside `pkg/sim/command.go` | 0 | 0 |
| `CommandLiteral` | test literals (the ratcheted count) | 867 | 936 |
| `Coordinators` | field count (`Fields`, unchanged) | 81 | 81 |
| `Coordinators` | coordination points | 20 files | 18 functions |

867 to 936 is the undercount the prior walk carried: it resolved an elided
element type one level deep, so it missed doubly nested elision, and all 69
missed literals are that form. The prior lane's adversarial review computed
936 for the same tree with a type-checked tool; this story's walk reproduces
it.

20 to 18 is not "two coordinators fixed". The unit changes from the FILE to
the FUNCTION (see As built), which drops seven prior file entries that were
unions of individually-innocent methods, adds one the old walk could not see
at all (a real local-alias blind spot, not a probe — see Proof), and
thirteen carry over. `Fields` does not move; this story does not touch
the struct.

## Intent

Both ratchets landed as syntactic walks and were reviewed as such.
`internal/archtest/commandliteral.go`'s own doc comment said so outright, and
`internal/archtest/composition.go`'s said the coordination count was "a lower
bound rather than a census". Both reviews found the gap was not theoretical:
`CommandLiteral` undercounted the real test population by 69 literals across
eleven files, and `Coordinators` could be moved a point in either direction by
a plain local variable, `same := f` to lower it or `local := fe` to raise it
invisibly, neither one touching any structure the guard is meant to watch.

The fix is not a bigger syntax table. `go/types` already resolves every one of
these forms as a side effect of type-checking the tree the compiler will
build anyway; the walk needed to ask it the question rather than reimplement
its answer one case at a time.

## As built

### The shared type-check

`internal/archtest/typecheck.go` builds a `TypeCheckedModule` once per
process: `go list -export -json -deps ./...` gives every package's `Dir`,
`GoFiles`, `TestGoFiles`, `XTestGoFiles` and, for every dependency including
the standard library, a path to its compiled export data. Every import -
whether of a third-party package or of this module's own other packages - is
resolved from that export data through `go/importer`'s `"gc"` compiler mode
with a custom `Lookup`, as `go build` resolves a production dependency. An
external test that uses a name an internal `_test.go` file exports (the
`export_test.go` idiom) therefore fails to check, loudly; the tree has no
such file. `unsafe` is special-cased directly to `types.Unsafe`, since
`go list` reports no export data for it. A handful of import paths only a
test file reaches (`testing`, `os/user`) are not in the bulk `-deps` graph and
are resolved lazily, one `go list` call each, the first time they are needed.

Each of this module's own packages is then type-checked with its own
`GoFiles` + `TestGoFiles` as one package (the internal test augmentation) and,
if the directory has one, its `XTestGoFiles` as a second, external test
package - both against the same shared importer, the same `token.FileSet`.
No `golang.org/x/tools` dependency was added; `go.mod` is unchanged.

`LoadCommandLiterals` and `LoadComposition` both call
`loadTypeCheckedModule`, which caches the built module per module root for
the life of the process. `LoadComposition` only needs `pkg/game`, which
`LoadCommandLiterals` visits anyway as part of the whole module, so within one
`go test` run the second loader's call is a cache hit - this is the caching
split the brief asked for, done by sharing the result rather than by adding a
second cache key.

### `CommandLiteral`

`isCommandType(t types.Type) bool` unwraps `types.Unalias` and one pointer,
the type an elided `&Command{}` element carries, and checks the resulting
`*types.Named` against `(pkg/sim, "Command")`. The walk visits
every `*ast.CompositeLit` in every file and asks `info.Types[lit].Type` -
that single question. `go/types` already resolves the type of an elided
composite literal at every nesting depth, as a value or as a key, and already
resolves a dot-imported or aliased name to the same named type a qualified
reference would produce, so none of those forms needs its own AST case any
more. The ~70 lines that used to track container element types, unaliased
imports, and one level of elision are gone.

What the walk cannot see is a file `go list` does not select for this build:
another GOOS/GOARCH (5 production files here), a custom build tag (34 test
files, which no gate measures), `//go:build ignore`, and `testdata/` or
`_`-prefixed directories. None of those files holds a Command literal today. A
type parameter, or a conversion from another struct type, is not a Command
literal to it either.

### `Coordinators`

`isFrontEndType(t types.Type) bool` unwraps a pointer and any alias and
checks the result against `(pkg/game, "FrontEnd")`. For every non-test
`*ast.FuncDecl` in `pkg/game` the body is
inspected for `*ast.SelectorExpr` nodes whose selector name is an
owned-component field, and `isFrontEndType(info.Types[sel.X].Type)` decides
whether the reach counts. That one question replaces three separate syntactic
cases the prior walk needed (a receiver or parameter identifier, and a
package-declared struct field known in advance to hold a `*FrontEnd`): a
receiver, a parameter, a local alias at any depth, and a struct-field chain
ending in one that holds a `FrontEnd` are all just expressions with a type,
and `go/types` answers the same way for all of them.

**The unit is the FUNCTION, not the file.** The prior walk unioned every
selector across a whole file, so a file with three methods that each touch
one component read as a single coordinator - concentration nothing in that
file actually has. Scoring each `*ast.FuncDecl` on its own body says exactly
what the ratchet's own name claims: a function that reaches three or more
components AT ONCE. `FileComponents` gains a `Func` field for this; `File`
alone is no longer a unique key, since `frontend.go` holds four
coordinating functions and `gameoptions.go` two.

What the walk cannot see: a package-level func literal, a struct embedding
`*FrontEnd` reached by promotion, and a reach spelled through a component
itself. `pc := &f.PersistenceContext` or `f.PersistenceContext.hallStore`
each lower the count from 18 to 17 with no structural change. A field is not
reachable through an interface without an assertion, and an assertion is
counted.

## Proof

### The counts, measured on this tree

`go run ./internal/archtest/cmd/commandliteral`: `Tests: 936`, zero
`production` lines. `go run ./internal/archtest/cmd/composition`: `Fields: 81`,
`Coordinators: 18`, over 14 files - `frontend.go` holds four (`App`,
`arriveInTown`, `loadMap`, `missionOpenerMode`), `gameoptions.go` two
(`gameOptionValues`, `setGameOption`), and twelve files one each.

Compared against the same tool built from the pre-story committed
`composition.go`/`cmd/composition/main.go` (temporarily restored, run, then
restored back to this story's version, confirmed by an empty `git status
--porcelain` after): the OLD walk reports 20 files, of which seven drop out
under the new per-function walk (`docsart.go`, `sound_channels.go`,
`speech.go`, `speech_witness.go`, `townexterior.go`, `townscreen.go`,
`worldmap.go` - each a union of methods no one of which alone reaches three
components) and one is newly visible: `pkg/game/originalsave.go`'s
`RestoreOriginal`, which copies the receiver to a local value, `draft := *f`,
partway through the function, and reaches `InstallResources`,
`CampaignSession` and `Presentation` through `draft.*` afterward - invisible
to the old walk's receiver/parameter-only tracking, on the UNMODIFIED tree,
with no probe needed to show it.

### The five missed `CommandLiteral` forms

Each was added as a temporary file under `pkg/game`, shown failing
`TestLiveCommandLiteralsMatchTheirBaseline` by naming its file and line, then
removed (`git status --porcelain pkg/game/` printed nothing after each):

| form | probe | result |
|---|---|---|
| elided map key | `map[sim.Command]bool{{Kind: sim.KindMoveTo}: true}` | `zzprobe1208.go:6 writes a sim.Command composite literal` |
| doubly nested elision | `[][]sim.Command{{{Kind: sim.KindMoveTo}}}` | `zzprobe1208.go:7 writes...` (the innermost literal's own line) |
| named slice type | `type zzprobe1208List []sim.Command; zzprobe1208List{{Kind: ...}}` | `zzprobe1208.go:8 writes...` |
| type alias | `type zzprobe1208Alias = sim.Command; zzprobe1208Alias{Kind: ...}` | `zzprobe1208.go:7 writes...` |
| dot import | `import . "againrom/pkg/sim"`; `Command{Kind: KindMoveTo}` | `zzprobe1208.go:5 writes...` |

A clean tree passes `TestLiveCommandLiteralsMatchTheirBaseline` after every
probe was removed.

### The `Coordinators` dodges, both directions plus the unit question

All three probes were applied to the working tree, run, and reverted
(`git status --porcelain pkg/game/` empty after each):

- **Downward (N2, `same := f`).** Both occurrences of `store := f.hallStore`
  in `pkg/game/ending.go` (lines 51 and 69) rewritten to `same := f` then
  `store := same.hallStore`. `Coordinators` stayed at 18 and
  `campaignEnding` still listed all three components including
  `PersistenceContext` - the old walk dropped this file's count from 20 to
  19 for the identical edit.
- **Upward-hiding (N3, `local := fe`).** A temporary file with
  `func zzprobe1208FourComponents(fe *FrontEnd) int { local := fe; ... }`
  touching `Maps`, `Carried`, `tipsOff` and `showPathfinding` through `local`.
  `Coordinators` rose from 18 to 19, the new function named with all four
  components, and `TestLiveCompositionMatchesItsBaseline` failed naming the
  rise. The old walk left this file absent from the listing at an unchanged
  20.
- **Per-method vs per-file.** A temporary file with three methods, each
  touching exactly one distinct component (`Maps`, `Carried`, `tipsOff`).
  Under this story's walk `Coordinators` stayed at 18 and the file did not
  appear. Rebuilding the OLD (pre-story) tool against the identical probe
  file reports `Coordinators: 21` with the file listed, unioning the three
  single-component methods into a false three-component file.

### Runtime

`go test -count=1 ./internal/archtest/...`, three runs each, GOCACHE warm
(`.gocache` already populated by ordinary work in this tree; a cold cache was
not separately measured):

| tree | package-reported time | wall time (`time`) |
|---|---|---|
| before (pre-story, syntactic) | 2.07s / 2.09s / 2.07s | 2.5s-2.8s |
| after (this story, type-aware, shared pass) | 3.65s / 3.55s / 3.60s | 4.1s-4.2s |

About 1.5s slower, roughly a 1.6x factor - not the roughly-doubled cost a
naive unshared implementation would pay, because `LoadComposition`'s call
inside the same `go test` process hits `loadTypeCheckedModule`'s cache
instead of repeating `go list -export -deps ./...` and re-checking every
package. The standalone `go run` tools (`cmd/commandliteral`,
`cmd/composition`) do not share a process and each pay the full pass alone.

The adversarial review measured more: warm, 3.50 / 3.95 / 4.68 s package
time against base's 2.05 / 2.09 / 2.12 s, a 1.7x to 2.2x factor, and 8.4 s for
`go run ./internal/archtest/cmd/commandliteral`. Cold, in a scratch GOCACHE:
18.1 s package, 23.8 s wall and a 223 MB cache, against base's 2.3 s, 10.2 s
and 58 MB.

### Gates

- `gofmt -l .` (excluding the `knowledge` gitlink): clean.
- `go test -trimpath -count=1 ./...`: exit 0, 54 packages report `ok`, none
  `FAIL`; `real 0m44.3s`.
- `go vet ./internal/archtest/... ./internal/storyguard/...`: clean.
- `bash scripts/check-no-game-assets.sh`: `clean (tree scan)`.
- `internal/storyguard`: `CommentBytes` rises from 7659424 to 7665402
  (+5978), over the six files `internal/storyguard/baseline.go`'s own note
  names - five doc comments on the two ratchets turning type-aware
  (`internal/archtest/typecheck.go` new, the other four rewritten function
  and baseline docs) and the note itself. All six comment forms
  (`specclause`, `storymention`, `calendardate`, `rom1address`, `funaddr`,
  `expmention`) stay at zero; an early draft of both baseline notes named the
  prior lane by its own story number and had to be reworded once
  `storymention` caught it.
- Milestone census (`AGAINROM_ASSETS=<en root> missionrun -mission N -trace
  -ticks 1 | grep -c UNSUPPORTED`): mission 10 = 0, mission 20 = 0 -
  unchanged from `pipeline/milestone-baseline.txt`'s prior lane. This story
  touches no simulation or script code, so the census was not expected to
  move and did not.

Release tests, scenarios and `check-milestone2-acceptance.sh` were not run:
nothing here reaches a screen, shipped data or a save, per the brief.

## Open debt

- **The literal walk's build selection.** A file `go list` does not select
  for this build is not checked (see As built). A tagged pass, or a syntactic
  pass over ignored files, would close it.
- **The coordination dodges that remain.** A reach spelled through a
  component, a package-level func literal and a promoted `*FrontEnd`
  embedding. Counting a selector whose base is an owned component type, or
  reading `types.Selections` index paths, closes the first.
- **936 test literals remain ratcheted, not migrated**, same as before this
  story; migrating them to constructors is still separately scoped debt.
- **The `export_test.go` idiom fails both instruments** with an undefined
  name, because an external test is checked against plain export data rather
  than the test-augmented package. The tree has no such file.
