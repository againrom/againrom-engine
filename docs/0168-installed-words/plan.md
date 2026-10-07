# 0168 — plan

## Shape

One seam and four consumers.

```
pkg/game/installtext.go   TextTable (FR-1, FR-2), InstallWords (FR-3), Words() (FR-4)
        |
        +-- pkg/game/frontend.go        loads it once, hands ui.Words to the App (P-1)
        |
pkg/ui/words.go           Words, AuthoredWords() (FR-4)
        |
        +-- pkg/ui/flow.go              App.SetWords -> flow.words -> viewer at the attach point
        +-- pkg/ui/gamemenu.go          rows built from Words (FR-7, FR-8)
        +-- pkg/ui/viewer.go            notice button labels and outcome sentences (FR-5, FR-6)
        +-- pkg/game/world.go           outcome sentence asked of the viewer (FR-5)
```

`pkg/game/chargenassets.go` loses its private splitter and reads through `TextTable` (FR-10).

## Design decisions

**DD-1 — the reader is one type in `pkg/game`, not a copy per surface.** `chargenassets.go` already
holds a private `chargenTextRows`/`chargenTextAt` pair. A second surface reading `main.txt` would be
a second answer to "what is a line", and `TEXT-STRTAB-023` gives exactly one. `TextTable` is that
answer and the generator moves onto it. Rejected: leaving the generator's reader alone and adding a
second. It would put two splitters in one package with no rule saying which is right.

**DD-2 — the splitter is the decoded loader's walk, not `bytes.Split`.** `TEXT-STRTAB-023` says the
scan finds a `CR`, terminates the line there, and advances the cursor **by 2** — the byte after the
`CR` is skipped whatever it is. `bytes.Split(b, "\r\n")` agrees with that on well-formed input and
disagrees on `A\rXB`, where the decoded walk yields `A` then `B` and the split yields one line. The
walk is what is decoded, so the walk is what is implemented. The inherited private splitter also
guessed a `NUL` separator when the payload held more than one `NUL`; nothing decodes that and it is
dropped.

**DD-3 — two accessors named for the two indexing styles.** `Global(i)` subscripts
`[L04369]` directly, which is what `main.txt` slots 77, 140 and 141 are. `Dialogs(i)` is
`R0668`'s table-local read against that table's descriptor, which is what the eight
`dialogs.txt` labels are. Both exist on
`InstallWords` and the caller cannot mix them up by passing a number to a single `At(i)`. `main.txt`
is loaded first so its global subscript equals its own line number; that identity is asserted in a
test rather than left as a comment, because it is the one place the two styles coincide.

**DD-4 — a struct of named fields, not a map keyed by index.** `ui.Words` has one field per
program-chosen word the seam carries. A map would make a missing key a run-time empty string; a
field makes an unresolved word a compile-time visible default. `AuthoredWords()` returns the eleven
English strings exactly as this build shipped them at `2e662e6`, and resolution overwrites a field
only when the install has that line.

**DD-5 — absence falls back and does not fail.** `TEXT-STRTAB-023` states no reader bounds-checks a
subscript, so the original has no authored behaviour at an absent index to reproduce. An install
missing a line produces this build's English word. Rejected: refusing to construct. It would make a
patched or partial install unplayable over a button caption, and the front end already carries the
opposite rule for the font, the attack pointer and the spell art.

**DD-6 — resolution happens once, at front-end construction.** `NewFrontEnd` already reads the
language selector there for the font's own stated reason: it is a property of the install, and the
drawing path must stay free of archive reads (P-1). The word set is read in the same place and by
the same rule.

**DD-7 — the words reach the two surfaces by the paths those surfaces already have.** `App.SetWords`
stores on `flow`; `flow.go`'s single `f.viewer = v` attach point copies them onto the viewer, where
they rewrite `noticeLayouts[0].ButtonLabel` and `noticeLayouts[1].ButtonLabel`. `flow.menuRows()`
builds its rows from `f.words`. Nothing gains a second source of truth: the flow's copy is what the
menu reads and the viewer's copy is what the notices read, and both are assigned from the one value
the App was handed. Rejected: a package-level variable in `pkg/ui`. `AuthoredPanelLayout` and
`AuthoredDialogueLayout` are functions rather than variables for exactly that reason — two viewers
must not be able to reach each other by mutation.

