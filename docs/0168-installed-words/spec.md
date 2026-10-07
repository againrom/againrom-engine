# 0168 — installed words

## Terms

**Program-chosen word.** A string this build authors and draws on a player-facing surface: a menu
row, a button, a panel label, a notice sentence. It is distinct from a word the install supplies
through a decoded contract already in this tree (a unit name, an item name, an event line), which
already moves with the root.

**String index space.** The line-pointer array `TEXT-STRTAB-023` decodes: sixteen text files loaded
in a fixed order into one array. `main.txt` loads first, so its own line numbers 0..273 are the
global subscripts. The other fifteen tables are read with a table-local index through
`R0668(table, i)`, which adds the table's own base.

**Decoded index.** A line number of one of those files whose meaning is fixed by an active claim
that names a consumer instruction. An index whose text merely looks like a word this build draws is
not decoded.

**Install word.** A program-chosen word that this build resolves from a decoded index at run time.

**Authored word.** The English string this build ships in its own source, used when the install has
no entry at the index.

## Problem

G1 is the game on original ROM1 assets, English **and** Russian. On a Russian install today every
word the program chooses is English and every word the install supplies is Russian, on one screen.
The install's own words for several of those surfaces are decoded, cited, and unread by this build:
`main.res::text/main.txt` slots 77, 140 and 141, and the in-game menu's rows in
`main.res::text/dialogs.txt`.

Nothing in this tree reads either table outside character generation, which has its own private
reader in `pkg/game/chargenassets.go`.

## Word inventory

Every program-chosen word this build draws, by surface, split by whether a decoded index covers it.

**(a) Covered by a decoded `main.txt` index.**

| Words | Count | Slot | Claim |
|---|---|---|---|
| Notice button (`OK`), dialogue and outcome layouts | 1 | 77 | `MENU-STRTAB-008` |
| Mission outcome sentences | 2 | 140, 141 | `MENU-STRTAB-008` |
| Character-name prompt — already read | 1 | 125 | `TEXT-CHARGEN-028` |
| Chargen skill hover prose — already read | 10 | 171..180 | `TEXT-CHARGEN-027` |
| Chargen navigation rows — already read | 3 | 238, 239, 260 | `TEXT-CHARGEN-027` |
| Chargen name refusals — already read | 2 | 193, 194 | `MENU-STRTAB-008` (no consumer) |

Three of those words are new to the seam: 77, 140, 141. The sixteen below them were already read by
the generator's private reader and move to the shared one unchanged.

**(b) Covered by a decoded index in another install table.**

| Words | Count | `dialogs.txt` line | Claim |
|---|---|---|---|
| Mission menu rows | 7 | 34, 35, 36, 37, 38, 39, 40 | `MENU-ITEM-011` |
| Town menu `Abort Game` row | 1 | 77 | `MENU-ITEM-012` |

`MENU-ITEM-011` binds the descriptor at `L06197` to `main\text\dialogs.txt` at two adjacent
instructions and states that `R0668` resolves a table-local index. That closes `0158` AU-2,
which held the descriptor base undecoded.

**(c) Program-chosen with no decoded index.**

| Surface | Count | Why |
|---|---|---|
| Unit information panel captions | 32 | `UNIT-PANEL-011`: the original's panel layout is positively unrecoverable, this panel is the owner's authored arrangement, and no claim names an index for its captions |
| Character skill and school names | 11 | same panel, same reason |
| Panel always-hits mark | 1 | same panel, same reason |
| Developer readout captions | 15 | diagnostic surface with no original |
| Developer readout state words | 6 | same surface, same reason |
| Shop screen sentences | 12 | `main.txt` 60..82 holds shop words, and no claim names a consumer for any of them |
| Shop shelf names | 4 | same surface, same reason |
| Town screen sentences | 4 | authored sentences with no counterpart index |
| Map-list notice after a won mission | 1 | authored sentence with no counterpart index |
| HUD toggle letters | 4 | authored control panel, `0140`, no original |
| Chargen preview placeholder | 1 | authored diagnostic |

91 program-chosen words stay English. That is the disclosed divergence, and `FR-9` states it in the
player's own units. The counts are per distinct string and the commands that produce them are
recorded in `verification.md`.

## Functional requirements

**FR-1.** A text table is read from the install through the container filesystem and split into
lines by the decoded loader contract: each line ends with `CR`, the byte after the `CR` is skipped
whatever it is, and the bytes between are used exactly as shipped with no code-page pass at load.

**FR-2.** A table read this way exposes a line by index. An index below zero, at or past the line
count, or naming an empty line is **absent**, not an error and not an empty string.

