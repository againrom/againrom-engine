# Verification — the campaign mission start

Environment: Windows 11, Go 1.26.1 (`go.mod`'s pin), worktree `impl-0065-mission-start`. The
install-facing runs read the two preserved roots read-only; no game data entered the repository, and
the tool prints counts, indices and cells only.

## The gate

```
go build ./...        clean
go vet ./...          clean
gofmt -l $(git ls-files '*.go')   printed nothing
go test -count=1 -trimpath ./...  32 packages ok, 0 fail, 3 with no test files

sh scripts/check-no-game-assets.sh   check-no-game-assets: clean (tree scan)   EXIT=0
sh scripts/check-doc-budget.sh       EXIT=0, FAIL count 0
sh scripts/check-sdd-audit.sh        EXIT=0, FAIL count 0
```

Deletion sweep over the whole branch, `git diff --diff-filter=D --name-only 13e3e4a HEAD`: **empty**.
28 files changed, 2550 insertions, 80 deletions, none of them a file.

## The census, over both preserved roots

`classdump -campaign <root>`, EN. Every column is this tree's own loader answering; the figures to
the right of the map name are what `MISSION-ARM-006` publishes for the same corpus, and this run
reproduces the totals and every per-map row it names.

```
map                 type6    units server-id      npc   humans   drop cells
10.alm                 35       19       14        2        0   (17,66)
100.alm               113       69       41        3        0   (15,16)
110.alm                39       22       17        0        0   (10,63)
120.alm                96       43       53        0        0   (31,12)
131.alm               222      211       11        0        0   (10,12)
151.alm               237      132      103        2        0   (72,132)
Beast.ALM             964      786      178        0        0   (130,132)
Horror.alm           1815     1394      420        0        1   (18,15)
LuMoir.alm            228      226        2        0        0   (84,77)
Tomb.ALM              462      407       55        0        0   (145,243)
   … 28 further rows, one per shipped map …
TOTAL                8094     6672     1405       15        2   38 cell(s) over 38 map(s)
  arm units      taken   6672  reached an entry   6672
  arm server-id  taken   1405  reached an entry   1405
  arm npc        taken     15  reached an entry     13
  arm humans     taken      2  reached an entry      2
```

The RU root, same command, is the discriminating second run:

```
TOTAL                3991     3366      609       15        1   33 cell(s) over 34 map(s)
Horror.alm              0        0        0        0        0   none
```

34 maps, not 38: that root drops four loose maps. And its `Horror.alm` is the one map in either root
that yields no drop cell — 0 placements, no trigger record — which is `ALM-CORP-060`'s two-byte
truncation showing up in this reader without being looked for.

**AC-2 / SC-2 — witnessed, and the rival is named.** 6672 / 1405 / 15 / 2 of 8094, and `10.alm` at
19 / 14 / 2 / 0. The superseded band gave different numbers, recorded in
`docs/0049-databin-classes/verification.md`: `Horror.alm` as `npc 0 / server-id 474 / humans 1 /
units 1340` against this run's `units 1394 / server-id 420 / humans 1`, and `LuMoir` 15/213 against
2/226, `Beast` 186/778 against 178/786, `Cross` 148/626 against 142/632, `Tomb` 58/404 against
55/407. 84 placements over five loose maps changed the collection they resolve against.

**AC-4 / SC-4 — witnessed.** 38 cells over 38 maps, one each, and `10.alm`'s is (17, 66) —
`MISSION-M10-009`'s own figure, from a payload this reader required to tile exactly.

## The NPC arm's two routes

`classdump -campaign <root> 10.alm`:

```
10.alm: placements on the npc arm
  record      npc   serverID    entry  own defID    entry
       2       51        509      200       1001       99
      32       52        510      201       1033      131
```

**AC-3 / SC-3 — witnessed.** Both take the NPC arm, both reach a Humans entry, and neither reaches
the entry its own definition id names — 200 against 99, 201 against 131. Two of the 15 shipped
NPC-arm placements reach nothing (`100.alm` record 112, `70.alm` record 105): their subscripts, 24
and 23, are sections the registry names no definition id for. `100.alm`'s record 112 carries a
definition id of its own that *would* have resolved (1024 → entry 122) and does not reach it, which
is the ordering's cost measured rather than asserted.

**Observed, and owed to research rather than used here:** five shipped NPC-arm placements route
through subscripts whose `DataBinID` is one of `42..46` — the values `REG-NPC-058` records as naming
nothing in `Templates.ini` — and every one of them *does* reach a Humans entry as a server id
(`40.alm` 25→42, `140.alm` 26→43, `150.alm`/`151.alm` 29→46). Two different id spaces, so this
contradicts nothing; it is a measurement that may narrow that row's Unknown.

## Unit evidence

Run under `go test ./...`; the named test is the witness.

| Id | Witness |
|---|---|
| **AC-1**, **SC-1** | `game.TestMissionMapAddress` (10 → `scenario/10.alm`, 0 and negatives refused); `game.TestStartMissionBuildsAWorldFromANumber` (world bounds are the map's) |
| **AC-5** | `mapload.TestStartMissionFallsBack` (no cell, a zero column, a zero row — each inside 30..100); `mapload.TestFallbackRangeIsClosedAndPerAxis` (both ends reached on both axes; the axes differ) |
| **AC-6**, **P-3** | `mapload.TestStartMissionPutsThePartyOnTheMapsOwnCell` (hero on the drop cell; no shared cell; no member past the first on a closed one); `mapload.TestStartMissionCountsACrowdedParty` (the exhaustion case is counted) |
| **AC-7**, **SC-7** | `mapload.TestDropCellsRefusesAPayloadTheFramingDoesNotTile` (six untiling shapes, each carrying a real drop node); `mapload.TestStartMissionOverAnUntilingPayload` (the start falls back) |
| **AC-8**, **P-1** | `mapload.TestStartMissionIsDeterministic` (two starts agree on the drop cell, every party cell and the world digest); `sim.TestDrawsFromOneSeedAgree`; `archtest`'s determinism scan over `pkg/sim` |
| **AC-9** | `mapload.TestEveryArmIsTakenAndCounted` cases *the floor beats the npc flag* and *the floor beats a definition id*; `mapload.TestAWorldBuiltWithATableCarriesTheResolvedHealth` |
| **P-2** | `game.TestStartMissionBuildsAWorldFromANumber` and `game.TestStartMissionFailuresNameTheAddress` drive the entry through a filesystem over one synthetic archive; `check-no-game-assets.sh` is clean and no shipped source spells an install path |
| **P-4** | `mapload.TestResolveWithNoTableStillNamesTheArm` — every arm is named with no table at all |
| **SC-5** | the three rows above (`AC-5`, `AC-6`, `AC-8`) |
| **SC-6** | the gate above, and `mapload.TestFromALMKeepsThePreStoryDigest`, which still passes unchanged |

## Not witnessed, and why

- **The class-key band's edges.** A negative key and a key above 255 are decisions, not
  measurements: the shipped domain is `1..80` and no map exercises either. `TestEveryArmIsTakenAndCounted`
  pins both so the decision is visible; nothing shows it is the original's.
- **The `DataBinID == 26` sentinel's meaning.** Dropped at the table, so a placement routed through
  such a section reaches no entry. What the original composes instead is undecoded.
- **A map with more than one drop cell.** 38 of 38 authorise exactly one, so the uniform pick is
  degenerate on everything that ships and the multi-cell path is exercised only by
  `TestDropCellsReadsTheDropNodes`'s synthetic payload.
- **The fallback on shipped data.** No shipped map takes it. It is also the same cell every time,
  because the seed is a constant — stated in `start.go` as the divergence it is.
- **The party against real art.** Nothing draws a mission's world: this story ships no front-end
  door, so "populated and playable" is witnessed as entities in a world and their cells, not on a
  screen.
- **The `10.alm` win chain.** Reaching a win needs the script runtime, which is another story's.