**DD-8 — the outcome sentence is asked of the viewer, not carried in `pkg/game` constants.**
`missionOutcomeText` becomes a method on the words the viewer holds. `MissionWonText` and
`MissionLostText` stay as the authored defaults, in `pkg/ui` beside the other ten, and `pkg/game`
keeps exported aliases so existing tests and the headless trace name the same values.

**DD-9 — the menu rows keep their `Fallback` immediate and gain no source flag.** A row is
`{Label, Fallback, Action, Enabled}` today; the only change is that `Label` may come from the
install. `gameMenuAccelerator`, `gameMenuLabelText` and `gameMenuAcceleratorColumn` already operate
on the row's own label, so FR-8 needs no code change — it needs a test proving the walk holds for a
non-ASCII marked byte. `lowerASCII` is left ASCII-only (SC-3); a CP866 accelerator is returned
unfolded and no ASCII key matches it.

**DD-10 — the English row labels stay upper-case and the installed ones are used as shipped.** This
build draws `~SAVE GAME` where the install holds `~Save Game`. Upper-casing an installed Cyrillic
label would need the CP866 fold this story excludes, and lower-casing the authored ones would change
the English screen for no reason. The consequence is visible and is disclosed: on the English root
the menu rows change case when they resolve from the install.

**DD-11 — the install-gated count is a test, not a tool.** `pkg/game` already skips
`release_integration_test.go` and `patrol_test.go` on an absent `AGAINROM_ASSETS`. AC-12's count
joins them: it opens the root, builds the word set, and reports how many of the eleven resolve and
whether the bytes differ from the authored English. A second root is named by
`AGAINROM_ASSETS_RU`, and the test skips that half when it is unset rather than assuming a sibling
directory.

**DD-12 — the fixtures are synthetic.** No game bytes enter the repository. A fixture `main.txt` is
built in the test as `strings.Repeat("\r\n", n)` with the wanted lines written into it, and a
fixture archive is the same `res` builder the existing chargen tests use.

**DD-14 — the divergence is stated where a player would look, in his units, and
in one place (FR-9).** `spec.md`'s category-(c) table names the surfaces and the counts;
`verification.md` restates it as the sentence a player can check against his screen — which words are
Russian on a Russian install and which stay English. Rejected: a `README` note in
`builds/0168-installed-words/`, which would go stale the moment `builds/current/` is rebuilt from a
later master. Rejected: an in-game notice, which would put a limitation of the decode on the
player's screen every session.

The disclosure is in the READER's units and not the format's. "`main.txt` indices 15..46 have no
cited consumer" is true and useless to him; "the unit panel's captions, the shop's sentences and the
town's notices stay English" is what he sees.

**DD-13 — `formatVersion` is untouched.** Nothing here reaches simulation state. The words are
resolved at construction and drawn; no word is hashed, serialized or compared in the digest.

## Superseded and cut

- `0158` AU-2 said the menu's label source was undecoded and the labels were therefore authored.
  `MENU-ITEM-011` decoded the descriptor binding after that story landed. The AU is closed by
  FR-7; the authored labels remain as the fallback.
- `pkg/game`'s `chargenTextRows` `NUL`-separator guess is cut (DD-2). Nothing decodes it and the
  generator's own fixtures use `CRLF`.
- Not built: a translation layer, a message catalogue, or any binding of a category-(c) word to an
  index that resembles it (SC-1).
- Not built: the `Diplomacy` row (SC-2). `0158` AU-3 does not build it and nothing here reads
  `dialogs.txt` 76.
- Not built: any read of the in-mission message layer's 26 indices (SC-4). `MENU-STRTAB-008`
  resolves that the layer reads them and matches them to families by contiguity; no single index
  has a decoded meaning, so none is buildable on.

## Risks

- **A wrong index prints a wrong word on a real screen.** Mitigated by taking every index from a
  cited instruction and by AC-12, which reads the eleven lines off both preserved roots and requires
  all eleven present and root-dependent.
- **The English screen changes case** (DD-10). It is the visible cost of using the shipped label and
  it is disclosed rather than papered over.

## Success criteria

Every FR has a test; AC-12 reports 11 of 11 on both roots; the category-(c) strings at `2e662e6`
are unchanged; `check-milestone.sh`'s two script-gap counts are unchanged.
