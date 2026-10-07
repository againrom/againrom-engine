# 0121 — mission flow: plan

Two halves that share no file: the door (FR-1..FR-6, FR-12) in `pkg/game/maplist.go`,
`pkg/game/frontend.go` and `pkg/ui/picker.go`; the hero (FR-7..FR-11) in `pkg/game/world.go`.

## Decisions

**DD-1 — the mission rows are composed over the finished list, not inside it.** `BuildMapList`
keeps its whole contract: it answers "what maps does this install hold", its ordering rule is
unchanged and its tests are untouched. A second exported function takes that list and returns the
mission rows followed by it. The front end composes the two at the one place it already builds
`Maps`. Folding the rows into `BuildMapList` would put two questions in one function and would make
every existing ordering assertion a statement about missions as well. (FR-1)

**DD-2 — a mission row is a `MapEntry` with a positive `Mission`.** One new field; zero means "not a
mission row", which is what every existing literal already says. `Source`, `Err` and `Name` are
copied from the map row the mission row was made from, so `Choosable()` needs no arm of its own and
FR-5 holds by construction rather than by a second rule. `Text()` gains one leading branch.
(FR-1, FR-5, FR-6)

**DD-3 — the number comes from `strconv.Atoi` on the stem, and a failure is silence.** The stem is
the entry name with its extension removed; the parse must succeed and be positive. `MissionMap`
composes the same address from the same integer by `strconv.Itoa`, so the two are inverses and no
mission number is ever spelled in source. A stem that does not parse — `npc.reg` never reaches here,
but a modded container may hold `bonus.alm` — yields no mission row and no complaint. (FR-2, P-3)

**DD-4 — mission rows are ordered by the parsed number, stably.** Ascending, `sort.SliceStable`, so
two entries claiming one number keep container order. Ordering by text would put `100` before `20`.
(FR-1)

**DD-5 — the routing is a delegation, not a merge.** `loadMap` gains one branch immediately after
it takes the row: a positive `Mission` returns `f.MissionOpener(n)()` and nothing else. The mission
closure re-reads the entry from the same address `mapBytes` would have used, which costs one archive
read and buys P-4 — every line of viewer dressing, every failure message and the notice seam stay in
the one place they already are, so the two doors cannot drift. (FR-3, FR-4, P-4)

**DD-6 — the picker's title is the only `pkg/ui` change.** `SELECT A MAP` becomes
`SELECT A MISSION OR MAP`. No new key, no new input field, no signature change: the row kind is
carried by the row, so the whole distinction is expressible in the text `pkg/game` already supplies.
(FR-6)

**DD-7 — the hero is recorded when the mission opens, guarded by a flag.** `missionNotices` gains
`hero sim.EntityID` and `heroSet bool`, filled in `openMission` from `ms.Start.IDs[0]` when the
start recorded any id. This is `invSubjectSet`'s own shape and for its own reason: the zero id is a
real entity, so the value cannot say by itself whether there is a hero. `Start.IDs[0]` is the hero
because the start places the first member at the drop cell exactly and crowds the rest around him.
(FR-7, FR-10)

**DD-8 — the test is the first statement of the undecided arm of `settleNotices`.** Inside
`if !m.announced`, before the `mw.world.Outcome()` read: if `m.heroSet` and the hero is not a living
entity, take `sim.OutcomeLost`; otherwise take the world's outcome. Both arms then run the one
existing block that latches `announced`/`outcome` and opens the banner, so FR-9 is the path that was
already there and not a second one. The world's own reporter tests lose before win, so putting the
hero ahead of `Outcome()` reproduces the engine's three-test order end to end. (FR-8, FR-9)

**DD-9 — "living" is `mw.entity(id)` plus `Entity.Alive()`.** A world that no longer holds the id
and a world that holds it dead or downed are one answer. `mw.entity` is the driver's existing
copy-handing read. (FR-7)

**DD-10 — the map path is untouched, and that is what keeps it safe.** `settleNotices` returns on
its first line when `mw.mission` is nil, and `openMapWorld` sets no mission. Nothing is added to
that path; FR-11 is tested rather than coded. (FR-11)

**DD-11 — the stale sentence is corrected where it stands.** `openMapWorld`'s comment says the
picker path advances under its own script; it compiles none. The correction says what is now true of
both paths — this one compiles no script and advances under the AI alone, and a picker row that
names a mission does not come here at all. (FR-12)

## Risks

- **R-1 — the list doubles in the campaign's length.** 38 rows become 66 on a stock install, in a
  25-row window. Accepted: the mission rows lead, so the first row a player lands on is mission 10,
  and every previously reachable row is still reachable. The alternative — a mode key — costs a new
  input field on a path two other live lanes are editing.
- **R-2 — the mission row and the map row read the same entry twice.** Only when a mission row is
  chosen, once, at open. Accepted for DD-5's benefit.
- **R-3 — a headless run and a windowed run now disagree about a dead hero.** Real and disclosed:
  the rule is above `pkg/sim`, so `-check -mission N` and the world's byte form do not carry it.

## Success criteria

- **SC-1** `go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -trimpath
  -count=1 ./...` clean on the committed tree, with no game install present.
- **SC-2** `scripts/check-no-game-assets.sh`, `check-doc-budget.sh` and `check-sdd-audit.sh` pass;
  `check-sdd-audit.sh`'s FAIL set is empty.
- **SC-3** Witness by reverting: with DD-8's branch removed, the hero-death tests fail; restored,
  they pass.
- **SC-4** Witness by reverting: with DD-5's branch removed, the mission-routing test fails;
  restored, it passes.
- **SC-5** `git diff --diff-filter=D --name-only 67d2e57..HEAD` is empty.

## Traceability

| FR | DD | SC |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-4 | SC-1 |
| FR-2 | DD-3 | SC-1 |
| FR-3 | DD-5 | SC-1, SC-4 |
| FR-4 | DD-5 | SC-1 |
| FR-5 | DD-2 | SC-1 |
| FR-6 | DD-2, DD-6 | SC-1 |
| FR-7 | DD-7, DD-9 | SC-1, SC-3 |
| FR-8 | DD-8 | SC-1, SC-3 |
| FR-9 | DD-8 | SC-1 |
| FR-10 | DD-7 | SC-1 |
| FR-11 | DD-10 | SC-1 |
| FR-12 | DD-11 | SC-1 |
