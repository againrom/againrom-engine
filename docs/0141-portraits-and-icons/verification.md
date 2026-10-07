# 0141 — verification

Written after the code landed, against branch `0141-portraits-and-icons` at `7113a78`, five commits,
42 files, 3430 insertions. Research pin `cdfe33d`.

**Two kinds of evidence are separated below and are never mixed.** *Re-run* means the command was
executed against this tree while this file was being written and its output is pasted. *Recorded*
means the story measured it against the owner's two lawful installs at the time, with throwaway
probes that were deleted before the gate; those numbers are quoted as the story's and what would
re-take them is named. No recorded number is restated as if it had just been measured.

## Re-run: the gate

```
$ go build ./...
(clean)
$ go vet ./...
(clean)
$ gofmt -l $(git ls-files '*.go')
(clean)
$ go test -count=1 -trimpath ./...
ok  againrom/pkg/data 2.765s   ok  againrom/pkg/formats/bmp 0.469s   ok  againrom/pkg/game 3.573s
ok  againrom/pkg/sim  5.702s   ok  againrom/pkg/ui          5.460s   ok  againrom/internal/archtest 0.969s
   ... 39 packages, all ok; 4 with no test files; no FAIL
$ bash scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ bash scripts/check-doc-budget.sh
docs/0141-portraits-and-icons/provenance.md   9636 /  16384 bytes  ok (58%)
docs/0141-portraits-and-icons/spec.md        10890 /  20480 bytes  ok (53%)
docs/0141-portraits-and-icons/plan.md        10433 /  20480 bytes  ok (50%)
0141-portraits-and-icons: plan <= 1.2 x spec 10433 <=  13068 bytes  ok
$ bash scripts/check-hotfix-ledger.sh
check-hotfix-ledger: ok
$ bash scripts/check-sdd-audit.sh --story 0141-portraits-and-icons
note 0141-portraits-and-icons: no tasks.md (fanned out to no executor, or in flight)
check-sdd-audit: ok
$ git log --format='%h %(trailers:key=Co-Authored-By)' master..HEAD
7113a78    9657d9c    6d805b0    b1591f7    1a4bb70
```

No commit on this branch carries a `Co-Authored-By` trailer and none carries an `SDD-Task:` trailer;
there is no `tasks.md`, so the FR and DD witness moves to the table at the foot of this file.

## Re-run: the game

The script-gap census, from this worktree against the EN root:

```
$ go build -o /tmp/mr ./cmd/missionrun
$ AGAINROM_ASSETS=<gameversions>/en /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED
17
$ AGAINROM_ASSETS=<gameversions>/en /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED
13
```

`pipeline/milestone-baseline.txt` carries the same two numbers for master before this story —
mission 10: 1 groupcmd sub-command 11 + 2 sub-command 15 + 13 instant op 2 + 1 instant op 20 = 17;
mission 20: 1 instant op 12 + 11 instant op 2 + 1 instant op 23 = 13. **Unchanged, and that is the
right outcome**: this story runs no new script node. The result it owes is the one in
`builds/current/` — a selected creature showing its own shipped portrait at its own tier, a selected
person showing a composed doll, a dialogue showing the speaker's face on a black ground, and a
spellbook cell carrying an icon instead of three letters.

## Re-run: three corpus facts, with tools that ship

```
$ go run ./cmd/restool list <en>/graphics.res | grep -ci infowindow
86
$ go run ./cmd/restool list <ru>/graphics.res | grep -ci infowindow
81
$ go run ./cmd/restool list <ru>/graphics.res | grep -i 'infowindow/orc'
115256  infowindow/orc.bmp    115256  infowindow/orc2.bmp
115256  infowindow/orc3.bmp   115256  infowindow/orc4.bmp
$ go run ./cmd/restool cat <ru>/graphics.res interface/spellbook.bmp | head -c 54 | od -An -tu4 -j18 -N8
        480         85
$ go run ./cmd/restool cat <ru>/graphics.res interface/spellback.bmp | head -c 54 | od -An -tu4 -j18 -N8
         36         36
```

86 and 81 are `SPR256-PICT-043`'s own counts. `orc`, `orc2`, `orc3`, `orc4` with **no `orc1`** is the
tier-1 truncation's signature, read off the shipped index rather than off the claim. 480x85 and
36x36 are the atlas and its backing tile; both are two bytes longer than `54 + w*h*3`, the same
trailing pair the decoder is written to ignore.

## Re-run: two reversion witnesses

A line is witnessed if reverting it fails something. Both reversions were made in the working tree,
run, and reverted; `git status --short` was clean afterwards and the suite green again.

