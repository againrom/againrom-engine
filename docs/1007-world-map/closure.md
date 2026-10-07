# 1007 — world map — closure

## Result

The town gates now show the install-backed world map. Live accepted missions appear as scroll cards
with shipped title, briefing and `Payment`. Selection draws a flag and advances a sampled party
route one stamp per update. The TOWN scroll and Escape return to the town. A selected mission
starts only through the existing `MissionOpener` path.

The previous gates list and its stale `tasks.md` preparation brief were removed: both described the
list-only result the owner's later directive replaced. No production file or package was deleted.

## Twelve-aspect closure

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `ReadGlobalMap` retains declared indices, validates rectangles and resolves dynamic mappings. The lazy manifest reads BMP, `.16a`, `.256`, text and registry nodes through the container filesystem |
| Runtime state | PASS | `worldMapState` owns hover, selection, route and visible prefix. Re-entry reconstructs it from live offers |
| Simulation | N/A | No file under `pkg/sim` changed. No command, digest, tick rule or format version changed |
| Player input | PASS | UI tests cover card priority, object-order map hits, pointer placement, arrows, wheel, Enter and per-update animation |
| AI | N/A | No AI rule or source changed |
| UI / HUD | PASS | `ComposeWorldMap` produces one 640×480 frame with background, route, markers, flag, cards and fallbacks. The existing placement scales and letterboxes it |
| Triggers / scripts | PASS | The real-root witness invokes the existing mission opener and receives a mission viewer. No mission or script construction path was added |
| Inventory / equipment | N/A | The screen does not inspect or mutate items. The persistence comparison includes the carried party snapshot |
| Persistence / save-load | PASS | `TestGatesShowLiveScrollTextSelectRouteAndUseTheMissionDoor` compares the town snapshot and encoded save bytes before and after entry, selection and return |
| Campaign / session | PASS | The witness accepts an offer through a building, enters through GATES, returns, re-enters and opens the accepted mission |
| Shipped content | PASS | `WorldMapSweep` decodes all 13 fixed pictures or masks, every mapping, all five optional mission pictures and every title/briefing pair on EN and RU |
| Interactions | PASS | Building acceptance, `Town.Available`, install language font, frame placement, Back and `MissionOpener` are the existing seams |

No in-scope GAP remains.

## Synthetic verification

Focused tests cover:

- non-shipped object counts, stable malformed indices, rectangle conversion and out-of-range
  mappings;
- PathMap corridor search, sampled endpoints and missing-mask fallback;
- live scroll text, briefing, reward, selection, route animation, town return and mission opener;
- equal town snapshots and equal encoded save bytes before and after a map round trip;
- successful and missing manifest nodes cached across visits;
- paged cards without a live-offer count bound, card priority, object-order overlap, map composition
  and App input/draw wiring.

The focused command was:

```text
go test -trimpath ./pkg/ui ./pkg/game ./cmd/worldmapcheck
```

It passed.

## EN and RU production witness

The same command form was run separately with each lawful root through `AGAINROM_ASSETS`:

```text
AGAINROM_ASSETS=<root> go run ./cmd/worldmapcheck
```

EN output:

```text
world map: chapter 30 accepted mission 30; 1 scroll(s); title 10 bytes; briefing 75 bytes; payment 1000; route 24 stamps; frame composed; mission opened
world map sweep: 28 mappings, 28 title/briefing pairs, 15 main, 9 side and 4 mapping-only rows; 26 building-offered; 5 picture and 23 fallback markers; frame composed; all consumed
```

RU output:

```text
world map: chapter 30 accepted mission 30; 1 scroll(s); title 11 bytes; briefing 68 bytes; payment 1000; route 24 stamps; frame composed; mission opened
world map sweep: 28 mappings, 28 title/briefing pairs, 15 main, 9 side and 4 mapping-only rows; 26 building-offered; 5 picture and 23 fallback markers; frame composed; all consumed
```

The witness uses `NewFrontEnd`, the shipped town offer, shipped dialogue, `Town.Available`, the
production global-map adapter, `ComposeWorldMap` and `MissionOpener`. It composes both the selected
mission-30 fallback frame and the all-mapping sweep frame. It opens no window and writes no install
file.

## Adversarial-review correction

The first pushed revision let absent `*image.RGBA` values cross directly into `image.Image` fields.
Those interfaces were non-nil even though their held pointers were nil. The original headless drive
did not compose a frame, so it passed while production composition panicked in `Bounds`. Adding the
production composition to `WorldMapSweep` reproduced the panic independently on EN and RU at mission
30 before the correction.

`worldMapOptionalImage` now normalises every fixed picture, scroll and mission marker at the adapter
boundary. `worldMapImagePresent` is the compositor's independent guard for every optional image
field and every helper that calls `Bounds`. The missing-manifest production adapter test composes a
fallback frame, while the UI test supplies typed nils in the background, route, marker, selection
and scroll positions together.

Two mutations were killed:

- returning an absent `*image.RGBA` directly from the adapter fails
  `TestWorldMapAdapterNormalisesEveryMissingOptionalImage` on the background field;
- replacing the compositor guard with `pic != nil` panics
  `TestWorldMapCompositionTreatsTypedNilImagesAsMissing` at the background `Bounds` call.

## Research reconciliation and divergences

The implementation uses only claims active at pin `4aae01f` as ROM1 authority. The owner's later
route directive supersedes the prepared omission of `PathMap.bmp`. Unpublished EXP-0188 work is not
cited or imported.

- `DIV-104` records the authored PathMap mask indices, start object, path search, sampling and
  animation cadence.
- `DIV-105` records the authored scroll-card geometry, fallbacks and activation gestures.
- `DIV-106` records use of `Town.Available()` without ROM1's unresolved per-region flag.
- `DIV-107` records the omitted `PictureOffset` effect.
- Reserved IDs `DIV-108` through `DIV-110` are returned unused and may not be reused under the
  project's spent-ID rule.

No campaign-progress indicator was added. `TOWN-045` remains a scoped negative result and was not
strengthened.

## Observable result outside tests

The EN and RU drives both opened the selected shipped mission after the route completed. This story
does not change the mission 10 or mission 20 script census. A missionrun binary built from this tree
reported the following on both roots, identical to `pipeline/milestone-baseline.txt`:

```text
mission 10: 16 checks, 27 instants, 12 triggers
mission 20: 14 checks, 15 instants, 11 triggers
```

No unsupported-node total is inferred from the old uppercase-only selector: it can print zero while
the widened milestone census is nonzero. This story changes no script consumer.

The concrete result this story moves is the production town frame and campaign drive, not that
census: GATES changes from a text list to the world-map result described above.

## Clean gates

The implementation gate passed: `go build ./...`, `go vet ./...`, `gofmt -l` over tracked and
untracked Go files, and `go test -trimpath -count=1 ./...`. The asset scan reported
`check-no-game-assets: clean (tree scan)`.

At the exact submodule pin, the research `go build`, `go vet`, `gofmt` and `go test -trimpath
-count=1 ./...` chain passed. Every `research/scripts/check-*.sh` passed:

```text
check-claim-ids: ok (1255 ids, all distinct, 31 ledgers, 29 read back through tools/claim)
check-retraction-status: ok (195 overturned ids, every one marked)
```
