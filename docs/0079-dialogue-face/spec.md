# Spec — the dialogue window shows who is speaking

**Intensity: spec-anchored / static. Terrain: brownfield** — the dialogue window ships, and this
changes the layout it draws and what decides it.

A mission's script speaks through one box of text over the map. The original draws that box in one of
**two** layouts: one with a picture of the speaker beside the words, and one without. This tree ships
only the second, and on shipped data the original never uses it — so the box the player sees today is
the right window in the wrong shape, with the space the speaker's face occupies given to text.

This contract fixes both layouts, what chooses between them, what chooses which face, and what the
window does when the face cannot be had.

## Functional requirements

- **FR-1 — the pane's presence is decided ONCE, by the whole file.** A dialogue window carries a
  portrait pane if and only if the event text it is paging through mentions a speaker **anywhere in
  it**. The test is over the payload's whole bytes with ASCII case folded, and what it looks for is
  the three letters **`npc`** — *not* `npc=`. A file whose prose contains those three letters and
  whose tags name no speaker at all therefore carries the pane, on every one of its parts. **Every
  part of one file gets the same answer**; paging never changes the shape of the window.

- **FR-2 — the two layouts differ in the pane and in the text area, and in nothing else.** With a
  pane, the pane occupies `30,54-118,168` and the words occupy `128,36-428,172`; without one, the
  words occupy `48,36-428,172` and no pane is drawn. Both rectangles are relative to the window's own
  origin. The window's own rectangle, its button's rectangle, the word on the button, the line pitch,
  the wrapping rule, the clamp on how many lines are drawn, the dim behind it and every colour are
  **the same in both** and are what they are today.

- **FR-3 — which face the pane shows is decided PER PART, by a DIFFERENT test.** The per-part test is
  over that part's own tag with ASCII case folded, and what it looks for is **`npc=`**. A part whose
  tag contains it names a speaker, and the speaker is the decimal number immediately following. The
  two tests are separate: a window whose file mentions a speaker somewhere shows the pane on parts
  whose own tags name nobody.

- **FR-4 — a part that names no speaker LEAVES THE STANDING PICTURE ALONE.** It is not cleared, not
  replaced and not defaulted; the face put there by an earlier part of the same window keeps the
  pane. This is behaviour and not an omission: it is what the original does, and a window that
  blanked the pane on such a part would diverge from it. A window that has shown no speaker yet has
  an empty pane, which is the same statement with nothing standing.

- **FR-5 — a named speaker whose picture cannot be had EMPTIES the pane.** The pane is still drawn,
  in its own place and at its own size, with nothing in it. This is the case FR-4 must not be
  confused with: a part that names nobody keeps the last face, a part that names somebody
  unresolvable shows none. Nothing else about the window changes — the words are shown, the button
  works and the three dismissal routes answer.

- **FR-6 — what a speaker's face looks like is NOT decided by this contract.** The window shows the
  picture it is given for the speaker of the part it is showing, and where that picture comes from is
  supplied from outside. Supplying nothing is the ordinary case today and is what FR-5 governs; it is
  not an error, produces no message, and is not a state the player can observe other than as an empty
  pane.

- **FR-7 — the notice that reports how a mission ended is untouched.** It carries no pane, whatever
  its words contain, and its geometry, colours, button and destinations are exactly what they were.
  Neither test above is applied to it.

- **FR-8 — a map screen that cannot draw a notice is untouched by every requirement above.** Where
  the lettering failed to load nothing is drawn and nothing is held, exactly as today, and the
  standalone developer viewer is unaffected.

## Acceptance criteria

