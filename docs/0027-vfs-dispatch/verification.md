# Verification — multi-archive resolution by identity prefix

Task commits, oldest first: `e6ae5bc` (T1), `3aec480` (T2), `b77cd9b` (T3), `9e33b45` (T4),
`61b1a7d` (T5), `20a4689` (T6), `38d54ca` (T7). Base `117da06`. Eight untrailered commits sit in
the span — three correcting this story's own `tasks.md` file lists, five authoring
`docs/0028-app-unit-command/` — all docs-only, disclosed. Submodule pin frozen at research
`e7602bb` throughout; nothing in this chain reads it.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. Every criterion is a unit test over
synthetic containers and temporary directories built in test code: **no test reads a game
install**. Figures marked *(install)* were measured in the orchestrator's seat over a lawful
install — evidence about that install, and no part of any criterion.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output)
$ go test -trimpath -count=1 ./...           (0 FAIL, 0 SKIP; 648 tests, 2124 counting
                                              subtests, 25 packages ok, 3 without tests)
      pkg/vfs   25 tests, 29 counting subtests   (the package was a doc comment before T1)
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0027-vfs-dispatch
  analysis.md 7052 / 7168        provenance.md 7073 / 7168
  spec.md 13288 / 13312          plan.md 13268 / 13312
  tasks.md T1..T7 all under 1400; legend+traceability 965 / 1200
  plan <= 1.2 x spec ok; tasks <= 1.2 x plan ok
  verification.md (prose) 8736 / 9216 — fenced blocks free                    (exit 0)
$ sh scripts/check-sdd-audit.sh              at 38d54ca, in a detached worktree
check-sdd-audit: 89 trailered commit(s) in ac6bd87..HEAD checked
FAIL 0027-vfs-dispatch: every task in tasks.md has landed and there is no verification.md
check-sdd-audit: FAILED                      exit 1; 38 lines, one FAIL, 36 notes. A
                                             worktree has no builds/ tree, so the 8
                                             builds/ warnings a working checkout also
                                             prints are absent here.
$ sh scripts/check-sdd-audit.sh              with this commit
check-sdd-audit: 89 trailered commit(s) in ac6bd87..HEAD checked
check-sdd-audit: ok (44 note(s)/warning(s), none enforced — 34 on stories before
                     WITNESS_FROM, 9 builds/ warnings, and 0028 in flight. 0027's only
                     line is the builds/ one, which is untracked and warning-only)
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 117da06..HEAD | sed '/^$/d' | sort | uniq -c
      1 0027-vfs-dispatch/T1  ...  1 0027-vfs-dispatch/T7   (7 ids, each exactly once)
$ git log --format='%B' 117da06..HEAD | grep -ci co-authored-by
0
```

That one FAIL is what a story with every task landed and no evidence file is supposed to look
like. This file clears it and carries no trailer.

## Witnesses

Every id and the named thing that answers for it — `pkg/vfs/fs_test.go` and `pkg/vfs/enum_test.go`
unless another file is named.

```
AC-1   SC-1   TestOpenEmptyListsReadsAbsent — both nil and both empty, the open succeeding
       and a read then absent.
AC-2   SC-1   TestIdentityFromHostPath — the criterion's three rows (`Graphics.RES`,
       `sub.d/graphics.res`, a 20-byte stem cut to 15) plus the separator, dot and fold cases.
AC-8   AC-9   P-6  SC-1   TestOpenFailsOnMalformedArchive, TestOpenRefusesEmptyIdentity,
       TestOpenFailsOnMissingArchive — each through assertOpenFails, which is P-6 over one
       fixture: nil *FS, an error naming the host, and no read possible from the result.
       Sampled over every failing-open fixture, as SC-1 says; not proved.
AC-3   SC-2   TestReadFileFoldsAndRequiresIdentitySegment — backslashes, upper case, and the
       identityless third form absent.
AC-4   P-5    SC-2   TestReadFileDistinctIdentitiesIgnoreOrder — both permutations of a
       two-archive list, every address re-read. P-5 sampled there; not proved.
AC-6   SC-2   TestReadFileRefusesEmptyLeadingSegment — the listed directory holds both forms,
       so the refusal is the grammar's and not a missing file's.
