# The campaign record goes one way into a SAV

## Intent

A campaign field reached a SAV through five routes, each with its own order
of campaign hook, native projection, tail word and home word: the mission
document, the town document, the town snapshot's carried progress record,
the resumed mission snapshot, and the offer labels a SAVE writes. This story
builds every campaign record a SAV carries through one projection per
document kind, with the hooks, the native projection, the tail, the home word
and the offer label source as its inputs and profile data. No player-visible
or hashed-state change.

A mission's first frame (the application state LOAD installs: view,
selection, panels, speed, fog plane, animation clock) was restored on two
arms of `restoreApplicationState`, one per source. This story restores it
once, from the source the save has, with what each source restores as data
of that source.

## Authority

- Owner direction: the third unification after the World carrier set (story
  1542) and the projectile records (story 1544); after story 1526 it also
  takes the second branch of original-save LOAD (part B).
- Audit shape: rank 3 of `pipeline/reviews/arch-audit-c40f21fe.md` (rank 9 of
  `arch-audit-a5c60f20.md` and the rows it points back to), in the seat.
- No ROM1 or ROM2 behaviour changes. No DIV row changes.

## As built

### Part A: one campaign record projection

`pkg/game/campaignrecord.go` holds the projection. `FrontEnd.campaignRecord(s,
kind)` is the one builder of the campaign record of a document kind. A
kind's route (`campaignRecordRouteOf`) lists the record's sources in the
order they are tried and the overlays each source's record takes:

| kind | stated | carried | chapter |
|---|---|---|---|
| mission document | the profile's home word | tail | native projection of the generated chapter; played child, selection, successor, mission's map object, tail |
| town document | the profile's home word, marker paths | tail, marker paths | native projection of the current main; tail, marker paths, map objects |
| resumed mission snapshot | (none) | (none) | native projection of the generated chapter; selection, mission's map object |

- Sources: stated is the record the campaign states itself (the profile's
  `stated`); carried is `Snapshot.Campaign` while `CampaignState` holds; chapter
  is `nativeCampaignProjectionForChapter` over the kind's chapter. The chapter
  source is the chapter progress shape: its child record rule, its successor
  and its tail word.
- Overlays (`campaignRecordOverlays`), applied in field order: the played side
  mission's child record when absent, the played mission's selection unless a
  guest plays it, its AutoGetMission, the mission's map object (home word
  cleared), the home word (`homeAsBuilt`, `homeTail` through `campaignTail`,
  `homeTown`), the marker picture paths, the registry map objects.
- Profile (`campaignRecordProfile`, from `campaignService.recordProfile`,
  which replaces the `campaignProjection` hook): the first game states no
  record and writes its chapter offers; the second game states the bank's
  record of a session that holds the destinations campaign
  (`secondStatedRecord`), keeps its home word in a mission document and sets
  it in a town document, and writes the offer labels as LOAD stated them.
- The snapshot's capture (`captureCampaignRecord`, called by `snapshotTown`,
  which now takes the campaign service) seats the carried progress record
  (`progressCampaignRecord`: the progress's record with the town's live
  tavern pool, document payload and documents) and the offer labels
  (`capturedOfferLabels`).
- `seatCarriedCampaignRecord` is what `ExportCurrentSave` adds to the carried
  record before the document is built: fame, world-map markers with their
  paths, and map objects unless the record is a town's loaded progress. The
  payload bound now runs before it; the two touch disjoint fields.
- Offer label source as data: `offerLabelsAsLoaded` (second profile) writes
  the labels LOAD restored while the town holds no progress;
  `offerLabelsOfChapter` (first profile) writes the current chapter's
  offers, and a campaign with no main list holds chapter 0 and writes the
  loaded labels, as before. On the four installs the first game's main list
  has 15 missions and the second game's none (probe over `NewFrontEnd` on
  `gameversions/{en,ru,rom2-en,rom2-ru}`), so the second profile's rule and
  the replaced `len(t.camp.Main) == 0` test answer alike there. A second-game
  mod that adds a main list would now keep the loaded labels.
- Guard: `internal/archtest/campaignrecord.go` names, for every source,
  overlay, profile and capture function, the functions that may reference it.

### Part B: one first-frame restore

`pkg/game/firstframe.go` holds the source. `mapWorld.restoreApplicationState`
is the one restore; `firstFrameSourceOf(mission, original)` picks the row,
and `firstFrameSource.firstFrame` builds the state through
`fileApplication.view`, then overlays the current view where the row has one.
The fog and animation steps of the restore run when the row supplies them.

