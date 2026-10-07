# 1206 — FrontEnd becomes a composition root

## Result

`pkg/game.FrontEnd` declares no field of its own. Its fields fell from **100 to
81**, and all 81 now belong to one of five embedded owned components:

| component | fields | what it owns |
|---|---|---|
| `InstallResources` | 33 | everything a lawful install yields, read once and never written |
| `RuntimeServices` | 17 | opened devices, clocks and the bounded presentation generators |
| `CampaignSession` | 11 | the game currently being played |
| `Presentation` | 16 | what is drawn and the state that decides how |
| `PersistenceContext` | 4 | where a preference or record is read from and written to |

Player-visible behaviour is unchanged. This is a refactor: no screen, save,
script census or hashed value moves.

## Intent

Owner direction, after an architecture review he accepted and quoted as his
own priority: FrontEnd had become a god coordinator holding too many
lifecycles, and the right shape is a composition root over roughly
`InstallResources`, `RuntimeServices`, `CampaignSession`, `Presentation` and
`PersistenceContext`, without necessarily one Go package each. It is a
direction, not a big bang; nothing here creates a package, and
`internal/archtest`'s import DAG is untouched.

## As built

### One `lazy[T]` for nineteen hand-written pairs

38 of the 100 fields were two idioms written out by hand 19 times, neither
carrying any ownership meaning. `pkg/game/lazy.go` is the one generic that
replaces both: a value, the reason there is none, and whether resolution was
attempted.

**The nine `xCache`/`xLoaded` pairs were one shape exactly** — test the flag,
set the flag, read, store, return — and their failure path is kept as it
stood: a load that fails records the attempt and answers its own zero value
forever rather than retrying next frame. Two have a tail that is not
mechanical and is written out rather than generated: `shopArt` stores its value
and then fills in the shared book bitmap through the stored pointer, and
`worldMapAssets` stores its manifest before finishing it, so the partial object
is the one retained. Two of the nine accessors (`gameMenuArt`,
`worldMapAssets`) carry no nil-receiver guard where the other seven do; that
asymmetry is pre-existing and was preserved, not tidied.

**The ten `X`/`XErr` pairs were one shape in nine cases and not in the tenth.**
Nine are an optional `*T` install asset resolved at construction with its
reason beside it. `Campaign` is not: it is a value type with a meaningful zero,
and `CampaignErr` had **no reader anywhere** — the only two mentions were the
constructor that wrote it and the fixture that seeded it. It is folded in
anyway, so "this install's campaign would not read" is now reachable through
the same accessor as every other one. Two more errors, `TownSquareArt`'s and
`CommandPanelArt`'s, are read only by tests.

Every call site was rewritten from the compiler's own error positions rather
than by pattern: 630 in `pkg/game`, 34 across six commands, and roughly twenty
by hand where the error had no single mechanical answer.

### Five components, and the three judgement calls

Field promotion does the work: Go promotes an embedded struct's fields, so
`f.Font`, `f.Campaign` and `f.Town` read exactly as before and not one method
body changed. The whole diff outside the declaration is composite literals,
which cannot set a promoted field — 181 regrouped from the compiler's own
parse, and the two with comments inside their braces written out by hand.

Three assignments are judgement, not lifecycle, and each is stated where it is
made:

- **`Campaign` is `InstallResources`**, not `CampaignSession`. It is the
  scenario registry's declared mission set, read once out of the install and
  handed to every game this process opens. `Town` is what is built over it, and
  `Town` is what resets.
- **A bounded presentation generator is `RuntimeServices`; the latch its draws
  advanced is `Presentation`.** A component owns the seam or the state the seam
  produces, not both. So `TownAmbientRandom` is a service and
  `townBirdDelayReady` is drawn state.
- **`originalCity` is save provenance but sits in `CampaignSession`**, because
  its lifetime is the session's exactly. That choice is what makes the
  invariant below true.

### The invariant and the ratchet

`CampaignSession` is now **exactly** what `resetSessionForNewGame` drops.
`frontend_session_test.go`'s reflect walk flattens one level through the
components, so it still enumerates all 81 fields and additionally refuses three
new things: a field declared straight on `FrontEnd`, a kept field declared in
`CampaignSession`, and a reset field declared anywhere else.

`internal/archtest/composition.go` measures the root and ratchets two numbers
downward only, committed in `composition_baseline.go`:

- **81** fields across the five components.
- **20** non-test files whose `FrontEnd` methods reach three or more components
  at once. Three, because two is usually one answer with two halves — the
  install's art and the session it is drawn for — and three is a file holding
  separate lifecycles together. `frontend.go` reaches all five;
  `gameoptions.go`, `generatedworld.go`, `resume.go` and `townexterior.go`
  reach four. `go run ./internal/archtest/cmd/composition` lists all 20, so the
  number is arguable rather than opaque.

The measurement is syntactic like the rest of `archtest`, and its own doc says
what that costs: an access counts when its base is an identifier the file
declared as a `FrontEnd` receiver or parameter, or reaches through a struct
field this package declares to hold a `FrontEnd` (`townScreen`'s own `f`). A
`FrontEnd` reached through an interface or a closure is not seen, so the count
is a **lower bound**, not a census.

## Proof

**The save wire format did not move, measured rather than assumed.** `gob`
matches struct fields by name, so a renamed exported field would read every
existing save clean with that value gone and no error. A `reflect` walk
enumerates **1806** exported field paths reachable from `game.Snapshot`, with
the type at each leaf; the listing over base `9cb6d92` and over this branch is
identical text. It was re-run after step 2 and is identical again.
`gob.NewEncoder`/`NewDecoder` appears in exactly two non-test files
(`save.go`, `savehistoricalwire.go`), both under that graph. `pkg/formats/sav`
is untouched; `OptionsStore` writes named `key=value` lines, not reflected
field names, and none of its fields moved.

**The game itself.** `cmd/missionrun -mission {10,20} -trace -ticks 1` against
the EN install prints **byte-identical output** on base and on this branch,
including mission 10's `16 checks, 27 instants, 12 triggers` and mission 20's
`14 checks, 15 instants, 11 triggers`, which match
`pipeline/milestone-baseline.txt`. The `UNSUPPORTED` count is **0 on both
missions, on base and on head** — this story was not meant to move it and did
not.

Gates on the landing commit are recorded in the lane's report: `gofmt -l .`
clean, one `go test -trimpath -count=1 ./...`, `check-no-game-assets.sh`,
`storyguard measure` with non-test identifiers, non-test file names and all six
comment forms at 0, `check-release-tests.sh` over both roots, and
`check-milestone2-acceptance.sh` reproducing untouched main's reason set
exactly.

`CommentBytes` rises in `internal/storyguard/baseline.go` for three new files:
`pkg/game/lazy.go`, `frontend.go`'s five component docs, and
`internal/archtest/composition.go`. The baseline names them, which is what the
ratchet's own rule requires of a rise.

## Open debt

- **20 coordination points.** The ratchet measures them and forbids a 21st; it
  does not reduce them. `frontend.go` reaching all five components is the
  concentration the owner named, and splitting it is a separate story.
- **The coordination count is a lower bound.** A `FrontEnd` reached through an
  interface or a closure variable is invisible to a syntactic measure. Making
  it a census needs type information the guard does not load today.
- **No component became a package**, and none should until the coordination
  count falls: today the import DAG would be satisfied and the coupling would
  simply move to the imports.
- **`FrontEnd` still has 141 exported methods.** This story counted them and
  changed none. Where a method belongs is the question the coordination metric
  now measures but does not answer.