AC-7   SC-2   TestReadFileUnregisteredIdentityIsAbsent; TestReadFileIdentityMustEqual-
       LeadingSegment — `graphicsx/x.bin` and `graphic/x.bin`, both directions of the
       prefix relation.
AC-5   SC-3   TestReadFileSameIdentityFirstInListAnswers — the bytes swap with the list;
       TestReadFileSameIdentityFallsPastAnArchiveWithoutTheEntry.
AC-10  SC-3   TestReadFileSurfacesUnreadableLooseFile — see below; EXERCISED, not skipped.
AC-13  SC-3   TestReadFileArchiveTierPrecedesDirectories; TestReadFileSeparatorless-
       AddressReachesTheLooseTier; TestReadFileWalksPastANonRegularFile (DD-4).
AC-11  P-1  P-3  SC-4   TestEntriesListsEveryAddressOnceInOrder — fixture entries given out
       of sorted order, so the ascent is the listing's; every listed address read back
       against its listed source (assertListedSourcesAnswerTheReads, which is P-1 over a
       listing); two enumerations and two reads compared equal (P-3). Both sampled.
AC-12  P-4  SC-4   TestLocateAnswersExactlyOneSourcePerAddress — 16 addresses across both
       tiers, each answering as exactly one of archive, directory or absent, the bytes
       naming their own source. P-4 sampled over those 16; not proved.
AC-14  P-2  P-3  SC-5   TestReadsAndEnumerationsLeaveEveryListedSourceUnchanged — returned
       bytes and returned listing mutated and re-fetched; looseTree over the whole tree
       (archive hosts included) compared before and after every read, locate and
       enumeration, the failing ones included. TestLooseTierRefusesDotDotDotAndEmptySegments
       — `../outside.bin` read directly first, so the refusal is the tier's; the escape
       target unchanged afterwards.
SC-6   the four mutants, re-run in this seat — below.
AC-15  SC-8   pkg/game/address_census_test.go TestAddressCensus (162 addresses over 3
       containers, plus 1 identity with no remainder — re-run in this seat);
       TestGameProductionReachesNoArchiveReader parses every non-test file of pkg/game and
       refuses the reader import, failing if it parsed none.
SC-7   the consumer suites, green with fixtures keyed by the flipped constants: pkg/ui
       unedited; pkg/render/menu edited only in menu_test.go's and state_test.go's pinned
       prefix literals; pkg/render/terrain only in TestTilePathAndSlotIndex's seven rows;
       cmd/againrom/main_test.go only where it writes container entries. Byte-identity of
       the frozen strings is the tasks' baseline-vs-tree dumps — see below.
```

## The mutants

SC-6's four, each applied to `pkg/vfs/fs.go` in this seat, run against the **whole** tree and
reverted; `fs.go` md5 `c69d3525d2e65c6d179180feb65a2b54` before and after every one.

```
M1  identity equality weakened to HasPrefix(head, identity)
    KILLED, tree-wide sole killer TestReadFileIdentityMustEqualLeadingSegment
    ("graphicsx/x.bin" returned 2 bytes, want none).
M2  the leading-separator refusal deleted from ReadFile
    SURVIVES. Whole tree green, 25 packages ok. An equivalent mutant — see below.
M3  the same-identity walk reversed
    KILLED, TestReadFileSameIdentityFirstInListAnswers AND TestEntriesEmitsTheReadsWinner-
    WithinOneIdentity in both permutations. T1 reported the first alone, correctly: the
    second test did not exist until T2.
M4  the enumeration dedup keeping the later record
    KILLED, TestEntriesEmitsTheReadsWinnerWithinOneIdentity alone, in both permutations and
    three ways at once — the listing names index 1 where the read serves index 0, it declares
    size 24 against the read's 12, and Locate contradicts it. Exactly as T2 reported.
TRIM  the adjacent variant DD-3 actually rejects: the reader's trim over the whole address,
    strings.Trim(fold(address), "/")
    KILLED, three tests, all in pkg/vfs — TestReadFileRefusesEmptyLeadingSegment (T1's
    figure) plus TestLocateAnswersExactlyOneSourcePerAddress and TestReadsAndEnumerations-
    LeaveEveryListedSourceUnchanged, which T2 added. All three report the same thing:
    ReadFile("\graphics\x.bin") returned bytes, want none.

