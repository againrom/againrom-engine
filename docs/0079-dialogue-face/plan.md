# Plan — the dialogue window shows who is speaking

The contract splits along the tier boundary that already exists. **What a file says** is the
driver's — it holds the payload and the part number and it already scans tags. **What the window
looks like** is the viewer's — it holds the geometry, the composition and the picture cache. So the
tag vocabulary stays entirely below `pkg/ui`, and what crosses the seam is a flag, a string and an
optional picture.

## Design decisions

- **DD-1 — ONE layout value carries both text rectangles and the pane rectangle, and the composition
  picks.** `NoticeLayout` gains `TextBesidePortrait` and `Portrait`, and an exported
  `WithPortrait(bool)` returns the layout resolved for one branch: `Text` set to the applicable
  rectangle, `Portrait` emptied when there is none. This is the original's own shape — one routine
  holding both literals and branching on one flag. *Rejected:* a second `NoticeLayout` value beside
  the first, which makes `SetNoticeLayouts` a three-argument seam whose third argument is not a kind
  and leaves two values that must be kept in step on every field they share (FR-2 is mostly a
  statement that they are). *Rejected:* a third `NoticeKind`, which would let the outcome notice be
  asked for a pane and would make "which notice" and "which shape" one enumeration.
  Serves FR-2, AC-4.

- **DD-2 — whether the pane is drawn is state of the OPEN NOTICE, beside its words and its kind.**
  The viewer holds `noticePortrait bool`. FR-1 makes it a property of the file, so it changes when a
  window opens and never while one is up — which is exactly the lifetime the words already have.
  *Rejected:* deriving it in the viewer by testing the words it was handed. That would apply FR-1's
  test to one part's text instead of the file's, so a file whose part 3 alone names a speaker would
  change shape as the player paged. Serves FR-1, AC-1, AC-2, AC-3.

- **DD-3 — the driver opens a dialogue in ONE statement.** `Viewer.SetDialogue(Dialogue)` carries the
  words, the pane flag, whether this part names a speaker and the picture. *Rejected:* keeping
  `SetNotice` and adding two setters beside it. Three statements can be made in the wrong order, or
  two of the three made at all, and the picture cache would then be presentable under a shape it was
  not composed for — the failure the composition's existing one-statement rule already names for the
  picture and its freshness flag. Serves FR-1, FR-3, P-4.

- **DD-4 — "does this part name a speaker" and "what is the picture" are two fields, not one nullable
  picture.** `Dialogue.Speaks` decides whether the pane is written at all; `Dialogue.Face` is what is
  written when it is. FR-4 and FR-5 are otherwise indistinguishable at the seam: both would arrive as
  a nil picture, and the window cannot tell "nobody spoke, keep what is there" from "somebody spoke
  and I could not find them". *Rejected:* a sentinel picture for the second case, which is a value
  every consumer must know to test. Serves FR-4, FR-5, AC-5, AC-7.

- **DD-5 — the rebuild key gains the pane flag and a face SERIAL, not the picture's address.** The
  viewer bumps a counter whenever the face is written and the key carries the counter. A pointer
  comparison would miss a caller that hands back the same buffer with different pixels, and comparing
  pixels would put an image compare on every frame. Serves P-3, AC-6.

- **DD-6 — both of FR-1's and FR-3's tests live in `pkg/game`, beside the part scan that already
  exists.** `EventHasSpeaker(payload)` is the file test and `EventPartSpeaker(payload, n)` the
  per-part one, and the per-part one reuses the SAME single scan that finds a part's body — extracted
  to one unexported walk so the tag a part's text comes from and the tag its speaker comes from
  cannot be two different tags. `pkg/ui` learns no tag literal. Serves FR-1, FR-3, AC-8.

- **DD-7 — a speaker's picture comes from a `FaceSource` the driver is handed, and today nothing
  implements it.** The front-end carries the field, passes it to the mission driver, and leaves it
  nil; a nil source resolves nothing, so every named speaker takes FR-5's path. This is the whole of
  the story's open hop: when the decode lands, an implementation of one method is what closes it, and
  no contract, layout, test or call site moves. Serves FR-6.

- **DD-8 — the empty pane is PAINTED, with the window's own frame treatment in two new layout
  colours.** Leaving it as window fill would make FR-5 indistinguishable from FR-3 having failed —
  the player, and a reviewer, would see a window with no pane rather than a pane with no picture.
  The two colours join the layout's existing eight, so the pane's paint is substituted the same way
  the rest of the window's is. Serves FR-5, AC-7.