| # | GIVEN | WHEN | THEN | level |
|---|---|---|---|---|
| **AC-1** | an event text whose tags name a speaker | its parts are shown in turn | every part is drawn with the pane and with the words in the with-pane rectangle | unit |
| **AC-2** | an event text containing the three letters `npc` only in its prose, with no tag naming a speaker | it is shown | the pane is drawn and no face is ever put in it | unit |
| **AC-3** | an event text containing neither the three letters nor any speaker tag | it is shown | no pane is drawn and the words occupy the without-pane rectangle | unit |
| **AC-4** | the two layouts | each composes a window | the window rectangle, the button rectangle, the button's word, the pitch, the wrap, the clamp and every colour are identical, and only the pane and the text rectangle differ | unit |
| **AC-5** | a file whose part 1 names a speaker and whose part 2 names none | it is paged from 1 to 2 | part 2 is drawn with the pane still carrying part 1's face | unit |
| **AC-6** | a file whose part 1 names one speaker and whose part 2 names a different one | it is paged from 1 to 2 | the pane carries part 2's speaker's face | unit |
| **AC-7** | a part naming a speaker for whom no picture can be had | it is shown, after an earlier part whose speaker's picture could | the pane is drawn empty, and the words, the button and the three dismissal routes are unaffected | unit |
| **AC-8** | tags spelling the speaker in mixed case, with spaces around the tag's other terms | they are shown | the speaker is found and is the number that follows | unit |
| **AC-9** | a mission-outcome notice whose words contain the three letters | it is shown | no pane is drawn, and its geometry, colours and destinations are what they were before this story | unit |
| **AC-10** | a map screen whose lettering failed to load | speaker-bearing dialogues are pushed and frames are driven | nothing is drawn, nothing is held, and no input is intercepted | unit |
| **AC-11** | the shipped campaign corpus of both preserved roots | every event file is measured against both tests | both answer the same on every file of both roots, and every file answers yes | dev-run |
| **AC-12** | a mission run to a dialogue on each of the two preserved roots | the window is looked at | the pane stands in its place beside the words on both, and the two roots agree | manual |

AC-7 is the error case: a picture that cannot be had must cost the window nothing but the picture.

## Derived properties

- **P-1 — invariant.** Nothing here reaches simulation state. No `pkg/sim` file changes, no world
  field is added, no serialized byte-form version moves, and two worlds driven the same number of
  ticks agree exactly as they did before this story.

- **P-2 — negative invariant.** No game asset enters the repository or its history. The pane is
  drawn from values held here; every fixture is bytes built in test code.

- **P-3 — idempotence.** Showing the same part twice draws the same pixels: the shape of the window,
  the face standing in the pane and the words are a function of the file, the part and the picture
  supplied, and of nothing that changes with time.

- **P-4 — completeness.** Every way a dialogue window can be opened or advanced in this tree settles
  both questions — whether there is a pane, and what stands in it. There is no path that opens the
  window with one answered and the other left over from an earlier window.

## Constraints

The two tests MUST be reproduced as **containments**, not corrected into equality or into a parse.
The first one's missing `=` is the original's, and it is what makes a file with no speaker able to
open the pane; narrowing it would diverge from the original on authored data, which is the wrong
direction for a reimplementation to diverge in.

## Out of scope, and disclosed

**The face art is not resolved, and that is this story's one open hop.** Where a speaker's picture is
kept is not established by any published claim: the ids the corpus uses reach a registry whose
picture-bearing keys are published with their domains measured and their meaning graded Unknown. The
window, both layouts, both tests and the per-part selection are built; the picture arrives through
FR-6 when the decode lands, and until then FR-5 is what the player sees. The question is filed for
research rather than guessed.

**The five other windows of this class are not touched.** The inn, the two mercenary-hall entries,
the shop keeper and the training hall share the panel class and are not built in this tree at all;
they inherit this contract when they are.

**No speech is played.** The original composes a sound name per part; no such subtree ships on
either root, so there is nothing to reproduce.

**The pane's own paint is ours.** The original's is shipped bitmap art this project does not read, so
the empty pane's fill and border are this project's own, chosen to match the window they sit in —
the same substitution the window's frame and its button already are.

## Verification mapping

AC-1 to AC-10 are CI-automatable with no game install. AC-11 is a developer run over both preserved
roots. AC-12 is a manual observation on both roots, because whether the window reads as the right
shape is a judgement this contract does not make.

## Gate check

FR-1 → AC-1, AC-2, AC-3, AC-9, P-4. FR-2 → AC-1, AC-3, AC-4. FR-3 → AC-1, AC-6, AC-8, AC-11.
FR-4 → AC-5, P-4. FR-5 → AC-7. FR-6 → AC-6, AC-7. FR-7 → AC-9. FR-8 → AC-10.
P-1 covers every FR at once. P-2 → AC-11's evidence carries no bytes. P-3 → AC-4, AC-5.