Hollowness checks, run by the entries that owned no mutant. The tasks' own figures.
T5  graphicsPrefix -> `graphix/`: seven cmd/terraintool tests, plus TestTilePathAndSlotIndex.
T7  the same, once the census existed: TestAddressCensus, TestLoadStatics (8 subtests),
    TestLoadUnits (5), TestLoadMapViewerStatics (6), TestOpenGraphics, terraintool's statics
    and unit-census suites, mapview's two statics suites, and cmd/againrom's TestCheck,
    TestMarkerFlag, TestStaticBundle, TestUnitBundle and TestStartupFailures.
T6  scenarioIdentity -> `scenarix.res`: TestArchiveMapsScan, TestMapBytesAddresses' archive
    case, cmd/againrom's TestCheck on both row counts. The two loose-route swaps back to an
    os.ReadFile: TestDirMapsScan, TestMapBytesAddresses' loose case, TestFrontEndMarkers.
T7  the census against itself: decoy bytes without the container name fail the decoy
    assertion on every address; a read without the identity segment fails every resolution.
```

**M2 is an equivalent mutant.** T1 reported it as a survivor rather than arguing it away; the
argument, re-derived here from the shipped code, is that two independent rules produce the same
absence. With the guard gone, `/graphics/x.bin` reaches the archive walk carrying an empty leading
segment, which no identity can equal because an empty identity refuses the open outright; it then
reaches the loose tier as an address with an empty first segment, which `segments` already
refuses. Removing either rule alone leaves the absence standing, so nothing can separate the two
forms — the same shape as 0026's `M23`. **No plan edit is owed**; the record is, and TRIM is the
mutant that does separate them.

## Disclosures

- **`cmd/mapview`'s suite does not discriminate which container answers — and does discriminate
  whether the filesystem is over the archive at all.** T5 measured the first: under `graphix/` that
  suite stayed green, reading the `tile slots` substring and pinning only the `0/128` case. T7
  measured the second: an `OpenGraphics` returning a filesystem over *no* archive fails
  `TestStaticsFlagsSummary` and `TestStaticsFlagsWireToTheViewer` there. Either half alone
  misdescribes the suite. `cmd/terraintool`'s pixel oracles discriminate the container, which is
  why SC-7 is a byte comparison of baseline-vs-tree dumps rather than a mutant.
- **DD-7's freeze claim has an unstated domain.** On a container holding one `.alm` path twice,
  FR-9 lists each unique address once, so the picker's row count falls by the duplicate records —
  T6 measured 13 → 12 rows on a fixture built for it, with every surviving `Source`, `Text`,
  ordering position and byte digest identical. DD-7's wording does not admit that. *(install)*
  `cmd/restool list` over the EN `scenario.res`, the RU `SCENARIO.RES` and the GOG `scenario.res`
  returns **31 records each (28 `.alm` + 3 `.reg`, no nested archive), and no duplicate folded
  path in any of the three**, so no shipped
  container reaches the case. **No amendment is owed**: the domain is witnessed here rather than
  asserted there, and a duplicate row was indistinguishable from its twin and opened the same map.
- **R-2 and C-3 have no trigger on any shipped install.** *(install)* The same sweep finds **no
  folded collision among any file in any of the three install roots**, while the loose `.alm` names
  ship mixed-case throughout — so the fold is exercised on every shipped row and moves none of
  them. C-3's case-sensitive-host defect is real and stays a disclosed limitation; this bounds it.

```
EN and GOG   Beast.ALM Cross.ALM Forester.alm Horror.alm Islands.alm KIDS.LM Kids.alm
             Kids2.ALM LuMoir.alm Tomb.ALM Waters.alm