```
$ # FigureFor's sex axis put back on bit 7 of the face column:
$ #   return FigureDirFor(mage, face&0x80 != 0), int(face & 0x7f)
--- FAIL: TestFigureForReadsTheGenderColumnAndTheMageRange       (pkg/data)
        a_woman_fighter / a_woman_mage / a_face_byte_with_its_top_bit_set_is_that_face
--- FAIL: TestFigureForIsTotalAndItsTwoAxesAreIndependent        (pkg/data)
--- FAIL: TestAPlacedPersonsFigureComesFromTheGenderColumn       (pkg/game)
        a_woman_fighter_—_the_same_face,_the_other_directory / a_woman_mage
        a_face_byte_with_bit_7_set_is_that_face_and_still_a_man
$ # the pane's ground lowered one step below opaque: PortraitFill A: 0xff -> 0xfe
--- FAIL: TestATransparentPictureLeavesThePanesOwnGround         (pkg/ui)
```

Three test functions and six named subtests across two packages hold the gender-column reading, one
of them on exactly the line that a man and a woman of one face must not share a directory. One test
holds the opaque ground. Commit `9657d9c` states "fails four tests" for the first of these; measured
here it is three functions / six subtests, so that count is off by whatever it was counting.

## Recorded by the story, not re-run here

Taken against both preserved roots by probes that no longer exist. **What would re-take any of them
is a fresh throwaway probe of the same shape**, walking the campaign missions off both roots and
resolving each placement or tag through the same exported functions the game uses — the tree ships
no standing tool for it, deliberately, because none of it is a developer command anyone needs twice.

- 18 unit classes compose a figure and 16 take a flat picture; all 16 read and decode at 160x240,
  none absent, on both roots; thirteen carry tiers 2, 3 and 4 and three carry only tier 1.
- Six RU campaign missions (10, 20, 30, 40, 71, 130): **299 resolved placements — 184 portraits,
  115 figures, 0 misses on either arm**; tiers 2, 3 and 4 all appear and all read.
- Both roots' fifteen campaign missions: **462 human placements**. The gender column lands all 462
  on a shipped sheet and draws **186 as women** (157 `ffighter`, 29 `fmage`); the bit-7 reading also
  lands 462 and draws **0** as women. Inverting the sex axis leaves 238 landing, the class axis 141
  — the four directories ship 31, 13, 10 and 5 faces, so landing is a test, not a tautology. **No
  shipped `Humans` row sets bit 7 of its face column: 0 of 216.**
- **337 `npc=` tags on the RU root and 333 on the EN**, every one naming a record and resolving to a
  picture that reads: 163 compose a figure, 7 load a class portrait, 167 are the player's own
  character. The competing reading — the tag names a placement's npc subscript — matched 53 of 337,
  and seven of those missions carry dialogue while placing no npc at all.
- **49 npc records resolve to a composed figure and every one moves RIGHT** under the decoded
  window, by +4 to +13 px, mean 7.9, 38 of them by exactly +8; 13 state their own window and 36 take
  the engine's default; the window opens at picture row 8..27 instead of row 0. That is the owner's
  own "5-10px to the right", arrived at from the registry.
- All **33** creature speaker records now address 33 distinct files instead of 9, and all 33 read on
  both roots.

## The witnesses