**FR-3.** The install word set is built once, at front-end construction, from `text/main.txt` and
`text/dialogs.txt`. Failure to read either file is not a construction failure: every index in the
missing table is absent.

**FR-4.** Each program-chosen word in inventory categories (a) and (b) resolves to the install's
line at its decoded index when that index is present, and to this build's authored English word when
it is absent. No other program-chosen word changes.

**FR-5.** The mission outcome notice states `main.txt` line 140 when the mission is won and line 141
when it is lost.

**FR-6.** The dialogue notice's button and the outcome notice's button both state `main.txt` line
77.

**FR-7.** The seven mission in-game-menu rows state `dialogs.txt` lines 34, 35, 36, 37, 38, 39 and
40 in that screen order. The five town rows state lines 34, 35, 37, 77 and 40 in that screen order,
line 77 being the abort row.

**FR-8.** A row's accelerator and its underlined column are computed from the row's **drawn** label,
by the walk `MENU-KEY-013` decodes, whichever source the label came from. A label whose accelerator
byte is not ASCII is drawn with that column underlined and cannot be reached from this build's
keyboard input.

**FR-9.** The divergence is stated in `docs/0168-installed-words/` in the player's units: on a
Russian install the in-game menu, the notice buttons and the two mission-outcome sentences are in
Russian, and the unit panel's captions, the shop's sentences, the town's notices and the developer
readout stay in English because no decoded index names them.

**FR-10.** Character generation reads its sixteen slots through the shared table rather than through
its own private splitter, and its resolved strings are unchanged.

## Acceptance criteria

**AC-1** (FR-1). Splitting `A\r\nB\r\n` yields lines `A` and `B`. Splitting `A\rXB\r\n` yields `A`
then `B`: the byte after the `CR` is skipped whatever it is.

**AC-2** (FR-1). A line's bytes are returned unchanged for byte values `0x80..0xFF`.

**AC-3** (FR-2). Index -1, index equal to the line count, and an index naming an empty line each
report absent.

**AC-4** (FR-3). A source stating neither text file builds a word set equal to the authored English
set, with no error. This says nothing about front-end construction: `LoadChargenAssets` requires
eleven `main.txt` slots of its own and refuses an install without them, which is that loader's
pre-existing rule and is not changed here.

**AC-5** (FR-4, FR-5). Over a fixture whose `main.txt` carries lines 140 and 141, the outcome notice
states those lines. Over a fixture that carries neither, it states `MISSION COMPLETE` and
`MISSION FAILED`.

**AC-6** (FR-4, FR-6). Over a fixture carrying line 77, both notice layouts' button labels are that
line. Over a fixture without it, both are `OK`.

**AC-7** (FR-4, FR-7). Over a fixture carrying the eight decoded `dialogs.txt` lines, the mission
surface's seven rows and the town surface's five rows draw those lines in the specified order. Over
a fixture without them, both surfaces draw exactly the labels this build drew before this story.

**AC-8** (FR-7). A fixture supplying only some of the eight lines resolves those and leaves the rest
authored: resolution is per index, not per table.

**AC-9** (FR-8). A label `~ABC` yields accelerator `a` and underline column 0. A label whose marked
byte is `0xAA` yields that byte as the accelerator and column 0, and no ASCII key matches it.

**AC-10** (FR-10). The generator's sixteen resolved strings over a fixture are byte-identical to
what the private reader produced for the same fixture.

**AC-11** (FR-4). The 91 category-(c) words are unchanged: the panel layout, the readout layout, the
shop view and the town notices produce the same strings they produced at `2e662e6`.

**AC-12** (FR-3, FR-9). An install-gated count reports, per root, how many of the eleven seam-carried
words resolve from the install. It is 11 of 11 on both preserved roots, and the resolved bytes differ
between them for all eleven.

## Non-functional

**P-1.** No archive read happens on a drawing path. Every install word is resolved once, at
front-end construction, into values the viewer and the flow hold.

**P-2.** The serialized byte-form version is untouched. Nothing here reaches simulation state; the
words are presentation.

## Out of scope

**SC-1.** The 91 category-(c) words. Renumbering `main.txt` indices to cover them is a research
question, not an implementation one.

**SC-2.** The `Diplomacy` row (`dialogs.txt` 76). `0158` AU-3 does not build that row and this story
does not add it.

**SC-3.** CP866 case folding for a Cyrillic accelerator. `MENU-KEY-013` decodes the fold; this build
takes ASCII keys only, so folding a byte no key produces would be dead code. FR-8 discloses the
consequence.

**SC-4.** The in-mission message layer's 26 indices (85..89, 129, 142..149, 204..209, 221..226).
`MENU-STRTAB-008` resolves that the layer reads them and matches them to message families by
pattern; no single index has a decoded meaning, so none is buildable on.