RU           FORESTER.ALM Horror.alm ISLANDS.ALM KIDS.ALM LUMOIR.ALM WATERS.ALM
```

- **The first panic in a `pkg/` production file.** `mustIdentity` in `pkg/game/maplist.go`, the
  `MustCompile` shape over a package constant — DD-1 requires the scenario prefix be derived rather
  than spelt, and the alternative was a swallowed error whose only symptom is every campaign row
  listing and none of them reading. Re-derived here: it is the **only** `panic(` in any `pkg/`
  production file — the other eight in non-test code are all in `internal/synth`, a fixture
  builder — and no tracked normative text mentions panics at all. A policy gap, not a violation;
  no policy is written here.
- **The two tiers disagree about a doubled separator.** `graphics//x.bin` is an archive **hit** —
  `findArchive` trims the remainder, the reader's own key rule (DD-2) — while the loose tier
  answers it absent, `segments` refusing an empty segment (DD-4). Both halves are the design as
  written; the asymmetry is not, and it is pinned as a hit in
  `TestLocateAnswersExactlyOneSourcePerAddress` rather than left to be found.
- **A bare identity with no remainder is not an address, and the census asserts it.** An object
  class with no `File` key resolves the empty sheet path, so `sheetCache` asks for `graphics` with
  nothing after it. The grammar answers absent and reports no source — `statics.go`'s absent-sheet
  exclusion holding by the address rule rather than by a branch of its own. The census keeps that
  one name aside and asserts it absent through both `ReadFile` and `Locate` rather than filtering
  it out, because "this is not an address" is a claim about it.
- **The loaders' nil guard narrowed when they re-typed.** `LoadStatics` and `LoadUnits` took
  `*res.Archive` and now take `terrain.EntrySource` (DD-6), so `src == nil` no longer catches a
  **typed** nil: probed in this seat, `LoadStatics((*vfs.FS)(nil))` panics with a nil-pointer
  dereference where `LoadStatics(nil)` still returns `graphics/objects/objects.reg: no graphics
  archive`. No shipped call site reaches it — all four pass a filesystem from a successful `Open` —
  but a hand-assembled `Archives` in a test can, and T6 met that class in `world_test.go` and
  `frontend_statics_test.go`. Disclosed, not fixed: no contract covers it.
- **`docs/ARCHITECTURE.md` still grants `pkg/formats/res` to both developer front-ends, and neither
  imports it any more.** T7 left `cmd/mapview`'s row deliberately: it mirrors the allow-map grant
  in `internal/archtest/dag.go`, which is unchanged and which the plan's Baseline forbids touching.
  **Re-derived here, it is two rows, not one** — `cmd/terraintool` dropped the import in T5 and
  `cmd/mapview` in T4, and neither package names it in any file, tests included. Both are now
  unused permissions, for a later story to drop with the allow-map rows they mirror.
- **AC-10 was exercised on this host, not skipped.** The earlier of two listed directories holds
  the address as a regular file carrying a deny-read ACE for the current user, applied with
  `icacls` and removed in a cleanup registered before `t.TempDir`'s own. The helper self-probes: if
  a direct read still succeeds, or fails not-found, the case skips with the reason logged rather
  than passing on nothing. The whole-tree run above reports **0 SKIP**, so it ran.
- **The census spells no address of its own, and that fence was checked.** Verified here: outside
  its imports and its failure messages, the only string literals in
  `pkg/game/address_census_test.go` are two container-relative fixture entries (`10.alm`,
  `sub/91.ALM`) and one map name, a `|` parting the payload's halves, the `/` composing an address
  from a derived identity and an enumerated name, and — in the *other* test in the file — the
  reader's import path and two globs. No identity segment is written anywhere in it.
- **`OpenTileset` came out in T7, and the plan names only `OpenGraphics` as the survivor.** T7
  reported it rather than treating it as ruled on: the wrapper's own stated reason — a caller
  needing terrain alone — was falsified by the teardown, its one call site wanting the filesystem
  too. Its text was probed equal to `OpenGraphics`' at all four failing shapes before removal.

## Contract findings

Nothing in `spec.md` (13288 B) or `plan.md` (13268 B) was found false at the implemented tree, and
neither is edited here. DD-7's wording is incomplete rather than wrong, recorded above with the
measurement that bounds it; C-3 and R-2 hold as written and are shown unreachable on shipped data;
SC-6's second mutant is equivalent, its intended behaviour witnessed by the adjacent variant. Where
a task's figure and mine differ — M3 and TRIM — the tree now kills wider than T1 could measure,
because T2's cases did not yet exist. A strengthening, recorded so those two commit bodies read
correctly against this file.
