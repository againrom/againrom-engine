# Provenance — the dialogue window shows who is speaking

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 (the pane is decided once by the whole file) | `DLG-FACE-008` — the constructor lowercases the **entire payload** and sets the layout flag from a containment of `npc`, and that flag is what the window builder reads to choose between its two layouts | High for the two tests and their different consumers — three named instructions in two routines |
| FR-1's missing `=` | `DLG-FACE-008` — the row states the hazard in its own terms: prose containing the letters `npc` would open the portrait layout on a file with no speaker at all | High, same instruction |
| FR-2 (the two layouts, and the rectangles) | `DLG-WIN-001` — the window builder adds the portrait child at `30,54-118,168` **only when** the flag is set, and adds the text control at `128,36-428,172` with the portrait and `48,36-428,172` without; the button and the window rect are outside the branch | High for the geometry and the two layouts: each is a named `PUSH` in one routine read whole |
| FR-2's "panel-relative" reading | `DLG-WIN-001` — read as screen coordinates the pane's top edge would sit 66 px above the window's own; read as panel-relative all four children fall inside the window with nothing overhanging | High; the rect convention is fixed by an instruction (`bottom − top`), not by fit |
| FR-3 (the per-part test, and that it differs) | `DLG-FACE-008` — inside the pager the same field is rewritten from the current part's own tag, from a containment of `npc=`, and a third routine uses it to decide whether to refresh the portrait child | High for the separation, which is established from the instructions rather than from the corpus |
| FR-3's tag vocabulary and case folding | `DLG-MARKUP-007` — the content model is a tag scan whose bodies are lowercased and substring-searched, and `npc=` is the second of its fifteen literals | High for the vocabulary and the scan; `npc=` is one of the three literals the row followed to a consumer |
| FR-4 (a part naming nobody leaves the face standing) | `DLG-FACE-008` — stated as behaviour: parts without `npc=` leave the previous face standing | High |
| FR-7 (the outcome notice carries no pane) | `DLG-PATH-002` — the two outcome panels are built by their own routines from string-table entries, not by the dialogue window's constructor, so neither test is on their path | High for the two arms and their string-table indices |
| AC-11's expected result | `DLG-FACE-008` — 225/225 EN and 228/228 RU agree on both tests, so the corpus cannot discriminate them | High for the corpus half; exhaustive over both roots |
| AC-12's two roots | `DLG-LANG-010` — nothing on this path depends on the language root except the resources, and the executable is byte-identical across both | High for the identical image; the event corpora differ by three files |

## Ours by choice

| Spec anchor | What we fixed that no source asserts |
|---|---|
| FR-5 (an unresolvable speaker empties the pane) | The original always has a picture, so it has no such case and cannot rule on it. Authored: the pane keeps its place so the window's shape is the game's whatever the picture situation is, and blanking rather than keeping a stale face is what stops the window from attributing one speaker's face to another. |
| FR-6 (the picture comes from outside the contract) | Ours, and it exists because of the Open row below. The window is specified against a picture it is given so the contract is complete today and unchanged on the day the decode lands. |
| the empty pane's fill and border | Ours. The original's pane holds shipped bitmap art this project does not read, exactly as the window's frame and the button's word already are; the values are chosen to match what already ships. |
| FR-8 (a screen that cannot draw a notice) | Ours, inherited unchanged from `0066` and `0077`. No source addresses lettering that failed to load. |
| P-3 (showing the same part twice draws the same pixels) | Ours. It is this tree's composition discipline rather than a fact about the original, which repaints destructively. |

## Open

| What | Why it is left open |
|---|---|
| **Where a speaker's face is kept** | `REG-NPC-058` publishes `Face`, `Picture` and `PortraitX1/X2/Y1/Y2` with their presence counts and measured domains, and grades **Unknown** what any of them indexes: "the domains are measured, the target resource is not identified". No other ledger names a resource for this window's portrait child, and no row states what the child is loaded from. This is the story's one unresolved hop and the reason FR-6 is a seam rather than a rule. |
| What the registry's `Flags` tokens select | `REG-NPC-058` publishes the eleven-token vocabulary and its fifteen combinations, and puts no gloss on what any token does. Some tokens plainly concern the player rather than a picture; which key a given combination makes authoritative is unread. |
| Whether the speaker number in a tag is a registry subscript at all | Nothing published states it. The spec does not depend on it: FR-3 fixes the number as what the tag carries, and FR-6 leaves what to do with it outside the contract. |
| What the window does when its file yields no part 1 | `DLG-EMPTY-004` establishes a fallback literal reachable only from an authored file, and no shipped file reaches it. Untouched by this story, which changes what the window looks like and not when it opens. |

## Removed

| What was dropped | Why |
|---|---|
| A requirement naming the resource the pane's picture is read from | It would have been an unverified fact asserted as fidelity. Replaced by FR-6, which states the seam, and by the Open row above, which states why. |
| A requirement that the pane show the speaker's **name** | The pager does hand a name out alongside the speaker number, but no published row states where that name comes from or what it is for. Naming a speaker from a source we cannot cite is the same defect as guessing the picture. |
| A requirement reproducing the per-part speech sound | The name the pager composes resolves to nothing on either root (`DLG-LANG-010`, Medium for that clause), so there is no behaviour to reproduce. Kept as a disclosure. |
| A requirement that the five other entries into this panel class gain the pane | `DLG-WIN-001` establishes that they share the class, but none of the five is built in this tree. Kept as a disclosure that they inherit the contract. |