- **DD-9 — `SetNotice` keeps its signature and settles both new questions to their empty values.**
  The outcome path is unchanged at its call site, and FR-7 is then locked twice over: that path sets
  the pane flag false, AND the outcome layout's `Portrait` and `TextBesidePortrait` are empty, so
  `WithPortrait(true)` on it is still a window with no pane. `ClearNotice` drops the face with the
  words. Serves FR-7, P-4, AC-9.

- **DD-10 — the corpus census is a new developer tool, `cmd/dlgtool`.** It answers AC-11 by running
  the two exported predicates over every event file of a root and printing counts only. It is a new
  `cmd/` package rather than a verb on an existing one because none of the existing tools has this
  function: `texttool` measures fonts, `missionrun` drives one mission. It prints **figures, never
  bytes**, so its output can be pasted into evidence without carrying game text. Serves AC-11, P-2.

- **DD-11 — nothing under `pkg/sim` is opened.** The two tests read a payload the driver already
  holds and the picture never enters the world. No world field, no byte-form version, no digest.
  Serves P-1.

## Risks

- **R-1 — the two tests collapse into one.** They differ by one character and agree on every shipped
  file, so a build that used `npc=` for both, or `npc` for both, passes every corpus check and every
  playthrough. Only a fixture that separates them can fail. *Mitigation:* AC-2 is that fixture, and
  it is the only test in the set that discriminates.

- **R-2 — a face outlives its window.** FR-4 keeps the picture across parts of ONE file; carrying it
  into the NEXT window would attribute one mission's speaker to another's. *Mitigation:* the face is
  dropped by both `SetNotice` and `ClearNotice`, and every path that opens a window goes through one
  of those or through `SetDialogue`, which writes it.

- **R-3 — the cached picture goes stale.** The composition is rebuilt only when the key changes, so a
  new pane flag or a new face with the same words would present the previous picture. *Mitigation:*
  DD-5, and a test that pages between two speakers on identical words.

- **R-4 — the narrower text rectangle changes the wrap and the clamp.** With a pane the words have
  300 px instead of 380 and the same height. That is the contract, not a defect, but it means every
  shipped assertion about wrapping is now an assertion about the WITHOUT-pane rectangle.
  *Mitigation:* keep those assertions on the without-pane layout and add the with-pane case beside
  them rather than editing them.

- **R-5 — a game asset reaches the repository through the new tool.** *Mitigation:* DD-10 prints
  counts only and writes no file; `check-no-game-assets.sh` is the gate.

## Success criteria

1. A file whose tags name a speaker composes with the pane and the with-pane text rectangle, on
   every part. (FR-1, FR-2)
2. A file containing the three letters only in prose composes with the pane and never a face.
   (FR-1) — the discriminating case.
3. A file containing neither composes without the pane and with the without-pane text rectangle.
   (FR-1, FR-2)
4. The two resolved layouts differ in exactly the pane rectangle and the text rectangle. (FR-2)
5. Paging from a speaking part to a silent one keeps the face; paging to a differently-speaking one
   replaces it. (FR-3, FR-4)
6. A named speaker with no resolvable picture draws the pane empty and leaves the words, the button
   and the three dismissal routes working. (FR-5, FR-6)
7. An outcome notice whose words contain the three letters draws no pane. (FR-7)
8. A viewer with no font draws nothing and holds nothing, with speaker-bearing dialogues pushed.
   (FR-8)
9. The census over both preserved roots reports both tests agreeing on every event file, and every
   file answering yes. (FR-1, FR-3)
10. A mission is run to a dialogue on each root and the window is looked at. (AC-12)
11. `pkg/sim` is untouched and the digest tests are unchanged. (P-1)

## Traceability

| Spec | Design decision | Criterion |
|---|---|---|
| FR-1 | DD-2, DD-3, DD-6 | 1, 2, 3, 9 |
| FR-2 | DD-1 | 1, 3, 4 |
| FR-3 | DD-3, DD-4, DD-6 | 5, 9 |
| FR-4 | DD-4 | 5 |
| FR-5 | DD-4, DD-8 | 6 |
| FR-6 | DD-7 | 6 |
| FR-7 | DD-9 | 7 |
| FR-8 | — (the shipped gate is unchanged) | 8 |
| P-1 | DD-11 | 11 |
| P-2 | DD-10 | 9 |
| P-3 | DD-5 | 4, 5 |
| P-4 | DD-3, DD-9 | 5, 7 |
| AC-11 | DD-10 | 9 |
| AC-12 | — (a runnable build) | 10 |
