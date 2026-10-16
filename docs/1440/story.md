# Editor workspace and mission package

## Intent and authority

Complete the first game's authoring workspace in the window: rectangular maps,
map metadata, mission pages, mod creation and source selection, and readable help.
Use the existing ALM constructor, transactional edit model, field prompt, mod
loader, mission routes and report list. The owner directs this editor's layout.
`ALM-NEWMAP-237` bounds new-map dimensions. `TEXT-083` locates the map name and
description. `DIV-3386` records the explicit Windows-1251 name authoring policy;
it makes no claim about the original editor's name input codec.

## As built

- N accepts width x height through the existing constructor: each dimension is
  32..256 in steps of 8. The installed-map catalogue remains available.
- M edits name, description and the seven existing raw scalar setters. Unknown
  effects remain labelled unknown. Other metadata, text slots, records and
  framing stay byte-exact. The old ASCII-only name API keeps its contract;
  the window uses a bounded, round-tripping Windows-1251 name writer.
- X selects title, briefing, briefmap or canonical numbered event/tips pages and edits its
  UTF-8 body. Shift+Enter inserts a line; Ctrl+A clears the current field.
  Unsupported characters, NUL, oversized pages and more than 64 pages refuse
  without adopting the edit. ALM and page/route edits share ordered Undo/Redo.
- W creates an owned single-mission mod package in an explicit new folder,
  declares the supported guest or campaign-replacement route, and selects an
  existing package. A multi-mission source uses the same field-choice catalogue
  to select its map and pages. Imported packages without the ownership record
  remain read-only and can be saved as an owned copy. The prior explicit
  `SetMod` save interface keeps its behaviour.
- Ctrl+S saves the whole owned package. F5 and durable creation share one
  manifest/map/mission/text builder and the normal mod-load validation stage.
  File conflicts, input/install overlap and invalid declarations refuse before
  publication. Retained unrelated files survive replacement. ALM-only Save As
  refuses unsaved pages or route changes. Opening another source refuses a
  dirty draft.
- P and F5 have visible labels. Help is laid out above save controls. F1 opens
  all key descriptions from the existing key table in the scrollable report.
  Record, detail, report and modal bounds use the same footer limit.

## Proof

`TestEncodedMapNameIsBoundedAndKeepsOtherBytes` checks independent name images,
the 63-byte bound, malformed input and exact Undo/Redo bytes.
`TestMapEditorWorkspaceMetadataAndPagesShareHistory` drives real window keys,
checks independent CP1251 images and each raw scalar span, mixes page and map
history, and proves refusal retention. The other workspace tests cover both
route declarations, F5 payloads, fresh editor reopen, staged loader refusal,
unrelated files and choosing the second map/text set of a manual source.

`TestReleaseMapEditorWorkspaceRoutesAndTexts` passes on EN and RU installs.
It creates a 40x56 map through N, authors Cyrillic metadata and seven
pages through M/X/W, plays both normal mod routes, reads the live dialogue and
world-map title/briefmap, serves event100 through live dialogue, and cold-reopens
the package. Name bytes are checked
independently before the installed-font boundary. The installed help-width
witness also passes on both fonts.

The help-layout witness covers 640x480, 1280x800 and 1920x1080. Existing wrapped
reference interaction and modal caret/refusal controls remain active. Their
explicit layout expectations and the full key/gesture golden are updated for
the new footer and rectangle prompt.

Ordinary tests pass for `pkg/mapedit`, `pkg/ui` and `pkg/game`.
`go build ./...`, `go vet ./...`, the gated-test population, story guard and
divergence ledger checks pass. Comment bytes fall from 9045035 to 9044837.
The installed-test population gains one subject and loses none.

The sole review returned F1 for omitting contextual help. The correction uses
every distinct key-table line and retains the save/history row. The regression
fails on the reviewed candidate, independently names add, brush-size and Tab
controls, checks full table coverage, and scrolls to the final row with Down.
The UI package and key/gesture matrix pass after the correction.
Page admission takes its numbered tip basename from the runtime tip table;
the architecture check retains the table as its one declaration.

The EN `missionrun -trace -ticks 1` drive reports zero unsupported nodes for
missions 10 and 20. Its script populations are respectively 16 checks,
27 instants, 12 triggers and 14 checks, 15 instants, 11 triggers, matching the
coordination milestone baseline. Both instrument outputs end undecided at
tick 64 after mission startup. This workspace changes no simulation node rule.

## Open boundaries

Unknown raw metadata effects and trailing text-slot meanings remain engine and
research debt; this workspace does not invent light/time semantics. Durable
output is one supported mission per package; multiple source declarations can
be selected and copied individually. No native editor or desktop interaction
was run. Repository-wide final gates, sole review, publication and build
promotion belong to the landing.