| row | chosen when | application state and actor IDs | current view | fog plane | animation |
|---|---|---|---|---|---|
| decoded document (`documentFirstFrame`) | the LOAD is of an original save with its document | `readOriginalApplicationState`, IDs from the document's actor table (`applicationActorIDs`), original book rows | `readCurrentActions` through `applyCurrentView`; the baseline is that view | `originalFogPlane` over the opener's plane | yes, from the current actions |
| file store (`fileStoreFirstFrame`) | not an original save, and the mission kept `Mission.fileApplication` | the state store's leaves read at LOAD (`readFileApplication`), IDs from the import's source bindings, the viewer's book rows | none: the store holds no engine action leaf | none: the opener's plane (`readOriginalFog`) stands | none: no action leaf to read |

- A row with no state (no mission, a native mission, an original save
  without its document) restores nothing when the snapshot carries none,
  as before.
- A snapshot that carries its own state keeps it; its row still supplies the
  fog plane and the animation, as the document arm did.
- Guard: `internal/archtest/firstframe.go` names, for each reader of either
  source and for the restore's steps, the functions that may reference it:
  `readFileApplication` only from the original source's mission preparation,
  the document readers only from the document row, the rows only from
  `firstFrameSourceOf`, and the restore only from `settleMission`.

### Kinds and builders

| kind | builder |
|---|---|
| campaign record in a SAV (mission, town documents; resumed mission snapshot) | `FrontEnd.campaignRecord` |
| offer labels a SAVE writes | `captureCampaignRecord` through `capturedOfferLabels`, source from the profile |
| campaign rules per game | `campaignService.recordProfile` (`campaignfirst.go`, `campaignsecond.go`) |
| SAV document writing | `ExportCurrentSave`, unchanged; its campaign part calls the projection |
| first-frame application state of a LOAD | `mapWorld.restoreApplicationState` over `firstFrameSourceOf` |

### Unchanged

The byte formats (`sav.ProjectDocumentCampaign`, the dialect codecs), the
native projection's own fields, the World carrier set, and every LOAD path
except the first-frame restore of part B.
The capture-time fame and marker overlays of `FrontEnd.Snapshot` and
`citySnapshotFromMission` stay where they are: `ExportCurrentSave` writes
both again from the same fame and selection, so they decide no SAV byte; they
serve the snapshot's in-memory state.

## Proof

- Base digests: `TestCampaignRecordRoutesWriteTheBaseBytes` was run on the
  base tree (`git archive ff8496c96`) twice with equal results; its digests
  (SAV without label, snapshot record, snapshot labels) are its expected
  values: a town over its chapter, a town over loaded progress with a
  selected marker, a mission over loaded progress, a second-game mission
  file's SAVE and resave (offer chapter 10 in both).
- Route witnesses: `TestTheChapterSourceTakesItsRouteOverlays`,
  `TestTheSecondGamesStatedRecordKeepsItsHomeWord`,
  `TestTheOfferLabelSourceIsTheProfiles`.
- First-frame base digests: `TestFirstFrameSourcesRestoreTheBaseFrames`
  digests the restored state, explored and visible planes, animation clock
  and actor IDs of two second-game mission files (speed 4; speed 8 with the
  book open and a pressed spell) and of our SAV's decoded document; the
  expected values were recorded on the base tree twice and on the part A
  tree, all equal. `TestTheFirstFrameSourceStatesWhatItRestores` pins which
  row a LOAD takes and what each row restores.
- Loss controls, base and after each part on the same inputs: the SAV corpus
  gate on `2026-08-02/game0003.sav` (EN), `secondoriginalload_release_test.go`
  with `TestASaveKeepsTheLiveWorldAndAResaveItsBytes` on
  `gameversions/rom2-en` and `rom2-ru`, and the four town-kept tests on
  `gameversions/en`. These are lane witnesses, not release tests; counts in
  the lane's return.
- `go test ./pkg/game` (ordinary set) passes after each part.

## Open debt

- Rank 3: the capture-time fame and marker overlays (see Unchanged) remain
  outside the projection.
- Rank 2: the file store's state is still read at LOAD by
  `readFileApplication` and kept as `Mission.fileApplication`, while the
  document row reads at restore; a later step could read both at restore.
  The SAVE side (`captureApplicationState`) still reads the document's
  application state itself.