| Id | Witness |
|---|---|
| FR-1 | `data.ComposesFigure`, one comparison against `composedClassLimit`; `TestComposesFigureSplitsAtTheEnginesOwnBoundary`; the 18/16 split recorded above. |
| FR-2 | `data.PortraitPath`; `TestPortraitPaths` covers the empty name. |
| FR-3 | `data.PortraitTierPath`; `TestPortraitPaths`; and the re-run `orc`/`orc2`/`orc3`/`orc4`-with-no-`orc1` listing. |
| FR-4 | `pkg/formats/bmp`; six tests including `TestDecodeIgnoresImgSizeAndAnyTail`, `TestDecodeHonoursTheRowPadding` and `TestAtIsTotal`. |
| FR-5 | `game.classPortrait`; `TestClassPortraitAddressesTheTier` drives tier 1 through 4 and the single-file fallback. The divergence is disclosed in the function's doc, in `spec.md` and in the build README. |
| FR-6 | `game.composeUnitFigure`, base sheet then ascending occupied slots, nil base returning nil; `TestAPlacedPersonsFigureComesFromTheGenderColumn`. |
| FR-7 | `data.FigureFor` over the three columns; `HumanDef.Gender` from slot 18; `TestFigureForReadsTheGenderColumnAndTheMageRange`, and the 462/186/0-of-216 measurement recorded above. |
| FR-8 | `game.pushPortrait` pushing owner 0 with no picture; `ui.dollSubject`'s three arms; `TestTheDollBoxPrefersFigureThenPortraitThenFrame`. |
| FR-9 | `data.LoadNPCFaces`'s three kinds; `game.SpeakerFace`; `TestSpeakerFaceAnswersTheThreeKinds`, `TestNPCFacesLetsTheFlagBeatTheKey`, `TestNPCFacesReadsNegationAsAbsence`; 337/333 tags recorded above. |
| FR-10 | `ui.NoticeFaceWindow` and `drawNoticePortrait`; `TestTheFaceWindowIsCutFromThePictureAndPlacedInThePane`, `TestASpeakersOwnWindowMovesWhatThePaneShows`. |
| FR-11 | `drawNoticePortrait`'s source-over blend over `PortraitFill` black; `TestATransparentPictureLeavesThePanesOwnGround`, `TestAFaceWindowOffThePictureDrawsWhatOverlaps`. |
| FR-12 | `SpeakerFace` returning false on every miss; `TestSpeakerFaceIsTotalWithNoTable`, `TestAnEmptyPaneIsStillPainted`. |
| FR-13 | `data.SpellIconCell` / `SpellIconSlot` and `game.cutSpellIcon`; `TestCutSpellIconTakesTheSlotsOwnSquare`, `TestSpellAtlasRefusesAPictureOfTheWrongExtent`; 480x85 re-run above. |
| FR-14 | `ui.composeSpellBar` drawing the label only where `Icon == nil`; `TestASpellCellDrawsItsIconOrItsLetters`, `TestSpellbookOfCarriesTheIconAndIsTotalWithout`. |
| AC-1 | `TestComposesFigureSplitsAtTheEnginesOwnBoundary`. |
| AC-2 | `TestPortraitPaths`. |
| AC-3 | `TestDecodeRefusesEveryShapeThisCorpusDoesNotShip`, `TestDecodeReadsTheRowsTopDownAndTheChannelsInOrder`, `TestDecodeIgnoresImgSizeAndAnyTail`, `TestDecodeHonoursTheRowPadding`. |
| AC-4 | `TestClassPortraitAddressesTheTier`, `TestClassPortraitCachesByNameIncludingItsMisses`. |
| AC-5 | `TestUnitPictureAsksTheClassBeforeTheArchive` (party subject answers nil), `TestAPlacedPersonsFigureComesFromTheGenderColumn`; the worn set is half the cache key, `figureCacheKey`. |
| AC-6 | `TestFigureForIsTotalAndItsTwoAxesAreIndependent`, and the reversion witness above. |
| AC-7 | `TestUnitPictureAsksTheClassBeforeTheArchive` drives the nil-mission map; `mw.archive()` is the guard a test found before a player did. |
| AC-8 | `TestNPCFacesResolvesTheThreeKinds`, `TestNPCFacesLetsTheFlagBeatTheKey`, `TestNPCFacesIsTotalOverAnAbsentRegistry`. |
| AC-9 | `TestNPCFacesCarriesThePaneWindow` (both-or-neither), `TestSpeakerFaceCarriesTheWindowAndTheTier`, `TestAFaceWindowOffThePictureDrawsWhatOverlaps`, `TestATransparentPictureLeavesThePanesOwnGround`, `TestAFaceWithNoPaneIsDrawnNowhere`. |
| AC-10 | `TestSpellIconSlotIsTheEnginesOwnTable`, `TestSpellIconGridFitsItsAtlas`, `TestCutSpellIconTakesTheSlotsOwnSquare`, `TestASpellCellDrawsItsIconOrItsLetters`. |
| P-1 | `check-no-game-assets.sh` clean on the tree scan; every fixture in the new test files is bytes assembled in Go; the full suite is green with no `AGAINROM_ASSETS` set. |
| P-2 | `git diff --stat master..HEAD` touches no file under `pkg/sim`; the byte-form version does not move and no digest test changed. `TestDrivenWorldReachesTheDigestOfAHeadlessRun` is untouched and green. |
| P-3 | `SetUnitPortrait`, `SpellEntry.Icon` and `Dialogue.Face` carry an `*image.RGBA` and a rectangle; no class id, key or address crosses into `pkg/ui`. |
| P-4 | `internal/archtest`'s allow-map gives `pkg/formats/bmp` an empty import set; `TestImportGraph` enforces it. |
| P-5 | `TestClassPortraitCachesByNameIncludingItsMisses`, `TestSpellIconCachesTheAtlasAndTheMisses`; `figurePics` is keyed by figure and worn set; `spellAtlasTried` holds the atlas miss. |
| DD-1 | The constant `composedClassLimit = 0x1a` and one `<` in `ComposesFigure`; the partition is in the doc, not in the code. |
| DD-2 | Two exported functions, `PortraitPath` and `PortraitTierPath`, neither implemented in terms of a nullable tier. |
| DD-3 | The divergence paragraph sits in `classPortrait`'s doc, at the line that forms the address. |
| DD-4 | `classPortrait`'s single recursive call, guarded by an address comparison so the ordinary case costs one read; `TestClassPortraitAddressesTheTier`. |
| DD-5 | `pkg/formats/bmp/doc.go` states the refusal; `Decode` names each rejected field; `TestDecodeRefusesEveryShapeThisCorpusDoesNotShip`. |
| DD-6 | `HumanDef.Gender` restored at slot 18 and `FigureFor`'s `gender == genderFemale`; the reversion witness fails three tests. |
| DD-7 | `mageTypeLo`/`mageTypeHi` with the proxy declared in their doc block, and the two corroborations stated there. |
| DD-8 | `game.LoadNPCFaces` reads the campaign registry; `speakers.go`'s doc carries the 53-of-337 result that eliminated the placement reading. |
| DD-9 | `data.LoadNPCFaces` reads `Face` before the switch and tests the flag before the key; `TestNPCFacesLetsTheFlagBeatTheKey`. |
| DD-10 | `FrontEnd.NPCFaces` built once in `NewFrontEnd`, passed into `openMission`; `openMission` assigns `faces = mw` only where the caller supplied none. |
| DD-11 | `FaceSource.SpeakerFace` returns picture and rectangle together; `Viewer.noticeFace`/`noticeFaceWindow` are written at every path under one serial; `ui.NoticeFaceWindow` holds the 72x96. |
| DD-12 | `noticeFaceAtX`/`noticeFaceAtY` with the `8 + 72 + 8 = 88` argument in their doc block. |
| DD-13 | The premultiplied source-over loop in `drawNoticePortrait` and `PortraitFill: color.RGBA{A: 0xff}`; the alpha reversion witness. |
| DD-14 | `spellAtlas`'s extent check, `cutSpellIcon`, and `bookCellSize = 38`; `TestSpellAtlasRefusesAPictureOfTheWrongExtent`, `TestAShippedPortraitFitsTheFigureBoxExactly`. |
| DD-15 | `mw.push()` calls `pushPortrait` beside `pushSpellbook`; `portraitOwner` is compared against the drawn entity in `dollSubject`. |
| DD-16 | `mw.archive()`, the single nil-mission guard, used by `unitPicture`, `unitFigure`, `speakerFigure` and `spellAtlas`. |
| SC-1 | The gate block above, re-run on a clean tree. |
| SC-2 | `internal/archtest` green, including the `pkg/sim` source scan; the allow-map entry is the only change to it. |
| SC-3 | Recorded: 18/16, all 16 reading at 160x240, thirteen with four tiers and three with one. Partly re-derived here — 86 EN and 81 RU `infowindow` nodes, and the `orc`..`orc4` naming. |
| SC-4 | Recorded: 299 placements over six RU missions, 184 portraits and 115 figures, 0 misses. |
| SC-5 | Recorded: 462 human placements, 186 women by the gender column against 0 by the face byte, 238 and 141 landing under the two inversions, 0 of 216 rows with bit 7 set. |
| SC-6 | Recorded: 337 RU and 333 EN tags, all naming a record and all resolving, split 163/7/167. |
| SC-7 | Recorded: 49 records, every one moving right by +4..+13 px (mean 7.9, 38 by +8), 13 stating a window and 36 taking the default. |
| SC-8 | Re-run above: 480x85 and 36x36 off the shipped headers. The per-cell colour census (24 cells, none a single colour) is recorded, not re-run. |
| SC-9 | Both reversion witnesses re-run above, with the count correction noted. |
| SC-10 | Re-run above: 17 and 13, equal to `pipeline/milestone-baseline.txt`. |

## What is not verified

- The window origin's whole reading rests on `REG-NPC-091`'s **Medium** grade for the canvas being
  bottom-up. If that is overturned every speaker's window moves. The claim names the cheap falsifier.
- The two bytes past the pixel run on 85 of 86 portrait nodes are Unknown and this decoder ignores
  them; nothing here would notice if they meant something.
- One campaign placement of 462 states no gender cell. The constructor's default in that state is
  not established anywhere and this build draws a man.
- The fifteen `npc.reg` person records carrying both a `Face` and a `DataBinID` disagree with their
  `Humans` row on the face in five cases and on the sex in one. Nothing here reconciles them, and
  nothing tests that they agree, because they do not.
